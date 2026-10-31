package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"comics-backend/internal/domain"
)

// handleGetAdminAffiliates returns all affiliate products
func (s *Server) handleGetAdminAffiliates(w http.ResponseWriter, r *http.Request) {
	items, err := s.affiliateRepo.GetAll()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, items)
}

// handleCreateAffiliate creates a new affiliate link
func (s *Server) handleCreateAffiliate(w http.ResponseWriter, r *http.Request) {
	var item domain.AffiliateProduct
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Dữ liệu JSON không hợp lệ."})
		return
	}

	item.Title = strings.TrimSpace(item.Title)
	item.AffiliateURL = strings.TrimSpace(item.AffiliateURL)
	if item.Title == "" || item.AffiliateURL == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Vui lòng nhập tiêu đề và link Shopee."})
		return
	}

	if item.ImageURL == "" {
		item.ImageURL = "https://cf.shopee.vn/file/vn-11134207-7r98o-ls5h19875e5f0d"
	}
	if item.Position == "" {
		item.Position = "ticker"
	}

	if err := s.affiliateRepo.Create(&item); err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusCreated, item)
}

// handleUpdateAffiliate updates or toggles an affiliate product
func (s *Server) handleUpdateAffiliate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Thiếu mã liên kết ID."})
		return
	}

	var item domain.AffiliateProduct
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Dữ liệu JSON không hợp lệ."})
		return
	}
	item.ID = id

	if err := s.affiliateRepo.Update(&item); err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Cập nhật liên kết Shopee thành công!",
		"item":    item,
	})
}

// handleDeleteAffiliate deletes an affiliate link
func (s *Server) handleDeleteAffiliate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"error": "Thiếu mã liên kết ID."})
		return
	}

	if err := s.affiliateRepo.Delete(id); err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Đã xóa liên kết Shopee thành công."})
}

// handleGetPublicAffiliates returns active links for Client UI
func (s *Server) handleGetPublicAffiliates(w http.ResponseWriter, r *http.Request) {
	items, err := s.affiliateRepo.GetActive()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, items)
}

// handleClickAffiliate records click and returns target URL
func (s *Server) handleClickAffiliate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id != "" {
		_ = s.affiliateRepo.RecordClick(id)
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "recorded"})
}

