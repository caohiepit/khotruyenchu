package domain

import "time"

type Feedback struct {
	ID           string    `json:"id"`
	FeedbackType string    `json:"feedback_type"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

