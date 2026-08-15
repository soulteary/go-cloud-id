package cloudid

import (
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
func fetchTencentField(path string) (string, error) {
	body, err := get(tencentMetadataBase + path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// GetTencentInfo returns the Tencent metadata assembled as a JSON document,
// using the cache when fresh.
func GetTencentInfo() ([]byte, error) {
	if data, ok := defaultCache.get(TENCENT_CLOUD_TYPE); ok {
		return data, nil
	}

	info, err := fetchTencentIdentity()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshal tencent info failed: %w", err)
	}

	defaultCache.set(TENCENT_CLOUD_TYPE, data)
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
func fetchTencentIdentity() (TencentIdentity, error) {
	var info TencentIdentity

	// instance-id is mandatory: its absence means this is not a Tencent instance.
	instanceID, err := fetchTencentField(tencentPathInstanceID)
	if err != nil {
		return info, fmt.Errorf("getting tencent info failed: %w", err)
	}
	info.InstanceID = instanceID

	// Remaining fields are best-effort; ignore individual field errors.
	info.Region, _ = fetchTencentField(tencentPathRegion)
	info.Zone, _ = fetchTencentField(tencentPathZone)
	info.PrivateIpv4, _ = fetchTencentField(tencentPathLocalIPv4)
	info.Mac, _ = fetchTencentField(tencentPathMac)
	info.UUID, _ = fetchTencentField(tencentPathUUID)

	return info, nil
}

func parseTencentInfo() (TencentIdentity, error) {
	data, err := GetTencentInfo()
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
	info, err := parseTencentInfo()
	if err != nil {
		return "", err
	}
	return info.InstanceID, nil
}

// GetTencentRegion returns the instance region.
func GetTencentRegion() (string, error) {
	info, err := parseTencentInfo()
	if err != nil {
		return "", err
	}
	return info.Region, nil
}

// GetTencentZone returns the instance availability zone.
func GetTencentZone() (string, error) {
	info, err := parseTencentInfo()
	if err != nil {
		return "", err
	}
	return info.Zone, nil
}

// GetTencentPrivateIpv4 returns the private IPv4 address.
func GetTencentPrivateIpv4() (string, error) {
	info, err := parseTencentInfo()
	if err != nil {
		return "", err
	}
	return info.PrivateIpv4, nil
}

// GetTencentMac returns the MAC address of the primary network interface.
func GetTencentMac() (string, error) {
	info, err := parseTencentInfo()
	if err != nil {
		return "", err
	}
	return info.Mac, nil
}

// GetTencentUUID returns the instance UUID.
func GetTencentUUID() (string, error) {
	info, err := parseTencentInfo()
	if err != nil {
		return "", err
	}
	return info.UUID, nil
}

// tencentIdentity fetches and normalizes the Tencent identity for the generic API.
func tencentIdentity() (Identity, error) {
	info, err := parseTencentInfo()
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
