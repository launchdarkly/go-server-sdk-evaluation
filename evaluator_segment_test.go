package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldmodel"

	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

func TestSegmentMatchClauseRetrievesSegmentFromStore(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").Included("foo").Build()
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUser("foo")

	result := evaluator.Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestSegmentMatchClauseFallsThroughIfSegmentNotFound(t *testing.T) {
	f := booleanFlagWithSegmentMatch("unknown-segment-key")
	evaluator := NewEvaluator(basicDataProvider().withNonexistentSegment("unknown-segment-key"))
	user := lduser.NewUser("foo")

	result := evaluator.Evaluate(f, user, nil)
	assert.False(t, result.Value.BoolValue())
}

func TestCanMatchJustOneSegmentFromList(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").Included("foo").Build()
	f := booleanFlagWithSegmentMatch("unknown-segment-key", "segkey")
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment).withNonexistentSegment("unknown-segment-key"))
	user := lduser.NewUser("foo")

	result := evaluator.Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestUserIsExplicitlyIncludedInSegment(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").Included("foo", "bar").Build()
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUser("bar")

	result := evaluator.Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestUserIsMatchedBySegmentRule(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(lduser.NameAttribute, ldmodel.OperatorIn, ldvalue.String("Jane")))).
		Build()
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUserBuilder("key").Name("Jane").Build()

	result := evaluator.Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestUserIsExplicitlyExcludedFromSegment(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		Excluded("foo", "bar").
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(lduser.NameAttribute, ldmodel.OperatorIn, ldvalue.String("Jane")))).
		Build()
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUserBuilder("foo").Name("Jane").Build()

	result := evaluator.Evaluate(f, user, nil)
	assert.False(t, result.Value.BoolValue())
}

func TestSegmentIncludesOverrideExcludes(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		Excluded("bar").
		Included("foo", "bar").
		Build()
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUser("bar")

	result := evaluator.Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestSegmentDoesNotMatchUserIfNoIncludesOrRulesMatch(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		Included("other-key").
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(lduser.NameAttribute, ldmodel.OperatorIn, ldvalue.String("Jane")))).
		Build()
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := evaluator.Evaluate(f, user, nil)
	assert.False(t, result.Value.BoolValue())
}

func TestSegmentRuleCanMatchUserWithPercentageRollout(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(lduser.NameAttribute, ldmodel.OperatorIn, ldvalue.String("Jane"))).
			Weight(99999)).
		Build()
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUserBuilder("key").Name("Jane").Build()

	result := evaluator.Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestSegmentRuleCanNotMatchUserWithPercentageRollout(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segkey").
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(lduser.NameAttribute, ldmodel.OperatorIn, ldvalue.String("Jane"))).
			Weight(1)).
		Build()
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUserBuilder("key").Name("Jane").Build()

	result := evaluator.Evaluate(f, user, nil)
	assert.False(t, result.Value.BoolValue())
}
