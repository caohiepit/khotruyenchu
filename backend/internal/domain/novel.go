package domain

import "time"

// Novel represents metadata of a story/comic
type Novel struct {
	ID             string    `json:"id,omitempty"`
	Title          string    `json:"title"`
	Slug           string    `json:"slug"`
	AuthorName     string    `json:"author_name"`
	Genres         []string  `json:"genres"`
	Description    string    `json:"description"`
	CoverURL       string    `json:"cover_url"`
	Status         string    `json:"status"` // "Full", "Đang ra", "ongoing", "completed"
	TotalChapters  int       `json:"total_chapters"`
	ActualChapters int       `json:"actual_chapters,omitempty"`
	ViewsCount     int64     `json:"views_count"`
	RatingScore    float64   `json:"rating_score"`
	Realm          string    `json:"realm"`
	SourceURL      string    `json:"source_url"`
	SourceDomain   string    `json:"source_domain,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Chapter represents metadata of a chapter
type Chapter struct {
	ID            string    `json:"id,omitempty"`
	NovelID       string    `json:"novel_id,omitempty"`
	ChapterNumber int       `json:"chapter_number"`
	Title         string    `json:"title"`
	Slug          string    `json:"slug"`
	ContentText   string    `json:"content_text,omitempty"` // Clean text content
	WordCount     int       `json:"word_count"`
	ViewsCount    int64     `json:"views_count"`
	SourceURL     string    `json:"source_url"`
	NextURL       string    `json:"next_url,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Genre represents a category
type Genre struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	NovelCount int    `json:"novel_count,omitempty"`
}

// Stats represents dashboard statistics
type DashboardStats struct {
	TotalNovels    int   `json:"total_novels"`
	TotalChapters  int   `json:"total_chapters"`
	TotalWords     int64 `json:"total_words"`
	TotalGenres    int   `json:"total_genres"`
	QueuePending   int   `json:"queue_pending"`
	QueueCrawling  int   `json:"queue_crawling"`
	QueueCompleted int   `json:"queue_completed"`
	QueueFailed    int   `json:"queue_failed"`
	QueueTotal     int   `json:"queue_total"`
	WorkerRunning  bool  `json:"worker_running"`
}
