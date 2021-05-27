package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldbuilders"

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
	f := ldbuilders.NewFlagBuilder("feature").
		On(false).
		OffVariation(1).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(&f, flagUser, eventSink.record)
	assert.Equal(t, offValue, result.Value)
	assert.Equal(t, ldvalue.NewOptionalInt(1), result.VariationIndex)
	assert.Equal(t, ldreason.NewEvalReasonOff(), result.Reason)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsNilIfFlagIsOffAndOffVariationIsUnspecified(t *testing.T) {
	f := ldbuilders.NewFlagBuilder("feature").
		On(false).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(&f, flagUser, eventSink.record)
	assert.Equal(t, ldreason.EvaluationDetail{Reason: ldreason.NewEvalReasonOff()}, result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsFallthroughIfFlagIsOnAndThereAreNoRules(t *testing.T) {
	f := ldbuilders.NewFlagBuilder("feature").
		On(true).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(&f, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(fallthroughValue, 0, ldreason.NewEvalReasonFallthrough()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsErrorIfFallthroughHasTooHighVariation(t *testing.T) {
	f := ldbuilders.NewFlagBuilder("feature").
		On(true).
		FallthroughVariation(999).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(&f, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsErrorIfFallthroughHasNegativeVariation(t *testing.T) {
	f := ldbuilders.NewFlagBuilder("feature").
		On(true).
		FallthroughVariation(-1).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(&f, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsErrorIfFallthroughHasNeitherVariationNorRollout(t *testing.T) {
	f := ldbuilders.NewFlagBuilder("feature").
		On(true).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(&f, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsErrorIfFallthroughHasEmptyRolloutVariationList(t *testing.T) {
	f := ldbuilders.NewFlagBuilder("feature").
		On(true).
		Fallthrough(ldbuilders.Rollout()).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(&f, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagMatchesUserFromTargets(t *testing.T) {
	f := ldbuilders.NewFlagBuilder("feature").
		On(true).
		OffVariation(1).
		AddTarget(2, "whoever", "userkey").
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()
	user := lduser.NewUser("userkey")

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(&f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(onValue, 2, ldreason.NewEvalReasonTargetMatch()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagMatchesUserFromRules(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, ldbuilders.Variation(2))

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(&f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(onValue, 2, ldreason.NewEvalReasonRuleMatch(0, "rule-id")), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsWhetherUserWasInFallthroughExperiment(t *testing.T) {
	// seed here carefully chosen so users fall into different buckets
	user1 := lduser.NewUser("userKeyA")
	user2 := lduser.NewUser("userKeyB")
	user3 := lduser.NewUser("userKeyC")

	f := ldbuilders.NewFlagBuilder("experiment").
		On(true).
		Fallthrough(ldbuilders.Experiment(
			61,
			ldbuilders.Bucket(0, 10000),
			ldbuilders.Bucket(2, 20000),
			ldbuilders.BucketUntracked(0, 70000),
		)).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	result := basicEvaluator().Evaluate(&f, user1, nil)
	// bucketVal = 0.09801207
	assert.Equal(t, ldreason.NewEvaluationDetail(fallthroughValue, 0, ldreason.NewEvalReasonFallthroughExperiment(true)), result)

	result = basicEvaluator().Evaluate(&f, user2, nil)
	// bucketVal = 0.14483777
	assert.Equal(t, ldreason.NewEvaluationDetail(onValue, 2, ldreason.NewEvalReasonFallthroughExperiment(true)), result)

	result = basicEvaluator().Evaluate(&f, user3, nil)
	// bucketVal = 0.9242641
	assert.Equal(t, ldreason.NewEvaluationDetail(fallthroughValue, 0, ldreason.NewEvalReasonFallthroughExperiment(false)), result)
}

func TestFlagReturnsWhetherUserWasInRuleExperiment(t *testing.T) {
	// seed here carefully chosen so users fall into different buckets
	user1 := lduser.NewUser("userKeyA")
	user2 := lduser.NewUser("userKeyB")
	user3 := lduser.NewUser("userKeyC")

	f := ldbuilders.NewFlagBuilder("experiment").
		On(true).
		AddRule(makeRuleToMatchUserKeyPrefix("user", ldbuilders.Experiment(
			61,
			ldbuilders.Bucket(0, 10000),
			ldbuilders.Bucket(2, 20000),
			ldbuilders.BucketUntracked(0, 70000),
		))).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	result := basicEvaluator().Evaluate(&f, user1, nil)
	// bucketVal = 0.09801207
	assert.Equal(t, ldreason.NewEvaluationDetail(fallthroughValue, 0, ldreason.NewEvalReasonRuleMatchExperiment(0, "rule-id", true)), result)

	result = basicEvaluator().Evaluate(&f, user2, nil)
	// bucketVal = 0.14483777
	assert.Equal(t, ldreason.NewEvaluationDetail(onValue, 2, ldreason.NewEvalReasonRuleMatchExperiment(0, "rule-id", true)), result)

	result = basicEvaluator().Evaluate(&f, user3, nil)
	// bucketVal = 0.9242641
	assert.Equal(t, ldreason.NewEvaluationDetail(fallthroughValue, 0, ldreason.NewEvalReasonRuleMatchExperiment(0, "rule-id", false)), result)
}
