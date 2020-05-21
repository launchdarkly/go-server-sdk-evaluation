package evaluation

import (
	"fmt"
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldmodel"
)

var evalBenchmarkResult ldreason.EvaluationDetail

const evalBenchmarkSegmentKey = "segment-key"

func discardPrerequisiteEvents(params PrerequisiteFlagEvent) {}

type evalBenchmarkEnv struct {
	evaluator        Evaluator
	user             lduser.User
	targetFlag       *ldmodel.FeatureFlag
	otherFlags       map[string]*ldmodel.FeatureFlag
	segments         map[string]*ldmodel.Segment
	targetFeatureKey string
	targetUsers      []lduser.User
}

type evalBenchmarkCase struct {
	numTargets      int
	numRules        int
	numClauses      int
	numSegmentUsers int
	prereqsWidth    int
	prereqsDepth    int
	operator        ldmodel.Operator
	shouldMatch     bool
}

func newEvalBenchmarkEnv() *evalBenchmarkEnv {
	return &evalBenchmarkEnv{}
}

func (env *evalBenchmarkEnv) setUp(bc evalBenchmarkCase) {
	env.evaluator = basicEvaluator()

	env.user = makeEvalBenchmarkUser(bc)

	env.targetFlag, env.otherFlags, env.segments = makeEvalBenchmarkFlagData(bc)

	dataProvider := &simpleDataProvider{
		getFlag: func(key string) (ldmodel.FeatureFlag, bool) {
			if f, ok := env.otherFlags[key]; ok {
				return *f, true
			}
			return ldmodel.FeatureFlag{}, false
		},
		getSegment: func(key string) (ldmodel.Segment, bool) {
			if s, ok := env.segments[key]; ok {
				return *s, true
			}
			return ldmodel.Segment{}, false
		},
	}
	env.evaluator = NewEvaluator(dataProvider)

	env.targetUsers = make([]lduser.User, bc.numTargets)
	for i := 0; i < bc.numTargets; i++ {
		env.targetUsers[i] = lduser.NewUser(makeEvalBenchmarkTargetUserKey(i))
	}
}

