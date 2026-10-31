package fetcher

import (
	"fmt"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var realisticUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:126.0) Gecko/20100101 Firefox/126.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
}

func getRandomUserAgent() string {
	idx := time.Now().UnixNano() % int64(len(realisticUserAgents))
	return realisticUserAgents[idx]
}

// HTTPClient wraps standard http.Client with exponential backoff and rotating User-Agents
type HTTPClient struct {
	client      *http.Client
	MaxRetries  int
	BaseBackoff time.Duration
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		MaxRetries:  3,
		BaseBackoff: 2 * time.Second,
	}
}

// FetchDoc loads an HTML URL into a goquery.Document with automatic retries and realistic headers
func (c *HTTPClient) FetchDoc(targetURL string, referer string) (*goquery.Document, error) {
	var lastErr error

	for attempt := 1; attempt <= c.MaxRetries; attempt++ {
		req, err := http.NewRequest("GET", targetURL, nil)
		if err != nil {
			return nil, err
		}

		ua := getRandomUserAgent()
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en-US;q=0.8,en;q=0.7")
		req.Header.Set("Sec-Ch-Ua", `"Chromium";v="126", "Google Chrome";v="126", "Not-A.Brand";v="99"`)
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
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("network error (%v)", err)
		} else {
			if resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				return goquery.NewDocumentFromReader(resp.Body)
			}

			resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				return nil, fmt.Errorf("page or chapter not found (HTTP 404)")
			}

			lastErr = fmt.Errorf("server responded with HTTP %d", resp.StatusCode)
		}

		if attempt < c.MaxRetries {
			multiplier := 1 << (attempt - 1)
			jitterMs := int(time.Now().UnixNano()%500) + 150
			sleepDuration := time.Duration(multiplier)*c.BaseBackoff + time.Duration(jitterMs)*time.Millisecond
			time.Sleep(sleepDuration)
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", c.MaxRetries, lastErr)
}

// FetchRaw loads an HTML URL into a raw string
func (c *HTTPClient) FetchRaw(targetURL string, referer string) (string, error) {
	doc, err := c.FetchDoc(targetURL, referer)
	if err != nil {
		return "", err
	}
	html, err := doc.Html()
	if err != nil {
		return "", err
	}
	return html, nil
}
