package evaluation

import (
	"testing"

	"github.com/launchdarkly/go-server-sdk-evaluation/v3/ldbuilders"
	"github.com/launchdarkly/go-server-sdk-evaluation/v3/ldmodel"

	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldlog"
	"github.com/launchdarkly/go-sdk-common/v3/ldlogtest"
	"github.com/launchdarkly/go-sdk-common/v3/ldreason"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	m "github.com/launchdarkly/go-test-helpers/v3/matchers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var flagTestContext = ldcontext.New("x")

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
	m.In(t).Assert(result, ResultDetailProps(1, offValue, ldreason.NewEvalReasonOff()))
	assert.False(t, result.IsExperiment)
}

func TestFlagReturnsNilIfFlagIsOffAndOffVariationIsUnspecified(t *testing.T) {
	f := ldbuilders.NewFlagBuilder("feature").
		On(false).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	result := basicEvaluator().Evaluate(&f, flagTestContext, FailOnAnyPrereqEvent(t))
	m.In(t).Assert(result.Detail, EvalDetailEquals(ldreason.EvaluationDetail{Reason: ldreason.NewEvalReasonOff()}))
	assert.False(t, result.IsExperiment)
}

func TestFlagReturnsFallthroughIfFlagIsOnAndThereAreNoRules(t *testing.T) {
	f := ldbuilders.NewFlagBuilder("feature").
		On(true).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	result := basicEvaluator().Evaluate(&f, flagTestContext, FailOnAnyPrereqEvent(t))
	m.In(t).Assert(result, ResultDetailProps(0, fallthroughValue, ldreason.NewEvalReasonFallthrough()))
	assert.False(t, result.IsExperiment)
}

func TestFlagMatchesContextFromRules(t *testing.T) {
	f := makeFlagToMatchContext(flagTestContext, ldbuilders.Variation(2))

	result := basicEvaluator().Evaluate(&f, flagTestContext, FailOnAnyPrereqEvent(t))
	m.In(t).Assert(result, ResultDetailProps(2, onValue, ldreason.NewEvalReasonRuleMatch(0, "rule-id")))
	assert.False(t, result.IsExperiment)
}

func TestFlagReturnsWhetherContextWasInFallthroughExperiment(t *testing.T) {
	// seed here carefully chosen so users fall into different buckets
	user1, user2, user3 := ldcontext.New("userKeyA"), ldcontext.New("userKeyB"), ldcontext.New("userKeyC")

	f := ldbuilders.NewFlagBuilder("experiment").
		On(true).
		Fallthrough(ldbuilders.Experiment(
			ldvalue.NewOptionalInt(61),
			ldbuilders.Bucket(0, 10000),
			ldbuilders.Bucket(2, 20000),
			ldbuilders.BucketUntracked(0, 70000),
		)).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	result := basicEvaluator().Evaluate(&f, user1, nil)
	// bucketVal = 0.09801207
	m.In(t).Assert(result, ResultDetailProps(0, fallthroughValue, ldreason.NewEvalReasonFallthroughExperiment(true)))
	assert.True(t, result.IsExperiment)

	result = basicEvaluator().Evaluate(&f, user2, nil)
	// bucketVal = 0.14483777
	m.In(t).Assert(result, ResultDetailProps(2, onValue, ldreason.NewEvalReasonFallthroughExperiment(true)))
	assert.True(t, result.IsExperiment)

	result = basicEvaluator().Evaluate(&f, user3, nil)
	// bucketVal = 0.9242641
	m.In(t).Assert(result, ResultDetailProps(0, fallthroughValue, ldreason.NewEvalReasonFallthrough()))
	assert.False(t, result.IsExperiment)
}

func TestFlagReturnsWhetherContextWasInRuleExperiment(t *testing.T) {
	// seed here carefully chosen so users fall into different buckets
	user1, user2, user3 := ldcontext.New("userKeyA"), ldcontext.New("userKeyB"), ldcontext.New("userKeyC")

	f := ldbuilders.NewFlagBuilder("experiment").
		On(true).
		AddRule(makeRuleToMatchUserKeyPrefix("user", ldbuilders.Experiment(
			ldvalue.NewOptionalInt(61),
			ldbuilders.Bucket(0, 10000),
			ldbuilders.Bucket(2, 20000),
			ldbuilders.BucketUntracked(0, 70000),
		))).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	result := basicEvaluator().Evaluate(&f, user1, nil)
	// bucketVal = 0.09801207
	m.In(t).Assert(result, ResultDetailProps(0, fallthroughValue, ldreason.NewEvalReasonRuleMatchExperiment(0, "rule-id", true)))
	assert.True(t, result.IsExperiment)

	result = basicEvaluator().Evaluate(&f, user2, nil)
	// bucketVal = 0.14483777
	m.In(t).Assert(result, ResultDetailProps(2, onValue, ldreason.NewEvalReasonRuleMatchExperiment(0, "rule-id", true)))
	assert.True(t, result.IsExperiment)

	result = basicEvaluator().Evaluate(&f, user3, nil)
	// bucketVal = 0.9242641
	m.In(t).Assert(result, ResultDetailProps(0, fallthroughValue, ldreason.NewEvalReasonRuleMatch(0, "rule-id")))
	assert.False(t, result.IsExperiment)
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
				m.In(t).Assert(result, ResultDetailError(ldreason.EvalErrorMalformedFlag))
			})

			t.Run("logs error", func(t *testing.T) {
				logCapture := ldlogtest.NewMockLog()
				e := NewEvaluatorWithOptions(basicDataProvider().withConfigOverrides(
					ldbuilders.NewConfigOverrideBuilder("indexSamplingRatio").Value(ldvalue.Int(1)).Build(),
				),
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

func TestUserNotSpecifiedErrorForInvalidContext(t *testing.T) {
	badContext := ldcontext.New("")
	require.Error(t, badContext.Err())

	f := ldbuilders.NewFlagBuilder("feature").
		On(false).
		OffVariation(1).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()

	result := basicEvaluator().Evaluate(&f, badContext, FailOnAnyPrereqEvent(t))
	assertResultDetail(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorUserNotSpecified, ldvalue.Null()), result)
}
