package cloudid

import (
	"testing"
	"time"
)

func TestIdentityCache_SetGet(t *testing.T) {
	c := newIdentityCache(time.Minute)

	if _, ok := c.get(ALIYUN_CLOUD_TYPE); ok {
		t.Fatal("expected empty cache to miss")
	}

	c.set(ALIYUN_CLOUD_TYPE, []byte("payload"))
	data, ok := c.get(ALIYUN_CLOUD_TYPE)
	if !ok {
		t.Fatal("expected cache hit after set")
	}
	if string(data) != "payload" {
		t.Fatalf("unexpected cached data: %q", data)
	}
}

func TestIdentityCache_Expiry(t *testing.T) {
	c := newIdentityCache(time.Minute)

	now := time.Unix(0, 0)
	c.now = func() time.Time { return now }

	c.set(TENCENT_CLOUD_TYPE, []byte("x"))
	if _, ok := c.get(TENCENT_CLOUD_TYPE); !ok {
		t.Fatal("expected fresh entry to be returned")
	}

	// Advance beyond TTL.
	now = now.Add(2 * time.Minute)
	if _, ok := c.get(TENCENT_CLOUD_TYPE); ok {
		t.Fatal("expected expired entry to miss")
	}
}

func TestIdentityCache_InvalidateAndClear(t *testing.T) {
	c := newIdentityCache(time.Minute)
	c.set(ALIYUN_CLOUD_TYPE, []byte("a"))
	c.set(TENCENT_CLOUD_TYPE, []byte("b"))

	c.invalidate(ALIYUN_CLOUD_TYPE)
	if _, ok := c.get(ALIYUN_CLOUD_TYPE); ok {
		t.Fatal("expected invalidated entry to miss")
	}
	if _, ok := c.get(TENCENT_CLOUD_TYPE); !ok {
		t.Fatal("expected other entry to remain")
	}

	c.clear()
	if _, ok := c.get(TENCENT_CLOUD_TYPE); ok {
		t.Fatal("expected cleared cache to miss")
	}
}

func TestNewIdentityCache_DefaultTTL(t *testing.T) {
	c := newIdentityCache(0)
	if c.ttl != defaultCacheTTL {
		t.Fatalf("expected default TTL %v, got %v", defaultCacheTTL, c.ttl)
	}
}

func TestSetCacheTTLAndClearCache(t *testing.T) {
	t.Cleanup(func() {
		SetCacheTTL(defaultCacheTTL)
		ClearCache()
	})

	SetCacheTTL(30 * time.Second)
	defaultCache.mu.RLock()
	got := defaultCache.ttl
	defaultCache.mu.RUnlock()
	if got != 30*time.Second {
		t.Fatalf("expected TTL 30s, got %v", got)
	}

	// Non-positive resets to default.
	SetCacheTTL(0)
	defaultCache.mu.RLock()
	got = defaultCache.ttl
	defaultCache.mu.RUnlock()
	if got != defaultCacheTTL {
		t.Fatalf("expected default TTL, got %v", got)
	}

	defaultCache.set(ALIYUN_CLOUD_TYPE, []byte("x"))
	ClearCache()
	if _, ok := defaultCache.get(ALIYUN_CLOUD_TYPE); ok {
		t.Fatal("expected ClearCache to empty the cache")
	}
}
