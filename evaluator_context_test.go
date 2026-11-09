package evaluation

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/launchdarkly/go-sdk-common/v3/ldattr"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldreason"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	"github.com/launchdarkly/go-server-sdk-evaluation/v3/ldbuilders"
	"github.com/launchdarkly/go-server-sdk-evaluation/v3/ldmodel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingDataProvider serves stored flags and segments, returning the same pointer for a key every
// time (as a data store does), and records every request.
type recordingDataProvider struct {
	flags    map[string]*ldmodel.FeatureFlag
	segments map[string]*ldmodel.Segment
	calls    []string
}

func (p *recordingDataProvider) GetFeatureFlag(key string) *ldmodel.FeatureFlag {
	p.calls = append(p.calls, "flag:"+key)
	return p.flags[key]
}

func (p *recordingDataProvider) GetSegment(key string) *ldmodel.Segment {
	p.calls = append(p.calls, "segment:"+key)
	return p.segments[key]
}

type recordingBigSegmentProvider struct {
	members map[string]map[string]bool // context key -> segment ref -> included
	calls   []string
}

func (p *recordingBigSegmentProvider) GetMembership(contextKey string) (BigSegmentMembership, ldreason.BigSegmentsStatus) {
	p.calls = append(p.calls, "membership:"+contextKey)
	if m, ok := p.members[contextKey]; ok {
		return mapMembership(m), ldreason.BigSegmentsHealthy
	}
	return nil, ldreason.BigSegmentsStale
}

type mapMembership map[string]bool

func (m mapMembership) CheckMembership(segmentRef string) ldvalue.OptionalBool {
	if v, ok := m[segmentRef]; ok {
		return ldvalue.NewOptionalBool(v)
	}
	return ldvalue.OptionalBool{}
}

// contextEvalCorpus is a random environment: flags, segments, big segment memberships and contexts.
type contextEvalCorpus struct {
	flags    []*ldmodel.FeatureFlag
	data     *recordingDataProvider
	bigSegs  *recordingBigSegmentProvider
	contexts []ldcontext.Context
}

type contextEvalGen struct{ r *rand.Rand }

func (g contextEvalGen) n(k int) int    { return g.r.Intn(k) }
func (g contextEvalGen) p(pct int) bool { return g.r.Intn(100) < pct }
func (g contextEvalGen) kind() ldcontext.Kind {
	return []ldcontext.Kind{"", "user", "org", "device"}[g.n(4)]
}

func (g contextEvalGen) attr() ldattr.Ref {
	switch g.n(9) {
	case 0:
		return ldattr.NewRef("/address/city")
	case 1:
		return ldattr.NewLiteralRef("key")
	case 2:
		return ldattr.NewLiteralRef("score")
	case 3:
		return ldattr.NewLiteralRef("version")
	case 4:
		return ldattr.NewLiteralRef("created")
	case 5:
		return ldattr.NewLiteralRef("tags")
	case 6:
		return ldattr.NewLiteralRef("missing")
	default:
		return ldattr.NewLiteralRef("email")
	}
}

