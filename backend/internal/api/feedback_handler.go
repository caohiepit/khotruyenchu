package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"comics-backend/internal/domain"
)

func (s *Server) handleCreateFeedback(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name         string `json:"name"`
		Contact      string `json:"contact"`
		FeedbackType string `json:"feedback_type"`
		Content      string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Dữ liệu không hợp lệ"})
		return
	}

	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Vui lòng nhập nội dung truyền âm / góp ý"})
		return
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = "Đạo hữu ẩn danh"
	}
	fType := strings.TrimSpace(input.FeedbackType)
	if fType == "" {
		fType = "Góp ý chung"
	}

	fb := &domain.Feedback{
		Name:         name,
		Contact:      strings.TrimSpace(input.Contact),
		FeedbackType: fType,
		Content:      input.Content,
	}

	if err := s.feedbackRepo.CreateFeedback(fb); err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "Lỗi lưu ý kiến: " + err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Phi thư đã được truyền đi thành công! Đa tạ đạo hữu đã quan tâm và góp ý cho KhoTruyenChu.",
		"id":      fb.ID,
	})
}

func (s *Server) handleGetFeedback(w http.ResponseWriter, r *http.Request) {
	list, err := s.feedbackRepo.GetAllFeedback()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data":  list,
		"total": len(list),
	})
}

func (s *Server) handleDeleteFeedback(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Thiếu mã ý kiến ID."})
		return
	}

	if err := s.feedbackRepo.DeleteFeedback(id); err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Đã xóa ý kiến đóng góp thành công.",
	})
}


