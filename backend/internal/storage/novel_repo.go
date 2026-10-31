package storage

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"comics-backend/internal/domain"
)

type NovelRepository struct {
	db *sql.DB
	fs *FileStorage
}

func NewNovelRepository(db *sql.DB) *NovelRepository {
	return &NovelRepository{
		db: db,
		fs: NewFileStorage("storage"),
	}
}

func (r *NovelRepository) dbAvailable() bool {
	return r.db != nil && r.db.Ping() == nil
}

// SaveNovelAndChapters saves or updates novel metadata, genres, and all chapters with LZ4 compression
func (r *NovelRepository) SaveNovelAndChapters(novel *domain.Novel, chapters []*domain.Chapter) error {
	if !r.dbAvailable() {
		return nil
	}
	tx, err := r.db.Begin()
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
			total_chapters = GREATEST(novels.total_chapters, EXCLUDED.total_chapters),
			updated_at = NOW()
		RETURNING id;
	`
	err = tx.QueryRow(queryNovel,
		novel.Title,
		novel.Slug,
		novel.AuthorName,
		novel.Description,
		novel.CoverURL,
		novel.Status,
		novel.TotalChapters,
		novel.RatingScore,
		novel.Realm,
		novel.SourceURL,
	).Scan(&novelID)
	if err != nil {
		return fmt.Errorf("error saving novel: %w", err)
	}

	// 2. Genres
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
			_, _ = tx.Exec(queryLink, novelID, genreID)
		}
	}

	// 3. Upsert Chapters & LZ4 Compressed Content
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
			return fmt.Errorf("error saving chapter %d: %w", chap.ChapterNumber, err)
		}

		if chap.ContentText != "" {
			_, err = tx.Exec(queryContent, chapterID, chap.ContentText)
			if err != nil {
				return fmt.Errorf("error saving chapter content %d: %w", chap.ChapterNumber, err)
			}
		}
	}

	// Update total chapters in novel
	_, _ = tx.Exec(`
		UPDATE novels 
		SET total_chapters = (SELECT COUNT(*) FROM chapters WHERE novel_id = $1),
		    updated_at = NOW() 
		WHERE id = $1
	`, novelID)

	return tx.Commit()
}

// GetNovels queries novels with pagination and optional search filter (falls back to disk storage if DB unavailable or empty)
func (r *NovelRepository) GetNovels(search string, page int, limit int, sortBy ...string) ([]domain.Novel, int, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	sortParam := ""
	if len(sortBy) > 0 {
		sortParam = sortBy[0]
	}

	if !r.dbAvailable() {
		return r.getNovelsFromDisk(search, page, limit, sortParam)
	}

	var total int
	baseCountQuery := "SELECT COUNT(*) FROM novels"
	baseDataQuery := `
		SELECT n.id, n.title, n.slug, n.author_name, COALESCE(n.description, ''), COALESCE(n.cover_url, ''), 
		       n.status, n.total_chapters, n.views_count, n.rating_score, COALESCE(n.realm, ''),
		       COALESCE(n.source_url, ''), n.created_at, n.updated_at,
		       COALESCE((SELECT COUNT(*) FROM chapters c WHERE c.novel_id = n.id), 0) AS actual_chapters,
		       COALESCE((
		           SELECT string_agg(g.name, ', ') 
		           FROM novel_genres ng 
		           JOIN genres g ON g.id = ng.genre_id 
		           WHERE ng.novel_id = n.id
		       ), '') AS genres_str
		FROM novels n
	`

	orderClause := " ORDER BY n.updated_at DESC"
	if sortParam == "views" || sortParam == "views_all" {
		orderClause = " ORDER BY n.views_count DESC, actual_chapters DESC"
	} else if sortParam == "chapters" {
		orderClause = " ORDER BY actual_chapters DESC"
	}

	var rows *sql.Rows
	var err error

	if search != "" {
		filter := "%" + search + "%"
		_ = r.db.QueryRow("SELECT COUNT(*) FROM novels WHERE title ILIKE $1 OR author_name ILIKE $1", filter).Scan(&total)
		query := baseDataQuery + " WHERE n.title ILIKE $1 OR n.author_name ILIKE $1" + orderClause + " LIMIT $2 OFFSET $3"
		rows, err = r.db.Query(query, filter, limit, offset)
	} else {
		_ = r.db.QueryRow(baseCountQuery).Scan(&total)
		if total == 0 {
			return r.getNovelsFromDisk(search, page, limit, sortParam)
		}
		query := baseDataQuery + orderClause + " LIMIT $1 OFFSET $2"
		rows, err = r.db.Query(query, limit, offset)
	}

	if err != nil {
		return r.getNovelsFromDisk(search, page, limit, sortParam)
	}
	defer rows.Close()

	items := make([]domain.Novel, 0)
	for rows.Next() {
		var n domain.Novel
		var genresStr string
		err := rows.Scan(
			&n.ID, &n.Title, &n.Slug, &n.AuthorName, &n.Description, &n.CoverURL,
			&n.Status, &n.TotalChapters, &n.ViewsCount, &n.RatingScore, &n.Realm,
			&n.SourceURL, &n.CreatedAt, &n.UpdatedAt, &n.ActualChapters, &genresStr,
		)
		if err == nil {
			if genresStr != "" {
				n.Genres = strings.Split(genresStr, ", ")
			} else {
				n.Genres = []string{}
			}
			items = append(items, n)
		}
	}

	return items, total, nil
}

func (r *NovelRepository) getNovelsFromDisk(search string, page int, limit int, sortBy ...string) ([]domain.Novel, int, error) {
	all := r.fs.LoadAllNovelsFromDisk()
	if search != "" {
		q := strings.ToLower(search)
		var filtered []domain.Novel
		for _, n := range all {
			if strings.Contains(strings.ToLower(n.Title), q) || strings.Contains(strings.ToLower(n.AuthorName), q) {
				filtered = append(filtered, n)
			}
		}
		all = filtered
	}

	sortParam := ""
	if len(sortBy) > 0 {
		sortParam = sortBy[0]
	}
	if sortParam == "views" || sortParam == "views_all" {
		sort.Slice(all, func(i, j int) bool {
			vi := all[i].ViewsCount
			if vi <= 0 {
				vi = int64(all[i].ActualChapters * 1250)
			}
			vj := all[j].ViewsCount
			if vj <= 0 {
				vj = int64(all[j].ActualChapters * 1250)
			}
			if vi != vj {
				return vi > vj
			}
			return all[i].ActualChapters > all[j].ActualChapters
		})
	} else if sortParam == "chapters" {
		sort.Slice(all, func(i, j int) bool {
			return all[i].ActualChapters > all[j].ActualChapters
		})
	}
	total := len(all)
	offset := (page - 1) * limit
	if offset >= total {
		return []domain.Novel{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

// GetNovelDetail returns novel info and its list of chapters (falls back to disk storage)
func (r *NovelRepository) GetNovelDetail(idOrSlug string) (*domain.Novel, []domain.Chapter, error) {
	if !r.dbAvailable() {
		return r.fs.LoadNovelDetailFromDisk(idOrSlug)
	}

	var n domain.Novel
	var desc, cover, source sql.NullString
	query := `
		SELECT id, title, slug, author_name, description, cover_url, status, total_chapters, views_count, rating_score, source_url, created_at, updated_at
		FROM novels
		WHERE id::text = $1 OR slug = $1
	`
	err := r.db.QueryRow(query, idOrSlug).Scan(
		&n.ID, &n.Title, &n.Slug, &n.AuthorName, &desc, &cover,
		&n.Status, &n.TotalChapters, &n.ViewsCount, &n.RatingScore, &source,
		&n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		return r.fs.LoadNovelDetailFromDisk(idOrSlug)
	}
	n.Description = desc.String
	n.CoverURL = cover.String
	n.SourceURL = source.String

	// Genres
	genreRows, err := r.db.Query(`
		SELECT g.name 
		FROM genres g 
		JOIN novel_genres ng ON ng.genre_id = g.id 
		WHERE ng.novel_id = $1
	`, n.ID)
	if err == nil {
		defer genreRows.Close()
		for genreRows.Next() {
			var gName string
			if err := genreRows.Scan(&gName); err == nil {
				n.Genres = append(n.Genres, gName)
			}
		}
	}
	if n.Genres == nil {
		n.Genres = []string{}
	}

	// Chapters
	chapterRows, err := r.db.Query(`
		SELECT id, chapter_number, title, slug, word_count, views_count, created_at
		FROM chapters
		WHERE novel_id = $1
		ORDER BY chapter_number ASC
	`, n.ID)

	chapters := make([]domain.Chapter, 0)
	if err == nil {
		defer chapterRows.Close()
		for chapterRows.Next() {
			var c domain.Chapter
			if err := chapterRows.Scan(&c.ID, &c.ChapterNumber, &c.Title, &c.Slug, &c.WordCount, &c.ViewsCount, &c.CreatedAt); err == nil {
				chapters = append(chapters, c)
			}
		}
	}

	return &n, chapters, nil
}

// GetChapterContent returns a specific chapter with its decompressed content
func (r *NovelRepository) GetChapterContent(chapterID string) (*domain.Chapter, error) {
	if strings.Contains(chapterID, "__") || !r.dbAvailable() {
		return r.fs.LoadChapterContentFromDisk(chapterID)
	}

	var chap domain.Chapter
	query := `
		SELECT c.id, c.novel_id, c.chapter_number, c.title, c.slug, c.word_count, cc.content
		FROM chapters c
		JOIN chapter_contents cc ON cc.chapter_id = c.id
		WHERE c.id::text = $1
	`
	err := r.db.QueryRow(query, chapterID).Scan(
		&chap.ID, &chap.NovelID, &chap.ChapterNumber, &chap.Title, &chap.Slug, &chap.WordCount, &chap.ContentText,
	)
	if err != nil {
		return r.fs.LoadChapterContentFromDisk(chapterID)
	}
	return &chap, nil
}

// GetGenres returns list of genres with novel count
func (r *NovelRepository) GetGenres() ([]domain.Genre, error) {
	if !r.dbAvailable() {
		return r.getGenresFromDisk(), nil
	}

	rows, err := r.db.Query(`
		SELECT g.id, g.name, g.slug, COUNT(ng.novel_id) AS novel_count
		FROM genres g
		LEFT JOIN novel_genres ng ON ng.genre_id = g.id
		GROUP BY g.id, g.name, g.slug
		ORDER BY novel_count DESC, g.name ASC
	`)
	if err != nil {
		return r.getGenresFromDisk(), nil
	}
	defer rows.Close()

	genres := make([]domain.Genre, 0)
	for rows.Next() {
		var g domain.Genre
		if err := rows.Scan(&g.ID, &g.Name, &g.Slug, &g.NovelCount); err == nil {
			genres = append(genres, g)
		}
	}
	if len(genres) == 0 {
		return r.getGenresFromDisk(), nil
	}
	return genres, nil
}

func (r *NovelRepository) getGenresFromDisk() []domain.Genre {
	novels := r.fs.LoadAllNovelsFromDisk()
	counts := make(map[string]int)
	for _, n := range novels {
		for _, g := range n.Genres {
			counts[g]++
		}
	}
	var genres []domain.Genre
	for name, c := range counts {
		slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		genres = append(genres, domain.Genre{
			ID:         slug,
			Name:       name,
			Slug:       slug,
			NovelCount: c,
		})
	}
	return genres
}

// GetOngoingNovels returns novels that are still updating
func (r *NovelRepository) GetOngoingNovels(limit int) ([]domain.Novel, error) {
	if !r.dbAvailable() {
		return nil, nil
	}
	rows, err := r.db.Query(`
		SELECT id, title, slug, source_url, total_chapters
		FROM novels
		WHERE status ILIKE '%đang%' OR status ILIKE '%ongoing%'
		ORDER BY updated_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var novels []domain.Novel
	for rows.Next() {
		var n domain.Novel
		if err := rows.Scan(&n.ID, &n.Title, &n.Slug, &n.SourceURL, &n.TotalChapters); err == nil {
			novels = append(novels, n)
		}
	}
	return novels, nil
}
