package cloudid

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// awsSampleDoc is a trimmed EC2 instance identity document as returned by the
// metadata service.
const awsSampleDoc = `{
	"accountId": "123456789012",
	"architecture": "x86_64",
	"availabilityZone": "us-east-1a",
	"imageId": "ami-0abcd1234efgh5678",
	"instanceId": "i-0abc123def456",
	"instanceType": "t3.micro",
	"privateIp": "172.31.1.2",
	"region": "us-east-1"
}`

const awsSampleToken = "AQAEA-token-value"

// awsTestServer serves the IMDSv2 token endpoint and the identity document plus
// the MAC path. When requireToken is true, GET requests without a valid token
// header are rejected with 401, mirroring IMDSv2 enforcement.
func awsTestServer(t *testing.T, requireToken bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == awsPathToken {
			if r.Header.Get(awsHeaderTokenTTL) == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(awsSampleToken))
			return
		}

		if requireToken && r.Header.Get(awsHeaderToken) != awsSampleToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case awsPathIdentityDocument:
			_, _ = w.Write([]byte(awsSampleDoc))
		case awsPathMac:
			_, _ = w.Write([]byte("06:1a:2b:3c:4d:5e"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestSerializeAWSInfo(t *testing.T) {
	info, err := SerializeAWSInfo([]byte(`{"instance-id":"i-1","region":"us-east-1"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.InstanceID != "i-1" || info.Region != "us-east-1" {
		t.Errorf("unexpected identity: %+v", info)
	}
}

func TestSerializeAWSInfo_Invalid(t *testing.T) {
	if _, err := SerializeAWSInfo([]byte("nope")); err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestAWSGetters(t *testing.T) {
	server := awsTestServer(t, true)
	defer server.Close()
	withTestClient(t, server)

	id, err := GetAWSIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.InstanceID != "i-0abc123def456" {
		t.Errorf("unexpected identity: %+v", id)
	}
	if id.ImageID != "ami-0abcd1234efgh5678" {
		t.Errorf("unexpected image id: %q", id.ImageID)
	}
	if id.InstanceType != "t3.micro" {
		t.Errorf("unexpected instance type: %q", id.InstanceType)
	}

	cases := []struct {
		name string
		fn   func() (string, error)
		want string
	}{
		{"instance", GetAWSInstanceID, "i-0abc123def456"},
		{"region", GetAWSRegion, "us-east-1"},
		{"zone", GetAWSZone, "us-east-1a"},
		{"privateIPv4", GetAWSPrivateIpv4, "172.31.1.2"},
		{"mac", GetAWSMac, "06:1a:2b:3c:4d:5e"},
		{"account", GetAWSAccountID, "123456789012"},
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

// TestAWSIMDSv1Fallback ensures identity is retrievable even when the token
// endpoint is unavailable (IMDSv1-only instances).
func TestAWSIMDSv1Fallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == awsPathToken {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.URL.Path {
		case awsPathIdentityDocument:
			_, _ = w.Write([]byte(awsSampleDoc))
		case awsPathMac:
			_, _ = w.Write([]byte("06:1a:2b:3c:4d:5e"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	withTestClient(t, server)

	id, err := GetAWSInstanceID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "i-0abc123def456" {
		t.Errorf("unexpected instance id: %q", id)
	}
}

func TestAWSBestEffortMac(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == awsPathToken {
			_, _ = w.Write([]byte(awsSampleToken))
			return
		}
		if r.URL.Path == awsPathIdentityDocument {
			_, _ = w.Write([]byte(awsSampleDoc))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	withTestClient(t, server)

	id, err := GetAWSIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.InstanceID == "" {
		t.Error("expected instance id to be populated")
	}
	if id.Mac != "" {
		t.Errorf("expected empty mac, got %q", id.Mac)
	}
}

func TestAWSEmptyInstanceIDErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == awsPathToken {
			_, _ = w.Write([]byte(awsSampleToken))
			return
		}
		if r.URL.Path == awsPathIdentityDocument {
			_, _ = w.Write([]byte(`{"region":"us-east-1"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	withTestClient(t, server)

	if _, err := GetAWSInstanceID(); err == nil {
		t.Fatal("expected error when instance id is empty")
	}
}

func TestAWSMissingDocumentErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == awsPathToken {
			_, _ = w.Write([]byte(awsSampleToken))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	withTestClient(t, server)

	if _, err := GetAWSInstanceID(); err == nil {
		t.Fatal("expected error when identity document is unavailable")
	}
}

func TestAWSUsesCache(t *testing.T) {
	var docHits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == awsPathToken {
			_, _ = w.Write([]byte(awsSampleToken))
			return
		}
		switch r.URL.Path {
		case awsPathIdentityDocument:
			docHits++
			_, _ = w.Write([]byte(awsSampleDoc))
		default:
			_, _ = w.Write([]byte("06:1a:2b:3c:4d:5e"))
		}
	}))
	defer server.Close()
	withTestClient(t, server)

	if _, err := GetAWSInfo(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := GetAWSInfo(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if docHits != 1 {
		t.Fatalf("expected caching to avoid re-fetch, identity document hits = %d", docHits)
	}
}
