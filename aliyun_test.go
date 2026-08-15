package cloudid

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const aliyunSampleDoc = `{
	"zone-id": "cn-hangzhou-a",
	"serial-number": "abc-123",
	"instance-id": "i-abc123",
	"region-id": "cn-hangzhou",
	"private-ipv4": "10.0.0.5",
	"owner-account-id": "1234567890",
	"mac": "00:16:3e:00:00:01",
	"image-id": "img-1",
	"instance-type": "ecs.g6.large"
}`

func aliyunTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(aliyunSampleDoc))
	}))
}

func TestSerializeAliyunInfo(t *testing.T) {
	info, err := SerializeAliyunInfo([]byte(aliyunSampleDoc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.InstanceID != "i-abc123" {
		t.Errorf("unexpected instance id: %q", info.InstanceID)
	}
	if info.RegionID != "cn-hangzhou" {
		t.Errorf("unexpected region: %q", info.RegionID)
	}
}

func TestSerializeAliyunInfo_Invalid(t *testing.T) {
	if _, err := SerializeAliyunInfo([]byte("not-json")); err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestAliyunGetters(t *testing.T) {
	server := aliyunTestServer(t)
	defer server.Close()
	withTestClient(t, server)

	id, err := GetAliyunIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.InstanceID != "i-abc123" {
		t.Errorf("unexpected identity: %+v", id)
	}

	cases := []struct {
		name string
		fn   func() (string, error)
		want string
	}{
		{"zone", GetAliyunZoneID, "cn-hangzhou-a"},
		{"region", GetAliyunRegionID, "cn-hangzhou"},
		{"instance", GetAliyunInstanceID, "i-abc123"},
		{"privateIPv4", GetAliyunPrivateIpv4, "10.0.0.5"},
		{"mac", GetAliyunMac, "00:16:3e:00:00:01"},
		{"serial", GetAliyunSerialNumber, "abc-123"},
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

func TestAliyunUsesCache(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte(aliyunSampleDoc))
	}))
	defer server.Close()
	withTestClient(t, server)

	if _, err := GetAliyunInfo(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := GetAliyunInfo(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hits != 1 {
		t.Fatalf("expected 1 upstream hit due to caching, got %d", hits)
	}
}

func TestAliyunError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	withTestClient(t, server)

	if _, err := GetAliyunInstanceID(); err == nil {
		t.Fatal("expected error when metadata endpoint fails")
	}
}
