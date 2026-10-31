package handlers

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"comics-backend/internal/crawler/cleaner"
	"comics-backend/internal/crawler/fetcher"
	"comics-backend/internal/domain"
)

type TangThuVienHandler struct {
	client *fetcher.HTTPClient
}

func NewTangThuVienHandler(client *fetcher.HTTPClient) *TangThuVienHandler {
	return &TangThuVienHandler{client: client}
}

func (h *TangThuVienHandler) Name() string {
	return "Tàng Thư Viện"
}

func (h *TangThuVienHandler) Domain() string {
	return "tangthuvien.net"
}

func (h *TangThuVienHandler) CanHandle(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return strings.Contains(rawURL, "tangthuvien")
	}
	host := strings.ToLower(parsed.Hostname())
	return strings.Contains(host, "tangthuvien")
}

func (h *TangThuVienHandler) DiscoverNovels(catalogURL string, maxPages int) ([]domain.DiscoveredNovel, error) {
	var results []domain.DiscoveredNovel
	seen := make(map[string]bool)

	baseURL := strings.TrimRight(catalogURL, "/")
	if maxPages <= 0 {
		maxPages = 1
	}

	for page := 1; page <= maxPages; page++ {
		pageURL := baseURL
		if page > 1 {
			pageURL = fmt.Sprintf("%s?page=%d", baseURL, page)
		}

		doc, err := h.client.FetchDoc(pageURL, baseURL)
		if err != nil {
			break
		}

		items := doc.Find(".book-img-text li, .rank-view-list li, .story-list li")
		if items.Length() == 0 {
			break
		}

		items.Each(func(i int, s *goquery.Selection) {
			titleElem := s.Find("h4 a, .book-mid-info h4 a, .title a").First()
			title := strings.TrimSpace(titleElem.Text())
			href, _ := titleElem.Attr("href")
			if title == "" || href == "" {
				return
			}

			href = strings.TrimRight(href, "/")
			slug := href
			if idx := strings.LastIndex(href, "/"); idx != -1 {
				slug = href[idx+1:]
			}

			if seen[slug] {
				return
			}
			seen[slug] = true

			author := strings.TrimSpace(s.Find(".author a, .name").First().Text())
			cover, _ := s.Find("img").First().Attr("src")

			results = append(results, domain.DiscoveredNovel{
				Title:        title,
				Slug:         slug,
				SourceURL:    href,
				SourceDomain: h.Domain(),
				AuthorName:   author,
				CoverURL:     cover,
			})
		})

		time.Sleep(250 * time.Millisecond)
	}

	return results, nil
}

func (h *TangThuVienHandler) ParseNovel(novelURL string) (*domain.Novel, string, error) {
	doc, err := h.client.FetchDoc(novelURL, "https://tangthuvien.net/")
	if err != nil {
		return nil, "", err
	}

	title := strings.TrimSpace(doc.Find(".book-info h1, .story-info h1").First().Text())
	if title == "" {
		title = strings.TrimSpace(doc.Find("h1").First().Text())
	}

	author := strings.TrimSpace(doc.Find(".book-info p.tag a, .story-info .author").First().Text())
	if author == "" {
		author = "Đang cập nhật"
	}

	var genres []string
	doc.Find(".book-info p.tag a, .story-info .tags a").Each(func(i int, s *goquery.Selection) {
		g := strings.TrimSpace(s.Text())
		if g != "" && g != author {
			genres = append(genres, g)
		}
	})

	desc := strings.TrimSpace(doc.Find(".book-intro, .story-intro").First().Text())
	cover, _ := doc.Find(".book-img img, .story-img img").First().Attr("src")

	slug := strings.Trim(strings.TrimPrefix(novelURL, "https://tangthuvien.net/doc-truyen/"), "/")
	slug = strings.TrimPrefix(slug, "https://tangthuvien.net/")
	parts := strings.Split(slug, "/")
	if len(parts) > 0 {
		slug = parts[0]
	}

	firstChapterURL, _ := doc.Find("a.btn-read, .read-btn a, #list-chapter a").First().Attr("href")
	if firstChapterURL != "" && !strings.HasPrefix(firstChapterURL, "http") {
		firstChapterURL = "https://tangthuvien.net" + firstChapterURL
	}

	novel := &domain.Novel{
		Title:        title,
		Slug:         slug,
		AuthorName:   author,
		Genres:       genres,
		Description:  desc,
		CoverURL:     cover,
		Status:       "Đang ra",
		RatingScore:  9.5,
		SourceURL:    novelURL,
		SourceDomain: h.Domain(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return novel, firstChapterURL, nil
}

func (h *TangThuVienHandler) ParseChapter(chapterURL string, defaultChapterNum int, referer string) (*domain.Chapter, string, error) {
	doc, err := h.client.FetchDoc(chapterURL, referer)
	if err != nil {
		return nil, "", err
	}

	title := strings.TrimSpace(doc.Find("h2.chapter-title, .title-chapter, h2").First().Text())
	if title == "" {
		title = fmt.Sprintf("Chương %d", defaultChapterNum)
	}

	chapterNum := defaultChapterNum
	if numVal, exists := doc.Find("#chapter-num").Attr("value"); exists {
		if n, err := strconv.Atoi(numVal); err == nil && n > 0 {
			chapterNum = n
		}
	}

	contentDiv := doc.Find(".box-chap, .chapter-c, .content-chapter")
	cleanedText := cleaner.CleanChapterHTML(contentDiv)
	wordCount := len(strings.Fields(cleanedText))

	nextURL, _ := doc.Find("a.btn-next, #next_chap, a:contains('Chương sau')").First().Attr("href")
	if nextURL != "" && !strings.HasPrefix(nextURL, "http") {
		nextURL = "https://tangthuvien.net" + nextURL
	}
	if strings.Contains(nextURL, "javascript") || nextURL == chapterURL {
		nextURL = ""
	}

	chapter := &domain.Chapter{
		ChapterNumber: chapterNum,
		Title:         title,
		Slug:          fmt.Sprintf("chuong-%d", chapterNum),
		ContentText:   cleanedText,
		WordCount:     wordCount,
		SourceURL:     chapterURL,
		NextURL:       nextURL,
		CreatedAt:     time.Now(),
	}

	return chapter, nextURL, nil
}
