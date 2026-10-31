package handlers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"comics-backend/internal/crawler/fetcher"
	"comics-backend/internal/domain"
)

// TruyenFullDichHandler crawls novels from https://truyenfulldich.top/
type TruyenFullDichHandler struct {
	client *fetcher.HTTPClient
}

func NewTruyenFullDichHandler(client *fetcher.HTTPClient) *TruyenFullDichHandler {
	return &TruyenFullDichHandler{client: client}
}

func (h *TruyenFullDichHandler) Name() string {
	return "Truyện Full Dịch"
}

func (h *TruyenFullDichHandler) Domain() string {
	return "truyenfulldich.top"
}

func (h *TruyenFullDichHandler) CanHandle(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return strings.Contains(rawURL, "truyenfulldich")
	}
	host := strings.ToLower(parsed.Hostname())
	return strings.Contains(host, "truyenfulldich")
}

// InertiaAppPayload models the JSON structure embedded in <script data-page="app" type="application/json">
type InertiaAppPayload struct {
	Props struct {
		StoryProp struct {
			ID            int    `json:"id"`
			Name          string `json:"name"`
			Alias         string `json:"alias"`
			Image         string `json:"image"`
			View          int    `json:"view"`
			Status        int    `json:"status"`
			TotalChapters int    `json:"total_chapters"`
			Description   string `json:"description"`
			Authors       []struct {
				Name string `json:"name"`
			} `json:"authors"`
			Categories []struct {
				Name string `json:"name"`
			} `json:"categories"`
		} `json:"storyProp"`
		ChapterProp struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Subname string `json:"subname"`
			Alias   string `json:"alias"`
			Content string `json:"content"`
		} `json:"chapterProp"`
		NextChapterProp *struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Subname string `json:"subname"`
			Alias   string `json:"alias"`
		} `json:"nextChapterProp"`
	} `json:"props"`
}

func extractInertiaPayload(html string) (*InertiaAppPayload, error) {
	marker := `type="application/json">`
	idx := strings.Index(html, marker)
	if idx == -1 {
		return nil, fmt.Errorf("inertia app payload marker not found")
	}

	start := idx + len(marker)
	end := strings.Index(html[start:], "</script>")
	if end == -1 {
		return nil, fmt.Errorf("closing script tag not found")
	}

	rawJSON := html[start : start+end]
	var payload InertiaAppPayload
	if err := json.Unmarshal([]byte(rawJSON), &payload); err != nil {
		return nil, fmt.Errorf("failed to parse inertia json: %w", err)
	}

	return &payload, nil
}

