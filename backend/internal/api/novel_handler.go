package api

import (
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.queueRepo.GetDashboardStats()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	stats.WorkerRunning = s.engine.IsRunning()
	jsonResponse(w, http.StatusOK, stats)
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

	items, total, err := s.novelRepo.GetNovels(search, page, limit)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	totalPages := (total + limit - 1) / limit
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data":        items,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

func (s *Server) handleGetNovelDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Missing novel ID or slug"})
		return
	}

	novel, chapters, err := s.novelRepo.GetNovelDetail(id)
	if err != nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "Novel not found"})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"novel":    novel,
		"chapters": chapters,
	})
}

func (s *Server) handleGetChapterContent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Missing chapter ID"})
		return
	}

	chap, err := s.novelRepo.GetChapterContent(id)
	if err != nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": "Chapter content not found"})
		return
	}

	jsonResponse(w, http.StatusOK, chap)
}

func (s *Server) handleGetGenres(w http.ResponseWriter, r *http.Request) {
	genres, err := s.novelRepo.GetGenres()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, genres)
}
