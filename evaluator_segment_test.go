package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldmodel"

	m "github.com/launchdarkly/go-test-helpers/v2/matchers"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldlog"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldlogtest"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v3/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"

	"github.com/stretchr/testify/assert"
)

func assertSegmentMatch(t *testing.T, segment ldmodel.Segment, context ldcontext.Context, expected bool) {
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	result := evaluator.Evaluate(&f, context, nil)
	assert.Equal(t, expected, result.Detail.Value.BoolValue())
}

func TestSegmentMatchClauseRetrievesSegmentFromStore(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").Included("foo").Build()
	user := lduser.NewUser("foo")
	assertSegmentMatch(t, segment, user, true)
}

func TestSegmentMatchClauseFallsThroughIfSegmentNotFound(t *testing.T) {
	f := booleanFlagWithSegmentMatch("unknown-segment-key")
	evaluator := NewEvaluator(basicDataProvider().withNonexistentSegment("unknown-segment-key"))
	user := lduser.NewUser("foo")

	result := evaluator.Evaluate(&f, user, nil)
	assert.False(t, result.Detail.Value.BoolValue())
}

func TestCanMatchJustOneSegmentFromList(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").Included("foo").Build()
	f := booleanFlagWithSegmentMatch("unknown-segment-key", "segkey")
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment).withNonexistentSegment("unknown-segment-key"))
	user := lduser.NewUser("foo")

	result := evaluator.Evaluate(&f, user, nil)
	assert.True(t, result.Detail.Value.BoolValue())
}

func TestUserIsExplicitlyIncludedInSegment(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").Included("foo", "bar").Build()
	user := lduser.NewUser("bar")
	assertSegmentMatch(t, segment, user, true)
}

func TestUserIsMatchedBySegmentRule(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(ldattr.NameAttr, ldmodel.OperatorIn, ldvalue.String("Jane")))).
		Build()
	user := lduser.NewUserBuilder("key").Name("Jane").Build()
	assertSegmentMatch(t, segment, user, true)
}

func TestUserIsExplicitlyExcludedFromSegment(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		Excluded("foo", "bar").
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(ldattr.NameAttr, ldmodel.OperatorIn, ldvalue.String("Jane")))).
		Build()
	user := lduser.NewUserBuilder("foo").Name("Jane").Build()
	assertSegmentMatch(t, segment, user, false)
}

func TestSegmentIncludesOverrideExcludes(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		Excluded("bar").
		Included("foo", "bar").
		Build()
	user := lduser.NewUser("bar")
	assertSegmentMatch(t, segment, user, true)
}

func TestSegmentDoesNotMatchUserIfNoIncludesOrRulesMatch(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		Included("other-key").
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(ldattr.NameAttr, ldmodel.OperatorIn, ldvalue.String("Jane")))).
		Build()
	user := lduser.NewUserBuilder("key").Name("Bob").Build()
	assertSegmentMatch(t, segment, user, false)
}

func TestSegmentRuleCanMatchUserWithPercentageRollout(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(ldattr.NameAttr, ldmodel.OperatorIn, ldvalue.String("Jane"))).
			Weight(99999)).
		Build()
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUserBuilder("key").Name("Jane").Build()

	result := evaluator.Evaluate(&f, user, nil)
	assert.True(t, result.Detail.Value.BoolValue())
}

func TestSegmentRuleCanNotMatchUserWithPercentageRollout(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(ldattr.NameAttr, ldmodel.OperatorIn, ldvalue.String("Jane"))).
			Weight(1)).
		Build()
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUserBuilder("key").Name("Jane").Build()

	result := evaluator.Evaluate(&f, user, nil)
	assert.False(t, result.Detail.Value.BoolValue())
}

func TestSegmentRuleCanHavePercentageRollout(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorContains, ldvalue.String("user"))).
			Weight(30000)).
		Salt("salty").
		Build()

	// Weight: 30000 means that the rule returns a match if the user's bucket value >= 0.3
	user1 := lduser.NewUser("userKeyA") // bucket value = 0.14574753
	assertSegmentMatch(t, segment, user1, true)

	user2 := lduser.NewUser("userKeyZ") // bucket value = 0.45679215
	assertSegmentMatch(t, segment, user2, false)
}

