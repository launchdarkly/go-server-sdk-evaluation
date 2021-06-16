package ldmodel

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldtime"
	ldevents "gopkg.in/launchdarkly/go-sdk-events.v1"
)

func asFlagEventProperties(f FeatureFlag) ldevents.FlagEventProperties {
	return &f
}

func TestFlagEventPropertiesBasicProperties(t *testing.T) {
	flag := FeatureFlag{Key: "key", Version: 2}
	assert.Equal(t, "key", asFlagEventProperties(flag).GetKey())
	assert.Equal(t, 2, asFlagEventProperties(flag).GetVersion())
}

func TestIsFullEventTrackingEnabled(t *testing.T) {
	flag1 := FeatureFlag{Key: "key"}
	assert.False(t, asFlagEventProperties(flag1).IsFullEventTrackingEnabled())

	flag2 := FeatureFlag{Key: "key", TrackEvents: true}
	assert.True(t, asFlagEventProperties(flag2).IsFullEventTrackingEnabled())
}

func TestGetDebugEventsUntilDate(t *testing.T) {
	flag1 := FeatureFlag{Key: "key"}
	assert.Equal(t, ldtime.UnixMillisecondTime(0), asFlagEventProperties(flag1).GetDebugEventsUntilDate())

	date := ldtime.UnixMillisecondTime(100000)
	flag2 := FeatureFlag{Key: "key", DebugEventsUntilDate: date}
	assert.Equal(t, date, asFlagEventProperties(flag2).GetDebugEventsUntilDate())
}

func TestIsExperimentationEnabledDefaultsToFalse(t *testing.T) {
	flag := FeatureFlag{Key: "key"}
	assert.False(t, asFlagEventProperties(flag).IsExperimentationEnabled(ldreason.NewEvalReasonOff()))
}

func TestExperimentationIsNotEnabledForFallthroughIfTrackEventsFallthroughIsFalse(t *testing.T) {
	flag := FeatureFlag{Key: "key"}
	assert.False(t, asFlagEventProperties(flag).IsExperimentationEnabled(ldreason.NewEvalReasonFallthrough()))
}

func TestExperimentationIsEnabledForFallthroughIfTrackEventsFallthroughIsTrue(t *testing.T) {
	flag := FeatureFlag{Key: "key", TrackEventsFallthrough: true}
	assert.True(t, asFlagEventProperties(flag).IsExperimentationEnabled(ldreason.NewEvalReasonFallthrough()))
}

func TestExperimentationIsEnabledForFallthroughExperimentIfReasonSaysSo(t *testing.T) {
	flag := FeatureFlag{
		Key: "key",
		Fallthrough: VariationOrRollout{
			Rollout: Rollout{Kind: RolloutKindExperiment},
		},
	}
	flagProps := asFlagEventProperties(flag)

	assert.True(t, flagProps.IsExperimentationEnabled(ldreason.NewEvalReasonFallthroughExperiment(true)))

	// these cases should be equivalent
	assert.False(t, flagProps.IsExperimentationEnabled(ldreason.NewEvalReasonFallthroughExperiment(false)))
	assert.False(t, flagProps.IsExperimentationEnabled(ldreason.NewEvalReasonFallthrough()))

	t.Run(`should fall back to rule exclusion logic when not IsInExperiment and TrackEventsFallthrough is true`, func(t *testing.T) {
		flagTrackFallthrough := flag
		flagTrackFallthrough.TrackEventsFallthrough = true
		flagPropsTrackFallthrough := asFlagEventProperties(flagTrackFallthrough)

		assert.True(t, flagPropsTrackFallthrough.IsExperimentationEnabled(ldreason.NewEvalReasonFallthroughExperiment(true)))

		// these cases should be equivalent
		assert.True(t, flagPropsTrackFallthrough.IsExperimentationEnabled(ldreason.NewEvalReasonFallthroughExperiment(false)))
		assert.True(t, flagPropsTrackFallthrough.IsExperimentationEnabled(ldreason.NewEvalReasonFallthrough()))
	})
}

