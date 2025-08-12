//go:build !launchdarkly_intern_json && !launchdarkly_intern_json_retain_values

package intern

import "github.com/launchdarkly/go-sdk-common/v3/ldvalue"

// No-op interning.

func String(s string) string           { return s }
func StringSlice(ss []string) []string { return ss }

func Value(v ldvalue.Value) ldvalue.Value           { return v }
func ValueSlice(vs []ldvalue.Value) []ldvalue.Value { return vs }

