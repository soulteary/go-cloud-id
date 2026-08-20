package cloudid

import "context"

// providerProbe is a context-aware identity fetcher for a single provider. The
// winning provider is read from the returned Identity.Provider, so the probe
// itself carries no separate name.
type providerProbe func(context.Context) (Identity, error)

// providerProbes lists the supported providers probed by Detect/DetectContext,
// in priority order (earlier entries win when several providers respond).
var providerProbes = []providerProbe{
	aliyunIdentity,
	tencentIdentity,
	huaweiIdentity,
	awsIdentity,
}

// Detect probes the supported cloud metadata services concurrently and returns
// a normalized Identity for the current environment. All probes run in
// parallel, but the winner is chosen by provider priority (the earliest probe
// in providerProbes that responds successfully), so the result is
// deterministic regardless of which probe returns first. Remaining probes are
// cancelled. If none respond, ErrNotDetected is returned.
func Detect() (Identity, error) {
	return DetectContext(context.Background())
}

// DetectContext behaves like Detect but honors the supplied context, allowing
// callers to bound the total probe time or cancel it early. Probes run
// concurrently (latency stays close to a single probe) while the winner is
// selected deterministically by provider priority.
func DetectContext(ctx context.Context) (Identity, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Each probe writes to its own buffered channel so a slow or cancelled
	// probe never blocks and no goroutine outlives this call.
	results := make([]chan Identity, len(providerProbes))
	for i, probe := range providerProbes {
		results[i] = make(chan Identity, 1)
		go func(ch chan Identity, probe providerProbe) {
			if id, err := probe(ctx); err == nil {
				ch <- id
			} else {
				ch <- Identity{}
			}
		}(results[i], probe)
	}

	// Select the winner by priority: walk the probes in order and take the
	// first one that reports a provider. Once found we cancel the rest, then
	// drain the remaining channels so every probe goroutine terminates.
	var winner Identity
	for i := range results {
		id := <-results[i]
		if winner.Provider == "" && id.Provider != "" {
			winner = id
			cancel()
		}
	}
	if winner.Provider != "" {
		return winner, nil
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
//
// Unlike Detect, this is an explicit single-provider lookup: transport or
// server-side failures are returned wrapped in ErrMetadataUnavailable so
// callers can distinguish an outage from a "not this cloud" outcome.
func GetIdentity(provider string) (Identity, error) {
	return GetIdentityContext(context.Background(), provider)
}

// GetIdentityContext behaves like GetIdentity but honors the supplied context.
func GetIdentityContext(ctx context.Context, provider string) (Identity, error) {
	switch provider {
	case ALIYUN_CLOUD_TYPE:
		return aliyunIdentity(ctx)
	case TENCENT_CLOUD_TYPE:
		return tencentIdentity(ctx)
	case HUAWEI_CLOUD_TYPE:
		return huaweiIdentity(ctx)
	case AWS_CLOUD_TYPE:
		return awsIdentity(ctx)
	default:
		return Identity{}, ErrNotDetected
	}
}
