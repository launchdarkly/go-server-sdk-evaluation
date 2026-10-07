package ldmodel

import (
	"fmt"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
)

func TestPreprocessFlagBuildsTargetMap(t *testing.T) {
	f := FeatureFlag{
		Targets: []Target{
			{
				Variation: 0,
				Values:    nil,
			},
			{
				Variation: 1,
				Values:    []string{"a", "b"},
			},
		},
	}

	assert.Nil(t, f.Targets[0].preprocessed.valuesMap)
	assert.Nil(t, f.Targets[1].preprocessed.valuesMap)

	PreprocessFlag(&f)

	assert.Nil(t, f.Targets[0].preprocessed.valuesMap)

	assert.Len(t, f.Targets[1].preprocessed.valuesMap, 2)
	assert.Contains(t, f.Targets[1].preprocessed.valuesMap, "a")
	assert.Contains(t, f.Targets[1].preprocessed.valuesMap, "b")
}

func makeClauseValuesAtSetThreshold() []ldvalue.Value {
	values := []ldvalue.Value{ldvalue.Bool(true), ldvalue.Int(0)}
	for len(values) < clauseInValuesSetMinSize {
		values = append(values, ldvalue.String(fmt.Sprintf("value%d", len(values))))
	}
	return values
}

func makeFlagWithInClause(values []ldvalue.Value) FeatureFlag {
	return FeatureFlag{
		Rules: []FlagRule{
			{Clauses: []Clause{{Op: OperatorIn, Values: values}}},
		},
	}
}

func TestPreprocessFlagCreatesClauseValueSetAtThreshold(t *testing.T) {
	values := makeClauseValuesAtSetThreshold()
	f := makeFlagWithInClause(values)

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.inValueSet)

	PreprocessFlag(&f)

	expected := make(map[jsonPrimitiveValueKey]int32)
	for i, v := range values {
		expected[asPrimitiveValueKey(v)] = int32(i)
	}
	assert.Equal(t, expected, f.Rules[0].Clauses[0].preprocessed.inValueSet)
	assert.Equal(t, values, f.Rules[0].Clauses[0].Values)
}

func TestPreprocessFlagDoesNotCreateClauseValueSetBelowThreshold(t *testing.T) {
	values := makeClauseValuesAtSetThreshold()[:clauseInValuesSetMinSize-1]
	f := makeFlagWithInClause(values)

	PreprocessFlag(&f)

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.inValueSet)
	assert.Equal(t, values, f.Rules[0].Clauses[0].Values)
}

func TestPreprocessFlagDoesNotCreateClauseValueSetForNonPrimitiveValue(t *testing.T) {
	values := append(makeClauseValuesAtSetThreshold(), ldvalue.ArrayOf(ldvalue.String("a")))
	f := makeFlagWithInClause(values)

	PreprocessFlag(&f)

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.inValueSet)
	assert.Equal(t, values, f.Rules[0].Clauses[0].Values)
}

func TestReleaseClauseValuesReleasesListOfClauseWithSet(t *testing.T) {
	values := makeClauseValuesAtSetThreshold()
	f := makeFlagWithInClause(values)
	PreprocessFlag(&f)

	ReleaseClauseValues(&f)

	c := &f.Rules[0].Clauses[0]
	assert.Nil(t, c.Values)
	assert.Len(t, c.preprocessed.inValueSet, len(values))
	for _, v := range values {
		assert.True(t, EvaluatorAccessors.ClauseFindValue(c, v), "value: %s", v)
	}
	assert.False(t, EvaluatorAccessors.ClauseFindValue(c, ldvalue.String("other")))
	assert.ElementsMatch(t, values, slices.Collect(c.AllValues()))
}

