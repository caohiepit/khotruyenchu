package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"comics-backend/internal/domain"
)

type CommentRepository struct {
	db       *sql.DB
	filePath string
	mu       sync.RWMutex
	items    []domain.Comment
	nextID   int
}

func NewCommentRepository(db *sql.DB, storageDir string) *CommentRepository {
	fPath := filepath.Join(storageDir, "comments.json")
	repo := &CommentRepository{
		db:       db,
		filePath: fPath,
		items:    make([]domain.Comment, 0),
		nextID:   1,
	}
	repo.loadFromDisk()
	return repo
}

func (r *CommentRepository) dbAvailable() bool {
	return r.db != nil && r.db.Ping() == nil
}

func (r *CommentRepository) loadFromDisk() {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return
	}
	var list []domain.Comment
	if err := json.Unmarshal(data, &list); err == nil {
		r.items = list
		for _, c := range list {
			if c.ID >= r.nextID {
				r.nextID = c.ID + 1
			}
		}
	}
}

func (r *CommentRepository) saveToDiskLocked() {
	_ = os.MkdirAll(filepath.Dir(r.filePath), 0755)
	data, err := json.MarshalIndent(r.items, "", "  ")
	if err == nil {
		_ = os.WriteFile(r.filePath, data, 0644)
	}
}

// AddComment inserts a new comment for a novel
func (r *CommentRepository) AddComment(novelSlug, userID, username, userAvatar, userRealm, role, content string) (*domain.Comment, error) {
	now := time.Now()

	if r.dbAvailable() {
		var id int
		query := `
			INSERT INTO comments (novel_slug, user_id, username, user_avatar, user_realm, role, content, likes, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 0, $8)
			RETURNING id;
		`
		err := r.db.QueryRow(query, novelSlug, userID, username, userAvatar, userRealm, role, content, now).Scan(&id)
		if err == nil {
			newComment := &domain.Comment{
				ID:         id,
				NovelSlug:  novelSlug,
				UserID:     userID,
				Username:   username,
				UserAvatar: userAvatar,
				UserRealm:  userRealm,
				Role:       role,
				Content:    content,
				Likes:      0,
				CreatedAt:  now,
			}
			// Keep local memory/disk in sync
			r.mu.Lock()
			r.items = append(r.items, *newComment)
			r.saveToDiskLocked()
			r.mu.Unlock()
			return newComment, nil
		}
	}

	// JSON fallback
	r.mu.Lock()
	defer r.mu.Unlock()

	newComment := domain.Comment{
		ID:         r.nextID,
		NovelSlug:  novelSlug,
		UserID:     userID,
		Username:   username,
		UserAvatar: userAvatar,
		UserRealm:  userRealm,
		Role:       role,
		Content:    content,
		Likes:      0,
		CreatedAt:  now,
	}
	r.nextID++
	r.items = append(r.items, newComment)
	r.saveToDiskLocked()

	return &newComment, nil
}

// GetCommentsByNovel retrieves comments for a novel ordered by newest first
func (r *CommentRepository) GetCommentsByNovel(novelSlug string, limit, offset int) ([]domain.Comment, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	if r.dbAvailable() {
		var total int
		err := r.db.QueryRow("SELECT COUNT(*) FROM comments WHERE novel_slug = $1", novelSlug).Scan(&total)
		if err == nil {
			rows, err := r.db.Query(`
				SELECT id, novel_slug, user_id, username, user_avatar, user_realm, role, content, likes, created_at
				FROM comments
				WHERE novel_slug = $1
				ORDER BY created_at DESC
				LIMIT $2 OFFSET $3
			`, novelSlug, limit, offset)
			if err == nil {
				defer rows.Close()
				var results []domain.Comment
				for rows.Next() {
					var c domain.Comment
					if err := rows.Scan(&c.ID, &c.NovelSlug, &c.UserID, &c.Username, &c.UserAvatar, &c.UserRealm, &c.Role, &c.Content, &c.Likes, &c.CreatedAt); err == nil {
						results = append(results, c)
					}
				}
				return results, total, nil
			}
		}
	}

	// JSON fallback
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []domain.Comment
	for _, c := range r.items {
		if c.NovelSlug == novelSlug {
			matched = append(matched, c)
		}
	}

	// Sort newest first
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})

	total := len(matched)
	if offset >= total {
		return []domain.Comment{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return matched[offset:end], total, nil
}

// DeleteComment removes a comment by ID if user is owner or admin
func (r *CommentRepository) DeleteComment(commentID int, userID string, isAdmin bool) error {
	if r.dbAvailable() {
		var ownerID string
		err := r.db.QueryRow("SELECT user_id FROM comments WHERE id = $1", commentID).Scan(&ownerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("bình luận không tồn tại")
			}
			return err
		}

		if !isAdmin && ownerID != userID {
			return errors.New("bạn không có quyền xóa bình luận này")
		}

		_, err = r.db.Exec("DELETE FROM comments WHERE id = $1", commentID)
		if err != nil {
			return err
		}

		// Also remove from local list
		r.mu.Lock()
		for i, c := range r.items {
			if c.ID == commentID {
				r.items = append(r.items[:i], r.items[i+1:]...)
				r.saveToDiskLocked()
				break
			}
		}
		r.mu.Unlock()
		return nil
	}

	// JSON fallback
	r.mu.Lock()
	defer r.mu.Unlock()

	foundIdx := -1
	for i, c := range r.items {
		if c.ID == commentID {
			if !isAdmin && c.UserID != userID {
				return errors.New("bạn không có quyền xóa bình luận này")
			}
			foundIdx = i
			break
		}
	}

	if foundIdx == -1 {
		return errors.New("bình luận không tồn tại")
	}

	r.items = append(r.items[:foundIdx], r.items[foundIdx+1:]...)
	r.saveToDiskLocked()
	return nil
}

// LikeComment increments the like counter for a comment
func (r *CommentRepository) LikeComment(commentID int) (int, error) {
	if r.dbAvailable() {
		var newLikes int
		err := r.db.QueryRow("UPDATE comments SET likes = likes + 1 WHERE id = $1 RETURNING likes", commentID).Scan(&newLikes)
		if err == nil {
			r.mu.Lock()
			for i := range r.items {
				if r.items[i].ID == commentID {
					r.items[i].Likes = newLikes
					r.saveToDiskLocked()
					break
				}
			}
			r.mu.Unlock()
			return newLikes, nil
		}
	}

	// JSON fallback
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.items {
		if r.items[i].ID == commentID {
			r.items[i].Likes++
			newLikes := r.items[i].Likes
			r.saveToDiskLocked()
			return newLikes, nil
		}
	}

	return 0, errors.New("bình luận không tồn tại")
}
