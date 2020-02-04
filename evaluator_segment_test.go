package evaluation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

func TestSegmentMatchClauseRetrievesSegmentFromStore(t *testing.T) {
	segment := Segment{
		Key:      "segkey",
		Included: []string{"foo"},
	}
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUser("foo")

	result := evaluator.Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestSegmentMatchClauseFallsThroughIfSegmentNotFound(t *testing.T) {
	f := booleanFlagWithSegmentMatch("unknown-segment-key")
	evaluator := NewEvaluator(basicDataProvider().withNonexistentSegment("unknown-segment-key"))
	user := lduser.NewUser("foo")

	result := evaluator.Evaluate(f, user, nil)
	assert.False(t, result.Value.BoolValue())
}

func TestCanMatchJustOneSegmentFromList(t *testing.T) {
	segment := Segment{
		Key:      "segkey",
		Included: []string{"foo"},
	}
	f := booleanFlagWithSegmentMatch("unknown-segment-key", "segkey")
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment).withNonexistentSegment("unknown-segment-key"))
	user := lduser.NewUser("foo")

	result := evaluator.Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestUserIsExplicitlyIncludedInSegment(t *testing.T) {
	segment := Segment{
		Key:      "segkey",
		Included: []string{"foo", "bar"},
	}
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUser("bar")

	result := evaluator.Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestUserIsMatchedBySegmentRule(t *testing.T) {
	segment := Segment{
		Key: "segkey",
		Rules: []SegmentRule{
			SegmentRule{
				Clauses: []Clause{
					Clause{
						Attribute: lduser.NameAttribute,
						Op:        OperatorIn,
						Values:    []ldvalue.Value{ldvalue.String("Jane")},
					},
				},
			},
		},
	}
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUserBuilder("key").Name("Jane").Build()

	result := evaluator.Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestUserIsExplicitlyExcludedFromSegment(t *testing.T) {
	segment := Segment{
		Key:      "segkey",
		Excluded: []string{"foo", "bar"},
		Rules: []SegmentRule{
			SegmentRule{
				Clauses: []Clause{
					Clause{
						Attribute: lduser.NameAttribute,
						Op:        OperatorIn,
						Values:    []ldvalue.Value{ldvalue.String("Jane")},
					},
				},
			},
		},
	}
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUserBuilder("foo").Name("Jane").Build()

	result := evaluator.Evaluate(f, user, nil)
	assert.False(t, result.Value.BoolValue())
}

func TestSegmentIncludesOverrideExcludes(t *testing.T) {
	segment := Segment{
		Key:      "segkey",
		Included: []string{"foo", "bar"},
		Excluded: []string{"bar"},
	}
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUser("bar")

	result := evaluator.Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestSegmentDoesNotMatchUserIfNoIncludesOrRulesMatch(t *testing.T) {
	segment := Segment{
		Key:      "segkey",
		Included: []string{"other-key"},
		Rules: []SegmentRule{
			SegmentRule{
				Clauses: []Clause{
					Clause{
						Attribute: lduser.NameAttribute,
						Op:        OperatorIn,
						Values:    []ldvalue.Value{ldvalue.String("Jane")},
					},
				},
			},
		},
	}
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUserBuilder("key").Name("Bob").Build()

	result := evaluator.Evaluate(f, user, nil)
	assert.False(t, result.Value.BoolValue())
}

func TestSegmentRuleCanMatchUserWithPercentageRollout(t *testing.T) {
	segment := Segment{
		Key: "segkey",
		Rules: []SegmentRule{
			SegmentRule{
				Clauses: []Clause{
					Clause{
						Attribute: lduser.NameAttribute,
						Op:        OperatorIn,
						Values:    []ldvalue.Value{ldvalue.String("Jane")},
					},
				},
				Weight: intPtr(99999),
			},
		},
	}
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUserBuilder("key").Name("Jane").Build()

	result := evaluator.Evaluate(f, user, nil)
	assert.True(t, result.Value.BoolValue())
}

func TestSegmentRuleCanNotMatchUserWithPercentageRollout(t *testing.T) {
	segment := Segment{
		Key: "segkey",
		Rules: []SegmentRule{
			SegmentRule{
				Clauses: []Clause{
					Clause{
						Attribute: lduser.NameAttribute,
						Op:        OperatorIn,
						Values:    []ldvalue.Value{ldvalue.String("Jane")},
					},
				},
				Weight: intPtr(1),
			},
		},
	}
	f := booleanFlagWithSegmentMatch(segment.Key)
	evaluator := NewEvaluator(basicDataProvider().withStoredSegments(segment))
	user := lduser.NewUserBuilder("key").Name("Jane").Build()

	result := evaluator.Evaluate(f, user, nil)
	assert.False(t, result.Value.BoolValue())
}
