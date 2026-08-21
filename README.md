# go-cloud-id

[![Go Reference](https://pkg.go.dev/badge/github.com/soulteary/go-cloud-id.svg)](https://pkg.go.dev/github.com/soulteary/go-cloud-id)
[![Go Report Card](.github/goreportcard.svg)](.github/goreportcard-report.md)
[![Coverage](.github/coverage.svg)](.github/go-test-report.md)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![codecov](https://codecov.io/gh/soulteary/go-cloud-id/graph/badge.svg)](https://codecov.io/gh/soulteary/go-cloud-id)

[中文文档](README_CN.md)

A small Go library that reads cloud instance identity information from the cloud
provider's metadata service. It ships both provider-specific helpers and a
provider-agnostic API that auto-detects the current environment.

## Features

- **Multi-Cloud**: Alibaba Cloud (Aliyun), Tencent Cloud (QCloud), Huawei Cloud, and Amazon Web Services (AWS)
- **Auto-Detection**: `Detect()` probes providers concurrently and returns a normalized identity, significantly reducing detection latency in non-cloud environments; it also supports context cancellation
- **Normalized Identity**: One `Identity` struct across providers (instance ID, region, zone, private IPv4, MAC)
- **Built-in Cache**: Concurrency-safe cache with configurable TTL avoids hammering the metadata endpoint
- **Safe HTTP**: Per-request timeout and a response size cap to guard against oversized responses
- **Zero Dependencies**: Standard library only

## Supported Clouds

- [Alibaba Cloud / Aliyun](https://www.alibabacloud.com/help/en/ecs/user-guide/use-instance-identities) — instance identity document
- [Tencent Cloud / QCloud](https://www.tencentcloud.com/document/product/213/4934) — instance metadata
- [Huawei Cloud](https://support.huaweicloud.com/eu/usermanual-ecs/ecs_03_0166.html) — instance metadata (OpenStack metadata + EC2-compatible paths)
- [Amazon Web Services / AWS](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html) — EC2 instance identity document (IMDSv2, with IMDSv1 fallback)

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

// Huawei Cloud
instanceID, err := cloudid.GetHuaweiInstanceID()
region, err := cloudid.GetHuaweiRegion()
identity, err := cloudid.GetHuaweiIdentity() // full document

// Amazon Web Services (AWS)
instanceID, err := cloudid.GetAWSInstanceID()
region, err := cloudid.GetAWSRegion()
identity, err := cloudid.GetAWSIdentity() // full document
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
    Provider    string // "aliyun" | "tencent" | "huawei" | "aws"
    InstanceID  string
    Region      string
    Zone        string
    PrivateIPv4 string
    Mac         string
}

func Detect() (Identity, error)            // first responding provider wins
func DetectContext(ctx context.Context) (Identity, error) // Detect with a context; can be cancelled early. Detect() is equivalent to DetectContext(context.Background())
func DetectProvider() (string, error)       // detected cloud type
func GetIdentity(provider string) (Identity, error)
func GetIdentityContext(ctx context.Context, provider string) (Identity, error) // GetIdentity with a context

const ALIYUN_CLOUD_TYPE  = "aliyun"
const TENCENT_CLOUD_TYPE = "tencent"
const HUAWEI_CLOUD_TYPE  = "huawei"
const AWS_CLOUD_TYPE     = "aws"

var ErrNotDetected error        // no supported cloud detected
var ErrMetadataUnavailable error // returned when the metadata service fails due to network error/timeout/5xx (distinct from ErrNotDetected's "no supported cloud detected"); check with errors.Is
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

### Huawei Cloud

```go
func GetHuaweiInfo() ([]byte, error)                // metadata assembled as JSON
func SerializeHuaweiInfo([]byte) (HuaweiIdentity, error)
func GetHuaweiIdentity() (HuaweiIdentity, error)
func GetHuaweiInstanceID() (string, error)
func GetHuaweiRegion() (string, error)
func GetHuaweiZone() (string, error)
func GetHuaweiPrivateIpv4() (string, error)
func GetHuaweiProjectID() (string, error)
```

### Amazon Web Services (AWS)

```go
func GetAWSInfo() ([]byte, error)                  // metadata assembled as JSON
func SerializeAWSInfo([]byte) (AWSIdentity, error)
func GetAWSIdentity() (AWSIdentity, error)
func GetAWSInstanceID() (string, error)
func GetAWSRegion() (string, error)
func GetAWSZone() (string, error)
func GetAWSPrivateIpv4() (string, error)
func GetAWSMac() (string, error)
func GetAWSAccountID() (string, error)
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
- **Huawei** exposes an OpenStack-style document at
  `http://169.254.169.254/openstack/latest/meta_data.json` (instance id, zone,
  region, project, VPC/image via `meta`). The private IPv4 is read from the
  EC2-compatible path `/latest/meta-data/local-ipv4`. The library assembles
  these into a JSON document and caches the result; a missing/empty `uuid` is
  treated as "not a Huawei instance". If `region_id` is absent it is derived
  from the availability zone as a best-effort fallback.
- **AWS** exposes a JSON instance identity document at
  `http://169.254.169.254/latest/dynamic/instance-identity/document` (instance
  id, region, availability zone, private IPv4, account, image). Modern
  instances default to IMDSv2: the library first requests a short-lived session
  token via `PUT /latest/api/token` and attaches it on subsequent reads,
  transparently falling back to token-less IMDSv1 requests when the token
  endpoint is unavailable. The MAC is read from `/latest/meta-data/mac`. A
  missing/empty `instanceId` is treated as "not an AWS instance".
- Successful responses are cached for the configured TTL (default 10 minutes)
  to avoid repeated metadata calls. Best-effort field fetches that fail do not
  cache a partial result, so the next call will retry.

## License

Apache License 2.0
