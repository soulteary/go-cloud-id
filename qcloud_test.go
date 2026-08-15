package cloudid

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// tencentTestServer serves canned responses for each metadata sub-path.
func tencentTestServer(t *testing.T, fields map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value, ok := fields[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(value))
	}))
}

func fullTencentFields() map[string]string {
	base := "/latest/meta-data"
	return map[string]string{
		base + tencentPathInstanceID: "ins-3g445roi\n",
		base + tencentPathRegion:     "ap-guangzhou",
		base + tencentPathZone:       "ap-guangzhou-3",
		base + tencentPathLocalIPv4:  "10.104.13.59",
		base + tencentPathMac:        "52:54:00:00:00:01",
		base + tencentPathUUID:       "uuid-xyz",
	}
}

func TestTencentGetters(t *testing.T) {
	server := tencentTestServer(t, fullTencentFields())
	defer server.Close()
	withTestClient(t, server)

	id, err := GetTencentIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.InstanceID != "ins-3g445roi" {
		t.Errorf("expected whitespace trimmed instance id, got %q", id.InstanceID)
	}

	cases := []struct {
		name string
		fn   func() (string, error)
		want string
	}{
		{"instance", GetTencentInstanceID, "ins-3g445roi"},
		{"region", GetTencentRegion, "ap-guangzhou"},
		{"zone", GetTencentZone, "ap-guangzhou-3"},
		{"privateIPv4", GetTencentPrivateIpv4, "10.104.13.59"},
		{"mac", GetTencentMac, "52:54:00:00:00:01"},
		{"uuid", GetTencentUUID, "uuid-xyz"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.fn()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTencentBestEffortFields(t *testing.T) {
	// Only instance-id is available; other fields 404 and should be empty.
	base := "/latest/meta-data"
	server := tencentTestServer(t, map[string]string{
		base + tencentPathInstanceID: "ins-only",
	})
	defer server.Close()
	withTestClient(t, server)

	id, err := GetTencentIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.InstanceID != "ins-only" {
		t.Errorf("unexpected instance id: %q", id.InstanceID)
	}
	if id.Region != "" || id.Zone != "" {
		t.Errorf("expected missing fields to be empty, got %+v", id)
	}
}

func TestTencentMissingInstanceIDErrors(t *testing.T) {
	server := tencentTestServer(t, map[string]string{})
	defer server.Close()
	withTestClient(t, server)

	if _, err := GetTencentInstanceID(); err == nil {
		t.Fatal("expected error when instance-id is unavailable")
	}
}

func TestSerializeTencentInfo_Invalid(t *testing.T) {
	if _, err := SerializeTencentInfo([]byte("nope")); err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestTencentUsesCache(t *testing.T) {
	var hits int
	base := "/latest/meta-data"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == base+tencentPathInstanceID {
			hits++
		}
		_, _ = w.Write([]byte("value"))
	}))
	defer server.Close()
	withTestClient(t, server)

	if _, err := GetTencentInfo(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := GetTencentInfo(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hits != 1 {
		t.Fatalf("expected caching to avoid re-fetch, instance-id hits = %d", hits)
	}
}
