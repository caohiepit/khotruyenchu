package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"comics-backend/internal/domain"
)

type SubmitURLRequest struct {
	URL   string `json:"url"`
	Limit int    `json:"limit"` // 0 = all chapters
	Pages int    `json:"pages"` // For catalog URLs
}

func (s *Server) handleSubmitURL(w http.ResponseWriter, r *http.Request) {
	var req SubmitURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.URL) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Vui lòng nhập URL hợp lệ."})
		return
	}

	rawURL := strings.TrimSpace(req.URL)
	handler := s.registry.Resolve(rawURL)
	if handler == nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Không tìm thấy bộ phân tích phù hợp cho URL: %s", rawURL),
		})
		return
	}

	// Detect if it's a catalog / category URL
	isCatalog := strings.Contains(rawURL, "/danh-sach/") ||
		strings.Contains(rawURL, "/the-loai/") ||
		strings.Contains(rawURL, "/tong-hop") ||
		strings.Contains(rawURL, "/rank") ||
		strings.Contains(rawURL, "?page=")

	if isCatalog {
		pages := req.Pages
		if pages <= 0 {
			pages = 1
		}
		if pages > 10 {
			pages = 10
		}

		go func(catURL string, maxP int, hDomain string) {
			novels, err := handler.DiscoverNovels(catURL, maxP)
			if err == nil && len(novels) > 0 {
				_, _ = s.queueRepo.EnqueueBatch(novels)
				s.engine.WakeUp()
			}
		}(rawURL, pages, handler.Domain())

		jsonResponse(w, http.StatusAccepted, map[string]interface{}{
			"type":           "catalog",
			"handler":        handler.Name(),
			"message":        fmt.Sprintf("Đã bắt đầu quét danh mục (%d trang) từ 【%s】. Các truyện mới sẽ tự động được cào liên tục!", pages, handler.Name()),
			"worker_running": s.engine.IsRunning(),
		})
		return
	}

	// It's a single novel URL -> Extract preview info & Enqueue
	go func(novelURL string, limitChapters int) {
		novel, _, err := handler.ParseNovel(novelURL)
		var item domain.CrawlQueueItem
		if err == nil && novel != nil {
			item = domain.CrawlQueueItem{
				Title:         novel.Title,
				Slug:          novel.Slug,
				SourceURL:     novelURL,
				SourceDomain:  handler.Domain(),
				AuthorName:    novel.AuthorName,
				CoverURL:      novel.CoverURL,
				Status:        "pending",
				TotalChapters: limitChapters,
			}
		} else {
			// Fallback with URL slug
			slug := strings.Trim(novelURL, "/")
			parts := strings.Split(slug, "/")
			lastPart := parts[len(parts)-1]
			item = domain.CrawlQueueItem{
				Title:         lastPart,
				Slug:          lastPart,
				SourceURL:     novelURL,
				SourceDomain:  handler.Domain(),
				Status:        "pending",
				TotalChapters: limitChapters,
			}
		}

		_ = s.queueRepo.EnqueueNovel(&item)

		// ⚡ Wake up worker instantly so it starts crawling without manual trigger!
		s.engine.WakeUp()
	}(rawURL, req.Limit)

	jsonResponse(w, http.StatusAccepted, map[string]interface{}{
		"type":           "novel",
		"handler":        handler.Name(),
		"message":        fmt.Sprintf("Đã tạo job cào thành công cho 【%s】! Worker nền đang tự động xử lý liên tục mà không cần trigger tay.", handler.Name()),
		"worker_running": s.engine.IsRunning(),
	})
}

func (s *Server) handleGetQueue(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	limit := 30
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	counts, _ := s.queueRepo.GetCounts()
	items, err := s.queueRepo.GetQueueList(statusFilter, limit)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"counts":         counts,
		"worker_running": s.engine.IsRunning(),
		"current_job":    s.engine.GetCurrentJob(),
		"live_status":    s.engine.GetLiveStatus(),
		"items":          items,
	})
}

func (s *Server) handleResetFailedQueue(w http.ResponseWriter, r *http.Request) {
	rows, err := s.queueRepo.ResetFailed()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if rows > 0 {
		s.engine.WakeUp()
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("Đã đặt lại %d truyện lỗi về trạng thái chờ (pending). Worker sẽ tự động cào lại!", rows),
	})
}

func (s *Server) handleToggleWorker(w http.ResponseWriter, r *http.Request) {
	if s.engine.IsRunning() {
		s.engine.Stop()
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message":        "Đã tạm dừng Worker nền.",
			"worker_running": false,
		})
	} else {
		s.engine.Start()
		s.engine.WakeUp()
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message":        "Đã khởi động lại Worker nền và tự động cào tiếp hàng đợi.",
			"worker_running": true,
		})
	}
}
