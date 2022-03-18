package ldmodel

import (
	"testing"
	"time"

	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	"github.com/launchdarkly/go-semver"

	"github.com/stretchr/testify/assert"
)

func TestValueToSemanticVersion(t *testing.T) {
	t.Run("valid values", func(t *testing.T) {
		valueGroups := [][]string{
			{"1.2.3"},
			{"1.2", "1.2.0"},
			{"1", "1.0.0"},
			{"1.0.0-beta.1"},
			{"1.0.0+build1"},
			{"1.0.0-beta.1+build1"},
		}
		for _, group := range valueGroups {
			var firstResult semver.Version
			for i, input := range group {
				t.Run(input, func(t *testing.T) {
					result, ok := TypeConversions.ValueToSemanticVersion(ldvalue.String(input))
					assert.True(t, ok)
					if i == 0 {
						firstResult = result
					} else {
						assert.Equal(t, firstResult, result)
					}
				})
			}
		}
	})

	t.Run("invalid values", func(t *testing.T) {
		for _, value := range []ldvalue.Value{
			ldvalue.Null(),
			ldvalue.Bool(false),
			ldvalue.Int(2),
			ldvalue.String("hey what's this?"),
			ldvalue.ArrayOf(),
			ldvalue.ObjectBuild().Build(),
		} {
			t.Run(value.JSONString(), func(t *testing.T) {
				_, ok := TypeConversions.ValueToSemanticVersion(value)
				assert.False(t, ok)
			})
		}
	})
}

func TestValueToTimestamp(t *testing.T) {
	t.Run("valid values", func(t *testing.T) {
		valueGroups := [][]ldvalue.Value{
			{ldvalue.String("1970-01-01T00:00:00Z"), ldvalue.Int(0)},
			{ldvalue.String("2016-04-16T22:57:31.684Z"), ldvalue.Int(1460847451684)},
			{ldvalue.String("2016-04-16T17:09:12.759-07:00"), ldvalue.Int(1460851752759)},
			{ldvalue.String("2016-04-16T17:09:12-07:00"), ldvalue.Int(1460851752000)},
			{ldvalue.String("1969-12-31T23:57:56.544-00:00"), ldvalue.Int(-123456)},
		}
		for _, group := range valueGroups {
			var firstResult time.Time
			for i, input := range group {
				t.Run(input.StringValue(), func(t *testing.T) {
					result, ok := TypeConversions.ValueToTimestamp(input)
					assert.True(t, ok)
					if i == 0 {
						firstResult = result
					} else {
						assert.Equal(t, firstResult, result)
					}
				})
			}
		}
	})

	t.Run("invalid values", func(t *testing.T) {
		for _, value := range []ldvalue.Value{
			ldvalue.Null(),
			ldvalue.Bool(false),
			ldvalue.String("hey what's this?"),
			ldvalue.ArrayOf(),
			ldvalue.ObjectBuild().Build(),
		} {
			t.Run(value.JSONString(), func(t *testing.T) {
				_, ok := TypeConversions.ValueToTimestamp(value)
				assert.False(t, ok)
			})
		}
	})
}