func (g contextEvalGen) clause(nSegs int) ldmodel.Clause {
	var c ldmodel.Clause
	switch g.n(12) {
	case 0:
		var keys []ldvalue.Value
		for i := 1 + g.n(2); i > 0; i-- {
			keys = append(keys, ldvalue.String(fmt.Sprintf("s%d", g.n(nSegs+1)))) // s<nSegs> does not exist
		}
		c = ldbuilders.SegmentMatchClause()
		c.Values = keys
	case 1:
		c = ldbuilders.ClauseRef(ldattr.NewLiteralRef(ldattr.KindAttr), ldmodel.OperatorIn,
			ldvalue.String("user"), ldvalue.String("org"))
	case 2:
		c = ldbuilders.ClauseRefWithKind(g.kind(), ldattr.NewRef("/a//b"), ldmodel.OperatorIn, ldvalue.String("x")) // bad ref
	default:
		op := []ldmodel.Operator{
			ldmodel.OperatorIn, ldmodel.OperatorStartsWith, ldmodel.OperatorEndsWith, ldmodel.OperatorContains,
			ldmodel.OperatorMatches, ldmodel.OperatorLessThan, ldmodel.OperatorGreaterThanOrEqual,
			ldmodel.OperatorBefore, ldmodel.OperatorAfter, ldmodel.OperatorSemVerEqual, ldmodel.OperatorSemVerLessThan,
		}[g.n(11)]
		values := map[ldmodel.Operator][]ldvalue.Value{
			ldmodel.OperatorIn:                 {ldvalue.String("u1"), ldvalue.String("Oslo"), ldvalue.Int(5), ldvalue.String("beta")},
			ldmodel.OperatorStartsWith:         {ldvalue.String("qa"), ldvalue.String("u")},
			ldmodel.OperatorEndsWith:           {ldvalue.String(".com"), ldvalue.String("lo")},
			ldmodel.OperatorContains:           {ldvalue.String("@ex"), ldvalue.String("s")},
			ldmodel.OperatorMatches:            {ldvalue.String("^u[0-9]$"), ldvalue.String("[")},
			ldmodel.OperatorLessThan:           {ldvalue.Int(3), ldvalue.Float64(4.5)},
			ldmodel.OperatorGreaterThanOrEqual: {ldvalue.Int(3)},
			ldmodel.OperatorBefore:             {ldvalue.String("2024-01-01T00:00:00Z"), ldvalue.Int(1700000000000)},
			ldmodel.OperatorAfter:              {ldvalue.String("2023-06-01T00:00:00Z")},
			ldmodel.OperatorSemVerEqual:        {ldvalue.String("1.2.3")},
			ldmodel.OperatorSemVerLessThan:     {ldvalue.String("2.0.0"), ldvalue.String("x")},
		}[op]
		c = ldbuilders.ClauseRefWithKind(g.kind(), g.attr(), op, values[:1+g.n(len(values))]...)
	}
	if g.p(20) {
		c = ldbuilders.Negate(c)
	}
	return c
}

func (g contextEvalGen) rollout(nVars int) ldmodel.VariationOrRollout {
	if g.p(50) {
		return ldmodel.VariationOrRollout{Variation: ldvalue.NewOptionalInt(g.n(nVars))}
	}
	r := ldmodel.Rollout{ContextKind: g.kind()}
	if g.p(40) {
		r.Kind = ldmodel.RolloutKindExperiment
		if g.p(50) {
			r.Seed = ldvalue.NewOptionalInt(g.n(1000))
		}
	} else if g.p(40) {
		r.BucketBy = g.attr()
	}
	left := 100000
	for v := 0; v < nVars && left > 0; v++ {
		w := g.n(left + 1)
		r.Variations = append(r.Variations, ldmodel.WeightedVariation{Variation: v, Weight: w, Untracked: g.p(20)})
		left -= w
	}
	return ldmodel.VariationOrRollout{Rollout: r}
}

func (g contextEvalGen) segment(i, nSegs int) *ldmodel.Segment {
	b := ldbuilders.NewSegmentBuilder(fmt.Sprintf("s%d", i)).Version(1).Salt("salt")
	if g.p(15) {
		b = b.Unbounded(true).Generation(1)
		if g.p(50) {
			b = b.UnboundedContextKind("org")
		}
	}
	if g.p(50) {
		b = b.Included(fmt.Sprintf("u%d", g.n(6)))
	}
	if g.p(30) {
		b = b.Excluded(fmt.Sprintf("u%d", g.n(6)))
	}
	if g.p(30) {
		b = b.IncludedContextKind("org", fmt.Sprintf("o%d", g.n(4)))
	}
	for r := g.n(3); r > 0; r-- {
		rb := ldbuilders.NewSegmentRuleBuilder().ID(fmt.Sprintf("sr%d", r))
		clauses := []ldmodel.Clause{g.clause(nSegs)}
		if clauses[0].Op == ldmodel.OperatorSegmentMatch && g.p(50) {
			clauses[0] = ldbuilders.Clause("email", ldmodel.OperatorEndsWith, ldvalue.String(".com"))
		}
		rb = rb.Clauses(clauses...)
		if g.p(40) {
			rb = rb.Weight(g.n(100001)).RolloutContextKind(g.kind())
			if g.p(30) {
				rb = rb.BucketByRef(g.attr())
			}
		}
		b = b.AddRule(rb)
	}
	s := b.Build()
	s.IsOverride = g.p(10)
	return &s
}

