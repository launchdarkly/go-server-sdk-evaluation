//go:build go1.23 && launchdarkly_intern_json

package intern

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"unique"

	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
)

func TestLRUCache(t *testing.T) {
	// Create a small cache for testing
	cache := newLRUCache(3)

	// Test adding items
	h1 := unique.Make("test1")
	h2 := unique.Make("test2")
	h3 := unique.Make("test3")

	cache.put("test1", h1)
	cache.put("test2", h2)
	cache.put("test3", h3)

	// Verify all items are in cache
	if _, exists := cache.get("test1"); !exists {
		t.Error("test1 should exist in cache")
	}
	if _, exists := cache.get("test2"); !exists {
		t.Error("test2 should exist in cache")
	}
	if _, exists := cache.get("test3"); !exists {
		t.Error("test3 should exist in cache")
	}

	// Add a fourth item, should evict test1 (least recently used)
	h4 := unique.Make("test4")
	cache.put("test4", h4)

	// test1 should be evicted
	if _, exists := cache.get("test1"); exists {
		t.Error("test1 should have been evicted")
	}

	// Others should still exist
	if _, exists := cache.get("test2"); !exists {
		t.Error("test2 should still exist")
	}
	if _, exists := cache.get("test3"); !exists {
		t.Error("test3 should still exist")
	}
	if _, exists := cache.get("test4"); !exists {
		t.Error("test4 should exist")
	}

	// Check statistics
	hits, misses, evicted, size, capacity := cache.stats()
	if capacity != 3 {
		t.Errorf("Expected capacity 3, got %d", capacity)
	}
	if size != 3 {
		t.Errorf("Expected size 3, got %d", size)
	}
	if evicted != 1 {
		t.Errorf("Expected 1 eviction, got %d", evicted)
	}
	if hits == 0 {
		t.Error("Expected some cache hits")
	}
	if misses == 0 {
		t.Error("Expected some cache misses")
	}
}

func TestLRUCacheAccessOrder(t *testing.T) {
	cache := newLRUCache(2)

	h1 := unique.Make("a")
	h2 := unique.Make("b")
	h3 := unique.Make("c")

	// Add two items
	cache.put("a", h1)
	cache.put("b", h2)

	// Access "a" to make it most recent
	cache.get("a")

	// Add "c", should evict "b" (least recently used)
	cache.put("c", h3)

	// "a" should still exist
	if _, exists := cache.get("a"); !exists {
		t.Error("'a' should still exist after access")
	}

	// "b" should be evicted
	if _, exists := cache.get("b"); exists {
		t.Error("'b' should have been evicted")
	}

	// "c" should exist
	if _, exists := cache.get("c"); !exists {
		t.Error("'c' should exist")
	}
}

func TestStringInternWithLRU(t *testing.T) {
	// Reset global state for testing
	cache = newLRUCache(2) // Small cache for testing

	// Intern some strings
	s1 := String("hello")
	_ = String("world")   // Fill cache
	s3 := String("hello") // Should hit cache

	if s1 != s3 {
		t.Error("Same string should return identical values")
	}

	// Add a third unique string, should evict one
	_ = String("test")

	// Check cache stats
	hits, misses, evicted, size, capacity := CacheStats()

	t.Logf("Cache stats: hits=%d, misses=%d, evicted=%d, size=%d, capacity=%d",
		hits, misses, evicted, size, capacity)

	if hits == 0 {
		t.Error("Expected at least one cache hit")
	}

	if capacity != 2 {
		t.Errorf("Expected capacity 2, got %d", capacity)
	}

	// Test that identical strings still work after eviction
	s5 := String("hello")
	if s1 != s5 {
		t.Error("Identical strings should still be equal even after cache operations")
	}
}

func TestCacheSizeConfiguration(t *testing.T) {
	// Test default size
	size := getRetainCacheSize()
	if size != 8192 {
		t.Errorf("Default cache size should be 8192, got %d", size)
	}
}

func BenchmarkLRUCacheOperations(b *testing.B) {
	cache := newLRUCache(1000)

	b.ResetTimer()
	b.Run("put", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i%1000)
			handle := unique.Make(key)
			cache.put(key, handle)
		}
	})

	// Pre-populate cache for get benchmark
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		handle := unique.Make(key)
		cache.put(key, handle)
	}

	b.Run("get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i%1000)
			cache.get(key)
		}
	})
}

func BenchmarkStringInternLRU(b *testing.B) {
	// Reset with a reasonably sized cache
	cache = newLRUCache(10000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Mix of repeated and unique strings to test cache effectiveness
			var s string
			if i%3 == 0 {
				s = fmt.Sprintf("repeated-%d", i%100) // Repeated strings
			} else {
				s = fmt.Sprintf("unique-%d", i) // Unique strings
			}
			String(s)
			i++
		}
	})
}

