package ldmodel

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
)

// Test that stdlib vs custom unmarshal produce equivalent results after
// pre-process. Keep this light-weight and deterministic.
func TestSerializationEquivalence_SimpleFlag(t *testing.T) {
	bytes, _ := json.Marshal(flagWithAllPropertiesJSON)

	var std FeatureFlag
	if err := json.Unmarshal(bytes, &std); err != nil {
		t.Fatalf("stdlib unmarshal: %v", err)
	}
	PreprocessFlag(&std)

	custom, err := unmarshalFeatureFlagFromBytes(bytes)
	if err != nil {
		t.Fatalf("custom unmarshal: %v", err)
	}

	if diff := cmp.Diff(std, custom, cmpOpts()...); diff != "" {
		t.Errorf("FeatureFlag mismatch (-want +got):\n%s", diff)
	}
}

func TestSerializationEquivalence_SimpleSegment(t *testing.T) {
	bytes, _ := json.Marshal(segmentWithAllPropertiesJSON)

	var std Segment
	if err := json.Unmarshal(bytes, &std); err != nil {
		t.Fatalf("stdlib unmarshal: %v", err)
	}
	PreprocessSegment(&std)

	custom, err := unmarshalSegmentFromBytes(bytes)
	if err != nil {
		t.Fatalf("custom unmarshal: %v", err)
	}

	if diff := cmp.Diff(std, custom, cmpOpts()...); diff != "" {
		t.Errorf("Segment mismatch (-want +got):\n%s", diff)
	}
}

func cmpOpts() []cmp.Option {
	return []cmp.Option{
		cmpopts.EquateEmpty(),
		cmpopts.IgnoreUnexported(Target{}, Clause{}, Segment{}, ldvalue.OptionalInt{}),
		cmp.Comparer(func(a, b ldvalue.Value) bool { return a.Equal(b) }),
	}
}
