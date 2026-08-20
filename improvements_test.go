package cloudid

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// errorServer returns a server that always responds with the given status.
func errorServer(t *testing.T, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
}

// TestGettersPropagateErrors exercises the error branch of every single-field
// getter: when the upstream identity document is unavailable, each getter must
// return an error rather than an empty value.
func TestGettersPropagateErrors(t *testing.T) {
	server := errorServer(t, http.StatusInternalServerError)
	defer server.Close()
	withTestClient(t, server)

	getters := []struct {
		name string
		fn   func() (string, error)
	}{
		{"aliyun-zone", GetAliyunZoneID},
		{"aliyun-region", GetAliyunRegionID},
		{"aliyun-instance", GetAliyunInstanceID},
		{"aliyun-ipv4", GetAliyunPrivateIpv4},
		{"aliyun-mac", GetAliyunMac},
		{"aliyun-serial", GetAliyunSerialNumber},
		{"tencent-instance", GetTencentInstanceID},
		{"tencent-region", GetTencentRegion},
		{"tencent-zone", GetTencentZone},
		{"tencent-ipv4", GetTencentPrivateIpv4},
		{"tencent-mac", GetTencentMac},
		{"tencent-uuid", GetTencentUUID},
		{"huawei-instance", GetHuaweiInstanceID},
		{"huawei-region", GetHuaweiRegion},
		{"huawei-zone", GetHuaweiZone},
		{"huawei-ipv4", GetHuaweiPrivateIpv4},
		{"huawei-project", GetHuaweiProjectID},
		{"aws-instance", GetAWSInstanceID},
		{"aws-region", GetAWSRegion},
		{"aws-zone", GetAWSZone},
		{"aws-ipv4", GetAWSPrivateIpv4},
		{"aws-mac", GetAWSMac},
		{"aws-account", GetAWSAccountID},
	}
	for _, g := range getters {
		t.Run(g.name, func(t *testing.T) {
			ClearCache()
			got, err := g.fn()
			if err == nil {
				t.Fatalf("expected error, got value %q", got)
			}
			if got != "" {
				t.Fatalf("expected empty value on error, got %q", got)
			}
		})
	}
}

// TestGettersReturnIdentityErrors covers the *Identity getters' error paths.
func TestGettersReturnIdentityErrors(t *testing.T) {
	server := errorServer(t, http.StatusInternalServerError)
	defer server.Close()
	withTestClient(t, server)

	if _, err := GetAliyunIdentity(); err == nil {
		t.Error("expected aliyun identity error")
	}
	ClearCache()
	if _, err := GetTencentIdentity(); err == nil {
		t.Error("expected tencent identity error")
	}
	ClearCache()
	if _, err := GetHuaweiIdentity(); err == nil {
		t.Error("expected huawei identity error")
	}
	ClearCache()
	if _, err := GetAWSIdentity(); err == nil {
		t.Error("expected aws identity error")
	}
}

// TestErrMetadataUnavailable_5xx verifies that a 5xx response is classified as
// ErrMetadataUnavailable and propagated through GetIdentity.
func TestErrMetadataUnavailable_5xx(t *testing.T) {
	server := errorServer(t, http.StatusBadGateway)
	defer server.Close()
	withTestClient(t, server)

	_, err := GetIdentity(ALIYUN_CLOUD_TYPE)
	if err == nil {
		t.Fatal("expected error for 5xx metadata response")
	}
	if !errors.Is(err, ErrMetadataUnavailable) {
		t.Fatalf("expected ErrMetadataUnavailable, got %v", err)
	}
}

// TestErrMetadataUnavailable_Transport verifies transport-layer failures (here
// a closed server) are classified as ErrMetadataUnavailable.
func TestErrMetadataUnavailable_Transport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	withTestClient(t, server)
	server.Close() // force a connection error on the next request.

	_, err := GetIdentity(TENCENT_CLOUD_TYPE)
	if err == nil {
		t.Fatal("expected transport error")
	}
	if !errors.Is(err, ErrMetadataUnavailable) {
		t.Fatalf("expected ErrMetadataUnavailable, got %v", err)
	}
}

// TestNotThisCloud_4xx verifies a 4xx response is NOT classified as
// ErrMetadataUnavailable (it is a "not this cloud" signal).
func TestNotThisCloud_4xx(t *testing.T) {
	server := errorServer(t, http.StatusNotFound)
	defer server.Close()
	withTestClient(t, server)

	_, err := GetIdentity(AWS_CLOUD_TYPE)
	if err == nil {
		t.Fatal("expected error for missing document")
	}
	if errors.Is(err, ErrMetadataUnavailable) {
		t.Fatalf("did not expect ErrMetadataUnavailable for 4xx, got %v", err)
	}
}

