package ldmodel

import (
	"github.com/launchdarkly/go-jsonstream/v4/jreader"
	"github.com/launchdarkly/go-jsonstream/v4/jwriter"
)

// DataModelSerialization is an abstraction of an encoding for SDK data model objects.
//
// The ldmodel package defines a standard JSON schema for FeatureFlag and Segment. There are two
// implementations of DataModelSerialization. NewJSONDataModelSerialization() writes the full JSON
// encoding. NewCompactJSONDataModelSerialization() writes the compact JSON encoding, which omits
// properties that have their default values. Both implementations read either encoding.
//
// There are also other ways to convert these types to and from the JSON encoding:
//
// 1. FeatureFlag and Segment define MarshalJSON and UnmarshalJSON methods so that they wil be
// correctly encoded or decoded if you call Go's standard json.Marshal or json.Unmarshal.
//
// 2. There are equivalent methods for encoding and decoding via the go-jsonstream API
// (https://pkg.go.dev/github.com/launchdarkly/go-jsonstream/v4). These are used internally by the
// SDK to avoid inefficiencies in json.Marshal and json.Unmarshal.
//
// There is no separately defined encoding for lower-level data model types such as FlagRule, since
// there is no guarantee that those will always be represented as individual JSON objects in future
// versions of the schema. If you want to create a JSON representation of those data structures you
// must define your own type and copy values into it.
type DataModelSerialization interface {
	// MarshalFeatureFlag converts a FeatureFlag into its serialized encoding.
	MarshalFeatureFlag(item FeatureFlag) ([]byte, error)

	// MarshalSegment converts a Segment into its serialized encoding.
	MarshalSegment(item Segment) ([]byte, error)

	// UnmarshalFeatureFlag attempts to convert a FeatureFlag from its serialized encoding.
	UnmarshalFeatureFlag(data []byte) (FeatureFlag, error)

	// UnmarshalSegment attempts to convert a Segment from its serialized encoding.
	UnmarshalSegment(data []byte) (Segment, error)
}

// MarshalFeatureFlagToJSONWriter attempts to convert a FeatureFlag to JSON using the jsonstream API.
// For details, see: https://github.com/launchdarkly/go-jsonstream/v4
func MarshalFeatureFlagToJSONWriter(item FeatureFlag, writer *jwriter.Writer) {
	marshalFeatureFlagToWriter(item, writer)
}

// MarshalSegmentToJSONWriter attempts to convert a Segment to JSON using the jsonstream API.
// For details, see: https://github.com/launchdarkly/go-jsonstream/v4
func MarshalSegmentToJSONWriter(item Segment, writer *jwriter.Writer) {
	marshalSegmentToWriter(item, writer)
}

// MarshalFeatureFlagToJSONWriterCompact converts a FeatureFlag to the compact JSON encoding using the
// jsonstream API.
//
// The compact encoding omits each property whose absence the unmarshaler reads as the same value,
// such as a false boolean, a zero integer, an empty string, or an empty array. Only send it to SDKs
// that can read it. For the full encoding, use MarshalFeatureFlagToJSONWriter.
func MarshalFeatureFlagToJSONWriterCompact(item FeatureFlag, writer *jwriter.Writer) {
	marshalFeatureFlagToWriterCompact(item, writer)
}

// MarshalSegmentToJSONWriterCompact converts a Segment to the compact JSON encoding using the
// jsonstream API.
//
// The compact encoding omits each property whose absence the unmarshaler reads as the same value,
// such as a false boolean, a zero integer, an empty string, or an empty array. Only send it to SDKs
// that can read it. For the full encoding, use MarshalSegmentToJSONWriter.
func MarshalSegmentToJSONWriterCompact(item Segment, writer *jwriter.Writer) {
	marshalSegmentToWriterCompact(item, writer)
}

// UnmarshalFeatureFlagFromJSONReader attempts to convert a FeatureFlag from JSON using the jsonstream
// API. For details, see: https://github.com/launchdarkly/go-jsonstream/v4
func UnmarshalFeatureFlagFromJSONReader(reader *jreader.Reader) FeatureFlag {
	return unmarshalFeatureFlagFromReader(reader)
}

// UnmarshalSegmentFromJSONReader attempts to convert a Segment from JSON using the jsonstream API.
// For details, see: https://github.com/launchdarkly/go-jsonstream/v4
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

type compactJSONDataModelSerialization struct{}

// NewCompactJSONDataModelSerialization provides the compact JSON encoding for SDK data model objects.
//
// The compact encoding omits each property whose absence the unmarshaler reads as the same value,
// such as a false boolean, a zero integer, an empty string, or an empty array. Only send it to SDKs
// that can read it. Unmarshaling is the same as for NewJSONDataModelSerialization(), and reads
// either encoding.
func NewCompactJSONDataModelSerialization() DataModelSerialization {
	return compactJSONDataModelSerialization{}
}

func (s compactJSONDataModelSerialization) MarshalFeatureFlag(item FeatureFlag) ([]byte, error) {
	return marshalFeatureFlagCompact(item)
}

func (s compactJSONDataModelSerialization) MarshalSegment(item Segment) ([]byte, error) {
	return marshalSegmentCompact(item)
}

func (s compactJSONDataModelSerialization) UnmarshalFeatureFlag(data []byte) (FeatureFlag, error) {
	return unmarshalFeatureFlagFromBytes(data)
}

func (s compactJSONDataModelSerialization) UnmarshalSegment(data []byte) (Segment, error) {
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