func (g contextEvalGen) flag(key string, prereqs []string, nSegs int) *ldmodel.FeatureFlag {
	nVars := 2 + g.n(2)
	vars := make([]ldvalue.Value, nVars)
	for i := range vars {
		vars[i] = ldvalue.Int(i)
	}
	b := ldbuilders.NewFlagBuilder(key).Version(1).On(g.p(90)).Salt("fsalt").Variations(vars...).
		OffVariation(g.n(nVars)).TrackEventsFallthrough(g.p(20))
	for _, pk := range prereqs {
		if g.p(30) {
			b = b.AddPrerequisite(pk, g.n(nVars))
		}
	}
	if g.p(30) {
		b = b.AddTarget(g.n(nVars), fmt.Sprintf("u%d", g.n(6)))
	}
	if g.p(20) {
		b = b.AddContextTarget("org", g.n(nVars), fmt.Sprintf("o%d", g.n(4)))
	}
	for r := g.n(4); r > 0; r-- {
		var clauses []ldmodel.Clause
		for c := 1 + g.n(3); c > 0; c-- {
			clauses = append(clauses, g.clause(nSegs))
		}
		b = b.AddRule(ldbuilders.NewRuleBuilder().ID(fmt.Sprintf("%s-r%d", key, r)).TrackEvents(g.p(20)).
			VariationOrRollout(g.rollout(nVars)).Clauses(clauses...))
	}
	f := b.Fallthrough(g.rollout(nVars)).Build()
	f.IsOverride = g.p(10)
	return &f
}

func (g contextEvalGen) context(i int) ldcontext.Context {
	user := ldcontext.NewBuilder(fmt.Sprintf("u%d", g.n(6))).
		SetString("email", []string{"qa+u@ex.com", "u@ex.org", ""}[g.n(3)]).
		SetInt("score", g.n(8)).
		SetString("version", []string{"1.2.3", "2.0.0-beta", "x"}[g.n(3)]).
		SetString("created", []string{"2023-12-31T23:00:00Z", "2024-02-01T00:00:00Z"}[g.n(2)]).
		SetValue("tags", ldvalue.ArrayOf(ldvalue.String("beta"), ldvalue.Int(5))).
		SetValue("address", ldvalue.ObjectBuild().SetString("city", "Oslo").Build()).
		Build()
	org := ldcontext.NewBuilder(fmt.Sprintf("o%d", g.n(4))).Kind("org").SetInt("score", g.n(8)).Build()
	device := ldcontext.NewBuilder(fmt.Sprintf("d%d", g.n(3))).Kind("device").SetString("email", "d@ex.com").Build()
	app := ldcontext.NewBuilder("a").Kind("app").Build()
	switch i % 4 {
	case 0:
		return user
	case 1:
		return org
	case 2:
		return ldcontext.NewMulti(user, org)
	default:
		return ldcontext.NewMulti(user, org, device, app) // more kinds than contextCache holds
	}
}

