package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"

	"github.com/stretchr/testify/assert"
)

func TestRuleWithTooHighVariationIndexReturnsMalformedFlagError(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, VariationOrRollout{Variation: intPtr(999)})

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestRuleWithNegativeVariationIndexReturnsMalformedFlagError(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, VariationOrRollout{Variation: intPtr(-1)})

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestRuleWithNoVariationOrRolloutReturnsMalformedFlagError(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, VariationOrRollout{})

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestRuleWithRolloutWithEmptyVariationsListReturnsMalformedFlagError(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, VariationOrRollout{Rollout: &Rollout{Variations: []WeightedVariation{}}})

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}
