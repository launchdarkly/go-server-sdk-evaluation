package evaluation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
)

func TestIsExperimentDefaultsToFalse(t *testing.T) {
	flag := FeatureFlag{}
	assert.False(t, IsExperimentationEnabled(flag, ldreason.NewEvalReasonOff()))
}

func TestIsExperimentReturnsFalseForFallthroughIfTrackEventsFallthroughIsFalse(t *testing.T) {
	flag := FeatureFlag{}
	assert.False(t, IsExperimentationEnabled(flag, ldreason.NewEvalReasonFallthrough()))
}

func TestIsExperimentReturnsTrueForFallthroughIfTrackEventsFallthroughIsTrue(t *testing.T) {
	flag := FeatureFlag{TrackEventsFallthrough: true}
	assert.True(t, IsExperimentationEnabled(flag, ldreason.NewEvalReasonFallthrough()))
}

func TestIsExperimentReturnsFalseForRuleMatchIfTrackEventsIsFalseForThatRule(t *testing.T) {
	flag := FeatureFlag{
		Rules: []FlagRule{
			FlagRule{ID: "rule0", TrackEvents: true},
			FlagRule{ID: "rule1", TrackEvents: false},
		},
	}
	reason := ldreason.NewEvalReasonRuleMatch(1, "rule1")
	assert.False(t, IsExperimentationEnabled(flag, reason))
}

func TestIsExperimentReturnsTrueForRuleMatchIfTrackEventsIsTrueForThatRule(t *testing.T) {
	flag := FeatureFlag{
		Rules: []FlagRule{
			FlagRule{ID: "rule0", TrackEvents: true},
			FlagRule{ID: "rule1", TrackEvents: false},
		},
	}
	reason := ldreason.NewEvalReasonRuleMatch(0, "rule0")
	assert.True(t, IsExperimentationEnabled(flag, reason))
}

func TestIsExperimentReturnsFalseForRuleMatchIfRuleIndexIsNegative(t *testing.T) {
	flag := FeatureFlag{
		Rules: []FlagRule{
			FlagRule{ID: "rule0", TrackEvents: true},
			FlagRule{ID: "rule1", TrackEvents: false},
		},
	}
	reason := ldreason.NewEvalReasonRuleMatch(-1, "rule1")
	assert.False(t, IsExperimentationEnabled(flag, reason))
}

func TestIsExperimentReturnsFalseForRuleMatchIfRuleIndexIsTooHigh(t *testing.T) {
	flag := FeatureFlag{
		Rules: []FlagRule{
			FlagRule{ID: "rule0", TrackEvents: true},
			FlagRule{ID: "rule1", TrackEvents: false},
		},
	}
	reason := ldreason.NewEvalReasonRuleMatch(2, "rule1")
	assert.False(t, IsExperimentationEnabled(flag, reason))
}
