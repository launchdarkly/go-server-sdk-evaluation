package evaluation

import (
	"fmt"
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v3/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldmodel"

	"github.com/stretchr/testify/assert"
)

const basicUserKey = "userkey"

type simpleBigSegmentProvider struct {
	getMembership         func(contextKey string) BigSegmentMembership
	getStatus             func(contextKey string) ldreason.BigSegmentsStatus
	membershipKeysQueried []string
}

func basicBigSegmentsProvider() *simpleBigSegmentProvider {
	return &simpleBigSegmentProvider{}
}

func (s *simpleBigSegmentProvider) GetUserMembership(contextKey string) (BigSegmentMembership, ldreason.BigSegmentsStatus) {
	s.membershipKeysQueried = append(s.membershipKeysQueried, contextKey)
	var membership BigSegmentMembership
	if s.getMembership != nil {
		membership = s.getMembership(contextKey)
	}
	status := ldreason.BigSegmentsHealthy
	if s.getStatus != nil {
		status = s.getStatus(contextKey)
	}
	return membership, status
}

func (s *simpleBigSegmentProvider) withStatus(status ldreason.BigSegmentsStatus) *simpleBigSegmentProvider {
	return &simpleBigSegmentProvider{
		getStatus:     func(string) ldreason.BigSegmentsStatus { return status },
		getMembership: s.getMembership,
	}
}

func (s *simpleBigSegmentProvider) withStatusForKey(key string, status ldreason.BigSegmentsStatus) *simpleBigSegmentProvider {
	previousGetStatus := s.getStatus
	return &simpleBigSegmentProvider{
		getStatus: func(queriedKey string) ldreason.BigSegmentsStatus {
			if key == queriedKey {
				return status
			}
			if previousGetStatus != nil {
				return previousGetStatus(queriedKey)
			}
			return ldreason.BigSegmentsHealthy
		},
		getMembership: s.getMembership,
	}
}

func (s *simpleBigSegmentProvider) withMembership(
	key string,
	membership *simpleUserMembership,
) *simpleBigSegmentProvider {
	previousGetMembership := s.getMembership
	return &simpleBigSegmentProvider{
		getStatus: s.getStatus,
		getMembership: func(queriedKey string) BigSegmentMembership {
			if key == queriedKey {
				return membership
			}
			if previousGetMembership != nil {
				return previousGetMembership(queriedKey)
			}
			return nil
		},
	}
}

type simpleUserMembership struct {
	segmentChecks []string
	included      []string
	excluded      []string
}

func (m *simpleUserMembership) CheckMembership(segmentRef string) ldvalue.OptionalBool {
	m.segmentChecks = append(m.segmentChecks, segmentRef)
	for _, inc := range m.included {
		if inc == segmentRef {
			return ldvalue.NewOptionalBool(true)
		}
	}
	for _, exc := range m.excluded {
		if exc == segmentRef {
			return ldvalue.NewOptionalBool(false)
		}
	}
	return ldvalue.OptionalBool{}
}

func basicUserMembership() *simpleUserMembership { return &simpleUserMembership{} }

func (s *simpleUserMembership) include(segmentRefs ...string) *simpleUserMembership {
	s.included = append(s.included, segmentRefs...)
	return s
}

func (s *simpleUserMembership) exclude(segmentRefs ...string) *simpleUserMembership {
	s.excluded = append(s.excluded, segmentRefs...)
	return s
}

func TestBigSegmentWithNoProviderIsNotMatched(t *testing.T) {
	evaluator := NewEvaluator(
		basicDataProvider().withStoredSegments(
			ldbuilders.NewSegmentBuilder("segmentkey").
				Unbounded(true).
				Generation(1).
				Included(basicUserKey). // Included should be ignored for a big segment
				Build(),
		),
	)
	f := makeBooleanFlagToMatchAnyOfSegments("segmentkey")

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(false), result.Detail.Value)
	assert.Equal(t, ldreason.BigSegmentsNotConfigured, result.Detail.Reason.GetBigSegmentsStatus())
}

func TestBigSegmentWithNoGenerationIsNotMatched(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segmentkey").
		Unbounded(true). // but we didn't set Generation
		Build()
	evaluator := NewEvaluatorWithOptions(
		basicDataProvider().withStoredSegments(segment),
		EvaluatorOptionBigSegmentProvider(basicBigSegmentsProvider().withMembership(basicUserKey,
			basicUserMembership().include(makeBigSegmentRef(&segment)))),
	)
	f := makeBooleanFlagToMatchAnyOfSegments(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(false), result.Detail.Value)
	assert.Equal(t, ldreason.BigSegmentsNotConfigured, result.Detail.Reason.GetBigSegmentsStatus())
}

