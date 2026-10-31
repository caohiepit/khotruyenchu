package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type Server struct {
	db            *sql.DB
	dbConn        string
	mux           *http.ServeMux
	jobMutex      sync.Mutex
	jobs          []CrawlerJob
	workerRunning bool
	workerStop    chan struct{}
}

type CrawlerJob struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Limit     int       `json:"limit"`
	Status    string    `json:"status"` // running, completed, failed
	Message   string    `json:"message"`
	StartedAt time.Time `json:"started_at"`
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	dbConn := os.Getenv("DATABASE_URL")
	if dbConn == "" {
		dbConn = "postgres://postgres:comics_password_2026@localhost:5432/comics_db?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbConn)
	if err != nil {
		log.Fatalf("❌ Cannot connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("⚠️ Warning: PostgreSQL ping failed at startup: %v. Will retry upon requests.", err)
	} else {
		log.Println("✅ Connected to PostgreSQL (comics_db) successfully!")
	}

	s := &Server{
		db:         db,
		dbConn:     dbConn,
		mux:        http.NewServeMux(),
		jobs:       make([]CrawlerJob, 0),
		workerStop: make(chan struct{}),
	}

	s.routes()

	handler := enableCORS(s.mux)

	fmt.Printf("\n=======================================================\n")
	fmt.Printf("🚀 COMICS BACKEND REST API SERVER IS RUNNING\n")
	fmt.Printf("📡 API Base URL:  http://localhost:%s/api\n", port)
	fmt.Printf("🖥️ Admin UI URL:  http://localhost:%s/admin/\n", port)
	fmt.Printf("=======================================================\n\n")

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func (s *Server) routes() {
	// API Endpoints
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/admin/stats", s.handleStats)
	s.mux.HandleFunc("GET /api/admin/novels", s.handleGetNovels)
	s.mux.HandleFunc("GET /api/admin/novels/{id}", s.handleGetNovelDetail)
	s.mux.HandleFunc("GET /api/admin/chapters/{id}", s.handleGetChapterContent)
	s.mux.HandleFunc("GET /api/admin/genres", s.handleGetGenres)

	// Crawler single novel
	s.mux.HandleFunc("POST /api/admin/crawler/start", s.handleStartCrawler)
	s.mux.HandleFunc("GET /api/admin/crawler/jobs", s.handleGetCrawlerJobs)

	// Two-Phase Queue Crawler
	s.mux.HandleFunc("POST /api/admin/crawler/discover", s.handleDiscoverCatalog)
	s.mux.HandleFunc("GET /api/admin/crawler/queue", s.handleGetQueue)
	s.mux.HandleFunc("POST /api/admin/crawler/queue/start", s.handleStartQueueWorker)
	s.mux.HandleFunc("POST /api/admin/crawler/queue/stop", s.handleStopQueueWorker)
	s.mux.HandleFunc("POST /api/admin/crawler/queue/reset-failed", s.handleResetFailedQueue)

	// Serve Frontend Admin UI & Client UI files directly
	adminDir := filepath.Join("..", "..", "frontend", "admin-ui")
	if absAdminDir, err := filepath.Abs(adminDir); err == nil {
		fs := http.FileServer(http.Dir(absAdminDir))
		s.mux.Handle("GET /admin/", http.StripPrefix("/admin/", fs))
	}

	clientDir := filepath.Join("..", "..", "frontend", "client-ui")
	if absClientDir, err := filepath.Abs(clientDir); err == nil {
		clientFS := http.FileServer(http.Dir(absClientDir))
		s.mux.Handle("GET /client/", http.StripPrefix("/client/", clientFS))
		s.mux.Handle("GET /", clientFS)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbErr := s.db.Ping()
	dbStatus := "connected"
	if dbErr != nil {
		dbStatus = "disconnected"
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"database": dbStatus,
		"time":     time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	var totalNovels int
	var totalChapters int
	var totalWords sql.NullInt64
	var totalGenres int

	_ = s.db.QueryRow("SELECT COUNT(*) FROM novels").Scan(&totalNovels)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM chapters").Scan(&totalChapters)
	_ = s.db.QueryRow("SELECT COALESCE(SUM(word_count), 0) FROM chapters").Scan(&totalWords)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM genres").Scan(&totalGenres)

	// Queue stats
	var queuePending, queueCrawling, queueCompleted, queueFailed int
	_ = s.db.QueryRow("SELECT COUNT(*) FROM crawl_queue WHERE status = 'pending'").Scan(&queuePending)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM crawl_queue WHERE status = 'crawling'").Scan(&queueCrawling)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM crawl_queue WHERE status = 'completed'").Scan(&queueCompleted)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM crawl_queue WHERE status = 'failed'").Scan(&queueFailed)

	words := int64(0)
	if totalWords.Valid {
		words = totalWords.Int64
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"total_novels":     totalNovels,
		"total_chapters":   totalChapters,
		"total_words":      words,
		"total_genres":     totalGenres,
		"queue_pending":    queuePending,
		"queue_crawling":   queueCrawling,
		"queue_completed":  queueCompleted,
		"queue_failed":     queueFailed,
		"queue_total":      queuePending + queueCrawling + queueCompleted + queueFailed,
		"worker_running":   s.workerRunning,
	})
}

func (s *Server) handleGetNovels(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	offset := (page - 1) * limit

	var total int
	baseCountQuery := "SELECT COUNT(*) FROM novels"
	baseDataQuery := `
		SELECT n.id, n.title, n.slug, n.author_name, n.description, n.cover_url, 
		       n.status, n.total_chapters, n.views_count, n.rating_score, n.realm,
		       n.source_url, n.created_at, n.updated_at,
		       COALESCE((SELECT COUNT(*) FROM chapters c WHERE c.novel_id = n.id), 0) AS actual_chapters,
		       COALESCE((
		           SELECT string_agg(g.name, ', ') 
		           FROM novel_genres ng 
		           JOIN genres g ON g.id = ng.genre_id 
		           WHERE ng.novel_id = n.id
		       ), '') AS genres_str
		FROM novels n
	`

	var rows *sql.Rows
	var err error

	if search != "" {
		filter := "%" + search + "%"
		_ = s.db.QueryRow("SELECT COUNT(*) FROM novels WHERE title ILIKE $1 OR author_name ILIKE $1", filter).Scan(&total)
		query := baseDataQuery + " WHERE n.title ILIKE $1 OR n.author_name ILIKE $1 ORDER BY n.updated_at DESC LIMIT $2 OFFSET $3"
		rows, err = s.db.Query(query, filter, limit, offset)
	} else {
		_ = s.db.QueryRow(baseCountQuery).Scan(&total)
		query := baseDataQuery + " ORDER BY n.updated_at DESC LIMIT $1 OFFSET $2"
		rows, err = s.db.Query(query, limit, offset)
	}

	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	type NovelItem struct {
		ID             string    `json:"id"`
		Title          string    `json:"title"`
		Slug           string    `json:"slug"`
		AuthorName     string    `json:"author_name"`
		Description    string    `json:"description"`
		CoverURL       string    `json:"cover_url"`
		Status         string    `json:"status"`
		TotalChapters  int       `json:"total_chapters"`
		ActualChapters int       `json:"actual_chapters"`
		ViewsCount     int64     `json:"views_count"`
		RatingScore    float64   `json:"rating_score"`
		Realm          string    `json:"realm"`
		SourceURL      string    `json:"source_url"`
		Genres         []string  `json:"genres"`
		CreatedAt      time.Time `json:"created_at"`
		UpdatedAt      time.Time `json:"updated_at"`
	}

	novels := make([]NovelItem, 0)
	for rows.Next() {
		var n NovelItem
		var desc, cover, realm, source, genresStr sql.NullString
		err := rows.Scan(
			&n.ID, &n.Title, &n.Slug, &n.AuthorName, &desc, &cover,
			&n.Status, &n.TotalChapters, &n.ViewsCount, &n.RatingScore, &realm,
			&source, &n.CreatedAt, &n.UpdatedAt, &n.ActualChapters, &genresStr,
		)
		if err == nil {
			n.Description = desc.String
			n.CoverURL = cover.String
			n.Realm = realm.String
			n.SourceURL = source.String
			if genresStr.String != "" {
				n.Genres = strings.Split(genresStr.String, ", ")
			} else {
				n.Genres = []string{}
			}
			novels = append(novels, n)
		}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"total":  total,
		"page":   page,
		"limit":  limit,
		"novels": novels,
	})
}

func (s *Server) handleGetNovelDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Missing novel ID"})
		return
	}

	var n struct {
		ID            string    `json:"id"`
		Title         string    `json:"title"`
		Slug          string    `json:"slug"`
		AuthorName    string    `json:"author_name"`
		Description   string    `json:"description"`
		CoverURL      string    `json:"cover_url"`
		Status        string    `json:"status"`
		TotalChapters int       `json:"total_chapters"`
		ViewsCount    int64     `json:"views_count"`
		RatingScore   float64   `json:"rating_score"`
		SourceURL     string    `json:"source_url"`
		CreatedAt     time.Time `json:"created_at"`
		UpdatedAt     time.Time `json:"updated_at"`
		Genres        []string  `json:"genres"`
	}

	var desc, cover, source sql.NullString
	query := `
		SELECT id, title, slug, author_name, description, cover_url, status, total_chapters, views_count, rating_score, source_url, created_at, updated_at
		FROM novels
		WHERE id::text = $1 OR slug = $1
	`
	err := s.db.QueryRow(query, id).Scan(
		&n.ID, &n.Title, &n.Slug, &n.AuthorName, &desc, &cover,
		&n.Status, &n.TotalChapters, &n.ViewsCount, &n.RatingScore, &source,
		&n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "Novel not found"})
		return
	}
	n.Description = desc.String
	n.CoverURL = cover.String
	n.SourceURL = source.String

	// Get genres
	genreRows, err := s.db.Query(`
		SELECT g.name 
		FROM genres g 
		JOIN novel_genres ng ON ng.genre_id = g.id 
		WHERE ng.novel_id = $1
	`, n.ID)
	if err == nil {
		defer genreRows.Close()
		for genreRows.Next() {
			var gName string
			if err := genreRows.Scan(&gName); err == nil {
				n.Genres = append(n.Genres, gName)
			}
		}
	}
	if n.Genres == nil {
		n.Genres = []string{}
	}

	// Get chapters list
	chapterRows, err := s.db.Query(`
		SELECT id, chapter_number, title, slug, word_count, views_count, created_at
		FROM chapters
		WHERE novel_id = $1
		ORDER BY chapter_number ASC
	`, n.ID)

	type ChapterMeta struct {
		ID            string    `json:"id"`
		ChapterNumber int       `json:"chapter_number"`
		Title         string    `json:"title"`
		Slug          string    `json:"slug"`
		WordCount     int       `json:"word_count"`
		ViewsCount    int64     `json:"views_count"`
		CreatedAt     time.Time `json:"created_at"`
	}

	chapters := make([]ChapterMeta, 0)
	if err == nil {
		defer chapterRows.Close()
		for chapterRows.Next() {
			var c ChapterMeta
			if err := chapterRows.Scan(&c.ID, &c.ChapterNumber, &c.Title, &c.Slug, &c.WordCount, &c.ViewsCount, &c.CreatedAt); err == nil {
				chapters = append(chapters, c)
			}
		}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"novel":    n,
		"chapters": chapters,
	})
}

