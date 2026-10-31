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

type FeedbackRepository struct {
	db       *sql.DB
	filePath string
	mu       sync.Mutex
}

func NewFeedbackRepository(db *sql.DB, storageDir string) *FeedbackRepository {
	fPath := filepath.Join(storageDir, "feedback.json")
	return &FeedbackRepository{
		db:       db,
		filePath: fPath,
	}
}

func (r *FeedbackRepository) CreateFeedback(fb *domain.Feedback) error {
	fb.CreatedAt = time.Now()
	if r.db != nil && r.db.Ping() == nil {
		query := `
			INSERT INTO feedback (name, contact, feedback_type, content, created_at)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id;
		`
		var id string
		err := r.db.QueryRow(query, fb.Name, fb.Contact, fb.FeedbackType, fb.Content, fb.CreatedAt).Scan(&id)
		if err == nil {
			fb.ID = id
			_ = r.saveToDisk(fb)
			return nil
		}
	}

	fb.ID = fmt.Sprintf("fb_%d", time.Now().UnixNano())
	return r.saveToDisk(fb)
}

func (r *FeedbackRepository) saveToDisk(fb *domain.Feedback) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_ = os.MkdirAll(filepath.Dir(r.filePath), 0755)
	var list []domain.Feedback
	if data, err := os.ReadFile(r.filePath); err == nil {
		_ = json.Unmarshal(data, &list)
	}
	list = append([]domain.Feedback{*fb}, list...)
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, 0644)
}

func (r *FeedbackRepository) GetAllFeedback() ([]domain.Feedback, error) {
	if r.db != nil && r.db.Ping() == nil {
		rows, err := r.db.Query("SELECT id, COALESCE(name, ''), COALESCE(contact, ''), COALESCE(feedback_type, ''), content, created_at FROM feedback ORDER BY created_at DESC LIMIT 100")
		if err == nil {
			defer rows.Close()
			var list []domain.Feedback
			for rows.Next() {
				var item domain.Feedback
				if err := rows.Scan(&item.ID, &item.Name, &item.Contact, &item.FeedbackType, &item.Content, &item.CreatedAt); err == nil {
					list = append(list, item)
				}
			}
			return list, nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	var list []domain.Feedback
	if data, err := os.ReadFile(r.filePath); err == nil {
		_ = json.Unmarshal(data, &list)
	}
	return list, nil
}

func (r *FeedbackRepository) DeleteFeedback(id string) error {
	if r.db != nil && r.db.Ping() == nil {
		_, err := r.db.Exec("DELETE FROM feedback WHERE id = $1", id)
		if err == nil {
			_ = r.deleteFromDisk(id)
			return nil
		}
	}
	return r.deleteFromDisk(id)
}

func (r *FeedbackRepository) deleteFromDisk(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var list []domain.Feedback
	if data, err := os.ReadFile(r.filePath); err == nil {
		_ = json.Unmarshal(data, &list)
	}
	var filtered []domain.Feedback
	for _, it := range list {
		if it.ID != id {
			filtered = append(filtered, it)
		}
	}
	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, 0644)
}

