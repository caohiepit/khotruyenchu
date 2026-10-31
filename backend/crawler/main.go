package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	novelURL := flag.String("url", "https://truyenfull.live/ma-than-troi-sinh/", "URL của bộ truyện hoặc trang danh mục")
	chapterLimit := flag.Int("limit", 3, "Số lượng chương cần cào thử nghiệm (0 là cào toàn bộ)")
	delayMs := flag.Int("delay", 500, "Thời gian nghỉ giữa mỗi chương (mili-giây)")
	outputDir := flag.String("out", "../storage", "Thư mục lưu dữ liệu truyện local")
	dbConn := flag.String("db", "", "PostgreSQL connection string (e.g. postgres://postgres:comics_password_2026@localhost:5432/comics_db?sslmode=disable)")
	isCatalog := flag.Bool("catalog", false, "Chế độ quét danh sách tên truyện từ danh mục và nạp vào crawl_queue")
	maxPages := flag.Int("pages", 1, "Số trang danh mục cần quét")
	flag.Parse()

	absOutDir, _ := filepath.Abs(*outputDir)

	crawler := NewCrawler()

	// Chế độ 1: Quét danh mục truyện và nạp vào hàng đợi crawl_queue
	if *isCatalog {
		fmt.Println("====================================================================")
		fmt.Println("📋 CHẾ ĐỘ QUÉT DANH MỤC TRUYỆN (CATALOG DISCOVERY)")
		fmt.Printf("🎯 Danh mục: %s\n", *novelURL)
		fmt.Printf("📄 Số trang quét: %d\n", *maxPages)
		fmt.Println("====================================================================")

		novels, err := crawler.CrawlCatalog(*novelURL, *maxPages)
		if err != nil {
			fmt.Printf("❌ Lỗi quét danh mục: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Đã tìm thấy %d bộ truyện từ trang danh mục!\n", len(novels))
		if *dbConn != "" {
			inserted, err := SaveToQueue(*dbConn, novels)
			if err != nil {
				fmt.Printf("❌ Lỗi lưu vào crawl_queue: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("🎉 Đã thêm thành công %d bộ truyện mới vào crawl_queue!\n", inserted)
		} else {
			fmt.Println("⚠️ Chưa truyền -db nên không lưu vào database.")
		}
		return
	}

	fmt.Println("====================================================================")
	fmt.Println("🚀 BẮT ĐẦU TOOL CRAWLER TRUYỆN CHỮ - BẢN HIGH PERFORMANCE (GOLANG)")
	fmt.Printf("🎯 Nguồn: %s\n", *novelURL)
	fmt.Printf("📊 Giới hạn cào: %d chương | Độ trễ: %dms\n", *chapterLimit, *delayMs)
	fmt.Printf("📂 Thư mục local: %s\n", absOutDir)
	if *dbConn != "" {
		fmt.Println("🐘 Chế độ: Đồng bộ trực tiếp vào PostgreSQL Docker")
	} else {
		fmt.Println("💾 Chế độ: Lưu file Local (.txt, .json)")
	}
	fmt.Println("====================================================================")

	// 1. Cào thông tin tổng quan truyện
	fmt.Println("\n⏳ Đang cào thông tin tổng quan bộ truyện...")
	novel, firstChapterURL, err := crawler.CrawlNovelInfo(*novelURL)
	if err != nil {
		fmt.Printf("❌ Lỗi cào thông tin truyện: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Đã lấy thông tin: 【%s】\n", novel.Title)
	fmt.Printf("   - Tác giả: %s\n", novel.Author)
	fmt.Printf("   - Thể loại: %v\n", novel.Genres)
	fmt.Printf("   - Trạng thái: %s\n", novel.Status)
	fmt.Printf("   - Cảnh giới: %s\n", novel.Realm)
	if firstChapterURL != "" {
		fmt.Printf("   - Chương đầu: %s\n", firstChapterURL)
	}

	// 2. Bắt đầu cào từng chương
	currentURL := firstChapterURL
	if currentURL == "" {
		currentURL = fmt.Sprintf("%schuong-1/", *novelURL)
	}

	var chapters []*Chapter
	count := 0
	currentChapterNum := 1
	previousURL := *novelURL

	fmt.Println("\n📚 Bắt đầu tải nội dung các chương...")

	for currentURL != "" {
		if *chapterLimit > 0 && count >= *chapterLimit {
			fmt.Printf("\n🏁 Đã đạt giới hạn thử nghiệm %d chương theo cài đặt.\n", *chapterLimit)
			break
		}

		count++
		fmt.Printf("   👉 [%d] Đang tải: %s ... ", count, currentURL)

		chapter, nextURL, err := crawler.CrawlChapter(currentURL, currentChapterNum, previousURL)
		if err != nil {
			fmt.Printf("❌ Thất bại sau các lần retry: %v\n", err)
			break
		}

		chapters = append(chapters, chapter)
		fmt.Printf("✅ Xong: %s (%d từ)\n", chapter.Title, chapter.WordCount)

		previousURL = currentURL
		currentURL = nextURL
		currentChapterNum = chapter.ChapterNumber + 1

		// Nghỉ ngơi kèm random jitter (100ms - 400ms) để giả lập hành vi người thật
		if currentURL != "" {
			jitter := int(time.Now().UnixNano()%300) + 100
			sleepTime := time.Duration(*delayMs+jitter) * time.Millisecond
			time.Sleep(sleepTime)
		}
	}

	novel.TotalChapters = len(chapters)

	// 3. Lưu dữ liệu vào file local
	fmt.Printf("\n💾 Đang lưu %d chương vào thư mục local...\n", len(chapters))
	if err := SaveLocally(novel, chapters, absOutDir); err != nil {
		fmt.Printf("❌ Lỗi lưu dữ liệu local: %v\n", err)
		os.Exit(1)
	}

	// 4. Nếu có kết nối DB thì đẩy thẳng vào PostgreSQL
	if *dbConn != "" {
		fmt.Println("🐘 Đang đẩy dữ liệu vào PostgreSQL Docker (Tách bảng + nén LZ4)...")
		if err := SaveToPostgres(*dbConn, novel, chapters); err != nil {
			fmt.Printf("❌ Lỗi đẩy vào PostgreSQL: %v\n", err)
		} else {
			fmt.Println("✅ Đã lưu thành công vào PostgreSQL Docker!")
		}
	}

	fmt.Println("====================================================================")
	fmt.Printf("🎉 HOÀN TẤT THÀNH CÔNG!\n")
	fmt.Printf("📁 Dữ liệu local: %s\\%s\n", absOutDir, novel.Slug)
	fmt.Println("====================================================================")
}
