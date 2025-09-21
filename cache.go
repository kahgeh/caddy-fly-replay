package flyreplay

import (
	"strings"
	"time"
)

// Get retrieves a cache entry for the given full path
func (c *PathCache) Get(fullPath string) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Check exact match first
	if entry, ok := c.store[fullPath]; ok {
		if time.Now().Before(entry.ExpiresAt) {
			return entry, true
		}
		// Expired, will be cleaned up later
	}
	
	// Check pattern matches
	for pattern, entry := range c.store {
		if matchesPattern(fullPath, pattern) && time.Now().Before(entry.ExpiresAt) {
			return entry, true
		}
	}
	
	return nil, false
}

// Set stores a new cache entry
func (c *PathCache) Set(path, pattern, target string, ttl int, allowBypass bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[pattern] = &CacheEntry{
		Path:        path,
		Target:      target,
		Pattern:     pattern,
		AllowBypass: allowBypass,
		ExpiresAt:   time.Now().Add(time.Duration(ttl) * time.Second),
	}
}

// Invalidate removes a cache entry by pattern
func (c *PathCache) Invalidate(pattern string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.store, pattern)
}

// Clean removes expired entries (can be called periodically)
func (c *PathCache) Clean() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	for pattern, entry := range c.store {
		if now.After(entry.ExpiresAt) {
			delete(c.store, pattern)
		}
	}
}

// matchesPattern checks if a path matches a pattern with wildcards
func matchesPattern(path, pattern string) bool {
	// Handle exact match
	if path == pattern {
		return true
	}
	
	// Handle wildcard patterns
	if strings.Contains(pattern, "*") {
		// Convert pattern to a simple prefix match for patterns like /en-US/user123/*
		prefix := strings.TrimSuffix(pattern, "*")
		if strings.HasPrefix(path, prefix) {
			return true
		}
		
		// Handle more complex patterns if needed
		// This is a simplified implementation
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			// Pattern like /prefix/*/suffix
			if strings.HasPrefix(path, parts[0]) && strings.HasSuffix(path, parts[1]) {
				return true
			}
		}
	}
	
	return false
}