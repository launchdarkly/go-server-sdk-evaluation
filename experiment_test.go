package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldbuilders"

	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
)

func TestIsExperimentDefaultsToFalse(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").Build()
	assert.False(t, IsExperimentationEnabled(flag, ldreason.NewEvalReasonOff()))
}

func TestIsExperimentReturnsFalseForFallthroughIfTrackEventsFallthroughIsFalse(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").Build()
	assert.False(t, IsExperimentationEnabled(flag, ldreason.NewEvalReasonFallthrough()))
}

func TestIsExperimentReturnsTrueForFallthroughIfTrackEventsFallthroughIsTrue(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").TrackEventsFallthrough(true).Build()
	assert.True(t, IsExperimentationEnabled(flag, ldreason.NewEvalReasonFallthrough()))
}

func TestIsExperimentReturnsFalseForRuleMatchIfTrackEventsIsFalseForThatRule(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").
		AddRule(ldbuilders.NewRuleBuilder().ID("rule0").TrackEvents(true)).
		AddRule(ldbuilders.NewRuleBuilder().ID("rule1").TrackEvents(false)).
		Build()
	reason := ldreason.NewEvalReasonRuleMatch(1, "rule1")
	assert.False(t, IsExperimentationEnabled(flag, reason))
}

func TestIsExperimentReturnsTrueForRuleMatchIfTrackEventsIsTrueForThatRule(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").
		AddRule(ldbuilders.NewRuleBuilder().ID("rule0").TrackEvents(true)).
		AddRule(ldbuilders.NewRuleBuilder().ID("rule1").TrackEvents(false)).
		Build()
	reason := ldreason.NewEvalReasonRuleMatch(0, "rule0")
	assert.True(t, IsExperimentationEnabled(flag, reason))
}

func TestIsExperimentReturnsFalseForRuleMatchIfRuleIndexIsNegative(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").
		AddRule(ldbuilders.NewRuleBuilder().ID("rule0").TrackEvents(true)).
		AddRule(ldbuilders.NewRuleBuilder().ID("rule1").TrackEvents(false)).
		Build()
	reason := ldreason.NewEvalReasonRuleMatch(-1, "rule1")
	assert.False(t, IsExperimentationEnabled(flag, reason))
}

func TestIsExperimentReturnsFalseForRuleMatchIfRuleIndexIsTooHigh(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").
		AddRule(ldbuilders.NewRuleBuilder().ID("rule0").TrackEvents(true)).
		AddRule(ldbuilders.NewRuleBuilder().ID("rule1").TrackEvents(false)).
		Build()
	reason := ldreason.NewEvalReasonRuleMatch(2, "rule1")
	assert.False(t, IsExperimentationEnabled(flag, reason))
}
