package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

// SaveToPostgres lưu trọn vẹn thông tin truyện và các chương vào PostgreSQL Docker
func SaveToPostgres(connStr string, novel *Novel, chapters []*Chapter) error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("không thể kết nối PostgreSQL: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping PostgreSQL thất bại: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Upsert Novel
	var novelID string
	queryNovel := `
		INSERT INTO novels (title, slug, author_name, description, cover_url, status, total_chapters, rating_score, realm, source_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (slug) DO UPDATE SET
			title = EXCLUDED.title,
			author_name = EXCLUDED.author_name,
			description = EXCLUDED.description,
			cover_url = EXCLUDED.cover_url,
			status = EXCLUDED.status,
			total_chapters = EXCLUDED.total_chapters,
			updated_at = NOW()
		RETURNING id;
	`
	err = tx.QueryRow(queryNovel,
		novel.Title,
		novel.Slug,
		novel.Author,
		novel.Description,
		novel.CoverURL,
		novel.Status,
		novel.TotalChapters,
		novel.RatingScore,
		novel.Realm,
		novel.SourceURL,
	).Scan(&novelID)
	if err != nil {
		return fmt.Errorf("lỗi lưu novel: %w", err)
	}

	// 2. Thêm thể loại (Genres)
	for _, g := range novel.Genres {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		var genreID string
		genreSlug := strings.ToLower(strings.ReplaceAll(g, " ", "-"))

		queryGenre := `
			INSERT INTO genres (name, slug)
			VALUES ($1, $2)
			ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
			RETURNING id;
		`
		if err := tx.QueryRow(queryGenre, g, genreSlug).Scan(&genreID); err == nil {
			queryLink := `
				INSERT INTO novel_genres (novel_id, genre_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING;
			`
			tx.Exec(queryLink, novelID, genreID)
		}
	}

	// 3. Upsert từng chương (Áp dụng Split Table: chapters + chapter_contents)
	queryChapter := `
		INSERT INTO chapters (novel_id, chapter_number, title, slug, word_count)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (novel_id, chapter_number) DO UPDATE SET
			title = EXCLUDED.title,
			word_count = EXCLUDED.word_count
		RETURNING id;
	`

	queryContent := `
		INSERT INTO chapter_contents (chapter_id, content)
		VALUES ($1, $2)
		ON CONFLICT (chapter_id) DO UPDATE SET
			content = EXCLUDED.content;
	`

	for _, chap := range chapters {
		var chapterID string
		err := tx.QueryRow(queryChapter,
			novelID,
			chap.ChapterNumber,
			chap.Title,
			chap.Slug,
			chap.WordCount,
		).Scan(&chapterID)
		if err != nil {
			return fmt.Errorf("lỗi lưu chapter metadata (%d): %w", chap.ChapterNumber, err)
		}

		// Lưu content vào bảng riêng (nén LZ4 tự động)
		_, err = tx.Exec(queryContent, chapterID, chap.ContentText)
		if err != nil {
			return fmt.Errorf("lỗi lưu chapter content (%d): %w", chap.ChapterNumber, err)
		}
	}

	return tx.Commit()
}

// SaveToQueue inserts novels into crawl_queue table
func SaveToQueue(connStr string, novels []*QueueNovel) (int, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	stmt, err := db.Prepare(`
		INSERT INTO crawl_queue (title, slug, source_url, author_name, cover_url, status)
		VALUES ($1, $2, $3, $4, $5, 'pending')
		ON CONFLICT (slug) DO NOTHING;
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	inserted := 0
	for _, n := range novels {
		res, err := stmt.Exec(n.Title, n.Slug, n.SourceURL, n.AuthorName, n.CoverURL)
		if err == nil {
			if r, _ := res.RowsAffected(); r > 0 {
				inserted++
			}
		}
	}
	return inserted, nil
}

