package ldmodel

import (
	"encoding/json"
	"testing"

	"gopkg.in/launchdarkly/go-jsonstream.v1/jreader"
)

var (
	benchmarkBytesResult   []byte
	benchmarkErrorResult   error
	benchmarkFlagResult    FeatureFlag
	benchmarkSegmentResult Segment
)

func BenchmarkMarshalFlag(b *testing.B) {
	b.Run("all properties", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchmarkBytesResult, benchmarkErrorResult =
				jsonDataModelSerialization{}.MarshalFeatureFlag(flagWithAllProperties)
		}
	})

	b.Run("minimal properties", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchmarkBytesResult, benchmarkErrorResult =
				jsonDataModelSerialization{}.MarshalFeatureFlag(flagWithMinimalProperties)
		}
	})
}

func BenchmarkMarshalSegment(b *testing.B) {
	b.Run("all properties", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchmarkBytesResult, benchmarkErrorResult =
				jsonDataModelSerialization{}.MarshalSegment(segmentWithAllProperties)
		}
	})

	b.Run("minimal properties", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchmarkBytesResult, benchmarkErrorResult =
				jsonDataModelSerialization{}.MarshalSegment(segmentWithMinimalProperties)
		}
	})
}

func BenchmarkUnmarshalFlag(b *testing.B) {
	b.Run("all properties", func(b *testing.B) {
		bytes, _ := json.Marshal(flagWithAllPropertiesJSON)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			benchmarkFlagResult, benchmarkErrorResult =
				jsonDataModelSerialization{}.UnmarshalFeatureFlag(bytes)
		}
	})

	b.Run("minimal properties", func(b *testing.B) {
		bytes, _ := json.Marshal(flagWithMinimalPropertiesJSON)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			benchmarkFlagResult, benchmarkErrorResult =
				jsonDataModelSerialization{}.UnmarshalFeatureFlag(bytes)
		}
	})
}

func BenchmarkUnmarshalSegment(b *testing.B) {
	b.Run("all properties", func(b *testing.B) {
		bytes, _ := json.Marshal(segmentWithAllPropertiesJSON)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			benchmarkSegmentResult, benchmarkErrorResult =
				jsonDataModelSerialization{}.UnmarshalSegment(bytes)
		}
	})

	b.Run("minimal properties", func(b *testing.B) {
		bytes, _ := json.Marshal(segmentWithMinimalPropertiesJSON)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			benchmarkSegmentResult, benchmarkErrorResult =
				jsonDataModelSerialization{}.UnmarshalSegment(bytes)
		}
	})
}

func BenchmarkLargeFlagComparative(b *testing.B) {
	b.Run("our unmarshaler", func(b *testing.B) {
		bytes := makeLargeFlagJSON()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			r := jreader.NewReader(bytes)
			var f FeatureFlag
			readFeatureFlag(&r, &f)
			// Calling the lower-level function readFeatureFlag means we're skipping the post-processing step,
			// since we're not doing that step in the comparative UnmarshalJSON benchmark.
			benchmarkErrorResult = r.Error()
			if benchmarkErrorResult != nil {
				b.Error(benchmarkErrorResult)
				b.FailNow()
			}
		}
	})

	b.Run("UnmarshalJSON for equivalent struct", func(b *testing.B) {
		bytes := makeLargeFlagJSON()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var f featureFlagEquivalentStruct
			benchmarkErrorResult = json.Unmarshal(bytes, &f)
			if benchmarkErrorResult != nil {
				b.Error(benchmarkErrorResult)
				b.FailNow()
			}
		}
	})
}

func BenchmarkLargeSegmentComparative(b *testing.B) {
	b.Run("our unmarshaler", func(b *testing.B) {
		bytes := makeLargeSegmentJSON()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			r := jreader.NewReader(bytes)
			var s Segment
			readSegment(&r, &s)
			// Calling the lower-level function readSegment means we're skipping the post-processing step,
			// since we're not doing that step in the comparative UnmarshalJSON benchmark.
			benchmarkErrorResult = r.Error()
			if benchmarkErrorResult != nil {
				b.Error(benchmarkErrorResult)
				b.FailNow()
			}
		}
	})

	b.Run("UnmarshalJSON for equivalent struct", func(b *testing.B) {
		bytes := makeLargeSegmentJSON()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var s segmentEquivalentStruct
			benchmarkErrorResult = json.Unmarshal(bytes, &s)
			if benchmarkErrorResult != nil {
				b.Error(benchmarkErrorResult)
				b.FailNow()
			}
		}
	})
}
