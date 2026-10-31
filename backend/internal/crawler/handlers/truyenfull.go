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

type TruyenFullHandler struct {
	client *fetcher.HTTPClient
}

func NewTruyenFullHandler(client *fetcher.HTTPClient) *TruyenFullHandler {
	return &TruyenFullHandler{client: client}
}

func (h *TruyenFullHandler) Name() string {
	return "Truyện Full"
}

func (h *TruyenFullHandler) Domain() string {
	return "truyenfull.live"
}

func (h *TruyenFullHandler) CanHandle(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return strings.Contains(rawURL, "truyenfull.live") || strings.Contains(rawURL, "truyenfull.vn")
	}
	host := strings.ToLower(parsed.Hostname())
	return strings.Contains(host, "truyenfull.live") || strings.Contains(host, "truyenfull.vn")
}

func (h *TruyenFullHandler) DiscoverNovels(catalogURL string, maxPages int) ([]domain.DiscoveredNovel, error) {
	var results []domain.DiscoveredNovel
	seen := make(map[string]bool)

	baseURL := strings.TrimRight(catalogURL, "/")
	if maxPages <= 0 {
		maxPages = 1
	}

	for page := 1; page <= maxPages; page++ {
		pageURL := baseURL
		if page > 1 {
			pageURL = fmt.Sprintf("%s/trang-%d/", baseURL, page)
		}

		doc, err := h.client.FetchDoc(pageURL, baseURL)
		if err != nil {
			break
		}

		rows := doc.Find(".list-truyen .row[itemscope], #list-page .list-truyen .row")
		if rows.Length() == 0 {
			break
		}

		rows.Each(func(i int, s *goquery.Selection) {
			titleElem := s.Find(".truyen-title a")
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

			author := strings.TrimSpace(s.Find(".author").Text())
			author = strings.TrimSpace(strings.TrimPrefix(author, "Tác giả:"))

			cover := ""
			if img, ok := s.Find(".lazyimg").Attr("data-image"); ok && img != "" {
				cover = img
			} else if img, ok := s.Find(".lazyimg").Attr("data-desk-image"); ok && img != "" {
				cover = img
			} else if img, ok := s.Find("img").Attr("src"); ok {
				cover = img
			}

			results = append(results, domain.DiscoveredNovel{
				Title:        title,
				Slug:         slug,
				SourceURL:    href + "/",
				SourceDomain: h.Domain(),
				AuthorName:   author,
				CoverURL:     cover,
			})
		})

		time.Sleep(250 * time.Millisecond)
	}

	return results, nil
}

func (h *TruyenFullHandler) ParseNovel(novelURL string) (*domain.Novel, string, error) {
	doc, err := h.client.FetchDoc(novelURL, "https://truyenfull.live/")
	if err != nil {
		return nil, "", err
	}

	title := strings.TrimSpace(doc.Find("h3.title[itemprop='name']").Text())
	if title == "" {
		title = strings.TrimSpace(doc.Find("h1.truyen-title").Text())
	}
	if title == "" {
		title = strings.TrimSpace(doc.Find("h1").First().Text())
	}

	author := strings.TrimSpace(doc.Find("a[itemprop='author']").Text())
	if author == "" {
		author = "Đang cập nhật"
	}

	var genres []string
	doc.Find("a[itemprop='genre']").Each(func(i int, s *goquery.Selection) {
		g := strings.TrimSpace(s.Text())
		if g != "" {
			genres = append(genres, g)
		}
	})

	statusText := strings.TrimSpace(doc.Find(".info span.text-success").Text())
	if statusText == "" {
		statusText = "Đang ra"
	}

	desc := strings.TrimSpace(doc.Find(".desc-text").Text())
	cover, _ := doc.Find(".books .book img").Attr("src")

	slug := strings.Trim(strings.TrimPrefix(novelURL, "https://truyenfull.live/"), "/")
	slug = strings.TrimPrefix(slug, "http://truyenfull.live/")
	parts := strings.Split(slug, "/")
	if len(parts) > 0 {
		slug = parts[0]
	}

	firstChapterURL, _ := doc.Find("#list-chapter .list-chapter li a").First().Attr("href")

	novel := &domain.Novel{
		Title:        title,
		Slug:         slug,
		AuthorName:   author,
		Genres:       genres,
		Description:  desc,
		CoverURL:     cover,
		Status:       statusText,
		RatingScore:  9.6,
		SourceURL:    novelURL,
		SourceDomain: h.Domain(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return novel, firstChapterURL, nil
}

func (h *TruyenFullHandler) ParseChapter(chapterURL string, defaultChapterNum int, referer string) (*domain.Chapter, string, error) {
	doc, err := h.client.FetchDoc(chapterURL, referer)
	if err != nil {
		return nil, "", err
	}

	title := strings.TrimSpace(doc.Find("h2 a.chapter-title").Text())
	if title == "" {
		title = fmt.Sprintf("Chương %d", defaultChapterNum)
	}

	chapterNum := defaultChapterNum
	if numVal, exists := doc.Find("#chapter-num").Attr("value"); exists {
		if n, err := strconv.Atoi(numVal); err == nil && n > 0 {
			chapterNum = n
		}
	}

	contentDiv := doc.Find("#chapter-c")
	cleanedText := cleaner.CleanChapterHTML(contentDiv)
	wordCount := len(strings.Fields(cleanedText))

	nextURL, _ := doc.Find("#next_chap").Attr("href")
	if strings.Contains(nextURL, "javascript:void") || nextURL == chapterURL {
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
