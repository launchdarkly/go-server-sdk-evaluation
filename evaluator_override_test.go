package evaluation

import (
	"testing"

	"github.com/launchdarkly/go-sdk-common/v4/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v4/ldreason"
	"github.com/launchdarkly/go-sdk-common/v4/ldvalue"
	"github.com/launchdarkly/go-server-sdk-evaluation/v4/ldbuilders"
	"github.com/launchdarkly/go-server-sdk-evaluation/v4/ldmodel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var overrideTestContext = ldcontext.New("userkey")

// assertOverrideAffected checks the reason indicator and the Result scalar together. Both report the
// same marking. The scalar is what event generation uses, so it must not lag behind the reason.
func assertOverrideAffected(t *testing.T, expected bool, result Result) {
	t.Helper()
	assert.Equal(t, expected, result.Detail.Reason.IsOverrideAffected(), "reason indicator")
	assert.Equal(t, expected, result.OverrideAffected, "Result.OverrideAffected")
}

// makeOverrideTestFlag builds a flag that is on and serves variation 1 ("on") by fallthrough, with an
// optional list of prerequisites that must each serve variation 1.
func makeOverrideTestFlag(key string, prereqKeys ...string) ldmodel.FeatureFlag {
	b := ldbuilders.NewFlagBuilder(key).On(true).FallthroughVariation(1).OffVariation(0).
		Variations(ldvalue.String("off"), ldvalue.String("on"))
	for _, prereqKey := range prereqKeys {
		b.AddPrerequisite(prereqKey, 1)
	}
	return b.Build()
}

func requirePrereqRecord(t *testing.T, sink *prereqEventSink, prereqKey string) Result {
	t.Helper()
	for _, e := range sink.events {
		if e.PrerequisiteFlag.Key == prereqKey {
			return e.PrerequisiteResult
		}
	}
	require.Failf(t, "missing prerequisite record", "no record for prerequisite %q", prereqKey)
	return Result{}
}

func TestOverrideFlagMarksEvaluationAsOverrideAffected(t *testing.T) {
	t.Run("off", func(t *testing.T) {
		flag := ldbuilders.NewFlagBuilder("feature").On(false).OffVariation(0).
			Variations(ldvalue.String("off"), ldvalue.String("on")).Build()
		flag.IsOverride = true

		result := basicEvaluator().Evaluate(&flag, overrideTestContext, nil)

		assert.Equal(t, ldreason.EvalReasonOff, result.Detail.Reason.GetKind())
		assert.Equal(t, ldvalue.String("off"), result.Detail.Value)
		assertOverrideAffected(t, true, result)
	})

	t.Run("fallthrough", func(t *testing.T) {
		flag := makeOverrideTestFlag("feature")
		flag.IsOverride = true

		result := basicEvaluator().Evaluate(&flag, overrideTestContext, nil)

		assert.Equal(t, ldreason.EvalReasonFallthrough, result.Detail.Reason.GetKind())
		assert.Equal(t, ldvalue.String("on"), result.Detail.Value)
		assertOverrideAffected(t, true, result)
	})

	t.Run("rule match", func(t *testing.T) {
		flag := ldbuilders.NewFlagBuilder("feature").On(true).FallthroughVariation(0).
			AddRule(ldbuilders.NewRuleBuilder().ID("rule-id").Variation(1).
				Clauses(makeClauseToMatchContext(overrideTestContext))).
			Variations(ldvalue.String("off"), ldvalue.String("on")).Build()
		flag.IsOverride = true

		result := basicEvaluator().Evaluate(&flag, overrideTestContext, nil)

		assert.Equal(t, ldreason.EvalReasonRuleMatch, result.Detail.Reason.GetKind())
		assert.Equal(t, ldvalue.String("on"), result.Detail.Value)
		assertOverrideAffected(t, true, result)
	})
}

// A plain LaunchDarkly evaluation reads no override definition, so nothing marks it.
func TestPlainEvaluationIsNotOverrideAffected(t *testing.T) {
	t.Run("flag alone", func(t *testing.T) {
		flag := ldbuilders.NewFlagBuilder("feature").On(false).OffVariation(0).
			Variations(ldvalue.String("off"), ldvalue.String("on")).Build()

		result := basicEvaluator().Evaluate(&flag, overrideTestContext, nil)

		assert.Equal(t, ldreason.EvalReasonOff, result.Detail.Reason.GetKind())
		assertOverrideAffected(t, false, result)
	})

	t.Run("flag with prerequisite and segment", func(t *testing.T) {
		segment := ldbuilders.NewSegmentBuilder("segment").Included(overrideTestContext.Key()).Build()
		prereq := makeBooleanFlagToMatchAnyOfSegments(segment.Key)
		prereq.Key = "prereq"
		flag := makeOverrideTestFlag("feature", "prereq")

		evaluator := NewEvaluator(basicDataProvider().withStoredFlags(prereq).withStoredSegments(segment))
		sink := prereqEventSink{}
		result := evaluator.Evaluate(&flag, overrideTestContext, sink.record)

		assert.Equal(t, ldreason.EvalReasonFallthrough, result.Detail.Reason.GetKind())
		assert.Equal(t, ldvalue.String("on"), result.Detail.Value)
		assertOverrideAffected(t, false, result)
		assertOverrideAffected(t, false, requirePrereqRecord(t, &sink, "prereq"))
	})
}

// An evaluation that fails is still marked when it read an override definition. The caller gets the
// default value with an error reason, and that result carries the marking.
func TestOverrideFlagErrorResultIsOverrideAffected(t *testing.T) {
	t.Run("malformed flag", func(t *testing.T) {
		flag := ldbuilders.NewFlagBuilder("feature").On(true).FallthroughVariation(99).
			Variations(ldvalue.String("off"), ldvalue.String("on")).Build()
		flag.IsOverride = true

		result := basicEvaluator().Evaluate(&flag, overrideTestContext, nil)

		assert.Equal(t, ldreason.EvalReasonError, result.Detail.Reason.GetKind())
		assert.Equal(t, ldreason.EvalErrorMalformedFlag, result.Detail.Reason.GetErrorKind())
		assert.Equal(t, ldvalue.Null(), result.Detail.Value)
		assertOverrideAffected(t, true, result)
	})

	t.Run("prerequisite cycle through an override flag", func(t *testing.T) {
		// feature -> prereq -> feature; only prereq is an override
		prereq := makeOverrideTestFlag("prereq", "feature")
		prereq.IsOverride = true
		flag := makeOverrideTestFlag("feature", "prereq")

		evaluator := NewEvaluator(basicDataProvider().withStoredFlags(flag, prereq))
		sink := prereqEventSink{}
		result := evaluator.Evaluate(&flag, overrideTestContext, sink.record)

		assert.Equal(t, ldreason.EvalReasonError, result.Detail.Reason.GetKind())
		assert.Equal(t, ldreason.EvalErrorMalformedFlag, result.Detail.Reason.GetErrorKind())
		assertOverrideAffected(t, true, result)
		assert.Empty(t, sink.events)
	})

	t.Run("invalid context", func(t *testing.T) {
		flag := makeOverrideTestFlag("feature")
		flag.IsOverride = true

		result := basicEvaluator().Evaluate(&flag, ldcontext.New(""), nil)

		assert.Equal(t, ldreason.EvalReasonError, result.Detail.Reason.GetKind())
		assert.Equal(t, ldreason.EvalErrorUserNotSpecified, result.Detail.Reason.GetErrorKind())
		assertOverrideAffected(t, true, result)
	})
}

func TestOverridePrerequisiteMarksPrerequisiteRecordAndTopLevel(t *testing.T) {
	prereq := makeOverrideTestFlag("prereq")
	prereq.IsOverride = true
	flag := makeOverrideTestFlag("feature", "prereq")

	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(prereq))
	sink := prereqEventSink{}
	result := evaluator.Evaluate(&flag, overrideTestContext, sink.record)

	assert.Equal(t, ldreason.EvalReasonFallthrough, result.Detail.Reason.GetKind())
	assertOverrideAffected(t, true, result)

	record := requirePrereqRecord(t, &sink, "prereq")
	assert.Equal(t, ldreason.EvalReasonFallthrough, record.Detail.Reason.GetKind())
	assertOverrideAffected(t, true, record)
}

