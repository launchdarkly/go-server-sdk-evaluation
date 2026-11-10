package ldmodel

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/launchdarkly/go-jsonstream/v4/jwriter"
	"github.com/launchdarkly/go-sdk-common/v4/ldattr"
	"github.com/launchdarkly/go-sdk-common/v4/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v4/ldvalue"
	"github.com/launchdarkly/go-test-helpers/v3/jsonhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// compactFlagMarshalers are all of the ways to get the compact encoding of a flag.
var compactFlagMarshalers = map[string]testMarshalFlagFn{
	"DataModelSerialization": NewCompactJSONDataModelSerialization().MarshalFeatureFlag,
	"JSONWriter": func(flag FeatureFlag) ([]byte, error) {
		w := jwriter.NewWriter()
		MarshalFeatureFlagToJSONWriterCompact(flag, &w)
		return w.Bytes(), w.Error()
	},
}

// compactSegmentMarshalers are all of the ways to get the compact encoding of a segment.
var compactSegmentMarshalers = map[string]testMarshalSegmentFn{
	"DataModelSerialization": NewCompactJSONDataModelSerialization().MarshalSegment,
	"JSONWriter": func(segment Segment) ([]byte, error) {
		w := jwriter.NewWriter()
		MarshalSegmentToJSONWriterCompact(segment, &w)
		return w.Bytes(), w.Error()
	},
}

// compactPropertiesWithDefaultValues are the properties that the compact encoding writes even if
// the value is false, zero, or empty, because an absent property has a different meaning. Array
// elements are shown as "[]".
var compactPropertiesWithDefaultValues = []string{
	"offVariation",
	"fallthrough.variation",
	"fallthrough.rollout.seed",
	"rules[].variation",
	"rules[].rollout.seed",
	"rules[].weight", // segment rules
	"clientSideAvailability.usingMobileKey",
	"clientSideAvailability.usingEnvironmentId",
	"migration",
	"migration.checkRatio",
	"samplingRatio",
}

// findDefaultValuedProperties returns the path of each property in data that has a false, zero,
// empty, or null value, other than the properties in compactPropertiesWithDefaultValues.
func findDefaultValuedProperties(t *testing.T, data []byte) []string {
	var parsed any
	require.NoError(t, json.Unmarshal(data, &parsed))
	var found []string
	var walk func(path string, value any)
	walk = func(path string, value any) {
		switch v := value.(type) {
		case map[string]any:
			for name, propValue := range v {
				propPath := name
				if path != "" {
					propPath = path + "." + name
				}
				if isDefaultJSONValue(propValue) && !slices.Contains(compactPropertiesWithDefaultValues, propPath) {
					found = append(found, propPath)
				}
				walk(propPath, propValue)
			}
		case []any:
			for _, item := range v {
				walk(path+"[]", item)
			}
		}
	}
	walk("", parsed)
	return found
}

func isDefaultJSONValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case bool:
		return !v
	case float64:
		return v == 0
	case string:
		return v == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	}
	return false
}

func TestCompactFlagEncodingDecodesToSameFlagAsFullEncoding(t *testing.T) {
	for marshalerName, marshalFn := range compactFlagMarshalers {
		t.Run(marshalerName, func(t *testing.T) {
			for _, p := range makeFlagSerializationTestParams() {
				t.Run(p.name, func(t *testing.T) {
					fullBytes, err := marshalFeatureFlag(p.flag)
					require.NoError(t, err)
					compactBytes, err := marshalFn(p.flag)
					require.NoError(t, err)

					fromFull, err := unmarshalFeatureFlagFromBytes(fullBytes)
					require.NoError(t, err)
					fromCompact, err := unmarshalFeatureFlagFromBytes(compactBytes)
					require.NoError(t, err)
					assert.Equal(t, fromFull, fromCompact, "compact JSON: %s", compactBytes)
				})
			}
		})
	}
}

func TestCompactSegmentEncodingDecodesToSameSegmentAsFullEncoding(t *testing.T) {
	for marshalerName, marshalFn := range compactSegmentMarshalers {
		t.Run(marshalerName, func(t *testing.T) {
			for _, p := range makeSegmentSerializationTestParams() {
				t.Run(p.name, func(t *testing.T) {
					fullBytes, err := marshalSegment(p.segment)
					require.NoError(t, err)
					compactBytes, err := marshalFn(p.segment)
					require.NoError(t, err)

					fromFull, err := unmarshalSegmentFromBytes(fullBytes)
					require.NoError(t, err)
					fromCompact, err := unmarshalSegmentFromBytes(compactBytes)
					require.NoError(t, err)
					assert.Equal(t, fromFull, fromCompact, "compact JSON: %s", compactBytes)
				})
			}
		})
	}
}

