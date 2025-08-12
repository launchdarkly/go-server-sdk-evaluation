package ldmodel

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"
)

// Test interning with realistic scenarios: many similar flags with shared strings

func TestRealisticInterningScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping realistic interning test in short mode")
	}

	// Create multiple similar flags with shared string patterns
	createSimilarFlag := func(id int) []byte {
		// Create flags with common patterns but slight variations
		flagData := map[string]any{
			"key":     fmt.Sprintf("flag-%d", id),
			"version": id,
			"on":      true,
			"salt":    "shared-salt-value", // Common across flags
			"rules": []any{
				map[string]any{
					"id": fmt.Sprintf("rule-%d", id),
					"clauses": []any{
						map[string]any{
							"attribute": "email",                       // Common
							"op":        "contains",                    // Common
							"values":    []any{"@company.com"}, // Common
							"negate":    false,
						},
						map[string]any{
							"attribute": "country",                       // Common
							"op":        "in",                            // Common
							"values":    []any{"US", "CA", "UK"}, // Common values
							"negate":    false,
						},
					},
					"variation": 1,
				},
			},
			"targets": []any{
				map[string]any{
					"contextKind": "user", // Common
					"values":      []any{fmt.Sprintf("user-%d", id)},
					"variation":   0,
				},
			},
			"variations": []any{true, false, "default"}, // Common values
			"fallthrough": map[string]any{
				"variation": 2,
			},
		}
		bytes, _ := json.Marshal(flagData)
		return bytes
	}

	numFlags := 1000

	t.Run("Without Interning", func(t *testing.T) {
		runtime.GC()
		runtime.GC()
		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		flags := make([]FeatureFlag, 0, numFlags)
		for i := 0; i < numFlags; i++ {
			flagBytes := createSimilarFlag(i)
			var f FeatureFlag
			json.Unmarshal(flagBytes, &f)
			PreprocessFlag(&f)
			flags = append(flags, f)
		}

		runtime.GC()
		runtime.GC()
		time.Sleep(10 * time.Millisecond)

		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)

		retainedBytes := memAfter.Alloc - memBefore.Alloc
		bytesPerFlag := retainedBytes / uint64(numFlags)

		t.Logf("WITHOUT Interning: %d flags = %d bytes total (%.1f bytes/flag)",
			numFlags, retainedBytes, float64(bytesPerFlag))
		// Report total allocation events during parsing
		allocs := memAfter.Mallocs - memBefore.Mallocs
		t.Logf("  Objects allocated: %d (%.1f/flag)", allocs, float64(allocs)/float64(numFlags))

		// Analyze string duplication
		stringCounts := analyzeStringDuplication(flags)
		var totalStringBytes, duplicateBytes uint64
		for str, count := range stringCounts {
			totalStringBytes += uint64(len(str) * count)
			if count > 1 {
				duplicateBytes += uint64(len(str) * (count - 1))
			}
		}

		t.Logf("  String bytes: %d total, %d duplicate (%.1f%% waste)",
			totalStringBytes, duplicateBytes, float64(duplicateBytes)/float64(totalStringBytes)*100)

		_ = flags[numFlags-1].Key
	})

	t.Run("With Interning", func(t *testing.T) {
		runtime.GC()
		runtime.GC()
		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		flags := make([]FeatureFlag, 0, numFlags)
		for i := 0; i < numFlags; i++ {
			flagBytes := createSimilarFlag(i)
			f, _ := unmarshalFeatureFlagFromBytes(flagBytes)
			flags = append(flags, f)
		}

		runtime.GC()
		runtime.GC()
		time.Sleep(10 * time.Millisecond)

		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)

		retainedBytes := memAfter.Alloc - memBefore.Alloc
		bytesPerFlag := retainedBytes / uint64(numFlags)

		t.Logf("WITH Interning: %d flags = %d bytes total (%.1f bytes/flag)",
			numFlags, retainedBytes, float64(bytesPerFlag))
		t.Logf("  Interning: keys pinned, values ephemeral (build-tag controlled)")
		// Report total allocation events during parsing
		allocs := memAfter.Mallocs - memBefore.Mallocs
		t.Logf("  Objects allocated: %d (%.1f/flag)", allocs, float64(allocs)/float64(numFlags))

		// Analyze string sharing
		stringCounts := analyzeStringDuplication(flags)
		uniqueStrings := len(stringCounts)
		var totalStringRefs int
		for _, count := range stringCounts {
			totalStringRefs += count
		}

		t.Logf("  String sharing: %d refs → %d unique strings (%.1fx deduplication)",
			totalStringRefs, uniqueStrings, float64(totalStringRefs)/float64(uniqueStrings))

		_ = flags[numFlags-1].Key
	})
}

func analyzeStringDuplication(flags []FeatureFlag) map[string]int {
	stringCounts := make(map[string]int)

	for _, flag := range flags {
		countString := func(s string) {
			if s != "" {
				stringCounts[s]++
			}
		}

		countString(flag.Key)
		countString(flag.Salt)

		for _, rule := range flag.Rules {
			countString(rule.ID)
			for _, clause := range rule.Clauses {
				countString(string(clause.Op))
				countString(string(clause.ContextKind))
				for _, val := range clause.Values {
					if val.IsString() {
						countString(val.StringValue())
					}
				}
			}
		}

		for _, target := range flag.Targets {
			countString(string(target.ContextKind))
			for _, val := range target.Values {
				countString(val)
			}
		}

		for _, variation := range flag.Variations {
			if variation.IsString() {
				countString(variation.StringValue())
			}
		}
	}

	return stringCounts
}

