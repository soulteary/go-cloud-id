package cloudid

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// newTestClient returns an http.Client whose requests are all routed to the
// given httptest server, regardless of the URL host. This lets tests keep the
// production metadata hostnames/URLs while serving canned responses locally.
func newTestClient(server *httptest.Server) *http.Client {
	return &http.Client{
		Timeout: 2 * time.Second,
		Transport: &redirectTransport{
			target: server.URL,
			base:   server.Client().Transport,
		},
	}
}

type redirectTransport struct {
	target string
	base   http.RoundTripper
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target, err := url.Parse(t.target)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// withTestClient swaps the package httpClient for the duration of the test.
func withTestClient(t *testing.T, server *httptest.Server) {
	t.Helper()
	original := httpClient
	httpClient = newTestClient(server)
	t.Cleanup(func() {
		httpClient = original
		defaultCache.clear()
	})
}
