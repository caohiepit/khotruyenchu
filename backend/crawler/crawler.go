package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Crawler struct {
	client      *http.Client
	MaxRetries  int
	BaseBackoff time.Duration
}

var realisticUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:125.0) Gecko/20100101 Firefox/125.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
}

func getRandomUserAgent() string {
	idx := time.Now().UnixNano() % int64(len(realisticUserAgents))
	return realisticUserAgents[idx]
}

func NewCrawler() *Crawler {
	return &Crawler{
		client: &http.Client{
			Timeout: 25 * time.Second,
		},
		MaxRetries:  4,
		BaseBackoff: 2 * time.Second,
	}
}

// fetchDoc thực hiện tải trang với cơ chế Exponential Backoff Auto-Retry và xoay vòng User-Agent
func (c *Crawler) fetchDoc(targetURL string, referer string) (*goquery.Document, error) {
	var lastErr error

	for attempt := 1; attempt <= c.MaxRetries; attempt++ {
		req, err := http.NewRequest("GET", targetURL, nil)
		if err != nil {
			return nil, err
		}

		// Giả lập Headers trình duyệt thật
		ua := getRandomUserAgent()
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en-US;q=0.8,en;q=0.7")
		req.Header.Set("Sec-Ch-Ua", `"Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"`)
		req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
		req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
		req.Header.Set("Sec-Fetch-Dest", "document")
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		req.Header.Set("Sec-Fetch-User", "?1")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		req.Header.Set("Connection", "keep-alive")

		if referer != "" {
			req.Header.Set("Referer", referer)
		} else {
			req.Header.Set("Referer", "https://truyenfull.live/")
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("lỗi mạng (%v)", err)
		} else {
			if resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				return goquery.NewDocumentFromReader(resp.Body)
			}

			resp.Body.Close()

			// Nếu gặp 404 Not Found thì dừng luôn, không cần retry
			if resp.StatusCode == http.StatusNotFound {
				return nil, fmt.Errorf("chương hoặc trang không tồn tại (HTTP 404)")
			}

			lastErr = fmt.Errorf("máy chủ phản hồi mã HTTP %d", resp.StatusCode)
		}

		// Nếu chưa phải lần thử cuối cùng, thực hiện Exponential Backoff + Jitter
		if attempt < c.MaxRetries {
			// Lũy tiến: lần 1: 2s, lần 2: 4s, lần 3: 8s... kèm random jitter 200-800ms
			multiplier := 1 << (attempt - 1) // 1, 2, 4...
			jitterMs := int(time.Now().UnixNano()%600) + 200
			sleepDuration := time.Duration(multiplier)*c.BaseBackoff + time.Duration(jitterMs)*time.Millisecond

			fmt.Printf("\n      ⚠️ [Gặp sự cố: %v] -> Đang đổi User-Agent và thử lại lần %d/%d sau %.1fs...",
				lastErr, attempt+1, c.MaxRetries, sleepDuration.Seconds())

			time.Sleep(sleepDuration)
		}
	}

	return nil, fmt.Errorf("thất bại sau %d lần thử: %w", c.MaxRetries, lastErr)
}

// CrawlNovelInfo bóc tách thông tin tổng quan của truyện
func (c *Crawler) CrawlNovelInfo(novelURL string) (*Novel, string, error) {
	doc, err := c.fetchDoc(novelURL, "https://truyenfull.live/")
	if err != nil {
		return nil, "", err
	}

	// 1. Tên truyện
	title := strings.TrimSpace(doc.Find("h3.title[itemprop='name']").Text())
	if title == "" {
		title = strings.TrimSpace(doc.Find("h1.truyen-title").Text())
	}

	// 2. Tác giả
	author := strings.TrimSpace(doc.Find("a[itemprop='author']").Text())

	// 3. Thể loại
	var genres []string
	doc.Find("a[itemprop='genre']").Each(func(i int, s *goquery.Selection) {
		g := strings.TrimSpace(s.Text())
		if g != "" {
			genres = append(genres, g)
		}
	})

	// 4. Trạng thái
	statusText := strings.TrimSpace(doc.Find(".info span.text-success").Text())
	if statusText == "" {
		statusText = "Đang ra"
	}

	// 5. Mô tả truyện
	desc := strings.TrimSpace(doc.Find(".desc-text").Text())

	// 6. Ảnh bìa
	cover, _ := doc.Find(".books .book img").Attr("src")

	// 7. Slug
	slug := strings.Trim(strings.TrimPrefix(novelURL, "https://truyenfull.live/"), "/")
	parts := strings.Split(slug, "/")
	if len(parts) > 0 {
		slug = parts[0]
	}

	// 8. Link chương đầu tiên
	firstChapterURL, _ := doc.Find("#list-chapter .list-chapter li a").First().Attr("href")

	novel := &Novel{
		Title:       title,
		Slug:        slug,
		Author:      author,
		Genres:      genres,
		Description: desc,
		CoverURL:    cover,
		Status:      statusText,
		RatingScore: 9.6,
		Realm:       "",
		SourceURL:   novelURL,
		CreatedAt:   time.Now(),
	}

	return novel, firstChapterURL, nil
}

