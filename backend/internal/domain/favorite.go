package domain

import "time"

// Favorite represents a reader's bookmarked / favorited novel
type Favorite struct {
	ID         int       `json:"id"`
	UserID     string    `json:"user_id"`
	NovelSlug  string    `json:"novel_slug"`
	NovelTitle string    `json:"novel_title"`
	NovelCover string    `json:"novel_cover"`
	CreatedAt  time.Time `json:"created_at"`
}

