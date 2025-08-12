package ldmodel

import (
	"encoding/json"
	"runtime"
	"testing"
	"time"
)

// Test actual memory retention of parsed structures held long-term

func TestMemoryRetentionComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory retention test in short mode")
	}
	
	largeFlagBytes := makeLargeFlagJSON()
	numFlags := 1000
	
	testRetention := func(t *testing.T, name string, parseFunc func([]byte) (FeatureFlag, error)) {
		runtime.GC()
		runtime.GC()
		
		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)
		
		// Parse and retain flags
		flags := make([]FeatureFlag, 0, numFlags)
		for i := 0; i < numFlags; i++ {
			f, err := parseFunc(largeFlagBytes)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			flags = append(flags, f)
		}
		
		// Force GC and measure retained memory
		runtime.GC()
		runtime.GC() 
		time.Sleep(10 * time.Millisecond) // Let GC settle
		
		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)
		
		// Clamp to avoid uint64 underflow if GC reduced Alloc.
		retainedBytes := uint64(0)
		if memAfter.Alloc >= memBefore.Alloc {
			retainedBytes = memAfter.Alloc - memBefore.Alloc
		}
		bytesPerFlag := retainedBytes / uint64(numFlags)

		t.Logf("%s: %d flags retained %d bytes total (%.1f bytes/flag)", 
			name, numFlags, retainedBytes, float64(bytesPerFlag))
		
		// Keep reference to prevent GC optimization
		_ = flags[numFlags-1].Key
	}
	
	t.Run("Standard JSON", func(t *testing.T) {
		testRetention(t, "Standard JSON", func(data []byte) (FeatureFlag, error) {
			var f FeatureFlag
			err := json.Unmarshal(data, &f)
			if err == nil {
				PreprocessFlag(&f)
			}
			return f, err
		})
	})
	
	t.Run("Custom Unmarshaler", func(t *testing.T) {
		testRetention(t, "Custom Unmarshaler", func(data []byte) (FeatureFlag, error) {
			return unmarshalFeatureFlagFromBytes(data)
		})
	})
}

func BenchmarkActualMemoryFootprint(b *testing.B) {
	largeFlagBytes := makeLargeFlagJSON()
	
	b.Run("Memory Per Flag", func(b *testing.B) {
		flags := make([]*FeatureFlag, 0, b.N)
		
		var memStart runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&memStart)
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f, _ := unmarshalFeatureFlagFromBytes(largeFlagBytes)
			flags = append(flags, &f)
		}
		b.StopTimer()
		
		// Measure actual heap usage
		runtime.GC()
		var memEnd runtime.MemStats
		runtime.ReadMemStats(&memEnd)
		
		// Clamp to avoid uint64 underflow if GC reduced HeapInuse.
		var heapUsed uint64
		if memEnd.HeapInuse >= memStart.HeapInuse {
			heapUsed = memEnd.HeapInuse - memStart.HeapInuse
		}
		b.ReportMetric(float64(heapUsed)/float64(b.N), "heap-bytes/flag")

		// Prevent optimization
		if len(flags) > 0 {
			_ = flags[0].Key
		}
	})
}

// Test for string deduplication opportunities
func TestStringDeduplication(t *testing.T) {
	largeFlagBytes := makeLargeFlagJSON()
	numFlags := 100
	
	flags := make([]FeatureFlag, 0, numFlags)
	
	// Parse multiple similar flags
	for i := 0; i < numFlags; i++ {
		f, _ := unmarshalFeatureFlagFromBytes(largeFlagBytes)
		flags = append(flags, f)
	}
	
	// Analyze string duplication
	stringCounts := make(map[string]int)
	var totalStringBytes, totalStrings int
	
	for _, flag := range flags {
		// Count strings in rules
		for _, rule := range flag.Rules {
			for _, clause := range rule.Clauses {
				for _, val := range clause.Values {
					if val.IsString() {
						str := val.StringValue()
						stringCounts[str]++
						totalStringBytes += len(str)
						totalStrings++
					}
				}
			}
		}
	}
	
	uniqueStrings := len(stringCounts)
	var duplicateBytes int
	for str, count := range stringCounts {
		if count > 1 {
			duplicateBytes += len(str) * (count - 1)
		}
	}
	
	t.Logf("String analysis for %d flags:", numFlags)
	t.Logf("  Total strings: %d (%d bytes)", totalStrings, totalStringBytes)
	t.Logf("  Unique strings: %d", uniqueStrings)
	t.Logf("  Duplicate bytes: %d (%.1f%% of total)", duplicateBytes, 
		float64(duplicateBytes)/float64(totalStringBytes)*100)
		
	if duplicateBytes > 0 {
		t.Logf("  Potential savings: %d bytes (%.1f%% reduction)", 
			duplicateBytes, float64(duplicateBytes)/float64(totalStringBytes)*100)
	}
}
