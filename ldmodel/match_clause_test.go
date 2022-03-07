package ldmodel

import (
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"

	"github.com/stretchr/testify/assert"
)

func makeClause(attr string, op Operator, values ...ldvalue.Value) *Clause {
	return &Clause{Attribute: ldattr.NewNameRef(attr), Op: op, Values: values}
}

func TestClauseCanMatchBuiltInAttribute(t *testing.T) {
	clause := makeClause(ldattr.NameAttr, OperatorIn, ldvalue.String("Bob"))
	user := lduser.NewUserBuilder("key").Name("Bob").Build()
	assert.True(t, ClauseMatchesContext(clause, &user))
}

func TestClauseCanMatchCustomAttribute(t *testing.T) {
	clause := makeClause("legs", OperatorIn, ldvalue.Int(4))
	user := lduser.NewUserBuilder("key").Custom("legs", ldvalue.Int(4)).Build()
	assert.True(t, ClauseMatchesContext(clause, &user))
}

func TestClauseReturnsFalseForMissingAttribute(t *testing.T) {
	clause := makeClause("legs", OperatorIn, ldvalue.Int(4))
	user := lduser.NewUserBuilder("key").Name("Bob").Build()
	assert.False(t, ClauseMatchesContext(clause, &user))
}

func TestClauseMatchesIfAnyClauseValueMatches(t *testing.T) {
	clause := makeClause(ldattr.KeyAttr, OperatorIn, ldvalue.String("key1"), ldvalue.String("key2"))
	user := lduser.NewUser("key2")
	assert.True(t, ClauseMatchesContext(clause, &user))
}

func TestClauseMatchesIfAnyElementInUserArrayValueMatchesAnyClauseValue(t *testing.T) {
	clause := makeClause("pets", OperatorIn, ldvalue.String("cat"), ldvalue.String("dog"))
	user := lduser.NewUserBuilder("key").Custom("pets", ldvalue.ArrayOf(ldvalue.String("fish"), ldvalue.String("dog"))).Build()
	assert.True(t, ClauseMatchesContext(clause, &user))
}

func TestClauseDoesNotMatchIfNoElementInUserArrayValueMatchesAnyClauseValue(t *testing.T) {
	clause := makeClause("pets", OperatorIn, ldvalue.String("cat"), ldvalue.String("dog"))
	user := lduser.NewUserBuilder("key").Custom("pets", ldvalue.ArrayOf(ldvalue.String("fish"), ldvalue.String("bird"))).Build()
	assert.False(t, ClauseMatchesContext(clause, &user))
}

func TestClauseCanBeNegated(t *testing.T) {
	clause := makeClause(ldattr.NameAttr, OperatorIn, ldvalue.String("Bob"))
	clause.Negate = true
	user := lduser.NewUserBuilder("key").Name("Bob").Build()
	assert.False(t, ClauseMatchesContext(clause, &user))
}

func TestClauseForMissingAttributeIsFalseEvenIfNegated(t *testing.T) {
	clause := makeClause("legs", OperatorIn, ldvalue.Int(4))
	clause.Negate = true
	user := lduser.NewUserBuilder("key").Name("Bob").Build()
	assert.False(t, ClauseMatchesContext(clause, &user))
}

func TestClauseWithUnknownOperatorDoesNotMatch(t *testing.T) {
	clause := makeClause(ldattr.NameAttr, "doesSomethingUnsupported", ldvalue.String("Bob"))
	user := lduser.NewUserBuilder("key").Name("Bob").Build()
	assert.False(t, ClauseMatchesContext(clause, &user))
}
