package api

import (
	"encoding/json"
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var commentTagRegex = regexp.MustCompile(`(?i)<[^>]*>`)

// handleGetNovelComments returns paginated comments for a novel
func (s *Server) handleGetNovelComments(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if slug == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Thiếu mã định danh linh thư (slug)."})
		return
	}

	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	comments, total, err := s.commentRepo.GetCommentsByNovel(slug, limit, offset)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "Không thể tải danh sách bình luận: " + err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"novel_slug": slug,
		"comments":   comments,
		"total":      total,
	})
}

// handleCreateNovelComment creates a new comment on a novel (requires auth)
func (s *Server) handleCreateNovelComment(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	if slug == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Thiếu mã định danh linh thư (slug)."})
		return
	}

	// 1. Authenticate user
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	var userID string

	if strings.HasPrefix(token, "token_") {
		userID = strings.TrimPrefix(token, "token_")
	} else if strings.HasPrefix(token, "admin_session_") {
		userID = strings.TrimPrefix(token, "admin_session_")
	} else {
		userID = r.URL.Query().Get("user_id")
	}

	if userID == "" {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{
			"error": "Đạo hữu vui lòng đăng nhập để tham gia đàm đạo bình phẩm!",
		})
		return
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{
			"error": "Tài khoản không tồn tại hoặc phiên đăng nhập đã hết hạn.",
		})
		return
	}

	// 2. Decode payload
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Dữ liệu JSON không hợp lệ."})
		return
	}

	// 3. Clean & Validate Content
	cleanText := strings.TrimSpace(req.Content)
	// Strip HTML tags for security
	cleanText = commentTagRegex.ReplaceAllString(cleanText, "")
	cleanText = html.UnescapeString(cleanText)
	cleanText = strings.TrimSpace(cleanText)

	if len(cleanText) < 2 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Nội dung bình luận quá ngắn (tối thiểu 2 ký tự)."})
		return
	}
	if len(cleanText) > 1500 {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Nội dung bình luận quá dài (tối đa 1500 ký tự)."})
		return
	}

	// 4. Save to Repository
	avatar := user.AvatarURL
	if avatar == "" {
		avatar = "https://api.dicebear.com/7.x/bottts/svg?seed=" + user.Username
	}
	realm := user.Realm
	if realm == "" {
		realm = "Phàm Nhân"
	}

	comment, err := s.commentRepo.AddComment(slug, user.ID, user.Username, avatar, realm, user.Role, cleanText)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "Không thể lưu bình luận: " + err.Error()})
		return
	}

	// 5. Reward +10 Tu Vi exp for engaging in discussion
	_, _ = s.userRepo.AddExp(user.ID, 10)

	jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"message": "Đã gửi lời đàm đạo thành công! (+10 Tu Vi)",
		"comment": comment,
	})
}

// handleDeleteNovelComment removes a comment if user is owner or admin
func (s *Server) handleDeleteNovelComment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("comment_id")
	commentID, err := strconv.Atoi(idStr)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "ID bình luận không hợp lệ."})
		return
	}

	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	var userID string

	if strings.HasPrefix(token, "token_") {
		userID = strings.TrimPrefix(token, "token_")
	} else if strings.HasPrefix(token, "admin_session_") {
		userID = strings.TrimPrefix(token, "admin_session_")
	} else {
		userID = r.URL.Query().Get("user_id")
	}

	if userID == "" {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "Chưa đăng nhập."})
		return
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "Tài khoản không hợp lệ."})
		return
	}

	isAdmin := user.Role == "admin"
	if err := s.commentRepo.DeleteComment(commentID, user.ID, isAdmin); err != nil {
		jsonResponse(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Đã xóa bình luận thành công.",
	})
}

// handleLikeNovelComment increments like counter for a comment
func (s *Server) handleLikeNovelComment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("comment_id")
	commentID, err := strconv.Atoi(idStr)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "ID bình luận không hợp lệ."})
		return
	}

	newLikes, err := s.commentRepo.LikeComment(commentID)
	if err != nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"likes":   newLikes,
	})
}
