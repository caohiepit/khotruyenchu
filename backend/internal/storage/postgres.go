package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// NewPostgresDB establishes a connection pool to PostgreSQL
func NewPostgresDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("cannot open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return db, fmt.Errorf("ping postgres failed: %w", err)
	}

	return db, nil
}

// AutoMigrate ensures all necessary tables (especially sources) exist and default sources are seeded
func AutoMigrate(db *sql.DB) error {
	schemaQuery := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

	-- Sources registry table
	CREATE TABLE IF NOT EXISTS sources (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(100) NOT NULL,
		slug VARCHAR(100) NOT NULL UNIQUE,
		domain VARCHAR(150) NOT NULL UNIQUE,
		base_url TEXT NOT NULL,
		catalog_url TEXT NOT NULL,
		is_active BOOLEAN DEFAULT TRUE,
		scan_interval_hours INT DEFAULT 24,
		last_scanned_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Crawl queue table
	CREATE TABLE IF NOT EXISTS crawl_queue (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		title VARCHAR(255) NOT NULL,
		slug VARCHAR(280) NOT NULL UNIQUE,
		source_url TEXT NOT NULL UNIQUE,
		source_domain VARCHAR(100),
		author_name VARCHAR(150),
		cover_url TEXT,
		status VARCHAR(30) DEFAULT 'pending',
		total_chapters INT DEFAULT 0,
		crawled_chapters INT DEFAULT 0,
		error_message TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Ensure source_domain exists if table was created previously
	ALTER TABLE crawl_queue ADD COLUMN IF NOT EXISTS source_domain VARCHAR(100);

	CREATE INDEX IF NOT EXISTS idx_crawl_queue_status ON crawl_queue(status, created_at ASC);

	-- Affiliate products table
	CREATE TABLE IF NOT EXISTS affiliate_products (
		id VARCHAR(100) PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		affiliate_url TEXT NOT NULL,
		image_url TEXT NOT NULL,
		price_original NUMERIC(12, 0),
		price_discount NUMERIC(12, 0),
		tag VARCHAR(50) DEFAULT 'Recommended',
		category VARCHAR(50) DEFAULT 'ticker',
		is_active BOOLEAN DEFAULT TRUE,
		clicks_count INT DEFAULT 0,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(100) PRIMARY KEY,
		username VARCHAR(80) NOT NULL UNIQUE,
		email VARCHAR(150) NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role VARCHAR(50) DEFAULT 'user',
		realm VARCHAR(80) DEFAULT 'Phàm Nhân',
		exp INT DEFAULT 0,
		avatar_url TEXT DEFAULT '',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);
	ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(50) DEFAULT 'user';

	-- User Favorites table (Truyện Yêu Thích)
	CREATE TABLE IF NOT EXISTS user_favorites (
		id SERIAL PRIMARY KEY,
		user_id VARCHAR(100) NOT NULL,
		novel_slug VARCHAR(255) NOT NULL,
		novel_title VARCHAR(255) DEFAULT '',
		novel_cover TEXT DEFAULT '',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		UNIQUE(user_id, novel_slug)
	);

	-- Feedback table (Phi thư truyền tin)
	CREATE TABLE IF NOT EXISTS feedback (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		feedback_type VARCHAR(100),
		content TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);
	ALTER TABLE feedback DROP COLUMN IF EXISTS name;
	ALTER TABLE feedback DROP COLUMN IF EXISTS contact;

	-- Seed default sources if table is empty
	INSERT INTO sources (name, slug, domain, base_url, catalog_url, is_active, scan_interval_hours)
	VALUES 
		('Truyện Full', 'truyenfull', 'truyenfull.live', 'https://truyenfull.live/', 'https://truyenfull.live/danh-sach/truyen-hot/', TRUE, 24),
		('Tàng Thư Viện', 'tangthuvien', 'tangthuvien.net', 'https://tangthuvien.net/', 'https://tangthuvien.net/tong-hop', TRUE, 24)
	ON CONFLICT (domain) DO UPDATE SET
		name = EXCLUDED.name,
		catalog_url = EXCLUDED.catalog_url,
		base_url = EXCLUDED.base_url;
	`

	_, err := db.Exec(schemaQuery)
	if err != nil {
		return fmt.Errorf("automigrate error: %w", err)
	}

	log.Println("✅ PostgreSQL Auto-Migration applied successfully (sources & crawl_queue verified).")
	return nil
}
