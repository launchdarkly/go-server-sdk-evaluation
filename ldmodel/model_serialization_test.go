package ldmodel

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldtime"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

var flagWithAllProperties = FeatureFlag{
	Key: "flag-key",
	On:  true,
	Prerequisites: []Prerequisite{
		Prerequisite{
			Key:       "prereq-key",
			Variation: 1,
		},
	},
	Targets: []Target{
		Target{
			Values:       []string{"user-key"},
			Variation:    2,
			preprocessed: targetPreprocessedData{valuesMap: map[string]bool{"user-key": true}}, // this is set by PreprocessFlag()
		},
	},
	Rules: []FlagRule{
		FlagRule{
			ID: "rule-id1",
			Clauses: []Clause{
				Clause{
					Attribute: lduser.NameAttribute,
					Op:        OperatorIn,
					Values:    []ldvalue.Value{ldvalue.String("clause-value")},
					Negate:    true,
				},
			},
			VariationOrRollout: VariationOrRollout{
				Variation: 1,
			},
			TrackEvents: true,
		},
		FlagRule{
			ID:      "rule-id2",
			Clauses: []Clause{},
			VariationOrRollout: VariationOrRollout{
				Variation: NoVariation,
				Rollout: Rollout{
					Variations: []WeightedVariation{
						WeightedVariation{
							Weight:    100000,
							Variation: 3,
						},
					},
					BucketBy: lduser.NameAttribute,
				},
			},
		},
	},
	Fallthrough: VariationOrRollout{
		Variation: NoVariation,
		Rollout: Rollout{
			Variations: []WeightedVariation{
				WeightedVariation{
					Weight:    100000,
					Variation: 3,
				},
			},
		},
	},
	OffVariation: 3,
	Variations:   []ldvalue.Value{ldvalue.Bool(false), ldvalue.Int(9), ldvalue.String("other")},
	ClientSideAvailability: ClientSideAvailability{
		UsingEnvironmentID: true,
		UsingMobileKey:     true,
	},
	Salt:                   "flag-salt",
	TrackEvents:            true,
	TrackEventsFallthrough: true,
	DebugEventsUntilDate:   ldtime.UnixMillisecondTime(1000),
	Version:                99,
	Deleted:                true,
}

var flagWithAllPropertiesJSON = map[string]interface{}{
	"key": "flag-key",
	"on":  true,
	"prerequisites": []interface{}{
		map[string]interface{}{
			"key":       "prereq-key",
			"variation": float64(1),
		},
	},
	"targets": []interface{}{
		map[string]interface{}{
			"values":    []interface{}{"user-key"},
			"variation": float64(2),
		},
	},
	"rules": []interface{}{
		map[string]interface{}{
			"id": "rule-id1",
			"clauses": []interface{}{
				map[string]interface{}{
					"attribute": "name",
					"op":        "in",
					"values":    []interface{}{"clause-value"},
					"negate":    true,
				},
			},
			"variation":   float64(1),
			"trackEvents": true,
		},
		map[string]interface{}{
			"id":      "rule-id2",
			"clauses": []interface{}{},
			"rollout": map[string]interface{}{
				"variations": []interface{}{
					map[string]interface{}{
						"weight":    float64(100000),
						"variation": float64(3),
					},
				},
				"bucketBy": "name",
			},
		},
	},
	"fallthrough": map[string]interface{}{
		"rollout": map[string]interface{}{
			"variations": []interface{}{
				map[string]interface{}{
					"weight":    float64(100000),
					"variation": float64(3),
				},
			},
		},
	},
	"offVariation": float64(3),
	"variations":   []interface{}{false, float64(9), "other"},
	"clientSideAvailability": map[string]interface{}{
		"usingEnvironmentId": true,
		"usingMobileKey":     true,
	},
	"clientSide":             true,
	"salt":                   "flag-salt",
	"trackEvents":            true,
	"trackEventsFallthrough": true,
	"debugEventsUntilDate":   float64(1000),
	"version":                float64(99),
	"deleted":                true,
}

