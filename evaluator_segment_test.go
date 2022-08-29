package evaluation

import (
	"fmt"
	"testing"

	"github.com/launchdarkly/go-server-sdk-evaluation/v2/ldbuilders"
	"github.com/launchdarkly/go-server-sdk-evaluation/v2/ldmodel"

	"github.com/launchdarkly/go-sdk-common/v3/ldattr"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldlog"
	"github.com/launchdarkly/go-sdk-common/v3/ldlogtest"
	"github.com/launchdarkly/go-sdk-common/v3/ldreason"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	m "github.com/launchdarkly/go-test-helpers/v3/matchers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertSegmentMatch(t *testing.T, segment ldmodel.Segment, context ldcontext.Context, expected bool) {
	f := makeBooleanFlagToMatchAnyOfSegments(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	result := evaluator.Evaluate(&f, context, nil)
	assert.Equal(t, expected, result.Detail.Value.BoolValue())
}

type segmentMatchParams struct {
	name        string
	segment     ldmodel.Segment
	context     ldcontext.Context
	shouldMatch bool
}

func buildSegment() *ldbuilders.SegmentBuilder { return ldbuilders.NewSegmentBuilder("segmentkey") }

func doSegmentMatchTest(t *testing.T, p segmentMatchParams) {
	desc := "should not match"
	if p.shouldMatch {
		desc = "should match"
	}
	t.Run(fmt.Sprintf("%s, %s", p.name, desc), func(t *testing.T) {
		assertSegmentMatch(t, p.segment, p.context, p.shouldMatch)
	})
}

func TestSegmentMatch(t *testing.T) {
	userKey, otherKey := "key1", "key2"
	otherKind := ldcontext.Kind("kind2")

	defaultKindParams := []segmentMatchParams{
		{
			name:        "neither included nor excluded, no rules",
			segment:     buildSegment().Build(),
			shouldMatch: false,
		},
		{
			name:        "included by key",
			segment:     buildSegment().Included(otherKey, userKey).Build(),
			shouldMatch: true,
		},
		{
			name:        "included by key and also excluded",
			segment:     buildSegment().Included(userKey).Excluded(userKey).Build(),
			shouldMatch: true,
		},
		{
			name:        "includedContexts for other kinds do not apply",
			segment:     buildSegment().IncludedContextKind(otherKind, userKey).Build(),
			shouldMatch: false,
		},
		{
			name: "neither included nor excluded, rule match",
			segment: buildSegment().
				AddRule(ldbuilders.NewSegmentRuleBuilder().Clauses(
					ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String(userKey)),
				)).
				Build(),
			shouldMatch: true,
		},
		{
			name: "excluded, so rules are ignored",
			segment: buildSegment().
				Excluded(userKey).
				AddRule(ldbuilders.NewSegmentRuleBuilder().Clauses(
					ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String(userKey)),
				)).
				Build(),
			shouldMatch: false,
		},
	}

	t.Run("single-kind context of default kind", func(t *testing.T) {
		context := ldcontext.New(userKey)
		for _, p := range defaultKindParams {
			p1 := p
			p1.context = context
			doSegmentMatchTest(t, p1)
		}
	})

	t.Run("multi-kind context, targeting default kind", func(t *testing.T) {
		context := ldcontext.NewMulti(ldcontext.New(userKey), ldcontext.NewWithKind("kind2", "irrelevantKey"))
		for _, p := range defaultKindParams {
			p1 := p
			p1.context = context
			doSegmentMatchTest(t, p1)
		}
	})

	t.Run("multi-kind context, targeting non-default kind", func(t *testing.T) {
		for _, alsoHasDefault := range []bool{false, true} {
			t.Run(fmt.Sprintf("also has default: %t", alsoHasDefault), func(t *testing.T) {
				context := ldcontext.NewWithKind(otherKind, otherKey)
				if alsoHasDefault {
					context = ldcontext.NewMulti(ldcontext.New(userKey), context)
				}
				for _, p := range []segmentMatchParams{
					{
						name:        "included by key",
						segment:     buildSegment().IncludedContextKind(otherKind, otherKey).Build(),
						shouldMatch: true,
					},
					{
						name:        "default-kind included list is ignored for other kind",
						segment:     buildSegment().Included(otherKey).Build(),
						shouldMatch: false,
					},
					{
						name:        "target list for nonexistent context does not match",
						segment:     buildSegment().IncludedContextKind("nonexistentKind", otherKey).Build(),
						shouldMatch: false,
					},
					{
						name:        "included by key and also excluded",
						segment:     buildSegment().IncludedContextKind(otherKind, otherKey).ExcludedContextKind(otherKind, otherKey).Build(),
						shouldMatch: true,
					},
					{
						name: "neither included nor excluded, rule match",
						segment: buildSegment().
							AddRule(ldbuilders.NewSegmentRuleBuilder().Clauses(
								ldbuilders.ClauseWithKind(otherKind, ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String(otherKey)),
							)).
							Build(),
						shouldMatch: true,
					},
					{
						name: "excluded, so rules are ignored",
						segment: buildSegment().
							ExcludedContextKind(otherKind, otherKey).
							AddRule(ldbuilders.NewSegmentRuleBuilder().Clauses(
								ldbuilders.ClauseWithKind(otherKind, ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String(otherKey)),
							)).
							Build(),
						shouldMatch: false,
					},
				} {
					p1 := p
					p1.context = context
					doSegmentMatchTest(t, p1)
				}
			})
		}
	})

	t.Run("multi-kind context with only non-default kinds", func(t *testing.T) {
		context := ldcontext.NewMulti(
			ldcontext.NewWithKind(otherKind, otherKey),
			ldcontext.NewWithKind("irrelevantKind", "irrelevantKey"),
		)
		for _, p := range []segmentMatchParams{
			{
				name:        "included by key",
				segment:     buildSegment().IncludedContextKind(otherKind, otherKey).Build(),
				shouldMatch: true,
			},
			{
				name:        "default-kind included list is ignored for other kind",
				segment:     buildSegment().Included(otherKey).Build(),
				shouldMatch: false,
			},
			{
				name:        "included by key and also excluded",
				segment:     buildSegment().IncludedContextKind(otherKind, otherKey).ExcludedContextKind(otherKind, otherKey).Build(),
				shouldMatch: true,
			},
			{
				name: "neither included nor excluded, rule match",
				segment: buildSegment().
					AddRule(ldbuilders.NewSegmentRuleBuilder().Clauses(
						ldbuilders.ClauseWithKind(otherKind, ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String(otherKey)),
					)).
					Build(),
				shouldMatch: true,
			},
			{
				name: "excluded, so rules are ignored",
				segment: buildSegment().
					ExcludedContextKind(otherKind, otherKey).
					AddRule(ldbuilders.NewSegmentRuleBuilder().Clauses(
						ldbuilders.ClauseWithKind(otherKind, ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String(otherKey)),
					)).
					Build(),
				shouldMatch: false,
			},
		} {
			p1 := p
			p1.context = context
			doSegmentMatchTest(t, p1)
		}
	})
}

