package storage

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"comics-backend/internal/domain"
)

type FavoriteRepository struct {
	db       *sql.DB
	filePath string
	mu       sync.RWMutex
	items    []domain.Favorite
}

func NewFavoriteRepository(db *sql.DB, storageDir string) *FavoriteRepository {
	fPath := filepath.Join(storageDir, "favorites.json")
	repo := &FavoriteRepository{
		db:       db,
		filePath: fPath,
		items:    make([]domain.Favorite, 0),
	}
	repo.loadFromDisk()
	return repo
}

func (r *FavoriteRepository) dbAvailable() bool {
	return r.db != nil && r.db.Ping() == nil
}

func (r *FavoriteRepository) loadFromDisk() {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return
	}
	var list []domain.Favorite
	if err := json.Unmarshal(data, &list); err == nil {
		r.items = list
	}
}

func (r *FavoriteRepository) saveToDiskLocked() {
	_ = os.MkdirAll(filepath.Dir(r.filePath), 0755)
	data, err := json.MarshalIndent(r.items, "", "  ")
	if err == nil {
		_ = os.WriteFile(r.filePath, data, 0644)
	}
}

// ToggleFavorite adds or removes a favorite for a user. Returns (isFavorited, error)
func (r *FavoriteRepository) ToggleFavorite(userID, novelSlug, novelTitle, novelCover string) (bool, error) {
	userID = strings.TrimSpace(userID)
	novelSlug = strings.TrimSpace(novelSlug)
	if userID == "" || novelSlug == "" {
		return false, sql.ErrNoRows
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check in-memory list
	existingIdx := -1
	for i, f := range r.items {
		if f.UserID == userID && f.NovelSlug == novelSlug {
			existingIdx = i
			break
		}
	}

	if existingIdx >= 0 {
		// Already favorited -> remove it (unfavorite)
		r.items = append(r.items[:existingIdx], r.items[existingIdx+1:]...)
		r.saveToDiskLocked()

		if r.dbAvailable() {
			_, _ = r.db.Exec("DELETE FROM user_favorites WHERE user_id = $1 AND novel_slug = $2", userID, novelSlug)
		}
		return false, nil
	}

	// Not yet favorited -> add it
	fav := domain.Favorite{
		ID:         int(time.Now().UnixNano() % 10000000),
		UserID:     userID,
		NovelSlug:  novelSlug,
		NovelTitle: novelTitle,
		NovelCover: novelCover,
		CreatedAt:  time.Now(),
	}

	if r.dbAvailable() {
		var newID int
		err := r.db.QueryRow(`
			INSERT INTO user_favorites (user_id, novel_slug, novel_title, novel_cover, created_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (user_id, novel_slug) DO UPDATE SET novel_title = EXCLUDED.novel_title, novel_cover = EXCLUDED.novel_cover
			RETURNING id;
		`, fav.UserID, fav.NovelSlug, fav.NovelTitle, fav.NovelCover, fav.CreatedAt).Scan(&newID)
		if err == nil {
			fav.ID = newID
		}
	}

	r.items = append([]domain.Favorite{fav}, r.items...)
	r.saveToDiskLocked()
	return true, nil
}

// GetFavoritesByUser returns all favorited novels for a user
func (r *FavoriteRepository) GetFavoritesByUser(userID string) ([]domain.Favorite, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return []domain.Favorite{}, nil
	}

	if r.dbAvailable() {
		rows, err := r.db.Query(`
			SELECT id, user_id, novel_slug, COALESCE(novel_title, ''), COALESCE(novel_cover, ''), created_at
			FROM user_favorites
			WHERE user_id = $1
			ORDER BY created_at DESC
		`, userID)
		if err == nil {
			defer rows.Close()
			var list []domain.Favorite
			for rows.Next() {
				var f domain.Favorite
				if err := rows.Scan(&f.ID, &f.UserID, &f.NovelSlug, &f.NovelTitle, &f.NovelCover, &f.CreatedAt); err == nil {
					list = append(list, f)
				}
			}
			return list, nil
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []domain.Favorite
	for _, f := range r.items {
		if f.UserID == userID {
			result = append(result, f)
		}
	}
	return result, nil
}

// IsFavorited checks if a novel is favorited by the user
func (r *FavoriteRepository) IsFavorited(userID, novelSlug string) bool {
	userID = strings.TrimSpace(userID)
	novelSlug = strings.TrimSpace(novelSlug)
	if userID == "" || novelSlug == "" {
		return false
	}

	if r.dbAvailable() {
		var exists bool
		err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM user_favorites WHERE user_id = $1 AND novel_slug = $2)", userID, novelSlug).Scan(&exists)
		if err == nil {
			return exists
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, f := range r.items {
		if f.UserID == userID && f.NovelSlug == novelSlug {
			return true
		}
	}
	return false
}

