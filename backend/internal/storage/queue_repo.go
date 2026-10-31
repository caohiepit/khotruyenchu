package storage

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"comics-backend/internal/domain"
)

type QueueRepository struct {
	db        *sql.DB
	fs        *FileStorage
	memMu     sync.RWMutex
	memQueue  []domain.CrawlQueueItem
	nextMemID int64
}

func NewQueueRepository(db *sql.DB) *QueueRepository {
	return &QueueRepository{
		db:        db,
		fs:        NewFileStorage("storage"),
		memQueue:  make([]domain.CrawlQueueItem, 0),
		nextMemID: 1,
	}
}

func (r *QueueRepository) dbAvailable() bool {
	return r.db != nil && r.db.Ping() == nil
}

// EnqueueNovel inserts or updates a single novel in crawl_queue
func (r *QueueRepository) EnqueueNovel(item *domain.CrawlQueueItem) error {
	if !r.dbAvailable() {
		r.memMu.Lock()
		defer r.memMu.Unlock()
		for i, q := range r.memQueue {
			if q.Slug == item.Slug {
				r.memQueue[i].Title = item.Title
				r.memQueue[i].SourceURL = item.SourceURL
				if item.Status != "" {
					r.memQueue[i].Status = item.Status
				}
				r.memQueue[i].UpdatedAt = time.Now()
				item.ID = r.memQueue[i].ID
				return nil
			}
		}
		item.ID = fmt.Sprintf("%d", r.nextMemID)
		r.nextMemID++
		item.CreatedAt = time.Now()
		item.UpdatedAt = time.Now()
		if item.Status == "" {
			item.Status = "pending"
		}
		r.memQueue = append(r.memQueue, *item)
		return nil
	}
	query := `
		INSERT INTO crawl_queue (title, slug, source_url, source_domain, author_name, cover_url, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		ON CONFLICT (slug) DO UPDATE SET
			title = EXCLUDED.title,
			source_url = EXCLUDED.source_url,
			source_domain = COALESCE(EXCLUDED.source_domain, crawl_queue.source_domain),
			author_name = COALESCE(EXCLUDED.author_name, crawl_queue.author_name),
			cover_url = COALESCE(EXCLUDED.cover_url, crawl_queue.cover_url),
			status = CASE WHEN crawl_queue.status = 'failed' THEN 'pending' ELSE crawl_queue.status END,
			updated_at = NOW()
		RETURNING id;
	`
	return r.db.QueryRow(query, item.Title, item.Slug, item.SourceURL, item.SourceDomain, item.AuthorName, item.CoverURL).Scan(&item.ID)
}

