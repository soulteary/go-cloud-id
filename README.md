# go-cloud-id

[![Go Reference](https://pkg.go.dev/badge/github.com/soulteary/go-cloud-id.svg)](https://pkg.go.dev/github.com/soulteary/go-cloud-id)
[![Go Report Card](.github/goreportcard.svg)](.github/goreportcard-report.md)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![codecov](https://codecov.io/gh/soulteary/go-cloud-id/graph/badge.svg)](https://codecov.io/gh/soulteary/go-cloud-id)

[中文文档](README_CN.md)

A small Go library that reads cloud instance identity information from the cloud
provider's metadata service. It ships both provider-specific helpers and a
provider-agnostic API that auto-detects the current environment.

## Features

- **Multi-Cloud**: Alibaba Cloud (Aliyun) and Tencent Cloud (QCloud)
- **Auto-Detection**: `Detect()` probes providers and returns a normalized identity
- **Normalized Identity**: One `Identity` struct across providers (instance ID, region, zone, private IPv4, MAC)
- **Built-in Cache**: Concurrency-safe cache with configurable TTL avoids hammering the metadata endpoint
- **Safe HTTP**: Per-request timeout and a response size cap to guard against oversized responses
- **Zero Dependencies**: Standard library only

## Supported Clouds

- [Alibaba Cloud / Aliyun](https://www.alibabacloud.com/help/en/ecs/user-guide/use-instance-identities) — instance identity document
- [Tencent Cloud / QCloud](https://www.tencentcloud.com/document/product/213/4934) — instance metadata

> Metadata services are only reachable from within a running cloud instance.

## Installation

```bash
go get github.com/soulteary/go-cloud-id
```

## Quick Start

### Auto-detect the current cloud

```go
package main

import (
    "fmt"
    "log"

    cloudid "github.com/soulteary/go-cloud-id"
)

func main() {
    id, err := cloudid.Detect()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("provider:", id.Provider)
    fmt.Println("instance:", id.InstanceID)
    fmt.Println("region:", id.Region)
    fmt.Println("zone:", id.Zone)
    fmt.Println("private ip:", id.PrivateIPv4)
    fmt.Println("mac:", id.Mac)
}
```

### Provider-specific helpers

```go
// Alibaba Cloud
instanceID, err := cloudid.GetAliyunInstanceID()
zone, err := cloudid.GetAliyunZoneID()
identity, err := cloudid.GetAliyunIdentity() // full document

// Tencent Cloud
instanceID, err := cloudid.GetTencentInstanceID()
region, err := cloudid.GetTencentRegion()
identity, err := cloudid.GetTencentIdentity() // full document
```

### Query a known provider

```go
id, err := cloudid.GetIdentity(cloudid.ALIYUN_CLOUD_TYPE)
```

## API Reference

### Provider-agnostic

```go
// Normalized identity across providers.
type Identity struct {
    Provider    string // "aliyun" | "tencent"
    InstanceID  string
    Region      string
    Zone        string
    PrivateIPv4 string
    Mac         string
}

func Detect() (Identity, error)            // first responding provider wins
func DetectProvider() (string, error)       // detected cloud type
func GetIdentity(provider string) (Identity, error)

const ALIYUN_CLOUD_TYPE  = "aliyun"
const TENCENT_CLOUD_TYPE = "tencent"

var ErrNotDetected error // no supported cloud detected
```

### Alibaba Cloud (Aliyun)

```go
func GetAliyunInfo() ([]byte, error)                 // raw identity document
func SerializeAliyunInfo([]byte) (AliyunIdentity, error)
func GetAliyunIdentity() (AliyunIdentity, error)
func GetAliyunInstanceID() (string, error)
func GetAliyunRegionID() (string, error)
func GetAliyunZoneID() (string, error)
func GetAliyunPrivateIpv4() (string, error)
func GetAliyunMac() (string, error)
func GetAliyunSerialNumber() (string, error)
```

### Tencent Cloud (QCloud)

```go
func GetTencentInfo() ([]byte, error)                // metadata assembled as JSON
func SerializeTencentInfo([]byte) (TencentIdentity, error)
func GetTencentIdentity() (TencentIdentity, error)
func GetTencentInstanceID() (string, error)
func GetTencentRegion() (string, error)
func GetTencentZone() (string, error)
func GetTencentPrivateIpv4() (string, error)
func GetTencentMac() (string, error)
func GetTencentUUID() (string, error)
```

### Cache control

```go
func SetCacheTTL(ttl time.Duration) // default 10m; non-positive resets to default
func ClearCache()                   // drop all cached documents
```

## How It Works

- **Aliyun** exposes a single JSON identity document at
  `http://100.100.100.200/latest/dynamic/instance-identity/document`, which is
  fetched, cached, and parsed.
- **Tencent** exposes individual fields under
  `http://metadata.tencentyun.com/latest/meta-data/`; the library fetches the
  fields, assembles them into a JSON document, and caches the result. A missing
  `instance-id` is treated as "not a Tencent instance".
- Successful responses are cached for the configured TTL (default 10 minutes)
  to avoid repeated metadata calls.

## License

Apache License 2.0
