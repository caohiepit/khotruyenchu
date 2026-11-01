package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type ReadChapterRequest struct {
	UserID    string `json:"user_id"`
	NovelSlug string `json:"novel_slug"`
	ChapterNo int    `json:"chapter_no"`
}

// handleRegister registers a new cultivator
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Dữ liệu JSON không hợp lệ."})
		return
	}

	user, err := s.userRepo.Register(req.Username, req.Email, req.Password)
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"message": "Đăng ký Đạo Hiệu thành công! Chào mừng Đạo Hữu bước vào con đường Tu Tiên.",
		"user":    user,
		"token":   "token_" + user.ID,
	})
}

// handleLogin logs in an existing cultivator
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Dữ liệu JSON không hợp lệ."})
		return
	}

	user, err := s.userRepo.Authenticate(req.Login, req.Password)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Đăng nhập thành công! Chào mừng Đạo Hữu quy nguyên.",
		"user":    user,
		"token":   "token_" + user.ID,
	})
}

// handleAdminLogin authenticates administrator
func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Dữ liệu JSON không hợp lệ."})
		return
	}

	admin, err := s.userRepo.AuthenticateAdmin(req.Username, req.Password)
	if err != nil {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	token := "admin_session_" + admin.ID
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Đăng nhập trang quản trị thành công!",
		"token":   token,
		"admin": map[string]interface{}{
			"id":       admin.ID,
			"username": admin.Username,
			"role":     admin.Role,
			"email":    admin.Email,
		},
	})
}

// handleAdminVerify verifies if admin session is valid
func (s *Server) handleAdminVerify(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if !strings.HasPrefix(token, "admin_session_") {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "Phiên đăng nhập quản trị không hợp lệ."})
		return
	}

	adminID := strings.TrimPrefix(token, "admin_session_")
	admin, err := s.userRepo.GetByID(adminID)
	if err != nil || admin.Role != "admin" {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "Tài khoản không có quyền quản trị viên."})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"valid": true,
		"admin": map[string]interface{}{
			"id":       admin.ID,
			"username": admin.Username,
			"role":     admin.Role,
			"email":    admin.Email,
		},
	})
}

// handleGetMe returns current user info
func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	userID := strings.TrimPrefix(authHeader, "Bearer token_")
	if userID == "" || userID == authHeader {
		userID = r.URL.Query().Get("user_id")
	}

	if userID == "" {
		jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "Chưa đăng nhập."})
		return
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		jsonResponse(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, user)
}

// handleUserReadChapter grants cultivation exp to reader upon completing a chapter
func (s *Server) handleUserReadChapter(w http.ResponseWriter, r *http.Request) {
	var req ReadChapterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Thiếu thông tin người dùng."})
		return
	}

	// +25 tu vi per chapter read
	expGained := 25
	updatedUser, err := s.userRepo.AddExp(req.UserID, expGained)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"exp_gained": expGained,
		"user":       updatedUser,
		"message":    "Đọc xong chương: Tu vi tăng thêm 25 điểm!",
	})
}

