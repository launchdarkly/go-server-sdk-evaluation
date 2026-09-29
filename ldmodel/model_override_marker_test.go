package ldmodel

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The IsOverride marker is not part of the flag/segment data model, so it must have no
// effect on serialization, and a JSON property with the same name must not populate it.

func doFlagOverrideMarkerExclusionTest(
	t *testing.T,
	marshalFn testMarshalFlagFn,
	unmarshalFn testUnmarshalFlagFn,
) {
	for _, p := range makeFlagSerializationTestParams() {
		t.Run(p.name, func(t *testing.T) {
			unmarked := p.flag
			marked := p.flag
			marked.IsOverride = true

			unmarkedJSON, err := marshalFn(unmarked)
			require.NoError(t, err)
			markedJSON, err := marshalFn(marked)
			require.NoError(t, err)
			assert.Equal(t, string(unmarkedJSON), string(markedJSON))
			assert.NotContains(t, string(markedJSON), "sOverride")

			withProperty := addPropertyToJSONObject(t, unmarkedJSON, "isOverride", true)
			flag, err := unmarshalFn(withProperty)
			require.NoError(t, err)
			assert.False(t, flag.IsOverride)
		})
	}
}

func doSegmentOverrideMarkerExclusionTest(
	t *testing.T,
	marshalFn testMarshalSegmentFn,
	unmarshalFn testUnmarshalSegmentFn,
) {
	for _, p := range makeSegmentSerializationTestParams() {
		t.Run(p.name, func(t *testing.T) {
			unmarked := p.segment
			marked := p.segment
			marked.IsOverride = true

			unmarkedJSON, err := marshalFn(unmarked)
			require.NoError(t, err)
			markedJSON, err := marshalFn(marked)
			require.NoError(t, err)
			assert.Equal(t, string(unmarkedJSON), string(markedJSON))
			assert.NotContains(t, string(markedJSON), "sOverride")

			withProperty := addPropertyToJSONObject(t, unmarkedJSON, "isOverride", true)
			segment, err := unmarshalFn(withProperty)
			require.NoError(t, err)
			assert.False(t, segment.IsOverride)
		})
	}
}

func addPropertyToJSONObject(t *testing.T, objectJSON []byte, name string, value interface{}) []byte {
	var props map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(objectJSON, &props))
	valueJSON, err := json.Marshal(value)
	require.NoError(t, err)
	props[name] = valueJSON
	result, err := json.Marshal(props)
	require.NoError(t, err)
	return result
}

func TestFlagOverrideMarkerIsExcludedFromJSON(t *testing.T) {
	doFlagOverrideMarkerExclusionTest(t,
		func(flag FeatureFlag) ([]byte, error) { return json.Marshal(flag) },
		func(data []byte) (FeatureFlag, error) {
			var flag FeatureFlag
			err := json.Unmarshal(data, &flag)
			return flag, err
		},
	)
	t.Run("default serialization", func(t *testing.T) {
		doFlagOverrideMarkerExclusionTest(t,
			NewJSONDataModelSerialization().MarshalFeatureFlag,
			NewJSONDataModelSerialization().UnmarshalFeatureFlag,
		)
	})
}

func TestSegmentOverrideMarkerIsExcludedFromJSON(t *testing.T) {
	doSegmentOverrideMarkerExclusionTest(t,
		func(segment Segment) ([]byte, error) { return json.Marshal(segment) },
		func(data []byte) (Segment, error) {
			var segment Segment
			err := json.Unmarshal(data, &segment)
			return segment, err
		},
	)
	t.Run("default serialization", func(t *testing.T) {
		doSegmentOverrideMarkerExclusionTest(t,
			NewJSONDataModelSerialization().MarshalSegment,
			NewJSONDataModelSerialization().UnmarshalSegment,
		)
	})
}
