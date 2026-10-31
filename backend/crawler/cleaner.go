package main

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	// Regex tìm các đoạn watermark ngầm của TruyenFull
	watermarkRegex = regexp.MustCompile(`(?i)(truyen\s*full|truyenfull\.vn|truyenfulllive|truyenfull|đọc\s*truyện\s*tại\s*truyenfull)`)
	multipleSpaces = regexp.MustCompile(`[ \t]+`)
	multipleNewlines = regexp.MustCompile(`\n{3,}`)
)

// CleanChapterHTML làm sạch DOM của chương truyện
func CleanChapterHTML(s *goquery.Selection) string {
	// 1. Xóa toàn bộ các thẻ script, style, iframe, quảng cáo
	s.Find("script, style, iframe, noscript, .ads-unlock-container, .ads-unlock-reminder, #ads-chapter-top, #ads-chapter-bottom, .show_ads_google").Remove()

	// 2. Xóa các thẻ ẩn dùng để chống copy (display: none hoặc visibility: hidden)
	s.Find("*").Each(func(i int, el *goquery.Selection) {
		style, exists := el.Attr("style")
		if exists {
			lowerStyle := strings.ToLower(style)
			if strings.Contains(lowerStyle, "display: none") || strings.Contains(lowerStyle, "visibility: hidden") || strings.Contains(lowerStyle, "display:none") {
				el.Remove()
			}
		}
	})

	// 3. Thay thẻ <br> thành dấu xuống dòng \n
	s.Find("br").ReplaceWithHtml("\n")

	// 4. Lấy toàn bộ text sạch
	rawText := s.Text()

	// 5. Chuẩn hóa text
	lines := strings.Split(rawText, "\n")
	var cleanedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = multipleSpaces.ReplaceAllString(line, " ")

		// Bỏ qua dòng rỗng hoặc dòng chứa watermark
		if line == "" || watermarkRegex.MatchString(line) {
			continue
		}

		cleanedLines = append(cleanedLines, line)
	}

	result := strings.Join(cleanedLines, "\n\n")
	result = multipleNewlines.ReplaceAllString(result, "\n\n")

	return strings.TrimSpace(result)
}