// EnqueueBatch inserts a slice of discovered novels into crawl_queue
func (r *QueueRepository) EnqueueBatch(novels []domain.DiscoveredNovel) (int, error) {
	if !r.dbAvailable() {
		r.memMu.Lock()
		defer r.memMu.Unlock()
		inserted := 0
		for _, n := range novels {
			exists := false
			for _, q := range r.memQueue {
				if q.Slug == n.Slug {
					exists = true
					break
				}
			}
			if !exists {
				item := domain.CrawlQueueItem{
					ID:           fmt.Sprintf("%d", r.nextMemID),
					Title:        n.Title,
					Slug:         n.Slug,
					SourceURL:    n.SourceURL,
					SourceDomain: n.SourceDomain,
					AuthorName:   n.AuthorName,
					CoverURL:     n.CoverURL,
					Status:       "pending",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}
				r.nextMemID++
				r.memQueue = append(r.memQueue, item)
				inserted++
			}
		}
		return inserted, nil
	}
	stmt, err := r.db.Prepare(`
		INSERT INTO crawl_queue (title, slug, source_url, source_domain, author_name, cover_url, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		ON CONFLICT (slug) DO NOTHING;
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	inserted := 0
	for _, n := range novels {
		res, err := stmt.Exec(n.Title, n.Slug, n.SourceURL, n.SourceDomain, n.AuthorName, n.CoverURL)
		if err == nil {
			if count, _ := res.RowsAffected(); count > 0 {
				inserted++
			}
		}
	}
	return inserted, nil
}

// GetNextPendingItem picks the next pending novel in queue and marks it as 'crawling'
func (r *QueueRepository) GetNextPendingItem() (*domain.CrawlQueueItem, error) {
	if !r.dbAvailable() {
		r.memMu.Lock()
		defer r.memMu.Unlock()
		for i, q := range r.memQueue {
			if q.Status == "pending" {
				r.memQueue[i].Status = "crawling"
				r.memQueue[i].UpdatedAt = time.Now()
				res := r.memQueue[i]
				return &res, nil
			}
		}
		return nil, sql.ErrNoRows
	}
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var item domain.CrawlQueueItem
	var author, cover, domainStr sql.NullString
	query := `
		SELECT id, title, slug, source_url, source_domain, author_name, cover_url, status, total_chapters, crawled_chapters, created_at, updated_at
		FROM crawl_queue
		WHERE status = 'pending'
		ORDER BY created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1;
	`
	err = tx.QueryRow(query).Scan(
		&item.ID, &item.Title, &item.Slug, &item.SourceURL, &domainStr, &author, &cover,
		&item.Status, &item.TotalChapters, &item.CrawledChapters, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err // sql.ErrNoRows if empty
	}

	item.AuthorName = author.String
	item.CoverURL = cover.String
	item.SourceDomain = domainStr.String

	// Update status to crawling
	_, err = tx.Exec("UPDATE crawl_queue SET status = 'crawling', updated_at = NOW() WHERE id = $1", item.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	item.Status = "crawling"
	return &item, nil
}

// UpdateStatus updates queue item status upon completion or error
func (r *QueueRepository) UpdateStatus(id string, status string, crawledChapters int, errMsg string) error {
	if !r.dbAvailable() {
		r.memMu.Lock()
		defer r.memMu.Unlock()
		for i, q := range r.memQueue {
			if q.ID == id {
				r.memQueue[i].Status = status
				r.memQueue[i].CrawledChapters = crawledChapters
				r.memQueue[i].ErrorMessage = errMsg
				r.memQueue[i].UpdatedAt = time.Now()
				break
			}
		}
		return nil
	}
	var query string
	if errMsg != "" {
		if len(errMsg) > 500 {
			errMsg = errMsg[:500]
		}
		query = "UPDATE crawl_queue SET status = $1, error_message = $2, updated_at = NOW() WHERE id = $3"
		_, err := r.db.Exec(query, status, errMsg, id)
		return err
	}
	query = "UPDATE crawl_queue SET status = $1, crawled_chapters = $2, error_message = NULL, updated_at = NOW() WHERE id = $3"
	_, err := r.db.Exec(query, status, crawledChapters, id)
	return err
}

// GetQueueList returns items with optional status filtering
func (r *QueueRepository) GetQueueList(statusFilter string, limit int) ([]domain.CrawlQueueItem, error) {
	if !r.dbAvailable() {
		r.memMu.RLock()
		defer r.memMu.RUnlock()
		items := make([]domain.CrawlQueueItem, 0)
		for i := len(r.memQueue) - 1; i >= 0; i-- {
			q := r.memQueue[i]
			if statusFilter == "" || q.Status == statusFilter {
				items = append(items, q)
				if limit > 0 && len(items) >= limit {
					break
				}
			}
		}
		return items, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	var rows *sql.Rows
	var err error
	if statusFilter != "" {
		rows, err = r.db.Query(`
			SELECT id, title, slug, source_url, COALESCE(source_domain, ''), COALESCE(author_name, ''), COALESCE(cover_url, ''), status, total_chapters, crawled_chapters, COALESCE(error_message, ''), created_at, updated_at
			FROM crawl_queue
			WHERE status = $1
			ORDER BY updated_at DESC
			LIMIT $2
		`, statusFilter, limit)
	} else {
		rows, err = r.db.Query(`
			SELECT id, title, slug, source_url, COALESCE(source_domain, ''), COALESCE(author_name, ''), COALESCE(cover_url, ''), status, total_chapters, crawled_chapters, COALESCE(error_message, ''), created_at, updated_at
			FROM crawl_queue
			ORDER BY 
				CASE status 
					WHEN 'crawling' THEN 1 
					WHEN 'pending' THEN 2 
					WHEN 'failed' THEN 3 
					ELSE 4 
				END,
				updated_at DESC
			LIMIT $1
		`, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.CrawlQueueItem, 0)
	for rows.Next() {
		var q domain.CrawlQueueItem
		err := rows.Scan(
			&q.ID, &q.Title, &q.Slug, &q.SourceURL, &q.SourceDomain, &q.AuthorName, &q.CoverURL,
			&q.Status, &q.TotalChapters, &q.CrawledChapters, &q.ErrorMessage, &q.CreatedAt, &q.UpdatedAt,
		)
		if err == nil {
			items = append(items, q)
		}
	}
	return items, nil
}

// GetCounts returns aggregated counts of the queue
func (r *QueueRepository) GetCounts() (domain.QueueCounts, error) {
	var counts domain.QueueCounts
	if !r.dbAvailable() {
		r.memMu.RLock()
		defer r.memMu.RUnlock()
		for _, q := range r.memQueue {
			switch q.Status {
			case "pending":
				counts.Pending++
			case "crawling":
				counts.Crawling++
			case "completed":
				counts.Completed++
			case "failed":
				counts.Failed++
			}
		}
		novels := r.fs.LoadAllNovelsFromDisk()
		counts.Completed += len(novels)
		counts.Total = counts.Pending + counts.Crawling + counts.Completed + counts.Failed
		return counts, nil
	}
	_ = r.db.QueryRow("SELECT COUNT(*) FROM crawl_queue WHERE status = 'pending'").Scan(&counts.Pending)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM crawl_queue WHERE status = 'crawling'").Scan(&counts.Crawling)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM crawl_queue WHERE status = 'completed'").Scan(&counts.Completed)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM crawl_queue WHERE status = 'failed'").Scan(&counts.Failed)
	counts.Total = counts.Pending + counts.Crawling + counts.Completed + counts.Failed
	return counts, nil
}

// ResetFailed resets failed items back to pending
func (r *QueueRepository) ResetFailed() (int64, error) {
	res, err := r.db.Exec("UPDATE crawl_queue SET status = 'pending', error_message = NULL, updated_at = NOW() WHERE status = 'failed'")
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// GetDashboardStats aggregates all metrics across novels, chapters, and queue
func (r *QueueRepository) GetDashboardStats() (domain.DashboardStats, error) {
	var stats domain.DashboardStats
	if !r.dbAvailable() {
		novels := r.fs.LoadAllNovelsFromDisk()
		stats.TotalNovels = len(novels)
		genreSet := make(map[string]bool)
		for _, n := range novels {
			stats.TotalChapters += n.ActualChapters
			stats.TotalWords += int64(n.ActualChapters * 2800)
			for _, g := range n.Genres {
				genreSet[g] = true
			}
		}
		stats.TotalGenres = len(genreSet)
		stats.QueueCompleted = stats.TotalNovels
		stats.QueueTotal = stats.TotalNovels
		return stats, nil
	}

	var totalWords sql.NullInt64

	_ = r.db.QueryRow("SELECT COUNT(*) FROM novels").Scan(&stats.TotalNovels)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM chapters").Scan(&stats.TotalChapters)
	_ = r.db.QueryRow("SELECT COALESCE(SUM(word_count), 0) FROM chapters").Scan(&totalWords)
	_ = r.db.QueryRow("SELECT COUNT(*) FROM genres").Scan(&stats.TotalGenres)

	if totalWords.Valid {
		stats.TotalWords = totalWords.Int64
	}

	if stats.TotalNovels == 0 {
		novels := r.fs.LoadAllNovelsFromDisk()
		stats.TotalNovels = len(novels)
		for _, n := range novels {
			stats.TotalChapters += n.ActualChapters
			stats.TotalWords += int64(n.ActualChapters * 2800)
		}
	}

	qc, _ := r.GetCounts()
	stats.QueuePending = qc.Pending
	stats.QueueCrawling = qc.Crawling
	stats.QueueCompleted = qc.Completed
	stats.QueueFailed = qc.Failed
	stats.QueueTotal = qc.Total

	return stats, nil
}
