package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"comics-backend/internal/api"
	"comics-backend/internal/crawler"
	"comics-backend/internal/crawler/fetcher"
	"comics-backend/internal/crawler/handlers"
	"comics-backend/internal/scheduler"
	"comics-backend/internal/storage"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	dbConn := os.Getenv("DATABASE_URL")
	if dbConn == "" {
		dbConn = "postgres://postgres:comics_password_2026@localhost:5432/comics_db?sslmode=disable"
	}

	storageDir := os.Getenv("STORAGE_DIR")
	if storageDir == "" {
		for _, c := range []string{
			filepath.Join("storage"),
			filepath.Join("..", "storage"),
			filepath.Join("..", "..", "storage"),
		} {
			if _, err := os.Stat(c); err == nil {
				storageDir = c
				break
			}
		}
		if storageDir == "" {
			storageDir = "storage"
		}
	}

	adminUIDir := os.Getenv("ADMIN_UI_DIR")
	if adminUIDir == "" {
		for _, c := range []string{
			filepath.Join("..", "frontend", "admin-ui"),
			filepath.Join("frontend", "admin-ui"),
			filepath.Join("..", "..", "frontend", "admin-ui"),
		} {
			if _, err := os.Stat(c); err == nil {
				adminUIDir = c
				break
			}
		}
	}

	clientUIDir := os.Getenv("CLIENT_UI_DIR")
	if clientUIDir == "" {
		for _, c := range []string{
			filepath.Join("..", "frontend", "client-ui"),
			filepath.Join("frontend", "client-ui"),
			filepath.Join("..", "..", "frontend", "client-ui"),
		} {
			if _, err := os.Stat(c); err == nil {
				clientUIDir = c
				break
			}
		}
	}

	// 1. Connect to PostgreSQL
	log.Println("🔌 Connecting to PostgreSQL...")
	db, err := storage.NewPostgresDB(dbConn)
	if err != nil {
		log.Printf("⚠️ Warning: PostgreSQL initial ping failed: %v. Will retry upon request.", err)
	} else {
		log.Println("✅ Connected to PostgreSQL successfully!")
		// Run auto migration
		if err := storage.AutoMigrate(db); err != nil {
			log.Printf("⚠️ Auto-migrate warning: %v", err)
		}
	}

	// 2. Repositories
	novelRepo := storage.NewNovelRepository(db)
	queueRepo := storage.NewQueueRepository(db)
	sourceRepo := storage.NewSourceRepository(db)
	fileStorage := storage.NewFileStorage(storageDir)
	affiliateRepo := storage.NewAffiliateRepository(db, storageDir)
	userRepo := storage.NewUserRepository(db, storageDir)

	// 3. Crawler Handlers Registry
	httpClient := fetcher.NewHTTPClient()
	registry := crawler.NewHandlerRegistry()

	// Register site handlers
	registry.Register(handlers.NewTruyenFullHandler(httpClient))
	registry.Register(handlers.NewTangThuVienHandler(httpClient))
	registry.Register(handlers.NewTruyenFullDichHandler(httpClient))
	registry.Register(handlers.NewGenericHandler(httpClient))

	log.Println("🧩 Registered Source Handlers: Truyện Full (truyenfull.live), Tàng Thư Viện (tangthuvien.net), Truyện Full Dịch (truyenfulldich.top), Generic Fallback")

	// 4. Crawler Engine (Continuous Queue Worker Daemon)
	engine := crawler.NewEngine(novelRepo, queueRepo, fileStorage, registry)
	engine.Start()

	// 5. Daily Scheduler
	dailyScanner := scheduler.NewDailyScanner(sourceRepo, queueRepo, novelRepo, registry, engine, 12*time.Hour)
	dailyScanner.Start()

	// 6. HTTP API Server
	srv := api.NewServer(novelRepo, queueRepo, sourceRepo, affiliateRepo, userRepo, engine, dailyScanner, registry, adminUIDir, clientUIDir)

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: srv.Handler(),
	}

	fmt.Printf("\n=======================================================\n")
	fmt.Printf("🚀 COMICS AUTOMATED BACKEND & CRAWLER ENGINE IS RUNNING\n")
	fmt.Printf("📡 API Base URL:      http://localhost:%s/api\n", port)
	fmt.Printf("🌐 Client UI URL:     http://localhost:%s/\n", port)
	fmt.Printf("🖥️ Admin UI URL:      http://localhost:%s/admin/\n", port)
	fmt.Printf("⚙️ Worker Status:     Continuous Background Daemon [ACTIVE]\n")
	fmt.Printf("⏰ Daily Scheduler:   Every 12h Auto-Scan [ACTIVE]\n")
	fmt.Printf("=======================================================\n\n")

	// Graceful shutdown handling
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("\n🛑 Shutting down server gracefully...")
	engine.Stop()
	dailyScanner.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	if db != nil {
		_ = db.Close()
	}
	log.Println("👋 Server exited.")
}
