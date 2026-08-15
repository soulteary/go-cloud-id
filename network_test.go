package cloudid

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test"); got != "yes" {
			t.Errorf("expected header to be forwarded, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	defer server.Close()
	withTestClient(t, server)

	body, err := get(server.URL, withHeader("X-Test", "yes"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "hello" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestGet_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	withTestClient(t, server)

	_, err := get(server.URL)
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected status code in error, got %v", err)
	}
}

func TestGet_LimitsBody(t *testing.T) {
	// Serve more than maxResponseBytes; body must be capped.
	oversized := strings.Repeat("a", maxResponseBytes+1024)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(oversized))
	}))
	defer server.Close()
	withTestClient(t, server)

	body, err := get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) != maxResponseBytes {
		t.Fatalf("expected body capped at %d, got %d", maxResponseBytes, len(body))
	}
}

func TestGet_BadURL(t *testing.T) {
	if _, err := get("http://[::1]:namedport"); err == nil {
		t.Fatal("expected error for malformed URL")
	}
}
