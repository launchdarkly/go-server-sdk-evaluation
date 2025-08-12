//go:build launchdarkly_intern_json && !go1.23

package intern

// This file intentionally fails compilation when the build tag
// launchdarkly_intern_json is used with a Go version older than 1.23.
// Upgrade to Go 1.23+ to enable interning, or remove the tag to build
// with the no-op interning backend.

// The following undefined symbol produces a clear compile-time error.
var _ go1_23_is_required__enable_launchdarkly_intern_json_on_go1_23_plus_only

