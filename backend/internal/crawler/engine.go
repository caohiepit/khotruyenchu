package crawler

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"comics-backend/internal/cleaner"
	"comics-backend/internal/domain"
	"comics-backend/internal/storage"
)

// Engine manages the continuous background crawling worker pool
type Engine struct {
	mu                  sync.RWMutex
	novelRepo           *storage.NovelRepository
	queueRepo           *storage.QueueRepository
	fileStorage         *storage.FileStorage
	registry            *HandlerRegistry
	wakeUpChan          chan struct{}
	stopChan            chan struct{}
	isRunning           bool
	currentJob          *domain.CrawlQueueItem
	currentChapter      int
	currentChapterTitle string
	recentLogs          []string
	limitDefault        int
	delayDefault        int
}

func NewEngine(
	novelRepo *storage.NovelRepository,
	queueRepo *storage.QueueRepository,
	fileStorage *storage.FileStorage,
	registry *HandlerRegistry,
) *Engine {
	return &Engine{
		novelRepo:    novelRepo,
		queueRepo:    queueRepo,
		fileStorage:  fileStorage,
		registry:     registry,
		wakeUpChan:   make(chan struct{}, 10),
		stopChan:     make(chan struct{}),
		recentLogs:   make([]string, 0),
		limitDefault: 0,   // 0 means crawl all chapters
		delayDefault: 350, // ms safe delay
	}
}

// GetLiveStatus returns realtime telemetry for Admin dashboard
func (e *Engine) GetLiveStatus() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return map[string]interface{}{
		"is_running":            e.isRunning,
		"current_job":           e.currentJob,
		"current_chapter":       e.currentChapter,
		"current_chapter_title": e.currentChapterTitle,
		"recent_logs":           e.recentLogs,
	}
}

func (e *Engine) appendLog(msg string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	timestamp := time.Now().Format("15:04:05")
	formatted := fmt.Sprintf("[%s] %s", timestamp, msg)
	e.recentLogs = append(e.recentLogs, formatted)
	if len(e.recentLogs) > 30 {
		e.recentLogs = e.recentLogs[len(e.recentLogs)-30:]
	}
}

// Start begins the continuous worker goroutine
func (e *Engine) Start() {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return
	}
	e.isRunning = true
	e.stopChan = make(chan struct{})
	e.mu.Unlock()

	log.Println("🚀 [Crawler Engine] Continuous Queue Worker started in background...")
	go e.workerLoop()
}

// Stop gracefully signals the worker to pause
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.isRunning {
		return
	}
	close(e.stopChan)
	e.isRunning = false
	log.Println("⏸️ [Crawler Engine] Stop signal dispatched.")
}

// WakeUp notifies the worker that a new job is available in queue
func (e *Engine) WakeUp() {
	select {
	case e.wakeUpChan <- struct{}{}:
	default:
		// Channel already has pending notification
	}
}

// IsRunning checks worker running state
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.isRunning
}

// GetCurrentJob returns currently processing item
func (e *Engine) GetCurrentJob() *domain.CrawlQueueItem {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.currentJob
}

func (e *Engine) workerLoop() {
	for {
		select {
		case <-e.stopChan:
			log.Println("🏁 [Crawler Engine] Worker stopped.")
			return
		default:
		}

		// Try to fetch next pending novel from queue
		item, err := e.queueRepo.GetNextPendingItem()
		if err != nil {
			if err == sql.ErrNoRows {
				// Queue is empty, wait for wakeUpChan or timeout
				e.mu.Lock()
				e.currentJob = nil
				e.mu.Unlock()

				select {
				case <-e.wakeUpChan:
					continue
				case <-time.After(10 * time.Second):
					continue
				case <-e.stopChan:
					return
				}
			}

			// Transient DB error, sleep briefly
			time.Sleep(2 * time.Second)
			continue
		}

		// Process this novel item
		e.mu.Lock()
		e.currentJob = item
		e.mu.Unlock()

		e.processNovelItem(item)

		// Brief pause between novels
		time.Sleep(500 * time.Millisecond)
	}
}

