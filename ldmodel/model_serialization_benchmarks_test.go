package ldmodel

import (
	"encoding/json"
	"fmt"
	"testing"

	"gopkg.in/launchdarkly/go-jsonstream.v1/jreader"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
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

func makeLargeFlagJSON() []byte {
	makeManyStrings := func() []string {
		ret := []string{}
		for i := 0; i < 200; i++ {
			ret = append(ret, fmt.Sprintf("string%d", i))
		}
		return ret
	}
	makeRules := func() []map[string]interface{} {
		ret := []map[string]interface{}{}
		for i := 0; i < 20; i++ {
			ret = append(ret, map[string]interface{}{
				"id": fmt.Sprintf("rule-id%d", i),
				"clauses": []interface{}{
					map[string]interface{}{
						"attribute": "name",
						"op":        "in",
						"values":    []interface{}{"clause-value"},
						"negate":    true,
					},
				},
				"variation":   float64(1),
				"trackEvents": true,
			})
		}
		return ret
	}
	data := map[string]interface{}{
		"key": "large-flag-key",
		"on":  true,
		"prerequisites": []interface{}{
			map[string]interface{}{
				"key":       "prereq-key",
				"variation": float64(1),
			},
		},
		"targets": []interface{}{
			map[string]interface{}{
				"values":    makeManyStrings(),
				"variation": float64(2),
			},
		},
		"rules": makeRules(),
		"fallthrough": map[string]interface{}{
			"rollout": map[string]interface{}{
				"variations": []interface{}{
					map[string]interface{}{
						"weight":    float64(100000),
						"variation": float64(3),
					},
				},
			},
		},
		"offVariation": float64(3),
		"variations":   []interface{}{false, float64(9), "other"},
		"clientSideAvailability": map[string]interface{}{
			"usingEnvironmentId": true,
			"usingMobileKey":     true,
		},
		"clientSide":             true,
		"salt":                   "flag-salt",
		"trackEvents":            true,
		"trackEventsFallthrough": true,
		"debugEventsUntilDate":   float64(1000),
		"version":                float64(99),
		"deleted":                true,
	}
	bytes, _ := json.Marshal(data)
	return bytes
}

func makeLargeSegmentJSON() []byte {
	makeManyStrings := func() []string {
		ret := []string{}
		for i := 0; i < 200; i++ {
			ret = append(ret, fmt.Sprintf("string%d", i))
		}
		return ret
	}
	makeRules := func() []map[string]interface{} {
		ret := []map[string]interface{}{}
		for i := 0; i < 20; i++ {
			ret = append(ret, map[string]interface{}{
				"id": fmt.Sprintf("rule-id%d", i),
				"clauses": []interface{}{
					map[string]interface{}{
						"attribute": "name",
						"op":        "in",
						"values":    []interface{}{"clause-value"},
						"negate":    true,
					},
				},
				"weight":   float64(50000),
				"bucketBy": "name",
			})
		}
		return ret
	}
	data := map[string]interface{}{
		"key":       "large-segment-key",
		"included":  makeManyStrings(),
		"excluded":  makeManyStrings(),
		"rules":     makeRules(),
		"salt":      "segment-salt",
		"unbounded": true,
		"version":   float64(99),
		"deleted":   true,
	}
	bytes, _ := json.Marshal(data)
	return bytes
}

type featureFlagEquivalentStruct struct {
	Key           string `json:"key"`
	On            bool   `json:"on"`
	Prerequisites []struct {
		Key       string `json:"key"`
		Variation int    `json:"variation"`
	} `json:"prerequisites"`
	Targets []struct {
		Values    []string `json:"values"`
		Variation int      `json:"variation"`
	} `json:"targets"`
	Rules []struct {
		Variation *int `json:"variation"`
		Rollout   *struct {
			Variations []struct {
				Variation int `json:"variation"`
				Weight    int `json:"weight"`
			} `json:"variations"`
			BucketBy *string `json:"bucketBy"`
		} `json:"rollout"`
		ID      string `json:"id"`
		Clauses []struct {
			Attribute string          `json:"attribute"`
			Op        string          `json:"op"`
			Values    []ldvalue.Value `json:"values"`
			Negate    bool            `json:"negate"`
		} `json:"clauses"`
		TrackEvents bool `json:"trackEvents"`
	} `json:"rules"`
	Fallthrough struct {
		Variation *int `json:"variation"`
		Rollout   struct {
			Variations []struct {
			} `json:"variations"`
			BucketBy *string `json:"bucketBy"`
		} `json:"rollout"`
	} `json:"fallthrough"`
	OffVariation           *int            `json:"offVariation"`
	Variations             []ldvalue.Value `json:"variations"`
	ClientSideAvailability *struct {
		UsingMobileKey     bool `json:"usingMobileKey"`
		UsingEnvironmentID bool `json:"usingEnvironmentId"`
	} `json:"clientSideAvailability"`
	Salt                   string  `json:"salt"`
	TrackEvents            bool    `json:"trackEvents"`
	TrackEventsFallthrough bool    `json:"trackEventsFallthrough"`
	DebugEventsUntilDate   *uint64 `json:"debugEventsUntilDate"`
	Version                int     `json:"version"`
	Deleted                bool    `json:"deleted"`
}

type segmentEquivalentStruct struct {
	Key      string   `json:"key"`
	Included []string `json:"included"`
	Excluded []string `json:"excluded"`
	Salt     string   `json:"salt"`
	Rules    []struct {
		ID      string `json:"id"`
		Clauses []struct {
			Attribute string          `json:"attribute"`
			Op        string          `json:"op"`
			Values    []ldvalue.Value `json:"values"`
			Negate    bool            `json:"negate"`
		} `json:"clauses"`
		Weight   *int    `json:"weight"`
		BucketBy *string `json:"bucketBy"`
	} `json:"rules"`
	Unbounded bool `json:"unbounded"`
	Version   int  `json:"version"`
	Deleted   bool `json:"deleted"`
}
