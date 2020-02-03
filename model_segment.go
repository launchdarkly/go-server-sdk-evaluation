package evaluation

import "gopkg.in/launchdarkly/go-sdk-common.v2/lduser"

// Segment describes a group of users
type Segment struct {
	Key      string        `json:"key" bson:"key"`
	Included []string      `json:"included" bson:"included"`
	Excluded []string      `json:"excluded" bson:"excluded"`
	Salt     string        `json:"salt" bson:"salt"`
	Rules    []SegmentRule `json:"rules" bson:"rules"`
	Version  int           `json:"version" bson:"version"`
	Deleted  bool          `json:"deleted" bson:"deleted"`
}

// SegmentRule describes a set of clauses that
type SegmentRule struct {
	Id       string                `json:"id,omitempty" bson:"id,omitempty"`
	Clauses  []Clause              `json:"clauses" bson:"clauses"`
	Weight   *int                  `json:"weight,omitempty" bson:"weight,omitempty"`
	BucketBy *lduser.UserAttribute `json:"bucketBy,omitempty" bson:"bucketBy,omitempty"`
}
