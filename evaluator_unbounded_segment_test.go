package evaluation

import (
	"fmt"
	"testing"

	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldmodel"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"

	"github.com/stretchr/testify/assert"
)

const basicUserKey = "userkey"

func makeSegmentRef(s ldmodel.Segment) string {
	return fmt.Sprintf("%s:%d", s.Key, s.Generation.IntValue())
}

type simpleUnboundedSegmentProvider struct {
	status                   ldreason.UnboundedSegmentsStatus
	getUserMembership        func(userKey string) UnboundedSegmentMembership
	membershipUserQueryCount int
}

func basicUnboundedSegmentsProvider() *simpleUnboundedSegmentProvider {
	return &simpleUnboundedSegmentProvider{status: ldreason.UnboundedSegmentsHealthy}
}

func (s *simpleUnboundedSegmentProvider) GetUserMembership(userKey string) (UnboundedSegmentMembership, ldreason.UnboundedSegmentsStatus) {
	s.membershipUserQueryCount++
	var membership UnboundedSegmentMembership
	if s.getUserMembership != nil {
		membership = s.getUserMembership(userKey)
	}
	return membership, s.status
}

func (s *simpleUnboundedSegmentProvider) withStatus(status ldreason.UnboundedSegmentsStatus) *simpleUnboundedSegmentProvider {
	return &simpleUnboundedSegmentProvider{status: status, getUserMembership: s.getUserMembership}
}

func (s *simpleUnboundedSegmentProvider) withUserMembership(
	userKey string,
	membership *simpleUserMembership,
) *simpleUnboundedSegmentProvider {
	return &simpleUnboundedSegmentProvider{
		status: s.status,
		getUserMembership: func(key string) UnboundedSegmentMembership {
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

func TestUnboundedSegmentWithNoProviderIsNotMatched(t *testing.T) {
	evaluator := NewEvaluator(
		basicDataProvider().withStoredSegments(
			ldbuilders.NewSegmentBuilder("segmentkey").
				Unbounded(true).
				Generation(1).
				Included(basicUserKey). // Included should be ignored for an unbounded segment
				Build(),
		),
	)
	f := booleanFlagWithSegmentMatch("segmentkey")

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(false), result.Value)
	assert.Equal(t, ldreason.UnboundedSegmentsNotConfigured, result.Reason.GetUnboundedSegmentsStatus())
}

func TestUnboundedSegmentWithNoGenerationIsNotMatched(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segmentkey").
		Unbounded(true). // but we didn't set Generation
		Build()
	evaluator := NewEvaluatorWithUnboundedSegments(
		basicDataProvider().withStoredSegments(segment),
		basicUnboundedSegmentsProvider().withUserMembership(basicUserKey,
			basicUserMembership().include(makeSegmentRef(segment))),
	)
	f := booleanFlagWithSegmentMatch(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(false), result.Value)
	assert.Equal(t, ldreason.UnboundedSegmentsNotConfigured, result.Reason.GetUnboundedSegmentsStatus())
}

func TestUnboundedSegmentIsMatchedWithInclude(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segmentkey").
		Unbounded(true).
		Generation(2).
		Build()
	evaluator := NewEvaluatorWithUnboundedSegments(
		basicDataProvider().withStoredSegments(segment),
		basicUnboundedSegmentsProvider().withUserMembership(basicUserKey,
			basicUserMembership().include(makeSegmentRef(segment))),
	)
	f := booleanFlagWithSegmentMatch(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(true), result.Value)
	assert.Equal(t, ldreason.UnboundedSegmentsHealthy, result.Reason.GetUnboundedSegmentsStatus())
}

func TestUnboundedSegmentIsMatchedWithRule(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segmentkey").
		Unbounded(true).
		Generation(2).
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(lduser.KeyAttribute, ldmodel.OperatorIn, ldvalue.String(basicUserKey)))).
		Build()
	evaluator := NewEvaluatorWithUnboundedSegments(
		basicDataProvider().withStoredSegments(segment),
		basicUnboundedSegmentsProvider().withUserMembership(basicUserKey, basicUserMembership()),
	)
	f := booleanFlagWithSegmentMatch(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(true), result.Value)
	assert.Equal(t, ldreason.UnboundedSegmentsHealthy, result.Reason.GetUnboundedSegmentsStatus())
}