func makeEvalBenchmarkUser(bc evalBenchmarkCase) lduser.User {
	if bc.shouldMatch {
		builder := lduser.NewUserBuilder("user-match")
		switch bc.operator {
		case ldmodel.OperatorGreaterThan:
			builder.Custom("numAttr", ldvalue.Int(10000))
		case ldmodel.OperatorContains:
			builder.Name("name-0")
		case ldmodel.OperatorMatches:
			builder.Custom("stringAttr", ldvalue.String("stringAttr-0"))
		case ldmodel.OperatorAfter:
			builder.Custom("dateAttr", ldvalue.String("2999-12-31T00:00:00.000-00:00"))
		case ldmodel.OperatorSemVerEqual:
			builder.Custom("semVerAttr", ldvalue.String("1.0.0"))
		case ldmodel.OperatorIn:
			builder.Custom("stringAttr", ldvalue.String("stringAttr-0"))
		}
		return builder.Build()
	}
	// default is that the user will not be matched by any clause or target
	return lduser.NewUserBuilder("user-nomatch").
		Name("name-nomatch").
		Custom("stringAttr", ldvalue.String("stringAttr-nomatch")).
		Custom("numAttr", ldvalue.Int(0)).
		Custom("dateAttr", ldvalue.String("1980-01-01T00:00:00.000-00:00")).
		Custom("semVerAttr", ldvalue.String("0.0.5")).
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

func BenchmarkEvaluationFallthrough(b *testing.B) {
	benchmarkEval(b, makeEvalBenchmarkCases(false), func(env *evalBenchmarkEnv) {
		evalBenchmarkResult = env.evaluator.Evaluate(*env.targetFlag, env.user, discardPrerequisiteEvents)
		if evalBenchmarkResult.Value.BoolValue() { // verify that we did not get a match
			b.FailNow()
		}
	})
}

func BenchmarkEvaluationRuleMatch(b *testing.B) {
	benchmarkEval(b, makeEvalBenchmarkCases(true), func(env *evalBenchmarkEnv) {
		evalBenchmarkResult = env.evaluator.Evaluate(*env.targetFlag, env.user, discardPrerequisiteEvents)
		if !evalBenchmarkResult.Value.BoolValue() { // verify that we got a match
			b.FailNow()
		}
	})
}

func BenchmarkEvaluationUserFoundInTargets(b *testing.B) {
	// This attempts to match a user from the middle of the target list. Currently, the execution time is roughly
	// linear based on the length of the list, since we are iterating it.
	benchmarkEval(b, makeTargetMatchBenchmarkCases(), func(env *evalBenchmarkEnv) {
		user := env.targetUsers[len(env.targetUsers)/2]
		evalBenchmarkResult := env.evaluator.Evaluate(*env.targetFlag, user, discardPrerequisiteEvents)
		if !evalBenchmarkResult.Value.BoolValue() {
			b.FailNow()
		}
	})
}

func BenchmarkEvaluationUsersNotFoundInTargets(b *testing.B) {
	// This attempts to match a user who is not in the list. Currently, the execution time is roughly
	// linear based on the length of the list, since we are iterating it.
	benchmarkEval(b, makeTargetMatchBenchmarkCases(), func(env *evalBenchmarkEnv) {
		evalBenchmarkResult := env.evaluator.Evaluate(*env.targetFlag, env.user, discardPrerequisiteEvents)
		if evalBenchmarkResult.Value.BoolValue() {
			b.FailNow()
		}
	})
}

func BenchmarkEvaluationUserIncludedInSegment(b *testing.B) {
	// This attempts to match a user from the middle of the segment's include list. Currently, the execution
	// time is  roughly linear based on the length of the list, since we are iterating it.
	benchmarkEval(b, makeSegmentMatchBenchmarkCases(), func(env *evalBenchmarkEnv) {
		s := env.segments[evalBenchmarkSegmentKey]
		user := lduser.NewUser(s.Included[len(s.Included)/2])
		evalBenchmarkResult := env.evaluator.Evaluate(*env.targetFlag, user, discardPrerequisiteEvents)
		if !evalBenchmarkResult.Value.BoolValue() {
			b.FailNow()
		}
	})
}

func BenchmarkEvaluationUserNotIncludedInSegment(b *testing.B) {
	// This attempts to match a user who is not included in the segment. Currently, the execution time is
	// roughly linear based on the length of the include and exclude lists, since we are iterating them.
	benchmarkEval(b, makeSegmentMatchBenchmarkCases(), func(env *evalBenchmarkEnv) {
		evalBenchmarkResult := env.evaluator.Evaluate(*env.targetFlag, env.user, discardPrerequisiteEvents)
		if evalBenchmarkResult.Value.BoolValue() {
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
			numRules:    1,
			numClauses:  1,
			operator:    op,
			shouldMatch: shouldMatch,
		})

		// prereqs
		ret = append(ret, evalBenchmarkCase{
			numRules:     1,
			numClauses:   1,
			operator:     op,
			shouldMatch:  shouldMatch,
			prereqsWidth: 5,
			prereqsDepth: 1,
		})
		ret = append(ret, evalBenchmarkCase{
			numRules:     1,
			numClauses:   1,
			operator:     op,
			shouldMatch:  shouldMatch,
			prereqsWidth: 1,
			prereqsDepth: 5,
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

func makeEvalBenchmarkClauses(numClauses int, op ldmodel.Operator) []ldmodel.Clause {
	clauses := make([]ldmodel.Clause, 0, numClauses)
	for i := 0; i < numClauses; i++ {
		clause := ldmodel.Clause{Op: op}
		switch op {
		case ldmodel.OperatorGreaterThan:
			clause.Attribute = "numAttr"
			clause.Values = []ldvalue.Value{ldvalue.Int(i)}
		case ldmodel.OperatorContains:
			clause.Attribute = "name"
			clause.Values = []ldvalue.Value{
				ldvalue.String(fmt.Sprintf("name-%d", i)),
				ldvalue.String(fmt.Sprintf("name-%d", i+1)),
				ldvalue.String(fmt.Sprintf("name-%d", i+2)),
			}
		case ldmodel.OperatorMatches:
			clause.Attribute = "stringAttr"
			clause.Values = []ldvalue.Value{
				ldvalue.String(fmt.Sprintf("stringAttr-%d", i)),
				ldvalue.String(fmt.Sprintf("stringAttr-%d", i+1)),
				ldvalue.String(fmt.Sprintf("stringAttr-%d", i+2)),
			}
		case ldmodel.OperatorAfter:
			clause.Attribute = "dateAttr"
			clause.Values = []ldvalue.Value{
				ldvalue.String(fmt.Sprintf("%d-01-01T00:00:00.000-00:00", 2000+i)),
				ldvalue.String(fmt.Sprintf("%d-01-01T00:00:00.000-00:00", 2001+i)),
				ldvalue.String(fmt.Sprintf("%d-01-01T00:00:00.000-00:00", 2002+i)),
			}
		case ldmodel.OperatorSemVerEqual:
			clause.Attribute = "semVerAttr"
			clause.Values = []ldvalue.Value{ldvalue.String("1.0.0")}
		case ldmodel.OperatorSegmentMatch:
			clause.Values = []ldvalue.Value{ldvalue.String(evalBenchmarkSegmentKey)}
		default:
			clause.Op = ldmodel.OperatorIn
			clause.Attribute = "stringAttr"
			clause.Values = []ldvalue.Value{
				ldvalue.String(fmt.Sprintf("stringAttr-%d", i)),
				ldvalue.String(fmt.Sprintf("stringAttr-%d", i+1)),
				ldvalue.String(fmt.Sprintf("stringAttr-%d", i+2)),
			}
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

func makeSegmentMatchBenchmarkCases() []evalBenchmarkCase {
	ret := []evalBenchmarkCase{}
	for _, n := range []int{10, 100, 1000} {
		ret = append(ret, evalBenchmarkCase{
			numSegmentUsers: n,
			numRules:        1,
			numClauses:      1,
			operator:        ldmodel.OperatorSegmentMatch,
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
		builder.AddRule(ldbuilders.NewRuleBuilder().
			ID(fmt.Sprintf("%s-%d", key, j)).
			Clauses(makeEvalBenchmarkClauses(bc.numClauses, bc.operator)...).
			Variation(1))
	}
	return builder
}

func makeEvalBenchmarkFlagData(bc evalBenchmarkCase) (*ldmodel.FeatureFlag, map[string]*ldmodel.FeatureFlag, map[string]*ldmodel.Segment) {
	mainFlag := buildEvalBenchmarkFlag(bc, "flag-0")

	otherFlags := make(map[string]*ldmodel.FeatureFlag)
	if bc.prereqsDepth > 0 && bc.prereqsWidth > 0 {
		flagCounter := 1
		makeEvalBenchmarkPrerequisites(mainFlag, &flagCounter, otherFlags, bc, bc.prereqsDepth)
	}

	segments := make(map[string]*ldmodel.Segment)
	if bc.numSegmentUsers > 0 {
		sb := ldbuilders.NewSegmentBuilder(evalBenchmarkSegmentKey).Version(1)
		included := make([]string, bc.numSegmentUsers)
		for i := range included {
			included[i] = makeEvalBenchmarkTargetUserKey(i)
		}
		sb.Included(included...)
		excluded := make([]string, bc.numSegmentUsers)
		for i := range excluded {
			excluded[i] = makeEvalBenchmarkTargetUserKey(i + bc.numSegmentUsers)
		}
		sb.Excluded(excluded...)
		s := sb.Build()
		segments[s.Key] = &s
	}

	f := mainFlag.Build()
	return &f, otherFlags, segments
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
