package evaluation

import (
	"fmt"
	"testing"

	"github.com/launchdarkly/go-sdk-common/v3/ldattr"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	"github.com/launchdarkly/go-server-sdk-evaluation/v2/ldbuilders"
	"github.com/launchdarkly/go-server-sdk-evaluation/v2/ldmodel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tests in this file cover the logical behavior of clause matching *except* for the operator. The
// behavior of individual operators is covered in detail in evaluator_clause_operator_test.go, so the
// tests in this file all use the "in" operator on the assumption that the choice of operator does not
// affect the other aspects of clause behavior.

func assertClauseMatch(t *testing.T, shouldMatch bool, clause ldmodel.Clause, context ldcontext.Context) {
	match, err := makeEvalScope(context).clauseMatchesContext(&clause, evaluationStack{})
	assert.NoError(t, err)
	assert.Equal(t, shouldMatch, match)
}

type clauseMatchParams struct {
	name        string
	clause      ldmodel.Clause
	context     ldcontext.Context
	shouldMatch bool
}

func doClauseMatchTest(t *testing.T, p clauseMatchParams) {
	desc := "should not match"
	if p.shouldMatch {
		desc = "should match"
	}
	t.Run(fmt.Sprintf("%s, %s", p.name, desc), func(t *testing.T) {
		match, err := makeEvalScope(p.context).clauseMatchesContext(&p.clause, evaluationStack{})
		require.NoError(t, err)
		assert.Equal(t, p.shouldMatch, match)
	})
}

func TestClauseMatch(t *testing.T) {
	allParams := []clauseMatchParams{
		// cases that should match as long as the context kind is right
		{
			name:        "built-in: key",
			clause:      ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String("a")),
			context:     ldcontext.New("a"),
			shouldMatch: true,
		},
		{
			name:        "built-in: name",
			clause:      ldbuilders.Clause(ldattr.NameAttr, ldmodel.OperatorIn, ldvalue.String("b")),
			context:     ldcontext.NewBuilder("a").Name("b").Build(),
			shouldMatch: true,
		},
		{
			name:        "built-in: anonymous",
			clause:      ldbuilders.Clause(ldattr.AnonymousAttr, ldmodel.OperatorIn, ldvalue.Bool(true)),
			context:     ldcontext.NewBuilder("a").Anonymous(true).Build(),
			shouldMatch: true,
		},
		{
			name:        "custom",
			clause:      ldbuilders.Clause("attr1", ldmodel.OperatorIn, ldvalue.String("b")),
			context:     ldcontext.NewBuilder("a").SetString("attr1", "b").Build(),
			shouldMatch: true,
		},
		{
			name:        "single context value, multiple clause values",
			clause:      ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String("a"), ldvalue.String("b")),
			context:     ldcontext.New("b"),
			shouldMatch: true,
		},
		{
			name:   "multiple context values, single clause value",
			clause: ldbuilders.Clause("attr1", ldmodel.OperatorIn, ldvalue.String("c")),
			context: ldcontext.NewBuilder("a").SetValue("attr1",
				ldvalue.ArrayOf(ldvalue.String("b"), ldvalue.String("c"))).Build(),
			shouldMatch: true,
		},
		{
			name:   "multiple context values, multiple clause values",
			clause: ldbuilders.Clause("attr1", ldmodel.OperatorIn, ldvalue.String("c"), ldvalue.String("d")),
			context: ldcontext.NewBuilder("a").SetValue("attr1",
				ldvalue.ArrayOf(ldvalue.String("b"), ldvalue.String("c"))).Build(),
			shouldMatch: true,
		},
		{
			name:        "single value non-match negated",
			clause:      ldbuilders.Negate(ldbuilders.Clause("attr1", ldmodel.OperatorIn, ldvalue.String("b"))),
			context:     ldcontext.NewBuilder("a").SetString("attr1", "c").Build(),
			shouldMatch: true,
		},
		{
			name:        "multi-value non-match negated",
			clause:      ldbuilders.Negate(ldbuilders.Clause("attr1", ldmodel.OperatorIn, ldvalue.String("b"), ldvalue.String("c"))),
			context:     ldcontext.NewBuilder("a").SetString("attr1", "d").Build(),
			shouldMatch: true,
		},

		// cases that should never match
		{
			name:        "built-in: key",
			clause:      ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String("a")),
			context:     ldcontext.New("b"),
			shouldMatch: false,
		},
		{
			name:        "built-in: name",
			clause:      ldbuilders.Clause(ldattr.NameAttr, ldmodel.OperatorIn, ldvalue.String("b")),
			context:     ldcontext.NewBuilder("a").Name("c").Build(),
			shouldMatch: false,
		},
		{
			name:        "built-in: anonymous",
			clause:      ldbuilders.Clause(ldattr.AnonymousAttr, ldmodel.OperatorIn, ldvalue.Bool(true)),
			context:     ldcontext.NewBuilder("a").Anonymous(false).Build(),
			shouldMatch: false,
		},
		{
			name:        "custom, wrong value",
			clause:      ldbuilders.Clause("attr1", ldmodel.OperatorIn, ldvalue.String("b")),
			context:     ldcontext.NewBuilder("a").SetString("attr1", "c").Build(),
			shouldMatch: false,
		},
		{
			name:        "custom, no such attribute",
			clause:      ldbuilders.Clause("attr1", ldmodel.OperatorIn, ldvalue.String("b")),
			context:     ldcontext.NewBuilder("a").SetString("attr2", "b").Build(),
			shouldMatch: false,
		},
		{
			// If the attribute does not exist then the clause is a non-match no matter what - even if negated
			name:        "custom, no such attribute, negated",
			clause:      ldbuilders.Negate(ldbuilders.Clause("attr1", ldmodel.OperatorIn, ldvalue.String("b"))),
			context:     ldcontext.NewBuilder("a").SetString("attr2", "b").Build(),
			shouldMatch: false,
		},
		{
			name:        "single context value, multiple clause values",
			clause:      ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String("a"), ldvalue.String("b")),
			context:     ldcontext.New("c"),
			shouldMatch: false,
		},
		{
			name:   "multiple context values, single clause value",
			clause: ldbuilders.Clause("attr1", ldmodel.OperatorIn, ldvalue.String("c")),
			context: ldcontext.NewBuilder("a").SetValue("attr1",
				ldvalue.ArrayOf(ldvalue.String("b"), ldvalue.String("d"))).Build(),
			shouldMatch: false,
		},
		{
			name:   "multiple context values, multiple clause values",
			clause: ldbuilders.Clause("attr1", ldmodel.OperatorIn, ldvalue.String("c"), ldvalue.String("d")),
			context: ldcontext.NewBuilder("a").SetValue("attr1",
				ldvalue.ArrayOf(ldvalue.String("b"), ldvalue.String("e"))).Build(),
			shouldMatch: false,
		},
		{
			name:        "single value match negated",
			clause:      ldbuilders.Negate(ldbuilders.Clause("attr1", ldmodel.OperatorIn, ldvalue.String("b"))),
			context:     ldcontext.NewBuilder("a").SetString("attr1", "b").Build(),
			shouldMatch: false,
		},
		{
			name: "multi-value match negated",
			clause: ldbuilders.Negate(
				ldbuilders.Clause("attr1", ldmodel.OperatorIn, ldvalue.String("b"), ldvalue.String("c"))),
			context:     ldcontext.NewBuilder("a").SetString("attr1", "b").Build(),
			shouldMatch: false,
		},
		{
			name:        "unknown operator",
			clause:      ldbuilders.Clause(ldattr.KeyAttr, "doesSomethingUnsupported", ldvalue.String("a")),
			context:     ldcontext.New("a"),
			shouldMatch: false,
		},
	}

	t.Run("single-kind context of default kind, clause is default kind", func(t *testing.T) {
		for _, p := range allParams {
			doClauseMatchTest(t, p)
		}
	})

	t.Run("single-kind context of non-default kind, clause is same kind", func(t *testing.T) {
		for _, p := range allParams {
			p1 := p
			p1.clause.ContextKind = "org"
			p1.context = ldcontext.NewBuilderFromContext(p.context).Kind("org").Build()
			doClauseMatchTest(t, p1)
		}
	})

	t.Run("single-kind context of non-default kind, clause is default kind", func(t *testing.T) {
		for _, p := range allParams {
			p1 := p
			p1.context = ldcontext.NewBuilderFromContext(p.context).Kind("org").Build()
			p1.shouldMatch = false
			doClauseMatchTest(t, p1)
		}
	})

	t.Run("single-kind context of non-default kind, clause is different non-default kind", func(t *testing.T) {
		for _, p := range allParams {
			p1 := p
			p1.clause.ContextKind = "other"
			p1.context = ldcontext.NewBuilderFromContext(p.context).Kind("org").Build()
			p1.shouldMatch = false
			doClauseMatchTest(t, p1)
		}
	})

	t.Run("multi-kind context with default kind, clause is default kind", func(t *testing.T) {
		for _, p := range allParams {
			p1 := p
			p1.context = ldcontext.NewMulti(ldcontext.NewWithKind("org", "x"), p.context)
			doClauseMatchTest(t, p1)
		}
	})

	t.Run("multi-kind context with non-default kind, clause is default kind", func(t *testing.T) {
		for _, p := range allParams {
			p1 := p
			p1.context = ldcontext.NewMulti(ldcontext.NewWithKind("other", "x"),
				ldcontext.NewBuilderFromContext(p.context).Kind("org").Build())
			p1.shouldMatch = false
			doClauseMatchTest(t, p1)
		}
	})

	t.Run("multi-kind context with non-default kind, clause is same kind", func(t *testing.T) {
		for _, p := range allParams {
			p1 := p
			p1.clause.ContextKind = "org"
			p1.context = ldcontext.NewMulti(ldcontext.NewWithKind("other", "x"),
				ldcontext.NewBuilderFromContext(p.context).Kind("org").Build())
			doClauseMatchTest(t, p1)
		}
	})

	t.Run("multi-kind context with non-default kind, clause is different kind", func(t *testing.T) {
		for _, p := range allParams {
			p1 := p
			p1.clause.ContextKind = "whatever"
			p1.context = ldcontext.NewMulti(ldcontext.NewWithKind("other", "x"),
				ldcontext.NewBuilderFromContext(p.context).Kind("org").Build())
			p1.shouldMatch = false
			doClauseMatchTest(t, p1)
		}
	})
}

