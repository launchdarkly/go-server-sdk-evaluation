package evaluation

import (
	"testing"

	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldreason"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	"github.com/launchdarkly/go-server-sdk-evaluation/v3/ldbuilders"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var overrideTestContext = ldcontext.New("userkey")

func TestOverrideFlagMarksReason(t *testing.T) {
	t.Run("off", func(t *testing.T) {
		flag := ldbuilders.NewFlagBuilder("feature").On(false).OffVariation(0).
			Variations(ldvalue.String("off"), ldvalue.String("on")).Build()
		flag.IsOverride = true

		result := basicEvaluator().Evaluate(&flag, overrideTestContext, nil)

		assert.Equal(t, ldreason.EvalReasonOff, result.Detail.Reason.GetKind())
		assert.True(t, result.Detail.Reason.IsOverride())
		assert.Equal(t, ldvalue.String("off"), result.Detail.Value)
	})

	t.Run("fallthrough", func(t *testing.T) {
		flag := ldbuilders.NewFlagBuilder("feature").On(true).FallthroughVariation(1).
			Variations(ldvalue.String("off"), ldvalue.String("on")).Build()
		flag.IsOverride = true

		result := basicEvaluator().Evaluate(&flag, overrideTestContext, nil)

		assert.Equal(t, ldreason.EvalReasonFallthrough, result.Detail.Reason.GetKind())
		assert.True(t, result.Detail.Reason.IsOverride())
	})

	t.Run("rule match", func(t *testing.T) {
		flag := ldbuilders.NewFlagBuilder("feature").On(true).FallthroughVariation(0).
			AddRule(ldbuilders.NewRuleBuilder().ID("rule-id").Variation(1).
				Clauses(makeClauseToMatchContext(overrideTestContext))).
			Variations(ldvalue.String("off"), ldvalue.String("on")).Build()
		flag.IsOverride = true

		result := basicEvaluator().Evaluate(&flag, overrideTestContext, nil)

		assert.Equal(t, ldreason.EvalReasonRuleMatch, result.Detail.Reason.GetKind())
		assert.True(t, result.Detail.Reason.IsOverride())
	})
}

func TestNonOverrideFlagReasonIsNotMarked(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("feature").On(false).OffVariation(0).
		Variations(ldvalue.String("off"), ldvalue.String("on")).Build()

	result := basicEvaluator().Evaluate(&flag, overrideTestContext, nil)

	assert.Equal(t, ldreason.EvalReasonOff, result.Detail.Reason.GetKind())
	assert.False(t, result.Detail.Reason.IsOverride())
}

func TestOverrideFlagErrorReasonIsNotMarked(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("feature").On(true).FallthroughVariation(99).
		Variations(ldvalue.String("off"), ldvalue.String("on")).Build()
	flag.IsOverride = true

	result := basicEvaluator().Evaluate(&flag, overrideTestContext, nil)

	assert.Equal(t, ldreason.NewEvalReasonError(ldreason.EvalErrorMalformedFlag), result.Detail.Reason)
	assert.False(t, result.Detail.Reason.IsOverride())
}

func TestOverridePrerequisiteMarksOnlyThePrerequisiteEventReason(t *testing.T) {
	prereq := ldbuilders.NewFlagBuilder("prereq").On(true).FallthroughVariation(1).
		Variations(ldvalue.String("nogo"), ldvalue.String("go")).Build()
	prereq.IsOverride = true
	flag := ldbuilders.NewFlagBuilder("feature").On(true).FallthroughVariation(1).
		AddPrerequisite("prereq", 1).OffVariation(0).
		Variations(ldvalue.String("off"), ldvalue.String("on")).Build()

	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(prereq))
	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(&flag, overrideTestContext, eventSink.record)

	assert.Equal(t, ldreason.EvalReasonFallthrough, result.Detail.Reason.GetKind())
	assert.False(t, result.Detail.Reason.IsOverride())

	require.Len(t, eventSink.events, 1)
	prereqReason := eventSink.events[0].PrerequisiteResult.Detail.Reason
	assert.Equal(t, ldreason.EvalReasonFallthrough, prereqReason.GetKind())
	assert.True(t, prereqReason.IsOverride())
}

func TestNonOverridePrerequisiteEventReasonIsNotMarked(t *testing.T) {
	prereq := ldbuilders.NewFlagBuilder("prereq").On(true).FallthroughVariation(1).
		Variations(ldvalue.String("nogo"), ldvalue.String("go")).Build()
	flag := ldbuilders.NewFlagBuilder("feature").On(true).FallthroughVariation(1).
		AddPrerequisite("prereq", 1).OffVariation(0).
		Variations(ldvalue.String("off"), ldvalue.String("on")).Build()
	flag.IsOverride = true

	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(prereq))
	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(&flag, overrideTestContext, eventSink.record)

	assert.True(t, result.Detail.Reason.IsOverride())

	require.Len(t, eventSink.events, 1)
	assert.False(t, eventSink.events[0].PrerequisiteResult.Detail.Reason.IsOverride())
}
