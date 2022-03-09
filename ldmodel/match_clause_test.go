package ldmodel

import (
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"

	"github.com/stretchr/testify/assert"
)

func makeClause(attr string, op Operator, values ...ldvalue.Value) Clause {
	return Clause{Attribute: ldattr.NewNameRef(attr), Op: op, Values: values}
}

func makeClauseWithKind(kind ldcontext.Kind, attr string, op Operator, values ...ldvalue.Value) Clause {
	return Clause{Kind: kind, Attribute: ldattr.NewNameRef(attr), Op: op, Values: values}
}

func assertClauseMatch(t *testing.T, shouldMatch bool, clause Clause, context ldcontext.Context) {
	match, err := ClauseMatchesContext(&clause, &context)
	assert.NoError(t, err)
	assert.Equal(t, shouldMatch, match)
}

func TestClauseCanMatchBuiltInAttribute(t *testing.T) {
	clause := makeClause(ldattr.NameAttr, OperatorIn, ldvalue.String("Bob"))
	user := lduser.NewUserBuilder("key").Name("Bob").Build()
	assertClauseMatch(t, true, clause, user)
}

func TestClauseCanMatchCustomAttribute(t *testing.T) {
	clause := makeClause("legs", OperatorIn, ldvalue.Int(4))
	user := lduser.NewUserBuilder("key").Custom("legs", ldvalue.Int(4)).Build()
	assertClauseMatch(t, true, clause, user)
}

func TestClauseReturnsFalseForMissingAttribute(t *testing.T) {
	clause := makeClause("legs", OperatorIn, ldvalue.Int(4))
	user := lduser.NewUserBuilder("key").Name("Bob").Build()
	assertClauseMatch(t, false, clause, user)
}

func TestClauseReturnsFalseForUnspecifiedAttribute(t *testing.T) {
	clause := Clause{Op: OperatorIn, Values: []ldvalue.Value{ldvalue.Int(4)}}
	user := lduser.NewUserBuilder("key").Name("Bob").Build()
	match, err := ClauseMatchesContext(&clause, &user)
	assert.Error(t, err)
	assert.False(t, match)
}

func TestClauseReturnsErrorForInvalidAttributeReference(t *testing.T) {
	clause := Clause{Attribute: ldattr.NewRef("///"), Op: OperatorIn, Values: []ldvalue.Value{ldvalue.Int(4)}}
	user := lduser.NewUserBuilder("key").Name("Bob").Build()
	match, err := ClauseMatchesContext(&clause, &user)
	assert.Error(t, err)
	assert.False(t, match)
}

func TestClauseMatchesIfAnyClauseValueMatches(t *testing.T) {
	clause := makeClause(ldattr.KeyAttr, OperatorIn, ldvalue.String("key1"), ldvalue.String("key2"))
	user := lduser.NewUser("key2")
	assertClauseMatch(t, true, clause, user)
}

func TestClauseMatchesIfAnyElementInUserArrayValueMatchesAnyClauseValue(t *testing.T) {
	clause := makeClause("pets", OperatorIn, ldvalue.String("cat"), ldvalue.String("dog"))
	user := lduser.NewUserBuilder("key").Custom("pets", ldvalue.ArrayOf(ldvalue.String("fish"), ldvalue.String("dog"))).Build()
	assertClauseMatch(t, true, clause, user)
}

func TestClauseDoesNotMatchIfNoElementInUserArrayValueMatchesAnyClauseValue(t *testing.T) {
	clause := makeClause("pets", OperatorIn, ldvalue.String("cat"), ldvalue.String("dog"))
	user := lduser.NewUserBuilder("key").Custom("pets", ldvalue.ArrayOf(ldvalue.String("fish"), ldvalue.String("bird"))).Build()
	assertClauseMatch(t, false, clause, user)
}

func TestClauseCanBeNegated(t *testing.T) {
	clause := makeClause(ldattr.NameAttr, OperatorIn, ldvalue.String("Bob"))
	clause.Negate = true
	user := lduser.NewUserBuilder("key").Name("Bob").Build()
	assertClauseMatch(t, false, clause, user)
}

func TestClauseForMissingAttributeIsFalseEvenIfNegated(t *testing.T) {
	clause := makeClause("legs", OperatorIn, ldvalue.Int(4))
	clause.Negate = true
	user := lduser.NewUserBuilder("key").Name("Bob").Build()
	assertClauseMatch(t, false, clause, user)
}

func TestClauseWithUnknownOperatorDoesNotMatch(t *testing.T) {
	clause := makeClause(ldattr.NameAttr, "doesSomethingUnsupported", ldvalue.String("Bob"))
	user := lduser.NewUserBuilder("key").Name("Bob").Build()
	assertClauseMatch(t, false, clause, user)
}