func TestClauseMatchOnKindAttribute(t *testing.T) {
	// This is a separate test suite because the rules are a little different when Attribute is
	// "kind"-- the clause's ContextKind field is irrelevant in that case, so we don't run
	// permutations of these parameters with different values of that field as we do in
	// TestClauseMatch.

	allParams := []clauseMatchParams{}

	for _, multiKindContext := range []bool{true, false} {
		context := ldcontext.New("a")
		contextDesc := "individual context of default kind"
		if multiKindContext {
			contextDesc = "multi-kind context with default kind"
			context = ldcontext.NewMulti(ldcontext.NewWithKind("irrelevantKind99", "b"), context)
		}
		allParams = append(allParams,
			clauseMatchParams{
				name:        contextDesc + ", clause wants default kind",
				clause:      ldbuilders.Clause(ldattr.KindAttr, ldmodel.OperatorIn, ldvalue.String(string(ldcontext.DefaultKind))),
				context:     context,
				shouldMatch: true,
			},
			clauseMatchParams{
				name: contextDesc + ", multiple clause values with default kind",
				clause: ldbuilders.Clause(ldattr.KindAttr, ldmodel.OperatorIn,
					ldvalue.String("irrelevantKind2"), ldvalue.String(string(ldcontext.DefaultKind))),
				context:     context,
				shouldMatch: true,
			},
			clauseMatchParams{
				name:        contextDesc + ", clause wants different kind",
				clause:      ldbuilders.Clause(ldattr.KindAttr, ldmodel.OperatorIn, ldvalue.String("irrelevantKind2")),
				context:     context,
				shouldMatch: false,
			},
			clauseMatchParams{
				name: contextDesc + ", multiple clause values without default kind",
				clause: ldbuilders.Clause(ldattr.KindAttr, ldmodel.OperatorIn,
					ldvalue.String("irrelevantKind2"), ldvalue.String("irrelevantKind3")),
				context:     context,
				shouldMatch: false,
			})
	}

	for _, multiKindContext := range []bool{true, false} {
		myContextKind := "org"
		context := ldcontext.NewWithKind(ldcontext.Kind(myContextKind), "a")
		contextDesc := "individual context of non-default kind"
		if multiKindContext {
			contextDesc = "multi-kind context with non-default kind"
			context = ldcontext.NewMulti(ldcontext.NewWithKind("irrelevantKind99", "b"), context)
		}
		allParams = append(allParams,
			clauseMatchParams{
				name:        contextDesc + ", clause wants same kind",
				clause:      ldbuilders.Clause(ldattr.KindAttr, ldmodel.OperatorIn, ldvalue.String(myContextKind)),
				context:     context,
				shouldMatch: true,
			},
			clauseMatchParams{
				name: contextDesc + ", multiple clause values with same kind",
				clause: ldbuilders.Clause(ldattr.KindAttr, ldmodel.OperatorIn,
					ldvalue.String("irrelevantKind2"), ldvalue.String("irrelevantKind1"), ldvalue.String(myContextKind)),
				context:     context,
				shouldMatch: true,
			},
			clauseMatchParams{
				name:        contextDesc + ", clause wants different kind",
				clause:      ldbuilders.Clause(ldattr.KindAttr, ldmodel.OperatorIn, ldvalue.String("irrelevantKind2")),
				context:     context,
				shouldMatch: false,
			},
			clauseMatchParams{
				name: contextDesc + ", multiple clause values without same kind",
				clause: ldbuilders.Clause(ldattr.KindAttr, ldmodel.OperatorIn,
					ldvalue.String("irrelevantKind1"), ldvalue.String("irrelevantKind2")),
				context:     context,
				shouldMatch: false,
			})
	}

	t.Run("not negated", func(t *testing.T) {
		for _, p := range allParams {
			doClauseMatchTest(t, p)
		}
	})

	t.Run("negated", func(t *testing.T) {
		for _, p := range allParams {
			p1 := p
			p1.clause = ldbuilders.Negate(p.clause)
			p1.shouldMatch = !p.shouldMatch
			doClauseMatchTest(t, p1)
		}
	})
}

func TestClauseMatchErrorConditions(t *testing.T) {
	t.Run("unspecified attribute", func(t *testing.T) {
		clause := ldbuilders.ClauseRef(ldattr.Ref{}, ldmodel.OperatorIn, ldvalue.Int(4))
		context := ldcontext.New("key")
		match, err := makeEvalScope(context).clauseMatchesContext(&clause, evaluationStack{})
		assert.Equal(t, emptyAttrRefError{}, err)
		assert.False(t, match)
	})

	t.Run("invalid attribute reference", func(t *testing.T) {
		clause := ldbuilders.ClauseRef(ldattr.NewRef("///"), ldmodel.OperatorIn, ldvalue.Int(4))
		context := ldcontext.New("key")
		match, err := makeEvalScope(context).clauseMatchesContext(&clause, evaluationStack{})
		assert.Equal(t, badAttrRefError("///"), err)
		assert.False(t, match)
	})
}