func TestUnboundedSegmentIsUnmatchedByExcludeRegardlessOfRule(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segmentkey").
		Unbounded(true).
		Generation(2).
		AddRule(ldbuilders.NewSegmentRuleBuilder().
			Clauses(ldbuilders.Clause(lduser.KeyAttribute, ldmodel.OperatorIn, ldvalue.String(basicUserKey)))).
		Build()
	evaluator := NewEvaluatorWithUnboundedSegments(
		basicDataProvider().withStoredSegments(segment),
		basicUnboundedSegmentsProvider().withUserMembership(basicUserKey,
			basicUserMembership().exclude(makeSegmentRef(segment))),
	)
	f := booleanFlagWithSegmentMatch(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(false), result.Value)
	assert.Equal(t, ldreason.UnboundedSegmentsHealthy, result.Reason.GetUnboundedSegmentsStatus())
}

func TestUnboundedSegmentStatusIsReturnedFromProvider(t *testing.T) {
	segment := ldbuilders.NewSegmentBuilder("segmentkey").
		Unbounded(true).
		Generation(2).
		Build()
	evaluator := NewEvaluatorWithUnboundedSegments(
		basicDataProvider().withStoredSegments(segment),
		basicUnboundedSegmentsProvider().
			withUserMembership(basicUserKey, basicUserMembership().include(makeSegmentRef(segment))).
			withStatus(ldreason.UnboundedSegmentsStale),
	)
	f := booleanFlagWithSegmentMatch(segment.Key)

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(true), result.Value)
	assert.Equal(t, ldreason.UnboundedSegmentsStale, result.Reason.GetUnboundedSegmentsStatus())
}

func TestUnboundedSegmentStateIsQueriedOnlyOncePerUserEvenIfFlagReferencesMultipleSegments(t *testing.T) {
	segment1 := ldbuilders.NewSegmentBuilder("segmentkey1").
		Unbounded(true).
		Generation(2).
		Build()
	segment2 := ldbuilders.NewSegmentBuilder("segmentkey2").
		Unbounded(true).
		Generation(3).
		Build()
	membership := basicUserMembership().include(makeSegmentRef(segment2))
	unboundedSegmentsProvider := basicUnboundedSegmentsProvider().withUserMembership(basicUserKey, membership)
	evaluator := NewEvaluatorWithUnboundedSegments(
		basicDataProvider().withStoredSegments(segment1, segment2),
		unboundedSegmentsProvider,
	)
	f := ldbuilders.NewFlagBuilder("flagkey").On(true).
		Variations(ldvalue.Bool(false), ldvalue.Bool(true)).FallthroughVariation(0).
		AddRule(ldbuilders.NewRuleBuilder().Variation(1).Clauses(ldbuilders.SegmentMatchClause(segment1.Key))).
		AddRule(ldbuilders.NewRuleBuilder().Variation(1).Clauses(ldbuilders.SegmentMatchClause(segment2.Key))).
		Build()

	result := evaluator.Evaluate(&f, lduser.NewUser(basicUserKey), nil)
	assert.Equal(t, ldvalue.Bool(true), result.Value)
	assert.Equal(t, ldreason.UnboundedSegmentsHealthy, result.Reason.GetUnboundedSegmentsStatus())
	assert.Equal(t, 1, unboundedSegmentsProvider.membershipUserQueryCount)
	assert.Equal(t, []string{makeSegmentRef(segment1), makeSegmentRef(segment2)}, membership.segmentChecks)
}
