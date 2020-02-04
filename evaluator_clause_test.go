package evaluation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

func TestClauseCanMatchBuiltInAttribute(t *testing.T) {
	clause := Clause{
		Attribute: "name",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.String("Bob")},
	}
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := basicEvaluator().Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestClauseCanMatchCustomAttribute(t *testing.T) {
	clause := Clause{
		Attribute: "legs",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.Int(4)},
	}
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Custom("legs", ldvalue.Int(4)).Build()

	result := basicEvaluator().Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestClauseReturnsFalseForMissingAttribute(t *testing.T) {
	clause := Clause{
		Attribute: "legs",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.Int(4)},
	}
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := basicEvaluator().Evaluate(f, user, nil)
	assert.False(t, result.Value.BoolValue())
}

func TestClauseMatchesIfAnyClauseValueMatches(t *testing.T) {
	clause := Clause{
		Attribute: "key",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.String("key1"), ldvalue.String("key2")},
	}
	f := booleanFlagWithClause(clause)
	user := lduser.NewUser("key2")

	result := basicEvaluator().Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())

}

func TestClauseMatchesIfAnyElementInUserArrayValueMatchesAnyClauseValue(t *testing.T) {
	clause := Clause{
		Attribute: "pets",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.String("cat"), ldvalue.String("dog")},
	}
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Custom("pets", ldvalue.ArrayOf(ldvalue.String("fish"), ldvalue.String("dog"))).Build()

	result := basicEvaluator().Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())

}

func TestClauseDoesNotMatchIfNoElementInUserArrayValueMatchesAnyClauseValue(t *testing.T) {
	clause := Clause{
		Attribute: "pets",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.String("cat"), ldvalue.String("dog")},
	}
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Custom("pets", ldvalue.ArrayOf(ldvalue.String("fish"), ldvalue.String("bird"))).Build()

	result := basicEvaluator().Evaluate(f, user, nil)
	assert.False(t, result.Value.BoolValue())

}

func TestClauseCanBeNegated(t *testing.T) {
	clause := Clause{
		Attribute: "name",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.String("Bob")},
		Negate:    true,
	}
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := basicEvaluator().Evaluate(f, user, nil)
	assert.False(t, result.Value.BoolValue())
}

func TestClauseForMissingAttributeIsFalseEvenIfNegated(t *testing.T) {
	clause := Clause{
		Attribute: "legs",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.Int(4)},
		Negate:    true,
	}
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := basicEvaluator().Evaluate(f, user, nil)
	assert.False(t, result.Value.BoolValue())
}

func TestClauseWithUnknownOperatorDoesNotMatch(t *testing.T) {
	clause := Clause{
		Attribute: "name",
		Op:        "doesSomethingUnsupported",
		Values:    []ldvalue.Value{ldvalue.String("Bob")},
	}
	f := booleanFlagWithClause(clause)
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := basicEvaluator().Evaluate(f, user, nil)
	assert.False(t, result.Value.BoolValue())
}

func TestClauseWithUnknownOperatorDoesNotStopSubsequentRuleFromMatching(t *testing.T) {
	badClause := Clause{
		Attribute: "name",
		Op:        "doesSomethingUnsupported",
		Values:    []ldvalue.Value{ldvalue.String("Bob")},
	}
	badRule := FlagRule{ID: "bad", Clauses: []Clause{badClause}, VariationOrRollout: VariationOrRollout{Variation: intPtr(1)}}
	goodClause := Clause{
		Attribute: "name",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.String("Bob")},
	}
	goodRule := FlagRule{ID: "good", Clauses: []Clause{goodClause}, VariationOrRollout: VariationOrRollout{Variation: intPtr(1)}}
	f := FeatureFlag{
		Key:         "feature",
		On:          true,
		Rules:       []FlagRule{badRule, goodRule},
		Fallthrough: VariationOrRollout{Variation: intPtr(0)},
		Variations:  []ldvalue.Value{ldvalue.Bool(false), ldvalue.Bool(true)},
	}
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := basicEvaluator().Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
	assert.Equal(t, ldreason.NewEvalReasonRuleMatch(1, "good"), result.Reason)
}
