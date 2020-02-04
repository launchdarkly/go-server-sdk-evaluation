package evaluation

import (
	"fmt"

	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

func intPtr(n int) *int {
	return &n
}

func uint64Ptr(n uint64) *uint64 {
	return &n
}

func attrPtr(a lduser.UserAttribute) *lduser.UserAttribute {
	return &a
}

type simpleDataProvider struct {
	getFlag    func(string) (FeatureFlag, bool)
	getSegment func(string) (Segment, bool)
}

func (s *simpleDataProvider) GetFeatureFlag(key string) (FeatureFlag, bool) {
	return s.getFlag(key)
}

func (s *simpleDataProvider) GetSegment(key string) (Segment, bool) {
	return s.getSegment(key)
}

func (s *simpleDataProvider) withStoredFlags(flags ...FeatureFlag) *simpleDataProvider {
	return &simpleDataProvider{
		getFlag: func(key string) (FeatureFlag, bool) {
			for _, f := range flags {
				if f.Key == key {
					return f, true
				}
			}
			return s.getFlag(key)
		},
		getSegment: s.getSegment,
	}
}

func (s *simpleDataProvider) withNonexistentFlag(flagKey string) *simpleDataProvider {
	return &simpleDataProvider{
		getFlag: func(key string) (FeatureFlag, bool) {
			if key == flagKey {
				return FeatureFlag{}, false
			}
			return s.getFlag(key)
		},
		getSegment: s.getSegment,
	}
}

func (s *simpleDataProvider) withStoredSegments(segments ...Segment) *simpleDataProvider {
	return &simpleDataProvider{
		getFlag: s.getFlag,
		getSegment: func(key string) (Segment, bool) {
			for _, seg := range segments {
				if seg.Key == key {
					return seg, true
				}
			}
			return s.getSegment(key)
		},
	}
}

func (s *simpleDataProvider) withNonexistentSegment(segmentKey string) *simpleDataProvider {
	return &simpleDataProvider{
		getFlag: s.getFlag,
		getSegment: func(key string) (Segment, bool) {
			if key == segmentKey {
				return Segment{}, false
			}
			return s.getSegment(key)
		},
	}
}

func basicDataProvider() *simpleDataProvider {
	return &simpleDataProvider{
		getFlag: func(key string) (FeatureFlag, bool) {
			panic(fmt.Errorf("unexpectedly queried feature flag: %s", key))
		},
		getSegment: func(key string) (Segment, bool) {
			panic(fmt.Errorf("unexpectedly queried segment: %s", key))
		},
	}
}

func basicEvaluator() Evaluator {
	return NewEvaluator(basicDataProvider())
}

type prereqEventSink struct {
	events []PrerequisiteFlagEvent
}

func (p *prereqEventSink) record(event PrerequisiteFlagEvent) {
	p.events = append(p.events, event)
}

func makeClauseToMatchUser(user lduser.User) Clause {
	return Clause{
		Attribute: "key",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.String(user.GetKey())},
	}
}

func makeClauseToNotMatchUser(user lduser.User) Clause {
	return Clause{
		Attribute: "key",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.String("not-" + user.GetKey())},
	}
}

func makeFlagToMatchUser(user lduser.User, variationOrRollout VariationOrRollout) FeatureFlag {
	return FeatureFlag{
		Key:          "feature",
		On:           true,
		OffVariation: intPtr(1),
		Rules: []FlagRule{
			FlagRule{
				ID:                 "rule-id",
				Clauses:            []Clause{makeClauseToMatchUser(user)},
				VariationOrRollout: variationOrRollout,
			},
		},
		Fallthrough: VariationOrRollout{Variation: intPtr(0)},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}
}

func booleanFlagWithClause(clause Clause) FeatureFlag {
	return FeatureFlag{
		Key: "feature",
		On:  true,
		Rules: []FlagRule{
			FlagRule{Clauses: []Clause{clause}, VariationOrRollout: VariationOrRollout{Variation: intPtr(1)}},
		},
		Fallthrough: VariationOrRollout{Variation: intPtr(0)},
		Variations:  []ldvalue.Value{ldvalue.Bool(false), ldvalue.Bool(true)},
	}
}

func booleanFlagWithSegmentMatch(segmentKeys ...string) FeatureFlag {
	clause := Clause{Attribute: "", Op: "segmentMatch"}
	for _, key := range segmentKeys {
		clause.Values = append(clause.Values, ldvalue.String(key))
	}
	return booleanFlagWithClause(clause)
}
