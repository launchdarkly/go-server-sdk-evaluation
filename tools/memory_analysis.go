package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"github.com/launchdarkly/go-server-sdk-evaluation/v3/ldmodel"
)

// Standalone memory analysis tool for comparing JSON libraries

func main() {
	fmt.Println("Memory Analysis Tool for JSON Libraries")
	fmt.Println("=====================================")
	
	// Create test data
	largeFlagBytes := makeLargeFlagJSON()
	numFlags := 1000
	
	fmt.Printf("Testing with %d flags, %d bytes JSON each\n\n", numFlags, len(largeFlagBytes))
	
	// Test Standard JSON
	testMemoryUsage("Standard JSON", numFlags, func(data []byte) (ldmodel.FeatureFlag, error) {
		var f ldmodel.FeatureFlag
		err := json.Unmarshal(data, &f)
		if err == nil {
			ldmodel.PreprocessFlag(&f)
		}
		return f, err
	}, largeFlagBytes)
	
	// Test Custom Unmarshaler  
	testMemoryUsage("Custom Unmarshaler", numFlags, func(data []byte) (ldmodel.FeatureFlag, error) {
		return UnmarshalFeatureFlagFromBytes(data)
	}, largeFlagBytes)
	
	// Test With Interning
	testMemoryUsage("With Interning", numFlags, func(data []byte) (ldmodel.FeatureFlag, error) {
		return UnmarshalFeatureFlagFromBytes(data)
	}, largeFlagBytes)
}

func testMemoryUsage(name string, numFlags int, parseFunc func([]byte) (ldmodel.FeatureFlag, error), data []byte) {
	fmt.Printf("Testing %s:\n", name)
	
	// Force GC and get baseline
	runtime.GC()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var memStart runtime.MemStats
	runtime.ReadMemStats(&memStart)
	
	// Parse flags
	flags := make([]ldmodel.FeatureFlag, 0, numFlags)
	start := time.Now()
	
	for i := 0; i < numFlags; i++ {
		f, err := parseFunc(data)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			return
		}
		flags = append(flags, f)
	}
	
	parseTime := time.Since(start)
	
	// Force GC and measure resident memory
	runtime.GC() 
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	
	var memEnd runtime.MemStats
	runtime.ReadMemStats(&memEnd)
	
	// Calculate metrics
	heapGrowth := memEnd.HeapInuse - memStart.HeapInuse
	allocGrowth := memEnd.TotalAlloc - memStart.TotalAlloc
	
	fmt.Printf("  Parse time: %v (%.2f μs/flag)\n", parseTime, float64(parseTime.Nanoseconds())/float64(numFlags)/1000)
	fmt.Printf("  Heap growth: %d bytes (%.1f bytes/flag)\n", heapGrowth, float64(heapGrowth)/float64(numFlags))
	fmt.Printf("  Total allocations: %d bytes (%.1f bytes/flag)\n", allocGrowth, float64(allocGrowth)/float64(numFlags))
	fmt.Printf("  Objects allocated: %d (%.1f/flag)\n", memEnd.Mallocs - memStart.Mallocs, float64(memEnd.Mallocs - memStart.Mallocs)/float64(numFlags))
	

	// Analyze structure
	analyzeStructure(flags[0])
	
	fmt.Println()
	
	// Keep reference
	_ = flags[numFlags-1].Key
}

func analyzeStructure(flag ldmodel.FeatureFlag) {
	fmt.Println("  Structure analysis:")
	
	// Count various elements
	var totalStrings, totalStringBytes int
	var totalSlices int
	
	// Flag-level strings
	totalStringBytes += len(flag.Key) + len(flag.Salt)
	if flag.Key != "" || flag.Salt != "" {
		totalStrings += 2
	}
	
	// Prerequisites
	totalSlices += len(flag.Prerequisites)
	for _, prereq := range flag.Prerequisites {
		totalStringBytes += len(prereq.Key)
		if prereq.Key != "" {
			totalStrings++
		}
	}
	
	// Targets
	totalSlices += len(flag.Targets)
	for _, target := range flag.Targets {
		totalSlices += len(target.Values)
		for _, val := range target.Values {
			totalStringBytes += len(val)
			totalStrings++
		}
	}
	
	// Rules
	totalSlices += len(flag.Rules)
	for _, rule := range flag.Rules {
		totalStringBytes += len(rule.ID)
		if rule.ID != "" {
			totalStrings++
		}
		totalSlices += len(rule.Clauses)
		for _, clause := range rule.Clauses {
			totalSlices += len(clause.Values)
			for _, val := range clause.Values {
				if val.IsString() {
					totalStringBytes += len(val.StringValue())
					totalStrings++
				}
			}
		}
	}
	
	// Variations
	totalSlices += len(flag.Variations)
	for _, variation := range flag.Variations {
		if variation.IsString() {
			totalStringBytes += len(variation.StringValue())
			totalStrings++
		}
	}
	
	fmt.Printf("    Strings: %d (%d bytes)\n", totalStrings, totalStringBytes)
	fmt.Printf("    Slices/arrays: %d\n", totalSlices)
}

// Helper function to create large flag JSON (copied from test)
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

// Helper function to unmarshal using the public API
func UnmarshalFeatureFlagFromBytes(data []byte) (ldmodel.FeatureFlag, error) {
	serializer := ldmodel.NewJSONDataModelSerialization()
	return serializer.UnmarshalFeatureFlag(data)
}
