package ldmodel

import (
	"encoding/json"
	"runtime"
	"testing"
	"unsafe"
)

// Memory footprint benchmarks for long-term storage of parsed structures

func benchmarkResidentMemory[T any](
		b *testing.B, unmarshalFunc func([]byte) T, data []byte, metricName string, keepRef func(T),
) {
	items := make([]T, 0, b.N)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		item := unmarshalFunc(data)
		items = append(items, item)
	}

	b.StopTimer()
	runtime.GC()
	runtime.GC()

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)
	keepRef(items[len(items)-1])
	runtime.ReadMemStats(&m2)
	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), metricName)
}

func BenchmarkMemoryFootprint(b *testing.B) {
	largeFlagBytes := makeLargeFlagJSON()
	largeSegmentBytes := makeLargeSegmentJSON()

	b.Run("Large Flag Resident Memory", func(b *testing.B) {
		b.Run("Standard JSON", func(b *testing.B) {
			benchmarkResidentMemory(b, func(data []byte) FeatureFlag {
				var f FeatureFlag
				_ = json.Unmarshal(data, &f)
				return f
			}, largeFlagBytes, "resident-bytes/flag", func(f FeatureFlag) { _ = f.Key })
		})

		b.Run("EasyJSON", func(b *testing.B) {
			benchmarkResidentMemory(b, func(data []byte) FeatureFlag {
				f, _ := unmarshalFeatureFlagFromBytes(data)
				return f
			}, largeFlagBytes, "resident-bytes/flag", func(f FeatureFlag) { _ = f.Key })
		})
	})

	b.Run("Large Segment Resident Memory", func(b *testing.B) {
		b.Run("Standard JSON", func(b *testing.B) {
			benchmarkResidentMemory(b, func(data []byte) Segment {
				var s Segment
				_ = json.Unmarshal(data, &s)
				return s
			}, largeSegmentBytes, "resident-bytes/segment", func(s Segment) { _ = s.Key })
		})

		b.Run("EasyJSON", func(b *testing.B) {
			benchmarkResidentMemory(b, func(data []byte) Segment {
				s, _ := unmarshalSegmentFromBytes(data)
				return s
			}, largeSegmentBytes, "resident-bytes/segment", func(s Segment) { _ = s.Key })
		})
	})
}

// Structural analysis of memory usage
func BenchmarkStructuralMemoryAnalysis(b *testing.B) {
	largeFlagBytes := makeLargeFlagJSON()

	b.Run("Flag Structure Size Analysis", func(b *testing.B) {
		f, _ := unmarshalFeatureFlagFromBytes(largeFlagBytes)

		flagSize := unsafe.Sizeof(f)
		b.ReportMetric(float64(flagSize), "base-struct-bytes")

		// Measure string memory usage
		var stringMemory uintptr
		stringMemory += uintptr(len(f.Key))
		stringMemory += uintptr(len(f.Salt))
		for _, prereq := range f.Prerequisites {
			stringMemory += uintptr(len(prereq.Key))
		}
		for _, target := range f.Targets {
			for _, val := range target.Values {
				stringMemory += uintptr(len(val))
			}
		}
		for _, rule := range f.Rules {
			stringMemory += uintptr(len(rule.ID))
		}

		b.ReportMetric(float64(stringMemory), "string-bytes")

		// Count total allocations in structure
		var totalSlices int
		totalSlices += len(f.Prerequisites)
		totalSlices += len(f.Targets)
		totalSlices += len(f.Rules)
		totalSlices += len(f.Variations)
		for _, target := range f.Targets {
			totalSlices += len(target.Values)
		}
		for _, rule := range f.Rules {
			totalSlices += len(rule.Clauses)
			for _, clause := range rule.Clauses {
				totalSlices += len(clause.Values)
			}
		}

		b.ReportMetric(float64(totalSlices), "slice-elements")
	})
}

// Deduplication potential analysis
func BenchmarkDeduplicationPotential(b *testing.B) {
	// Create multiple similar flags to measure duplication
	createSimilarFlag := func(id int) FeatureFlag {
		baseFlag, _ := unmarshalFeatureFlagFromBytes(makeLargeFlagJSON())
		// Simulate flags that are similar but not identical
		baseFlag.Key = baseFlag.Key + string(rune(id%10))
		return baseFlag
	}

	b.Run("Similar Flags Memory Usage", func(b *testing.B) {
		flags := make([]FeatureFlag, b.N)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			flags[i] = createSimilarFlag(i)
		}
		b.StopTimer()

		runtime.GC()
		runtime.GC()

		// Analyze string duplication
		stringSet := make(map[string]int)
		var totalStringBytes, uniqueStringBytes uintptr

		for _, flag := range flags {
			// Count string usage
			for _, rule := range flag.Rules {
				for _, clause := range rule.Clauses {
					for _, val := range clause.Values {
						if val.IsString() {
							str := val.StringValue()
							totalStringBytes += uintptr(len(str))
							if stringSet[str] == 0 {
								uniqueStringBytes += uintptr(len(str))
							}
							stringSet[str]++
						}
					}
				}
			}
		}

		duplicateBytes := totalStringBytes - uniqueStringBytes
		b.ReportMetric(float64(totalStringBytes), "total-string-bytes")
		b.ReportMetric(float64(uniqueStringBytes), "unique-string-bytes")
		b.ReportMetric(float64(duplicateBytes), "duplicate-string-bytes")
		if totalStringBytes > 0 {
			b.ReportMetric(float64(duplicateBytes)/float64(totalStringBytes)*100, "duplication-percent")
		}

		// Keep reference to prevent GC
		_ = flags[0].Key
	})
}