func TestCompactFlagEncodingOmitsDefaultValues(t *testing.T) {
	flags := []FeatureFlag{makeFlagWithDefaultValuesEverywhere()}
	for _, p := range makeFlagSerializationTestParams() {
		flags = append(flags, p.flag)
	}
	for i, flag := range flags {
		t.Run(fmt.Sprintf("flag %d", i), func(t *testing.T) {
			data, err := marshalFeatureFlagCompact(flag)
			require.NoError(t, err)
			assert.Empty(t, findDefaultValuedProperties(t, data), "compact JSON: %s", data)
		})
	}
}

func TestCompactSegmentEncodingOmitsDefaultValues(t *testing.T) {
	segments := []Segment{makeSegmentWithDefaultValuesEverywhere()}
	for _, p := range makeSegmentSerializationTestParams() {
		segments = append(segments, p.segment)
	}
	for i, segment := range segments {
		t.Run(fmt.Sprintf("segment %d", i), func(t *testing.T) {
			data, err := marshalSegmentCompact(segment)
			require.NoError(t, err)
			assert.Empty(t, findDefaultValuedProperties(t, data), "compact JSON: %s", data)
		})
	}
}

// makeFlagWithDefaultValuesEverywhere returns a flag in which each nested object has properties
// with default values: variation indexes and weights of zero, false booleans, and empty lists.
func makeFlagWithDefaultValuesEverywhere() FeatureFlag {
	return FeatureFlag{
		Key:     "flag-key",
		Version: 1,
		On:      true,
		Prerequisites: []Prerequisite{
			{Key: "prereq-key", Variation: 0},
		},
		Targets: []Target{
			{Values: []string{"a"}, Variation: 0},
			{Values: nil, Variation: 1},
		},
		ContextTargets: []Target{
			{ContextKind: "org", Values: []string{"b"}, Variation: 0},
		},
		Rules: []FlagRule{
			{
				ID:                 "rule-id",
				VariationOrRollout: VariationOrRollout{Variation: ldvalue.NewOptionalInt(0)},
				Clauses: []Clause{
					{Attribute: ldattr.NewLiteralRef("name"), Op: OperatorIn, Values: []ldvalue.Value{ldvalue.String("x")}},
					{Op: OperatorSegmentMatch, Values: []ldvalue.Value{ldvalue.String("segment-key")}},
					{Attribute: ldattr.NewLiteralRef("email"), Op: OperatorIn},
				},
			},
			{
				VariationOrRollout: VariationOrRollout{
					Rollout: Rollout{
						Seed: ldvalue.NewOptionalInt(0),
						Variations: []WeightedVariation{
							{Variation: 0, Weight: 0},
							{Variation: 1, Weight: 100000},
						},
					},
				},
			},
		},
		Fallthrough:  VariationOrRollout{Variation: ldvalue.NewOptionalInt(0)},
		OffVariation: ldvalue.NewOptionalInt(0),
		Variations:   []ldvalue.Value{ldvalue.Bool(false), ldvalue.Bool(true)},
		Salt:         "salt",
	}
}

// makeSegmentWithDefaultValuesEverywhere returns a segment in which each nested object has
// properties with default values: weights of zero, false booleans, and empty lists.
func makeSegmentWithDefaultValuesEverywhere() Segment {
	return Segment{
		Key:              "segment-key",
		Version:          1,
		Included:         []string{"a"},
		IncludedContexts: []SegmentTarget{{ContextKind: "org", Values: nil}},
		Rules: []SegmentRule{
			{
				Clauses: []Clause{
					{Attribute: ldattr.NewLiteralRef("name"), Op: OperatorIn, Values: []ldvalue.Value{ldvalue.String("x")}},
				},
				Weight: ldvalue.NewOptionalInt(0),
			},
			{},
		},
		Salt: "salt",
	}
}

func TestCompactFlagEncodingOutput(t *testing.T) {
	data, err := marshalFeatureFlagCompact(makeFlagWithDefaultValuesEverywhere())
	require.NoError(t, err)
	jsonhelpers.AssertEqual(t, `{
		"key": "flag-key",
		"version": 1,
		"on": true,
		"prerequisites": [{"key": "prereq-key"}],
		"targets": [{"values": ["a"]}, {"variation": 1}],
		"contextTargets": [{"contextKind": "org", "values": ["b"]}],
		"rules": [
			{
				"id": "rule-id",
				"variation": 0,
				"clauses": [
					{"attribute": "name", "op": "in", "values": ["x"]},
					{"op": "segmentMatch", "values": ["segment-key"]},
					{"attribute": "email", "op": "in"}
				]
			},
			{
				"rollout": {
					"seed": 0,
					"variations": [{}, {"variation": 1, "weight": 100000}]
				}
			}
		],
		"fallthrough": {"variation": 0},
		"offVariation": 0,
		"variations": [false, true],
		"salt": "salt"
	}`, data)
}

