package evaluation

import (
	"fmt"
	"testing"

	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	"github.com/launchdarkly/go-server-sdk-evaluation/v3/ldbuilders"
	"github.com/launchdarkly/go-server-sdk-evaluation/v3/ldmodel"
)

// contextBenchEnv is an environment of many flags whose rules test a few context attributes and
// reference a few shared segments, as when a server evaluates every flag for one client.
func contextBenchEnv(nFlags int) ([]*ldmodel.FeatureFlag, *recordingDataProvider) {
	data := &recordingDataProvider{flags: map[string]*ldmodel.FeatureFlag{}, segments: map[string]*ldmodel.Segment{}}
	for s := 0; s < 4; s++ {
		seg := ldbuilders.NewSegmentBuilder(fmt.Sprintf("s%d", s)).Included(fmt.Sprintf("other%d", s)).
			AddRule(ldbuilders.NewSegmentRuleBuilder().ID("r").Clauses(
				ldbuilders.Clause("email", ldmodel.OperatorEndsWith, ldvalue.String(fmt.Sprintf("@s%d.com", s))))).
			Build()
		data.segments[seg.Key] = &seg
	}
	flags := make([]*ldmodel.FeatureFlag, nFlags)
	for i := range flags {
		f := ldbuilders.NewFlagBuilder(fmt.Sprintf("f%d", i)).On(true).
			Variations(ldvalue.Bool(false), ldvalue.Bool(true)).FallthroughVariation(0).
			AddRule(ldbuilders.NewRuleBuilder().ID("seg").Variation(1).Clauses(
				ldbuilders.SegmentMatchClause(fmt.Sprintf("s%d", i%4)))).
			AddRule(ldbuilders.NewRuleBuilder().ID("country").Variation(1).Clauses(
				ldbuilders.Clause("country", ldmodel.OperatorIn, ldvalue.String("nz"), ldvalue.String("ca")))).
			AddRule(ldbuilders.NewRuleBuilder().ID("plan").Variation(1).Clauses(
				ldbuilders.Clause("plan", ldmodel.OperatorStartsWith, ldvalue.String("enterprise")))).
			Build()
		flags[i] = &f
	}
	return flags, data
}

func contextBenchContext() ldcontext.Context {
	return ldcontext.NewMulti(
		ldcontext.NewBuilder("user-key").SetString("email", "someone@example.com").
			SetString("country", "us").SetString("plan", "free").Build(),
		ldcontext.NewBuilder("org-key").Kind("org").SetString("plan", "free").Build(),
	)
}

func BenchmarkEvaluateFlagsForOneContext(b *testing.B) {
	for _, nFlags := range []int{10, 100, 500} {
		flags, data := contextBenchEnv(nFlags)
		e := NewEvaluator(data)
		ctx := contextBenchContext()
		b.Run(fmt.Sprintf("Evaluate/flags=%d", nFlags), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				data.calls = data.calls[:0]
				for _, f := range flags {
					_ = e.Evaluate(f, ctx, nil)
				}
			}
		})
		b.Run(fmt.Sprintf("ContextEvaluator/flags=%d", nFlags), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				data.calls = data.calls[:0]
				ce := NewContextEvaluator(e, ctx)
				for _, f := range flags {
					_ = ce.Evaluate(f, nil)
				}
			}
		})
	}
}
