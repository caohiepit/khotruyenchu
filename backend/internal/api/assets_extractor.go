package api

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"comics-backend/internal/cleaner"
)

// ExtractDesignAssets copies the design reference sheets into client-ui/assets/ for CSS sprite & artwork rendering
func ExtractDesignAssets(clientUIDir string) {
	if clientUIDir == "" {
		return
	}
	assetsDir := filepath.Join(clientUIDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return
	}

	var designsDir string
	for _, candidate := range []string{
		filepath.Join(clientUIDir, "..", "..", "designs"),
		filepath.Join("..", "designs"),
		filepath.Join("designs"),
	} {
		if abs, err := filepath.Abs(candidate); err == nil {
			if _, err := os.Stat(abs); err == nil {
				designsDir = abs
				break
			}
		}
	}
	if designsDir == "" {
		return
	}

	files := []string{
		"01_trang_chu_homepage.jpg",
		"02_chi_tiet_truyen_detail.jpg",
		"03_trinh_doc_truyen_reader.jpg",
	}
	for _, name := range files {
		src := filepath.Join(designsDir, name)
		dst := filepath.Join(assetsDir, name)
		if data, err := os.ReadFile(src); err == nil {
			_ = os.WriteFile(dst, data, 0644)
		}
	}
	log.Println("🎨 Synced design artwork sheets into client-ui/assets/ successfully.")
}

// handleProxyCover proxies book cover images from truyenfull / tangthuvien to bypass Referer hotlink blocks
func (s *Server) handleProxyCover(w http.ResponseWriter, r *http.Request) {
	rawURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if rawURL == "" || (!strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://")) {
		http.Error(w, "Invalid cover URL", http.StatusBadRequest)
		return
	}

	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	if strings.Contains(rawURL, "truyenfull") {
		req.Header.Set("Referer", "https://truyenfull.live/")
	} else if strings.Contains(rawURL, "tangthuvien") {
		req.Header.Set("Referer", "https://tangthuvien.net/")
	}

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			_ = resp.Body.Close()
		}
		http.Error(w, "Upstream cover unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 🧼 Clean Truyenfull watermark ("truyenfull.vn", bottom banner, etc.) on the fly!
	cleanBytes, err := cleaner.CleanCoverWatermark(resp.Body)
	if err == nil && len(cleanBytes) > 0 {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Header().Set("Content-Length", strconv.Itoa(len(cleanBytes)))
		_, _ = w.Write(cleanBytes)
		return
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = io.Copy(w, resp.Body)
}
