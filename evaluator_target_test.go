package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldmodel"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"

	m "github.com/launchdarkly/go-test-helpers/v2/matchers"
)

func TestFlagMatchesContextFromTargets(t *testing.T) {
	variations := []ldvalue.Value{ldvalue.String("fall"), ldvalue.String("match1"), ldvalue.String("match2"), ldvalue.String("off")}
	nonMatchVar, matchVar1, matchVar2, offVar := 0, 1, 2, 3

	makeBaseFlag := func() ldmodel.FeatureFlag {
		return ldbuilders.NewFlagBuilder("flagkey").
			Variations(variations...).On(true).OffVariation(offVar).FallthroughVariation(nonMatchVar).Build()
	}
	user := func(key string) ldcontext.Context { return ldcontext.New(key) }
	dog := func(key string) ldcontext.Context { return ldcontext.NewWithKind("dog", key) }

	assertResult := func(t *testing.T, flag ldmodel.FeatureFlag, context ldcontext.Context, expectedVariation int) {
		t.Helper()
		result := basicEvaluator().Evaluate(&flag, context, FailOnAnyPrereqEvent(t))
		expectedReason := ldreason.NewEvalReasonTargetMatch()
		if expectedVariation == nonMatchVar {
			expectedReason = ldreason.NewEvalReasonFallthrough()
		}
		m.In(t).Assert(result, EvalDetailProps(expectedVariation, variations[expectedVariation], expectedReason))
	}

	t.Run("flag has Targets only", func(t *testing.T) {
		flag := makeBaseFlag()
		flag.Targets = []ldmodel.Target{ // these targets are for the "user" kind only
			{Variation: matchVar1, Values: []string{"c"}},
			{Variation: matchVar2, Values: []string{"b", "a"}},
		}
		ldmodel.PreprocessFlag(&flag)

		assertResult(t, flag, user("a"), matchVar2)
		assertResult(t, flag, user("b"), matchVar2)
		assertResult(t, flag, user("c"), matchVar1)
		assertResult(t, flag, user("z"), nonMatchVar)

		assertResult(t, flag, ldcontext.NewMulti(dog("b"), user("a")), matchVar2)
		assertResult(t, flag, ldcontext.NewMulti(dog("a"), user("c")), matchVar1)
		assertResult(t, flag, ldcontext.NewMulti(dog("a"), user("z")), nonMatchVar)
		assertResult(t, flag, ldcontext.NewMulti(dog("a"), ldcontext.NewWithKind("cat", "b")), nonMatchVar)
	})

	t.Run("flag has Targets+ContextTargets", func(t *testing.T) {
		flag := makeBaseFlag()
		flag.Targets = []ldmodel.Target{
			{Variation: matchVar1, Values: []string{"c"}},
			{Variation: matchVar2, Values: []string{"b", "a"}},
		}
		flag.ContextTargets = []ldmodel.Target{
			{ContextKind: "dog", Variation: matchVar1, Values: []string{"a", "b"}},
			{ContextKind: "dog", Variation: matchVar2, Values: []string{"c"}},
			{ContextKind: "user", Variation: matchVar1},
			{ContextKind: "user", Variation: matchVar2},
		}
		ldmodel.PreprocessFlag(&flag)

		assertResult(t, flag, user("a"), matchVar2)
		assertResult(t, flag, user("b"), matchVar2)
		assertResult(t, flag, user("c"), matchVar1)
		assertResult(t, flag, user("z"), nonMatchVar)

		assertResult(t, flag, ldcontext.NewMulti(dog("b"), user("a")), matchVar1)   // the "dog" target takes precedence due to ordering
		assertResult(t, flag, ldcontext.NewMulti(dog("z"), user("a")), matchVar2)   // "dog" targets don't match, continue to "user" targets
		assertResult(t, flag, ldcontext.NewMulti(dog("x"), user("z")), nonMatchVar) // nothing matches
		assertResult(t, flag, ldcontext.NewMulti(dog("a"), ldcontext.NewWithKind("cat", "b")), matchVar1)
	})

}