func (e *Engine) processNovelItem(item *domain.CrawlQueueItem) {
	log.Printf("⏳ [Crawler Engine] Started crawling: %s (%s)", item.Title, item.SourceURL)
	e.appendLog(fmt.Sprintf("Bắt đầu cào truyện: %s (Nguồn: %s)", item.Title, item.SourceDomain))

	handler := e.registry.Resolve(item.SourceURL)
	if handler == nil {
		errStr := fmt.Sprintf("no handler found to parse url: %s", item.SourceURL)
		log.Printf("❌ [Crawler Engine] %s", errStr)
		e.appendLog(fmt.Sprintf("❌ Lỗi: %s", errStr))
		_ = e.queueRepo.UpdateStatus(item.ID, "failed", 0, errStr)
		return
	}

	log.Printf("   👉 Selected handler: 【%s】", handler.Name())

	// 1. Parse Novel Info
	novel, firstChapURL, err := handler.ParseNovel(item.SourceURL)
	if err != nil {
		log.Printf("❌ [Crawler Engine] Error parsing novel %s: %v", item.Title, err)
		e.appendLog(fmt.Sprintf("❌ Lỗi bóc tách truyện %s: %v", item.Title, err))
		_ = e.queueRepo.UpdateStatus(item.ID, "failed", 0, err.Error())
		return
	}

	if novel.Title == "" {
		novel.Title = item.Title
	}
	if novel.CoverURL == "" {
		novel.CoverURL = item.CoverURL
	}
	if novel.CoverURL != "" {
		novel.CoverURL = e.downloadAndCleanCover(novel.CoverURL, novel.Slug)
	}
	if novel.AuthorName == "" {
		novel.AuthorName = item.AuthorName
	}
	if novel.Slug == "" {
		novel.Slug = item.Slug
	}

	// 2. Crawl Chapters (with multi-source chapter aggregation)
	currentURL := firstChapURL
	if currentURL == "" {
		currentURL = fmt.Sprintf("%schuong-1/", item.SourceURL)
	}

	var chapters []*domain.Chapter
	previousURL := item.SourceURL
	currentChapterNum := 1
	limit := e.limitDefault
	if item.TotalChapters > 0 {
		limit = item.TotalChapters
	}

	consecutiveErrors := 0

	for currentURL != "" {
		select {
		case <-e.stopChan:
			log.Printf("⏸️ [Crawler Engine] Interrupted during novel %s", novel.Title)
			return
		default:
		}

		if limit > 0 && len(chapters) >= limit {
			break
		}

		chap, nextURL, err := handler.ParseChapter(currentURL, currentChapterNum, previousURL)
		if err != nil {
			log.Printf("⚠️ Chapter download error at %s: %v", currentURL, err)
			consecutiveErrors++
			if consecutiveErrors >= 3 {
				break
			}
			time.Sleep(1 * time.Second)
			continue
		}
		consecutiveErrors = 0

		chapters = append(chapters, chap)
		previousURL = currentURL
		currentURL = nextURL
		currentChapterNum = chap.ChapterNumber + 1

		// Update telemetry state
		e.mu.Lock()
		e.currentChapter = chap.ChapterNumber
		e.currentChapterTitle = chap.Title
		e.mu.Unlock()

		logMsg := fmt.Sprintf("[%s] Cào Chương %d: %s (%d chữ)", novel.Title, chap.ChapterNumber, chap.Title, chap.WordCount)
		log.Printf("   📖 %s", logMsg)
		e.appendLog(logMsg)

		// Save progress periodically (every 5 chapters)
		if len(chapters)%5 == 0 {
			_ = e.novelRepo.SaveNovelAndChapters(novel, chapters)
			_ = e.queueRepo.UpdateStatus(item.ID, "crawling", len(chapters), "")
		}

		// Jitter sleep to respect target site rate limits
		jitter := int(time.Now().UnixNano()%250) + 100
		time.Sleep(time.Duration(e.delayDefault+jitter) * time.Millisecond)
	}

	novel.TotalChapters = len(chapters)

	// 3. Save to disk & PostgreSQL (multi-source chapter merge handled in fileStorage)
	diskErr := e.fileStorage.SaveNovelAndChapters(novel, chapters)
	if diskErr != nil {
		log.Printf("⚠️ Failed saving to local disk: %v", diskErr)
	}

	dbErr := e.novelRepo.SaveNovelAndChapters(novel, chapters)
	if dbErr != nil {
		log.Printf("❌ Failed saving to PostgreSQL: %v", dbErr)
	}

	if diskErr != nil && dbErr != nil {
		_ = e.queueRepo.UpdateStatus(item.ID, "failed", len(chapters), "storage error: "+diskErr.Error())
		return
	}

	// 4. Mark completed in queue
	totalFinalChapters := e.fileStorage.CountTotalChapters(novel.Slug)
	if totalFinalChapters == 0 {
		totalFinalChapters = len(chapters)
	}
	_ = e.queueRepo.UpdateStatus(item.ID, "completed", totalFinalChapters, "")
	successMsg := fmt.Sprintf("✅ Hoàn thành cào truyện 【%s】: tổng cộng %d chương (kết hợp các nguồn)!", novel.Title, totalFinalChapters)
	log.Printf("✅ [Crawler Engine] %s", successMsg)
	e.appendLog(successMsg)
}

// downloadAndCleanCover tải ảnh bìa, tự động xóa watermark (chữ kiểu 'truyenfull.vn') và lưu cục bộ
func (e *Engine) downloadAndCleanCover(coverURL, slug string) string {
	if coverURL == "" || (!strings.HasPrefix(coverURL, "http://") && !strings.HasPrefix(coverURL, "https://")) {
		return coverURL
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", coverURL, nil)
	if err != nil {
		return coverURL
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	if strings.Contains(coverURL, "truyenfull") {
		req.Header.Set("Referer", "https://truyenfull.live/")
	} else if strings.Contains(coverURL, "tangthuvien") {
		req.Header.Set("Referer", "https://tangthuvien.net/")
	}

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			_ = resp.Body.Close()
		}
		return coverURL
	}
	defer resp.Body.Close()

	// 🧼 Xóa chữ kiểu 'truyenfull.vn' khỏi ảnh bìa
	cleanBytes, err := cleaner.CleanCoverWatermark(resp.Body)
	if err != nil || len(cleanBytes) == 0 {
		return coverURL
	}

	// Lưu ảnh sạch vào thư mục client-ui/covers/
	localRelPath := fmt.Sprintf("/covers/%s.jpg", slug)
	targetPaths := []string{
		filepath.Join("..", "frontend", "client-ui", "covers", fmt.Sprintf("%s.jpg", slug)),
		filepath.Join("frontend", "client-ui", "covers", fmt.Sprintf("%s.jpg", slug)),
		filepath.Join("storage", "covers", fmt.Sprintf("%s.jpg", slug)),
	}

	for _, p := range targetPaths {
		if abs, err := filepath.Abs(p); err == nil {
			_ = os.MkdirAll(filepath.Dir(abs), 0755)
			_ = os.WriteFile(abs, cleanBytes, 0644)
		}
	}

	log.Printf("🖼️ [Cover Cleaned] Đã tải ảnh bìa và xóa sạch watermark cho [%s] -> %s", slug, localRelPath)
	return localRelPath
}