func TestReleaseClauseValuesKeepsListOfClauseWithoutSet(t *testing.T) {
	values := makeClauseValuesAtSetThreshold()[:clauseInValuesSetMinSize-1]
	f := makeFlagWithInClause(values)
	PreprocessFlag(&f)

	ReleaseClauseValues(&f)

	assert.Equal(t, values, f.Rules[0].Clauses[0].Values)
	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.inValueSet)
}

func TestPreprocessFlagKeepsReleasedClauseValueSet(t *testing.T) {
	f := makeFlagWithInClause(makeClauseValuesAtSetThreshold())
	PreprocessFlag(&f)
	ReleaseClauseValues(&f)
	expected := f.Rules[0].Clauses[0].preprocessed.inValueSet

	PreprocessFlag(&f)

	assert.Equal(t, expected, f.Rules[0].Clauses[0].preprocessed.inValueSet)
	assert.Nil(t, f.Rules[0].Clauses[0].Values)
}

func TestClauseAllValues(t *testing.T) {
	values := append(makeClauseValuesAtSetThreshold(), ldvalue.String("value2"))

	t.Run("not preprocessed", func(t *testing.T) {
		c := Clause{Op: OperatorIn, Values: values}
		assert.Equal(t, values, slices.Collect(c.AllValues()))
	})

	t.Run("preprocessed, with original order and duplicates", func(t *testing.T) {
		f := makeFlagWithInClause(values)
		PreprocessFlag(&f)
		assert.Equal(t, values, slices.Collect(f.Rules[0].Clauses[0].AllValues()))
	})

	t.Run("released, without duplicates", func(t *testing.T) {
		f := makeFlagWithInClause(values)
		PreprocessFlag(&f)
		ReleaseClauseValues(&f)
		assert.ElementsMatch(t, values[:len(values)-1], slices.Collect(f.Rules[0].Clauses[0].AllValues()))
	})

	t.Run("operator other than in", func(t *testing.T) {
		f := FeatureFlag{
			Rules: []FlagRule{{Clauses: []Clause{{Op: OperatorMatches, Values: values}}}},
		}
		PreprocessFlag(&f)
		assert.Equal(t, values, slices.Collect(f.Rules[0].Clauses[0].AllValues()))
	})
}

func TestPreprocessFlagDoesNotCreateClauseValuesMapForSingleValueEqualityTest(t *testing.T) {
	f := FeatureFlag{
		Rules: []FlagRule{
			{
				Clauses: []Clause{
					{
						Op:     OperatorIn,
						Values: []ldvalue.Value{ldvalue.String("a")},
					},
				},
			},
		},
	}

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.inValueSet)

	PreprocessFlag(&f)

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.inValueSet)
}

func TestPreprocessFlagDoesNotCreateClauseValuesMapForEmptyEqualityTest(t *testing.T) {
	f := FeatureFlag{
		Rules: []FlagRule{
			{Clauses: []Clause{{Op: OperatorIn, Values: []ldvalue.Value{}}}},
		},
	}

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.inValueSet)

	PreprocessFlag(&f)

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.inValueSet)
}

var nonEqualityOperators = []Operator{ //nolint:gochecknoglobals
	OperatorEndsWith, OperatorStartsWith, OperatorMatches, OperatorContains, OperatorLessThan,
	OperatorLessThanOrEqual, OperatorGreaterThan, OperatorGreaterThanOrEqual, OperatorBefore,
	OperatorAfter, OperatorSegmentMatch, OperatorSemVerEqual, OperatorSemVerLessThan,
	OperatorSemVerGreaterThan, Operator("unknownOperator"),
}

func TestPreprocessFlagDoesNotCreateClauseValuesMapForNonEqualityOperators(t *testing.T) {
	values := makeClauseValuesAtSetThreshold()
	// The values & types aren't very important here because we won't actually evaluate the clause; all that
	// matters is that they're primitives and there are enough of them, so that it *would* build a map
	// if the operator were OperatorIn
	for _, op := range nonEqualityOperators {
		t.Run(string(op), func(t *testing.T) {
			f := FeatureFlag{
				Rules: []FlagRule{
					{
						Clauses: []Clause{{Op: op, Values: values}},
					},
				},
			}

			PreprocessFlag(&f)

			assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.inValueSet)
			assert.Equal(t, values, f.Rules[0].Clauses[0].Values)
		})
	}
}

