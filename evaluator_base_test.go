package evaluation

import (
	"fmt"
	"testing"

	"github.com/launchdarkly/go-sdk-common/v3/ldattr"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldreason"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	"github.com/launchdarkly/go-server-sdk-evaluation/v2/ldbuilders"
	"github.com/launchdarkly/go-server-sdk-evaluation/v2/ldmodel"
	"github.com/stretchr/testify/assert"
)

func assertResultDetail(t *testing.T, expected ldreason.EvaluationDetail, result Result) {
	assert.Equal(t, expected, result.Detail)
}

type simpleDataProvider struct {
	getFlag           func(string) *ldmodel.FeatureFlag
	getSegment        func(string) *ldmodel.Segment
	getConfigOverride func(string) *ldmodel.ConfigOverride
	getMetric         func(string) *ldmodel.Metric
}

func (s *simpleDataProvider) GetFeatureFlag(key string) *ldmodel.FeatureFlag {
	return s.getFlag(key)
}

func (s *simpleDataProvider) GetSegment(key string) *ldmodel.Segment {
	return s.getSegment(key)
}

func (s *simpleDataProvider) GetConfigOverride(key string) *ldmodel.ConfigOverride {
	return s.getConfigOverride(key)
}

func (s *simpleDataProvider) GetMetric(key string) *ldmodel.Metric {
	return s.getMetric(key)
}

func (s *simpleDataProvider) withStoredFlags(flags ...ldmodel.FeatureFlag) *simpleDataProvider {
	return &simpleDataProvider{
		getFlag: func(key string) *ldmodel.FeatureFlag {
			for _, f := range flags {
				if f.Key == key {
					ff := f
					return &ff
				}
			}
			return s.getFlag(key)
		},
		getSegment:        s.getSegment,
		getConfigOverride: s.getConfigOverride,
		getMetric:         s.getMetric,
	}
}

func (s *simpleDataProvider) withNonexistentFlag(flagKey string) *simpleDataProvider {
	return &simpleDataProvider{
		getFlag: func(key string) *ldmodel.FeatureFlag {
			if key == flagKey {
				return nil
			}
			return s.getFlag(key)
		},
		getSegment: s.getSegment,
	}
}

func (s *simpleDataProvider) withStoredSegments(segments ...ldmodel.Segment) *simpleDataProvider {
	return &simpleDataProvider{
		getFlag: s.getFlag,
		getSegment: func(key string) *ldmodel.Segment {
			for _, seg := range segments {
				if seg.Key == key {
					ss := seg
					return &ss
				}
			}
			return s.getSegment(key)
		},
		getConfigOverride: s.getConfigOverride,
		getMetric:         s.getMetric,
	}
}

func (s *simpleDataProvider) withConfigOverrides(overrides ...ldmodel.ConfigOverride) *simpleDataProvider {
	return &simpleDataProvider{
		getFlag:    s.getFlag,
		getSegment: s.getSegment,
		getConfigOverride: func(key string) *ldmodel.ConfigOverride {
			for _, override := range overrides {
				if override.Key == key {
					o := override
					return &o
				}
			}
			return s.getConfigOverride(key)
		},
		getMetric: s.getMetric,
	}
}

func (s *simpleDataProvider) withNonexistentSegment(segmentKey string) *simpleDataProvider {
	return &simpleDataProvider{
		getFlag: s.getFlag,
		getSegment: func(key string) *ldmodel.Segment {
			if key == segmentKey {
				return nil
			}
			return s.getSegment(key)
		},
	}
}

func basicDataProvider() *simpleDataProvider {
	return &simpleDataProvider{
		getFlag: func(key string) *ldmodel.FeatureFlag {
			panic(fmt.Errorf("unexpectedly queried feature flag: %s", key))
		},
		getSegment: func(key string) *ldmodel.Segment {
			panic(fmt.Errorf("unexpectedly queried segment: %s", key))
		},
		getConfigOverride: func(key string) *ldmodel.ConfigOverride {
			panic(fmt.Errorf("unexpectedly queried config override: %s", key))
		},
		getMetric: func(key string) *ldmodel.Metric {
			panic(fmt.Errorf("unexpectedly queried metric: %s", key))
		},
	}
}

func basicEvaluator() Evaluator {
	return NewEvaluator(
		basicDataProvider().
			withConfigOverrides(
				ldbuilders.NewConfigOverrideBuilder("indexSamplingRatio").Value(ldvalue.Int(1)).Build(),
			),
	)
}

func makeClauseToMatchContext(context ldcontext.Context) ldmodel.Clause {
	return ldbuilders.ClauseWithKind(context.Kind(), ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String(context.Key()))
}

func makeClauseToMatchAnyContextOfKind(kind ldcontext.Kind) ldmodel.Clause {
	return ldbuilders.Negate(ldbuilders.ClauseWithKind(kind, ldattr.KeyAttr, ldmodel.OperatorIn, ldvalue.String("")))
}

func makeClauseToMatchAnyContextOfAnyKind() ldmodel.Clause {
	return ldbuilders.Negate(ldbuilders.Clause(ldattr.KindAttr, ldmodel.OperatorIn, ldvalue.String("")))
}

func makeFlagToMatchContext(user ldcontext.Context, variationOrRollout ldmodel.VariationOrRollout) ldmodel.FeatureFlag {
	return ldbuilders.NewFlagBuilder("feature").
		On(true).
		OffVariation(1).
		AddRule(ldbuilders.NewRuleBuilder().ID("rule-id").VariationOrRollout(variationOrRollout).
			Clauses(makeClauseToMatchContext(user))).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()
}

func makeRuleToMatchUserKeyPrefix(prefix string, variationOrRollout ldmodel.VariationOrRollout) *ldbuilders.RuleBuilder {
	return ldbuilders.NewRuleBuilder().ID("rule-id").
		VariationOrRollout(variationOrRollout).
		Clauses(ldbuilders.Clause(ldattr.KeyAttr, ldmodel.OperatorStartsWith, ldvalue.String(prefix)))
}

func makeBooleanFlagWithClauses(clauses ...ldmodel.Clause) ldmodel.FeatureFlag {
	return ldbuilders.NewFlagBuilder("feature").
		On(true).
		AddRule(ldbuilders.NewRuleBuilder().Variation(1).Clauses(clauses...)).
		Variations(ldvalue.Bool(false), ldvalue.Bool(true)).
		FallthroughVariation(0).
		Build()
}

func makeBooleanFlagToMatchAnyOfSegments(segmentKeys ...string) ldmodel.FeatureFlag {
	return makeBooleanFlagWithClauses(ldbuilders.SegmentMatchClause(segmentKeys...))
}

func makeBooleanFlagToMatchAllOfSegments(segmentKeys ...string) ldmodel.FeatureFlag {
	var clauses []ldmodel.Clause
	for _, segmentKey := range segmentKeys {
		clauses = append(clauses, ldbuilders.SegmentMatchClause(segmentKey))
	}
	return makeBooleanFlagWithClauses(clauses...)
}
