package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"comics-backend/internal/domain"
)

type AffiliateRepository struct {
	db       *sql.DB
	memMu    sync.RWMutex
	memItems []domain.AffiliateProduct
	jsonPath string
}

func NewAffiliateRepository(db *sql.DB, dataDir ...string) *AffiliateRepository {
	dir := "storage"
	if len(dataDir) > 0 && dataDir[0] != "" {
		dir = dataDir[0]
	}
	repo := &AffiliateRepository{
		db:       db,
		jsonPath: filepath.Join(dir, "affiliates.json"),
		memItems: make([]domain.AffiliateProduct, 0),
	}
	repo.loadFromJSON()
	if len(repo.memItems) == 0 {
		repo.seedDefaultAffiliates()
	}
	repo.syncToDBIfEmpty()
	return repo
}

func (r *AffiliateRepository) syncToDBIfEmpty() {
	if !r.dbAvailable() {
		return
	}
	var count int
	_ = r.db.QueryRow("SELECT COUNT(*) FROM affiliate_products").Scan(&count)
	if count == 0 {
		for _, item := range r.memItems {
			_, _ = r.db.Exec(`
				INSERT INTO affiliate_products (id, title, affiliate_url, image_url, tag, category, is_active, clicks_count, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				ON CONFLICT (id) DO NOTHING
			`, item.ID, item.Title, item.AffiliateURL, item.ImageURL, item.Tag, item.Position, item.IsActive, item.ClicksCount, item.CreatedAt)
		}
	}
}

func (r *AffiliateRepository) dbAvailable() bool {
	return r.db != nil && r.db.Ping() == nil
}

func (r *AffiliateRepository) seedDefaultAffiliates() {
	r.memMu.Lock()
	defer r.memMu.Unlock()
	now := time.Now()
	r.memItems = []domain.AffiliateProduct{
		{
			ID:            "aff-1",
			Title:         "Bàn Phím Cơ Tu Tiên Led RGB Switch Êm Ái",
			AffiliateURL:  "https://shopee.vn/search?keyword=ban%20phim%20co",
			ImageURL:      "https://cf.shopee.vn/file/vn-11134207-7r98o-ls5h19875e5f0d",
			PriceOriginal: "450.000đ",
			PriceDiscount: "289.000đ",
			Tag:           "Flash Sale",
			Category:      "electronics",
			Position:      "ticker",
			IsActive:      true,
			ClicksCount:   24,
			CreatedAt:     now,
		},
		{
			ID:            "aff-2",
			Title:         "Trà Ô Long Thiết Quan Âm Hảo Hạng Đọc Truyện",
			AffiliateURL:  "https://shopee.vn/search?keyword=tra%20thiet%20quan%20am",
			ImageURL:      "https://cf.shopee.vn/file/vn-11134207-7r98o-lt0i23948381bb",
			PriceOriginal: "180.000đ",
			PriceDiscount: "125.000đ",
			Tag:           "Tu Tiên Trà",
			Category:      "tea",
			Position:      "ticker",
			IsActive:      true,
			ClicksCount:   18,
			CreatedAt:     now,
		},
		{
			ID:            "aff-3",
			Title:         "Đèn Đọc Sách Chống Cận Thị Bảo Vệ Mắt Ban Đêm",
			AffiliateURL:  "https://shopee.vn/search?keyword=den%20doc%20sach%20chong%20can",
			ImageURL:      "https://cf.shopee.vn/file/vn-11134207-7r98o-lt9j348572834a",
			PriceOriginal: "210.000đ",
			PriceDiscount: "149.000đ",
			Tag:           "Khuyên Dùng",
			Category:      "lamp",
			Position:      "ticker",
			IsActive:      true,
			ClicksCount:   35,
			CreatedAt:     now,
		},
	}
	r.saveToJSONLocked()
}

func (r *AffiliateRepository) loadFromJSON() {
	data, err := os.ReadFile(r.jsonPath)
	if err != nil {
		return
	}
	var items []domain.AffiliateProduct
	if err := json.Unmarshal(data, &items); err == nil {
		r.memItems = items
	}
}

func (r *AffiliateRepository) saveToJSONLocked() {
	_ = os.MkdirAll(filepath.Dir(r.jsonPath), 0755)
	data, err := json.MarshalIndent(r.memItems, "", "  ")
	if err == nil {
		_ = os.WriteFile(r.jsonPath, data, 0644)
	}
}

