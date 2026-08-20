package cloudid

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Huawei Cloud (HuaweiCloud) instance metadata.
// Documentation: https://support.huaweicloud.com/eu/usermanual-ecs/ecs_03_0166.html
//
// Huawei Cloud exposes an OpenStack-style metadata document at
// /openstack/latest/meta_data.json (instance id, availability zone, region,
// etc.) but not the private IPv4 address. The private IPv4 and MAC are read
// from the EC2-compatible metadata paths instead. The library fetches these
// sources, assembles them into a single JSON document, and caches the result.

// huaweiMetadataBase is the metadata service root. It is a variable so tests
// can point it at a local server.
var huaweiMetadataBase = "http://169.254.169.254"

// Huawei metadata sub-paths.
const (
	huaweiPathMetaData  = "/openstack/latest/meta_data.json"
	huaweiPathLocalIPv4 = "/latest/meta-data/local-ipv4"
	huaweiPathAZ        = "/latest/meta-data/placement/availability-zone"
)

// huaweiOpenStackMeta mirrors the fields of the OpenStack meta_data.json
// document that we care about.
type huaweiOpenStackMeta struct {
	UUID             string            `json:"uuid"`
	AvailabilityZone string            `json:"availability_zone"`
	RegionID         string            `json:"region_id"`
	ProjectID        string            `json:"project_id"`
	Name             string            `json:"name"`
	Meta             map[string]string `json:"meta"`
}

// HuaweiIdentity holds the normalized Huawei Cloud instance metadata fields.
type HuaweiIdentity struct {
	InstanceID  string `json:"instance-id"`
	Region      string `json:"region"`
	Zone        string `json:"zone"`
	PrivateIpv4 string `json:"local-ipv4"`
	ProjectID   string `json:"project-id"`
	VpcID       string `json:"vpc-id"`
	ImageID     string `json:"image-id"`
}