func TestLRUCacheEdgeCases(t *testing.T) {
	t.Run("zero_capacity", func(t *testing.T) {
		cache := newLRUCache(0)
		if cache.capacity != 8192 {
			t.Errorf("Zero capacity should default to 8192, got %d", cache.capacity)
		}
	})

	t.Run("negative_capacity", func(t *testing.T) {
		cache := newLRUCache(-5)
		if cache.capacity != 8192 {
			t.Errorf("Negative capacity should default to 8192, got %d", cache.capacity)
		}
	})

	t.Run("single_capacity", func(t *testing.T) {
		cache := newLRUCache(1)
		h1 := unique.Make("test1")
		h2 := unique.Make("test2")

		cache.put("test1", h1)
		if _, exists := cache.get("test1"); !exists {
			t.Error("test1 should exist")
		}

		// Adding second item should evict first
		cache.put("test2", h2)
		if _, exists := cache.get("test1"); exists {
			t.Error("test1 should be evicted")
		}
		if _, exists := cache.get("test2"); !exists {
			t.Error("test2 should exist")
		}
	})

	t.Run("update_existing_key", func(t *testing.T) {
		cache := newLRUCache(2)
		h1 := unique.Make("test1")
		h2 := unique.Make("test1_updated")

		cache.put("test1", h1)
		cache.put("test1", h2) // Update existing key

		handle, exists := cache.get("test1")
		if !exists {
			t.Error("test1 should exist after update")
		}
		if handle.Value() != "test1_updated" {
			t.Error("Handle should be updated")
		}

		// Size should still be 1
		_, _, _, size, _ := cache.stats()
		if size != 1 {
			t.Errorf("Cache size should be 1 after update, got %d", size)
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	// Reset global cache for testing
	cache = newLRUCache(1000)

	const numGoroutines = 10
	const opsPerGoroutine = 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent read/write operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				// Mix of shared and unique strings
				var s string
				if j%3 == 0 {
					s = "shared_string" // Frequently accessed
				} else {
					s = fmt.Sprintf("goroutine_%d_string_%d", id, j)
				}
				result := String(s)
				if result != s {
					t.Errorf("String(%s) returned %s", s, result)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify cache statistics make sense
	hits, misses, evicted, size, capacity := CacheStats()
	t.Logf("Concurrent test stats: hits=%d, misses=%d, evicted=%d, size=%d, capacity=%d",
		hits, misses, evicted, size, capacity)

	if hits == 0 {
		t.Error("Expected some cache hits in concurrent test")
	}
	if size > capacity {
		t.Errorf("Cache size (%d) should not exceed capacity (%d)", size, capacity)
	}
}

func TestMemoryPressure(t *testing.T) {
	// Skip this test in short mode
	if testing.Short() {
		t.Skip("Skipping memory pressure test in short mode")
	}

	// Create a small cache to force evictions
	cache = newLRUCache(100)

	// Generate many unique strings to force evictions
	const numStrings = 1000
	strings := make([]string, numStrings)
	for i := 0; i < numStrings; i++ {
		strings[i] = String(fmt.Sprintf("memory_pressure_test_string_%d", i))
	}

	// Force GC and check that we don't have memory leaks
	runtime.GC()
	runtime.GC()

	hits, misses, evicted, size, capacity := CacheStats()
	t.Logf("Memory pressure stats: hits=%d, misses=%d, evicted=%d, size=%d, capacity=%d",
		hits, misses, evicted, size, capacity)

	// Should have evicted many items
	if evicted == 0 {
		t.Error("Expected some evictions under memory pressure")
	}

	// Cache should be at capacity
	if size != capacity {
		t.Errorf("Cache size (%d) should equal capacity (%d) after pressure test", size, capacity)
	}

	// Verify some early strings were evicted by checking cache misses
	earlyMisses := 0
	for i := 0; i < 10; i++ {
		if _, exists := cache.get(fmt.Sprintf("memory_pressure_test_string_%d", i)); !exists {
			earlyMisses++
		}
	}

	if earlyMisses == 0 {
		t.Error("Expected some early strings to be evicted")
	}
}

func TestCacheConsistency(t *testing.T) {
	// Test that identical strings always return the same canonical value
	// even after cache evictions
	cache = newLRUCache(2)

	// Get canonical form of a string
	s1 := String("consistency_test")

	// Add many other strings to force eviction
	for i := 0; i < 10; i++ {
		String(fmt.Sprintf("evict_string_%d", i))
	}

	// Get the same string again - should still be canonical
	s2 := String("consistency_test")

	// They should be identical (same pointer)
	if s1 != s2 {
		t.Errorf("Identical strings should return same canonical value: %p vs %p", &s1, &s2)
	}
}

func TestValueSliceOperations(t *testing.T) {
	cache = newLRUCache(100)

	// Test ValueSlice function with LRU cache
	values := []ldvalue.Value{
		ldvalue.String("test1"),
		ldvalue.String("test2"),
		ldvalue.String("test1"), // Duplicate
		ldvalue.Int(42),         // Non-string
		ldvalue.String("test3"),
	}

	result := ValueSlice(values)

	// Should process all values
	if len(result) != len(values) {
		t.Errorf("ValueSlice should preserve length: expected %d, got %d", len(values), len(result))
	}

	// String values should be interned
	if result[0].StringValue() != "test1" {
		t.Error("First string value should be interned correctly")
	}

	// Non-string values should be unchanged
	if result[3].IntValue() != 42 {
		t.Error("Non-string values should be unchanged")
	}

	// Check cache had some hits from duplicates
	hits, _, _, _, _ := CacheStats()
	if hits == 0 {
		t.Error("Expected cache hits from duplicate string values")
	}
}

func BenchmarkCacheUnderPressure(b *testing.B) {
	// Test performance under various cache pressure scenarios
	cache = newLRUCache(1000)

	b.Run("high_hit_rate", func(b *testing.B) {
		// 90% cache hits
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				var s string
				if i%10 < 9 {
					s = fmt.Sprintf("frequent-%d", i%100) // Frequent strings
				} else {
					s = fmt.Sprintf("unique-%d", i) // Unique strings
				}
				String(s)
				i++
			}
		})
	})

	b.Run("high_miss_rate", func(b *testing.B) {
		// 90% cache misses
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				var s string
				if i%10 < 1 {
					s = fmt.Sprintf("frequent-%d", i%10) // Infrequent repeated strings
				} else {
					s = fmt.Sprintf("unique-%d", i) // Mostly unique strings
				}
				String(s)
				i++
			}
		})
	})
}
