// Package worstcase analyzes the worst-case probability that single-precision
// bucketing causes two experiments in the same mutual-exclusion layer to both
// classify a context as "in experiment" (an exclusivity violation).
//
// Scope (agreed with the investigation owner):
//   - Single-SDK, within-layer: the SAME context hashes to the SAME bucket value
//     across every flag in the layer (shared layer seed). Each experiment is its
//     own flag whose rollout is the layer's slot layout, MERGED per that flag's
//     tracked/untracked labeling. Overlap arises purely because different flags
//     sum different (merged) weight arrays to reach the same partition edge, and
//     float32 addition is not associative.
//   - Merged / arbitrary integer weights (the proposed payload optimization).
//
// The bucketing arithmetic here is a bit-for-bit replica of the production Go
// evaluation engine (go-server-sdk-evaluation v4):
//
//	evaluator_bucketing.go: bucket := float32(intVal) / longScale
//	evaluator.go:           var sum float32; sum += float32(bucket.Weight) / 100000.0; if bucketVal < sum {...}
//
// A separate validation test cross-checks this replica against the real
// Evaluator.Evaluate, so the measures reported here reflect production behavior.
package worstcase

import (
	"math"
	"math/rand"
	"sort"
)

// WV mirrors ldmodel.WeightedVariation: an integer weight (of 100000) plus
// whether the bucket is untracked (served, but NOT counted as in-experiment).
type WV struct {
	Variation int
	Weight    int
	Untracked bool
}

// Boundaries replicates the engine's float32 running sum exactly. boundaries[i]
// is the cumulative float32 sum through bucket i; a context value v is assigned
// to the first bucket i with v < boundaries[i].
func Boundaries(ws []WV) []float32 {
	out := make([]float32, len(ws))
	var sum float32
	for i, w := range ws {
		sum += float32(w.Weight) / 100000.0 // bit-identical to evaluator.go:410
		out[i] = sum
	}
	return out
}

// TrackedIntervals returns the disjoint [lo,hi) sub-intervals of [0,1] that this
// flag classifies as in-experiment (tracked), using the engine's `v < sum`
// semantics and its last-bucket fallthrough for v past the final boundary.
// Boundaries are promoted to float64 for exact measure arithmetic.
func TrackedIntervals(ws []WV) [][2]float64 {
	b := Boundaries(ws)
	var out [][2]float64
	var lo float64
	for i, w := range ws {
		hi := float64(b[i])
		if i == len(ws)-1 {
			hi = 1.0 // fallthrough: the last bucket absorbs everything up to 1.0
		}
		if !w.Untracked && hi > lo {
			out = append(out, [2]float64{lo, hi})
		}
		lo = float64(b[i])
	}
	return mergeAdjacent(out)
}

func mergeAdjacent(ivs [][2]float64) [][2]float64 {
	if len(ivs) == 0 {
		return ivs
	}
	sort.Slice(ivs, func(i, j int) bool { return ivs[i][0] < ivs[j][0] })
	out := ivs[:1]
	for _, iv := range ivs[1:] {
		last := &out[len(out)-1]
		if iv[0] <= last[1] {
			if iv[1] > last[1] {
				last[1] = iv[1]
			}
		} else {
			out = append(out, iv)
		}
	}
	return out
}

// OverlapMeasure returns the total length of [0,1] where two or more flags
// classify the context as in-experiment simultaneously. Under a uniform PRF
// this equals the per-context probability of a mutual-exclusivity violation.
// Implemented as an event sweep: each flag's tracked intervals are disjoint, so
// a point's active-count is the number of flags tracking it; we accumulate the
// length where that count is >= 2.
func OverlapMeasure(flags [][]WV) float64 {
	type ev struct {
		x float64
		d int
	}
	var events []ev
	for _, f := range flags {
		for _, iv := range TrackedIntervals(f) {
			events = append(events, ev{iv[0], +1}, ev{iv[1], -1})
		}
	}
	if len(events) == 0 {
		return 0
	}
	sort.Slice(events, func(i, j int) bool { return events[i].x < events[j].x })
	var total float64
	active := 0
	prevX := events[0].x
	for i := 0; i < len(events); {
		x := events[i].x
		if x > prevX {
			if active >= 2 {
				total += x - prevX
			}
			prevX = x
		}
		for i < len(events) && events[i].x == x {
			active += events[i].d
			i++
		}
	}
	return total
}

// RandomAssign produces a random slot->experiment assignment: each of N slots is
// leftover (-1) with probability leftoverFrac, else a uniformly random experiment
// in [0,k). This models the shuffled bitwise layer design.
func RandomAssign(rng *rand.Rand, N, k int, leftoverFrac float64) []int {
	a := make([]int, N)
	for s := range a {
		if rng.Float64() < leftoverFrac {
			a[s] = -1
		} else {
			a[s] = rng.Intn(k)
		}
	}
	return a
}

// SearchMaxOverlap hill-climbs slot assignments to maximize the >=2 overlap
// measure, returning the worst (largest) overlap found and the assignment.
func SearchMaxOverlap(seed int64, N, k, g, restarts, iters int) (float64, []int) {
	rng := rand.New(rand.NewSource(seed))
	best := 0.0
	var bestA []int
	for r := 0; r < restarts; r++ {
		a := RandomAssign(rng, N, k, 0.5)
		cur := OverlapMeasure(LayerFlags(a, k, g))
		for it := 0; it < iters; it++ {
			s := rng.Intn(N)
			old := a[s]
			nv := rng.Intn(k + 1)
			if nv == k {
				nv = -1
			}
			if nv == old {
				continue
			}
			a[s] = nv
			cand := OverlapMeasure(LayerFlags(a, k, g))
			if cand >= cur {
				cur = cand
			} else {
				a[s] = old
			}
		}
		if cur > best {
			best = cur
			bestA = append([]int(nil), a...)
		}
	}
	return best, bestA
}

