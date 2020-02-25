package evaluation

import (
	"fmt"

	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldmodel"
)

type simpleDataProvider struct {
	getFlag    func(string) (ldmodel.FeatureFlag, bool)
	getSegment func(string) (ldmodel.Segment, bool)
}

func (s *simpleDataProvider) GetFeatureFlag(key string) (ldmodel.FeatureFlag, bool) {
	return s.getFlag(key)
}

func (s *simpleDataProvider) GetSegment(key string) (ldmodel.Segment, bool) {
	return s.getSegment(key)
}

func (s *simpleDataProvider) withStoredFlags(flags ...ldmodel.FeatureFlag) *simpleDataProvider {
	return &simpleDataProvider{
		getFlag: func(key string) (ldmodel.FeatureFlag, bool) {
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
		getFlag: func(key string) (ldmodel.FeatureFlag, bool) {
			if key == flagKey {
				return ldmodel.FeatureFlag{}, false
			}
			return s.getFlag(key)
		},
		getSegment: s.getSegment,
	}
}

func (s *simpleDataProvider) withStoredSegments(segments ...ldmodel.Segment) *simpleDataProvider {
	return &simpleDataProvider{
		getFlag: s.getFlag,
		getSegment: func(key string) (ldmodel.Segment, bool) {
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
		getSegment: func(key string) (ldmodel.Segment, bool) {
			if key == segmentKey {
				return ldmodel.Segment{}, false
			}
			return s.getSegment(key)
		},
	}
}

func basicDataProvider() *simpleDataProvider {
	return &simpleDataProvider{
		getFlag: func(key string) (ldmodel.FeatureFlag, bool) {
			panic(fmt.Errorf("unexpectedly queried feature flag: %s", key))
		},
		getSegment: func(key string) (ldmodel.Segment, bool) {
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

func makeClauseToMatchUser(user lduser.User) ldmodel.Clause {
	return ldbuilders.Clause(lduser.KeyAttribute, ldmodel.OperatorIn, ldvalue.String(user.GetKey()))
}

func makeClauseToNotMatchUser(user lduser.User) ldmodel.Clause {
	return ldbuilders.Clause(lduser.KeyAttribute, ldmodel.OperatorIn, ldvalue.String("not-"+user.GetKey()))
}

func makeFlagToMatchUser(user lduser.User, variationOrRollout ldmodel.VariationOrRollout) ldmodel.FeatureFlag {
	return ldbuilders.NewFlagBuilder("feature").
		On(true).
		OffVariation(1).
		AddRule(ldbuilders.NewRuleBuilder().ID("rule-id").VariationOrRollout(variationOrRollout).Clauses(makeClauseToMatchUser(user))).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()
}

func booleanFlagWithClause(clause ldmodel.Clause) ldmodel.FeatureFlag {
	return ldbuilders.NewFlagBuilder("feature").
		On(true).
		AddRule(ldbuilders.NewRuleBuilder().Variation(1).Clauses(clause)).
		Variations(ldvalue.Bool(false), ldvalue.Bool(true)).
		FallthroughVariation(0).
		Build()
}

func booleanFlagWithSegmentMatch(segmentKeys ...string) ldmodel.FeatureFlag {
	return booleanFlagWithClause(ldbuilders.SegmentMatchClause(segmentKeys...))
}
