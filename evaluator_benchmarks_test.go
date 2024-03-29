package evaluation

import (
	"fmt"
	"testing"

	"github.com/launchdarkly/go-server-sdk-evaluation/v3/ldbuilders"
	"github.com/launchdarkly/go-server-sdk-evaluation/v3/ldmodel"

	"github.com/launchdarkly/go-sdk-common/v3/ldattr"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
)

// Note about heap allocations:
//
// Benchmarks whose names end in "NoAlloc" are expected _not_ to cause any heap allocations (not counting
// setup work done before ResetTimer()). This is enforced by the Makefile's benchmarks target.
//
// See notes about heap allocations in CONTRIBUTING.md.

var evalBenchmarkResult Result
var evalBenchmarkErr error

const evalBenchmarkSegmentKey = "segment-key"

func discardPrerequisiteEvents(params PrerequisiteFlagEvent) {}

type evalBenchmarkEnv struct {
	evaluator        Evaluator
	user             ldcontext.Context
	targetFlag       *ldmodel.FeatureFlag
	otherFlags       map[string]*ldmodel.FeatureFlag
	targetSegment    *ldmodel.Segment
	targetFeatureKey string
	targetUsers      []ldcontext.Context
}

type evalBenchmarkCase struct {
	numTargets        int
	numRules          int
	numClauses        int
	extraClauseValues int
	withSegments      bool
	prereqsWidth      int
	prereqsDepth      int
	operator          ldmodel.Operator
	shouldMatchClause bool
}

func newEvalBenchmarkEnv() *evalBenchmarkEnv {
	return &evalBenchmarkEnv{}
}

func (env *evalBenchmarkEnv) setUp(bc evalBenchmarkCase) {
	env.evaluator = basicEvaluator()

	env.user = makeEvalBenchmarkUser(bc)

	env.targetFlag, env.otherFlags, env.targetSegment = makeEvalBenchmarkFlagData(bc)

	dataProvider := &simpleDataProvider{
		getFlag: func(key string) *ldmodel.FeatureFlag {
			return env.otherFlags[key]
		},
		getSegment: func(key string) *ldmodel.Segment {
			if key == evalBenchmarkSegmentKey {
				return env.targetSegment
			}
			return nil
		},
	}
	env.evaluator = NewEvaluator(dataProvider)

	env.targetUsers = make([]ldcontext.Context, bc.numTargets)
	for i := 0; i < bc.numTargets; i++ {
		env.targetUsers[i] = ldcontext.New(makeEvalBenchmarkTargetUserKey(i))
	}
}

func makeEvalBenchmarkUser(bc evalBenchmarkCase) ldcontext.Context {
	if bc.shouldMatchClause {
		builder := ldcontext.NewBuilder("user-match")
		switch bc.operator {
		case ldmodel.OperatorGreaterThan:
			builder.SetInt("numAttr", 10000)
		case ldmodel.OperatorContains:
			builder.Name("name-0")
		case ldmodel.OperatorMatches:
			builder.SetString("stringAttr", "stringAttr-0")
		case ldmodel.OperatorAfter:
			builder.SetString("dateAttr", "2999-12-31T00:00:00.000-00:00")
		case ldmodel.OperatorSemVerEqual:
			builder.SetString("semVerAttr", "1.0.0")
		case ldmodel.OperatorIn:
			builder.SetString("stringAttr", "stringAttr-0")
		}
		return builder.Build()
	}
	// default is that the user will not be matched by any clause or target
	return ldcontext.NewBuilder("user-nomatch").
		Name("name-nomatch").
		SetString("stringAttr", "stringAttr-nomatch").
		SetInt("numAttr", 0).
		SetString("dateAttr", "1980-01-01T00:00:00.000-00:00").
		SetString("semVerAttr", "0.0.5").
		Build()
}

func benchmarkEval(b *testing.B, cases []evalBenchmarkCase, action func(*evalBenchmarkEnv)) {
	env := newEvalBenchmarkEnv()
	for _, bc := range cases {
		env.setUp(bc)

		b.Run(fmt.Sprintf("%+v", bc), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				action(env)
			}
		})
	}
}

func BenchmarkEvaluationFallthroughNoAlloc(b *testing.B) {
	benchmarkEval(b, makeEvalBenchmarkCases(false), func(env *evalBenchmarkEnv) {
		evalBenchmarkResult = env.evaluator.Evaluate(env.targetFlag, env.user, discardPrerequisiteEvents)
		if evalBenchmarkResult.Detail.Value.BoolValue() { // verify that we did not get a match
			b.FailNow()
		}
	})
}

