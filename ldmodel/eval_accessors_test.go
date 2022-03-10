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

	t.Run("nil pointer", func(t *testing.T) {
		assert.False(t, EvaluatorAccessors.ClauseFindValue(nil, ldvalue.String("")))
	})
}

func TestSegmentFindValueInExcludes(t *testing.T) {

}

func TestSegmentFindValueInIncludes(t *testing.T) {

}

func TestTargetFindKey(t *testing.T) {
	foundValues := []string{"a", "b", "c"}
	notFoundValues := []string{"d", "e", "f"}

	for _, withPreprocessing := range []bool{false, true} {
		t.Run(fmt.Sprintf("preprocessed: %t", withPreprocessing), func(t *testing.T) {
			target := Target{Values: foundValues}
			if withPreprocessing {
				target.preprocessed = preprocessTarget(target)
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
