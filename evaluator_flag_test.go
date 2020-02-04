package evaluation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

var flagUser = lduser.NewUser("x")

var fallthroughValue = ldvalue.String("fall")
var offValue = ldvalue.String("off")
var onValue = ldvalue.String("on")

func TestFlagReturnsOffVariationIfFlagIsOff(t *testing.T) {
	f := FeatureFlag{
		Key:          "feature",
		On:           false,
		OffVariation: intPtr(1),
		Fallthrough:  VariationOrRollout{Variation: intPtr(0)},
		Variations:   []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, flagUser, eventSink.record)
	assert.Equal(t, offValue, result.Value)
	assert.Equal(t, 1, result.VariationIndex)
	assert.Equal(t, ldreason.NewEvalReasonOff(), result.Reason)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsNilIfFlagIsOffAndOffVariationIsUnspecified(t *testing.T) {
	f := FeatureFlag{
		Key:         "feature",
		On:          false,
		Fallthrough: VariationOrRollout{Variation: intPtr(0)},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(ldvalue.Null(), -1, ldreason.NewEvalReasonOff()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsFallthroughIfFlagIsOnAndThereAreNoRules(t *testing.T) {
	f := FeatureFlag{
		Key:         "feature",
		On:          true,
		Rules:       []FlagRule{},
		Fallthrough: VariationOrRollout{Variation: intPtr(0)},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(fallthroughValue, 0, ldreason.NewEvalReasonFallthrough()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsErrorIfFallthroughHasTooHighVariation(t *testing.T) {
	f := FeatureFlag{
		Key:         "feature",
		On:          true,
		Rules:       []FlagRule{},
		Fallthrough: VariationOrRollout{Variation: intPtr(999)},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsErrorIfFallthroughHasNegativeVariation(t *testing.T) {
	f := FeatureFlag{
		Key:         "feature",
		On:          true,
		Rules:       []FlagRule{},
		Fallthrough: VariationOrRollout{Variation: intPtr(-1)},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsErrorIfFallthroughHasNeitherVariationNorRollout(t *testing.T) {
	f := FeatureFlag{
		Key:         "feature",
		On:          true,
		Rules:       []FlagRule{},
		Fallthrough: VariationOrRollout{},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsErrorIfFallthroughHasEmptyRolloutVariationList(t *testing.T) {
	f := FeatureFlag{
		Key:         "feature",
		On:          true,
		Rules:       []FlagRule{},
		Fallthrough: VariationOrRollout{Rollout: &Rollout{Variations: []WeightedVariation{}}},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagMatchesUserFromTargets(t *testing.T) {
	f := FeatureFlag{
		Key:          "feature",
		On:           true,
		OffVariation: intPtr(1),
		Targets:      []Target{Target{[]string{"whoever", "userkey"}, 2}},
		Fallthrough:  VariationOrRollout{Variation: intPtr(0)},
		Variations:   []ldvalue.Value{fallthroughValue, offValue, onValue},
	}
	user := lduser.NewUser("userkey")

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(onValue, 2, ldreason.NewEvalReasonTargetMatch()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagMatchesUserFromRules(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, VariationOrRollout{Variation: intPtr(2)})

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(onValue, 2, ldreason.NewEvalReasonRuleMatch(0, "rule-id")), result)
	assert.Equal(t, 0, len(eventSink.events))
}
