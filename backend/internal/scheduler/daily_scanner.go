package scheduler

import (
	"log"
	"sync"
	"time"

	"comics-backend/internal/crawler"
	"comics-backend/internal/storage"
)

// DailyScanner periodically inspects registered sources for new novels and ongoing updates
type DailyScanner struct {
	mu         sync.Mutex
	sourceRepo *storage.SourceRepository
	queueRepo  *storage.QueueRepository
	novelRepo  *storage.NovelRepository
	registry   *crawler.HandlerRegistry
	engine     *crawler.Engine
	ticker     *time.Ticker
	stopChan   chan struct{}
	isRunning  bool
	interval   time.Duration
}

func NewDailyScanner(
	sourceRepo *storage.SourceRepository,
	queueRepo *storage.QueueRepository,
	novelRepo *storage.NovelRepository,
	registry *crawler.HandlerRegistry,
	engine *crawler.Engine,
	interval time.Duration,
) *DailyScanner {
	if interval <= 0 {
		interval = 12 * time.Hour // Default run every 12 hours
	}

	return &DailyScanner{
		sourceRepo: sourceRepo,
		queueRepo:  queueRepo,
		novelRepo:  novelRepo,
		registry:   registry,
		engine:     engine,
		stopChan:   make(chan struct{}),
		interval:   interval,
	}
}

// Start begins periodic background scanning
func (s *DailyScanner) Start() {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = true
	s.ticker = time.NewTicker(s.interval)
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	log.Printf("⏰ [Daily Scanner] Background scheduler started (Interval: %v)", s.interval)
	go s.loop()

	// ⚡ Automatic source scan on code startup so worker starts crawling immediately without manual trigger!
	go func() {
		time.Sleep(2 * time.Second)
		log.Println("⚡ [Startup Auto-Discovery] Initiating automatic source scan for all active web sources...")
		s.ScanAllActiveSources()
	}()
}

// Stop terminates the scheduler
func (s *DailyScanner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isRunning {
		return
	}
	s.ticker.Stop()
	close(s.stopChan)
	s.isRunning = false
	log.Println("⏰ [Daily Scanner] Scheduler stopped.")
}

func (s *DailyScanner) loop() {
	for {
		select {
		case <-s.ticker.C:
			s.ScanAllActiveSources()
		case <-s.stopChan:
			return
		}
	}
}

// ScanAllActiveSources iterates over active sources and enqueues new novels
func (s *DailyScanner) ScanAllActiveSources() {
	log.Println("🔍 [Daily Scanner] Starting scheduled scan of all active web sources...")

	sources, err := s.sourceRepo.GetActiveSources()
	if err != nil {
		log.Printf("❌ [Daily Scanner] Error fetching active sources: %v", err)
		return
	}

	totalDiscovered := 0
	for _, src := range sources {
		if src.CatalogURL == "" {
			continue
		}

		handler, err := s.registry.GetHandlerByDomain(src.Domain)
		if err != nil || handler == nil {
			log.Printf("⚠️ [Daily Scanner] No handler for source: %s (%s)", src.Name, src.Domain)
			continue
		}

		log.Printf("   🔎 Scanning catalog for 【%s】: %s ...", src.Name, src.CatalogURL)
		novels, err := handler.DiscoverNovels(src.CatalogURL, 2)
		if err != nil {
			log.Printf("   ❌ Error scanning %s: %v", src.Name, err)
			continue
		}

		if len(novels) > 0 {
			inserted, err := s.queueRepo.EnqueueBatch(novels)
			if err == nil {
				totalDiscovered += inserted
				log.Printf("   ✅ Found %d novels (%d new inserted into queue)", len(novels), inserted)
			}
		}

		_ = s.sourceRepo.UpdateLastScanned(src.ID)
		time.Sleep(1 * time.Second)
	}

	log.Printf("🎉 [Daily Scanner] Scan completed! %d new novels added to queue.", totalDiscovered)

	// Wake up crawler engine if new items were found
	if totalDiscovered > 0 {
		s.engine.WakeUp()
	}
}

// ScanSourceNow triggers an immediate scan for a specific source
func (s *DailyScanner) ScanSourceNow(sourceID string) (int, error) {
	sources, err := s.sourceRepo.GetAllSources()
	if err != nil {
		return 0, err
	}

	for _, src := range sources {
		if src.ID == sourceID {
			handler, err := s.registry.GetHandlerByDomain(src.Domain)
			if err != nil || handler == nil {
				return 0, err
			}

			novels, err := handler.DiscoverNovels(src.CatalogURL, 2)
			if err != nil {
				return 0, err
			}

			inserted, err := s.queueRepo.EnqueueBatch(novels)
			if err != nil {
				return 0, err
			}

			_ = s.sourceRepo.UpdateLastScanned(src.ID)
			s.engine.WakeUp()
			return inserted, nil
		}
	}

	return 0, nil
}
