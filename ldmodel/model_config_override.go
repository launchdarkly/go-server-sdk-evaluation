package ldmodel

import "github.com/launchdarkly/go-sdk-common/v3/ldvalue"

// ConfigOverride contains configuration overrides that apply across environment.
//
// Configuration overrides are used to modify the runtime behavior of an SDK by
// modifying the upstream data source.
type ConfigOverride struct {
	// Key is the unique string key of the override.
	Key string
	// Value contains the configuration override value.
	Value ldvalue.Value
	// Version is an integer that is incremented by LaunchDarkly every time the configuration of the flag is
	// changed.
	Version int
	// Deleted is true if this is not actually a config override but rather a placeholder (tombstone) for a
	// deleted override. This is only relevant in data store implementations.
	Deleted bool
}
