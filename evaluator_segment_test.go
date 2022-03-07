package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldmodel"

	"github.com/stretchr/testify/assert"
)

func assertSegmentMatch(t *testing.T, segment ldmodel.Segment, context ldcontext.Context, expected bool) {
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	result := evaluator.Evaluate(&f, context, nil)
	assert.Equal(t, expected, result.Value.BoolValue())
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
	assert.False(t, result.Value.BoolValue())
}

func TestCanMatchJustOneSegmentFromList(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").Included("foo").Build()
	f := booleanFlagWithSegmentMatch("unknown-segment-key", "segkey")
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment).withNonexistentSegment("unknown-segment-key"))
	user := lduser.NewUser("foo")

	result := evaluator.Evaluate(&f, user, nil)
	assert.True(t, result.Value.BoolValue())
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
	assert.True(t, result.Value.BoolValue())
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
	assert.False(t, result.Value.BoolValue())
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