// TestGetIdentityContext_Unknown covers the default branch.
func TestGetIdentityContext_Unknown(t *testing.T) {
	if _, err := GetIdentityContext(context.Background(), "unknown"); err != ErrNotDetected {
		t.Fatalf("expected ErrNotDetected, got %v", err)
	}
}

// TestGetIdentityContext_Cancelled verifies a cancelled context surfaces as an
// error through the metadata path.
func TestGetIdentityContext_Cancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(aliyunSampleDoc))
	}))
	defer server.Close()
	withTestClient(t, server)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := GetIdentityContext(ctx, ALIYUN_CLOUD_TYPE); err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// TestDetectContext_Cancelled verifies DetectContext returns ErrNotDetected when
// the parent context is already cancelled (all probes fail fast).
func TestDetectContext_Cancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(aliyunSampleDoc))
	}))
	defer server.Close()
	withTestClient(t, server)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := DetectContext(ctx); err != ErrNotDetected {
		t.Fatalf("expected ErrNotDetected for cancelled context, got %v", err)
	}
}

func TestHuaweiRegionFromZone(t *testing.T) {
	cases := []struct {
		name string
		zone string
		want string
	}{
		{"standard", "cn-north-4a", "cn-north-4"},
		{"east", "cn-east-3b", "cn-east-3"},
		{"dotted-suffix", "az1.dc1", ""},
		{"no-letter-suffix", "cn-north-4", ""},
		{"empty", "", ""},
		{"whitespace", "   ", ""},
		{"multi-letter", "region-1ab", "region-1"},
		{"only-letters-tail", "region-abc", ""},
		{"no-dash", "singleword", ""},
		{"trailing-dash", "cn-north-", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := huaweiRegionFromZone(tc.zone); got != tc.want {
				t.Fatalf("huaweiRegionFromZone(%q) = %q, want %q", tc.zone, got, tc.want)
			}
		})
	}
}

// TestPutBackground exercises the background put wrapper directly.
func TestPutBackground(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		_, _ = w.Write([]byte("token-value"))
	}))
	defer server.Close()
	withTestClient(t, server)

	body, err := put(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "token-value" {
		t.Fatalf("unexpected body: %q", body)
	}
}