// GetAll returns all affiliate links for Admin management
func (r *AffiliateRepository) GetAll() ([]domain.AffiliateProduct, error) {
	if r.dbAvailable() {
		rows, err := r.db.Query(`
			SELECT id, title, affiliate_url, image_url, COALESCE(price_original::text, ''), 
			       COALESCE(price_discount::text, ''), COALESCE(tag, ''), COALESCE(category, ''), 
			       COALESCE(category, 'ticker'), is_active, clicks_count, created_at
			FROM affiliate_products
			ORDER BY created_at DESC
		`)
		if err == nil {
			defer rows.Close()
			items := make([]domain.AffiliateProduct, 0)
			for rows.Next() {
				var item domain.AffiliateProduct
				_ = rows.Scan(
					&item.ID, &item.Title, &item.AffiliateURL, &item.ImageURL,
					&item.PriceOriginal, &item.PriceDiscount, &item.Tag, &item.Category,
					&item.Position, &item.IsActive, &item.ClicksCount, &item.CreatedAt,
				)
				items = append(items, item)
			}
			if len(items) > 0 {
				return items, nil
			}
		}
	}

	r.memMu.RLock()
	defer r.memMu.RUnlock()
	res := make([]domain.AffiliateProduct, 0, len(r.memItems))
	for _, it := range r.memItems {
		res = append(res, it)
	}
	return res, nil
}

// GetActive returns only active links for Client UI
func (r *AffiliateRepository) GetActive() ([]domain.AffiliateProduct, error) {
	all, err := r.GetAll()
	if err != nil {
		return []domain.AffiliateProduct{}, err
	}
	active := make([]domain.AffiliateProduct, 0)
	for _, it := range all {
		if it.IsActive {
			active = append(active, it)
		}
	}
	return active, nil
}

// Create inserts a new affiliate link
func (r *AffiliateRepository) Create(item *domain.AffiliateProduct) error {
	if item.ID == "" {
		item.ID = fmt.Sprintf("aff-%d", time.Now().UnixNano()%1000000)
	}
	item.CreatedAt = time.Now()
	item.IsActive = true

	if r.dbAvailable() {
		_, _ = r.db.Exec(`
			INSERT INTO affiliate_products (id, title, affiliate_url, image_url, tag, category, is_active, clicks_count, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, item.ID, item.Title, item.AffiliateURL, item.ImageURL, item.Tag, item.Position, item.IsActive, item.ClicksCount, item.CreatedAt)
	}

	r.memMu.Lock()
	defer r.memMu.Unlock()
	r.memItems = append([]domain.AffiliateProduct{*item}, r.memItems...)
	r.saveToJSONLocked()
	return nil
}

// Update updates an affiliate link
func (r *AffiliateRepository) Update(item *domain.AffiliateProduct) error {
	if r.dbAvailable() {
		_, _ = r.db.Exec(`
			UPDATE affiliate_products
			SET title = $1, affiliate_url = $2, image_url = $3, tag = $4, is_active = $5
			WHERE id = $6
		`, item.Title, item.AffiliateURL, item.ImageURL, item.Tag, item.IsActive, item.ID)
	}

	r.memMu.Lock()
	defer r.memMu.Unlock()
	for i, it := range r.memItems {
		if it.ID == item.ID {
			if item.Title != "" {
				r.memItems[i].Title = item.Title
			}
			if item.AffiliateURL != "" {
				r.memItems[i].AffiliateURL = item.AffiliateURL
			}
			if item.ImageURL != "" {
				r.memItems[i].ImageURL = item.ImageURL
			}
			if item.PriceDiscount != "" {
				r.memItems[i].PriceDiscount = item.PriceDiscount
			}
			if item.Position != "" {
				r.memItems[i].Position = item.Position
			}
			r.memItems[i].IsActive = item.IsActive
			break
		}
	}
	r.saveToJSONLocked()
	return nil
}

// ToggleActive toggles active state
func (r *AffiliateRepository) ToggleActive(id string, active bool) error {
	if r.dbAvailable() {
		_, _ = r.db.Exec("UPDATE affiliate_products SET is_active = $1 WHERE id = $2", active, id)
	}
	r.memMu.Lock()
	defer r.memMu.Unlock()
	for i, it := range r.memItems {
		if it.ID == id {
			r.memItems[i].IsActive = active
			break
		}
	}
	r.saveToJSONLocked()
	return nil
}

// Delete removes an affiliate product
func (r *AffiliateRepository) Delete(id string) error {
	if r.dbAvailable() {
		_, _ = r.db.Exec("DELETE FROM affiliate_products WHERE id = $1", id)
	}
	r.memMu.Lock()
	defer r.memMu.Unlock()
	newItems := make([]domain.AffiliateProduct, 0, len(r.memItems))
	for _, it := range r.memItems {
		if it.ID != id {
			newItems = append(newItems, it)
		}
	}
	r.memItems = newItems
	r.saveToJSONLocked()
	return nil
}

// RecordClick increments click counter
func (r *AffiliateRepository) RecordClick(id string) error {
	if r.dbAvailable() {
		_, _ = r.db.Exec("UPDATE affiliate_products SET clicks_count = clicks_count + 1 WHERE id = $1", id)
	}
	r.memMu.Lock()
	defer r.memMu.Unlock()
	for i, it := range r.memItems {
		if it.ID == id {
			r.memItems[i].ClicksCount++
			break
		}
	}
	r.saveToJSONLocked()
	return nil
}
