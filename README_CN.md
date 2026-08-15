# go-cloud-id

[![Go Reference](https://pkg.go.dev/badge/github.com/soulteary/go-cloud-id.svg)](https://pkg.go.dev/github.com/soulteary/go-cloud-id)
[![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/go-cloud-id)](https://goreportcard.com/report/github.com/soulteary/go-cloud-id)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![codecov](https://codecov.io/gh/soulteary/go-cloud-id/graph/badge.svg)](https://codecov.io/gh/soulteary/go-cloud-id)

[English](README.md)

一个轻量的 Go 库，用于从云厂商元数据服务读取云实例的身份信息。既提供各厂商专属的辅助函数，也提供可自动检测当前环境的通用接口。

## 特性

- **多云支持**: 阿里云（Aliyun）与腾讯云（QCloud）
- **自动检测**: `Detect()` 逐个探测厂商并返回归一化的身份信息
- **归一化身份**: 跨厂商统一的 `Identity` 结构（实例 ID、地域、可用区、内网 IPv4、MAC）
- **内置缓存**: 并发安全、TTL 可配置的缓存，避免频繁请求元数据端点
- **安全的 HTTP**: 单次请求超时与响应体大小上限，防止异常超大响应
- **零依赖**: 仅使用标准库

## 支持的云

- [阿里云 / Aliyun](https://www.alibabacloud.com/help/en/ecs/user-guide/use-instance-identities) — 实例身份文档
- [腾讯云 / QCloud](https://www.tencentcloud.com/document/product/213/4934) — 实例元数据

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
    Provider    string // "aliyun" | "tencent"
    InstanceID  string
    Region      string
    Zone        string
    PrivateIPv4 string
    Mac         string
}

func Detect() (Identity, error)            // 首个成功响应的厂商生效
func DetectProvider() (string, error)       // 检测到的云类型
func GetIdentity(provider string) (Identity, error)

const ALIYUN_CLOUD_TYPE  = "aliyun"
const TENCENT_CLOUD_TYPE = "tencent"

var ErrNotDetected error // 未检测到受支持的云
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
- 成功的响应会按配置的 TTL（默认 10 分钟）缓存，避免重复请求元数据。

## 许可证

Apache License 2.0