func (s *Server) handleGetChapterContent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Missing chapter ID"})
		return
	}

	var chap struct {
		ID            string `json:"id"`
		NovelID       string `json:"novel_id"`
		ChapterNumber int    `json:"chapter_number"`
		Title         string `json:"title"`
		Slug          string `json:"slug"`
		WordCount     int    `json:"word_count"`
		Content       string `json:"content"`
	}

	query := `
		SELECT c.id, c.novel_id, c.chapter_number, c.title, c.slug, c.word_count, cc.content
		FROM chapters c
		JOIN chapter_contents cc ON cc.chapter_id = c.id
		WHERE c.id::text = $1
	`
	err := s.db.QueryRow(query, id).Scan(
		&chap.ID, &chap.NovelID, &chap.ChapterNumber, &chap.Title, &chap.Slug, &chap.WordCount, &chap.Content,
	)
	if err != nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "Chapter content not found"})
		return
	}

	jsonResponse(w, http.StatusOK, chap)
}

func (s *Server) handleGetGenres(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT g.id, g.name, g.slug, COUNT(ng.novel_id) AS novel_count
		FROM genres g
		LEFT JOIN novel_genres ng ON ng.genre_id = g.id
		GROUP BY g.id, g.name, g.slug
		ORDER BY novel_count DESC, g.name ASC
	`)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	type GenreItem struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Slug       string `json:"slug"`
		NovelCount int    `json:"novel_count"`
	}

	genres := make([]GenreItem, 0)
	for rows.Next() {
		var g GenreItem
		if err := rows.Scan(&g.ID, &g.Name, &g.Slug, &g.NovelCount); err == nil {
			genres = append(genres, g)
		}
	}

	jsonResponse(w, http.StatusOK, genres)
}

func (s *Server) handleStartCrawler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL   string `json:"url"`
		Limit int    `json:"limit"`
		Delay int    `json:"delay"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request payload. URL is required."})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.Delay <= 0 {
		req.Delay = 300
	}

	jobID := fmt.Sprintf("job-%d", time.Now().Unix())
	job := CrawlerJob{
		ID:        jobID,
		URL:       req.URL,
		Limit:     req.Limit,
		Status:    "running",
		Message:   "Crawler task dispatched",
		StartedAt: time.Now(),
	}

	s.jobMutex.Lock()
	s.jobs = append([]CrawlerJob{job}, s.jobs...)
	s.jobMutex.Unlock()

	// Execute crawler in background
	go func(jID string, targetURL string, limit int, delay int) {
		crawlerBin := filepath.Join("..", "crawler", "crawler.exe")
		absBin, err := filepath.Abs(crawlerBin)
		if err != nil {
			absBin = crawlerBin
		}

		cmd := exec.Command(absBin,
			"-url", targetURL,
			"-limit", strconv.Itoa(limit),
			"-delay", strconv.Itoa(delay),
			"-db", s.dbConn,
		)

		output, err := cmd.CombinedOutput()

		s.jobMutex.Lock()
		defer s.jobMutex.Unlock()
		for i := range s.jobs {
			if s.jobs[i].ID == jID {
				if err != nil {
					s.jobs[i].Status = "failed"
					s.jobs[i].Message = fmt.Sprintf("Error: %v | Log: %s", err, string(output))
				} else {
					s.jobs[i].Status = "completed"
					s.jobs[i].Message = "Successfully crawled and saved to PostgreSQL!"
				}
				break
			}
		}
	}(jobID, req.URL, req.Limit, req.Delay)

	jsonResponse(w, http.StatusAccepted, map[string]interface{}{
		"message": "Crawler job started successfully",
		"job":     job,
	})
}

