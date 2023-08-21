package ldmodel

import (
	"encoding/json"
	"testing"

	"github.com/launchdarkly/go-jsonstream/v3/jreader"
	"github.com/launchdarkly/go-jsonstream/v3/jwriter"
	"github.com/launchdarkly/go-test-helpers/v3/jsonhelpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testMarshalFlagFn func(FeatureFlag) ([]byte, error)
type testUnmarshalFlagFn func([]byte) (FeatureFlag, error)

type testMarshalSegmentFn func(Segment) ([]byte, error)
type testUnmarshalSegmentFn func([]byte) (Segment, error)

type testMarshalConfigOverrideFn func(ConfigOverride) ([]byte, error)
type testUnmarshalConfigOverrideFn func([]byte) (ConfigOverride, error)

type testMarshalMetricFn func(Metric) ([]byte, error)
type testUnmarshalMetricFn func([]byte) (Metric, error)

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

func doMarshalConfigOverrideTest(t *testing.T, marshalFn testMarshalConfigOverrideFn) {
	for _, p := range makeConfigOverrideSerializationTestParams() {
		t.Run(p.name, func(t *testing.T) {
			bytes, err := marshalFn(p.override)
			require.NoError(t, err)
			expected := mergeDefaultProperties(json.RawMessage(p.jsonString), configOverrideTopLevelDefaultProperties)
			jsonhelpers.AssertEqual(t, expected, bytes)
		})
	}
}

func doMarshalMetricTest(t *testing.T, marshalFn testMarshalMetricFn) {
	for _, p := range makeMetricSerializationTestParams() {
		t.Run(p.name, func(t *testing.T) {
			bytes, err := marshalFn(p.metric)
			require.NoError(t, err)
			expected := mergeDefaultProperties(json.RawMessage(p.jsonString), metricTopLevelDefaultProperties)
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

func doUnmarshalConfigOverrideTest(t *testing.T, unmarshalFn testUnmarshalConfigOverrideFn) {
	for _, p := range makeConfigOverrideSerializationTestParams() {
		t.Run(p.name, func(t *testing.T) {
			override, err := unmarshalFn([]byte(p.jsonString))
			require.NoError(t, err)

			expectedConfigOverride := p.override

			assert.Equal(t, expectedConfigOverride, override)

			for _, altJSON := range p.jsonAltInputs {
				t.Run(altJSON, func(t *testing.T) {
					override, err := unmarshalFn([]byte(altJSON))
					require.NoError(t, err)
					assert.Equal(t, expectedConfigOverride, override)
				})
			}
		})
	}
}

func doUnmarshalMetricTest(t *testing.T, unmarshalFn testUnmarshalMetricFn) {
	for _, p := range makeMetricSerializationTestParams() {
		t.Run(p.name, func(t *testing.T) {
			metric, err := unmarshalFn([]byte(p.jsonString))
			require.NoError(t, err)

			expectedMetric := p.metric

			assert.Equal(t, expectedMetric, metric)

			for _, altJSON := range p.jsonAltInputs {
				t.Run(altJSON, func(t *testing.T) {
					metric, err := unmarshalFn([]byte(altJSON))
					require.NoError(t, err)
					assert.Equal(t, expectedMetric, metric)
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

func TestUnmarshalConfigOverrideErrors(t *testing.T) {
	_, err := NewJSONDataModelSerialization().UnmarshalConfigOverride([]byte(`{`))
	assert.Error(t, err)

	_, err = NewJSONDataModelSerialization().UnmarshalConfigOverride([]byte(`{"key":[]}`))
	assert.Error(t, err)
}

func TestUnmarshalMetricErrors(t *testing.T) {
	_, err := NewJSONDataModelSerialization().UnmarshalMetric([]byte(`{`))
	assert.Error(t, err)

	_, err = NewJSONDataModelSerialization().UnmarshalMetric([]byte(`{"key":[]}`))
	assert.Error(t, err)
}

func TestMarshalConfigOverrideWithJSONMarshal(t *testing.T) {
	doMarshalConfigOverrideTest(t, func(override ConfigOverride) ([]byte, error) {
		return json.Marshal(override)
	})
}

func TestMarshalConfigOverrideWithDefaultSerialization(t *testing.T) {
	doMarshalConfigOverrideTest(t, NewJSONDataModelSerialization().MarshalConfigOverride)
}

func TestMarshalConfigOverrideWithJSONWriter(t *testing.T) {
	doMarshalConfigOverrideTest(t, func(override ConfigOverride) ([]byte, error) {
		w := jwriter.NewWriter()
		MarshalConfigOverrideToJSONWriter(override, &w)
		return w.Bytes(), w.Error()
	})
}

func TestUnmarshalConfigOverrideWithJSONUnmarshal(t *testing.T) {
	doUnmarshalConfigOverrideTest(t, func(data []byte) (ConfigOverride, error) {
		var override ConfigOverride
		err := json.Unmarshal(data, &override)
		return override, err
	})
}

func TestUnmarshalConfigOverrideWithDefaultSerialization(t *testing.T) {
	doUnmarshalConfigOverrideTest(t, NewJSONDataModelSerialization().UnmarshalConfigOverride)
}

func TestUnmarshalConfigOverrideWithJSONReader(t *testing.T) {
	doUnmarshalConfigOverrideTest(t, func(data []byte) (ConfigOverride, error) {
		r := jreader.NewReader(data)
		override := UnmarshalConfigOverrideFromJSONReader(&r)
		return override, r.Error()
	})
}

func TestMarshalMetricWithJSONMarshal(t *testing.T) {
	doMarshalMetricTest(t, func(override Metric) ([]byte, error) {
		return json.Marshal(override)
	})
}

func TestMarshalMetricWithDefaultSerialization(t *testing.T) {
	doMarshalMetricTest(t, NewJSONDataModelSerialization().MarshalMetric)
}

func TestMarshalMetricWithJSONWriter(t *testing.T) {
	doMarshalMetricTest(t, func(override Metric) ([]byte, error) {
		w := jwriter.NewWriter()
		MarshalMetricToJSONWriter(override, &w)
		return w.Bytes(), w.Error()
	})
}

func TestUnmarshalMetricWithJSONUnmarshal(t *testing.T) {
	doUnmarshalMetricTest(t, func(data []byte) (Metric, error) {
		var override Metric
		err := json.Unmarshal(data, &override)
		return override, err
	})
}

func TestUnmarshalMetricWithDefaultSerialization(t *testing.T) {
	doUnmarshalMetricTest(t, NewJSONDataModelSerialization().UnmarshalMetric)
}

func TestUnmarshalMetricWithJSONReader(t *testing.T) {
	doUnmarshalMetricTest(t, func(data []byte) (Metric, error) {
		r := jreader.NewReader(data)
		override := UnmarshalMetricFromJSONReader(&r)
		return override, r.Error()
	})
}
