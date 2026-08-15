package cloudid

// Detect probes the supported cloud metadata services and returns a normalized
// Identity for the current environment. Providers are probed in order; the
// first one that responds successfully wins. If none respond, ErrNotDetected
// is returned.
func Detect() (Identity, error) {
	if id, err := aliyunIdentity(); err == nil {
		return id, nil
	}
	if id, err := tencentIdentity(); err == nil {
		return id, nil
	}
	return Identity{}, ErrNotDetected
}

// DetectProvider returns the cloud type of the current environment, one of the
// *_CLOUD_TYPE constants, or an empty string with ErrNotDetected if unknown.
func DetectProvider() (string, error) {
	id, err := Detect()
	if err != nil {
		return "", err
	}
	return id.Provider, nil
}

// GetIdentity returns the normalized Identity for a specific provider.
// provider must be one of the *_CLOUD_TYPE constants.
func GetIdentity(provider string) (Identity, error) {
	switch provider {
	case ALIYUN_CLOUD_TYPE:
		return aliyunIdentity()
	case TENCENT_CLOUD_TYPE:
		return tencentIdentity()
	default:
		return Identity{}, ErrNotDetected
	}
}