func TestSegmentMatchClauseFallsThroughIfSegmentNotFound(t *testing.T) {
	f := makeBooleanFlagToMatchAnyOfSegments("unknown-segment-key")
	evaluator := NewEvaluator(basicDataProvider().withNonexistentSegment("unknown-segment-key"))

	result := evaluator.Evaluate(&f, flagTestContext, nil)
	assert.False(t, result.Detail.Value.BoolValue())
}

func TestCanMatchJustOneSegmentFromList(t *testing.T) {
	segment := buildSegment().Included(flagTestContext.Key()).Build()
	f := makeBooleanFlagToMatchAnyOfSegments("unknown-segment-key", segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment).withNonexistentSegment("unknown-segment-key"))

	result := evaluator.Evaluate(&f, flagTestContext, nil)
	assert.True(t, result.Detail.Value.BoolValue())
}

func TestSegmentRulesCanReferenceOtherSegments(t *testing.T) {
	context1, context2, context3 := ldcontext.New("key1"), ldcontext.New("key2"), ldcontext.New("key3")

	segment0 := ldbuilders.NewSegmentBuilder("segmentkey0").
		AddRule(ldbuilders.NewSegmentRuleBuilder().Clauses(ldbuilders.SegmentMatchClause("segmentkey1"))).
		Build()
	segment1 := ldbuilders.NewSegmentBuilder("segmentkey1").
		Included(context1.Key()).
		AddRule(ldbuilders.NewSegmentRuleBuilder().Clauses(ldbuilders.SegmentMatchClause("segmentkey2"))).
		Build()
	segment2 := ldbuilders.NewSegmentBuilder("segmentkey2").
		Included(context2.Key()).
		Build()

	flag := makeBooleanFlagToMatchAnyOfSegments(segment0.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment0, segment1, segment2))

	assert.True(t, evaluator.Evaluate(&flag, context1, nil).Detail.Value.BoolValue())
	assert.True(t, evaluator.Evaluate(&flag, context2, nil).Detail.Value.BoolValue())
	assert.False(t, evaluator.Evaluate(&flag, context3, nil).Detail.Value.BoolValue())
}

