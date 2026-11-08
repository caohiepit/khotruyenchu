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

// AddComment inserts a new comment or reply for a novel
// parentID: 0 for root comments, >0 for replies. Enforces maximum 1 level of nesting.
func (r *CommentRepository) AddComment(novelSlug, userID, username, userAvatar, role, content string, parentID int) (*domain.Comment, error) {
	now := time.Now()

	// If parentID > 0, verify root parent exists and clamp to 1 level
	if parentID > 0 {
		actualParentID := parentID
		if r.dbAvailable() {
			var pID, pParentID int
			err := r.db.QueryRow("SELECT id, COALESCE(parent_id, 0) FROM comments WHERE id = $1 AND novel_slug = $2", parentID, novelSlug).Scan(&pID, &pParentID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, errors.New("bình luận gốc không tồn tại")
				}
				return nil, err
			}
			// If replying to an existing reply, fold into the same parent root (strictly 1 level)
			if pParentID > 0 {
				actualParentID = pParentID
			}
		} else {
			r.mu.RLock()
			var found bool
			for _, item := range r.items {
				if item.ID == parentID && item.NovelSlug == novelSlug {
					found = true
					if item.ParentID > 0 {
						actualParentID = item.ParentID
					}
					break
				}
			}
			r.mu.RUnlock()
			if !found {
				return nil, errors.New("bình luận gốc không tồn tại")
			}
		}
		parentID = actualParentID
	} else {
		parentID = 0
	}

	if r.dbAvailable() {
		var id int
		query := `
			INSERT INTO comments (novel_slug, parent_id, user_id, username, user_avatar, role, content, likes, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 0, $8)
			RETURNING id;
		`
		err := r.db.QueryRow(query, novelSlug, parentID, userID, username, userAvatar, role, content, now).Scan(&id)
		if err == nil {
			newComment := &domain.Comment{
				ID:         id,
				NovelSlug:  novelSlug,
				ParentID:   parentID,
				UserID:     userID,
				Username:   username,
				UserAvatar: userAvatar,
				Role:       role,
				Content:    content,
				Likes:      0,
				CreatedAt:  now,
				Replies:    make([]domain.Comment, 0),
			}
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
		ParentID:   parentID,
		UserID:     userID,
		Username:   username,
		UserAvatar: userAvatar,
		Role:       role,
		Content:    content,
		Likes:      0,
		CreatedAt:  now,
		Replies:    make([]domain.Comment, 0),
	}
	r.nextID++
	r.items = append(r.items, newComment)
	r.saveToDiskLocked()

	return &newComment, nil
}

// GetCommentsByNovel retrieves top-level comments with their replies nested inside Replies (max 1 level)
func (r *CommentRepository) GetCommentsByNovel(novelSlug string, limit, offset int) ([]domain.Comment, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	if r.dbAvailable() {
		rows, err := r.db.Query(`
			SELECT id, novel_slug, COALESCE(parent_id, 0), user_id, username, user_avatar, role, content, likes, created_at
			FROM comments
			WHERE novel_slug = $1
			ORDER BY created_at ASC
		`, novelSlug)
		if err == nil {
			defer rows.Close()
			var all []domain.Comment
			for rows.Next() {
				var c domain.Comment
				if err := rows.Scan(&c.ID, &c.NovelSlug, &c.ParentID, &c.UserID, &c.Username, &c.UserAvatar, &c.Role, &c.Content, &c.Likes, &c.CreatedAt); err == nil {
					c.Replies = make([]domain.Comment, 0)
					all = append(all, c)
				}
			}
			return buildThreadedComments(all, limit, offset)
		}
	}

	// JSON fallback
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []domain.Comment
	for _, c := range r.items {
		if c.NovelSlug == novelSlug {
			cCopy := c
			cCopy.Replies = make([]domain.Comment, 0)
			all = append(all, cCopy)
		}
	}

	// Sort chronological so replies line up properly
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.Before(all[j].CreatedAt)
	})

	return buildThreadedComments(all, limit, offset)
}

func buildThreadedComments(all []domain.Comment, limit, offset int) ([]domain.Comment, int, error) {
	totalAll := len(all)

	var roots []domain.Comment
	repliesMap := make(map[int][]domain.Comment)

	for _, c := range all {
		if c.ParentID == 0 {
			roots = append(roots, c)
		} else {
			repliesMap[c.ParentID] = append(repliesMap[c.ParentID], c)
		}
	}

	// Sort root comments newest first
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].CreatedAt.After(roots[j].CreatedAt)
	})

	// Attach replies to each root comment (already sorted oldest first)
	for i := range roots {
		if reps, ok := repliesMap[roots[i].ID]; ok {
			roots[i].Replies = reps
		} else {
			roots[i].Replies = make([]domain.Comment, 0)
		}
	}

	if offset >= len(roots) {
		return []domain.Comment{}, totalAll, nil
	}

	end := offset + limit
	if end > len(roots) {
		end = len(roots)
	}

	return roots[offset:end], totalAll, nil
}

// DeleteComment removes a comment by ID if user is owner or admin
// If target is a root comment, also removes all of its replies
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

		_, err = r.db.Exec("DELETE FROM comments WHERE id = $1 OR parent_id = $1", commentID)
		if err != nil {
			return err
		}

		r.mu.Lock()
		newItems := make([]domain.Comment, 0, len(r.items))
		for _, c := range r.items {
			if c.ID != commentID && c.ParentID != commentID {
				newItems = append(newItems, c)
			}
		}
		r.items = newItems
		r.saveToDiskLocked()
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

	newItems := make([]domain.Comment, 0, len(r.items))
	for _, c := range r.items {
		if c.ID != commentID && c.ParentID != commentID {
			newItems = append(newItems, c)
		}
	}
	r.items = newItems
	r.saveToDiskLocked()
	return nil
}

// LikeComment increments the like counter for a comment or reply
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