func cloneAssign(a []int) []int { return append([]int(nil), a...) }

// anneal runs simulated annealing from a starting assignment, maximizing the
// >=2 overlap measure. Moves: single-slot flip, contiguous block set, and
// evacuate-an-experiment. Returns the best assignment and its overlap.
func anneal(rng *rand.Rand, start []int, N, k, g, iters int) (float64, []int) {
	a := cloneAssign(start)
	cur := OverlapMeasure(LayerFlags(a, k, g))
	best, bestA := cur, cloneAssign(a)
	const t0, t1 = 2e-4, 1e-9
	owner := func() int {
		v := rng.Intn(k + 1)
		if v == k {
			return -1
		}
		return v
	}
	for it := 0; it < iters; it++ {
		T := t0 * math.Pow(t1/t0, float64(it)/float64(iters))
		trial := cloneAssign(a)
		switch p := rng.Float64(); {
		case p < 0.6: // single-slot flip
			trial[rng.Intn(N)] = owner()
		case p < 0.9: // contiguous block set (creates/breaks long merge runs)
			s := rng.Intn(N)
			l := 1 + rng.Intn(N/8+1)
			o := owner()
			for i := s; i < s+l && i < N; i++ {
				trial[i] = o
			}
		default: // evacuate one experiment to leftover
			ev := rng.Intn(k)
			for i := range trial {
				if trial[i] == ev {
					trial[i] = -1
				}
			}
		}
		cand := OverlapMeasure(LayerFlags(trial, k, g))
		if d := cand - cur; d >= 0 || rng.Float64() < math.Exp(d/T) {
			a, cur = trial, cand
			if cand > best {
				best, bestA = cand, cloneAssign(trial)
			}
		}
	}
	return best, bestA
}

// greedyPolish does strict-improve single/block flips from a starting point.
// Annealing explores; this exploits. Running it on the annealed result (and on
// raw seeds) reliably recovers the strong local optima a plain hill-climb finds.
func greedyPolish(rng *rand.Rand, start []int, N, k, g, iters int) (float64, []int) {
	a := cloneAssign(start)
	cur := OverlapMeasure(LayerFlags(a, k, g))
	owner := func() int {
		v := rng.Intn(k + 1)
		if v == k {
			return -1
		}
		return v
	}
	for it := 0; it < iters; it++ {
		trial := cloneAssign(a)
		if rng.Float64() < 0.7 {
			trial[rng.Intn(N)] = owner()
		} else {
			s := rng.Intn(N)
			l := 1 + rng.Intn(N/8+1)
			o := owner()
			for i := s; i < s+l && i < N; i++ {
				trial[i] = o
			}
		}
		if cand := OverlapMeasure(LayerFlags(trial, k, g)); cand >= cur {
			a, cur = trial, cand
		}
	}
	return cur, a
}

// SearchMaxOverlapHardened maximizes the >=2 overlap measure using simulated
// annealing followed by a greedy polish, from each provided seed assignment plus
// random restarts. It returns a lower bound on the true worst-case overlap (the
// best configuration found over all starts and both methods).
func SearchMaxOverlapHardened(seed int64, N, k, g, restarts, iters int, seeds [][]int) (float64, []int) {
	rng := rand.New(rand.NewSource(seed))
	best := 0.0
	var bestA []int
	consider := func(o float64, a []int) {
		if o > best {
			best, bestA = o, a
		}
	}
	starts := make([][]int, 0, restarts)
	for _, s := range seeds {
		if s != nil {
			starts = append(starts, s)
		}
	}
	for len(starts) < restarts {
		starts = append(starts, RandomAssign(rng, N, k, 0.5))
	}
	for _, s := range starts {
		oA, aA := anneal(rng, s, N, k, g, iters)
		consider(oA, aA)
		oG, aG := greedyPolish(rng, aA, N, k, g, iters/2) // exploit the annealed result
		consider(oG, aG)
		oS, aS := greedyPolish(rng, s, N, k, g, iters/2) // and the raw seed directly
		consider(oS, aS)
	}
	return best, bestA
}

// LayerFlags builds one merged weight array per experiment from a slot
// assignment. There are len(assign) equal slots of weight g each (g = 100000/N).
// assign[s] is the experiment index that owns slot s, or -1 for leftover/control
// (untracked pool). k is the number of experiments. Within flag X, X's slots are
// tracked (variation 1) and all other slots are the untracked default
// (variation 0); adjacent same-label slots are MERGED, which is what makes each
// flag's summation path differ.
func LayerFlags(assign []int, k, g int) [][]WV {
	flags := make([][]WV, k)
	for x := 0; x < k; x++ {
		var arr []WV
		runLen := 0
		runTracked := false
		flush := func() {
			if runLen == 0 {
				return
			}
			v, unt := 0, true
			if runTracked {
				v, unt = 1, false
			}
			arr = append(arr, WV{Variation: v, Weight: runLen * g, Untracked: unt})
		}
		for _, owner := range assign {
			tracked := owner == x
			if runLen > 0 && tracked != runTracked {
				flush()
				runLen = 0
			}
			runTracked = tracked
			runLen++
		}
		flush()
		flags[x] = arr
	}
	return flags
}
