package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldmodel"

	"github.com/stretchr/testify/assert"
)

func TestRuleWithTooHighVariationIndexReturnsMalformedFlagError(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, ldbuilders.Variation(999))

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestRuleWithNegativeVariationIndexReturnsMalformedFlagError(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, ldbuilders.Variation(-1))

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestRuleWithNoVariationOrRolloutReturnsMalformedFlagError(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, ldmodel.VariationOrRollout{})

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestRuleWithRolloutWithEmptyVariationsListReturnsMalformedFlagError(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, ldmodel.VariationOrRollout{Rollout: &ldmodel.Rollout{}})

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}
