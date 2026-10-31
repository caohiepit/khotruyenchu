package storage

import (
	"database/sql"
	"fmt"

	"comics-backend/internal/domain"
)

type SourceRepository struct {
	db *sql.DB
}

func NewSourceRepository(db *sql.DB) *SourceRepository {
	return &SourceRepository{db: db}
}

// GetAllSources returns all registered story sources
func (r *SourceRepository) GetAllSources() ([]domain.Source, error) {
	if r.db == nil || r.db.Ping() != nil {
		return []domain.Source{
			{ID: "src-1", Name: "Truyện Full", Slug: "truyenfull", Domain: "truyenfull.live", BaseURL: "https://truyenfull.live/", CatalogURL: "https://truyenfull.live/danh-sach/truyen-hot/", IsActive: true, ScanIntervalHours: 24},
			{ID: "src-2", Name: "Tàng Thư Viện", Slug: "tangthuvien", Domain: "tangthuvien.net", BaseURL: "https://tangthuvien.net/", CatalogURL: "https://tangthuvien.net/tong-hop", IsActive: true, ScanIntervalHours: 24},
			{ID: "src-3", Name: "Truyện Full Dịch", Slug: "truyenfulldich", Domain: "truyenfulldich.top", BaseURL: "https://truyenfulldich.top/", CatalogURL: "https://truyenfulldich.top/danh-sach?search_type=HOT_COMBO", IsActive: true, ScanIntervalHours: 12},
		}, nil
	}
	query := `
		SELECT id, name, slug, domain, base_url, catalog_url, is_active, scan_interval_hours, last_scanned_at, created_at, updated_at
		FROM sources
		ORDER BY name ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []domain.Source
	for rows.Next() {
		var s domain.Source
		var lastScanned sql.NullTime
		err := rows.Scan(
			&s.ID, &s.Name, &s.Slug, &s.Domain, &s.BaseURL, &s.CatalogURL,
			&s.IsActive, &s.ScanIntervalHours, &lastScanned, &s.CreatedAt, &s.UpdatedAt,
		)
		if err == nil {
			if lastScanned.Valid {
				s.LastScannedAt = &lastScanned.Time
			}
			sources = append(sources, s)
		}
	}
	return sources, nil
}

// GetActiveSources returns sources that have is_active = true
func (r *SourceRepository) GetActiveSources() ([]domain.Source, error) {
	if r.db == nil || r.db.Ping() != nil {
		return nil, nil
	}
	query := `
		SELECT id, name, slug, domain, base_url, catalog_url, is_active, scan_interval_hours, last_scanned_at, created_at, updated_at
		FROM sources
		WHERE is_active = TRUE
		ORDER BY name ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []domain.Source
	for rows.Next() {
		var s domain.Source
		var lastScanned sql.NullTime
		err := rows.Scan(
			&s.ID, &s.Name, &s.Slug, &s.Domain, &s.BaseURL, &s.CatalogURL,
			&s.IsActive, &s.ScanIntervalHours, &lastScanned, &s.CreatedAt, &s.UpdatedAt,
		)
		if err == nil {
			if lastScanned.Valid {
				s.LastScannedAt = &lastScanned.Time
			}
			sources = append(sources, s)
		}
	}
	return sources, nil
}

// UpdateLastScanned updates timestamp when a source was scanned
func (r *SourceRepository) UpdateLastScanned(sourceID string) error {
	_, err := r.db.Exec("UPDATE sources SET last_scanned_at = NOW(), updated_at = NOW() WHERE id::text = $1", sourceID)
	return err
}

// ToggleSource enables or disables a source
func (r *SourceRepository) ToggleSource(sourceID string, isActive bool) error {
	_, err := r.db.Exec("UPDATE sources SET is_active = $1, updated_at = NOW() WHERE id::text = $2", isActive, sourceID)
	return err
}

// UpsertSource adds or updates a source configuration
func (r *SourceRepository) UpsertSource(s *domain.Source) error {
	query := `
		INSERT INTO sources (name, slug, domain, base_url, catalog_url, is_active, scan_interval_hours)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (domain) DO UPDATE SET
			name = EXCLUDED.name,
			base_url = EXCLUDED.base_url,
			catalog_url = EXCLUDED.catalog_url,
			is_active = EXCLUDED.is_active,
			scan_interval_hours = EXCLUDED.scan_interval_hours,
			updated_at = NOW()
		RETURNING id;
	`
	err := r.db.QueryRow(query, s.Name, s.Slug, s.Domain, s.BaseURL, s.CatalogURL, s.IsActive, s.ScanIntervalHours).Scan(&s.ID)
	if err != nil {
		return fmt.Errorf("error upserting source: %w", err)
	}
	return nil
}
