package domain

import "time"

// AffiliateProduct represents a Shopee or merchant affiliate link
type AffiliateProduct struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	AffiliateURL  string    `json:"affiliate_url"`
	ImageURL      string    `json:"image_url"`
	PriceOriginal string    `json:"price_original,omitempty"`
	PriceDiscount string    `json:"price_discount,omitempty"`
	Tag           string    `json:"tag,omitempty"`
	Category      string    `json:"category,omitempty"`
	Position      string    `json:"position,omitempty"` // "ticker", "sidebar", "reader_footer"
	IsActive      bool      `json:"is_active"`
	ClicksCount   int       `json:"clicks_count"`
	CreatedAt     time.Time `json:"created_at"`
}

