package cleaner

import (
	"html"
	"regexp"
	"strings"

	"comics-backend/internal/domain"
)

var (
	// Regex for HTML tags
	reHTMLTags = regexp.MustCompile(`(?i)<[^>]*>`)
	reBR       = regexp.MustCompile(`(?i)<br\s*/?>`)
	rePStart   = regexp.MustCompile(`(?i)<p[^>]*>`)
	rePEnd     = regexp.MustCompile(`(?i)</p>`)
	reDivEnd   = regexp.MustCompile(`(?i)</div>`)

	// Regex for novel titles
	reTitlePrefix = regexp.MustCompile(`(?i)^\s*(\[dịch\]|\[convert\]|\[full\]|\(bản full\)|\(truyenfull\.?vn?\)|\(truyện full\)|\(truyenfulldich\))\s*`)
	reTitleSuffix = regexp.MustCompile(`(?i)\s*(\[dịch\]|\[convert\]|\[full\]|\(bản full\)|\(truyenfull\.?vn?\)|\(truyện full\)|\(truyenfulldich\))\s*$`)

	// Regex for chapter title duplicates like "Chương 1: Chương 1: Tên"
	reDupChapPrefix = regexp.MustCompile(`(?i)^(chương\s+\d+[\s:.-]*)+(chương\s+\d+[\s:.-]*)`)

	// Regex for watermark patterns in descriptions and chapter content
	reWatermarkLines = []*regexp.Regexp{
		regexp.MustCompile(`(?i)truyenfull\.vn|truyenfull\.live|truyenfulldich\.top|truyenfull\.com|tangthuvien\.net|metruyenchu\.com`),
		regexp.MustCompile(`(?i)^\s*nguồn\s*:\s*(truyenfull|truyện full|truyenfulldich|tangthuvien|sưu tầm|internet).*`),
		regexp.MustCompile(`(?i)bạn đang (đọc|xem|theo dõi) truyện (tại|trên)\s*(truyenfull|truyện full|truyenfulldich|tangthuvien).*`),
		regexp.MustCompile(`(?i)truyện được (đăng|dịch|re-?up|copy) (duy nhất )?(tại|trên)\s*.*`),
		regexp.MustCompile(`(?i)(đọc truyện|xem truyện|theo dõi) tại\s*(truyenfull|truyện full|truyenfulldich).*`),
		regexp.MustCompile(`(?i)chúc bạn đọc truyện vui vẻ tại\s*(truyenfull|truyện full).*`),
		regexp.MustCompile(`(?i)(like fanpage|tham gia group facebook|ủng hộ nhóm dịch|nhóm dịch:|mọi ý kiến đóng góp|mọi thắc mắc).*`),
		regexp.MustCompile(`(?i)https?://[^\s]+`),
	}

	// Regex for collapsing newlines
	reMultiNewlines = regexp.MustCompile(`\n{3,}`)
	reMultiSpaces   = regexp.MustCompile(`[ \t]{2,}`)
)

// CleanNovel sanitizes all novel metadata before inserting or updating in DB / Disk
func CleanNovel(n *domain.Novel) {
	if n == nil {
		return
	}

	// 1. Clean Title
	n.Title = CleanTitle(n.Title)

	// 2. Clean Author
	n.AuthorName = CleanAuthor(n.AuthorName)

	// 3. Clean Description
	n.Description = CleanDescription(n.Description)

	// 4. Clean Status
	n.Status = CleanStatus(n.Status)

	// 5. Clean Genres
	n.Genres = CleanGenres(n.Genres)
}

// CleanChapter sanitizes chapter title and content text before DB / Disk save
func CleanChapter(c *domain.Chapter) {
	if c == nil {
		return
	}

	// 1. Clean Chapter Title
	c.Title = CleanChapterTitle(c.Title, c.ChapterNumber)

	// 2. Clean Chapter Content
	c.ContentText = CleanChapterContent(c.ContentText)

	// 3. Recount words accurately
	c.WordCount = len(strings.Fields(c.ContentText))
}

// CleanTitle removes spam prefix/suffix and strips HTML
func CleanTitle(title string) string {
	t := html.UnescapeString(strings.TrimSpace(title))
	t = reHTMLTags.ReplaceAllString(t, "")
	t = reTitlePrefix.ReplaceAllString(t, "")
	t = reTitleSuffix.ReplaceAllString(t, "")
	t = reMultiSpaces.ReplaceAllString(t, " ")
	return strings.TrimSpace(t)
}