// Benchmark that measures the key benefit: memory scaling with similar flags
func BenchmarkInterningMemoryScaling(b *testing.B) {
	// Create template for similar flags
	createFlagTemplate := func(id int) []byte {
		flagData := map[string]any{
			"key":     fmt.Sprintf("feature-flag-%d", id),
			"version": id,
			"salt":    "production-salt-2024", // Same across all flags
			"rules": []any{
				map[string]any{
					"clauses": []any{
						map[string]any{
							"attribute": "email",                    // Common
							"op":        "endsWith",                 // Common
							"values":    []any{"@acme.com"}, // Common
						},
					},
					"variation": 1,
				},
			},
			"variations": []any{"control", "treatment", "disabled"}, // Common
		}
		bytes, _ := json.Marshal(flagData)
		return bytes
	}

	b.Run("Memory Per Flag Without Interning", func(b *testing.B) {
		flags := make([]*FeatureFlag, 0, b.N)

		var memStart runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memStart)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			flagBytes := createFlagTemplate(i)
			var f FeatureFlag
			json.Unmarshal(flagBytes, &f)
			PreprocessFlag(&f)
			flags = append(flags, &f)
		}
		b.StopTimer()

		runtime.GC()
		var memEnd runtime.MemStats
		runtime.ReadMemStats(&memEnd)

		heapUsed := memEnd.HeapInuse - memStart.HeapInuse
		b.ReportMetric(float64(heapUsed)/float64(b.N), "heap-bytes/flag")

		if len(flags) > 0 {
			_ = flags[0].Key
		}
	})

	b.Run("Memory Per Flag With Interning", func(b *testing.B) {
		flags := make([]*FeatureFlag, 0, b.N)

		var memStart runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memStart)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			flagBytes := createFlagTemplate(i)
			f, _ := unmarshalFeatureFlagFromBytes(flagBytes)
			flags = append(flags, &f)
		}
		b.StopTimer()

		runtime.GC()
		var memEnd runtime.MemStats
		runtime.ReadMemStats(&memEnd)

		heapUsed := memEnd.HeapInuse - memStart.HeapInuse
		b.ReportMetric(float64(heapUsed)/float64(b.N), "heap-bytes/flag")

		if len(flags) > 0 {
			_ = flags[0].Key
		}
	})
}

// Test to measure the effect as we scale up the number of flags
func TestInterningScalingEffect(t *testing.T) {
	createSharedStringsFlag := func(id int) []byte {
		// Lots of repeated strings
		flagData := map[string]any{
			"key":     fmt.Sprintf("experiment-%d", id),
			"version": id,
			"salt":    "shared-experiment-salt",
			"rules": []any{
				map[string]any{
					"clauses": []any{
						map[string]any{
							"attribute": "plan",
							"op":        "in",
							"values":    []any{"premium", "enterprise", "free"},
						},
						map[string]any{
							"attribute": "region",
							"op":        "in",
							"values":    []any{"us-east", "us-west", "eu-west"},
						},
					},
					"variation": 1,
				},
			},
			"variations": []any{"off", "on", "test"},
		}
		bytes, _ := json.Marshal(flagData)
		return bytes
	}

	// Test scaling from 100 to 1000 flags
	for _, numFlags := range []int{100, 500, 1000} {
		t.Run(fmt.Sprintf("Scaling_%d_flags", numFlags), func(t *testing.T) {
			// Without interning
			runtime.GC()
			var memBefore1 runtime.MemStats
			runtime.ReadMemStats(&memBefore1)

			flagsNoIntern := make([]FeatureFlag, 0, numFlags)
			for i := 0; i < numFlags; i++ {
				flagBytes := createSharedStringsFlag(i)
				var f FeatureFlag
				json.Unmarshal(flagBytes, &f)
				PreprocessFlag(&f)
				flagsNoIntern = append(flagsNoIntern, f)
			}

			runtime.GC()
			var memAfter1 runtime.MemStats
			runtime.ReadMemStats(&memAfter1)
			memWithoutIntern := memAfter1.Alloc - memBefore1.Alloc

			// With interning
			runtime.GC()
			var memBefore2 runtime.MemStats
			runtime.ReadMemStats(&memBefore2)

			flagsWithIntern := make([]FeatureFlag, 0, numFlags)
			for i := 0; i < numFlags; i++ {
				flagBytes := createSharedStringsFlag(i)
				f, _ := unmarshalFeatureFlagFromBytes(flagBytes)
				flagsWithIntern = append(flagsWithIntern, f)
			}

			runtime.GC()
			var memAfter2 runtime.MemStats
			runtime.ReadMemStats(&memAfter2)
			memWithIntern := memAfter2.Alloc - memBefore2.Alloc

			savings := float64(memWithoutIntern-memWithIntern) / float64(memWithoutIntern) * 100

			t.Logf("Flags: %d", numFlags)
			t.Logf("  Without interning: %d bytes (%.1f/flag)", memWithoutIntern, float64(memWithoutIntern)/float64(numFlags))
			t.Logf("  With interning:    %d bytes (%.1f/flag)", memWithIntern, float64(memWithIntern)/float64(numFlags))
			t.Logf("  Memory savings:    %.1f%%", savings)

			// Keep references
			_ = flagsNoIntern[numFlags-1].Key
			_ = flagsWithIntern[numFlags-1].Key
		})
	}
}
