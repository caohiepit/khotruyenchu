package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"comics-backend/internal/cleaner"
	"comics-backend/internal/domain"
)

// FileStorage handles saving and reading chapters and novel metadata directly on disk
type FileStorage struct {
	baseDir string
}

func NewFileStorage(baseDir string) *FileStorage {
	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		absPath = baseDir
	}
	return &FileStorage{baseDir: absPath}
}

// SaveNovelAndChapters writes novel.json and individual chapter .txt / .json files
func (fs *FileStorage) SaveNovelAndChapters(novel *domain.Novel, chapters []*domain.Chapter) error {
	// 🧹 Clean data trước khi lưu xuống disk
	cleaner.CleanNovel(novel)
	for _, c := range chapters {
		cleaner.CleanChapter(c)
	}

	novelDir := filepath.Join(fs.baseDir, novel.Slug)
	chaptersDir := filepath.Join(novelDir, "chapters")

	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 1. Save novel.json
	novelFile := filepath.Join(novelDir, "novel.json")
	novelBytes, err := json.MarshalIndent(novel, "", "  ")
	if err == nil {
		_ = os.WriteFile(novelFile, novelBytes, 0644)
	}

	// 2. Save each chapter as .txt and .json
	for _, chap := range chapters {
		if chap.ContentText != "" {
			txtFile := filepath.Join(chaptersDir, fmt.Sprintf("%d.txt", chap.ChapterNumber))
			_ = os.WriteFile(txtFile, []byte(chap.ContentText), 0644)
		}

		jsonFile := filepath.Join(chaptersDir, fmt.Sprintf("%d.json", chap.ChapterNumber))
		chapBytes, _ := json.MarshalIndent(chap, "", "  ")
		_ = os.WriteFile(jsonFile, chapBytes, 0644)
	}

	// 3. Count all chapters on disk to keep the maximum chapter count across multi-source aggregation
	diskCount := fs.CountTotalChapters(novel.Slug)
	if diskCount > novel.TotalChapters {
		novel.TotalChapters = diskCount
	}
	novel.ActualChapters = diskCount

	// Re-save novel.json with updated maximum chapter count
	novelBytes, err = json.MarshalIndent(novel, "", "  ")
	if err == nil {
		_ = os.WriteFile(novelFile, novelBytes, 0644)
	}

	return nil
}

// GetExistingChapterNumbers returns a set of all chapter numbers currently saved on disk
func (fs *FileStorage) GetExistingChapterNumbers(novelSlug string) map[int]bool {
	res := make(map[int]bool)
	for _, base := range fs.candidateDirs() {
		chapDir := filepath.Join(base, novelSlug, "chapters")
		entries, err := os.ReadDir(chapDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".txt") {
				numStr := strings.TrimSuffix(e.Name(), ".txt")
				if n, err := strconv.Atoi(numStr); err == nil {
					res[n] = true
				}
			}
		}
	}
	return res
}

// CountTotalChapters returns total unique chapters present on disk
func (fs *FileStorage) CountTotalChapters(novelSlug string) int {
	return len(fs.GetExistingChapterNumbers(novelSlug))
}

// NormalizeSlug strips common prefixes like "truyen-" to group identical novels from different sources
func NormalizeSlug(slug string) string {
	slug = strings.Trim(strings.ToLower(slug), "/")
	slug = strings.TrimPrefix(slug, "truyen-")
	slug = strings.TrimPrefix(slug, "bo-")
	slug = strings.TrimPrefix(slug, "truyen_")
	return slug
}

// ReadChapterText reads .txt chapter directly from disk if exists
func (fs *FileStorage) ReadChapterText(novelSlug string, chapterNum int) (string, error) {
	txtFile := filepath.Join(fs.baseDir, novelSlug, "chapters", fmt.Sprintf("%d.txt", chapterNum))
	bytes, err := os.ReadFile(txtFile)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

type rawDiskNovel struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Slug          string   `json:"slug"`
	AuthorName    string   `json:"author_name"`
	AuthorLegacy  string   `json:"author"`
	Genres        []string `json:"genres"`
	Description   string   `json:"description"`
	CoverURL      string   `json:"cover_url"`
	Status        string   `json:"status"`
	TotalChapters int      `json:"total_chapters"`
	RatingScore   float64  `json:"rating_score"`
	Realm         string   `json:"realm"`
	SourceURL     string   `json:"source_url"`
}

func (fs *FileStorage) candidateDirs() []string {
	dirs := []string{fs.baseDir}
	for _, alt := range []string{
		filepath.Join("storage"),
		filepath.Join("..", "storage"),
		filepath.Join("..", "..", "storage"),
		filepath.Join("backend", "storage"),
	} {
		if abs, err := filepath.Abs(alt); err == nil {
			exists := false
			for _, d := range dirs {
				if strings.EqualFold(d, abs) {
					exists = true
					break
				}
			}
			if !exists {
				dirs = append(dirs, abs)
			}
		}
	}
	return dirs
}

