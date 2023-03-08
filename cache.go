package gocloudid

import (
	"time"
)

var aliyunExpireTime time.Time
var aliyunCache []byte

var tencentCloudExpireTime time.Time

const cacheTime = "10m"

func updateCache(cloud string, data []byte) {
	switch cloud {
	case ALIYUN_CLOUD_TYPE:
		aliyunCache = data
		return
	}
}

func getCache(cloud string) []byte {
	switch cloud {
	case ALIYUN_CLOUD_TYPE:
		return aliyunCache
	}
	return nil
}

func addExpire(cloud string) {
	tenMinLater, _ := time.ParseDuration(cacheTime)
	expire := time.Now().Add(tenMinLater)

	switch cloud {
	case ALIYUN_CLOUD_TYPE:
		aliyunExpireTime = expire
		return
	case TENCENT_CLOUD_TYPE:
		tencentCloudExpireTime = expire
		return
	}
}

func isExpired(cloud string) bool {
	now := time.Now()

	switch cloud {
	case ALIYUN_CLOUD_TYPE:
		return aliyunExpireTime.After(now)
	case TENCENT_CLOUD_TYPE:
		return tencentCloudExpireTime.After(now)
	}
	return false
}
