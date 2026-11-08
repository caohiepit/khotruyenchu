package domain

import "time"

// Comment represents a reader's discussion or review on a novel
type Comment struct {
	ID         int       `json:"id"`
	NovelSlug  string    `json:"novel_slug"`
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	UserAvatar string    `json:"user_avatar"`
	UserRealm  string    `json:"user_realm"`
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	Likes      int       `json:"likes"`
	CreatedAt  time.Time `json:"created_at"`
}
