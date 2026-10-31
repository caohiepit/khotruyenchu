package handlers

import "comics-backend/internal/domain"

// SourceHandler defines the contract that each website crawler must implement
type SourceHandler interface {
	// Name returns a human-readable name of the source (e.g. "Truyện Full")
	Name() string

	// Domain returns the canonical domain string (e.g. "truyenfull.live")
	Domain() string

	// CanHandle checks if this handler is responsible for the given URL
	CanHandle(rawURL string) bool

	// DiscoverNovels parses catalog/category pages and extracts novel links for queue
	DiscoverNovels(catalogURL string, maxPages int) ([]domain.DiscoveredNovel, error)

	// ParseNovel extracts overview info of a story and returns novel metadata plus first chapter URL
	ParseNovel(novelURL string) (*domain.Novel, string, error)

	// ParseChapter extracts chapter text content, chapter title, word count, and next chapter URL
	ParseChapter(chapterURL string, defaultChapterNum int, referer string) (*domain.Chapter, string, error)
}
