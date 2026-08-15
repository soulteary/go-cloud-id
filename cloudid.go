// Package cloudid retrieves cloud instance identity information from the
// metadata service of the underlying cloud provider.
//
// It currently supports Alibaba Cloud (Aliyun) and Tencent Cloud (QCloud),
// exposing both provider-specific helpers and a provider-agnostic API that
// auto-detects the current environment.
//
// Example:
//
//	id, err := cloudid.Detect()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(id.Provider, id.InstanceID, id.Region)
package cloudid

import (
	"errors"
	"time"
)

// Cloud provider type identifiers.
const (
	ALIYUN_CLOUD_TYPE  = "aliyun"
	TENCENT_CLOUD_TYPE = "tencent"
)

// ErrNotDetected is returned when no supported cloud environment can be found.
var ErrNotDetected = errors.New("cloudid: no supported cloud environment detected")

// Identity is a provider-agnostic view of an instance's identity, normalized
// across supported clouds. Fields that a provider does not expose are empty.
type Identity struct {
	// Provider is the cloud type, e.g. "aliyun" or "tencent".
	Provider string `json:"provider"`
	// InstanceID uniquely identifies the instance.
	InstanceID string `json:"instance_id"`
	// Region is the region identifier.
	Region string `json:"region"`
	// Zone is the availability zone identifier.
	Zone string `json:"zone"`
	// PrivateIPv4 is the primary private IPv4 address.
	PrivateIPv4 string `json:"private_ipv4"`
	// Mac is the MAC address of the primary network interface.
	Mac string `json:"mac"`
}

// SetCacheTTL overrides the freshness window for cached metadata documents on
// the package-level cache. A non-positive value resets it to the default.
func SetCacheTTL(ttl time.Duration) {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	defaultCache.mu.Lock()
	defaultCache.ttl = ttl
	defaultCache.mu.Unlock()
}

// ClearCache removes all cached metadata documents.
func ClearCache() {
	defaultCache.clear()
}