func TestReleaseClauseValuesKeepsValuesForNonEqualityOperators(t *testing.T) {
	// Evaluation reads Values directly for every operator other than OperatorIn, so the release
	// must not change Values for these operators, even if there are enough values for a set.
	values := makeClauseValuesAtSetThreshold()
	for _, op := range nonEqualityOperators {
		t.Run(string(op), func(t *testing.T) {
			f := FeatureFlag{
				Rules: []FlagRule{{Clauses: []Clause{{Op: op, Values: values}}}},
			}
			PreprocessFlag(&f)
			ReleaseClauseValues(&f)
			assert.Equal(t, values, f.Rules[0].Clauses[0].Values)

			s := Segment{
				Rules: []SegmentRule{{Clauses: []Clause{{Op: op, Values: values}}}},
			}
			PreprocessSegment(&s)
			ReleaseSegmentClauseValues(&s)
			assert.Equal(t, values, s.Rules[0].Clauses[0].Values)
		})
	}
}

func TestPreprocessFlagParsesClauseRegex(t *testing.T) {
	f := FeatureFlag{
		Rules: []FlagRule{
			{
				Clauses: []Clause{
					{
						Op:     OperatorMatches,
						Values: []ldvalue.Value{ldvalue.String("x*"), ldvalue.String("\\"), ldvalue.Int(3)},
					},
				},
			},
		},
	}

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.values)

	PreprocessFlag(&f)

	p := f.Rules[0].Clauses[0].preprocessed.values
	require.Len(t, p, 3)

	assert.True(t, p[0].computed)
	assert.True(t, p[0].valid)
	assert.Equal(t, regexp.MustCompile("x*"), p[0].parsedRegexp)

	assert.True(t, p[1].computed)
	assert.False(t, p[1].valid)
	assert.True(t, p[2].computed)
	assert.False(t, p[2].valid)
}

func TestPreprocessFlagParsesClauseTime(t *testing.T) {
	time1Str := "2016-04-16T17:09:12-07:00"
	t1, _ := time.Parse(time.RFC3339Nano, time1Str)
	time1 := t1.UTC()
	time2Num := float64(1000000)
	time2 := time.Unix(0, int64(time2Num)*int64(time.Millisecond)).UTC()

	for _, operator := range []Operator{OperatorAfter, OperatorBefore} {
		t.Run(string(operator), func(t *testing.T) {
			f := FeatureFlag{
				Rules: []FlagRule{
					{
						Clauses: []Clause{
							{
								Op:     operator,
								Values: []ldvalue.Value{ldvalue.String(time1Str), ldvalue.Float64(time2Num), ldvalue.String("x"), ldvalue.Bool(false)},
							},
						},
					},
				},
			}

			assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.values)

			PreprocessFlag(&f)

			p := f.Rules[0].Clauses[0].preprocessed.values
			require.Len(t, p, 4)

			assert.True(t, p[0].computed)
			assert.True(t, p[0].valid)
			assert.Equal(t, time1, p[0].parsedTime)

			assert.True(t, p[1].computed)
			assert.True(t, p[1].valid)
			assert.Equal(t, time2, p[1].parsedTime)

			assert.True(t, p[2].computed)
			assert.False(t, p[2].valid)
			assert.True(t, p[3].computed)
			assert.False(t, p[3].valid)
		})
	}
}

