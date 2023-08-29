package ldbuilders

import (
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	"github.com/launchdarkly/go-server-sdk-evaluation/v3/ldmodel"
)

// ConfigOverrideBuilder provides a builder pattern for ConfigOverride.
type ConfigOverrideBuilder struct {
	override ldmodel.ConfigOverride
}

// NewConfigOverrideBuilder creates a ConfigOverrideBuilder.
func NewConfigOverrideBuilder(key string) *ConfigOverrideBuilder {
	return &ConfigOverrideBuilder{ldmodel.ConfigOverride{Key: key}}
}

// Value sets the override's configured value.
func (b *ConfigOverrideBuilder) Value(value ldvalue.Value) *ConfigOverrideBuilder {
	b.override.Value = value
	return b
}

// Version sets the segment's Version property.
func (b *ConfigOverrideBuilder) Version(value int) *ConfigOverrideBuilder {
	b.override.Version = value
	return b
}

// Build returns the configured ConfigOverride.
func (b *ConfigOverrideBuilder) Build() ldmodel.ConfigOverride {
	return b.override
}