func TestClauseForSpecificUserKind(t *testing.T) {
	t.Run("individual context match", func(t *testing.T) {
		clause := makeClauseWithKind(ldcontext.Kind("org"), ldattr.NameAttr, OperatorIn, ldvalue.String("Bobco"))
		context := ldcontext.NewBuilder("key").Kind("org").Name("Bobco").Build()
		assertClauseMatch(t, true, clause, context)
	})

	t.Run("individual context non-match", func(t *testing.T) {
		clause := makeClauseWithKind(ldcontext.Kind("org"), ldattr.NameAttr, OperatorIn, ldvalue.String("Bobco"))
		context := ldcontext.NewBuilder("key").Kind("user").Name("Bob").Build()
		assertClauseMatch(t, false, clause, context)
	})

	t.Run("multi-kind context match", func(t *testing.T) {
		clause := makeClauseWithKind(ldcontext.Kind("org"), ldattr.NameAttr, OperatorIn, ldvalue.String("Bobco"))
		context := ldcontext.NewMulti(
			ldcontext.NewBuilder("key").Kind("user").Name("Bob").Build(),
			ldcontext.NewBuilder("key").Kind("org").Name("Bobco").Build(),
		)
		assertClauseMatch(t, true, clause, context)
	})

	t.Run("multi-kind context non-match where desired kind exists", func(t *testing.T) {
		clause := makeClauseWithKind(ldcontext.Kind("org"), ldattr.NameAttr, OperatorIn, ldvalue.String("Bob"))
		context := ldcontext.NewMulti(
			ldcontext.NewBuilder("key").Kind("user").Name("Bob").Build(),
			ldcontext.NewBuilder("key").Kind("org").Name("Bobco").Build(),
		)
		assertClauseMatch(t, false, clause, context)
	})

	t.Run("multi-kind context non-match where desired kind does not exist", func(t *testing.T) {
		clause := makeClauseWithKind(ldcontext.Kind("dept"), ldattr.NameAttr, OperatorIn, ldvalue.String("Bob"))
		context := ldcontext.NewMulti(
			ldcontext.NewBuilder("key").Kind("user").Name("Bob").Build(),
			ldcontext.NewBuilder("key").Kind("org").Name("Bobco").Build(),
		)
		assertClauseMatch(t, false, clause, context)
	})
}

func TestClauseThatAppliesToKindAttribute(t *testing.T) {
	t.Run("individual context match", func(t *testing.T) {
		clause := makeClause(ldattr.KindAttr, OperatorContains, ldvalue.String("o"))
		context := ldcontext.NewBuilder("key").Kind("org").Name("Bob").Build()
		assertClauseMatch(t, true, clause, context)
	})

	t.Run("individual context non-match", func(t *testing.T) {
		clause := makeClause(ldattr.KindAttr, OperatorContains, ldvalue.String("o"))
		context := ldcontext.NewBuilder("key").Kind("user").Name("Bob").Build()
		assertClauseMatch(t, false, clause, context)
	})

	t.Run("individual context match negated", func(t *testing.T) {
		clause := makeClause(ldattr.KindAttr, OperatorContains, ldvalue.String("o"))
		clause.Negate = true
		context := ldcontext.NewBuilder("key").Kind("org").Name("Bob").Build()
		assertClauseMatch(t, false, clause, context)
	})

	t.Run("multi-kind context match", func(t *testing.T) {
		clause := makeClause(ldattr.KindAttr, OperatorContains, ldvalue.String("o"))
		context := ldcontext.NewMulti(
			ldcontext.NewBuilder("key").Kind("user").Name("Bob").Build(),
			ldcontext.NewBuilder("key").Kind("org").Name("Bobco").Build(),
		)
		assertClauseMatch(t, true, clause, context)
	})

	t.Run("multi-kind context non-match", func(t *testing.T) {
		clause := makeClause(ldattr.KindAttr, OperatorContains, ldvalue.String("z"))
		context := ldcontext.NewMulti(
			ldcontext.NewBuilder("key").Kind("user").Name("Bob").Build(),
			ldcontext.NewBuilder("key").Kind("org").Name("Bobco").Build(),
		)
		assertClauseMatch(t, false, clause, context)
	})

	t.Run("multi-kind context non-match negated", func(t *testing.T) {
		clause := makeClause(ldattr.KindAttr, OperatorContains, ldvalue.String("z"))
		clause.Negate = true
		context := ldcontext.NewMulti(
			ldcontext.NewBuilder("key").Kind("user").Name("Bob").Build(),
			ldcontext.NewBuilder("key").Kind("org").Name("Bobco").Build(),
		)
		assertClauseMatch(t, true, clause, context)
	})
}
