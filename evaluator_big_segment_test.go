package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v3/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldmodel"

	"github.com/stretchr/testify/assert"
)

const basicUserKey = "userkey"

type simpleBigSegmentProvider struct {
	status                   ldreason.BigSegmentsStatus
	getUserMembership        func(userKey string) BigSegmentMembership
	membershipUserQueryCount int
}

func basicBigSegmentsProvider() *simpleBigSegmentProvider {
	return &simpleBigSegmentProvider{status: ldreason.BigSegmentsHealthy}
}

func (s *simpleBigSegmentProvider) GetUserMembership(userKey string) (BigSegmentMembership, ldreason.BigSegmentsStatus) {
	s.membershipUserQueryCount++
	var membership BigSegmentMembership
	if s.getUserMembership != nil {
		membership = s.getUserMembership(userKey)
	}
	return membership, s.status
}

func (s *simpleBigSegmentProvider) withStatus(status ldreason.BigSegmentsStatus) *simpleBigSegmentProvider {
	return &simpleBigSegmentProvider{status: status, getUserMembership: s.getUserMembership}
}

func (s *simpleBigSegmentProvider) withUserMembership(
	userKey string,
	membership *simpleUserMembership,
) *simpleBigSegmentProvider {
	return &simpleBigSegmentProvider{
		status: s.status,
		getUserMembership: func(key string) BigSegmentMembership {
			if key == userKey {
				return membership
			}
			if s.getUserMembership != nil {
				return s.getUserMembership(key)
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
	f := booleanFlagWithSegmentMatch("segmentkey")

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
		EvaluatorOptionBigSegmentProvider(basicBigSegmentsProvider().withUserMembership(basicUserKey,
			basicUserMembership().include(makeBigSegmentRef(&segment)))),
	)
	f := booleanFlagWithSegmentMatch(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(false), result.Detail.Value)
	assert.Equal(t, ldreason.BigSegmentsNotConfigured, result.Detail.Reason.GetBigSegmentsStatus())
}

func TestBigSegmentIsMatchedWithInclude(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segmentkey").
		Unbounded(true).
		Generation(2).
		Build()
	evaluator := NewEvaluatorWithOptions(
		basicDataProvider().withStoredSegments(segment),
		EvaluatorOptionBigSegmentProvider(basicBigSegmentsProvider().withUserMembership(basicUserKey,
			basicUserMembership().include(makeBigSegmentRef(&segment)))),
	)
	f := booleanFlagWithSegmentMatch(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
	assert.Equal(t, ldreason.BigSegmentsHealthy, result.Detail.Reason.GetBigSegmentsStatus())
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
			basicBigSegmentsProvider().withUserMembership(basicUserKey, basicUserMembership())),
	)
	f := booleanFlagWithSegmentMatch(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
	assert.Equal(t, ldreason.BigSegmentsHealthy, result.Detail.Reason.GetBigSegmentsStatus())
}

func TestBigSegmentIsMatchedWithRuleWhenSegmentDataForUserDoesNotExist(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segmentkey").
		Unbounded(true).
		Generation(2).
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String(basicUserKey)))).
		Build()
	evaluator := NewEvaluatorWithOptions(
		basicDataProvider().withStoredSegments(segment),
		EvaluatorOptionBigSegmentProvider(basicBigSegmentsProvider()),
	)
	f := booleanFlagWithSegmentMatch(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
	assert.Equal(t, ldreason.BigSegmentsHealthy, result.Detail.Reason.GetBigSegmentsStatus())
}

func TestBigSegmentIsUnmatchedByExcludeRegardlessOfRule(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segmentkey").
		Unbounded(true).
		Generation(2).
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String(basicUserKey)))).
		Build()
	evaluator := NewEvaluatorWithOptions(
		basicDataProvider().withStoredSegments(segment),
		EvaluatorOptionBigSegmentProvider(basicBigSegmentsProvider().withUserMembership(basicUserKey,
			basicUserMembership().exclude(makeBigSegmentRef(&segment)))),
	)
	f := booleanFlagWithSegmentMatch(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(false), result.Detail.Value)
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
			withUserMembership(basicUserKey, basicUserMembership().include(makeBigSegmentRef(&segment))).
			withStatus(ldreason.BigSegmentsStale)),
	)
	f := booleanFlagWithSegmentMatch(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
	assert.Equal(t, ldreason.BigSegmentsStale, result.Detail.Reason.GetBigSegmentsStatus())
}

func TestBigSegmentStateIsQueriedOnlyOncePerUserEvenIfFlagReferencesMultipleSegments(t *testing.T) {
	segment1 := ldbuilders.NewSegmentBuilder("segmentkey1").
		Unbounded(true).
		Generation(2).
		Build()
	segment2 := ldbuilders.NewSegmentBuilder("segmentkey2").
		Unbounded(true).
		Generation(3).
		Build()
	membership := basicUserMembership().include(makeBigSegmentRef(&segment2))
	bigSegmentsProvider := basicBigSegmentsProvider().withUserMembership(basicUserKey, membership)
	evaluator := NewEvaluatorWithOptions(
		basicDataProvider().withStoredSegments(segment1, segment2),
		EvaluatorOptionBigSegmentProvider(bigSegmentsProvider),
	)
	f := ldbuilders.NewFlagBuilder("flagkey").On(true).
		Variations(ldvalue.Bool(false), ldvalue.Bool(true)).FallthroughVariation(0).
		AddRule(ldbuilders.NewRuleBuilder().Variation(1).Clauses(ldbuilders.SegmentMatchClause(segment1.Key))).
		AddRule(ldbuilders.NewRuleBuilder().Variation(1).Clauses(ldbuilders.SegmentMatchClause(segment2.Key))).
		Build()

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
	assert.Equal(t, ldreason.BigSegmentsHealthy, result.Detail.Reason.GetBigSegmentsStatus())
	assert.Equal(t, 1, bigSegmentsProvider.membershipUserQueryCount)
	assert.Equal(t, []string{makeBigSegmentRef(&segment1), makeBigSegmentRef(&segment2)}, membership.segmentChecks)
}

func TestBigSegmentStatusIsReturnedWhenBigSegmentWasReferencedFromPrerequisiteFlag(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segmentkey").
		Unbounded(true).
		Generation(2).
		Build()

	f1 := booleanFlagWithSegmentMatch(segment.Key)
	f0 := ldbuilders.NewFlagBuilder("feature0").
		On(true).
		Variations(ldvalue.Bool(false), ldvalue.Bool(true)).FallthroughVariation(1).
		AddPrerequisite(f1.Key, 1).
		Build()

	evaluator := NewEvaluatorWithOptions(
		basicDataProvider().withStoredFlags(f1).withStoredSegments(segment),
		EvaluatorOptionBigSegmentProvider(basicBigSegmentsProvider().
			withUserMembership(basicUserKey, basicUserMembership().include(makeBigSegmentRef(&segment))).
			withStatus(ldreason.BigSegmentsStale)),
	)

	result := evaluator.Evaluate(&f0, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(true), result.Detail.Value)
	assert.Equal(t, ldreason.BigSegmentsStale, result.Detail.Reason.GetBigSegmentsStatus())
}