func TestExperimentationIsNotEnabledForRuleMatchIfTrackEventsIsFalseForThatRule(t *testing.T) {
	flag := FeatureFlag{
		Key: "key",
		Rules: []FlagRule{
			{ID: "rule0", TrackEvents: true},
			{ID: "rule1", TrackEvents: false},
		},
	}
	reason := ldreason.NewEvalReasonRuleMatch(1, "rule1")
	assert.False(t, asFlagEventProperties(flag).IsExperimentationEnabled(reason))
}

func TestExperimentationIsEnabledForRuleMatchIfTrackEventsIsTrueForThatRule(t *testing.T) {
	flag := FeatureFlag{
		Key: "key",
		Rules: []FlagRule{
			{ID: "rule0", TrackEvents: true},
			{ID: "rule1", TrackEvents: false},
		},
	}
	reason := ldreason.NewEvalReasonRuleMatch(0, "rule0")
	assert.True(t, asFlagEventProperties(flag).IsExperimentationEnabled(reason))
}

func TestExperimentationIsEnabledForRuleMatchExperimentIfReasonSaysSo(t *testing.T) {
	flag := FeatureFlag{
		Key: "key",
		Rules: []FlagRule{{
			ID: "rule0",
			VariationOrRollout: VariationOrRollout{
				Rollout: Rollout{Kind: RolloutKindExperiment},
			},
		}},
	}
	flagProps := asFlagEventProperties(flag)

	assert.True(t, flagProps.IsExperimentationEnabled(ldreason.NewEvalReasonRuleMatchExperiment(0, "rule0", true)))

	// these cases should be equivalent
	assert.False(t, flagProps.IsExperimentationEnabled(ldreason.NewEvalReasonRuleMatchExperiment(0, "rule0", false)))
	assert.False(t, flagProps.IsExperimentationEnabled(ldreason.NewEvalReasonRuleMatch(0, "rule0")))

	t.Run(`should fall back to rule exclusion logic when not IsInExperiment and rule.TrackEvents is true"`, func(t *testing.T) {
		flagTrackRule := FeatureFlag{
			Key: "key",
			Rules: []FlagRule{{
				ID:          "rule0",
				TrackEvents: true,
				VariationOrRollout: VariationOrRollout{
					Rollout: Rollout{Kind: RolloutKindExperiment},
				},
			}},
		}
		flagPropsTrackRule := asFlagEventProperties(flagTrackRule)

		assert.True(t, flagPropsTrackRule.IsExperimentationEnabled(ldreason.NewEvalReasonRuleMatchExperiment(0, "rule0", true)))

		// these cases should be equivalent
		assert.True(t, flagPropsTrackRule.IsExperimentationEnabled(ldreason.NewEvalReasonRuleMatchExperiment(0, "rule0", false)))
		assert.True(t, flagPropsTrackRule.IsExperimentationEnabled(ldreason.NewEvalReasonRuleMatch(0, "rule0")))
	})
}

func TestExperimentationIsNotEnabledForRuleMatchIfRuleIndexIsNegative(t *testing.T) {
	flag := FeatureFlag{
		Key: "key",
		Rules: []FlagRule{
			{ID: "rule0", TrackEvents: true},
			{ID: "rule1", TrackEvents: false},
		},
	}
	reason := ldreason.NewEvalReasonRuleMatch(-1, "rule1")
	assert.False(t, asFlagEventProperties(flag).IsExperimentationEnabled(reason))
}

func TestExperimentationIsNotEnabledForRuleMatchIfRuleIndexIsTooHigh(t *testing.T) {
	flag := FeatureFlag{
		Key: "key",
		Rules: []FlagRule{
			{ID: "rule0", TrackEvents: true},
			{ID: "rule1", TrackEvents: false},
		},
	}
	reason := ldreason.NewEvalReasonRuleMatch(2, "rule1")
	assert.False(t, asFlagEventProperties(flag).IsExperimentationEnabled(reason))
}