func TestBigSegmentMatch(t *testing.T) {
	contextKey := "contextkey"
	segmentKey := "segmentkey"
	flag := makeBooleanFlagToMatchAnyOfSegments(segmentKey)
	makeEvaluator := func(segment ldmodel.Segment, contextMembership *simpleUserMembership) Evaluator {
		return NewEvaluatorWithOptions(
			basicDataProvider().withStoredSegments(segment),
			EvaluatorOptionBigSegmentProvider(basicBigSegmentsProvider().withMembership(contextKey, contextMembership)),
		)
	}
	for _, contextKind := range []ldcontext.Kind{"", ldcontext.DefaultKind, "other"} {
		for _, isMultiKind := range []bool{false, true} {
			t.Run(fmt.Sprintf("kind=%s, isMultiKind=%t", contextKind, isMultiKind), func(t *testing.T) {
				context := ldcontext.NewWithKind(contextKind, contextKey)
				if isMultiKind {
					context = ldcontext.NewMulti(context, ldcontext.NewWithKind("irrelevantKind", "irrelevantKey"))
				}
				contextWithoutDesiredKind := ldcontext.NewWithKind("irrelevantKind", contextKey)
				if isMultiKind {
					contextWithoutDesiredKind = ldcontext.NewMulti(contextWithoutDesiredKind,
						ldcontext.NewWithKind("irrelevantKind2", "irrelevantKey"))
				}

				t.Run("includes", func(t *testing.T) {
					segment := ldbuilders.NewSegmentBuilder(segmentKey).
						Unbounded(true).
						UnboundedContextKind(contextKind).
						Generation(2).
						Build()
					evaluator := makeEvaluator(segment, basicUserMembership().include(makeBigSegmentRef(&segment)))

					t.Run("matched by include", func(t *testing.T) {
						result := evaluator.Evaluate(&flag, context, nil)
						assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
						assert.Equal(t, ldreason.BigSegmentsHealthy, result.Detail.Reason.GetBigSegmentsStatus())
					})

					t.Run("unmatched if context does not have specified kind", func(t *testing.T) {
						result := evaluator.Evaluate(&flag, contextWithoutDesiredKind, nil)
						assert.Equal(t, ldvalue.Bool(false), result.Detail.Value)
						// BigSegmentsStatus should *not* have been set because we don't even do the big segment store
						// query if the context doesn't have the right kind.
						assert.Equal(t, ldreason.BigSegmentsStatus(""), result.Detail.Reason.GetBigSegmentsStatus())
					})
				})

				t.Run("rule match", func(t *testing.T) {
					segment := ldbuilders.NewSegmentBuilder(segmentKey).
						Unbounded(true).
						UnboundedContextKind(contextKind).
						Generation(2).
						AddRule(
							ldbuilders.NewSegmentRuleBuilder().Clauses(makeClauseToMatchAnyContextOfKind(contextKind)),
						).
						Build()
					evaluator := makeEvaluator(segment, basicUserMembership())

					t.Run("matched by rule", func(t *testing.T) {
						result := evaluator.Evaluate(&flag, context, nil)
						assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
						assert.Equal(t, ldreason.BigSegmentsHealthy, result.Detail.Reason.GetBigSegmentsStatus())
					})

					t.Run("rules ignored if context does not have specified kind", func(t *testing.T) {
						result := evaluator.Evaluate(&flag, contextWithoutDesiredKind, nil)
						assert.Equal(t, ldvalue.Bool(false), result.Detail.Value)
						// BigSegmentsStatus should *not* have been set because we don't even do the big segment store
						// query if the context doesn't have the right kind.
						assert.Equal(t, ldreason.BigSegmentsStatus(""), result.Detail.Reason.GetBigSegmentsStatus())
					})

					t.Run("exclude takes priority over rule", func(t *testing.T) {
						evaluatorWithExclude := makeEvaluator(segment,
							basicUserMembership().exclude(makeBigSegmentRef(&segment)))
						result := evaluatorWithExclude.Evaluate(&flag, context, nil)
						assert.Equal(t, ldvalue.Bool(false), result.Detail.Value)
						assert.Equal(t, ldreason.BigSegmentsHealthy, result.Detail.Reason.GetBigSegmentsStatus())
					})
				})
			})
		}
	}
}

func TestBigSegmentIsMatchedWithRuleWhenSegmentDataForUserShowsNoMatch(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segmentkey").
		Unbounded(true).
		Generation(2).
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String(basicUserKey)))).
		Build()
	evaluator := NewEvaluatorWithOptions(
		basicDataProvider().withStoredSegments(segment),
		EvaluatorOptionBigSegmentProvider(
			basicBigSegmentsProvider().withMembership(basicUserKey, basicUserMembership())),
	)
	f := makeBooleanFlagToMatchAnyOfSegments(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
	assert.Equal(t, ldreason.BigSegmentsHealthy, result.Detail.Reason.GetBigSegmentsStatus())
}

