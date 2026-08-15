package cloudid

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Amazon Web Services (AWS) EC2 instance identity.
// Documentation: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
//
// AWS exposes a JSON instance identity document at
// /latest/dynamic/instance-identity/document. Modern instances default to
// IMDSv2, which requires a short-lived session token obtained via a PUT to
// /latest/api/token; the token is then sent on subsequent requests via the
// X-aws-ec2-metadata-token header. The MAC address is not part of the identity
// document and is read from the EC2 metadata path instead.

// awsMetadataBase is the metadata service root. It is a variable so tests can
// point it at a local server.
var awsMetadataBase = "http://169.254.169.254"

// AWS metadata sub-paths.
const (
	awsPathToken            = "/latest/api/token"
	awsPathIdentityDocument = "/latest/dynamic/instance-identity/document"
	awsPathMac              = "/latest/meta-data/mac"
)

// awsTokenTTLSeconds is the requested lifetime of the IMDSv2 session token.
const awsTokenTTLSeconds = "60"

// IMDSv2 token request/response header names.
const (
	awsHeaderTokenTTL = "X-aws-ec2-metadata-token-ttl-seconds"
	awsHeaderToken    = "X-aws-ec2-metadata-token"
)

// awsIdentityDocument mirrors the fields of the EC2 instance identity document
// that we care about.
type awsIdentityDocument struct {
	InstanceID       string `json:"instanceId"`
	Region           string `json:"region"`
	AvailabilityZone string `json:"availabilityZone"`
	PrivateIP        string `json:"privateIp"`
	AccountID        string `json:"accountId"`
	ImageID          string `json:"imageId"`
	InstanceType     string `json:"instanceType"`
	Architecture     string `json:"architecture"`
}

// AWSIdentity holds the normalized AWS EC2 instance identity fields.
type AWSIdentity struct {
	InstanceID   string `json:"instance-id"`
	Region       string `json:"region"`
	Zone         string `json:"zone"`
	PrivateIpv4  string `json:"private-ipv4"`
	Mac          string `json:"mac"`
	AccountID    string `json:"account-id"`
	ImageID      string `json:"image-id"`
	InstanceType string `json:"instance-type"`
}

// awsToken obtains an IMDSv2 session token. On IMDSv1-only instances the PUT
// endpoint may be unavailable; callers treat an error here as "no token" and
// fall back to unauthenticated requests.
func awsToken() (string, error) {
	body, err := put(awsMetadataBase+awsPathToken, withHeader(awsHeaderTokenTTL, awsTokenTTLSeconds))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// awsGet fetches a metadata path, attaching the IMDSv2 token header when a
// non-empty token is provided.
func awsGet(path, token string) ([]byte, error) {
	if token != "" {
		return get(awsMetadataBase+path, withHeader(awsHeaderToken, token))
	}
	return get(awsMetadataBase + path)
}

// GetAWSInfo returns the AWS metadata assembled as a normalized JSON document,
// using the cache when fresh.
func GetAWSInfo() ([]byte, error) {
	if data, ok := defaultCache.get(AWS_CLOUD_TYPE); ok {
		return data, nil
	}

	info, err := fetchAWSIdentity()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshal aws info failed: %w", err)
	}

	defaultCache.set(AWS_CLOUD_TYPE, data)
	return data, nil
}

// SerializeAWSInfo parses a raw AWS identity JSON document.
func SerializeAWSInfo(data []byte) (info AWSIdentity, err error) {
	if err = json.Unmarshal(data, &info); err != nil {
		return info, err
	}
	return info, nil
}

// fetchAWSIdentity queries the instance identity document (using an IMDSv2
// token when available) and normalizes it into an AWSIdentity.
func fetchAWSIdentity() (AWSIdentity, error) {
	var info AWSIdentity

	// A token is best-effort: IMDSv2 requires it, IMDSv1 ignores it.
	token, _ := awsToken()

	// The identity document is mandatory: its absence (or an unparsable body)
	// means this is not an AWS EC2 instance.
	body, err := awsGet(awsPathIdentityDocument, token)
	if err != nil {
		return info, fmt.Errorf("getting aws info failed: %w", err)
	}

	var doc awsIdentityDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return info, fmt.Errorf("serialize aws identity document failed: %w", err)
	}
	if doc.InstanceID == "" {
		return info, fmt.Errorf("getting aws info failed: empty instance id")
	}

	info.InstanceID = doc.InstanceID
	info.Region = doc.Region
	info.Zone = doc.AvailabilityZone
	info.PrivateIpv4 = doc.PrivateIP
	info.AccountID = doc.AccountID
	info.ImageID = doc.ImageID
	info.InstanceType = doc.InstanceType

	// MAC is not part of the identity document; fetch it best-effort.
	if mac, err := awsGet(awsPathMac, token); err == nil {
		info.Mac = strings.TrimSpace(string(mac))
	}

	return info, nil
}

func parseAWSInfo() (AWSIdentity, error) {
	data, err := GetAWSInfo()
	if err != nil {
		return AWSIdentity{}, err
	}

	parsed, err := SerializeAWSInfo(data)
	if err != nil {
		return AWSIdentity{}, fmt.Errorf("serialize aws info failed: %w", err)
	}
	return parsed, nil
}

// GetAWSIdentity returns the full parsed AWS identity.
func GetAWSIdentity() (AWSIdentity, error) {
	return parseAWSInfo()
}

// GetAWSInstanceID returns the instance ID.
func GetAWSInstanceID() (string, error) {
	info, err := parseAWSInfo()
	if err != nil {
		return "", err
	}
	return info.InstanceID, nil
}

// GetAWSRegion returns the instance region.
func GetAWSRegion() (string, error) {
	info, err := parseAWSInfo()
	if err != nil {
		return "", err
	}
	return info.Region, nil
}

// GetAWSZone returns the instance availability zone.
func GetAWSZone() (string, error) {
	info, err := parseAWSInfo()
	if err != nil {
		return "", err
	}
	return info.Zone, nil
}

// GetAWSPrivateIpv4 returns the private IPv4 address.
func GetAWSPrivateIpv4() (string, error) {
	info, err := parseAWSInfo()
	if err != nil {
		return "", err
	}
	return info.PrivateIpv4, nil
}

// GetAWSMac returns the MAC address of the primary network interface.
func GetAWSMac() (string, error) {
	info, err := parseAWSInfo()
	if err != nil {
		return "", err
	}
	return info.Mac, nil
}

// GetAWSAccountID returns the owner account ID.
func GetAWSAccountID() (string, error) {
	info, err := parseAWSInfo()
	if err != nil {
		return "", err
	}
	return info.AccountID, nil
}

// awsIdentity fetches and normalizes the AWS identity for the generic API.
func awsIdentity() (Identity, error) {
	info, err := parseAWSInfo()
	if err != nil {
		return Identity{}, err
	}
	return Identity{
		Provider:    AWS_CLOUD_TYPE,
		InstanceID:  info.InstanceID,
		Region:      info.Region,
		Zone:        info.Zone,
		PrivateIPv4: info.PrivateIpv4,
		Mac:         info.Mac,
	}, nil
}
