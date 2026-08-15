package cloudid

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetect_Aliyun(t *testing.T) {
	// The aliyun identity path returns a valid document.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/dynamic/instance-identity/document" {
			_, _ = w.Write([]byte(aliyunSampleDoc))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	withTestClient(t, server)

	id, err := Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Provider != ALIYUN_CLOUD_TYPE || id.InstanceID != "i-abc123" {
		t.Fatalf("unexpected identity: %+v", id)
	}

	provider, err := DetectProvider()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != ALIYUN_CLOUD_TYPE {
		t.Fatalf("unexpected provider: %q", provider)
	}
}

func TestDetect_Tencent(t *testing.T) {
	base := "/latest/meta-data"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case base + tencentPathInstanceID:
			_, _ = w.Write([]byte("ins-detect"))
		case base + tencentPathRegion:
			_, _ = w.Write([]byte("ap-shanghai"))
		default:
			// Aliyun document path and other tencent fields 404.
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	withTestClient(t, server)

	id, err := Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Provider != TENCENT_CLOUD_TYPE || id.InstanceID != "ins-detect" {
		t.Fatalf("unexpected identity: %+v", id)
	}
	if id.Region != "ap-shanghai" {
		t.Fatalf("unexpected region: %q", id.Region)
	}
}

func TestDetect_Huawei(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case huaweiPathMetaData:
			_, _ = w.Write([]byte(huaweiSampleMeta))
		case huaweiPathLocalIPv4:
			_, _ = w.Write([]byte("192.1.1.2"))
		default:
			// Aliyun document path and tencent instance-id 404 so those
			// providers are skipped and huawei wins.
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	withTestClient(t, server)

	id, err := Detect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Provider != HUAWEI_CLOUD_TYPE {
		t.Fatalf("unexpected provider: %q", id.Provider)
	}
	if id.InstanceID != "ca9e8b7c-f2be-4b6d-a639-f10b4d994d04" {
		t.Fatalf("unexpected identity: %+v", id)
	}
	if id.Region != "cn-north-4" || id.PrivateIPv4 != "192.1.1.2" {
		t.Fatalf("unexpected identity: %+v", id)
	}

	got, err := GetIdentity(HUAWEI_CLOUD_TYPE)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Provider != HUAWEI_CLOUD_TYPE {
		t.Fatalf("unexpected provider: %q", got.Provider)
	}
}

func TestDetect_None(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	withTestClient(t, server)

	if _, err := Detect(); err != ErrNotDetected {
		t.Fatalf("expected ErrNotDetected, got %v", err)
	}
	if _, err := DetectProvider(); err != ErrNotDetected {
		t.Fatalf("expected ErrNotDetected, got %v", err)
	}
}

func TestGetIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/dynamic/instance-identity/document" {
			_, _ = w.Write([]byte(aliyunSampleDoc))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	withTestClient(t, server)

	id, err := GetIdentity(ALIYUN_CLOUD_TYPE)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Provider != ALIYUN_CLOUD_TYPE {
		t.Fatalf("unexpected provider: %q", id.Provider)
	}

	if _, err := GetIdentity("unknown"); err != ErrNotDetected {
		t.Fatalf("expected ErrNotDetected for unknown provider, got %v", err)
	}
}
