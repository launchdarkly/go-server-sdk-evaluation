package evaluation_test

// Validation: the worstcase package computes overlap from a float32 replica of
// the bucketing arithmetic. This test proves that replica matches the REAL
// production engine (Evaluator.Evaluate) by driving actual experiment flags that
// share a layer seed and measuring how often a context is "in experiment" for
// two flags at once. The empirical rate must agree with worstcase.OverlapMeasure.

import (
	"math"
	"strconv"
	"testing"

	eval "github.com/launchdarkly/go-server-sdk-evaluation/v4"
	"github.com/launchdarkly/go-server-sdk-evaluation/v4/ldbuilders"
	"github.com/launchdarkly/go-server-sdk-evaluation/v4/ldmodel"
	wc "github.com/launchdarkly/go-server-sdk-evaluation/v4/worstcase"

	"github.com/launchdarkly/go-sdk-common/v4/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v4/ldvalue"
)

type nilProvider struct{}

func (nilProvider) GetFeatureFlag(string) *ldmodel.FeatureFlag { return nil }
func (nilProvider) GetSegment(string) *ldmodel.Segment         { return nil }

// buildExperimentFlag turns a worstcase weight array into a real experiment flag
// sharing the given layer seed.
func buildExperimentFlag(key string, seed int, arr []wc.WV) ldmodel.FeatureFlag {
	buckets := make([]ldmodel.WeightedVariation, len(arr))
	for i, w := range arr {
		if w.Untracked {
			buckets[i] = ldbuilders.BucketUntracked(w.Variation, w.Weight)
		} else {
			buckets[i] = ldbuilders.Bucket(w.Variation, w.Weight)
		}
	}
	return ldbuilders.NewFlagBuilder(key).
		On(true).
		Variations(ldvalue.Bool(false), ldvalue.Bool(true)).
		Fallthrough(ldbuilders.Experiment(ldvalue.NewOptionalInt(seed), buckets...)).
		Build()
}

func TestReplicaMatchesRealEngineOverlap(t *testing.T) {
	const (
		N    = 1000
		g    = 100000 / N
		k    = 2
		seed = 42      // shared layer seed: every flag buckets the same context identically
		M    = 3000000 // contexts sampled through the real engine
	)

	// Find a high-overlap layer config so the (small) overlap rate is measurable.
	predicted, assign := wc.SearchMaxOverlap(1537, N, k, g, 12, 1500)
	if predicted < 1e-4 {
		t.Fatalf("search did not find a high-overlap config (predicted=%.3e); test would be statistically weak", predicted)
	}
	arrays := wc.LayerFlags(assign, k, g)

	// Sanity: each flag's weights sum to 100000 (valid rollout).
	for i, arr := range arrays {
		sum := 0
		for _, w := range arr {
			sum += w.Weight
		}
		if sum != 100000 {
			t.Fatalf("flag %d weights sum to %d, want 100000", i, sum)
		}
	}

	flags := make([]ldmodel.FeatureFlag, k)
	for i := range arrays {
		flags[i] = buildExperimentFlag("layer-exp-"+strconv.Itoa(i), seed, arrays[i])
	}

	evaluator := eval.NewEvaluator(nilProvider{})
	overlap := 0
	for i := 0; i < M; i++ {
		ctx := ldcontext.New("ctx-" + strconv.Itoa(i))
		inExpCount := 0
		for j := range flags {
			r := evaluator.Evaluate(&flags[j], ctx, nil)
			if r.IsExperiment {
				inExpCount++
			}
		}
		if inExpCount >= 2 {
			overlap++ // this context is in two experiments at once: an exclusivity violation
		}
	}
	empirical := float64(overlap) / float64(M)

	t.Logf("predicted (replica continuous measure) = %.4e", predicted)
	t.Logf("empirical (real engine, %d contexts)    = %.4e (%d violations)", M, empirical, overlap)

	if overlap == 0 {
		t.Fatalf("real engine produced zero violations; replica predicted %.3e", predicted)
	}
	// Agreement within 15% (MC noise + float32 b-quantization in the real engine
	// vs the continuous measure). A match here proves the replica's float32
	// arithmetic reproduces the production engine, so the reported worst-case
	// probabilities reflect real Go SDK behavior.
	relErr := math.Abs(empirical-predicted) / predicted
	if relErr > 0.15 {
		t.Fatalf("replica and real engine disagree: predicted=%.4e empirical=%.4e relErr=%.1f%%",
			predicted, empirical, relErr*100)
	}
}