func TestSegmentCycleDetection(t *testing.T) {
	for _, cycleGoesToOriginalSegment := range []bool{true, false} {
		t.Run(fmt.Sprintf("cycleGoesToOriginalFlag=%t", cycleGoesToOriginalSegment), func(t *testing.T) {

			segment0 := ldbuilders.NewSegmentBuilder("segmentkey0").
				AddRule(ldbuilders.NewSegmentRuleBuilder().Clauses(ldbuilders.SegmentMatchClause("segmentkey1"))).
				Build()
			segment1 := ldbuilders.NewSegmentBuilder("segmentkey1").
				AddRule(ldbuilders.NewSegmentRuleBuilder().Clauses(ldbuilders.SegmentMatchClause("segmentkey2"))).
				Build()
			cycleTargetKey := segment1.Key
			if cycleGoesToOriginalSegment {
				cycleTargetKey = segment0.Key
			}
			segment2 := ldbuilders.NewSegmentBuilder("segmentkey2").
				AddRule(ldbuilders.NewSegmentRuleBuilder().Clauses(ldbuilders.SegmentMatchClause(cycleTargetKey))).
				Build()

			flag := makeBooleanFlagToMatchAnyOfSegments(segment0.Key)
			logCapture := ldlogtest.NewMockLog()
			evaluator := NewEvaluatorWithOptions(
				basicDataProvider().withStoredSegments(segment0, segment1, segment2),
				EvaluatorOptionErrorLogger(logCapture.Loggers.ForLevel(ldlog.Error)),
			)

			result := evaluator.Evaluate(&flag, flagTestContext, FailOnAnyPrereqEvent(t))
			m.In(t).Assert(result, ResultDetailError(ldreason.EvalErrorMalformedFlag))

			errorLines := logCapture.GetOutput(ldlog.Error)
			require.Len(t, errorLines, 1)
			assert.Regexp(t, `.*segment rule.*circular reference`, errorLines[0])
		})
	}
}

func TestSegmentRulePercentageRollout(t *testing.T) {
	// Note: segment key and salt are significant in bucketing, so they're specified explicitly for this test
	segmentKey, salt := "segkey", "salty"
	key1, key2 := "userKeyA", "userKeyZ"
	customAttr := "attr1"
	weightCutoff := 30000
	// key1 is known to have a bucket value of 0.14574753 (14574) and therefore falls within the cutoff;
	// key2 is known to have a bucket value of 0.45679215 (45679) so it is outside of the cutoff.

	type params struct {
		kind      ldcontext.Kind
		multiKind bool
		bucketBy  string
	}
	var allParams []params
	// Note: currently we're not testing any scenarios where the target kind is not "user",
	// pending spec updates which will add support for this to the model
	for _, multiKind := range []bool{true, false} {
		for _, bucketBy := range []string{"", customAttr} {
			allParams = append(allParams, params{
				kind:      ldcontext.DefaultKind,
				multiKind: multiKind,
				bucketBy:  bucketBy,
			})
		}
	}
	for _, p := range allParams {
		t.Run(fmt.Sprintf("%+v", p), func(t *testing.T) {
			clauseMatchingAnyKeyForContextKind := ldbuilders.Negate(
				ldbuilders.ClauseWithKind(p.kind, ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String("")))
			rule := ldbuilders.NewSegmentRuleBuilder().
				Clauses(clauseMatchingAnyKeyForContextKind).
				Weight(weightCutoff)
			if p.bucketBy != "" {
				rule.BucketBy(p.bucketBy)
			}
			segment := ldbuilders.NewSegmentBuilder(segmentKey).
				AddRule(rule).
				Salt(salt).
				Build()
			makeSingleKindContext := func(key string) ldcontext.Context {
				if p.bucketBy == "" {
					return ldcontext.NewWithKind(p.kind, key)
				}
				return ldcontext.NewBuilder("irrelevantKey").Kind(p.kind).SetString(p.bucketBy, key).Build()
			}
			makeContext := makeSingleKindContext
			if p.multiKind {
				makeContext = func(key string) ldcontext.Context {
					return ldcontext.NewMulti(makeSingleKindContext(key),
						ldcontext.NewWithKind("irrelevantKind", "irrelevantKey"))
				}
			}
			assertSegmentMatch(t, segment, makeContext(key1), true)
			assertSegmentMatch(t, segment, makeContext(key2), false)
		})
	}
}