// The marking propagates upward only. The top-level flag's own marker does not leak into the record
// of a prerequisite whose subtree read no override definition.
func TestOverrideFlagDoesNotMarkUnaffectedPrerequisiteRecord(t *testing.T) {
	prereq := makeOverrideTestFlag("prereq")
	flag := makeOverrideTestFlag("feature", "prereq")
	flag.IsOverride = true

	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(prereq))
	sink := prereqEventSink{}
	result := evaluator.Evaluate(&flag, overrideTestContext, sink.record)

	assertOverrideAffected(t, true, result)
	assertOverrideAffected(t, false, requirePrereqRecord(t, &sink, "prereq"))
}

func TestOverridePrerequisiteAtDepthTwoMarksAllAffectedScopes(t *testing.T) {
	prereq2 := makeOverrideTestFlag("prereq2")
	prereq2.IsOverride = true
	prereq1 := makeOverrideTestFlag("prereq1", "prereq2")
	flag := makeOverrideTestFlag("feature", "prereq1")

	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(prereq1, prereq2))
	sink := prereqEventSink{}
	result := evaluator.Evaluate(&flag, overrideTestContext, sink.record)

	assert.Equal(t, ldreason.EvalReasonFallthrough, result.Detail.Reason.GetKind())
	assertOverrideAffected(t, true, result)

	// The nested record is produced first, during the evaluation of prereq1.
	require.Len(t, sink.events, 2)
	assert.Equal(t, "prereq2", sink.events[0].PrerequisiteFlag.Key)
	assertOverrideAffected(t, true, sink.events[0].PrerequisiteResult)
	assert.Equal(t, "prereq1", sink.events[1].PrerequisiteFlag.Key)
	assertOverrideAffected(t, true, sink.events[1].PrerequisiteResult)
}