// DiscoverNovels parses the catalog or listing pages of truyenfulldich.top
func (h *TruyenFullDichHandler) DiscoverNovels(catalogURL string, maxPages int) ([]domain.DiscoveredNovel, error) {
	var results []domain.DiscoveredNovel
	seen := make(map[string]bool)

	// If catalogURL is the sitemap or if it's the default listing, sitemap-stories.xml provides 862 novels cleanly
	if strings.Contains(catalogURL, "sitemap") || strings.Contains(catalogURL, "danh-sach") || strings.Contains(catalogURL, "HOT_COMBO") {
		sitemapURL := "https://truyenfulldich.top/sitemap-stories.xml"
		rawXML, err := h.client.FetchRaw(sitemapURL, "https://truyenfulldich.top/")
		if err == nil && len(rawXML) > 0 {
			reLoc := regexp.MustCompile(`<loc>(https://truyenfulldich\.top/([^<]+))</loc>`)
			matches := reLoc.FindAllStringSubmatch(rawXML, -1)
			limit := len(matches)
			for i := 0; i < limit; i++ {
				m := matches[i]
				fullURL := m[1]
				slug := m[2]
				if slug != "" && !seen[slug] && !strings.Contains(slug, ".xml") {
					seen[slug] = true
					title := strings.Title(strings.ReplaceAll(slug, "-", " "))
					results = append(results, domain.DiscoveredNovel{
						Title:        title,
						Slug:         slug,
						SourceURL:    fullURL,
						SourceDomain: h.Domain(),
						CoverURL:     fmt.Sprintf("https://truyenfulldich.top/%s", slug),
					})
				}
			}
			if len(results) > 0 {
				return results, nil
			}
		}
	}

	baseURL := strings.TrimRight(catalogURL, "/")
	if maxPages <= 0 {
		maxPages = 1
	}

	for page := 1; page <= maxPages; page++ {
		pageURL := baseURL
		if page > 1 {
			if strings.Contains(baseURL, "?") {
				pageURL = fmt.Sprintf("%s&page=%d", baseURL, page)
			} else {
				pageURL = fmt.Sprintf("%s?page=%d", baseURL, page)
			}
		}

		doc, err := h.client.FetchDoc(pageURL, "https://truyenfulldich.top/")
		if err != nil {
			break
		}

		foundOnPage := 0
		doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			href = strings.TrimSpace(href)

			// Match links like /tao-hoa-chi-mon or https://truyenfulldich.top/tao-hoa-chi-mon
			if href == "" || strings.HasPrefix(href, "/danh-sach") || strings.HasPrefix(href, "/tai-khoan") ||
				strings.HasPrefix(href, "/dang-") || strings.HasPrefix(href, "/tos") || strings.Contains(href, ".apk") ||
				strings.Contains(href, "/chuong-") || href == "/" || href == "#" {
				return
			}

			fullURL := href
			if strings.HasPrefix(href, "/") {
				fullURL = "https://truyenfulldich.top" + href
			} else if !strings.HasPrefix(href, "http") {
				fullURL = "https://truyenfulldich.top/" + href
			}

			parsed, err := url.Parse(fullURL)
			if err != nil || !strings.Contains(parsed.Hostname(), "truyenfulldich.top") {
				return
			}

			slug := strings.Trim(parsed.Path, "/")
			if slug == "" || strings.Contains(slug, "/") || strings.Contains(slug, "?") {
				return
			}

			if seen[slug] {
				return
			}
			seen[slug] = true
			foundOnPage++

			title := strings.TrimSpace(s.Text())
			if title == "" {
				title = strings.ReplaceAll(slug, "-", " ")
			}

			cover := ""
			img := s.Find("img").First()
			if img.Length() > 0 {
				if src, ok := img.Attr("src"); ok {
					cover = src
				}
			}

			results = append(results, domain.DiscoveredNovel{
				Title:        title,
				Slug:         slug,
				SourceURL:    fullURL,
				SourceDomain: h.Domain(),
				CoverURL:     cover,
			})
		})

		if foundOnPage == 0 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}

	return results, nil
}

// ParseNovel extracts overview info of a story from truyenfulldich.top
func (h *TruyenFullDichHandler) ParseNovel(novelURL string) (*domain.Novel, string, error) {
	body, err := h.client.FetchRaw(novelURL, "https://truyenfulldich.top/")
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch novel page: %w", err)
	}

	payload, err := extractInertiaPayload(body)
	if err != nil {
		// Fallback to HTML parser if Inertia JSON fails
		return h.parseNovelFallbackHTML(novelURL, body)
	}

	story := payload.Props.StoryProp
	title := strings.TrimSpace(story.Name)
	slug := story.Alias
	if slug == "" {
		parsed, _ := url.Parse(novelURL)
		slug = strings.Trim(parsed.Path, "/")
	}

	author := "Đang cập nhật"
	if len(story.Authors) > 0 && story.Authors[0].Name != "" {
		author = strings.TrimSpace(story.Authors[0].Name)
	}

	var genres []string
	for _, c := range story.Categories {
		if c.Name != "" {
			genres = append(genres, strings.TrimSpace(c.Name))
		}
	}

	statusText := "Đang ra"
	if story.Status == 1 || story.Status == 2 {
		statusText = "Full"
	}

	cleanDesc := stripSimpleHTML(story.Description)
	cover := story.Image

	// Discover first chapter URL by scanning the HTML for /chuong-1
	firstChapterURL := ""
	reChap := regexp.MustCompile(fmt.Sprintf(`href="(/%s/chuong-1)"`, regexp.QuoteMeta(slug)))
	match := reChap.FindStringSubmatch(body)
	if len(match) > 1 {
		firstChapterURL = "https://truyenfulldich.top" + match[1]
	} else {
		firstChapterURL = fmt.Sprintf("https://truyenfulldich.top/%s/chuong-1", slug)
	}

	novel := &domain.Novel{
		Title:         title,
		Slug:          slug,
		AuthorName:    author,
		Genres:        genres,
		Description:   cleanDesc,
		CoverURL:      cover,
		Status:        statusText,
		TotalChapters: story.TotalChapters,
		ViewsCount:    int64(story.View),
		RatingScore:   9.8,
		SourceURL:     novelURL,
	}

	return novel, firstChapterURL, nil
}

