package ldmodel

import (
	"github.com/launchdarkly/go-jsonstream/jreader"
	"github.com/launchdarkly/go-jsonstream/jwriter"
)

// DataModelSerialization defines an encoding for SDK data model objects.
//
// For the default JSON encoding used by LaunchDarkly SDKs, use NewJSONDataModelSerialization.
type DataModelSerialization interface {
	// MarshalFeatureFlag converts a FeatureFlag into its serialized encoding.
	MarshalFeatureFlag(item FeatureFlag) ([]byte, error)

	// MarshalSegment converts a Segment into its serialized encoding.
	MarshalSegment(item Segment) ([]byte, error)

	// UnmarshalFeatureFlag attempts to convert a FeatureFlag from its serialized encoding.
	UnmarshalFeatureFlag(data []byte) (FeatureFlag, error)

	// UnmarshalFeatureFlag attempts to convert a FeatureFlag from its serialized encoding.
	UnmarshalSegment(data []byte) (Segment, error)
}

// MarshalFeatureFlagToJSONWriter attempts to convert a FeatureFlag to JSON using the jsonstream API.
// For details, see: https://github.com/launchdarkly/go-jsonstream
func MarshalFeatureFlagToJSONWriter(item FeatureFlag, writer *jwriter.Writer) {
	marshalFeatureFlagToWriter(item, writer)
}

// MarshalSegmentToJSONWriter attempts to convert a Segment to JSON using the jsonstream API.
// For details, see: https://github.com/launchdarkly/go-jsonstream
func MarshalSegmentToJSONWriter(item Segment, writer *jwriter.Writer) {
	marshalSegmentToWriter(item, writer)
}

// UnmarshalFeatureFlagFromJSONReader attempts to convert a FeatureFlag from JSON using the jsonstream
// API. For details, see: https://github.com/launchdarkly/go-jsonstream
func UnmarshalFeatureFlagFromJSONReader(reader *jreader.Reader) FeatureFlag {
	return unmarshalFeatureFlagFromReader(reader)
}

// UnmarshalSegmentFromJSONReader attempts to convert a Segment from JSON using the jsonstream API.
// For details, see: https://github.com/launchdarkly/go-jsonstream
func UnmarshalSegmentFromJSONReader(reader *jreader.Reader) Segment {
	return unmarshalSegmentFromReader(reader)
}

type jsonDataModelSerialization struct{}

// NewJSONDataModelSerialization provides the default JSON encoding for SDK data model objects.
//
// Always use this rather than relying on json.Marshal() and json.Unmarshal(). The data model
// structs are guaranteed to serialize and deserialize correctly with json.Marshal() and
// json.Unmarshal(), but JSONDataModelSerialization may be enhanced in the future to use a
// more efficient mechanism.
func NewJSONDataModelSerialization() DataModelSerialization {
	return jsonDataModelSerialization{}
}

func (s jsonDataModelSerialization) MarshalFeatureFlag(item FeatureFlag) ([]byte, error) {
	return marshalFeatureFlag(item)
}

func (s jsonDataModelSerialization) MarshalSegment(item Segment) ([]byte, error) {
	return marshalSegment(item)
}

func (s jsonDataModelSerialization) UnmarshalFeatureFlag(data []byte) (FeatureFlag, error) {
	return unmarshalFeatureFlagFromBytes(data)
}

func (s jsonDataModelSerialization) UnmarshalSegment(data []byte) (Segment, error) {
	return unmarshalSegmentFromBytes(data)
}

// MarshalJSON overrides the default json.Marshal behavior to provide the same marshalling behavior that is
// used by NewJSONDataModelSerialization().
func (f FeatureFlag) MarshalJSON() ([]byte, error) {
	return marshalFeatureFlag(f)
}

// MarshalJSON overrides the default json.Marshal behavior to provide the same marshalling behavior that is
// used by NewJSONDataModelSerialization().
func (s Segment) MarshalJSON() ([]byte, error) {
	return marshalSegment(s)
}

// UnmarshalJSON overrides the default json.Unmarshal behavior to provide the same unmarshalling behavior that
// is used by NewJSONDataModelSerialization().
func (f *FeatureFlag) UnmarshalJSON(data []byte) error {
	result, err := unmarshalFeatureFlagFromBytes(data)
	if err == nil {
		*f = result
	}
	return err
}

// UnmarshalJSON overrides the default json.Unmarshal behavior to provide the same unmarshalling behavior that
// is used by NewJSONDataModelSerialization().
func (s *Segment) UnmarshalJSON(data []byte) error {
	result, err := unmarshalSegmentFromBytes(data)
	if err == nil {
		*s = result
	}
	return err
}
