package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldmodel"

	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

func TestClauseCanMatchBuiltInAttribute(t *testing.T) {
	clause := ldbuilders.Clause(lduser.NameAttribute, ldmodel.OperatorIn, ldvalue.String("Bob"))
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := basicEvaluator().Evaluate(&f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestClauseCanMatchCustomAttribute(t *testing.T) {
	clause := ldbuilders.Clause(lduser.UserAttribute("legs"), ldmodel.OperatorIn, ldvalue.Int(4))
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Custom("legs", ldvalue.Int(4)).Build()

	result := basicEvaluator().Evaluate(&f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestClauseReturnsFalseForMissingAttribute(t *testing.T) {
	clause := ldbuilders.Clause(lduser.UserAttribute("legs"), ldmodel.OperatorIn, ldvalue.Int(4))
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := basicEvaluator().Evaluate(&f, user, nil)
	assert.False(t, result.Value.BoolValue())
}

func TestClauseMatchesIfAnyClauseValueMatches(t *testing.T) {
	clause := ldbuilders.Clause(lduser.KeyAttribute, ldmodel.OperatorIn, ldvalue.String("key1"), ldvalue.String("key2"))
	f := booleanFlagWithClause(clause)
	user := lduser.NewUser("key2")

	result := basicEvaluator().Evaluate(&f, user, nil)
	assert.True(t, result.Value.BoolValue())

}

func TestClauseMatchesIfAnyElementInUserArrayValueMatchesAnyClauseValue(t *testing.T) {
	clause := ldbuilders.Clause(lduser.UserAttribute("pets"), ldmodel.OperatorIn, ldvalue.String("cat"), ldvalue.String("dog"))
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Custom("pets", ldvalue.ArrayOf(ldvalue.String("fish"), ldvalue.String("dog"))).Build()

	result := basicEvaluator().Evaluate(&f, user, nil)
	assert.True(t, result.Value.BoolValue())

}

func TestClauseDoesNotMatchIfNoElementInUserArrayValueMatchesAnyClauseValue(t *testing.T) {
	clause := ldbuilders.Clause(lduser.UserAttribute("pets"), ldmodel.OperatorIn, ldvalue.String("cat"), ldvalue.String("dog"))
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Custom("pets", ldvalue.ArrayOf(ldvalue.String("fish"), ldvalue.String("bird"))).Build()

	result := basicEvaluator().Evaluate(&f, user, nil)
	assert.False(t, result.Value.BoolValue())

}

func TestClauseCanBeNegated(t *testing.T) {
	clause := ldbuilders.Negate(ldbuilders.Clause(lduser.NameAttribute, ldmodel.OperatorIn, ldvalue.String("Bob")))
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := basicEvaluator().Evaluate(&f, user, nil)
	assert.False(t, result.Value.BoolValue())
}

func TestClauseForMissingAttributeIsFalseEvenIfNegated(t *testing.T) {
	clause := ldbuilders.Negate(ldbuilders.Clause(lduser.UserAttribute("legs"), ldmodel.OperatorIn, ldvalue.Int(4)))
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := basicEvaluator().Evaluate(&f, user, nil)
	assert.False(t, result.Value.BoolValue())
}

func TestClauseWithUnknownOperatorDoesNotMatch(t *testing.T) {
	clause := ldbuilders.Clause(lduser.NameAttribute, "doesSomethingUnsupported", ldvalue.String("Bob"))
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := basicEvaluator().Evaluate(&f, user, nil)
	assert.False(t, result.Value.BoolValue())
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