var flagWithMinimalProperties = FeatureFlag{
	Key:          "flag-key",
	Fallthrough:  VariationOrRollout{Variation: 1},
	OffVariation: NoVariation,
	Variations:   []ldvalue.Value{ldvalue.Bool(false), ldvalue.Int(9), ldvalue.String("other")},
	Salt:         "flag-salt",
	Version:      99,
}

var flagWithMinimalPropertiesJSON = map[string]interface{}{
	"key": "flag-key",
	"fallthrough": map[string]interface{}{
		"variation": float64(1),
	},
	"variations": []interface{}{false, float64(9), "other"},
	"salt":       "flag-salt",
	"version":    float64(99),
}

var segmentWithAllProperties = Segment{
	Key:      "segment-key",
	Included: []string{"user1"},
	Excluded: []string{"user2"},
	preprocessed: segmentPreprocessedData{
		includeMap: map[string]bool{"user1": true},
		excludeMap: map[string]bool{"user2": true},
	},
	Rules: []SegmentRule{
		SegmentRule{
			ID: "rule-id",
			Clauses: []Clause{
				Clause{
					Attribute: lduser.NameAttribute,
					Op:        OperatorIn,
					Values:    []ldvalue.Value{ldvalue.String("clause-value")},
					Negate:    true,
				},
			},
			Weight: -1,
		},
		SegmentRule{
			Clauses:  []Clause{},
			Weight:   50000,
			BucketBy: lduser.NameAttribute,
		},
	},
	Salt:    "segment-salt",
	Version: 99,
	Deleted: true,
}

var segmentWithAllPropertiesJSON = map[string]interface{}{
	"key":      "segment-key",
	"included": []interface{}{"user1"},
	"excluded": []interface{}{"user2"},
	"rules": []interface{}{
		map[string]interface{}{
			"id": "rule-id",
			"clauses": []interface{}{
				map[string]interface{}{
					"attribute": "name",
					"op":        "in",
					"values":    []interface{}{"clause-value"},
					"negate":    true,
				},
			},
		},
		map[string]interface{}{
			"clauses":  []interface{}{},
			"weight":   float64(50000),
			"bucketBy": "name",
		},
	},
	"salt":    "segment-salt",
	"version": float64(99),
	"deleted": true,
}

var segmentWithMinimalProperties = Segment{
	Key:     "segment-key",
	Salt:    "segment-salt",
	Version: 99,
}

var segmentWithMinimalPropertiesJSON = map[string]interface{}{
	"key":     "segment-key",
	"salt":    "segment-salt",
	"version": float64(99),
}

func parseJsonMap(t *testing.T, bytes []byte) map[string]interface{} {
	var ret map[string]interface{}
	require.NoError(t, json.Unmarshal(bytes, &ret))
	return ret
}

func TestMarshalFlagWithAllProperties(t *testing.T) {
	bytes, err := NewJSONDataModelSerialization().MarshalFeatureFlag(flagWithAllProperties)
	require.NoError(t, err)
	json := parseJsonMap(t, bytes)
	assert.Equal(t, flagWithAllPropertiesJSON, json)
}

func TestMarshalFlagWithMinimalProperties(t *testing.T) {
	bytes, err := NewJSONDataModelSerialization().MarshalFeatureFlag(flagWithMinimalProperties)
	require.NoError(t, err)
	json := parseJsonMap(t, bytes)
	assert.Equal(t, flagWithMinimalPropertiesJSON, json)
}

func TestUnmarshalFlagWithAllProperties(t *testing.T) {
	bytes, err := json.Marshal(flagWithAllPropertiesJSON)
	require.NoError(t, err)
	flag, err := NewJSONDataModelSerialization().UnmarshalFeatureFlag(bytes)
	require.NoError(t, err)
	assert.Equal(t, flagWithAllProperties, flag)
}

func TestUnmarshalFlagWithMinimalProperties(t *testing.T) {
	bytes, err := json.Marshal(flagWithMinimalPropertiesJSON)
	require.NoError(t, err)
	flag, err := NewJSONDataModelSerialization().UnmarshalFeatureFlag(bytes)
	require.NoError(t, err)
	assert.Equal(t, flagWithMinimalProperties, flag)
}