func TestCompactSegmentEncodingOutput(t *testing.T) {
	data, err := marshalSegmentCompact(makeSegmentWithDefaultValuesEverywhere())
	require.NoError(t, err)
	jsonhelpers.AssertEqual(t, `{
		"key": "segment-key",
		"version": 1,
		"included": ["a"],
		"includedContexts": [{"contextKind": "org"}],
		"rules": [
			{"clauses": [{"attribute": "name", "op": "in", "values": ["x"]}], "weight": 0},
			{}
		],
		"salt": "salt"
	}`, data)
}

func TestCompactEncodingOfEmptyObjects(t *testing.T) {
	flagData, err := marshalFeatureFlagCompact(FeatureFlag{})
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(flagData))

	segmentData, err := marshalSegmentCompact(Segment{})
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(segmentData))
}

func TestCompactFlagEncodingOfClientSideAvailability(t *testing.T) {
	for _, p := range []struct {
		name     string
		csa      ClientSideAvailability
		expected string
	}{
		{"old schema, not client-side", ClientSideAvailability{UsingMobileKey: true}, `{}`},
		{"old schema, client-side", ClientSideAvailability{UsingMobileKey: true, UsingEnvironmentID: true},
			`{"clientSide": true}`},
		{"new schema, not available", ClientSideAvailability{Explicit: true},
			`{"clientSideAvailability": {"usingMobileKey": false, "usingEnvironmentId": false}}`},
		{"new schema, environment ID", ClientSideAvailability{Explicit: true, UsingEnvironmentID: true},
			`{"clientSideAvailability": {"usingMobileKey": false, "usingEnvironmentId": true}, "clientSide": true}`},
	} {
		t.Run(p.name, func(t *testing.T) {
			flag := FeatureFlag{ClientSideAvailability: p.csa}
			data, err := marshalFeatureFlagCompact(flag)
			require.NoError(t, err)
			assert.JSONEq(t, p.expected, string(data))

			decoded, err := unmarshalFeatureFlagFromBytes(data)
			require.NoError(t, err)
			assert.Equal(t, p.csa, decoded.ClientSideAvailability)
		})
	}
}

func TestCompactFlagEncodingOfReleasedClauseValues(t *testing.T) {
	flag, err := NewJSONDataModelSerialization().UnmarshalFeatureFlag(
		[]byte(`{"key": "f", "version": 1, "rules": ` + clauseInValuesRulesJSON + `}`))
	require.NoError(t, err)
	ReleaseClauseValues(&flag)
	require.Nil(t, flag.Rules[0].Clauses[0].Values)

	data, err := marshalFeatureFlagCompact(flag)
	require.NoError(t, err)
	assert.ElementsMatch(t, clauseInValuesExpected[:len(clauseInValuesExpected)-1],
		parseMarshaledClauseValues(t, data))

	decoded, err := unmarshalFeatureFlagFromBytes(data)
	require.NoError(t, err)
	ReleaseClauseValues(&decoded)
	assert.Equal(t, flag, decoded)
}

func TestCompactEncodingKeepsContextKindsAndAttributeReferences(t *testing.T) {
	flag := FeatureFlag{
		Rules: []FlagRule{{
			Clauses: []Clause{{
				ContextKind: ldcontext.Kind("org"),
				Attribute:   ldattr.NewRef("/address/city"),
				Op:          OperatorIn,
				Values:      []ldvalue.Value{ldvalue.String("x")},
				Negate:      true,
			}},
			TrackEvents: true,
		}},
	}
	data, err := marshalFeatureFlagCompact(flag)
	require.NoError(t, err)
	assert.JSONEq(t, `{"rules": [{"clauses": [{"contextKind": "org", "attribute": "/address/city",
		"op": "in", "values": ["x"], "negate": true}], "trackEvents": true}]}`, string(data))
}

func TestCompactDataModelSerializationUnmarshalsCompactEncoding(t *testing.T) {
	serialization := NewCompactJSONDataModelSerialization()

	flag := makeFlagWithDefaultValuesEverywhere()
	flagData, err := serialization.MarshalFeatureFlag(flag)
	require.NoError(t, err)
	decodedFlag, err := serialization.UnmarshalFeatureFlag(flagData)
	require.NoError(t, err)
	expectedFlag, err := unmarshalFeatureFlagFromBytes(flagData)
	require.NoError(t, err)
	assert.Equal(t, expectedFlag, decodedFlag)

	segment := makeSegmentWithDefaultValuesEverywhere()
	segmentData, err := serialization.MarshalSegment(segment)
	require.NoError(t, err)
	decodedSegment, err := serialization.UnmarshalSegment(segmentData)
	require.NoError(t, err)
	expectedSegment, err := unmarshalSegmentFromBytes(segmentData)
	require.NoError(t, err)
	assert.Equal(t, expectedSegment, decodedSegment)
}
