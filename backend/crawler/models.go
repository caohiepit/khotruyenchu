package main

import "time"

// Novel đại diện cho thông tin một bộ truyện
type Novel struct {
	ID            string    `json:"id,omitempty"`
	Title         string    `json:"title"`
	Slug          string    `json:"slug"`
	Author        string    `json:"author"`
	Genres        []string  `json:"genres"`
	Description   string    `json:"description"`
	CoverURL      string    `json:"cover_url"`
	Status        string    `json:"status"` // "Full" hoặc "Đang ra"
	RatingScore   float64   `json:"rating_score"`
	TotalChapters int       `json:"total_chapters"`
	Realm         string    `json:"realm"` // Cảnh giới tu tiên
	SourceURL     string    `json:"source_url"`
	CreatedAt     time.Time `json:"created_at"`
}

// Chapter đại diện cho một chương truyện
type Chapter struct {
	ChapterNumber int       `json:"chapter_number"`
	Title         string    `json:"title"`
	Slug          string    `json:"slug"`
	ContentText   string    `json:"content_text"` // Nội dung đã được lọc sạch watermark
	WordCount     int       `json:"word_count"`
	SourceURL     string    `json:"source_url"`
	NextURL       string    `json:"next_url,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// QueueNovel represents a novel discovered from catalog pages
type QueueNovel struct {
	Title      string `json:"title"`
	Slug       string `json:"slug"`
	SourceURL  string `json:"source_url"`
	AuthorName string `json:"author_name"`
	CoverURL   string `json:"cover_url"`
}

