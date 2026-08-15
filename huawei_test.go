package cloudid

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// huaweiSampleMeta is a trimmed OpenStack meta_data.json document as returned
// by the Huawei Cloud metadata service.
const huaweiSampleMeta = `{
	"uuid": "ca9e8b7c-f2be-4b6d-a639-f10b4d994d04",
	"availability_zone": "cn-north-4a",
	"region_id": "cn-north-4",
	"project_id": "6e8b0c94265645f39c5abbe63c4113c6",
	"name": "ecs-ddd4",
	"meta": {
		"metering.image_id": "3a64bd37-955e-40cd-ab9e-129db56bc05d",
		"image_name": "CentOS 7.6 64bit",
		"vpc_id": "3b6c201f-aeb3-4bce-b841-64756e66cb49"
	}
}`

// huaweiTestServer serves the OpenStack meta_data.json document plus the
// EC2-compatible fields keyed by path.
func huaweiTestServer(t *testing.T, fields map[string]string) *httptest.Server {
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

func fullHuaweiFields() map[string]string {
	return map[string]string{
		huaweiPathMetaData:  huaweiSampleMeta,
		huaweiPathLocalIPv4: "192.1.1.2",
	}
}

func TestHuaweiGetters(t *testing.T) {
	server := huaweiTestServer(t, fullHuaweiFields())
	defer server.Close()
	withTestClient(t, server)

	id, err := GetHuaweiIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.InstanceID != "ca9e8b7c-f2be-4b6d-a639-f10b4d994d04" {
		t.Errorf("unexpected instance id: %q", id.InstanceID)
	}
	if id.VpcID != "3b6c201f-aeb3-4bce-b841-64756e66cb49" {
		t.Errorf("unexpected vpc id: %q", id.VpcID)
	}
	if id.ImageID != "3a64bd37-955e-40cd-ab9e-129db56bc05d" {
		t.Errorf("unexpected image id: %q", id.ImageID)
	}

	cases := []struct {
		name string
		fn   func() (string, error)
		want string
	}{
		{"instance", GetHuaweiInstanceID, "ca9e8b7c-f2be-4b6d-a639-f10b4d994d04"},
		{"region", GetHuaweiRegion, "cn-north-4"},
		{"zone", GetHuaweiZone, "cn-north-4a"},
		{"privateIPv4", GetHuaweiPrivateIpv4, "192.1.1.2"},
		{"project", GetHuaweiProjectID, "6e8b0c94265645f39c5abbe63c4113c6"},
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

func TestHuaweiRegionFallbackFromZone(t *testing.T) {
	// meta_data.json without region_id: region is derived from the AZ.
	meta := `{
		"uuid": "u-1",
		"availability_zone": "cn-east-3b"
	}`
	server := huaweiTestServer(t, map[string]string{huaweiPathMetaData: meta})
	defer server.Close()
	withTestClient(t, server)

	id, err := GetHuaweiIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.Region != "cn-east-3" {
		t.Errorf("expected region derived from zone, got %q", id.Region)
	}
}

func TestHuaweiBestEffortFields(t *testing.T) {
	// Only meta_data.json is available; local-ipv4 404s and stays empty.
	server := huaweiTestServer(t, map[string]string{huaweiPathMetaData: huaweiSampleMeta})
	defer server.Close()
	withTestClient(t, server)

	id, err := GetHuaweiIdentity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.InstanceID == "" {
		t.Error("expected instance id to be populated")
	}
	if id.PrivateIpv4 != "" {
		t.Errorf("expected empty private ipv4, got %q", id.PrivateIpv4)
	}
}

func TestHuaweiMissingMetaErrors(t *testing.T) {
	server := huaweiTestServer(t, map[string]string{})
	defer server.Close()
	withTestClient(t, server)

	if _, err := GetHuaweiInstanceID(); err == nil {
		t.Fatal("expected error when meta_data.json is unavailable")
	}
}

func TestHuaweiEmptyUUIDErrors(t *testing.T) {
	server := huaweiTestServer(t, map[string]string{
		huaweiPathMetaData: `{"availability_zone": "cn-north-4a"}`,
	})
	defer server.Close()
	withTestClient(t, server)

	if _, err := GetHuaweiInstanceID(); err == nil {
		t.Fatal("expected error when uuid is empty")
	}
}

func TestSerializeHuaweiInfo_Invalid(t *testing.T) {
	if _, err := SerializeHuaweiInfo([]byte("nope")); err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

func TestHuaweiUsesCache(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == huaweiPathMetaData {
			hits++
			_, _ = w.Write([]byte(huaweiSampleMeta))
			return
		}
		_, _ = w.Write([]byte("192.1.1.2"))
	}))
	defer server.Close()
	withTestClient(t, server)

	if _, err := GetHuaweiInfo(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := GetHuaweiInfo(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hits != 1 {
		t.Fatalf("expected caching to avoid re-fetch, meta_data.json hits = %d", hits)
	}
}