func BenchmarkEvaluationRuleMatchNoAlloc(b *testing.B) {
	benchmarkEval(b, makeEvalBenchmarkCases(true), func(env *evalBenchmarkEnv) {
		evalBenchmarkResult = env.evaluator.Evaluate(env.targetFlag, env.user, discardPrerequisiteEvents)
		if !evalBenchmarkResult.Detail.Value.BoolValue() { // verify that we got a match
			b.FailNow()
		}
	})
}

func BenchmarkEvaluationUserFoundInTargetsNoAlloc(b *testing.B) {
	// This attempts to match a user from the middle of the target list. As long as the flag has been
	// preprocessed, which it always should be in normal usage, this is a simple map lookup and should
	// not increase linearly with the length of the list.
	benchmarkEval(b, makeTargetMatchBenchmarkCases(), func(env *evalBenchmarkEnv) {
		user := env.targetUsers[len(env.targetUsers)/2]
		evalBenchmarkResult := env.evaluator.Evaluate(env.targetFlag, user, discardPrerequisiteEvents)
		if !evalBenchmarkResult.Detail.Value.BoolValue() {
			b.FailNow()
		}
	})
}

func BenchmarkEvaluationUsersNotFoundInTargetsNoAlloc(b *testing.B) {
	// This attempts to match a user who is not in the list.  As long as the flag has been preprocessed,
	// which it always should be in normal usage, this is a simple map lookup and should not increase
	// linearly with the length of the list.
	benchmarkEval(b, makeTargetMatchBenchmarkCases(), func(env *evalBenchmarkEnv) {
		evalBenchmarkResult := env.evaluator.Evaluate(env.targetFlag, env.user, discardPrerequisiteEvents)
		if evalBenchmarkResult.Detail.Value.BoolValue() {
			b.FailNow()
		}
	})
}

func BenchmarkEvaluationUserIncludedInSegmentNoAlloc(b *testing.B) {
	// This attempts to match a user from the middle of the segment's include list. As long as the segment
	// has been preprocessed, which it should always be in normal usage, this is a simple map lookup and
	// should not increase linearly with the length of the list.
	benchmarkEval(b, makeSegmentIncludeExcludeBenchmarkCases(), func(env *evalBenchmarkEnv) {
		user := ldcontext.New(env.targetSegment.Included[len(env.targetSegment.Included)/2])
		evalBenchmarkResult := env.evaluator.Evaluate(env.targetFlag, user, discardPrerequisiteEvents)
		if !evalBenchmarkResult.Detail.Value.BoolValue() {
			b.FailNow()
		}
	})
}

func BenchmarkEvaluationUserExcludedFromSegmentNoAlloc(b *testing.B) {
	// This attempts to match a user who is explicitly excluded from the segment.  As long as the segment
	// has been preprocessed, which it should always be in normal usage, this is a simple map lookup and
	// should not increase linearly with the length of the list.
	benchmarkEval(b, makeSegmentIncludeExcludeBenchmarkCases(), func(env *evalBenchmarkEnv) {
		user := ldcontext.New(env.targetSegment.Excluded[len(env.targetSegment.Excluded)/2])
		evalBenchmarkResult := env.evaluator.Evaluate(env.targetFlag, user, discardPrerequisiteEvents)
		if evalBenchmarkResult.Detail.Value.BoolValue() {
			b.FailNow()
		}
	})
}

func BenchmarkEvaluationUserMatchedBySegmentRuleNoAlloc(b *testing.B) {
	benchmarkEval(b, makeSegmentRuleMatchBenchmarkCases(), func(env *evalBenchmarkEnv) {
		evalBenchmarkResult := env.evaluator.Evaluate(env.targetFlag, env.user, discardPrerequisiteEvents)
		if !evalBenchmarkResult.Detail.Value.BoolValue() {
			b.FailNow()
		}
	})
}