func newContextEvalCorpus(seed int64) contextEvalCorpus {
	g := contextEvalGen{r: rand.New(rand.NewSource(seed))} //nolint:gosec // deterministic test data
	nSegs := 4 + g.n(8)
	data := &recordingDataProvider{flags: map[string]*ldmodel.FeatureFlag{}, segments: map[string]*ldmodel.Segment{}}
	for i := 0; i < nSegs; i++ {
		s := g.segment(i, nSegs)
		data.segments[s.Key] = s
	}
	var c contextEvalCorpus
	var prereqs []string
	for i := 0; i < 12+g.n(20); i++ {
		f := g.flag(fmt.Sprintf("f%d", i), prereqs, nSegs)
		data.flags[f.Key] = f
		c.flags = append(c.flags, f)
		if g.p(40) {
			prereqs = append(prereqs, f.Key)
		}
	}
	bigSegs := &recordingBigSegmentProvider{members: map[string]map[string]bool{}}
	for i := 0; i < 6; i++ {
		if g.p(60) {
			m := map[string]bool{}
			for s := 0; s < nSegs; s++ {
				if g.p(30) {
					m[fmt.Sprintf("s%d.g1", s)] = g.p(70)
				}
			}
			bigSegs.members[fmt.Sprintf("u%d", i)] = m
		}
	}
	c.data, c.bigSegs = data, bigSegs
	for i := 0; i < 8; i++ {
		c.contexts = append(c.contexts, g.context(i))
	}
	return c
}

type prereqRecord struct {
	target, prereq string
	result         Result
}

// TestContextEvaluatorMatchesEvaluate compares a ContextEvaluator, evaluating every flag of a random
// environment for one context twice and in random order, with Evaluator.Evaluate: results,
// prerequisite events and data provider requests must be identical.
func TestContextEvaluatorMatchesEvaluate(t *testing.T) {
	for seed := int64(0); seed < 300; seed++ {
		c := newContextEvalCorpus(seed)
		e := NewEvaluatorWithOptions(c.data, EvaluatorOptionBigSegmentProvider(c.bigSegs),
			EvaluatorOptionEnableSecondaryKey(seed%2 == 0))
		order := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic test data
		for ci, ctx := range c.contexts {
			ce := NewContextEvaluator(e, ctx)
			for pass := 0; pass < 2; pass++ {
				for _, fi := range order.Perm(len(c.flags)) {
					flag := c.flags[fi]
					var wantEvents, gotEvents []prereqRecord
					c.data.calls, c.bigSegs.calls = nil, nil
					want := e.Evaluate(flag, ctx, func(ev PrerequisiteFlagEvent) {
						wantEvents = append(wantEvents, prereqRecord{ev.TargetFlagKey, ev.PrerequisiteFlag.Key, ev.PrerequisiteResult})
					})
					wantCalls, wantBig := c.data.calls, c.bigSegs.calls
					c.data.calls, c.bigSegs.calls = nil, nil
					got := ce.Evaluate(flag, func(ev PrerequisiteFlagEvent) {
						gotEvents = append(gotEvents, prereqRecord{ev.TargetFlagKey, ev.PrerequisiteFlag.Key, ev.PrerequisiteResult})
					})
					where := fmt.Sprintf("seed=%d context=%d pass=%d flag=%s", seed, ci, pass, flag.Key)
					require.Equal(t, want, got, where)
					require.Equal(t, wantEvents, gotEvents, where)
					require.Equal(t, wantCalls, c.data.calls, where)
					require.Equal(t, wantBig, c.bigSegs.calls, where)
				}
			}
		}
	}
}

// The random corpus must reach every path the comparison is meant to cover.
func TestContextEvaluatorCorpusCoverage(t *testing.T) {
	kinds := map[ldreason.EvalReasonKind]int{}
	var bigSegment, experiment, overrideAffected int
	for seed := int64(0); seed < 300; seed++ {
		c := newContextEvalCorpus(seed)
		e := NewEvaluatorWithOptions(c.data, EvaluatorOptionBigSegmentProvider(c.bigSegs))
		for _, ctx := range c.contexts {
			ce := NewContextEvaluator(e, ctx)
			for _, flag := range c.flags {
				r := ce.Evaluate(flag, nil)
				kinds[r.Detail.Reason.GetKind()]++
				if r.Detail.Reason.GetBigSegmentsStatus() != "" {
					bigSegment++
				}
				if r.Detail.Reason.IsInExperiment() {
					experiment++
				}
				if r.OverrideAffected {
					overrideAffected++
				}
			}
		}
	}
	for _, k := range []ldreason.EvalReasonKind{ldreason.EvalReasonOff, ldreason.EvalReasonTargetMatch,
		ldreason.EvalReasonRuleMatch, ldreason.EvalReasonFallthrough, ldreason.EvalReasonPrerequisiteFailed,
		ldreason.EvalReasonError} {
		assert.NotZero(t, kinds[k], "no %s results", k)
	}
	assert.NotZero(t, bigSegment, "no big segment results")
	assert.NotZero(t, experiment, "no experiment results")
	assert.NotZero(t, overrideAffected, "no override-affected results")
}

