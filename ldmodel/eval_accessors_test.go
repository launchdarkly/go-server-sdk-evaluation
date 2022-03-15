package ldmodel

import (
	"fmt"
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"

	"github.com/stretchr/testify/assert"
)

func TestClauseFindValue(t *testing.T) {
	foundValues := []ldvalue.Value{ldvalue.Bool(true), ldvalue.Int(2), ldvalue.String("x")}
	notFoundValues := []ldvalue.Value{ldvalue.Bool(false), ldvalue.Int(3), ldvalue.String("y")}

	for _, withPreprocessing := range []bool{false, true} {
		t.Run(fmt.Sprintf("preprocessed: %t", withPreprocessing), func(t *testing.T) {
			clause := Clause{Op: OperatorIn, Values: foundValues}
			if withPreprocessing {
				clause.preprocessed = preprocessClause(clause)
			}
			for _, value := range foundValues {
				assert.True(t, EvaluatorAccessors.ClauseFindValue(&clause, value), "value: %s", value)
			}
			for _, value := range notFoundValues {
				assert.False(t, EvaluatorAccessors.ClauseFindValue(&clause, value), "value: %s", value)
			}
		})
	}

	t.Run("unsupported value types return false", func(t *testing.T) {
		badValues := []ldvalue.Value{ldvalue.Null(), ldvalue.ArrayOf(), ldvalue.ObjectBuild().Build()}
		for _, withPreprocessing := range []bool{false, true} {
			t.Run(fmt.Sprintf("preprocessed: %t", withPreprocessing), func(t *testing.T) {
				clause := Clause{Op: OperatorIn, Values: badValues}
				if withPreprocessing {
					clause.preprocessed = preprocessClause(clause)
				}
				for _, value := range badValues {
					assert.False(t, EvaluatorAccessors.ClauseFindValue(&clause, value), "value: %s", value)
				}
			})
		}
	})

	t.Run("nil pointer", func(t *testing.T) {
		assert.False(t, EvaluatorAccessors.ClauseFindValue(nil, ldvalue.String("")))
	})
}

func TestClauseGetValueAsRegexp(t *testing.T) {
	for _, withPreprocessing := range []bool{false, true} {
		t.Run(fmt.Sprintf("preprocessed: %t", withPreprocessing), func(t *testing.T) {
			clause := Clause{Op: OperatorMatches,
				Values: []ldvalue.Value{ldvalue.String("a.*b"), ldvalue.String("**"), ldvalue.Int(1)}}
			if withPreprocessing {
				clause.preprocessed = preprocessClause(clause)
			}

			r := EvaluatorAccessors.ClauseGetValueAsRegexp(&clause, 0)
			if assert.NotNil(t, r) {
				assert.Equal(t, "a.*b", r.String())
			}

			assert.Nil(t, EvaluatorAccessors.ClauseGetValueAsRegexp(&clause, 1))
			assert.Nil(t, EvaluatorAccessors.ClauseGetValueAsRegexp(&clause, 2))

			assert.Nil(t, EvaluatorAccessors.ClauseGetValueAsRegexp(&clause, -1)) // out of range
			assert.Nil(t, EvaluatorAccessors.ClauseGetValueAsRegexp(&clause, 3))  // out of range
		})
	}

	t.Run("nil pointer", func(t *testing.T) {
		assert.Nil(t, EvaluatorAccessors.ClauseGetValueAsRegexp(nil, 0))
	})
}

func TestClauseGetValueAsSemanticVersion(t *testing.T) {
	for _, withPreprocessing := range []bool{false, true} {
		t.Run(fmt.Sprintf("preprocessed: %t", withPreprocessing), func(t *testing.T) {
			clause := Clause{Op: OperatorSemVerEqual,
				Values: []ldvalue.Value{ldvalue.String("1.2.3"), ldvalue.Int(100000)}}
			if withPreprocessing {
				clause.preprocessed = preprocessClause(clause)
			}

			result, ok := EvaluatorAccessors.ClauseGetValueAsSemanticVersion(&clause, 0)
			assert.True(t, ok)
			expected, _ := TypeConversions.ValueToSemanticVersion(clause.Values[0])
			assert.Equal(t, expected, result)

			_, ok = EvaluatorAccessors.ClauseGetValueAsSemanticVersion(&clause, 1)
			assert.False(t, ok)

			_, ok = EvaluatorAccessors.ClauseGetValueAsSemanticVersion(&clause, -1) // out of range
			assert.False(t, ok)
			_, ok = EvaluatorAccessors.ClauseGetValueAsSemanticVersion(&clause, 2) // out of range
			assert.False(t, ok)
		})
	}

	t.Run("nil pointer", func(t *testing.T) {
		_, ok := EvaluatorAccessors.ClauseGetValueAsSemanticVersion(nil, 0)
		assert.False(t, ok)
	})
}