func (h *TruyenFullDichHandler) parseNovelFallbackHTML(novelURL string, body string) (*domain.Novel, string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil, "", err
	}

	title := strings.TrimSpace(doc.Find("h1").First().Text())
	slug := strings.Trim(strings.TrimPrefix(novelURL, "https://truyenfulldich.top/"), "/")

	cover, _ := doc.Find("img[src*='postimg.cc']").First().Attr("src")
	firstChapterURL := fmt.Sprintf("https://truyenfulldich.top/%s/chuong-1", slug)

	novel := &domain.Novel{
		Title:       title,
		Slug:        slug,
		AuthorName:  "Đang cập nhật",
		CoverURL:    cover,
		Status:      "Đang ra",
		RatingScore: 9.8,
		SourceURL:   novelURL,
	}

	return novel, firstChapterURL, nil
}

// ParseChapter extracts chapter content, title, word count, and next chapter URL
func (h *TruyenFullDichHandler) ParseChapter(chapterURL string, defaultChapterNum int, referer string) (*domain.Chapter, string, error) {
	body, err := h.client.FetchRaw(chapterURL, "https://truyenfulldich.top/")
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch chapter page: %w", err)
	}

	payload, err := extractInertiaPayload(body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to extract chapter payload: %w", err)
	}

	chapData := payload.Props.ChapterProp
	chapterNum := defaultChapterNum

	// Extract chapter number from subname (e.g. "Chương 1" -> 1)
	reNum := regexp.MustCompile(`(?i)(?:chương|c|chap)\s*(\d+)`)
	if m := reNum.FindStringSubmatch(chapData.Subname); len(m) > 1 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			chapterNum = n
		}
	} else if m := reNum.FindStringSubmatch(chapData.Name); len(m) > 1 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			chapterNum = n
		}
	}

	fullTitle := strings.TrimSpace(chapData.Subname)
	if chapData.Name != "" && chapData.Name != chapData.Subname {
		if fullTitle != "" {
			fullTitle = fullTitle + ": " + strings.TrimSpace(chapData.Name)
		} else {
			fullTitle = strings.TrimSpace(chapData.Name)
		}
	}
	if fullTitle == "" {
		fullTitle = fmt.Sprintf("Chương %d", chapterNum)
	}

	// Clean content: replace <br /> or <br> with newlines, then strip tags
	rawHTML := chapData.Content
	rawHTML = strings.ReplaceAll(rawHTML, "<br />", "\n")
	rawHTML = strings.ReplaceAll(rawHTML, "<br/>", "\n")
	rawHTML = strings.ReplaceAll(rawHTML, "<br>", "\n")
	rawHTML = strings.ReplaceAll(rawHTML, "</p>", "\n\n")

	cleanText := stripSimpleHTML(rawHTML)
	wordCount := len(strings.Fields(cleanText))

	// Find next chapter URL
	nextChapterURL := ""
	if payload.Props.NextChapterProp != nil && payload.Props.NextChapterProp.Alias != "" {
		storyAlias := payload.Props.StoryProp.Alias
		if storyAlias == "" {
			parsedURL, _ := url.Parse(chapterURL)
			pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
			if len(pathParts) > 0 {
				storyAlias = pathParts[0]
			}
		}
		nextChapterURL = fmt.Sprintf("https://truyenfulldich.top/%s/%s", storyAlias, payload.Props.NextChapterProp.Alias)
	} else {
		// Fallback check
		nextNum := chapterNum + 1
		parsedURL, _ := url.Parse(chapterURL)
		pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(pathParts) > 0 {
			novelSlug := pathParts[0]
			expectedNextSub := fmt.Sprintf("/%s/chuong-%d", novelSlug, nextNum)
			if strings.Contains(body, expectedNextSub) || strings.Contains(body, fmt.Sprintf(`"alias":"chuong-%d"`, nextNum)) {
				nextChapterURL = fmt.Sprintf("https://truyenfulldich.top/%s/chuong-%d", novelSlug, nextNum)
			}
		}
	}

	chapter := &domain.Chapter{
		ChapterNumber: chapterNum,
		Title:         fullTitle,
		ContentText:   cleanText,
		WordCount:     wordCount,
		SourceURL:     chapterURL,
	}

	return chapter, nextChapterURL, nil
}

var tagRegex = regexp.MustCompile(`<[^>]*>`)
var spaceRegex = regexp.MustCompile(`[ \t]+`)

func stripSimpleHTML(s string) string {
	s = tagRegex.ReplaceAllString(s, " ")
	s = spaceRegex.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
