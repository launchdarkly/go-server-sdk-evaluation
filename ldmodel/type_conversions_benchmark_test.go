package ldmodel

import (
	"testing"
	"time"

	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	"github.com/launchdarkly/go-semver"
)

var (
	benchmarkSemverResult semver.Version
	benchmarkTimeResult   time.Time
)

func BenchmarkValueToSemanticVersionNoAlloc(b *testing.B) {
	value := ldvalue.String("1.2.3")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var ok bool
		benchmarkSemverResult, ok = TypeConversions.ValueToSemanticVersion(value)
		if !ok {
			b.FailNow()
		}
	}
}

func BenchmarkValueToTimestampNoAlloc(b *testing.B) {
	b.Run("from numeric value", func(b *testing.B) {
		value := ldvalue.Int(1460851752000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var ok bool
			benchmarkTimeResult, ok = TypeConversions.ValueToTimestamp(value)
			if !ok {
				b.FailNow()
			}
		}
	})

	b.Run("from string value", func(b *testing.B) {
		value := ldvalue.String("2016-04-16T17:09:12-07:00")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var ok bool
			benchmarkTimeResult, ok = TypeConversions.ValueToTimestamp(value)
			if !ok {
				b.FailNow()
			}
		}
	})
}
