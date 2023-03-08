package cloudid

import (
	"encoding/json"
	"fmt"
)

// ALibaba Cloud Documentation
// https://www.alibabacloud.com/help/en/elastic-compute-service/latest/use-instance-identities
// Updated at: 2023-01-06 13:13

const ALIYUN_CLOUD_TYPE = "aliyun"

const ALIYUN_GET_INSTANCE_INDENTITY = `http://100.100.100.200/latest/dynamic/instance-identity/document`

type ALIYUN_INDENTITY struct {
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

func GetAliyunInfo() ([]byte, error) {
	data := getCache(ALIYUN_CLOUD_TYPE)
	if data == nil {
		remote, err := get(ALIYUN_GET_INSTANCE_INDENTITY)
		if err == nil {
			updateCache(ALIYUN_CLOUD_TYPE, remote)
			return remote, nil
		}
		return nil, err
	}

	expired := isExpired(ALIYUN_CLOUD_TYPE)
	if expired {
		remote, err := get(ALIYUN_GET_INSTANCE_INDENTITY)
		if err == nil {
			addExpire(ALIYUN_CLOUD_TYPE)
			updateCache(ALIYUN_CLOUD_TYPE, remote)
			return remote, nil
		}
		return nil, err
	}
	return data, nil
}

func SerializeAliyunInfo(data []byte) (info ALIYUN_INDENTITY, err error) {
	err = json.Unmarshal(data, &info)
	if err != nil {
		return info, err
	}
	return info, nil
}

func parseAliyunInfo() (info ALIYUN_INDENTITY, err error) {
	data, err := GetAliyunInfo()
	if err != nil {
		return info, fmt.Errorf("getting aliyun info failed: %v", err)
	}

	parsed, err := SerializeAliyunInfo(data)
	if err != nil {
		return info, fmt.Errorf("serialize aliyun info failed: %v", err)
	}
	return parsed, nil
}

func GetAliyunZoneID() (string, error) {
	info, err := parseAliyunInfo()
	if err != nil {
		return "", err
	}
	return info.ZoneID, nil
}

func GetAliyunInstanceID() (string, error) {
	info, err := parseAliyunInfo()
	if err != nil {
		return "", err
	}
	return info.InstanceID, nil
}

func GetAliyunPrivateIpv4() (string, error) {
	info, err := parseAliyunInfo()
	if err != nil {
		return "", err
	}
	return info.PrivateIpv4, nil
}

func GetAliyunMac() (string, error) {
	info, err := parseAliyunInfo()
	if err != nil {
		return "", err
	}
	return info.Mac, nil
}

func GetAliyunSerialNumber() (string, error) {
	info, err := parseAliyunInfo()
	if err != nil {
		return "", err
	}
	return info.SerialNumber, nil
}
