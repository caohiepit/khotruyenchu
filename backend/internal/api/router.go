package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"comics-backend/internal/crawler"
	"comics-backend/internal/scheduler"
	"comics-backend/internal/storage"
)

type Server struct {
	mux           *http.ServeMux
	novelRepo     *storage.NovelRepository
	queueRepo     *storage.QueueRepository
	sourceRepo    *storage.SourceRepository
	affiliateRepo *storage.AffiliateRepository
	userRepo      *storage.UserRepository
	feedbackRepo  *storage.FeedbackRepository
	favoriteRepo  *storage.FavoriteRepository
	engine        *crawler.Engine
	scheduler     *scheduler.DailyScanner
	registry      *crawler.HandlerRegistry
	adminUIDir    string
	clientUIDir   string
}

func NewServer(
	novelRepo *storage.NovelRepository,
	queueRepo *storage.QueueRepository,
	sourceRepo *storage.SourceRepository,
	affiliateRepo *storage.AffiliateRepository,
	userRepo *storage.UserRepository,
	feedbackRepo *storage.FeedbackRepository,
	favoriteRepo *storage.FavoriteRepository,
	engine *crawler.Engine,
	scheduler *scheduler.DailyScanner,
	registry *crawler.HandlerRegistry,
	adminUIDir string,
	clientUIDir ...string,
) *Server {
	cDir := ""
	if len(clientUIDir) > 0 {
		cDir = clientUIDir[0]
	}
	s := &Server{
		mux:           http.NewServeMux(),
		novelRepo:     novelRepo,
		queueRepo:     queueRepo,
		sourceRepo:    sourceRepo,
		affiliateRepo: affiliateRepo,
		userRepo:      userRepo,
		feedbackRepo:  feedbackRepo,
		favoriteRepo:  favoriteRepo,
		engine:        engine,
		scheduler:     scheduler,
		registry:      registry,
		adminUIDir:    adminUIDir,
		clientUIDir:   cDir,
	}

	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return enableCORS(s.mux)
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) registerRoutes() {
	// Health & System Stats
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/admin/stats", s.handleStats)

	// Novels & Reading API (Admin + Public Client aliases)
	s.mux.HandleFunc("GET /api/admin/novels", s.handleGetNovels)
	s.mux.HandleFunc("GET /api/admin/novels/{id}", s.handleGetNovelDetail)
	s.mux.HandleFunc("GET /api/admin/chapters/{id}", s.handleGetChapterContent)
	s.mux.HandleFunc("GET /api/admin/genres", s.handleGetGenres)

	s.mux.HandleFunc("GET /api/novels", s.handleGetNovels)
	s.mux.HandleFunc("GET /api/novels/{id}", s.handleGetNovelDetail)
	s.mux.HandleFunc("GET /api/chapters/{id}", s.handleGetChapterContent)
	s.mux.HandleFunc("GET /api/genres", s.handleGetGenres)
	s.mux.HandleFunc("GET /api/proxy-cover", s.handleProxyCover)

	// Automated Ingestion & Queue API
	// 🚀 Unified Endpoint: Paste ANY URL -> Auto Resolve -> Auto Enqueue -> Auto Worker Crawl!
	s.mux.HandleFunc("POST /api/admin/crawler/submit", s.handleSubmitURL)
	s.mux.HandleFunc("GET /api/admin/crawler/queue", s.handleGetQueue)
	s.mux.HandleFunc("POST /api/admin/crawler/queue/reset-failed", s.handleResetFailedQueue)
	s.mux.HandleFunc("POST /api/admin/crawler/queue/toggle", s.handleToggleWorker)

	// Sources Management API
	s.mux.HandleFunc("GET /api/admin/sources", s.handleGetSources)
	s.mux.HandleFunc("POST /api/admin/sources/toggle", s.handleToggleSource)
	s.mux.HandleFunc("POST /api/admin/sources/scan", s.handleScanSource)

	// Shopee Affiliate Management API
	s.mux.HandleFunc("GET /api/admin/affiliates", s.handleGetAdminAffiliates)
	s.mux.HandleFunc("POST /api/admin/affiliates", s.handleCreateAffiliate)
	s.mux.HandleFunc("PUT /api/admin/affiliates/{id}", s.handleUpdateAffiliate)
	s.mux.HandleFunc("DELETE /api/admin/affiliates/{id}", s.handleDeleteAffiliate)
	s.mux.HandleFunc("GET /api/affiliates", s.handleGetPublicAffiliates)
	s.mux.HandleFunc("POST /api/affiliates/{id}/click", s.handleClickAffiliate)

	// Reader Auth & Cultivation Realm API
	s.mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.HandleFunc("GET /api/auth/me", s.handleGetMe)
	s.mux.HandleFunc("POST /api/user/read-chapter", s.handleUserReadChapter)

	// Admin Auth API
	s.mux.HandleFunc("POST /api/admin/login", s.handleAdminLogin)
	s.mux.HandleFunc("GET /api/admin/auth/verify", s.handleAdminVerify)

	// Reader Favorites API (Tủ Sách Yêu Thích)
	s.mux.HandleFunc("POST /api/user/favorites/toggle", s.handleToggleFavorite)
	s.mux.HandleFunc("GET /api/user/favorites", s.handleGetFavorites)
	s.mux.HandleFunc("GET /api/user/favorites/check", s.handleCheckFavorite)

	// Feedback API (Phi Thư Truyền Tin)
	s.mux.HandleFunc("POST /api/feedback", s.handleCreateFeedback)
	s.mux.HandleFunc("GET /api/admin/feedback", s.handleGetFeedback)
	s.mux.HandleFunc("DELETE /api/admin/feedback/{id}", s.handleDeleteFeedback)

	// Static Admin UI Files
	if s.adminUIDir != "" {
		if absDir, err := filepath.Abs(s.adminUIDir); err == nil {
			fs := http.FileServer(http.Dir(absDir))
			s.mux.Handle("GET /admin/", http.StripPrefix("/admin/", fs))
		}
	}

	// Static Client UI Files (Root "/" and "/client/")
	if s.clientUIDir != "" {
		ExtractDesignAssets(s.clientUIDir)
		if absClientDir, err := filepath.Abs(s.clientUIDir); err == nil {
			clientFS := http.FileServer(http.Dir(absClientDir))
			s.mux.Handle("GET /client/", http.StripPrefix("/client/", clientFS))
			s.mux.Handle("GET /", clientFS)
		}
	} else if s.adminUIDir != "" {
		s.mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/admin/", http.StatusFound)
		})
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":         "ok",
		"worker_running": s.engine.IsRunning(),
	})
}
