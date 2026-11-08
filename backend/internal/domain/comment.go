package domain

import "time"

// Comment represents a reader's discussion or reply on a novel
type Comment struct {
	ID         int       `json:"id"`
	NovelSlug  string    `json:"novel_slug"`
	ParentID   int       `json:"parent_id"` // 0 = root comment, >0 = reply (max 1 level)
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	UserAvatar string    `json:"user_avatar"`
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	Likes      int       `json:"likes"`
	CreatedAt  time.Time `json:"created_at"`
	Replies    []Comment `json:"replies"`
}
