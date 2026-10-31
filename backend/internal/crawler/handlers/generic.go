package handlers

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"comics-backend/internal/crawler/cleaner"
	"comics-backend/internal/crawler/fetcher"
	"comics-backend/internal/domain"
)

// GenericHandler serves as a smart fallback parser for standard HTML story websites
type GenericHandler struct {
	client *fetcher.HTTPClient
}

func NewGenericHandler(client *fetcher.HTTPClient) *GenericHandler {
	return &GenericHandler{client: client}
}

func (h *GenericHandler) Name() string {
	return "Bộ Phân Tích Tổng Quát (Generic Novel Parser)"
}

func (h *GenericHandler) Domain() string {
	return "generic"
}

func (h *GenericHandler) CanHandle(rawURL string) bool {
	return true // Fallback matches any valid URL
}

func (h *GenericHandler) DiscoverNovels(catalogURL string, maxPages int) ([]domain.DiscoveredNovel, error) {
	doc, err := h.client.FetchDoc(catalogURL, "")
	if err != nil {
		return nil, err
	}

	var results []domain.DiscoveredNovel
	seen := make(map[string]bool)

	// Look for lists of links that look like stories
	doc.Find("article, .story-item, .novel-item, .item, .row").Each(func(i int, s *goquery.Selection) {
		link := s.Find("a[href]").First()
		href, _ := link.Attr("href")
		title := strings.TrimSpace(link.Text())
		if title == "" {
			title, _ = link.Attr("title")
		}

		if href != "" && title != "" && len(title) > 3 {
			if !seen[href] {
				seen[href] = true
				results = append(results, domain.DiscoveredNovel{
					Title:        title,
					Slug:         strings.ToLower(strings.ReplaceAll(title, " ", "-")),
					SourceURL:    href,
					SourceDomain: "generic",
				})
			}
		}
	})

	return results, nil
}

func (h *GenericHandler) ParseNovel(novelURL string) (*domain.Novel, string, error) {
	doc, err := h.client.FetchDoc(novelURL, "")
	if err != nil {
		return nil, "", err
	}

	title := strings.TrimSpace(doc.Find("h1").First().Text())
	if title == "" {
		title = strings.TrimSpace(doc.Find("meta[property='og:title']").AttrOr("content", "Truyện Chưa Đặt Tên"))
	}

	author := strings.TrimSpace(doc.Find(".author, a[rel='author'], [itemprop='author']").First().Text())
	if author == "" {
		author = "Đang cập nhật"
	}

	desc := strings.TrimSpace(doc.Find(".description, .desc, .intro, [itemprop='description'], meta[name='description']").First().Text())
	cover, _ := doc.Find("meta[property='og:image']").Attr("content")
	if cover == "" {
		cover, _ = doc.Find(".cover img, .book-img img, img.thumbnail").First().Attr("src")
	}

	parsed, _ := url.Parse(novelURL)
	domainName := "generic"
	if parsed != nil {
		domainName = parsed.Hostname()
	}

	slug := strings.Trim(parsed.Path, "/")
	parts := strings.Split(slug, "/")
	if len(parts) > 0 && parts[len(parts)-1] != "" {
		slug = parts[len(parts)-1]
	}

	// Look for first chapter link
	firstChapterURL := ""
	doc.Find("a[href*='chuong-1'], a[href*='chap-1'], a[href*='chapter-1']").First().Each(func(i int, s *goquery.Selection) {
		firstChapterURL, _ = s.Attr("href")
	})

	novel := &domain.Novel{
		Title:        title,
		Slug:         slug,
		AuthorName:   author,
		Description:  desc,
		CoverURL:     cover,
		Status:       "Đang ra",
		RatingScore:  9.5,
		SourceURL:    novelURL,
		SourceDomain: domainName,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return novel, firstChapterURL, nil
}

func (h *GenericHandler) ParseChapter(chapterURL string, defaultChapterNum int, referer string) (*domain.Chapter, string, error) {
	doc, err := h.client.FetchDoc(chapterURL, referer)
	if err != nil {
		return nil, "", err
	}

	title := strings.TrimSpace(doc.Find("h1, h2, .chapter-title, .title").First().Text())
	if title == "" {
		title = fmt.Sprintf("Chương %d", defaultChapterNum)
	}

	contentDiv := doc.Find(".chapter-content, #chapter-c, .content, #content, .entry-content, article")
	cleanedText := cleaner.CleanChapterHTML(contentDiv)
	wordCount := len(strings.Fields(cleanedText))

	nextURL, _ := doc.Find("a[rel='next'], a:contains('Chương sau'), a:contains('Next'), a#next_chap").First().Attr("href")
	if strings.Contains(nextURL, "javascript") || nextURL == chapterURL {
		nextURL = ""
	}

	chapter := &domain.Chapter{
		ChapterNumber: defaultChapterNum,
		Title:         title,
		Slug:          fmt.Sprintf("chuong-%d", defaultChapterNum),
		ContentText:   cleanedText,
		WordCount:     wordCount,
		SourceURL:     chapterURL,
		NextURL:       nextURL,
		CreatedAt:     time.Now(),
	}

	return chapter, nextURL, nil
}