func TestUnmarshalFlagErrors(t *testing.T) {
	_, err := NewJSONDataModelSerialization().UnmarshalFeatureFlag([]byte(`{`))
	assert.Error(t, err)

	_, err = NewJSONDataModelSerialization().UnmarshalFeatureFlag([]byte(`{"key":[]}`))
	assert.Error(t, err)
}

func TestMarshalSegmentWithAllProperties(t *testing.T) {
	bytes, err := NewJSONDataModelSerialization().MarshalSegment(segmentWithAllProperties)
	require.NoError(t, err)
	json := parseJsonMap(t, bytes)
	assert.Equal(t, segmentWithAllPropertiesJSON, json)
}

func TestMarshalSegmentWithMinimalProperties(t *testing.T) {
	bytes, err := NewJSONDataModelSerialization().MarshalSegment(segmentWithMinimalProperties)
	require.NoError(t, err)
	json := parseJsonMap(t, bytes)
	assert.Equal(t, segmentWithMinimalPropertiesJSON, json)
}

func TestUnmarshalSegmentWithAllProperties(t *testing.T) {
	bytes, err := json.Marshal(segmentWithAllPropertiesJSON)
	require.NoError(t, err)
	segment, err := NewJSONDataModelSerialization().UnmarshalSegment(bytes)
	require.NoError(t, err)
	assert.Equal(t, segmentWithAllProperties, segment)
	assert.Equal(t, segmentWithAllProperties.Key, segment.GetKey())
	assert.Equal(t, segmentWithAllProperties.Version, segment.GetVersion())
	assert.Equal(t, segmentWithAllProperties.Deleted, segment.IsDeleted())
}

func TestUnmarshalSegmentWithMinimalProperties(t *testing.T) {
	bytes, err := json.Marshal(segmentWithMinimalPropertiesJSON)
	require.NoError(t, err)
	segment, err := NewJSONDataModelSerialization().UnmarshalSegment(bytes)
	require.NoError(t, err)
	assert.Equal(t, segmentWithMinimalProperties, segment)
	assert.Equal(t, segmentWithMinimalProperties.Key, segment.GetKey())
	assert.Equal(t, segmentWithMinimalProperties.Version, segment.GetVersion())
	assert.Equal(t, segmentWithMinimalProperties.Deleted, segment.IsDeleted())
}

func TestUnmarshalSegmentErrors(t *testing.T) {
	_, err := NewJSONDataModelSerialization().UnmarshalSegment([]byte(`{`))
	assert.Error(t, err)

	_, err = NewJSONDataModelSerialization().UnmarshalSegment([]byte(`{"key":[]}`))
	assert.Error(t, err)
}

func TestJSONMarshalUsesSameSerialization(t *testing.T) {
	f1, _ := NewJSONDataModelSerialization().MarshalFeatureFlag(flagWithMinimalProperties)
	f2, _ := json.Marshal(flagWithMinimalProperties)
	assert.Equal(t, f1, f2)

	s1, _ := NewJSONDataModelSerialization().MarshalSegment(segmentWithMinimalProperties)
	s2, _ := json.Marshal(segmentWithMinimalProperties)
	assert.Equal(t, s1, s2)
}

func TestJSONUnmarshalUsesSameSerialization(t *testing.T) {
	fbytes, _ := json.Marshal(flagWithMinimalPropertiesJSON)
	f1, _ := NewJSONDataModelSerialization().UnmarshalFeatureFlag(fbytes)
	var f2 FeatureFlag
	_ = json.Unmarshal(fbytes, &f2)
	assert.Equal(t, f1, f2)

	sbytes, _ := json.Marshal(segmentWithMinimalPropertiesJSON)
	s1, _ := NewJSONDataModelSerialization().UnmarshalSegment(sbytes)
	var s2 Segment
	_ = json.Unmarshal(sbytes, &s2)
	assert.Equal(t, s1, s2)
}