// fetchHuaweiField retrieves a single metadata field, trimming surrounding whitespace.
func fetchHuaweiField(ctx context.Context, path string) (string, error) {
	body, err := getContext(ctx, huaweiMetadataBase+path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// GetHuaweiInfo returns the Huawei metadata assembled as a JSON document,
// using the cache when fresh.
func GetHuaweiInfo() ([]byte, error) {
	return getHuaweiInfoContext(context.Background())
}

func getHuaweiInfoContext(ctx context.Context) ([]byte, error) {
	if data, ok := defaultCache.get(HUAWEI_CLOUD_TYPE); ok {
		return data, nil
	}

	info, complete, err := fetchHuaweiIdentity(ctx)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshal huawei info failed: %w", err)
	}

	// Only cache complete results so a transiently missing best-effort field is
	// retried rather than frozen for the TTL window.
	if complete {
		defaultCache.set(HUAWEI_CLOUD_TYPE, data)
	}
	return data, nil
}

// SerializeHuaweiInfo parses a raw Huawei identity JSON document.
func SerializeHuaweiInfo(data []byte) (info HuaweiIdentity, err error) {
	if err = json.Unmarshal(data, &info); err != nil {
		return info, err
	}
	return info, nil
}

// fetchHuaweiIdentity queries the OpenStack metadata document and the
// EC2-compatible paths, then normalizes them into a HuaweiIdentity. It reports
// complete=false when a best-effort field (local-ipv4 or the AZ fallback) could
// not be fetched, so callers can avoid caching a partial result.
func fetchHuaweiIdentity(ctx context.Context) (info HuaweiIdentity, complete bool, err error) {
	// The OpenStack meta_data.json document is mandatory: its absence (or an
	// unparsable body) means this is not a Huawei Cloud instance.
	body, err := getContext(ctx, huaweiMetadataBase+huaweiPathMetaData)
	if err != nil {
		return info, false, fmt.Errorf("getting huawei info failed: %w", err)
	}

	var meta huaweiOpenStackMeta
	if err := json.Unmarshal(body, &meta); err != nil {
		return info, false, fmt.Errorf("serialize huawei meta_data.json failed: %w", err)
	}
	if meta.UUID == "" {
		return info, false, fmt.Errorf("getting huawei info failed: empty instance uuid")
	}

	info.InstanceID = meta.UUID
	info.Region = meta.RegionID
	info.Zone = meta.AvailabilityZone
	info.ProjectID = meta.ProjectID
	if meta.Meta != nil {
		info.VpcID = meta.Meta["vpc_id"]
		info.ImageID = meta.Meta["metering.image_id"]
	}

	// Region is not always present in meta_data.json; derive it from the AZ as
	// a best-effort fallback (e.g. "cn-north-4a" -> "cn-north-4").
	if info.Region == "" {
		info.Region = huaweiRegionFromZone(info.Zone)
	}

	complete = true

	// local-ipv4 is best-effort.
	if ip, ferr := fetchHuaweiField(ctx, huaweiPathLocalIPv4); ferr == nil {
		info.PrivateIpv4 = ip
	} else {
		complete = false
	}

	// AZ is only fetched from the EC2 path when meta_data.json omitted it.
	if info.Zone == "" {
		if az, ferr := fetchHuaweiField(ctx, huaweiPathAZ); ferr == nil {
			info.Zone = az
		} else {
			complete = false
		}
	}

	return info, complete, nil
}

// huaweiRegionFromZone derives a region id from an availability zone id by
// dropping a trailing single-letter/az suffix. It returns an empty string when
// no reasonable region can be derived.
func huaweiRegionFromZone(zone string) string {
	zone = strings.TrimSpace(zone)
	if zone == "" {
		return ""
	}
	// AZ ids commonly look like "<region><letter>" (cn-north-4a) or
	// "<region>.<suffix>" (az1.dc1). Only the former maps cleanly to a region.
	if i := strings.LastIndex(zone, "-"); i > 0 && i < len(zone)-1 {
		last := zone[i+1:]
		// e.g. "4a" -> strip trailing letters to get the region segment "4".
		trimmed := strings.TrimRightFunc(last, func(r rune) bool {
			return r >= 'a' && r <= 'z'
		})
		if trimmed != "" && trimmed != last {
			return zone[:i+1] + trimmed
		}
	}
	return ""
}

func parseHuaweiInfo() (HuaweiIdentity, error) {
	return parseHuaweiInfoContext(context.Background())
}

func parseHuaweiInfoContext(ctx context.Context) (HuaweiIdentity, error) {
	data, err := getHuaweiInfoContext(ctx)
	if err != nil {
		return HuaweiIdentity{}, err
	}

	parsed, err := SerializeHuaweiInfo(data)
	if err != nil {
		return HuaweiIdentity{}, fmt.Errorf("serialize huawei info failed: %w", err)
	}
	return parsed, nil
}

// GetHuaweiIdentity returns the full parsed Huawei identity.
func GetHuaweiIdentity() (HuaweiIdentity, error) {
	return parseHuaweiInfo()
}

// GetHuaweiInstanceID returns the instance ID.
func GetHuaweiInstanceID() (string, error) {
	return field(parseHuaweiInfo, func(i HuaweiIdentity) string { return i.InstanceID })
}

// GetHuaweiRegion returns the instance region.
func GetHuaweiRegion() (string, error) {
	return field(parseHuaweiInfo, func(i HuaweiIdentity) string { return i.Region })
}

// GetHuaweiZone returns the instance availability zone.
func GetHuaweiZone() (string, error) {
	return field(parseHuaweiInfo, func(i HuaweiIdentity) string { return i.Zone })
}

// GetHuaweiPrivateIpv4 returns the private IPv4 address.
func GetHuaweiPrivateIpv4() (string, error) {
	return field(parseHuaweiInfo, func(i HuaweiIdentity) string { return i.PrivateIpv4 })
}

// GetHuaweiProjectID returns the project ID accommodating the instance.
func GetHuaweiProjectID() (string, error) {
	return field(parseHuaweiInfo, func(i HuaweiIdentity) string { return i.ProjectID })
}

// huaweiIdentity fetches and normalizes the Huawei identity for the generic API.
func huaweiIdentity(ctx context.Context) (Identity, error) {
	info, err := parseHuaweiInfoContext(ctx)
	if err != nil {
		return Identity{}, err
	}
	return Identity{
		Provider:    HUAWEI_CLOUD_TYPE,
		InstanceID:  info.InstanceID,
		Region:      info.Region,
		Zone:        info.Zone,
		PrivateIPv4: info.PrivateIpv4,
	}, nil
}