func (s *Server) handleGetCrawlerJobs(w http.ResponseWriter, r *http.Request) {
	s.jobMutex.Lock()
	defer s.jobMutex.Unlock()
	jsonResponse(w, http.StatusOK, s.jobs)
}

// -------------------------------------------------------------
// Two-Phase Queue Crawler Handlers
// -------------------------------------------------------------

func (s *Server) handleDiscoverCatalog(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CatalogURL string `json:"url"`
		Pages      int    `json:"pages"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CatalogURL == "" {
		req.CatalogURL = "https://truyenfull.live/danh-sach/truyen-hot/"
	}
	if req.Pages <= 0 {
		req.Pages = 1
	}
	if req.Pages > 20 {
		req.Pages = 20
	}

	go func(catURL string, pages int) {
		crawlerBin := filepath.Join("..", "crawler", "crawler.exe")
		absBin, _ := filepath.Abs(crawlerBin)

		cmd := exec.Command(absBin,
			"-catalog",
			"-url", catURL,
			"-pages", strconv.Itoa(pages),
			"-db", s.dbConn,
		)
		_ = cmd.Run()
	}(req.CatalogURL, req.Pages)

	jsonResponse(w, http.StatusAccepted, map[string]string{
		"message": fmt.Sprintf("Started scanning %d catalog page(s) for novel list into queue.", req.Pages),
	})
}

func (s *Server) handleGetQueue(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	limit := 30
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	var counts struct {
		Pending   int `json:"pending"`
		Crawling  int `json:"crawling"`
		Completed int `json:"completed"`
		Failed    int `json:"failed"`
		Total     int `json:"total"`
	}

	_ = s.db.QueryRow("SELECT COUNT(*) FROM crawl_queue WHERE status = 'pending'").Scan(&counts.Pending)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM crawl_queue WHERE status = 'crawling'").Scan(&counts.Crawling)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM crawl_queue WHERE status = 'completed'").Scan(&counts.Completed)
	_ = s.db.QueryRow("SELECT COUNT(*) FROM crawl_queue WHERE status = 'failed'").Scan(&counts.Failed)
	counts.Total = counts.Pending + counts.Crawling + counts.Completed + counts.Failed

	type QueueItem struct {
		ID              string    `json:"id"`
		Title           string    `json:"title"`
		Slug            string    `json:"slug"`
		SourceURL       string    `json:"source_url"`
		AuthorName      string    `json:"author_name"`
		CoverURL        string    `json:"cover_url"`
		Status          string    `json:"status"`
		CrawledChapters int       `json:"crawled_chapters"`
		ErrorMessage    string    `json:"error_message"`
		UpdatedAt       time.Time `json:"updated_at"`
	}

	var rows *sql.Rows
	var err error
	if statusFilter != "" {
		rows, err = s.db.Query(`
			SELECT id, title, slug, source_url, COALESCE(author_name, ''), COALESCE(cover_url, ''), status, crawled_chapters, COALESCE(error_message, ''), updated_at
			FROM crawl_queue
			WHERE status = $1
			ORDER BY updated_at DESC
			LIMIT $2
		`, statusFilter, limit)
	} else {
		rows, err = s.db.Query(`
			SELECT id, title, slug, source_url, COALESCE(author_name, ''), COALESCE(cover_url, ''), status, crawled_chapters, COALESCE(error_message, ''), updated_at
			FROM crawl_queue
			ORDER BY 
				CASE status 
					WHEN 'crawling' THEN 1 
					WHEN 'pending' THEN 2 
					WHEN 'failed' THEN 3 
					ELSE 4 
				END,
				updated_at DESC
			LIMIT $1
		`, limit)
	}

	items := make([]QueueItem, 0)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var q QueueItem
			if err := rows.Scan(&q.ID, &q.Title, &q.Slug, &q.SourceURL, &q.AuthorName, &q.CoverURL, &q.Status, &q.CrawledChapters, &q.ErrorMessage, &q.UpdatedAt); err == nil {
				items = append(items, q)
			}
		}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"counts":         counts,
		"worker_running": s.workerRunning,
		"items":          items,
	})
}

func (s *Server) handleStartQueueWorker(w http.ResponseWriter, r *http.Request) {
	s.jobMutex.Lock()
	if s.workerRunning {
		s.jobMutex.Unlock()
		jsonResponse(w, http.StatusOK, map[string]string{"message": "Queue worker is already running."})
		return
	}
	s.workerRunning = true
	s.workerStop = make(chan struct{})
	s.jobMutex.Unlock()

	var req struct {
		LimitPerNovel int `json:"limit"`
		DelayMs       int `json:"delay"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.LimitPerNovel <= 0 {
		req.LimitPerNovel = 5
	}
	if req.DelayMs <= 0 {
		req.DelayMs = 400
	}

	go s.runQueueWorker(req.LimitPerNovel, req.DelayMs)

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Queue worker started successfully.",
		"status":  "running",
	})
}