// Flag a has prerequisites b and c. Only d, a prerequisite of b, is an override. The marking reaches
// a, b, and d. It does not reach the sibling c, and the plain segments s1 and s2 mark nothing.
func TestUnaffectedSiblingPrerequisiteRecordStaysUnmarked(t *testing.T) {
	s1 := ldbuilders.NewSegmentBuilder("s1").Included(overrideTestContext.Key()).Build()
	s2 := ldbuilders.NewSegmentBuilder("s2").Included(overrideTestContext.Key()).Build()
	d := makeOverrideTestFlag("d")
	d.IsOverride = true
	b := makeOverrideTestFlag("b", "d")
	c := makeBooleanFlagToMatchAnyOfSegments(s1.Key)
	c.Key = "c"
	a := ldbuilders.NewFlagBuilder("a").On(true).FallthroughVariation(0).OffVariation(0).
		AddPrerequisite("b", 1).AddPrerequisite("c", 1).
		AddRule(ldbuilders.NewRuleBuilder().ID("rule-s2").Variation(1).
			Clauses(ldbuilders.SegmentMatchClause(s2.Key))).
		Variations(ldvalue.String("off"), ldvalue.String("on")).Build()

	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(b, c, d).withStoredSegments(s1, s2))
	sink := prereqEventSink{}
	result := evaluator.Evaluate(&a, overrideTestContext, sink.record)

	assert.Equal(t, ldreason.EvalReasonRuleMatch, result.Detail.Reason.GetKind())
	assert.Equal(t, ldvalue.String("on"), result.Detail.Value)
	assertOverrideAffected(t, true, result)

	require.Len(t, sink.events, 3)
	assertOverrideAffected(t, true, requirePrereqRecord(t, &sink, "d"))
	assertOverrideAffected(t, true, requirePrereqRecord(t, &sink, "b"))
	assertOverrideAffected(t, false, requirePrereqRecord(t, &sink, "c"))
}

func TestOverrideSegmentReferencedByFlagRuleMarksEvaluation(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segment").Included(overrideTestContext.Key()).Build()
	segment.IsOverride = true
	flag := makeBooleanFlagToMatchAnyOfSegments(segment.Key)

	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	result := evaluator.Evaluate(&flag, overrideTestContext, nil)

	assert.Equal(t, ldreason.EvalReasonRuleMatch, result.Detail.Reason.GetKind())
	assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
	assertOverrideAffected(t, true, result)
}

// A read is enough to mark the evaluation. The segment does not need to match: the non-match shaped
// the outcome, and a negated clause turns the non-match into a match.
func TestOverrideSegmentReadWithoutMatchingMarksEvaluation(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segment").Included("someone-else").Build()
	segment.IsOverride = true

	t.Run("non-matching clause", func(t *testing.T) {
		flag := makeBooleanFlagToMatchAnyOfSegments(segment.Key)

		evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
		result := evaluator.Evaluate(&flag, overrideTestContext, nil)

		assert.Equal(t, ldreason.EvalReasonFallthrough, result.Detail.Reason.GetKind())
		assert.Equal(t, ldvalue.Bool(false), result.Detail.Value)
		assertOverrideAffected(t, true, result)
	})

	t.Run("negated clause", func(t *testing.T) {
		flag := makeBooleanFlagWithClauses(ldbuilders.Negate(ldbuilders.SegmentMatchClause(segment.Key)))

		evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
		result := evaluator.Evaluate(&flag, overrideTestContext, nil)

		assert.Equal(t, ldreason.EvalReasonRuleMatch, result.Detail.Reason.GetKind())
		assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
		assertOverrideAffected(t, true, result)
	})
}

