package ldmodel

import (
	"encoding/json"
	"testing"
)

// Comprehensive benchmark comparison between all 3 JSON libraries:
// 1. Standard library (json package)
// 2. EasyJSON (with launchdarkly_easyjson build tag)

var (
	benchmarkComparisonBytesResult   []byte
	benchmarkComparisonErrorResult   error
	benchmarkComparisonFlagResult    FeatureFlag
	benchmarkComparisonSegmentResult Segment
)

// BenchmarkJSONLibraryComparison provides side-by-side comparison of all JSON libraries
func BenchmarkJSONLibraryComparison(b *testing.B) {
	// Test data
	flagBytes, _ := json.Marshal(flagWithAllPropertiesJSON)
	segmentBytes, _ := json.Marshal(segmentWithAllPropertiesJSON)
	largeFlagBytes := makeLargeFlagJSON()
	largeSegmentBytes := makeLargeSegmentJSON()

	b.Run("Flag Unmarshaling", func(b *testing.B) {
		b.Run("Standard JSON", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var f FeatureFlag
				benchmarkComparisonErrorResult = json.Unmarshal(flagBytes, &f)
			}
		})

		b.Run("EasyJSON", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchmarkComparisonFlagResult, benchmarkComparisonErrorResult =
						unmarshalFeatureFlagFromBytes(flagBytes)
			}
		})
	})

	b.Run("Segment Unmarshaling", func(b *testing.B) {
		b.Run("Standard JSON", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var s Segment
				benchmarkComparisonErrorResult = json.Unmarshal(segmentBytes, &s)
			}
		})

		b.Run("EasyJSON", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchmarkComparisonSegmentResult, benchmarkComparisonErrorResult =
						unmarshalSegmentFromBytes(segmentBytes)
			}
		})
	})

	b.Run("Large Flag Unmarshaling", func(b *testing.B) {
		b.Run("Standard JSON", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var f FeatureFlag
				benchmarkComparisonErrorResult = json.Unmarshal(largeFlagBytes, &f)
			}
		})

		b.Run("EasyJSON", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchmarkComparisonFlagResult, benchmarkComparisonErrorResult =
						unmarshalFeatureFlagFromBytes(largeFlagBytes)
			}
		})
	})

	b.Run("Large Segment Unmarshaling", func(b *testing.B) {
		b.Run("Standard JSON", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var s Segment
				benchmarkComparisonErrorResult = json.Unmarshal(largeSegmentBytes, &s)
			}
		})

		b.Run("EasyJSON", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.ReportAllocs()

				benchmarkComparisonSegmentResult, benchmarkComparisonErrorResult =
						unmarshalSegmentFromBytes(largeSegmentBytes)
			}
		})
	})
}

// BenchmarkMemoryUsageComparison focuses on memory allocation patterns
func BenchmarkMemoryUsageComparison(b *testing.B) {
	largeFlagBytes := makeLargeFlagJSON()
	largeSegmentBytes := makeLargeSegmentJSON()

	b.Run("Large Flag Memory", func(b *testing.B) {
		b.Run("Standard JSON", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var f FeatureFlag
				benchmarkComparisonErrorResult = json.Unmarshal(largeFlagBytes, &f)
			}
		})

		b.Run("EasyJSON", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchmarkComparisonFlagResult, benchmarkComparisonErrorResult =
						unmarshalFeatureFlagFromBytes(largeFlagBytes)
			}
		})
	})

	b.Run("Large Segment Memory", func(b *testing.B) {
		b.Run("Standard JSON", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var s Segment
				benchmarkComparisonErrorResult = json.Unmarshal(largeSegmentBytes, &s)
			}
		})

		b.Run("EasyJSON", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchmarkComparisonSegmentResult, benchmarkComparisonErrorResult =
						unmarshalSegmentFromBytes(largeSegmentBytes)
			}
		})
	})
}