func (s *Server) handleStopQueueWorker(w http.ResponseWriter, r *http.Request) {
	s.jobMutex.Lock()
	defer s.jobMutex.Unlock()
	if !s.workerRunning {
		jsonResponse(w, http.StatusOK, map[string]string{"message": "Queue worker is not running."})
		return
	}
	close(s.workerStop)
	s.workerRunning = false

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Queue worker stop signal dispatched."})
}

func (s *Server) handleResetFailedQueue(w http.ResponseWriter, r *http.Request) {
	res, err := s.db.Exec("UPDATE crawl_queue SET status = 'pending', error_message = NULL, updated_at = NOW() WHERE status = 'failed'")
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	rows, _ := res.RowsAffected()
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Reset %d failed novels back to pending status.", rows),
	})
}

func (s *Server) runQueueWorker(limitPerNovel int, delayMs int) {
	log.Println("🚀 Queue Worker started in background...")
	defer func() {
		s.jobMutex.Lock()
		s.workerRunning = false
		s.jobMutex.Unlock()
		log.Println("🏁 Queue Worker exited.")
	}()

	for {
		select {
		case <-s.workerStop:
			return
		default:
		}

		var qID, title, slug, sourceURL string
		err := s.db.QueryRow(`
			SELECT id, title, slug, source_url 
			FROM crawl_queue 
			WHERE status = 'pending' 
			ORDER BY created_at ASC 
			LIMIT 1
		`).Scan(&qID, &title, &slug, &sourceURL)

		if err != nil {
			// No more pending novels in queue
			log.Println("ℹ️ No more pending novels in queue.")
			return
		}

		// Update to 'crawling'
		_, _ = s.db.Exec("UPDATE crawl_queue SET status = 'crawling', updated_at = NOW() WHERE id = $1", qID)
		log.Printf("⏳ [Queue Worker] Crawling novel: %s (%s)...", title, sourceURL)

		crawlerBin := filepath.Join("..", "crawler", "crawler.exe")
		absBin, _ := filepath.Abs(crawlerBin)

		cmd := exec.Command(absBin,
			"-url", sourceURL,
			"-limit", strconv.Itoa(limitPerNovel),
			"-delay", strconv.Itoa(delayMs),
			"-db", s.dbConn,
		)
		output, crawlErr := cmd.CombinedOutput()

		if crawlErr != nil {
			log.Printf("❌ [Queue Worker] Failed novel %s: %v", title, crawlErr)
			errMsg := fmt.Sprintf("%v: %s", crawlErr, string(output))
			if len(errMsg) > 500 {
				errMsg = errMsg[:500]
			}
			_, _ = s.db.Exec("UPDATE crawl_queue SET status = 'failed', error_message = $1, updated_at = NOW() WHERE id = $2", errMsg, qID)
		} else {
			// Query actual chapter count
			var actualChapters int
			_ = s.db.QueryRow("SELECT COUNT(*) FROM chapters c JOIN novels n ON n.id = c.novel_id WHERE n.slug = $1", slug).Scan(&actualChapters)
			log.Printf("✅ [Queue Worker] Completed novel %s (%d chapters)!", title, actualChapters)
			_, _ = s.db.Exec("UPDATE crawl_queue SET status = 'completed', crawled_chapters = $1, error_message = NULL, updated_at = NOW() WHERE id = $2", actualChapters, qID)
		}

		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}
}
