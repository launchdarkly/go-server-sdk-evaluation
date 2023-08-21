package ldmodel

import "github.com/launchdarkly/go-sdk-common/v3/ldvalue"

// Metric contains metric configuration values that apply across an environment.
//
// These overrides can be used to control things like the rate at which custom
// metrics are sampled.
type Metric struct {
	// Key is the unique string key of the metric.
	Key string
	// SamplingRatio controls the 1 in x chances a metric value will be included in the events sent to LaunchDarkly.
	SamplingRatio ldvalue.OptionalInt
	// Version is an integer that is incremented by LaunchDarkly every time the configuration of the flag is
	// changed.
	Version int
	// Deleted is true if this is not actually a metric override but rather a placeholder (tombstone) for a
	// deleted override. This is only relevant in data store implementations.
	Deleted bool
}
