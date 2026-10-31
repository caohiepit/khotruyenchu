package domain

import "time"

// CrawlQueueItem represents an item in the crawl_queue table
type CrawlQueueItem struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Slug            string    `json:"slug"`
	SourceURL       string    `json:"source_url"`
	SourceDomain    string    `json:"source_domain,omitempty"`
	AuthorName      string    `json:"author_name"`
	CoverURL        string    `json:"cover_url"`
	Status          string    `json:"status"` // "pending", "crawling", "completed", "failed"
	TotalChapters   int       `json:"total_chapters"`
	CrawledChapters int       `json:"crawled_chapters"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// QueueCounts represents aggregated counts of queue items by status
type QueueCounts struct {
	Pending   int `json:"pending"`
	Crawling  int `json:"crawling"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Total     int `json:"total"`
}