func TestClauseGetValueAsTimestamp(t *testing.T) {
	for _, withPreprocessing := range []bool{false, true} {
		t.Run(fmt.Sprintf("preprocessed: %t", withPreprocessing), func(t *testing.T) {
			clause := Clause{Op: OperatorBefore,
				Values: []ldvalue.Value{ldvalue.String("1970-01-01T00:00:00Z"), ldvalue.Int(100000), ldvalue.Bool(true)}}
			if withPreprocessing {
				clause.preprocessed = preprocessClause(clause)
			}

			result, ok := EvaluatorAccessors.ClauseGetValueAsTimestamp(&clause, 0)
			assert.True(t, ok)
			expected, _ := TypeConversions.ValueToTimestamp(clause.Values[0])
			assert.Equal(t, expected, result)

			result, ok = EvaluatorAccessors.ClauseGetValueAsTimestamp(&clause, 1)
			assert.True(t, ok)
			expected, _ = TypeConversions.ValueToTimestamp(clause.Values[1])
			assert.Equal(t, expected, result)

			_, ok = EvaluatorAccessors.ClauseGetValueAsTimestamp(&clause, 2)
			assert.False(t, ok)

			_, ok = EvaluatorAccessors.ClauseGetValueAsTimestamp(&clause, -1) // out of range
			assert.False(t, ok)
			_, ok = EvaluatorAccessors.ClauseGetValueAsTimestamp(&clause, 3) // out of range
			assert.False(t, ok)
		})
	}

	t.Run("nil pointer", func(t *testing.T) {
		_, ok := EvaluatorAccessors.ClauseGetValueAsTimestamp(nil, 0)
		assert.False(t, ok)
	})
}

func TestSegmentFindKeyInExcluded(t *testing.T) {
	foundValues := []string{"a", "b", "c"}
	notFoundValues := []string{"d", "e", "f"}

	for _, withPreprocessing := range []bool{false, true} {
		t.Run(fmt.Sprintf("preprocessed: %t", withPreprocessing), func(t *testing.T) {
			segment := Segment{Excluded: foundValues}
			if withPreprocessing {
				PreprocessSegment(&segment)
			}
			for _, value := range foundValues {
				assert.True(t, EvaluatorAccessors.SegmentFindKeyInExcluded(&segment, value), value)
			}
			for _, value := range notFoundValues {
				assert.False(t, EvaluatorAccessors.SegmentFindKeyInExcluded(&segment, value), value)
			}
		})
	}

	t.Run("nil pointer", func(t *testing.T) {
		assert.False(t, EvaluatorAccessors.SegmentFindKeyInExcluded(nil, ""))
	})
}

func TestSegmentFindValueInIncluded(t *testing.T) {
	foundValues := []string{"a", "b", "c"}
	notFoundValues := []string{"d", "e", "f"}

	for _, withPreprocessing := range []bool{false, true} {
		t.Run(fmt.Sprintf("preprocessed: %t", withPreprocessing), func(t *testing.T) {
			segment := Segment{Included: foundValues}
			if withPreprocessing {
				PreprocessSegment(&segment)
			}
			for _, value := range foundValues {
				assert.True(t, EvaluatorAccessors.SegmentFindKeyInIncluded(&segment, value), value)
			}
			for _, value := range notFoundValues {
				assert.False(t, EvaluatorAccessors.SegmentFindKeyInIncluded(&segment, value), value)
			}
		})
	}

	t.Run("nil pointer", func(t *testing.T) {
		assert.False(t, EvaluatorAccessors.SegmentFindKeyInIncluded(nil, ""))
	})
}

func TestSegmentTargetFindKey(t *testing.T) {
	foundValues := []string{"a", "b", "c"}
	notFoundValues := []string{"d", "e", "f"}

	for _, withPreprocessing := range []bool{false, true} {
		t.Run(fmt.Sprintf("preprocessed: %t", withPreprocessing), func(t *testing.T) {
			target := SegmentTarget{Values: foundValues}
			if withPreprocessing {
				target.preprocessed.valuesMap = preprocessStringSet(target.Values)
			}
			for _, value := range foundValues {
				assert.True(t, EvaluatorAccessors.SegmentTargetFindKey(&target, value), value)
			}
			for _, value := range notFoundValues {
				assert.False(t, EvaluatorAccessors.SegmentTargetFindKey(&target, value), value)
			}
		})
	}

	t.Run("nil pointer", func(t *testing.T) {
		assert.False(t, EvaluatorAccessors.SegmentTargetFindKey(nil, ""))
	})
}

func TestTargetFindKey(t *testing.T) {
	foundValues := []string{"a", "b", "c"}
	notFoundValues := []string{"d", "e", "f"}

	for _, withPreprocessing := range []bool{false, true} {
		t.Run(fmt.Sprintf("preprocessed: %t", withPreprocessing), func(t *testing.T) {
			target := Target{Values: foundValues}
			if withPreprocessing {
				target.preprocessed.valuesMap = preprocessStringSet(target.Values)
			}
			for _, value := range foundValues {
				assert.True(t, EvaluatorAccessors.TargetFindKey(&target, value), value)
			}
			for _, value := range notFoundValues {
				assert.False(t, EvaluatorAccessors.TargetFindKey(&target, value), value)
			}
		})
	}

	t.Run("nil pointer", func(t *testing.T) {
		assert.False(t, EvaluatorAccessors.TargetFindKey(nil, ""))
	})
}