func TestSegmentRuleRolloutFailureConditions(t *testing.T) {
	t.Run("conditions that produce zero bucket value causing a match", func(t *testing.T) {
		// See comments in evaluator_segment.go about failure modes of computeBucketValue.
		// In these tests, we're setting the weight to 1 so that the rule will only match
		// if the bucket value is 0, which is incredibly unlikely to be a real hash value.

		t.Run("bucketBy attribute not found", func(t *testing.T) {
			segment := buildSegment().Salt("salty").
				AddRule(ldbuilders.NewSegmentRuleBuilder().
					Clauses(makeClauseToMatchAnyContextOfAnyKind()).
					BucketBy("unknown-attribute").
					Weight(1)).
				Build()

			context := ldcontext.New("key")
			assertSegmentMatch(t, segment, context, true)
		})

		t.Run("bucketBy attribute has invalid value type", func(t *testing.T) {
			segment := buildSegment().Salt("salty").
				AddRule(ldbuilders.NewSegmentRuleBuilder().
					Clauses(makeClauseToMatchAnyContextOfAnyKind()).
					BucketBy("attr1").
					Weight(1)).
				Build()

			context := ldcontext.NewBuilder("key").SetBool("attr1", true).Build()
			assertSegmentMatch(t, segment, context, true)
		})
	})

	t.Run("conditions that force a non-match", func(t *testing.T) {
		t.Run("context kind not found", func(t *testing.T) {
			segment := buildSegment().
				AddRule(ldbuilders.NewSegmentRuleBuilder().
					Clauses(makeClauseToMatchAnyContextOfAnyKind()).
					Weight(100000)). // this would normally always be a match
				Salt("salty").
				Build()

			t.Run("single-kind context", func(t *testing.T) {
				context := ldcontext.NewWithKind("org", "userKeyA")
				assertSegmentMatch(t, segment, context, false)
			})

			t.Run("multi-kind context", func(t *testing.T) {
				context := ldcontext.NewMulti(ldcontext.NewWithKind("org", "userKeyA"),
					ldcontext.NewWithKind("other", "userKeyA"))
				assertSegmentMatch(t, segment, context, false)
			})
		})
	})
}

func TestSegmentRuleRolloutGetsAttributesFromSpecifiedContextKind(t *testing.T) {
	segment := buildSegment().
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorContains, ldvalue.String("x"))).
			Weight(30000)).
		Salt("salty").
		Build()

	t.Run("single-kind context", func(t *testing.T) {
		context := ldcontext.NewWithKind("org", "userKeyA")
		assertSegmentMatch(t, segment, context, false)
	})

	t.Run("multi-kind context", func(t *testing.T) {
		context := ldcontext.NewMulti(ldcontext.NewWithKind("org", "userKeyA"),
			ldcontext.NewWithKind("other", "userKeyA"))
		assertSegmentMatch(t, segment, context, false)
	})
}

func TestMalformedFlagErrorForBadSegmentProperties(t *testing.T) {
	basicContext := ldcontext.New("userkey")

	type testCaseParams struct {
		name    string
		context ldcontext.Context
		segment ldmodel.Segment
		message string
	}

	for _, p := range []testCaseParams{
		{
			name:    "bucketBy with invalid attribute",
			context: basicContext,
			segment: buildSegment().
				AddRule(ldbuilders.NewSegmentRuleBuilder().
					Clauses(ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String(basicContext.Key()))).
					BucketByRef(ldattr.NewRef("///")).
					Weight(30000)).
				Salt("salty").
				Build(),
			message: "attribute reference",
		},
		{
			name:    "clause with undefined attribute",
			context: basicContext,
			segment: buildSegment().
				AddRule(ldbuilders.NewSegmentRuleBuilder().
					Clauses(ldbuilders.ClauseRef(ldattr.Ref{}, ldmodel.OperatorIn, ldvalue.String("a"))).
					BucketByRef(ldattr.NewRef("///")).
					Weight(30000)).
				Salt("salty").
				Build(),
			message: "rule clause did not specify an attribute",
		},
		{
			name:    "clause with invalid attribute reference",
			context: basicContext,
			segment: buildSegment().
				AddRule(ldbuilders.NewSegmentRuleBuilder().
					Clauses(ldbuilders.ClauseRef(ldattr.NewRef("///"), ldmodel.OperatorIn, ldvalue.String("a"))).
					BucketByRef(ldattr.NewRef("///")).
					Weight(30000)).
				Build(),
			message: "invalid attribute reference",
		},
	} {
		t.Run(p.name, func(t *testing.T) {
			flag := makeBooleanFlagToMatchAnyOfSegments(p.segment.Key)

			t.Run("returns error", func(t *testing.T) {
				e := NewEvaluator(basicDataProvider().withStoredSegments(p.segment))
				result := e.Evaluate(&flag, p.context, FailOnAnyPrereqEvent(t))

				m.In(t).Assert(result, ResultDetailError(ldreason.EvalErrorMalformedFlag))
			})

			t.Run("logs error", func(t *testing.T) {
				logCapture := ldlogtest.NewMockLog()
				e := NewEvaluatorWithOptions(basicDataProvider().withStoredSegments(p.segment),
					EvaluatorOptionErrorLogger(logCapture.Loggers.ForLevel(ldlog.Error)))
				_ = e.Evaluate(&flag, p.context, FailOnAnyPrereqEvent(t))

				errorLines := logCapture.GetOutput(ldlog.Error)
				if assert.Len(t, errorLines, 1) {
					assert.Regexp(t, `segment "`+p.segment.Key+`".*`+p.message, errorLines[0])
				}
			})
		})
	}
}
