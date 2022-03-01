package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldlog"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldlogtest"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldmodel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleWithTooHighVariationIndexReturnsMalformedFlagError(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, ldbuilders.Variation(999))

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(&f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestRuleWithNegativeVariationIndexReturnsMalformedFlagError(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, ldbuilders.Variation(-1))

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(&f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestRuleWithNoVariationOrRolloutReturnsMalformedFlagError(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, ldbuilders.Rollout())

	eventSink := prereqEventSink{}
	result := basicEvaluator().Evaluate(&f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestMalformedFlagErrorForBadVariationIndexIsLogged(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, ldbuilders.Variation(999))

	logCapture := ldlogtest.NewMockLog()
	eventSink := prereqEventSink{}
	e := NewEvaluatorWithOptions(basicDataProvider(), EvaluatorOptionErrorLogger(logCapture.Loggers.ForLevel(ldlog.Error)))

	result := e.Evaluate(&f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))

	errorLines := logCapture.GetOutput(ldlog.Error)
	require.Len(t, errorLines, 1)
	assert.Regexp(t, `referenced nonexistent variation index 999`, errorLines[0])
}

func TestMalformedFlagErrorForEmptyRolloutIsLogged(t *testing.T) {
	user := lduser.NewUser("userkey")
	f := makeFlagToMatchUser(user, ldbuilders.Rollout())

	logCapture := ldlogtest.NewMockLog()
	eventSink := prereqEventSink{}
	e := NewEvaluatorWithOptions(basicDataProvider(), EvaluatorOptionErrorLogger(logCapture.Loggers.ForLevel(ldlog.Error)))

	result := e.Evaluate(&f, user, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)
	assert.Equal(t, 0, len(eventSink.events))

	errorLines := logCapture.GetOutput(ldlog.Error)
	require.Len(t, errorLines, 1)
	assert.Regexp(t, `had a rollout or experiment with no variations`, errorLines[0])
}

func TestClauseWithUnknownOperatorDoesNotStopSubsequentRuleFromMatching(t *testing.T) {
	badClause := ldbuilders.Clause(lduser.NameAttribute, "doesSomethingUnsupported", ldvalue.String("Bob"))
	goodClause := ldbuilders.Clause(lduser.NameAttribute, ldmodel.OperatorIn, ldvalue.String("Bob"))
	f := ldbuilders.NewFlagBuilder("feature").
		On(true).
		AddRule(ldbuilders.NewRuleBuilder().ID("bad").Variation(1).Clauses(badClause)).
		AddRule(ldbuilders.NewRuleBuilder().ID("good").Variation(1).Clauses(goodClause)).
		Variations(ldvalue.Bool(false), ldvalue.Bool(true)).
		Build()
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := basicEvaluator().Evaluate(&f, user, nil)
	assert.True(t, result.Value.BoolValue())
	assert.Equal(t, ldreason.NewEvalReasonRuleMatch(1, "good"), result.Reason)
}