func TestPreprocessFlagParsesClauseSemver(t *testing.T) {
	expected, ok := parseSemVer(ldvalue.String("1.2.3"))
	require.True(t, ok)

	for _, operator := range []Operator{OperatorSemVerEqual, OperatorSemVerGreaterThan, OperatorSemVerLessThan} {
		t.Run(string(operator), func(t *testing.T) {
			f := FeatureFlag{
				Rules: []FlagRule{
					{
						Clauses: []Clause{
							{
								Op:     operator,
								Values: []ldvalue.Value{ldvalue.String("1.2.3"), ldvalue.String("x"), ldvalue.Bool(false)},
							},
						},
					},
				},
			}

			assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.values)

			PreprocessFlag(&f)

			p := f.Rules[0].Clauses[0].preprocessed.values
			require.Len(t, p, 3)

			assert.True(t, p[0].computed)
			assert.True(t, p[0].valid)
			assert.Equal(t, expected, p[0].parsedSemver)

			assert.True(t, p[1].computed)
			assert.False(t, p[1].valid)
			assert.True(t, p[2].computed)
			assert.False(t, p[2].valid)
		})
	}
}

func TestPreprocessSegmentBuildsIncludeAndExcludeMaps(t *testing.T) {
	s := Segment{
		Included: []string{"a", "b"},
		Excluded: []string{"c"},
		IncludedContexts: []SegmentTarget{
			{ContextKind: ldcontext.Kind("org"), Values: []string{"x", "y"}},
		},
		ExcludedContexts: []SegmentTarget{
			{ContextKind: ldcontext.Kind("org"), Values: []string{"z"}},
		},
	}

	assert.Nil(t, s.preprocessed.includeMap)
	assert.Nil(t, s.preprocessed.excludeMap)

	PreprocessSegment(&s)

	assert.Equal(t, map[string]struct{}{"a": {}, "b": {}}, s.preprocessed.includeMap)
	assert.Equal(t, map[string]struct{}{"c": {}}, s.preprocessed.excludeMap)
	assert.Equal(t, map[string]struct{}{"x": {}, "y": {}}, s.IncludedContexts[0].preprocessed.valuesMap)
	assert.Equal(t, map[string]struct{}{"z": {}}, s.ExcludedContexts[0].preprocessed.valuesMap)
}

func TestPreprocessSegmentPreprocessesClausesInRules(t *testing.T) {
	// We'll just check one kind of clause, and assume that the preprocessing works the same as in flag rules
	s := Segment{
		Rules: []SegmentRule{
			{
				Clauses: []Clause{
					{
						Op:     OperatorMatches,
						Values: []ldvalue.Value{ldvalue.String("x*"), ldvalue.String("\\"), ldvalue.Int(3)},
					},
				},
			},
		},
	}

	assert.Nil(t, s.Rules[0].Clauses[0].preprocessed.values)

	PreprocessSegment(&s)

	p := s.Rules[0].Clauses[0].preprocessed.values
	require.Len(t, p, 3)

	assert.True(t, p[0].computed)
	assert.True(t, p[0].valid)
	assert.Equal(t, regexp.MustCompile("x*"), p[0].parsedRegexp)

	assert.True(t, p[1].computed)
	assert.False(t, p[1].valid)
	assert.True(t, p[2].computed)
	assert.False(t, p[2].valid)
}

func TestReleaseSegmentClauseValuesReleasesListOfClauseWithSet(t *testing.T) {
	values := makeClauseValuesAtSetThreshold()
	s := Segment{
		Rules: []SegmentRule{
			{Clauses: []Clause{{Op: OperatorIn, Values: values}}},
		},
	}
	PreprocessSegment(&s)
	assert.Len(t, s.Rules[0].Clauses[0].preprocessed.inValueSet, len(values))
	assert.Equal(t, values, s.Rules[0].Clauses[0].Values)

	ReleaseSegmentClauseValues(&s)

	assert.Nil(t, s.Rules[0].Clauses[0].Values)
	assert.ElementsMatch(t, values, slices.Collect(s.Rules[0].Clauses[0].AllValues()))
}
