// Package cloudid retrieves cloud instance identity information from the
// metadata service of the underlying cloud provider.
//
// It currently supports Alibaba Cloud (Aliyun), Tencent Cloud (QCloud), Huawei
// Cloud, and Amazon Web Services (AWS), exposing both provider-specific helpers
// and a provider-agnostic API that auto-detects the current environment.
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
	HUAWEI_CLOUD_TYPE  = "huawei"
	AWS_CLOUD_TYPE     = "aws"
)

// ErrNotDetected is returned when no supported cloud environment can be found.
var ErrNotDetected = errors.New("cloudid: no supported cloud environment detected")

// ErrMetadataUnavailable wraps failures where a metadata service could not be
// reached or returned a server-side error (transport failure, timeout, or 5xx
// status). It is distinct from a "not this cloud" outcome (4xx / 404), letting
// callers use errors.Is to tell an outage apart from a wrong-provider probe.
var ErrMetadataUnavailable = errors.New("cloudid: metadata service unavailable")

// Identity is a provider-agnostic view of an instance's identity, normalized
// across supported clouds. Fields that a provider does not expose are empty.
type Identity struct {
	// Provider is the cloud type, e.g. "aliyun", "tencent", "huawei", or "aws".
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

// field parses an identity via parse and projects a single string field out of
// it via pick. It centralizes the "parse then read one field" boilerplate that
// every single-field getter shares, so provider getters need only supply the
// projection.
func field[T any](parse func() (T, error), pick func(T) string) (string, error) {
	v, err := parse()
	if err != nil {
		return "", err
	}
	return pick(v), nil
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
