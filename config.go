package flyreplay

import (
	"sync"
	"time"
)

// FlyReplay is the main configuration structure for the plugin
type FlyReplay struct {
	Apps        map[string]AppConfig `json:"apps,omitempty"`
	CacheDir    string               `json:"cache_dir,omitempty"`
	CacheTTL    int                  `json:"cache_ttl,omitempty"`  // default TTL in seconds
	EnableCache bool                 `json:"enable_cache,omitempty"`
	Debug       bool                 `json:"debug,omitempty"`
	
	cache *PathCache
}

// AppConfig holds the configuration for each app
type AppConfig struct {
	Domain string `json:"domain"`  // where to forward (e.g., localhost:9001)
}

// PathCache manages the path-based caching
type PathCache struct {
	mu    sync.RWMutex
	store map[string]*CacheEntry  // full path -> cache entry
}

// CacheEntry represents a cached routing decision
type CacheEntry struct {
	Path      string    // full path including domain
	Target    string    // app name from fly-replay header
	Pattern   string    // pattern from fly-replay-cache header
	ExpiresAt time.Time
}

// NewPathCache creates a new PathCache instance
func NewPathCache() *PathCache {
	return &PathCache{
		store: make(map[string]*CacheEntry),
	}
}