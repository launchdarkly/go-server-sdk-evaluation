package evaluation

import (
	"testing"

	"github.com/launchdarkly/go-server-sdk-evaluation/v2/ldbuilders"
	"github.com/launchdarkly/go-server-sdk-evaluation/v2/ldmodel"

	"github.com/launchdarkly/go-sdk-common/v3/ldattr"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldlog"
	"github.com/launchdarkly/go-sdk-common/v3/ldlogtest"
	"github.com/launchdarkly/go-sdk-common/v3/ldreason"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	m "github.com/launchdarkly/go-test-helpers/v3/matchers"

	"github.com/stretchr/testify/assert"
)

func TestMalformedFlagErrorForBadRuleProperties(t *testing.T) {
	basicContext := ldcontext.New("userkey")

	type testCaseParams struct {
		name    string
		context ldcontext.Context
		flag    ldmodel.FeatureFlag
		message string
	}

	for _, p := range []testCaseParams{
		{
			name:    "variation index too high",
			context: basicContext,
			flag:    makeFlagToMatchContext(basicContext, ldbuilders.Variation(999)),
			message: "nonexistent variation index 999",
		},
		{
			name:    "negative variation index",
			context: basicContext,
			flag:    makeFlagToMatchContext(basicContext, ldbuilders.Variation(-1)),
			message: "nonexistent variation index -1",
		},
		{
			name:    "no variation or rollout",
			context: basicContext,
			flag:    makeFlagToMatchContext(basicContext, ldbuilders.Rollout()),
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
				e := NewEvaluatorWithOptions(basicDataProvider().withConfigOverrides(ldbuilders.NewConfigOverrideBuilder("indexSamplingRatio").Value(ldvalue.Int(1)).Build()),
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

func TestMalformedFlagErrorForBadClauseProperties(t *testing.T) {
	basicContext := ldcontext.New("userkey")

	type testCaseParams struct {
		name    string
		context ldcontext.Context
		clause  ldmodel.Clause
		message string
	}

	for _, p := range []testCaseParams{
		{
			name:    "undefined attribute",
			context: basicContext,
			clause:  ldbuilders.ClauseRef(ldattr.Ref{}, ldmodel.OperatorIn, ldvalue.String("a")),
			message: "rule clause did not specify an attribute",
		},
		{
			name:    "invalid attribute reference",
			context: basicContext,
			clause:  ldbuilders.ClauseRef(ldattr.NewRef("///"), ldmodel.OperatorIn, ldvalue.String("a")),
			message: "invalid attribute reference",
		},
	} {
		t.Run(p.name, func(t *testing.T) {
			goodClause := makeClauseToMatchContext(p.context)
			flag := ldbuilders.NewFlagBuilder("feature").
				On(true).
				AddRule(ldbuilders.NewRuleBuilder().ID("bad").Variation(1).Clauses(p.clause)).
				AddRule(ldbuilders.NewRuleBuilder().ID("good").Variation(1).Clauses(goodClause)).
				Variations(ldvalue.Bool(false), ldvalue.Bool(true)).
				Build()

			t.Run("returns error", func(t *testing.T) {
				result := basicEvaluator().Evaluate(&flag, p.context, FailOnAnyPrereqEvent(t))
				m.In(t).Assert(result, ResultDetailError(ldreason.EvalErrorMalformedFlag))
			})

			t.Run("logs error", func(t *testing.T) {
				logCapture := ldlogtest.NewMockLog()
				e := NewEvaluatorWithOptions(basicDataProvider().withConfigOverrides(ldbuilders.NewConfigOverrideBuilder("indexSamplingRatio").Value(ldvalue.Int(1)).Build()),
					EvaluatorOptionErrorLogger(logCapture.Loggers.ForLevel(ldlog.Error)))
				_ = e.Evaluate(&flag, p.context, FailOnAnyPrereqEvent(t))

				errorLines := logCapture.GetOutput(ldlog.Error)
				if assert.Len(t, errorLines, 1) {
					assert.Regexp(t, p.message, errorLines[0])
				}
			})
		})
	}
}

func TestClauseWithUnknownOperatorDoesNotStopSubsequentRuleFromMatching(t *testing.T) {
	context := ldcontext.New("key")
	badClause := ldbuilders.Clause(ldattr.NameAttr, "doesSomethingUnsupported", ldvalue.String("Bob"))
	goodClause := makeClauseToMatchContext(context)
	f := ldbuilders.NewFlagBuilder("feature").
		On(true).
		AddRule(ldbuilders.NewRuleBuilder().ID("bad").Variation(1).Clauses(badClause)).
		AddRule(ldbuilders.NewRuleBuilder().ID("good").Variation(1).Clauses(goodClause)).
		Variations(ldvalue.Bool(false), ldvalue.Bool(true)).
		Build()

	result := basicEvaluator().Evaluate(&f, context, nil)
	m.In(t).Assert(result, ResultDetailProps(1, ldvalue.Bool(true), ldreason.NewEvalReasonRuleMatch(1, "good")))
}
