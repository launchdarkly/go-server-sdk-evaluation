package ldmodel

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
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

	assert.Nil(t, f.Targets[0].valuesMap)
	assert.Nil(t, f.Targets[1].valuesMap)

	PreprocessFlag(&f)

	assert.Nil(t, f.Targets[0].valuesMap)

	assert.Len(t, f.Targets[1].valuesMap, 2)
	assert.True(t, f.Targets[1].valuesMap["a"])
	assert.True(t, f.Targets[1].valuesMap["b"])
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

	assert.Nil(t, f.Rules[0].Clauses[0].preprocessedValues)

	PreprocessFlag(&f)

	p := f.Rules[0].Clauses[0].preprocessedValues
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

			assert.Nil(t, f.Rules[0].Clauses[0].preprocessedValues)

			PreprocessFlag(&f)

			p := f.Rules[0].Clauses[0].preprocessedValues
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

			assert.Nil(t, f.Rules[0].Clauses[0].preprocessedValues)

			PreprocessFlag(&f)

			p := f.Rules[0].Clauses[0].preprocessedValues
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
