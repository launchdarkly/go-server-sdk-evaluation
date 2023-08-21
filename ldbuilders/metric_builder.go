package ldbuilders

import (
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	"github.com/launchdarkly/go-server-sdk-evaluation/v2/ldmodel"
)

// MetricBuilder provides a builder pattern for Metric.
type MetricBuilder struct {
	metric ldmodel.Metric
}

// NewMetricBuilder creates a MetricBuilder.
func NewMetricBuilder(key string) *MetricBuilder {
	return &MetricBuilder{ldmodel.Metric{Key: key}}
}

// SamplingRatio sets the metric's sampling ratio.
func (b *MetricBuilder) SamplingRatio(ratio ldvalue.OptionalInt) *MetricBuilder {
	b.metric.SamplingRatio = ratio
	return b
}

// Version sets the metric's Version property.
func (b *MetricBuilder) Version(value int) *MetricBuilder {
	b.metric.Version = value
	return b
}

// Build returns the configured Metric.
func (b *MetricBuilder) Build() ldmodel.Metric {
	return b.metric
}
