package ldmodel

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/launchdarkly/go-jsonstream/v3/jreader"
	"github.com/launchdarkly/go-jsonstream/v3/jwriter"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	"github.com/launchdarkly/go-test-helpers/v3/jsonhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testMarshalFlagFn func(FeatureFlag) ([]byte, error)
type testUnmarshalFlagFn func([]byte) (FeatureFlag, error)

type testMarshalSegmentFn func(Segment) ([]byte, error)
type testUnmarshalSegmentFn func([]byte) (Segment, error)

func doMarshalFlagTest(t *testing.T, marshalFn testMarshalFlagFn) {
	for _, p := range makeFlagSerializationTestParams() {
		t.Run(p.name, func(t *testing.T) {
			bytes, err := marshalFn(p.flag)
			require.NoError(t, err)
			expected := mergeDefaultProperties(json.RawMessage(p.jsonString), flagTopLevelDefaultProperties)
			jsonhelpers.AssertEqual(t, expected, bytes)
		})
	}
}

func doMarshalSegmentTest(t *testing.T, marshalFn testMarshalSegmentFn) {
	for _, p := range makeSegmentSerializationTestParams() {
		t.Run(p.name, func(t *testing.T) {
			bytes, err := marshalFn(p.segment)
			require.NoError(t, err)
			expected := mergeDefaultProperties(json.RawMessage(p.jsonString), segmentTopLevelDefaultProperties)
			jsonhelpers.AssertEqual(t, expected, bytes)
		})
	}
}

func doUnmarshalFlagTest(t *testing.T, unmarshalFn testUnmarshalFlagFn) {
	for _, p := range makeFlagSerializationTestParams() {
		t.Run(p.name, func(t *testing.T) {
			flag, err := unmarshalFn([]byte(p.jsonString))
			require.NoError(t, err)

			expectedFlag := p.flag
			PreprocessFlag(&expectedFlag)
			if !p.isCustomClientSideAvailability {
				expectedFlag.ClientSideAvailability = ClientSideAvailability{UsingMobileKey: true} // this is the default
			}
			assert.Equal(t, expectedFlag, flag)

			for _, altJSON := range p.jsonAltInputs {
				t.Run(altJSON, func(t *testing.T) {
					flag, err := unmarshalFn([]byte(altJSON))
					require.NoError(t, err)
					assert.Equal(t, expectedFlag, flag)
				})
			}
		})
	}
}

func doUnmarshalSegmentTest(t *testing.T, unmarshalFn testUnmarshalSegmentFn) {
	for _, p := range makeSegmentSerializationTestParams() {
		t.Run(p.name, func(t *testing.T) {
			segment, err := unmarshalFn([]byte(p.jsonString))
			require.NoError(t, err)

			expectedSegment := p.segment
			PreprocessSegment(&expectedSegment)

			assert.Equal(t, expectedSegment, segment)

			for _, altJSON := range p.jsonAltInputs {
				t.Run(altJSON, func(t *testing.T) {
					segment, err := unmarshalFn([]byte(altJSON))
					require.NoError(t, err)
					assert.Equal(t, expectedSegment, segment)
				})
			}
		})
	}
}

func TestMarshalFlagWithJSONMarshal(t *testing.T) {
	doMarshalFlagTest(t, func(flag FeatureFlag) ([]byte, error) {
		return json.Marshal(flag)
	})
}

func TestMarshalFlagWithDefaultSerialization(t *testing.T) {
	doMarshalFlagTest(t, NewJSONDataModelSerialization().MarshalFeatureFlag)
}

func TestMarshalFlagWithJSONWriter(t *testing.T) {
	doMarshalFlagTest(t, func(flag FeatureFlag) ([]byte, error) {
		w := jwriter.NewWriter()
		MarshalFeatureFlagToJSONWriter(flag, &w)
		return w.Bytes(), w.Error()
	})
}

func TestUnmarshalFlagWithJSONUnmarshal(t *testing.T) {
	doUnmarshalFlagTest(t, func(data []byte) (FeatureFlag, error) {
		var flag FeatureFlag
		err := json.Unmarshal(data, &flag)
		return flag, err
	})
}

func TestUnmarshalFlagWithDefaultSerialization(t *testing.T) {
	doUnmarshalFlagTest(t, NewJSONDataModelSerialization().UnmarshalFeatureFlag)
}

func TestUnmarshalFlagWithJSONReader(t *testing.T) {
	doUnmarshalFlagTest(t, func(data []byte) (FeatureFlag, error) {
		r := jreader.NewReader(data)
		flag := UnmarshalFeatureFlagFromJSONReader(&r)
		return flag, r.Error()
	})
}

func TestMarshalSegmentWithJSONMarshal(t *testing.T) {
	doMarshalSegmentTest(t, func(segment Segment) ([]byte, error) {
		return json.Marshal(segment)
	})
}

func TestMarshalSegmentWithDefaultSerialization(t *testing.T) {
	doMarshalSegmentTest(t, NewJSONDataModelSerialization().MarshalSegment)
}

