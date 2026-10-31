package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) handleGetSources(w http.ResponseWriter, r *http.Request) {
	sources, err := s.sourceRepo.GetAllSources()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, sources)
}

func (s *Server) handleToggleSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string `json:"id"`
		IsActive bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Missing source ID"})
		return
	}

	if err := s.sourceRepo.ToggleSource(req.ID, req.IsActive); err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message":   "Cập nhật trạng thái nguồn thành công",
		"id":        req.ID,
		"is_active": req.IsActive,
	})
}

func (s *Server) handleScanSource(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Missing source ID"})
		return
	}

	go func(srcID string) {
		inserted, err := s.scheduler.ScanSourceNow(srcID)
		if err == nil {
			fmt.Printf("🔍 Manual scan completed for source %s: %d new novels queued.\n", srcID, inserted)
		}
	}(req.ID)

	jsonResponse(w, http.StatusAccepted, map[string]string{
		"message": "Đã bắt đầu quét nguồn truyện ngay lập tức. Dữ liệu mới sẽ tự động nạp vào hàng đợi và cào ngầm!",
	})
}
