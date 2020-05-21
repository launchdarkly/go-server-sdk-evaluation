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
			ID: "rule-id",
			Clauses: []Clause{
				Clause{
					Attribute: lduser.NameAttribute,
					Op:        OperatorIn,
					Values:    []ldvalue.Value{ldvalue.String("clause-value")},
					Negate:    true,
				},
			},
			VariationOrRollout: VariationOrRollout{
				Variation: intPtr(1),
			},
			TrackEvents: true,
		},
	},
	Fallthrough: VariationOrRollout{
		Rollout: &Rollout{
			Variations: []WeightedVariation{
				WeightedVariation{
					Weight:    100000,
					Variation: 3,
				},
			},
		},
	},
	OffVariation:           intPtr(3),
	Variations:             []ldvalue.Value{ldvalue.Bool(false), ldvalue.Int(9), ldvalue.String("other")},
	ClientSide:             true,
	Salt:                   "flag-salt",
	TrackEvents:            true,
	TrackEventsFallthrough: true,
	DebugEventsUntilDate:   unixTimePtr(1000),
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
			"id": "rule-id",
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
	"offVariation":           float64(3),
	"variations":             []interface{}{false, float64(9), "other"},
	"clientSide":             true,
	"salt":                   "flag-salt",
	"trackEvents":            true,
	"trackEventsFallthrough": true,
	"debugEventsUntilDate":   float64(1000),
	"version":                float64(99),
	"deleted":                true,
}

var flagWithMinimalProperties = FeatureFlag{
	Key: "flag-key",
	Fallthrough: VariationOrRollout{
		Variation: intPtr(1),
	},
	Variations: []ldvalue.Value{ldvalue.Bool(false), ldvalue.Int(9), ldvalue.String("other")},
	Salt:       "flag-salt",
	Version:    99,
}

var flagWithMinimalPropertiesJSON = map[string]interface{}{
	"key":           "flag-key",
	"on":            false,
	"prerequisites": nil,
	"targets":       nil,
	"rules":         nil,
	"fallthrough": map[string]interface{}{
		"variation": float64(1),
	},
	"offVariation":           nil,
	"variations":             []interface{}{false, float64(9), "other"},
	"clientSide":             false,
	"salt":                   "flag-salt",
	"trackEvents":            false,
	"trackEventsFallthrough": false,
	"debugEventsUntilDate":   nil,
	"version":                float64(99),
	"deleted":                false,
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
			Clauses: []Clause{
				Clause{
					Attribute: lduser.NameAttribute,
					Op:        OperatorIn,
					Values:    []ldvalue.Value{ldvalue.String("clause-value")},
					Negate:    true,
				},
			},
			Weight:   intPtr(50000),
			BucketBy: attrPtr(lduser.NameAttribute),
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
			"clauses": []interface{}{
				map[string]interface{}{
					"attribute": "name",
					"op":        "in",
					"values":    []interface{}{"clause-value"},
					"negate":    true,
				},
			},
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
	"key":      "segment-key",
	"included": nil,
	"excluded": nil,
	"rules":    nil,
	"salt":     "segment-salt",
	"version":  float64(99),
	"deleted":  false,
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
	assert.Equal(t, flagWithAllProperties.Key, flag.GetKey())
	assert.Equal(t, flagWithAllProperties.Version, flag.GetVersion())
	assert.Equal(t, flagWithAllProperties.Deleted, flag.IsDeleted())
}

func TestUnmarshalFlagWithMinimalProperties(t *testing.T) {
	bytes, err := json.Marshal(flagWithMinimalPropertiesJSON)
	require.NoError(t, err)
	flag, err := NewJSONDataModelSerialization().UnmarshalFeatureFlag(bytes)
	require.NoError(t, err)
	assert.Equal(t, flagWithMinimalProperties, flag)
	assert.Equal(t, flagWithMinimalProperties.Key, flag.GetKey())
	assert.Equal(t, flagWithMinimalProperties.Version, flag.GetVersion())
	assert.Equal(t, flagWithMinimalProperties.Deleted, flag.IsDeleted())
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

func intPtr(n int) *int {
	return &n
}

func unixTimePtr(t ldtime.UnixMillisecondTime) *ldtime.UnixMillisecondTime {
	return &t
}

func attrPtr(a lduser.UserAttribute) *lduser.UserAttribute {
	return &a
}