func makeEvalBenchmarkCases(shouldMatch bool) []evalBenchmarkCase {
	ret := []evalBenchmarkCase{}
	for _, op := range []ldmodel.Operator{
		ldmodel.OperatorIn,
		ldmodel.OperatorGreaterThan,
		ldmodel.OperatorContains,
		ldmodel.OperatorMatches,
		ldmodel.OperatorAfter,
		ldmodel.OperatorSemVerEqual,
	} {
		ret = append(ret, evalBenchmarkCase{
			numRules:          1,
			numClauses:        1,
			operator:          op,
			shouldMatchClause: shouldMatch,
		})
		if shouldMatch {
			// Add a case where we have to iterate through a lot of clauses, all of which match; this is
			// meant to detect any inefficiencies in how we're iterating
			ret = append(ret, evalBenchmarkCase{
				numRules:          1,
				numClauses:        100,
				operator:          op,
				shouldMatchClause: true,
			})
		} else {
			// Add a case where we have to iterate through a lot of rules (each with one clause, since a
			// single non-matching clause short-circuits the rule) before falling through
			ret = append(ret, evalBenchmarkCase{
				numRules:   100,
				numClauses: 1,
				operator:   op,
			})
		}
		// Add a case where there is just one clause, but it has non-matching values before the last value
		ret = append(ret, evalBenchmarkCase{
			numRules:          1,
			numClauses:        1,
			extraClauseValues: 99,
			operator:          op,
			shouldMatchClause: shouldMatch,
		})

		// prereqs
		ret = append(ret, evalBenchmarkCase{
			numRules:          1,
			numClauses:        1,
			operator:          op,
			shouldMatchClause: shouldMatch,
			prereqsWidth:      5,
			prereqsDepth:      1,
		})
		ret = append(ret, evalBenchmarkCase{
			numRules:          1,
			numClauses:        1,
			operator:          op,
			shouldMatchClause: shouldMatch,
			prereqsWidth:      1,
			prereqsDepth:      5,
		})
	}
	return ret
}

func makeEvalBenchmarkSegmentKey(i int) string {
	return fmt.Sprintf("segment-%d", i)
}

func makeEvalBenchmarkTargetUserKey(i int) string {
	return fmt.Sprintf("user-%d", i)
}

func makeEvalBenchmarkClauses(numClauses int, extraClauseValues int, op ldmodel.Operator) []ldmodel.Clause {
	clauses := make([]ldmodel.Clause, 0, numClauses)
	for i := 0; i < numClauses; i++ {
		clause := ldmodel.Clause{Op: op}
		var value ldvalue.Value
		var name string
		switch op {
		case ldmodel.OperatorGreaterThan:
			name = "numAttr"
			value = ldvalue.Int(i)
		case ldmodel.OperatorContains:
			name = "name"
			value = ldvalue.String("name-0")
		case ldmodel.OperatorMatches:
			name = "stringAttr"
			value = ldvalue.String("stringAttr-0")
		case ldmodel.OperatorAfter:
			name = "dateAttr"
			value = ldvalue.String("2000-01-01T00:00:00.000-00:00")
		case ldmodel.OperatorSemVerEqual:
			name = "semVerAttr"
			value = ldvalue.String("1.0.0")
		case ldmodel.OperatorSegmentMatch:
			value = ldvalue.String(evalBenchmarkSegmentKey)
		default:
			clause.Op = ldmodel.OperatorIn
			name = "stringAttr"
			value = ldvalue.String("stringAttr-0")
		}
		if name != "" {
			clause.Attribute = ldattr.NewLiteralRef(name)
		}
		if extraClauseValues == 0 {
			clause.Values = []ldvalue.Value{value}
		} else {
			for i := 0; i < extraClauseValues; i++ {
				clause.Values = append(clause.Values, ldvalue.String("not-a-match"))
			}
			clause.Values = append(clause.Values, value)
		}
		clauses = append(clauses, clause)
	}
	return clauses
}

func makeTargetMatchBenchmarkCases() []evalBenchmarkCase {
	return []evalBenchmarkCase{
		{numTargets: 10},
		{numTargets: 100},
		{numTargets: 1000},
	}
}

func makeSegmentIncludeExcludeBenchmarkCases() []evalBenchmarkCase {
	// Add cases to verify the performance of include/exclude matching, regardless of segment rules
	ret := []evalBenchmarkCase{}
	for _, n := range []int{10, 100, 1000} {
		ret = append(ret, evalBenchmarkCase{
			withSegments:      true,
			numTargets:        n,
			numRules:          1,
			numClauses:        1,
			shouldMatchClause: false,
		})
	}
	return ret
}

