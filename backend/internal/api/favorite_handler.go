package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) getUserIDFromRequest(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	userID := strings.TrimPrefix(authHeader, "Bearer token_")
	userID = strings.TrimPrefix(userID, "Bearer ")
	userID = strings.TrimSpace(userID)
	if userID == "" || userID == authHeader {
		userID = strings.TrimSpace(r.URL.Query().Get("user_id"))
	}
	return userID
}

// handleToggleFavorite toggles bookmark/favorite for a user
func (s *Server) handleToggleFavorite(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID     string `json:"user_id"`
		NovelSlug  string `json:"novel_slug"`
		NovelTitle string `json:"novel_title"`
		NovelCover string `json:"novel_cover"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Dữ liệu JSON không hợp lệ."})
		return
	}

	userID := input.UserID
	if userID == "" {
		userID = s.getUserIDFromRequest(r)
	}

	if userID == "" {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{
			"error": "Đạo hữu vui lòng đăng nhập để lưu truyện vào bộ yêu thích.",
		})
		return
	}

	if input.NovelSlug == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Thiếu mã định danh truyện (slug)."})
		return
	}

	isFav, err := s.favoriteRepo.ToggleFavorite(userID, input.NovelSlug, input.NovelTitle, input.NovelCover)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	msg := "Đã thêm bộ truyện vào Tủ Sách Yêu Thích!"
	if !isFav {
		msg = "Đã bỏ truyện khỏi Tủ Sách Yêu Thích."
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"favorited": isFav,
		"message":   msg,
	})
}

// handleGetFavorites returns user's favorited novels
func (s *Server) handleGetFavorites(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserIDFromRequest(r)
	if userID == "" {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{
			"error": "Đạo hữu vui lòng đăng nhập để xem danh sách yêu thích.",
		})
		return
	}

	list, err := s.favoriteRepo.GetFavoritesByUser(userID)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data":  list,
		"total": len(list),
	})
}

// handleCheckFavorite checks if a novel is favorited by the user
func (s *Server) handleCheckFavorite(w http.ResponseWriter, r *http.Request) {
	userID := s.getUserIDFromRequest(r)
	slug := strings.TrimSpace(r.URL.Query().Get("novel_slug"))
	if userID == "" || slug == "" {
		jsonResponse(w, http.StatusOK, map[string]bool{"favorited": false})
		return
	}

	isFav := s.favoriteRepo.IsFavorited(userID, slug)
	jsonResponse(w, http.StatusOK, map[string]bool{"favorited": isFav})
}
