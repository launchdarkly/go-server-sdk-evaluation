package evaluation

import (
	"encoding/json"
)

// DataModelSerialization defines an encoding for SDK data model objects.
//
// For the default JSON encoding used by LaunchDarkly SDKs, use JSONDataModelSerialization.
type DataModelSerialization interface {
	MarshalFeatureFlag(item FeatureFlag) ([]byte, error)
	MarshalSegment(item Segment) ([]byte, error)
	UnmarshalFeatureFlag(data []byte) (FeatureFlag, error)
	UnmarshalSegment(data []byte) (Segment, error)
}

// JSONDataModelSerialization is the default JSON encoding for SDK data model objects.
//
// Always use this rather than relying on json.Marshal() and json.Unmarshal(). The data model
// structs are guaranteed to serialize and deserialize correctly with json.Marshal() and
// json.Unmarshal(), but JSONDataModelSerialization may be enhanced in the future to use a
// more efficient mechanism.
type JSONDataModelSerialization struct{}

// NewJSONDataModelSerialization() provides the default JSON encoding for SDK data model objects.
func NewJSONDataModelSerialization() *JSONDataModelSerialization {
	return &JSONDataModelSerialization{}
}

func (s *JSONDataModelSerialization) MarshalFeatureFlag(item FeatureFlag) ([]byte, error) {
	return json.Marshal(item)
}

func (s *JSONDataModelSerialization) MarshalSegment(item Segment) ([]byte, error) {
	return json.Marshal(item)
}

func (s *JSONDataModelSerialization) UnmarshalFeatureFlag(data []byte) (FeatureFlag, error) {
	var item FeatureFlag
	err := json.Unmarshal(data, &item)
	return item, err
}

func (s *JSONDataModelSerialization) UnmarshalSegment(data []byte) (Segment, error) {
	var item Segment
	err := json.Unmarshal(data, &item)
	return item, err
}
