# go-cloud-id

[![Go Reference](https://pkg.go.dev/badge/github.com/soulteary/go-cloud-id.svg)](https://pkg.go.dev/github.com/soulteary/go-cloud-id)
[![Go Report Card](.github/goreportcard.svg)](.github/goreportcard-report.md)
[![Coverage](.github/coverage.svg)](.github/go-test-report.md)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[English](README.md)

一个轻量的 Go 库，用于从云厂商元数据服务读取云实例的身份信息。既提供各厂商专属的辅助函数，也提供可自动检测当前环境的通用接口。

## 特性

- **多云支持**: 阿里云（Aliyun）、腾讯云（QCloud）、华为云（Huawei Cloud）与亚马逊云（AWS）
- **自动检测**: `Detect()` 现在并发探测各厂商并返回归一化的身份信息，显著降低非云环境下的检测延迟；并支持 context 取消
- **归一化身份**: 跨厂商统一的 `Identity` 结构（实例 ID、地域、可用区、内网 IPv4、MAC）
- **内置缓存**: 并发安全、TTL 可配置的缓存，避免频繁请求元数据端点
- **安全的 HTTP**: 单次请求超时与响应体大小上限，防止异常超大响应
- **零依赖**: 仅使用标准库

## 支持的云

- [阿里云 / Aliyun](https://www.alibabacloud.com/help/en/ecs/user-guide/use-instance-identities) — 实例身份文档
- [腾讯云 / QCloud](https://www.tencentcloud.com/document/product/213/4934) — 实例元数据
- [华为云 / Huawei Cloud](https://support.huaweicloud.com/eu/usermanual-ecs/ecs_03_0166.html) — 实例元数据（OpenStack 元数据 + EC2 兼容路径）
- [亚马逊云 / AWS](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html) — EC2 实例身份文档（IMDSv2，并兼容 IMDSv1 兜底）

> 元数据服务只能在运行中的云实例内部访问。

## 安装

```bash
go get github.com/soulteary/go-cloud-id
```

## 快速开始

### 自动检测当前云环境

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

### 厂商专属辅助函数

```go
// 阿里云
instanceID, err := cloudid.GetAliyunInstanceID()
zone, err := cloudid.GetAliyunZoneID()
identity, err := cloudid.GetAliyunIdentity() // 完整文档

// 腾讯云
instanceID, err := cloudid.GetTencentInstanceID()
region, err := cloudid.GetTencentRegion()
identity, err := cloudid.GetTencentIdentity() // 完整文档

// 华为云
instanceID, err := cloudid.GetHuaweiInstanceID()
region, err := cloudid.GetHuaweiRegion()
identity, err := cloudid.GetHuaweiIdentity() // 完整文档

// 亚马逊云（AWS）
instanceID, err := cloudid.GetAWSInstanceID()
region, err := cloudid.GetAWSRegion()
identity, err := cloudid.GetAWSIdentity() // 完整文档
```

### 查询指定厂商

```go
id, err := cloudid.GetIdentity(cloudid.ALIYUN_CLOUD_TYPE)
```

## API 参考

### 通用接口

```go
// 跨厂商归一化的身份信息。
type Identity struct {
    Provider    string // "aliyun" | "tencent" | "huawei" | "aws"
    InstanceID  string
    Region      string
    Zone        string
    PrivateIPv4 string
    Mac         string
}

func Detect() (Identity, error)            // 首个成功响应的厂商生效
func DetectContext(ctx context.Context) (Identity, error) // 带 context 的 Detect，可提前取消；Detect() 等价于 DetectContext(context.Background())
func DetectProvider() (string, error)       // 检测到的云类型
func GetIdentity(provider string) (Identity, error)
func GetIdentityContext(ctx context.Context, provider string) (Identity, error) // 带 context 的 GetIdentity

const ALIYUN_CLOUD_TYPE  = "aliyun"
const TENCENT_CLOUD_TYPE = "tencent"
const HUAWEI_CLOUD_TYPE  = "huawei"
const AWS_CLOUD_TYPE     = "aws"

var ErrNotDetected error        // 未检测到受支持的云
var ErrMetadataUnavailable error // 元数据服务因网络故障/超时/5xx 而不可用时返回（区别于 ErrNotDetected 的“未检测到受支持的云”）；可用 errors.Is 判定
```

### 阿里云（Aliyun）

```go
func GetAliyunInfo() ([]byte, error)                 // 原始身份文档
func SerializeAliyunInfo([]byte) (AliyunIdentity, error)
func GetAliyunIdentity() (AliyunIdentity, error)
func GetAliyunInstanceID() (string, error)
func GetAliyunRegionID() (string, error)
func GetAliyunZoneID() (string, error)
func GetAliyunPrivateIpv4() (string, error)
func GetAliyunMac() (string, error)
func GetAliyunSerialNumber() (string, error)
```

### 腾讯云（QCloud）

```go
func GetTencentInfo() ([]byte, error)                // 元数据组装为 JSON
func SerializeTencentInfo([]byte) (TencentIdentity, error)
func GetTencentIdentity() (TencentIdentity, error)
func GetTencentInstanceID() (string, error)
func GetTencentRegion() (string, error)
func GetTencentZone() (string, error)
func GetTencentPrivateIpv4() (string, error)
func GetTencentMac() (string, error)
func GetTencentUUID() (string, error)
```

### 华为云（Huawei Cloud）

```go
func GetHuaweiInfo() ([]byte, error)                // 元数据组装为 JSON
func SerializeHuaweiInfo([]byte) (HuaweiIdentity, error)
func GetHuaweiIdentity() (HuaweiIdentity, error)
func GetHuaweiInstanceID() (string, error)
func GetHuaweiRegion() (string, error)
func GetHuaweiZone() (string, error)
func GetHuaweiPrivateIpv4() (string, error)
func GetHuaweiProjectID() (string, error)
```

### 亚马逊云（AWS）

```go
func GetAWSInfo() ([]byte, error)                  // 元数据组装为 JSON
func SerializeAWSInfo([]byte) (AWSIdentity, error)
func GetAWSIdentity() (AWSIdentity, error)
func GetAWSInstanceID() (string, error)
func GetAWSRegion() (string, error)
func GetAWSZone() (string, error)
func GetAWSPrivateIpv4() (string, error)
func GetAWSMac() (string, error)
func GetAWSAccountID() (string, error)
```

### 缓存控制

```go
func SetCacheTTL(ttl time.Duration) // 默认 10 分钟；非正值重置为默认
func ClearCache()                   // 清空所有缓存文档
```

## 工作原理

- **阿里云**在 `http://100.100.100.200/latest/dynamic/instance-identity/document`
  提供单个 JSON 身份文档，库会拉取、缓存并解析它。
- **腾讯云**在 `http://metadata.tencentyun.com/latest/meta-data/` 下以独立字段
  暴露元数据；库会拉取各字段、组装为 JSON 文档并缓存结果。若缺少
  `instance-id`，则视为“非腾讯云实例”。
- **华为云**在 `http://169.254.169.254/openstack/latest/meta_data.json` 提供
  OpenStack 风格文档（实例 ID、可用区、地域、项目、以及 `meta` 中的 VPC/镜像
  信息）；内网 IPv4 则从 EC2 兼容路径 `/latest/meta-data/local-ipv4` 读取。库会
  将其组装为 JSON 文档并缓存，若 `uuid` 缺失或为空则视为“非华为云实例”。当
  `region_id` 缺失时，会尽力从可用区推导出地域作为兜底。
- **亚马逊云（AWS）**在
  `http://169.254.169.254/latest/dynamic/instance-identity/document` 提供 JSON
  实例身份文档（实例 ID、地域、可用区、内网 IPv4、账号、镜像）。较新的实例默认
  启用 IMDSv2：库会先通过 `PUT /latest/api/token` 获取短期会话令牌并在后续请求中
  携带，当令牌端点不可用时自动回退为无令牌的 IMDSv1 请求。MAC 从
  `/latest/meta-data/mac` 读取。若 `instanceId` 缺失或为空则视为“非 AWS 实例”。
- 成功的响应会按配置的 TTL（默认 10 分钟）缓存，避免重复请求元数据。best-effort
  字段抓取失败时不会缓存残缺结果，下次调用会重试。

## 许可证

Apache License 2.0