func TestBigSegmentStatusIsReturnedFromProvider(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segmentkey").
		Unbounded(true).
		Generation(2).
		Build()
	evaluator := NewEvaluatorWithOptions(
		basicDataProvider().withStoredSegments(segment),
		EvaluatorOptionBigSegmentProvider(basicBigSegmentsProvider().
			withMembership(basicUserKey, basicUserMembership().include(makeBigSegmentRef(&segment))).
			withStatus(ldreason.BigSegmentsStale)),
	)
	f := makeBooleanFlagToMatchAnyOfSegments(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
	assert.Equal(t, ldreason.BigSegmentsStale, result.Detail.Reason.GetBigSegmentsStatus())
}

func TestBigSegmentStateIsQueriedOnlyOncePerUniqueContextKey(t *testing.T) {
	segmentKey1, segmentKey2 := "segmentKey1", "segmentKey2"
	flag := makeBooleanFlagToMatchAllOfSegments(segmentKey1, segmentKey2)
	// Note: in this flag configuration, both segment keys are referenced in one clause, so if
	// segmentKey1 is a match then we should still also see it testing segmentKey2.

	t.Run("single context kind", func(t *testing.T) {
		contextKey := "contextKey"
		context := ldcontext.New(contextKey)
		segment1 := ldbuilders.NewSegmentBuilder(segmentKey1).Unbounded(true).Generation(1).Build()
		segment2 := ldbuilders.NewSegmentBuilder(segmentKey2).Unbounded(true).Generation(2).Build()
		membership := basicUserMembership().include(makeBigSegmentRef(&segment1), makeBigSegmentRef(&segment2))
		bigSegmentsProvider := basicBigSegmentsProvider().withMembership(contextKey, membership)
		evaluator := NewEvaluatorWithOptions(
			basicDataProvider().withStoredSegments(segment1, segment2),
			EvaluatorOptionBigSegmentProvider(bigSegmentsProvider),
		)

		result := evaluator.Evaluate(&flag, context, nil)

		assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
		assert.Equal(t, ldreason.BigSegmentsHealthy, result.Detail.Reason.GetBigSegmentsStatus())
		assert.Equal(t, []string{contextKey}, bigSegmentsProvider.membershipKeysQueried)
		assert.Equal(t, []string{makeBigSegmentRef(&segment1), makeBigSegmentRef(&segment2)}, membership.segmentChecks)
	})

	t.Run("two context kinds referenced, both have same key", func(t *testing.T) {
		contextKey := "contextKey"
		otherKind := ldcontext.Kind("other")
		context := ldcontext.NewMulti(
			ldcontext.New(contextKey), // default context kind
			ldcontext.NewWithKind(otherKind, contextKey),
		)
		segment1 := ldbuilders.NewSegmentBuilder(segmentKey1).Unbounded(true).Generation(1).Build()
		segment2 := ldbuilders.NewSegmentBuilder(segmentKey2).Unbounded(true).Generation(2).
			UnboundedContextKind(otherKind).Build()
		membership := basicUserMembership().include(makeBigSegmentRef(&segment1), makeBigSegmentRef(&segment2))
		bigSegmentsProvider := basicBigSegmentsProvider().withMembership(contextKey, membership)
		evaluator := NewEvaluatorWithOptions(
			basicDataProvider().withStoredSegments(segment1, segment2),
			EvaluatorOptionBigSegmentProvider(bigSegmentsProvider),
		)

		result := evaluator.Evaluate(&flag, context, nil)

		assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
		assert.Equal(t, ldreason.BigSegmentsHealthy, result.Detail.Reason.GetBigSegmentsStatus())
		assert.Equal(t, []string{contextKey}, bigSegmentsProvider.membershipKeysQueried)
		assert.Equal(t, []string{makeBigSegmentRef(&segment1), makeBigSegmentRef(&segment2)}, membership.segmentChecks)
	})

	t.Run("two context kinds referenced, each with a different key", func(t *testing.T) {
		contextKey1, contextKey2 := "contextKey1", "contextKey2"
		otherKind := ldcontext.Kind("other")
		context := ldcontext.NewMulti(
			ldcontext.New(contextKey1), // default context kind
			ldcontext.NewWithKind(otherKind, contextKey2),
		)
		segment1 := ldbuilders.NewSegmentBuilder(segmentKey1).Unbounded(true).Generation(1).Build()
		segment2 := ldbuilders.NewSegmentBuilder(segmentKey2).Unbounded(true).Generation(2).
			UnboundedContextKind(otherKind).Build()
		membershipForKey1 := basicUserMembership().include(makeBigSegmentRef(&segment1))
		membershipForKey2 := basicUserMembership().include(makeBigSegmentRef(&segment2))
		bigSegmentsProvider := basicBigSegmentsProvider().
			withMembership(contextKey1, membershipForKey1).
			withMembership(contextKey2, membershipForKey2)
		evaluator := NewEvaluatorWithOptions(
			basicDataProvider().withStoredSegments(segment1, segment2),
			EvaluatorOptionBigSegmentProvider(bigSegmentsProvider),
		)

		result := evaluator.Evaluate(&flag, context, nil)

		assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
		assert.Equal(t, ldreason.BigSegmentsHealthy, result.Detail.Reason.GetBigSegmentsStatus())
		assert.Equal(t, []string{contextKey1, contextKey2}, bigSegmentsProvider.membershipKeysQueried)
		assert.Equal(t, []string{makeBigSegmentRef(&segment1)}, membershipForKey1.segmentChecks)
		assert.Equal(t, []string{makeBigSegmentRef(&segment2)}, membershipForKey2.segmentChecks)
	})
}

func TestBigSegmentStatusWithMultipleQueries(t *testing.T) {
	// A single evaluation could end up doing more than one big segments query if there are two different
	// context keys involved. If those queries don't return the same status, we want to make sure we
	// report whichever status is most problematic: StoreError is the worst, Stale is the second worst.
	segmentKey1, segmentKey2 := "segmentKey1", "segmentKey2"
	contextKey1, contextKey2 := "contextKey1", "contextKey2"
	otherKind := ldcontext.Kind("other")
	context := ldcontext.NewMulti(
		ldcontext.New(contextKey1), // default context kind
		ldcontext.NewWithKind(otherKind, contextKey2),
	)
	segment1 := ldbuilders.NewSegmentBuilder(segmentKey1).Unbounded(true).Generation(1).Build()
	segment2 := ldbuilders.NewSegmentBuilder(segmentKey2).Unbounded(true).Generation(2).
		UnboundedContextKind(otherKind).Build()
	membershipForKey1 := basicUserMembership().include(makeBigSegmentRef(&segment1))
	membershipForKey2 := basicUserMembership().include(makeBigSegmentRef(&segment2))
	flag := makeBooleanFlagToMatchAllOfSegments(segmentKey1, segmentKey2)

	type params struct{ status1, status2, expected ldreason.BigSegmentsStatus }
	var allParams []params
	allStatuses := []ldreason.BigSegmentsStatus{ldreason.BigSegmentsHealthy, ldreason.BigSegmentsStale,
		ldreason.BigSegmentsStoreError, ldreason.BigSegmentsNotConfigured}
	for i := 0; i < len(allStatuses)-1; i++ {
		better, worse := allStatuses[i], allStatuses[i+1]
		allParams = append(allParams, params{better, worse, worse})
		allParams = append(allParams, params{worse, better, worse})
	}
	for _, p := range allParams {
		t.Run(fmt.Sprintf("%s, %s", p.status1, p.status2), func(t *testing.T) {
			bigSegmentsProvider := basicBigSegmentsProvider().
				withMembership(contextKey1, membershipForKey1).
				withMembership(contextKey2, membershipForKey2).
				withStatusForKey(contextKey1, p.status1).
				withStatusForKey(contextKey2, p.status2)
			evaluator := NewEvaluatorWithOptions(
				basicDataProvider().withStoredSegments(segment1, segment2),
				EvaluatorOptionBigSegmentProvider(bigSegmentsProvider),
			)

			result := evaluator.Evaluate(&flag, context, nil)

			assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
			assert.Equal(t, []string{contextKey1, contextKey2}, bigSegmentsProvider.membershipKeysQueried)
			assert.Equal(t, p.expected, result.Detail.Reason.GetBigSegmentsStatus())
		})
	}
}

func TestBigSegmentStatusIsReturnedWhenBigSegmentWasReferencedFromPrerequisiteFlag(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segmentkey").
		Unbounded(true).
		Generation(2).
		Build()

	f1 := makeBooleanFlagToMatchAnyOfSegments(segment.Key)
	f0 := ldbuilders.NewFlagBuilder("feature0").
		On(true).
		Variations(ldvalue.Bool(false), ldvalue.Bool(true)).FallthroughVariation(1).
		AddPrerequisite(f1.Key, 1).
		Build()

	evaluator := NewEvaluatorWithOptions(
		basicDataProvider().withStoredFlags(f1).withStoredSegments(segment),
		EvaluatorOptionBigSegmentProvider(basicBigSegmentsProvider().
			withMembership(basicUserKey, basicUserMembership().include(makeBigSegmentRef(&segment))).
			withStatus(ldreason.BigSegmentsStale)),
	)

	result := evaluator.Evaluate(&f0, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
	assert.Equal(t, ldreason.BigSegmentsStale, result.Detail.Reason.GetBigSegmentsStatus())
}
