package domain

import "time"

// Source represents an external novel website configuration
type Source struct {
	ID                string     `json:"id"`
	Name              string     `json:"name"`
	Slug              string     `json:"slug"`
	Domain            string     `json:"domain"`
	BaseURL           string     `json:"base_url"`
	CatalogURL        string     `json:"catalog_url"`
	IsActive          bool       `json:"is_active"`
	ScanIntervalHours int        `json:"scan_interval_hours"`
	LastScannedAt     *time.Time `json:"last_scanned_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// DiscoveredNovel represents a novel found when scanning catalog or category pages
type DiscoveredNovel struct {
	Title        string `json:"title"`
	Slug         string `json:"slug"`
	SourceURL    string `json:"source_url"`
	SourceDomain string `json:"source_domain"`
	AuthorName   string `json:"author_name"`
	CoverURL     string `json:"cover_url"`
}