func TestSegmentRuleCanHavePercentageRolloutByAnyAttribute(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorContains, ldvalue.String("x"))).
			BucketBy(ldattr.NameAttr).
			Weight(30000)).
		Salt("salty").
		Build()

	// Weight: 30000 means that the rule returns a match if the user's bucket value >= 0.3
	user1 := lduser.NewUserBuilder("x").Name("userKeyA").Build() // bucket value = 0.14574753
	assertSegmentMatch(t, segment, user1, true)

	user2 := lduser.NewUserBuilder("x").Name("userKeyZ").Build() // bucket value = 0.45679215
	assertSegmentMatch(t, segment, user2, false)
}

func TestSegmentRuleIsNonMatchForInvalidBucketByReference(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorContains, ldvalue.String("x"))).
			BucketByRef(ldattr.NewRef("///")).
			Weight(30000)).
		Salt("salty").
		Build()

	user1 := lduser.NewUserBuilder("x").Name("userKeyA").Build() // bucket value = 0.14574753
	assertSegmentMatch(t, segment, user1, false)
}

func TestMalformedFlagErrorForBadSegmentProperties(t *testing.T) {
	basicContext := ldcontext.New("userkey")

	type testCaseParams struct {
		name    string
		context ldcontext.Context
		segment ldmodel.Segment
		message string
	}

	for _, p := range []testCaseParams{
		{
			name:    "bucketBy with invalid attribute",
			context: basicContext,
			segment: ldbuilders.NewSegmentBuilder("segkey").
				AddRule(ldbuilders.NewSegmentRuleBuilder().
					Clauses(ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String(basicContext.Key()))).
					BucketByRef(ldattr.NewRef("///")).
					Weight(30000)).
				Salt("salty").
				Build(),
			message: "attribute reference",
		},
		{
			name:    "clause with undefined attribute",
			context: basicContext,
			segment: ldbuilders.NewSegmentBuilder("segkey").
				AddRule(ldbuilders.NewSegmentRuleBuilder().
					Clauses(ldbuilders.ClauseRef(ldattr.Ref{}, ldmodel.OperatorIn, ldvalue.String("a"))).
					BucketByRef(ldattr.NewRef("///")).
					Weight(30000)).
				Salt("salty").
				Build(),
			message: "rule clause did not specify an attribute",
		},
		{
			name:    "clause with invalid attribute reference",
			context: basicContext,
			segment: ldbuilders.NewSegmentBuilder("segkey").
				AddRule(ldbuilders.NewSegmentRuleBuilder().
					Clauses(ldbuilders.ClauseRef(ldattr.NewRef("///"), ldmodel.OperatorIn, ldvalue.String("a"))).
					BucketByRef(ldattr.NewRef("///")).
					Weight(30000)).
				Build(),
			message: "invalid context attribute reference",
		},
	} {
		t.Run(p.name, func(t *testing.T) {
			flag := booleanFlagWithSegmentMatch(p.segment.Key)

			t.Run("returns error", func(t *testing.T) {
				e := NewEvaluator(basicDataProvider().withStoredSegments(p.segment))
				result := e.Evaluate(&flag, p.context, FailOnAnyPrereqEvent(t))

				m.In(t).Assert(result, ResultDetailError(ldreason.EvalErrorMalformedFlag))
			})

			t.Run("logs error", func(t *testing.T) {
				logCapture := ldlogtest.NewMockLog()
				e := NewEvaluatorWithOptions(basicDataProvider().withStoredSegments(p.segment),
					EvaluatorOptionErrorLogger(logCapture.Loggers.ForLevel(ldlog.Error)))
				_ = e.Evaluate(&flag, p.context, FailOnAnyPrereqEvent(t))

				errorLines := logCapture.GetOutput(ldlog.Error)
				if assert.Len(t, errorLines, 1) {
					assert.Regexp(t, `segment "segkey".*`+p.message, errorLines[0])
				}
			})
		})
	}
}