func TestMarshalSegmentWithJSONWriter(t *testing.T) {
	doMarshalSegmentTest(t, func(segment Segment) ([]byte, error) {
		w := jwriter.NewWriter()
		MarshalSegmentToJSONWriter(segment, &w)
		return w.Bytes(), w.Error()
	})
}

func TestUnmarshalSegmentWithJSONUnmarshal(t *testing.T) {
	doUnmarshalSegmentTest(t, func(data []byte) (Segment, error) {
		var segment Segment
		err := json.Unmarshal(data, &segment)
		return segment, err
	})
}

func TestUnmarshalSegmentWithDefaultSerialization(t *testing.T) {
	doUnmarshalSegmentTest(t, NewJSONDataModelSerialization().UnmarshalSegment)
}

func TestUnmarshalSegmentWithJSONReader(t *testing.T) {
	doUnmarshalSegmentTest(t, func(data []byte) (Segment, error) {
		r := jreader.NewReader(data)
		segment := UnmarshalSegmentFromJSONReader(&r)
		return segment, r.Error()
	})
}

func TestUnmarshalFlagErrors(t *testing.T) {
	_, err := NewJSONDataModelSerialization().UnmarshalFeatureFlag([]byte(`{`))
	assert.Error(t, err)

	_, err = NewJSONDataModelSerialization().UnmarshalFeatureFlag([]byte(`{"key":[]}`))
	assert.Error(t, err)
}

func TestUnmarshalSegmentErrors(t *testing.T) {
	_, err := NewJSONDataModelSerialization().UnmarshalSegment([]byte(`{`))
	assert.Error(t, err)

	_, err = NewJSONDataModelSerialization().UnmarshalSegment([]byte(`{"key":[]}`))
	assert.Error(t, err)
}

// The clause has enough values for a set lookup, and one duplicate value.
const clauseInValuesRulesJSON = `[{"id": "r", "clauses": [{"attribute": "key", "op": "in", "negate": false,
	"values": ["a", "b", "c", "d", "e", "f", 1, true, "b"]}]}]`

var clauseInValuesExpected = []ldvalue.Value{ //nolint:gochecknoglobals
	ldvalue.String("a"), ldvalue.String("b"), ldvalue.String("c"), ldvalue.String("d"),
	ldvalue.String("e"), ldvalue.String("f"), ldvalue.Int(1), ldvalue.Bool(true), ldvalue.String("b"),
}

func parseMarshaledClauseValues(t *testing.T, data []byte) []ldvalue.Value {
	var parsed struct {
		Rules []struct {
			Clauses []struct {
				Values []ldvalue.Value `json:"values"`
			} `json:"clauses"`
		} `json:"rules"`
	}
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Len(t, parsed.Rules, 1)
	require.Len(t, parsed.Rules[0].Clauses, 1)
	return parsed.Rules[0].Clauses[0].Values
}

func TestFlagRoundTripWithClauseValueSet(t *testing.T) {
	serialization := NewJSONDataModelSerialization()
	flag, err := serialization.UnmarshalFeatureFlag(
		[]byte(`{"key": "f", "version": 1, "rules": ` + clauseInValuesRulesJSON + `}`))
	require.NoError(t, err)

	t.Run("not released, with original order and duplicates", func(t *testing.T) {
		data, err := serialization.MarshalFeatureFlag(flag)
		require.NoError(t, err)
		assert.Equal(t, clauseInValuesExpected, parseMarshaledClauseValues(t, data))
	})

	t.Run("released", func(t *testing.T) {
		released := flag
		released.Rules = []FlagRule{flag.Rules[0]}
		released.Rules[0].Clauses = slices.Clone(flag.Rules[0].Clauses)
		ReleaseClauseValues(&released)
		require.Nil(t, released.Rules[0].Clauses[0].Values)

		data, err := serialization.MarshalFeatureFlag(released)
		require.NoError(t, err)
		assert.ElementsMatch(t, clauseInValuesExpected[:len(clauseInValuesExpected)-1],
			parseMarshaledClauseValues(t, data))

		flag2, err := serialization.UnmarshalFeatureFlag(data)
		require.NoError(t, err)
		ReleaseClauseValues(&flag2)
		assert.Equal(t, released, flag2)
	})
}

func TestSegmentRoundTripWithReleasedClauseValueSet(t *testing.T) {
	serialization := NewJSONDataModelSerialization()
	segment, err := serialization.UnmarshalSegment(
		[]byte(`{"key": "s", "version": 1, "rules": ` + clauseInValuesRulesJSON + `}`))
	require.NoError(t, err)
	ReleaseSegmentClauseValues(&segment)
	require.Nil(t, segment.Rules[0].Clauses[0].Values)

	data, err := serialization.MarshalSegment(segment)
	require.NoError(t, err)
	assert.ElementsMatch(t, clauseInValuesExpected[:len(clauseInValuesExpected)-1],
		parseMarshaledClauseValues(t, data))

	segment2, err := serialization.UnmarshalSegment(data)
	require.NoError(t, err)
	ReleaseSegmentClauseValues(&segment2)
	assert.Equal(t, segment, segment2)
}
