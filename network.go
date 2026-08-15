package cloudid

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// defaultTimeout is the per-request timeout used by the default HTTP client.
const defaultTimeout = 3 * time.Second

// maxResponseBytes caps how much of a metadata response body is read to
// protect against unexpectedly large or malicious responses.
const maxResponseBytes = 1 << 20 // 1 MiB

// httpClient is the client used for all metadata requests. It is a variable so
// that tests can override it with a client pointing at an httptest server.
var httpClient = &http.Client{Timeout: defaultTimeout}

// requestOption customizes an outgoing metadata request (e.g. adding headers).
type requestOption func(*http.Request)

// withHeader sets a request header before the request is sent.
func withHeader(key, value string) requestOption {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// get performs an HTTP GET against url and returns the response body.
// It uses the package context-aware fetcher with a background context.
func get(url string, opts ...requestOption) ([]byte, error) {
	return getContext(context.Background(), url, opts...)
}

// getContext performs a context-aware HTTP GET against url and returns the body.
func getContext(ctx context.Context, url string, opts ...requestOption) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(req)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request to %s failed with status code %d", url, res.StatusCode)
	}

	return io.ReadAll(io.LimitReader(res.Body, maxResponseBytes))
}