type otherEvaluator struct{ calls int }

func (o *otherEvaluator) Evaluate(*ldmodel.FeatureFlag, ldcontext.Context, PrerequisiteFlagEventRecorder) Result {
	o.calls++
	return Result{}
}

func TestContextEvaluatorDelegatesToOtherEvaluators(t *testing.T) {
	o := &otherEvaluator{}
	flag := ldbuilders.NewFlagBuilder("f").Build()
	NewContextEvaluator(o, ldcontext.New("u")).Evaluate(&flag, nil)
	assert.Equal(t, 1, o.calls)
}

// Contexts with more kinds, attributes and segments than contextCache holds are still evaluated
// exactly; the overflow is simply not cached.
func TestContextEvaluatorCacheOverflow(t *testing.T) {
	var kinds []ldcontext.Context
	for i := 0; i < cachedKinds+2; i++ {
		b := ldcontext.NewBuilder(fmt.Sprintf("k%d", i)).Kind(ldcontext.Kind(fmt.Sprintf("kind%d", i)))
		for a := 0; a < cachedAttrs+2; a++ {
			b.SetInt(fmt.Sprintf("a%d", a), a)
		}
		kinds = append(kinds, b.Build())
	}
	ctx := ldcontext.NewMulti(kinds...)
	data := &recordingDataProvider{flags: map[string]*ldmodel.FeatureFlag{}, segments: map[string]*ldmodel.Segment{}}
	fb := ldbuilders.NewFlagBuilder("f").On(true).Variations(ldvalue.Bool(false), ldvalue.Bool(true)).FallthroughVariation(0)
	for i := range kinds {
		for a := 0; a < cachedAttrs+2; a++ {
			fb = fb.AddRule(ldbuilders.NewRuleBuilder().ID(fmt.Sprintf("r%d-%d", i, a)).Variation(1).Clauses(
				ldbuilders.ClauseWithKind(ldcontext.Kind(fmt.Sprintf("kind%d", i)), fmt.Sprintf("a%d", a), ldmodel.OperatorIn, ldvalue.Int(-1))))
		}
	}
	for s := 0; s < cachedSegments+2; s++ {
		seg := ldbuilders.NewSegmentBuilder(fmt.Sprintf("s%d", s)).Included("nobody").Build()
		data.segments[seg.Key] = &seg
		fb = fb.AddRule(ldbuilders.NewRuleBuilder().ID(fmt.Sprintf("seg%d", s)).Variation(1).Clauses(
			ldbuilders.SegmentMatchClause(seg.Key)))
	}
	last := len(kinds) - 1
	fb = fb.AddRule(ldbuilders.NewRuleBuilder().ID("last").Variation(1).Clauses(
		ldbuilders.ClauseWithKind(ldcontext.Kind(fmt.Sprintf("kind%d", last)), fmt.Sprintf("a%d", cachedAttrs+1),
			ldmodel.OperatorIn, ldvalue.Int(cachedAttrs+1))))
	flag := fb.Build()
	e := NewEvaluator(data)
	ce := NewContextEvaluator(e, ctx)
	want := e.Evaluate(&flag, ctx, nil)
	assert.Equal(t, want, ce.Evaluate(&flag, nil))
	assert.Equal(t, want, ce.Evaluate(&flag, nil))
	assert.Equal(t, ldreason.NewEvalReasonRuleMatch(len(flag.Rules)-1, "last"), want.Detail.Reason)
}