// LoadAllNovelsFromDisk scans storage directories and returns all novels with chapter counts
func (fs *FileStorage) LoadAllNovelsFromDisk() []domain.Novel {
	seen := make(map[string]bool)
	var result []domain.Novel

	for _, dir := range fs.candidateDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			slug := entry.Name()
			if seen[slug] {
				continue
			}
			novelFile := filepath.Join(dir, slug, "novel.json")
			data, err := os.ReadFile(novelFile)
			if err != nil {
				continue
			}
			var raw rawDiskNovel
			if err := json.Unmarshal(data, &raw); err != nil {
				continue
			}
			seen[slug] = true

			author := raw.AuthorName
			if author == "" {
				author = raw.AuthorLegacy
			}
			if author == "" {
				author = "Đang cập nhật"
			}

			// Deduplicate genres
			genreSet := make(map[string]bool)
			var cleanGenres []string
			for _, g := range raw.Genres {
				g = strings.TrimSpace(g)
				if g != "" && !genreSet[g] {
					genreSet[g] = true
					cleanGenres = append(cleanGenres, g)
				}
			}

			// Count actual chapter .json files
			chapDir := filepath.Join(dir, slug, "chapters")
			actualChaps := 0
			if cEntries, err := os.ReadDir(chapDir); err == nil {
				for _, ce := range cEntries {
					if !ce.IsDir() && strings.HasSuffix(ce.Name(), ".json") {
						actualChaps++
					}
				}
			}

			totalChaps := raw.TotalChapters
			if actualChaps > totalChaps {
				totalChaps = actualChaps
			}
			rating := raw.RatingScore
			if rating <= 0 {
				rating = 9.5
			}

			result = append(result, domain.Novel{
				ID:             slug,
				Title:          raw.Title,
				Slug:           slug,
				AuthorName:     author,
				Genres:         cleanGenres,
				Description:    raw.Description,
				CoverURL:       raw.CoverURL,
				Status:         raw.Status,
				TotalChapters:  totalChaps,
				ActualChapters: actualChaps,
				ViewsCount:     int64(actualChaps * 180),
				RatingScore:    rating,
				Realm:          raw.Realm,
				SourceURL:      raw.SourceURL,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ActualChapters > result[j].ActualChapters
	})
	return result
}

// LoadNovelDetailFromDisk loads a novel and all its chapter metadata from disk
func (fs *FileStorage) LoadNovelDetailFromDisk(slugOrID string) (*domain.Novel, []domain.Chapter, error) {
	all := fs.LoadAllNovelsFromDisk()
	var target *domain.Novel
	for i := range all {
		if all[i].Slug == slugOrID || all[i].ID == slugOrID {
			target = &all[i]
			break
		}
	}
	if target == nil {
		return nil, nil, fmt.Errorf("novel not found on disk: %s", slugOrID)
	}

	var chapters []domain.Chapter
	for _, dir := range fs.candidateDirs() {
		chapDir := filepath.Join(dir, target.Slug, "chapters")
		cEntries, err := os.ReadDir(chapDir)
		if err != nil {
			continue
		}
		for _, ce := range cEntries {
			if ce.IsDir() || !strings.HasSuffix(ce.Name(), ".json") {
				continue
			}
			cBytes, err := os.ReadFile(filepath.Join(chapDir, ce.Name()))
			if err != nil {
				continue
			}
			var chap domain.Chapter
			if err := json.Unmarshal(cBytes, &chap); err == nil {
				chap.ID = fmt.Sprintf("%s__%d", target.Slug, chap.ChapterNumber)
				chap.NovelID = target.Slug
				chap.ContentText = "" // omit heavy content in directory listing
				chapters = append(chapters, chap)
			}
		}
		if len(chapters) > 0 {
			break
		}
	}

	sort.Slice(chapters, func(i, j int) bool {
		return chapters[i].ChapterNumber < chapters[j].ChapterNumber
	})
	return target, chapters, nil
}

// LoadChapterContentFromDisk loads a single chapter by composite ID ("slug__num")
func (fs *FileStorage) LoadChapterContentFromDisk(chapterID string) (*domain.Chapter, error) {
	parts := strings.Split(chapterID, "__")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid disk chapter ID: %s", chapterID)
	}
	slug := parts[0]
	num, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}

	for _, dir := range fs.candidateDirs() {
		jsonPath := filepath.Join(dir, slug, "chapters", fmt.Sprintf("%d.json", num))
		cBytes, err := os.ReadFile(jsonPath)
		if err != nil {
			continue
		}
		var chap domain.Chapter
		if err := json.Unmarshal(cBytes, &chap); err != nil {
			continue
		}
		chap.ID = chapterID
		chap.NovelID = slug
		if chap.ContentText == "" {
			if txtBytes, err := os.ReadFile(filepath.Join(dir, slug, "chapters", fmt.Sprintf("%d.txt", num))); err == nil {
				chap.ContentText = string(txtBytes)
			}
		}
		return &chap, nil
	}
	return nil, fmt.Errorf("chapter %s not found on disk", chapterID)
}