// CrawlChapter bóc tách nội dung của một chương
func (c *Crawler) CrawlChapter(chapterURL string, defaultChapterNum int, referer string) (*Chapter, string, error) {
	doc, err := c.fetchDoc(chapterURL, referer)
	if err != nil {
		return nil, "", err
	}

	// 1. Tiêu đề chương
	title := strings.TrimSpace(doc.Find("h2 a.chapter-title").Text())
	if title == "" {
		title = fmt.Sprintf("Chương %d", defaultChapterNum)
	}

	// 2. Số chương (từ input ẩn hoặc trích xuất từ URL)
	chapterNum := defaultChapterNum
	if numVal, exists := doc.Find("#chapter-num").Attr("value"); exists {
		if n, err := strconv.Atoi(numVal); err == nil && n > 0 {
			chapterNum = n
		}
	}

	// 3. Nội dung chữ sạch
	contentDiv := doc.Find("#chapter-c")
	cleanedText := CleanChapterHTML(contentDiv)

	// Đếm số từ
	wordCount := len(strings.Fields(cleanedText))

	// 4. Link chương tiếp theo (next_chap)
	nextURL, _ := doc.Find("#next_chap").Attr("href")
	if strings.Contains(nextURL, "javascript:void") || nextURL == chapterURL {
		nextURL = ""
	}

	chapter := &Chapter{
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

// SaveLocally lưu thông tin truyện và chương vào thư mục local
func SaveLocally(novel *Novel, chapters []*Chapter, baseDir string) error {
	novelDir := filepath.Join(baseDir, novel.Slug)
	chaptersDir := filepath.Join(novelDir, "chapters")

	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		return err
	}

	// 1. Lưu info truyện dạng JSON
	novelFile := filepath.Join(novelDir, "novel.json")
	novelBytes, err := json.MarshalIndent(novel, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(novelFile, novelBytes, 0644); err != nil {
		return err
	}

	// 2. Lưu từng chương truyện dạng text sạch và JSON
	for _, chap := range chapters {
		// File .txt để đọc offline hoặc upload R2
		txtFile := filepath.Join(chaptersDir, fmt.Sprintf("%d.txt", chap.ChapterNumber))
		if err := os.WriteFile(txtFile, []byte(chap.ContentText), 0644); err != nil {
			return err
		}

		// File .json chứa đầy đủ metadata
		jsonFile := filepath.Join(chaptersDir, fmt.Sprintf("%d.json", chap.ChapterNumber))
		chapBytes, _ := json.MarshalIndent(chap, "", "  ")
		if err := os.WriteFile(jsonFile, chapBytes, 0644); err != nil {
			return err
		}
	}

	return nil
}

// CrawlCatalog extracts list of novels from catalog/category pages
func (c *Crawler) CrawlCatalog(baseURL string, maxPages int) ([]*QueueNovel, error) {
	var results []*QueueNovel
	seen := make(map[string]bool)

	baseURL = strings.TrimRight(baseURL, "/")

	for page := 1; page <= maxPages; page++ {
		pageURL := baseURL
		if page > 1 {
			pageURL = fmt.Sprintf("%s/trang-%d/", baseURL, page)
		}

		doc, err := c.fetchDoc(pageURL, baseURL)
		if err != nil {
			fmt.Printf("⚠️ Lỗi tải trang %d: %v\n", page, err)
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

			results = append(results, &QueueNovel{
				Title:      title,
				Slug:       slug,
				SourceURL:  href + "/",
				AuthorName: author,
				CoverURL:   cover,
			})
		})

		time.Sleep(200 * time.Millisecond)
	}

	return results, nil
}