func makeSegmentRuleMatchBenchmarkCases() []evalBenchmarkCase {
	// Add cases to verify the performance of segment rules, with no include/exclude matching
	ret := []evalBenchmarkCase{}
	for _, operator := range []ldmodel.Operator{ldmodel.OperatorIn, ldmodel.OperatorMatches} {
		ret = append(ret, evalBenchmarkCase{
			withSegments:      true,
			numTargets:        0,
			numRules:          1,
			numClauses:        1,
			operator:          operator,
			shouldMatchClause: true,
		})
	}
	return ret
}

func buildEvalBenchmarkFlag(bc evalBenchmarkCase, key string) *ldbuilders.FlagBuilder {
	// All of the flags in these benchmarks are boolean flags with variations [false, true]. This is
	// because the process of evaluation at this level does not differ in any way based on the type or
	// number of the variations; that only affects the higher-level SDK logic.
	builder := ldbuilders.NewFlagBuilder("flag-0").
		Version(1).
		On(true).
		FallthroughVariation(0).
		Variations(ldvalue.Bool(false), ldvalue.Bool(true))
	if bc.numTargets > 0 {
		values := make([]string, bc.numTargets)
		for k := 0; k < bc.numTargets; k++ {
			values[k] = makeEvalBenchmarkTargetUserKey(k)
		}
		builder.AddTarget(1, values...)
	}
	for j := 0; j < bc.numRules; j++ {
		operator := bc.operator
		if bc.withSegments {
			operator = ldmodel.OperatorSegmentMatch
		}
		builder.AddRule(ldbuilders.NewRuleBuilder().
			ID(fmt.Sprintf("%s-%d", key, j)).
			Clauses(makeEvalBenchmarkClauses(bc.numClauses, bc.extraClauseValues, operator)...).
			Variation(1))
	}
	return builder
}

func makeEvalBenchmarkFlagData(bc evalBenchmarkCase) (*ldmodel.FeatureFlag, map[string]*ldmodel.FeatureFlag, *ldmodel.Segment) {
	mainFlag := buildEvalBenchmarkFlag(bc, "flag-0")

	otherFlags := make(map[string]*ldmodel.FeatureFlag)
	if bc.prereqsDepth > 0 && bc.prereqsWidth > 0 {
		flagCounter := 1
		makeEvalBenchmarkPrerequisites(mainFlag, &flagCounter, otherFlags, bc, bc.prereqsDepth)
	}

	var segment *ldmodel.Segment
	if bc.withSegments {
		sb := ldbuilders.NewSegmentBuilder(evalBenchmarkSegmentKey).Version(1)
		included := make([]string, bc.numTargets)
		for i := range included {
			included[i] = makeEvalBenchmarkTargetUserKey(i)
		}
		sb.Included(included...)
		excluded := make([]string, bc.numTargets)
		for i := range excluded {
			excluded[i] = makeEvalBenchmarkTargetUserKey(i + bc.numTargets)
		}
		sb.Excluded(excluded...)
		sb.AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(makeEvalBenchmarkClauses(bc.numClauses, bc.extraClauseValues, bc.operator)...))
		s := sb.Build()
		segment = &s
	}

	f := mainFlag.Build()
	return &f, otherFlags, segment
}

// When we test prerequisite matching, we want all of the prerequisite flags to be a match, because
// otherwise they will short-circuit evaluation of the main flag and we won't really be testing
// anything except the first prerequisite. Since we already test rule and target matching in other
// benchmarks, the prerequisites can just be fallthroughs.
func makeEvalBenchmarkPrerequisites(
	mainFlag *ldbuilders.FlagBuilder,
	flagCounter *int,
	otherFlags map[string]*ldmodel.FeatureFlag,
	bc evalBenchmarkCase,
	remainingDepth int,
) {
	for i := 0; i < bc.prereqsWidth; i++ {
		prereqBuilder := ldbuilders.NewFlagBuilder(fmt.Sprintf("flag-%d", *flagCounter)).
			Version(1).
			On(true).
			FallthroughVariation(1).
			Variations(ldvalue.Bool(false), ldvalue.Bool(true))
		*flagCounter++
		if remainingDepth > 1 {
			makeEvalBenchmarkPrerequisites(prereqBuilder, flagCounter, otherFlags, bc, remainingDepth-1)
		}
		prereqFlag := prereqBuilder.Build()
		otherFlags[prereqFlag.Key] = &prereqFlag
		mainFlag.AddPrerequisite(prereqFlag.Key, 1)
	}
}
