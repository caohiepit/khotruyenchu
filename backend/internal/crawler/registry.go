package crawler

import (
	"fmt"
	"strings"
	"sync"

	"comics-backend/internal/crawler/handlers"
)

// HandlerRegistry manages website-specific parser implementations
type HandlerRegistry struct {
	mu             sync.RWMutex
	handlers       []handlers.SourceHandler
	genericHandler handlers.SourceHandler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make([]handlers.SourceHandler, 0),
	}
}

// Register adds a new SourceHandler to the registry
func (r *HandlerRegistry) Register(h handlers.SourceHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if h.Domain() == "generic" {
		r.genericHandler = h
		return
	}
	r.handlers = append(r.handlers, h)
}

// Resolve finds the best SourceHandler for a given target URL
func (r *HandlerRegistry) Resolve(rawURL string) handlers.SourceHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, h := range r.handlers {
		if h.CanHandle(rawURL) {
			return h
		}
	}

	if r.genericHandler != nil {
		return r.genericHandler
	}

	return nil
}

// GetHandlerByDomain looks up handler by domain name
func (r *HandlerRegistry) GetHandlerByDomain(domain string) (handlers.SourceHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	domain = strings.ToLower(domain)
	for _, h := range r.handlers {
		if strings.EqualFold(h.Domain(), domain) || strings.Contains(domain, h.Domain()) {
			return h, nil
		}
	}

	if r.genericHandler != nil {
		return r.genericHandler, nil
	}

	return nil, fmt.Errorf("no handler found for domain: %s", domain)
}

// GetAllHandlers returns list of all registered handlers
func (r *HandlerRegistry) GetAllHandlers() []handlers.SourceHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]handlers.SourceHandler, len(r.handlers))
	copy(res, r.handlers)
	return res
}
