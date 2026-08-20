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
	return do(ctx, http.MethodGet, url, opts...)
}

// put performs an HTTP PUT against url and returns the response body, using a
// background context. It is the context-free counterpart to putContext.
func put(url string, opts ...requestOption) ([]byte, error) {
	return putContext(context.Background(), url, opts...)
}

// putContext performs a context-aware HTTP PUT against url and returns the
// body. It is used by IMDSv2-style metadata services (e.g. AWS) to fetch a
// session token.
func putContext(ctx context.Context, url string, opts ...requestOption) ([]byte, error) {
	return do(ctx, http.MethodPut, url, opts...)
}

// do performs a context-aware HTTP request against url and returns the body.
//
// Errors are classified so callers can distinguish "this is not that cloud"
// from "the metadata service is unreachable/broken":
//   - transport-layer failures (timeouts, connection refused, DNS) and 5xx
//     responses are wrapped as ErrMetadataUnavailable.
//   - 4xx responses (including 404) return a plain error and are treated as a
//     "not this cloud" signal by the detection path.
func do(ctx context.Context, method, url string, opts ...requestOption) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(req)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: request to %s failed: %v", ErrMetadataUnavailable, url, err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		if res.StatusCode >= 500 {
			return nil, fmt.Errorf("%w: request to %s failed with status code %d", ErrMetadataUnavailable, url, res.StatusCode)
		}
		return nil, fmt.Errorf("request to %s failed with status code %d", url, res.StatusCode)
	}

	return io.ReadAll(io.LimitReader(res.Body, maxResponseBytes))
}
