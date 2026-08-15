package cloudid

import (
	"sync"
	"time"
)

// defaultCacheTTL is how long a fetched identity document is considered fresh.
const defaultCacheTTL = 10 * time.Minute

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

// identityCache is a small concurrency-safe cache keyed by cloud type.
// It avoids hammering the metadata endpoint on every call.
type identityCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]cacheEntry
	now     func() time.Time // injectable clock for testing
}

func newIdentityCache(ttl time.Duration) *identityCache {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	return &identityCache{
		ttl:     ttl,
		entries: make(map[string]cacheEntry),
		now:     time.Now,
	}
}

// get returns cached data for cloud and reports whether it is present and still fresh.
func (c *identityCache) get(cloud string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[cloud]
	if !ok {
		return nil, false
	}
	if c.now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

// set stores data for cloud with the configured TTL.
func (c *identityCache) set(cloud string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[cloud] = cacheEntry{
		data:      data,
		expiresAt: c.now().Add(c.ttl),
	}
}

// invalidate removes a cached entry for cloud.
func (c *identityCache) invalidate(cloud string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, cloud)
}

// clear removes all cached entries.
func (c *identityCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]cacheEntry)
}

// defaultCache is the package-level cache shared by the exported helpers.
var defaultCache = newIdentityCache(defaultCacheTTL)