// CleanAuthor sanitizes author name
func CleanAuthor(author string) string {
	a := html.UnescapeString(strings.TrimSpace(author))
	a = reHTMLTags.ReplaceAllString(a, "")
	a = reMultiSpaces.ReplaceAllString(a, " ")
	a = strings.TrimSpace(a)

	lower := strings.ToLower(a)
	if a == "" || lower == "chưa có" || lower == "chưa cập nhật" || strings.Contains(lower, "truyenfull") {
		return "Đang cập nhật"
	}
	return a
}

// CleanStatus standardizes novel status to "Full" or "Đang ra"
func CleanStatus(status string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	if strings.Contains(s, "full") || strings.Contains(s, "hoàn") || strings.Contains(s, "trọn bộ") || strings.Contains(s, "completed") {
		return "Full"
	}
	return "Đang ra"
}

// CleanGenres strips whitespace and deduplicates genre tags
func CleanGenres(genres []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, g := range genres {
		cleanG := html.UnescapeString(strings.TrimSpace(g))
		cleanG = reHTMLTags.ReplaceAllString(cleanG, "")
		cleanG = strings.TrimSpace(cleanG)
		if cleanG == "" {
			continue
		}
		lower := strings.ToLower(cleanG)
		if !seen[lower] && !strings.Contains(lower, "truyenfull") {
			seen[lower] = true
			result = append(result, cleanG)
		}
	}
	if len(result) == 0 {
		return []string{"Tiên Hiệp", "Huyền Huyễn"}
	}
	return result
}

// CleanDescription removes HTML tags, advertising links and watermark lines
func CleanDescription(desc string) string {
	if strings.TrimSpace(desc) == "" {
		return ""
	}

	// 1. Convert block breaks to newlines
	text := reBR.ReplaceAllString(desc, "\n")
	text = rePEnd.ReplaceAllString(text, "\n")
	text = rePStart.ReplaceAllString(text, "")
	text = reDivEnd.ReplaceAllString(text, "\n")

	// 2. Strip remaining HTML tags & decode entities
	text = reHTMLTags.ReplaceAllString(text, "")
	text = html.UnescapeString(text)

	// 3. Filter lines line by line to remove watermarks and promo texts
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			cleanLines = append(cleanLines, "")
			continue
		}

		isWatermark := false
		for _, re := range reWatermarkLines {
			if re.MatchString(line) {
				isWatermark = true
				break
			}
		}

		if !isWatermark {
			cleanLines = append(cleanLines, line)
		}
	}

	cleaned := strings.Join(cleanLines, "\n")
	cleaned = reMultiNewlines.ReplaceAllString(cleaned, "\n\n")
	return strings.TrimSpace(cleaned)
}

// CleanChapterTitle formats chapter title cleanly without duplicate prefixes or watermarks
func CleanChapterTitle(title string, chapNum int) string {
	t := html.UnescapeString(strings.TrimSpace(title))
	t = reHTMLTags.ReplaceAllString(t, "")
	t = reMultiSpaces.ReplaceAllString(t, " ")

	// Fix duplicates: "Chương 10: Chương 10: Tên"
	if reDupChapPrefix.MatchString(t) {
		t = reDupChapPrefix.ReplaceAllString(t, "Chương $2")
	}

	// Remove source watermarks from title
	t = regexp.MustCompile(`(?i)\s*\[truyenfull(\.vn)?\]|\(truyenfull(\.vn)?\)`).ReplaceAllString(t, "")
	t = strings.TrimSpace(t)

	if t == "" && chapNum > 0 {
		t = strings.TrimSpace(html.EscapeString(string(rune(chapNum))))
	}
	return t
}

// CleanChapterContent strips HTML tags, advertising lines, and source watermarks from chapter text
func CleanChapterContent(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}

	// 1. Remove dangerous script/style tags completely
	text := regexp.MustCompile(`(?is)<(script|style|iframe)[^>]*>.*?</(script|style|iframe)>`).ReplaceAllString(content, "")

	// 2. Convert breaks to newlines
	text = reBR.ReplaceAllString(text, "\n")
	text = rePEnd.ReplaceAllString(text, "\n")
	text = rePStart.ReplaceAllString(text, "")
	text = reDivEnd.ReplaceAllString(text, "\n")

	// 3. Strip all HTML tags & unescape entities
	text = reHTMLTags.ReplaceAllString(text, "")
	text = html.UnescapeString(text)

	// 4. Line by line watermark filtering
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			cleanLines = append(cleanLines, "")
			continue
		}

		isWatermark := false
		for _, re := range reWatermarkLines {
			if re.MatchString(trimmed) {
				isWatermark = true
				break
			}
		}

		if !isWatermark {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	cleaned := strings.Join(cleanLines, "\n")
	cleaned = reMultiNewlines.ReplaceAllString(cleaned, "\n\n")
	return strings.TrimSpace(cleaned)
}

