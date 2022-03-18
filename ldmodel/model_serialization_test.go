package ldmodel

import (
	"encoding/json"
	"testing"

	"github.com/launchdarkly/go-jsonstream/v2/jreader"
	"github.com/launchdarkly/go-jsonstream/v2/jwriter"
	m "github.com/launchdarkly/go-test-helpers/v2/matchers"

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
			m.In(t).Assert(json.RawMessage(bytes), m.JSONEqual(expected))
		})
	}
}

func doMarshalSegmentTest(t *testing.T, marshalFn testMarshalSegmentFn) {
	for _, p := range makeSegmentSerializationTestParams() {
		t.Run(p.name, func(t *testing.T) {
			bytes, err := marshalFn(p.segment)
			require.NoError(t, err)
			expected := mergeDefaultProperties(json.RawMessage(p.jsonString), segmentTopLevelDefaultProperties)
			m.In(t).Assert(json.RawMessage(bytes), m.JSONEqual(expected))
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
