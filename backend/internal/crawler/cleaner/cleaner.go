package cleaner

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	// Regex finding hidden watermarks of story sites
	watermarkRegex = regexp.MustCompile(`(?i)(truyen\s*full|truyenfull\.vn|truyenfulllive|truyenfull|đọc\s*truyện\s*tại\s*truyenfull|tang\s*thu\s*vien|tangthuvien|metruyenchu)`)
	multipleSpaces = regexp.MustCompile(`[ \t]+`)
	multipleNewlines = regexp.MustCompile(`\n{3,}`)
)

// CleanChapterHTML cleans the DOM of chapter content, stripping ads, hidden anti-copy tags, and watermarks
func CleanChapterHTML(s *goquery.Selection) string {
	// 1. Remove scripts, styles, iframes, ads containers
	s.Find("script, style, iframe, noscript, .ads-unlock-container, .ads-unlock-reminder, #ads-chapter-top, #ads-chapter-bottom, .show_ads_google, .adsense, .banner").Remove()

	// 2. Remove hidden anti-copy elements (display: none or visibility: hidden)
	s.Find("*").Each(func(i int, el *goquery.Selection) {
		if style, exists := el.Attr("style"); exists {
			lowerStyle := strings.ToLower(style)
			if strings.Contains(lowerStyle, "display: none") || strings.Contains(lowerStyle, "visibility: hidden") || strings.Contains(lowerStyle, "display:none") {
				el.Remove()
			}
		}
	})

	// 3. Replace <br> with newline \n
	s.Find("br").ReplaceWithHtml("\n")

	// 4. Extract text
	rawText := s.Text()

	// 5. Clean up lines
	lines := strings.Split(rawText, "\n")
	var cleanedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = multipleSpaces.ReplaceAllString(line, " ")

		// Skip empty lines or obvious watermarks
		if line == "" || watermarkRegex.MatchString(line) {
			continue
		}

		cleanedLines = append(cleanedLines, line)
	}

	result := strings.Join(cleanedLines, "\n\n")
	result = multipleNewlines.ReplaceAllString(result, "\n\n")

	return strings.TrimSpace(result)
}
