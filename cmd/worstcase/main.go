// Command worstcase reports the worst-case mutual-exclusivity violation
// probability from single-precision bucketing, for the agreed scope:
// single-SDK within-layer, merged/arbitrary integer weights, parameterized over
// N, using the bit-identical float32 replica in package worstcase.
package main

import (
	"fmt"
	"math"
	"math/rand"

	wc "github.com/launchdarkly/go-server-sdk-evaluation/v4/worstcase"
)

const u32 = 1.0 / (1 << 24) // float32 unit roundoff, 2^-24

// perEdgeWorst returns the maximum float32 divergence between two flags that
// reach the SAME exact edge E = m*g/100000 via different summation paths:
//   - "merged" flag: one block of weight m*g  (one rounding)
//   - "unmerged" flag: m slots of weight g    (m roundings, max accumulation)
// This is the largest realizable per-edge overlap width, swept over all edges.
func perEdgeWorst(N int) (bestFrac float64, bestM int) {
	g := 100000 / N
	var un float32 // running unmerged sum, accumulated incrementally (O(N) total)
	for m := 1; m <= N; m++ {
		un += float32(g) / 100000.0           // m slots of weight g
		merged := float32(m*g) / 100000.0     // one merged block of weight m*g
		d := math.Abs(float64(merged) - float64(un))
		if d > bestFrac {
			bestFrac, bestM = d, m
		}
	}
	return
}

func randomAssign(rng *rand.Rand, N, k int, leftoverFrac float64) []int {
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

func overlapOf(a []int, k, g int) float64 {
	return wc.OverlapMeasure(wc.LayerFlags(a, k, g))
}

// constructedSeed builds an imbalanced layer deliberately designed for high
// overlap: experiments 0 and 1 are interleaved with leftover slots (so their
// flags merge the region differently), everything else leftover. Used as a
// strong starting point for the annealing search regardless of K.
func constructedSeed(N, k int) []int {
	a := make([]int, N)
	for s := range a {
		a[s] = -1
	}
	for s := 0; s < N; s += 2 { // only even slots owned; odd slots stay leftover
		switch {
		case s < N*45/100:
			a[s] = 0
		case s < N*90/100 && k >= 2:
			a[s] = 1
		}
	}
	return a
}

func main() {
	fmt.Println("=== SDK-1537 worst-case mutual-exclusivity violation probability ===")
	fmt.Println("Scope: single-SDK, within-layer, merged weights. float32 arithmetic is a")
	fmt.Println("bit-identical replica of go-server-sdk-evaluation (validated by test).")
	fmt.Printf("float32 unit roundoff u = 2^-24 = %.3e\n\n", u32)

	for _, N := range []int{1000, 100000} {
		g := 100000 / N
		fmt.Printf("---------- N = %d partitions (slot weight g = %d of 100000) ----------\n", N, g)

		// (B) constructive per-edge worst case
		frac, m := perEdgeWorst(N)
		// analytical ceiling for a single edge reached via m unmerged terms: ~ m*u*E, E=m*g/1e5
		E := float64(m*g) / 100000.0
		ceil := float64(m) * u32 * E
		fmt.Printf("Per-edge worst divergence |merged - unmerged|:\n")
		fmt.Printf("  max = %.3e  at edge E = %.4f (m = %d unmerged slots)\n", frac, E, m)
		fmt.Printf("  = %.4f%% of one partition width (1/N = %.2e)\n", frac/(1.0/float64(N))*100, 1.0/float64(N))
		fmt.Printf("  analytical ceiling m*u*E = %.3e (ratio realized/ceiling = %.2f)\n\n", ceil, frac/ceil)

		// Layer total overlap for several experiment counts (N=1000 only; N=100000 per-edge only for cost)
		if N == 1000 {
			rng := rand.New(rand.NewSource(1537))
			fmt.Printf("Layer >=2 overlap measure (= P[violation] under uniform PRF):\n")
			fmt.Printf("  typical(rand): mean over %d random shuffles (the realistic design)\n", 200)
			fmt.Printf("  max(rand):     largest of those random shuffles\n")
			fmt.Printf("  worst-found:   highest overlap the search could construct -- a LOWER BOUND\n")
			fmt.Printf("                 on the true worst case, carried forward across K (non-decreasing)\n")
			fmt.Printf("  %-4s %-16s %-16s %-16s\n", "K", "typical(rand)", "max(rand)", "worst-found(LB)")
			var prevBest []int
			for _, k := range []int{2, 3, 5, 10, 20} {
				var sum, mx float64
				const R = 200
				for i := 0; i < R; i++ {
					o := overlapOf(randomAssign(rng, N, k, 0.5), k, g)
					sum += o
					if o > mx {
						mx = o
					}
				}
				// seeds: a constructed imbalanced layout + the embedded best from K-1
				seeds := [][]int{constructedSeed(N, k), prevBest}
				best, bestA := wc.SearchMaxOverlapHardened(int64(1000+k), N, k, g, 8, 6000, seeds)
				prevBest = bestA
				fmt.Printf("  %-4d %-16.3e %-16.3e %-16.3e\n", k, sum/R, mx, best)
			}
			fmt.Println()
		}
	}
	fmt.Println("Note: P is the per-context violation probability. For a layer serving M")
	fmt.Println("contexts, P(>=1 violation) = 1 - (1-P)^M.")
}