// TestTencentIncompleteNotCached verifies a partial best-effort result is
// returned to the caller but not frozen in the cache, so a later retry sees the
// now-complete data.
func TestTencentIncompleteNotCached(t *testing.T) {
	base := "/latest/meta-data"
	var mu sync.Mutex
	regionAvailable := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		region := regionAvailable
		mu.Unlock()
		switch r.URL.Path {
		case base + tencentPathInstanceID:
			_, _ = w.Write([]byte("ins-1"))
		case base + tencentPathRegion:
			if region {
				_, _ = w.Write([]byte("ap-guangzhou"))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	withTestClient(t, server)

	// First call: region missing -> incomplete -> not cached.
	id, err := GetTencentIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Region != "" {
		t.Fatalf("expected empty region, got %q", id.Region)
	}

	// Region becomes available; because nothing was cached, the retry sees it.
	mu.Lock()
	regionAvailable = true
	mu.Unlock()

	id, err = GetTencentIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Region != "ap-guangzhou" {
		t.Fatalf("expected region on retry, got %q", id.Region)
	}
}

// TestAWSIncompleteMacNotCached verifies a missing MAC keeps the result out of
// the cache so a subsequent successful MAC fetch is reflected.
func TestAWSIncompleteMacNotCached(t *testing.T) {
	var mu sync.Mutex
	macAvailable := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == awsPathToken {
			_, _ = w.Write([]byte(awsSampleToken))
			return
		}
		switch r.URL.Path {
		case awsPathIdentityDocument:
			_, _ = w.Write([]byte(awsSampleDoc))
		case awsPathMac:
			mu.Lock()
			ok := macAvailable
			mu.Unlock()
			if ok {
				_, _ = w.Write([]byte("06:1a:2b:3c:4d:5e"))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	withTestClient(t, server)

	id, err := GetAWSIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Mac != "" {
		t.Fatalf("expected empty mac, got %q", id.Mac)
	}

	mu.Lock()
	macAvailable = true
	mu.Unlock()

	id, err = GetAWSIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Mac != "06:1a:2b:3c:4d:5e" {
		t.Fatalf("expected mac on retry, got %q", id.Mac)
	}
}

// TestHuaweiIncompleteNotCached verifies a missing local-ipv4 keeps the result
// out of the cache.
func TestHuaweiIncompleteNotCached(t *testing.T) {
	var mu sync.Mutex
	ipAvailable := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case huaweiPathMetaData:
			_, _ = w.Write([]byte(huaweiSampleMeta))
		case huaweiPathLocalIPv4:
			mu.Lock()
			ok := ipAvailable
			mu.Unlock()
			if ok {
				_, _ = w.Write([]byte("192.1.1.2"))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	withTestClient(t, server)

	id, err := GetHuaweiIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.PrivateIpv4 != "" {
		t.Fatalf("expected empty ipv4, got %q", id.PrivateIpv4)
	}

	mu.Lock()
	ipAvailable = true
	mu.Unlock()

	id, err = GetHuaweiIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.PrivateIpv4 != "192.1.1.2" {
		t.Fatalf("expected ipv4 on retry, got %q", id.PrivateIpv4)
	}
}

// TestHuaweiAZFallbackFromEC2Path exercises the AZ EC2-path fallback when
// meta_data.json omits the availability zone.
func TestHuaweiAZFallbackFromEC2Path(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case huaweiPathMetaData:
			_, _ = w.Write([]byte(`{"uuid":"u-1","region_id":"cn-north-4"}`))
		case huaweiPathAZ:
			_, _ = w.Write([]byte("cn-north-4a"))
		case huaweiPathLocalIPv4:
			_, _ = w.Write([]byte("192.1.1.2"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	withTestClient(t, server)

	id, err := GetHuaweiIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Zone != "cn-north-4a" {
		t.Fatalf("expected zone from EC2 path, got %q", id.Zone)
	}
}

// TestDetectConcurrentSafe runs Detect from many goroutines against a server
// that answers as Aliyun, ensuring the concurrent probe path is race-free and
// deterministic under -race.
func TestDetectConcurrentSafe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/dynamic/instance-identity/document" {
			_, _ = w.Write([]byte(aliyunSampleDoc))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	withTestClient(t, server)

	const workers = 16
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			id, err := Detect()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if id.Provider != ALIYUN_CLOUD_TYPE {
				t.Errorf("unexpected provider: %q", id.Provider)
			}
		}()
	}
	wg.Wait()
}

// TestDetectMultipleProvidersRespond verifies that when more than one provider
// could answer, Detect deterministically returns the highest-priority one
// (Aliyun, which precedes Tencent in providerProbes) rather than whichever
// probe happens to return first.
func TestDetectMultipleProvidersRespond(t *testing.T) {
	// Both the Aliyun document path and the Tencent instance-id path answer.
	base := "/latest/meta-data"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest/dynamic/instance-identity/document":
			_, _ = w.Write([]byte(aliyunSampleDoc))
		case base + tencentPathInstanceID:
			_, _ = w.Write([]byte("ins-x"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	withTestClient(t, server)

	id, err := Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Provider != ALIYUN_CLOUD_TYPE {
		t.Fatalf("expected priority winner %q, got %q", ALIYUN_CLOUD_TYPE, id.Provider)
	}
}

// TestDetectLowerPriorityProviderWins verifies the winner is the earliest
// provider that actually responds, not simply the first in the list: when
// Aliyun's document path 404s and only Tencent answers, Detect returns Tencent.
func TestDetectLowerPriorityProviderWins(t *testing.T) {
	base := "/latest/meta-data"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case base + tencentPathInstanceID:
			_, _ = w.Write([]byte("ins-x"))
		default:
			// Aliyun document path and everything else 404, so only the
			// (lower-priority) Tencent probe succeeds.
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	withTestClient(t, server)

	id, err := Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Provider != TENCENT_CLOUD_TYPE {
		t.Fatalf("expected %q when only Tencent responds, got %q", TENCENT_CLOUD_TYPE, id.Provider)
	}
}

// TestCacheConcurrentReadWrite hammers the package cache concurrently to catch
// data races in the shared cache and SetCacheTTL under -race.
func TestCacheConcurrentReadWrite(t *testing.T) {
	t.Cleanup(func() {
		SetCacheTTL(defaultCacheTTL)
		ClearCache()
	})

	const workers = 32
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				switch j % 4 {
				case 0:
					defaultCache.set(ALIYUN_CLOUD_TYPE, []byte("payload"))
				case 1:
					_, _ = defaultCache.get(ALIYUN_CLOUD_TYPE)
				case 2:
					SetCacheTTL(defaultCacheTTL)
				case 3:
					ClearCache()
				}
			}
		}(i)
	}
	wg.Wait()
}
