-- ====================================================================
-- DATABASE SCHEMA (HIGH-PERFORMANCE FOR POSTGRESQL 16)
-- Database: comics_db
-- Architecture: Split Table Pattern + TOAST LZ4 Compression
-- ====================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "unaccent";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 1. Genres Table
CREATE TABLE IF NOT EXISTS genres (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(120) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Authors Table
CREATE TABLE IF NOT EXISTS authors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(180) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Novels / Comics Table
CREATE TABLE IF NOT EXISTS novels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(280) NOT NULL UNIQUE,
    author_name VARCHAR(150) NOT NULL,
    description TEXT,
    cover_url TEXT,
    status VARCHAR(30) DEFAULT 'ongoing', -- ongoing, completed
    total_chapters INT DEFAULT 0,
    views_count BIGINT DEFAULT 0,
    rating_score NUMERIC(3, 2) DEFAULT 9.5,
    rating_count INT DEFAULT 0,
    realm VARCHAR(50) DEFAULT NULL,
    is_hot BOOLEAN DEFAULT FALSE,
    source_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_novels_slug ON novels(slug);
CREATE INDEX IF NOT EXISTS idx_novels_title_trgm ON novels USING gin (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_novels_views ON novels(views_count DESC);

-- 4. Novel / Comic to Genre Junction Table
CREATE TABLE IF NOT EXISTS novel_genres (
    novel_id UUID REFERENCES novels(id) ON DELETE CASCADE,
    genre_id UUID REFERENCES genres(id) ON DELETE CASCADE,
    PRIMARY KEY (novel_id, genre_id)
);

-- 5. Chapters Table (Metadata only for fast index & directory queries)
CREATE TABLE IF NOT EXISTS chapters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    novel_id UUID NOT NULL REFERENCES novels(id) ON DELETE CASCADE,
    chapter_number INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(280) NOT NULL,
    word_count INT DEFAULT 0,
    views_count BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_novel_chapter_num UNIQUE (novel_id, chapter_number),
    CONSTRAINT uq_novel_chapter_slug UNIQUE (novel_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_chapters_lookup ON chapters(novel_id, chapter_number ASC);
CREATE INDEX IF NOT EXISTS idx_chapters_slug ON chapters(novel_id, slug);

-- 6. Chapter Contents Table (Split Table - Automatic LZ4 TOAST compression)
CREATE TABLE IF NOT EXISTS chapter_contents (
    chapter_id UUID PRIMARY KEY REFERENCES chapters(id) ON DELETE CASCADE,
    content TEXT NOT NULL
);

-- Enable LZ4 compression on content column for high performance & disk space saving
ALTER TABLE chapter_contents ALTER COLUMN content SET COMPRESSION lz4;

-- 7. Affiliate Products Table
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

-- 8. Users Table (Cultivators)
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(100) PRIMARY KEY,
    username VARCHAR(80) NOT NULL UNIQUE,
    email VARCHAR(150) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    realm VARCHAR(80) DEFAULT 'Phàm Nhân',
    exp INT DEFAULT 0,
    avatar_url TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 8. Full-text search function (Unaccented Vietnamese support)
CREATE OR REPLACE FUNCTION search_novels(keyword TEXT)
RETURNS SETOF novels AS $$
BEGIN
    RETURN QUERY
    SELECT *
    FROM novels
    WHERE unaccent(title) ILIKE '%' || unaccent(keyword) || '%'
       OR unaccent(author_name) ILIKE '%' || unaccent(keyword) || '%'
    ORDER BY views_count DESC
    LIMIT 20;
END;
$$ LANGUAGE plpgsql;

-- 9. Crawl Queue Table (Two-Phase Queue-based Crawler)
CREATE TABLE IF NOT EXISTS crawl_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(280) NOT NULL UNIQUE,
    source_url TEXT NOT NULL UNIQUE,
    author_name VARCHAR(150),
    cover_url TEXT,
    status VARCHAR(30) DEFAULT 'pending', -- pending, crawling, completed, failed
    total_chapters INT DEFAULT 0,
    crawled_chapters INT DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_crawl_queue_status ON crawl_queue(status, created_at ASC);

-- Initial Sources Seeding
INSERT INTO sources (name, slug, domain, base_url, catalog_url, is_active, scan_interval_hours)
VALUES 
    ('Truyện Full', 'truyenfull', 'truyenfull.live', 'https://truyenfull.live/', 'https://truyenfull.live/danh-sach/truyen-hot/', true, 24),
    ('Tàng Thư Viện', 'tangthuvien', 'tangthuvien.net', 'https://tangthuvien.net/', 'https://tangthuvien.net/tong-hop', true, 24),
    ('Truyện Full Dịch', 'truyenfulldich', 'truyenfulldich.top', 'https://truyenfulldich.top/', 'https://truyenfulldich.top/danh-sach?search_type=HOT_COMBO', true, 12)
ON CONFLICT (slug) DO UPDATE SET 
    name = EXCLUDED.name,
    domain = EXCLUDED.domain,
    base_url = EXCLUDED.base_url,
    catalog_url = EXCLUDED.catalog_url,
    is_active = EXCLUDED.is_active,
    scan_interval_hours = EXCLUDED.scan_interval_hours,
    updated_at = NOW();

