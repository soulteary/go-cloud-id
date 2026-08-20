package cloudid

import (
	"context"
	"encoding/json"
	"fmt"
)

// Alibaba Cloud (Aliyun) instance identity.
// Documentation: https://www.alibabacloud.com/help/en/ecs/user-guide/use-instance-identities

// aliyunIdentityURL returns the instance identity document endpoint.
// It is a variable so tests can point it at a local server.
var aliyunIdentityURL = "http://100.100.100.200/latest/dynamic/instance-identity/document"

// AliyunIdentity mirrors the Aliyun instance identity document.
type AliyunIdentity struct {
	ZoneID         string `json:"zone-id"`
	SerialNumber   string `json:"serial-number"`
	InstanceID     string `json:"instance-id"`
	RegionID       string `json:"region-id"`
	PrivateIpv4    string `json:"private-ipv4"`
	OwnerAccountID string `json:"owner-account-id"`
	Mac            string `json:"mac"`
	ImageID        string `json:"image-id"`
	InstanceType   string `json:"instance-type"`
}

// ALIYUN_INDENTITY is a backward-compatible alias for AliyunIdentity.
//
// Deprecated: use AliyunIdentity instead. This alias preserves compatibility
// with releases prior to v0.2.0 and may be removed in a future major version.
type ALIYUN_INDENTITY = AliyunIdentity

// GetAliyunInfo returns the raw Aliyun identity document, using the cache when fresh.
func GetAliyunInfo() ([]byte, error) {
	return getAliyunInfoContext(context.Background())
}

func getAliyunInfoContext(ctx context.Context) ([]byte, error) {
	if data, ok := defaultCache.get(ALIYUN_CLOUD_TYPE); ok {
		return data, nil
	}

	remote, err := getContext(ctx, aliyunIdentityURL)
	if err != nil {
		return nil, err
	}

	defaultCache.set(ALIYUN_CLOUD_TYPE, remote)
	return remote, nil
}

// SerializeAliyunInfo parses a raw Aliyun identity document.
func SerializeAliyunInfo(data []byte) (info AliyunIdentity, err error) {
	if err = json.Unmarshal(data, &info); err != nil {
		return info, err
	}
	return info, nil
}

func parseAliyunInfo() (info AliyunIdentity, err error) {
	return parseAliyunInfoContext(context.Background())
}

func parseAliyunInfoContext(ctx context.Context) (info AliyunIdentity, err error) {
	data, err := getAliyunInfoContext(ctx)
	if err != nil {
		return info, fmt.Errorf("getting aliyun info failed: %w", err)
	}

	parsed, err := SerializeAliyunInfo(data)
	if err != nil {
		return info, fmt.Errorf("serialize aliyun info failed: %w", err)
	}
	return parsed, nil
}

// GetAliyunIdentity returns the full parsed Aliyun identity document.
func GetAliyunIdentity() (AliyunIdentity, error) {
	return parseAliyunInfo()
}

// GetAliyunZoneID returns the instance zone ID.
func GetAliyunZoneID() (string, error) {
	return field(parseAliyunInfo, func(i AliyunIdentity) string { return i.ZoneID })
}

// GetAliyunRegionID returns the instance region ID.
func GetAliyunRegionID() (string, error) {
	return field(parseAliyunInfo, func(i AliyunIdentity) string { return i.RegionID })
}

// GetAliyunInstanceID returns the instance ID.
func GetAliyunInstanceID() (string, error) {
	return field(parseAliyunInfo, func(i AliyunIdentity) string { return i.InstanceID })
}

// GetAliyunPrivateIpv4 returns the private IPv4 address.
func GetAliyunPrivateIpv4() (string, error) {
	return field(parseAliyunInfo, func(i AliyunIdentity) string { return i.PrivateIpv4 })
}

// GetAliyunMac returns the MAC address of the primary network interface.
func GetAliyunMac() (string, error) {
	return field(parseAliyunInfo, func(i AliyunIdentity) string { return i.Mac })
}

// GetAliyunSerialNumber returns the instance serial number.
func GetAliyunSerialNumber() (string, error) {
	return field(parseAliyunInfo, func(i AliyunIdentity) string { return i.SerialNumber })
}

// aliyunIdentity fetches and normalizes the Aliyun identity for the generic API.
func aliyunIdentity(ctx context.Context) (Identity, error) {
	info, err := parseAliyunInfoContext(ctx)
	if err != nil {
		return Identity{}, err
	}
	return Identity{
		Provider:    ALIYUN_CLOUD_TYPE,
		InstanceID:  info.InstanceID,
		Region:      info.RegionID,
		Zone:        info.ZoneID,
		PrivateIPv4: info.PrivateIpv4,
		Mac:         info.Mac,
	}, nil
}
