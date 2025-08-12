# Optional JSON interning (Go 1.23+)

This repo supports an optional build-time feature to reduce steady-state heap
and allocations by deduplicating repeated JSON strings via Go 1.23's `unique`
package. It is off by default and opt-in via a build tag.

- Default build: no interning (minimum Go remains 1.18)
- Opt-in interning: requires Go 1.23+

Enable interning:

- Build: `go build -tags launchdarkly_intern_json ./...`
- Test/bench: `go test -tags launchdarkly_intern_json ./...`

Fail-fast on older Go:

- The unique-backed code uses `//go:build go1.23 && launchdarkly_intern_json`
- A guard file `//go:build launchdarkly_intern_json && !go1.23` produces a
  compile-time error if the tag is used on Go < 1.23

How interning works:

- Only strings with high deduplication value are interned: operators (like "in", "matches"), 
  context kinds (like "user", "device"), and attribute strings
- Uses an LRU cache that retains `unique.Handle`s for frequently accessed strings
- Cache size defaults to 8192 entries, configurable at runtime via `ldmodel.SetStringInterningCacheSize(size)`
- Provides deduplication both within GC windows and across GCs for cached entries
- Cache can be disabled entirely by setting size to 0 or negative values

Makefile shortcuts:

- `make benchmarks-nointern` — default suite, no tag
- `make benchmarks-intern-json` — runs with `-tags launchdarkly_intern_json`

Runtime configuration:

```go
import "github.com/launchdarkly/go-server-sdk-evaluation/v3/ldmodel"

// Set cache size to 1000 entries
ldmodel.SetStringInterningCacheSize(1000)

// Disable interning entirely
ldmodel.SetStringInterningCacheSize(0)
```

Notes:

- This feature does not modify the public API beyond the optional cache control function
- Interning is applied selectively to strings with proven deduplication benefits
- If issues arise, interning can be disabled at runtime without rebuilding