// The outer segment is plain. A rule of the outer segment reads a nested segment that is an override.
func TestOverrideSegmentReferencedBySegmentRuleMarksEvaluation(t *testing.T) {
	nested := ldbuilders.NewSegmentBuilder("nested-segment").Included(overrideTestContext.Key()).Build()
	nested.IsOverride = true
	outer := ldbuilders.NewSegmentBuilder("outer-segment").
		AddRule(ldbuilders.NewSegmentRuleBuilder().Clauses(ldbuilders.SegmentMatchClause(nested.Key))).
		Build()
	flag := makeBooleanFlagToMatchAnyOfSegments(outer.Key)

	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(outer, nested))
	result := evaluator.Evaluate(&flag, overrideTestContext, nil)

	assert.Equal(t, ldreason.EvalReasonRuleMatch, result.Detail.Reason.GetKind())
	assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
	assertOverrideAffected(t, true, result)
}

// A segment read while evaluating a prerequisite marks the prerequisite's record and the top level.
func TestOverrideSegmentReferencedByPrerequisiteMarksPrerequisiteRecordAndTopLevel(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segment").Included(overrideTestContext.Key()).Build()
	segment.IsOverride = true
	prereq := makeBooleanFlagToMatchAnyOfSegments(segment.Key)
	prereq.Key = "prereq"
	flag := makeOverrideTestFlag("feature", "prereq")

	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(prereq).withStoredSegments(segment))
	sink := prereqEventSink{}
	result := evaluator.Evaluate(&flag, overrideTestContext, sink.record)

	assert.Equal(t, ldreason.EvalReasonFallthrough, result.Detail.Reason.GetKind())
	assertOverrideAffected(t, true, result)
	assertOverrideAffected(t, true, requirePrereqRecord(t, &sink, "prereq"))
}

// A definition that cannot be resolved contributes nothing, because nothing was read. The store holds
// an unrelated override definition in each case to show that only reads count.
func TestMissingDefinitionDoesNotMarkEvaluation(t *testing.T) {
	t.Run("missing prerequisite", func(t *testing.T) {
		unrelated := makeOverrideTestFlag("unrelated")
		unrelated.IsOverride = true
		flag := makeOverrideTestFlag("feature", "missing")

		evaluator := NewEvaluator(basicDataProvider().withStoredFlags(unrelated).withNonexistentFlag("missing"))
		sink := prereqEventSink{}
		result := evaluator.Evaluate(&flag, overrideTestContext, sink.record)

		assert.Equal(t, ldreason.NewEvalReasonPrerequisiteFailed("missing"), result.Detail.Reason)
		assert.Equal(t, ldvalue.String("off"), result.Detail.Value)
		assertOverrideAffected(t, false, result)
		assert.Empty(t, sink.events)
	})

	t.Run("missing segment", func(t *testing.T) {
		unrelated := ldbuilders.NewSegmentBuilder("unrelated").Included(overrideTestContext.Key()).Build()
		unrelated.IsOverride = true
		flag := makeBooleanFlagToMatchAnyOfSegments("missing")

		evaluator := NewEvaluator(basicDataProvider().withStoredSegments(unrelated).withNonexistentSegment("missing"))
		result := evaluator.Evaluate(&flag, overrideTestContext, nil)

		assert.Equal(t, ldreason.EvalReasonFallthrough, result.Detail.Reason.GetKind())
		assert.Equal(t, ldvalue.Bool(false), result.Detail.Value)
		assertOverrideAffected(t, false, result)
	})
}

// Result.OverrideAffected and the reason indicator report the same marking on the top-level result
// and on each prerequisite record, whichever definition carried the marker.
func TestResultOverrideAffectedMatchesReasonIndicator(t *testing.T) {
	cases := []struct {
		name           string
		flagOverride   bool
		prereqOverride bool
	}{
		{"neither", false, false},
		{"flag only", true, false},
		{"prerequisite only", false, true},
		{"both", true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prereq := makeOverrideTestFlag("prereq")
			prereq.IsOverride = c.prereqOverride
			flag := makeOverrideTestFlag("feature", "prereq")
			flag.IsOverride = c.flagOverride

			evaluator := NewEvaluator(basicDataProvider().withStoredFlags(prereq))
			sink := prereqEventSink{}
			result := evaluator.Evaluate(&flag, overrideTestContext, sink.record)

			assert.Equal(t, result.Detail.Reason.IsOverrideAffected(), result.OverrideAffected)
			assert.Equal(t, c.flagOverride || c.prereqOverride, result.OverrideAffected)

			record := requirePrereqRecord(t, &sink, "prereq")
			assert.Equal(t, record.Detail.Reason.IsOverrideAffected(), record.OverrideAffected)
			assert.Equal(t, c.prereqOverride, record.OverrideAffected)
		})
	}
}
