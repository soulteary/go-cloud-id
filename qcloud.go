package cloudid

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Tencent Cloud (QCloud) instance metadata.
// Documentation: https://www.tencentcloud.com/document/product/213/4934
//
// Unlike Aliyun, Tencent Cloud does not expose a single JSON identity document.
// Each metadata field is fetched from its own path under the metadata root.

// tencentMetadataBase is the metadata service root. It is a variable so tests
// can point it at a local server.
var tencentMetadataBase = "http://metadata.tencentyun.com/latest/meta-data"

// Tencent metadata sub-paths.
const (
	tencentPathInstanceID = "/instance-id"
	tencentPathRegion     = "/placement/region"
	tencentPathZone       = "/placement/zone"
	tencentPathLocalIPv4  = "/local-ipv4"
	tencentPathMac        = "/mac"
	tencentPathUUID       = "/uuid"
)

// TencentIdentity holds the Tencent Cloud instance metadata fields.
type TencentIdentity struct {
	InstanceID  string `json:"instance-id"`
	Region      string `json:"region"`
	Zone        string `json:"zone"`
	PrivateIpv4 string `json:"local-ipv4"`
	Mac         string `json:"mac"`
	UUID        string `json:"uuid"`
}

// TENCENT_INDENTITY is a backward-compatible alias for TencentIdentity.
//
// Deprecated: use TencentIdentity instead.
type TENCENT_INDENTITY = TencentIdentity

// fetchTencentField retrieves a single metadata field, trimming surrounding whitespace.
func fetchTencentField(ctx context.Context, path string) (string, error) {
	body, err := getContext(ctx, tencentMetadataBase+path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// GetTencentInfo returns the Tencent metadata assembled as a JSON document,
// using the cache when fresh.
func GetTencentInfo() ([]byte, error) {
	return getTencentInfoContext(context.Background())
}

func getTencentInfoContext(ctx context.Context) ([]byte, error) {
	if data, ok := defaultCache.get(TENCENT_CLOUD_TYPE); ok {
		return data, nil
	}

	info, complete, err := fetchTencentIdentity(ctx)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshal tencent info failed: %w", err)
	}

	// Only cache complete results: a best-effort field that failed this time
	// might succeed on retry, so we avoid freezing a partial document.
	if complete {
		defaultCache.set(TENCENT_CLOUD_TYPE, data)
	}
	return data, nil
}

// SerializeTencentInfo parses a raw Tencent identity JSON document.
func SerializeTencentInfo(data []byte) (info TencentIdentity, err error) {
	if err = json.Unmarshal(data, &info); err != nil {
		return info, err
	}
	return info, nil
}

// fetchTencentIdentity queries each metadata field directly from the service.
// It reports complete=false when any best-effort field could not be fetched, so
// callers can avoid caching a partial result.
func fetchTencentIdentity(ctx context.Context) (info TencentIdentity, complete bool, err error) {
	// instance-id is mandatory: its absence means this is not a Tencent instance.
	instanceID, err := fetchTencentField(ctx, tencentPathInstanceID)
	if err != nil {
		return info, false, fmt.Errorf("getting tencent info failed: %w", err)
	}
	info.InstanceID = instanceID

	// Remaining fields are best-effort; track whether all of them succeeded.
	complete = true
	fields := []struct {
		path string
		dst  *string
	}{
		{tencentPathRegion, &info.Region},
		{tencentPathZone, &info.Zone},
		{tencentPathLocalIPv4, &info.PrivateIpv4},
		{tencentPathMac, &info.Mac},
		{tencentPathUUID, &info.UUID},
	}
	for _, f := range fields {
		v, ferr := fetchTencentField(ctx, f.path)
		if ferr != nil {
			complete = false
			continue
		}
		*f.dst = v
	}

	return info, complete, nil
}

func parseTencentInfo() (TencentIdentity, error) {
	return parseTencentInfoContext(context.Background())
}

func parseTencentInfoContext(ctx context.Context) (TencentIdentity, error) {
	data, err := getTencentInfoContext(ctx)
	if err != nil {
		return TencentIdentity{}, err
	}

	parsed, err := SerializeTencentInfo(data)
	if err != nil {
		return TencentIdentity{}, fmt.Errorf("serialize tencent info failed: %w", err)
	}
	return parsed, nil
}

// GetTencentIdentity returns the full parsed Tencent identity.
func GetTencentIdentity() (TencentIdentity, error) {
	return parseTencentInfo()
}

// GetTencentInstanceID returns the instance ID.
func GetTencentInstanceID() (string, error) {
	return field(parseTencentInfo, func(i TencentIdentity) string { return i.InstanceID })
}

// GetTencentRegion returns the instance region.
func GetTencentRegion() (string, error) {
	return field(parseTencentInfo, func(i TencentIdentity) string { return i.Region })
}

// GetTencentZone returns the instance availability zone.
func GetTencentZone() (string, error) {
	return field(parseTencentInfo, func(i TencentIdentity) string { return i.Zone })
}

// GetTencentPrivateIpv4 returns the private IPv4 address.
func GetTencentPrivateIpv4() (string, error) {
	return field(parseTencentInfo, func(i TencentIdentity) string { return i.PrivateIpv4 })
}

// GetTencentMac returns the MAC address of the primary network interface.
func GetTencentMac() (string, error) {
	return field(parseTencentInfo, func(i TencentIdentity) string { return i.Mac })
}

// GetTencentUUID returns the instance UUID.
func GetTencentUUID() (string, error) {
	return field(parseTencentInfo, func(i TencentIdentity) string { return i.UUID })
}

// tencentIdentity fetches and normalizes the Tencent identity for the generic API.
func tencentIdentity(ctx context.Context) (Identity, error) {
	info, err := parseTencentInfoContext(ctx)
	if err != nil {
		return Identity{}, err
	}
	return Identity{
		Provider:    TENCENT_CLOUD_TYPE,
		InstanceID:  info.InstanceID,
		Region:      info.Region,
		Zone:        info.Zone,
		PrivateIPv4: info.PrivateIpv4,
		Mac:         info.Mac,
	}, nil
}
