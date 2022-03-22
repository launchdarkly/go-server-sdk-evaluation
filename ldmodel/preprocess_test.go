package ldmodel

import (
	"regexp"
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

func TestPreprocessFlagCreatesClauseValuesMapForMultiValueEqualityTest(t *testing.T) {
	f := FeatureFlag{
		Rules: []FlagRule{
			{
				Clauses: []Clause{
					{
						Op:     OperatorIn,
						Values: []ldvalue.Value{ldvalue.Bool(true), ldvalue.String("a"), ldvalue.Int(0)},
					},
				},
			},
		},
	}

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.valuesMap)

	PreprocessFlag(&f)

	m := f.Rules[0].Clauses[0].preprocessed.valuesMap
	assert.Equal(t, map[jsonPrimitiveValueKey]struct{}{
		asPrimitiveValueKey(ldvalue.Bool(true)):  {},
		asPrimitiveValueKey(ldvalue.String("a")): {},
		asPrimitiveValueKey(ldvalue.Int(0)):      {},
	}, m)
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

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.valuesMap)

	PreprocessFlag(&f)

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.valuesMap)
}

func TestPreprocessFlagDoesNotCreateClauseValuesMapForEmptyEqualityTest(t *testing.T) {
	f := FeatureFlag{
		Rules: []FlagRule{
			{Clauses: []Clause{{Op: OperatorIn, Values: []ldvalue.Value{}}}},
		},
	}

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.valuesMap)

	PreprocessFlag(&f)

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.valuesMap)
}

func TestPreprocessFlagDoesNotCreateClauseValuesMapForNonEqualityOperators(t *testing.T) {
	ops := []Operator{
		OperatorEndsWith, OperatorStartsWith, OperatorMatches, OperatorContains, OperatorLessThan,
		OperatorLessThanOrEqual, OperatorGreaterThan, OperatorGreaterThanOrEqual, OperatorBefore,
		OperatorAfter, OperatorSegmentMatch, OperatorSemVerEqual, OperatorSemVerLessThan,
		OperatorSemVerGreaterThan,
	}

	values := []ldvalue.Value{ldvalue.String("a"), ldvalue.String("b")}
	// The values & types aren't very important here because we won't actually evaluate the clause; all that
	// matters is that they're primitives and there's more than one of them, so that it *would* build a map
	// if the operator were OperatorIn
	for _, op := range ops {
		t.Run(string(op), func(t *testing.T) {
			f := FeatureFlag{
				Rules: []FlagRule{
					{
						Clauses: []Clause{{Op: op, Values: values}},
					},
				},
			}

			assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.valuesMap)

			PreprocessFlag(&f)

			assert.Nil(t, f.Rules[0].Clauses[0].preprocessed.valuesMap)
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
