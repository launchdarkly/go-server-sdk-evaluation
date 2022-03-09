package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldmodel"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldlog"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldlogtest"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v3/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"

	"github.com/stretchr/testify/assert"
)

var flagTestContext = lduser.NewUser("x")

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

	result := basicEvaluator().Evaluate(&f, flagTestContext, FailOnAnyPrereqEvent(t))
	assert.Equal(t, offValue, result.Value)
	assert.Equal(t, ldvalue.NewOptionalInt(1), result.VariationIndex)
	assert.Equal(t, ldreason.NewEvalReasonOff(), result.Reason)
}

func TestFlagReturnsNilIfFlagIsOffAndOffVariationIsUnspecified(t *testing.T) {
	f := ldbuilders.NewFlagBuilder("feature").
		On(false).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	result := basicEvaluator().Evaluate(&f, flagTestContext, FailOnAnyPrereqEvent(t))
	assert.Equal(t, ldreason.EvaluationDetail{Reason: ldreason.NewEvalReasonOff()}, result)
}

func TestFlagReturnsFallthroughIfFlagIsOnAndThereAreNoRules(t *testing.T) {
	f := ldbuilders.NewFlagBuilder("feature").
		On(true).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	result := basicEvaluator().Evaluate(&f, flagTestContext, FailOnAnyPrereqEvent(t))
	assert.Equal(t, ldreason.NewEvaluationDetail(fallthroughValue, 0, ldreason.NewEvalReasonFallthrough()), result)
}

func TestFlagMatchesUserFromRules(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, ldbuilders.Variation(2))

	result := basicEvaluator().Evaluate(&f, user, FailOnAnyPrereqEvent(t))
	assert.Equal(t, ldreason.NewEvaluationDetail(onValue, 2, ldreason.NewEvalReasonRuleMatch(0, "rule-id")), result)
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

func TestMalformedFlagErrorForBadFlagProperties(t *testing.T) {
	basicContext := ldcontext.New("userkey")

	type testCaseParams struct {
		name    string
		context ldcontext.Context
		flag    ldmodel.FeatureFlag
		message string
	}

	for _, p := range []testCaseParams{
		{
			name:    "fallthrough with variation index too high",
			context: basicContext,
			flag: ldbuilders.NewFlagBuilder("feature").
				On(true).
				FallthroughVariation(999).
				Variations(fallthroughValue, offValue, onValue).
				Build(),
			message: "nonexistent variation index 999",
		},
		{
			name:    "fallthrough with negative variation index",
			context: basicContext,
			flag: ldbuilders.NewFlagBuilder("feature").
				On(true).
				FallthroughVariation(-1).
				Variations(fallthroughValue, offValue, onValue).
				Build(),
			message: "nonexistent variation index -1",
		},
		{
			name:    "fallthrough with neither variation nor rollout",
			context: basicContext,
			flag: ldbuilders.NewFlagBuilder("feature").
				On(true).
				Variations(fallthroughValue, offValue, onValue).
				Build(),
			message: "rollout or experiment with no variations",
		},
		{
			name:    "fallthrough with empty rollout",
			context: basicContext,
			flag: ldbuilders.NewFlagBuilder("feature").
				On(true).
				Fallthrough(ldbuilders.Rollout()).
				Variations(fallthroughValue, offValue, onValue).
				Build(),
			message: "rollout or experiment with no variations",
		},
	} {
		t.Run(p.name, func(t *testing.T) {
			t.Run("returns error", func(t *testing.T) {
				result := basicEvaluator().Evaluate(&p.flag, p.context, FailOnAnyPrereqEvent(t))

				assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
			})

			t.Run("logs error", func(t *testing.T) {
				logCapture := ldlogtest.NewMockLog()
				e := NewEvaluatorWithOptions(basicDataProvider(),
					EvaluatorOptionErrorLogger(logCapture.Loggers.ForLevel(ldlog.Error)))
				_ = e.Evaluate(&p.flag, p.context, FailOnAnyPrereqEvent(t))

				errorLines := logCapture.GetOutput(ldlog.Error)
				if assert.Len(t, errorLines, 1) {
					assert.Regexp(t, p.message, errorLines[0])
				}
			})
		})
	}
}
