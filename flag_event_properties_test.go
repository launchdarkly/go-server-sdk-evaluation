package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldtime"

	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldbuilders"

	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
)

func TestFlagEventPropertiesBasicProperties(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").Version(2).Build()
	props := FlagEventProperties(flag)
	assert.Equal(t, "key", props.GetKey())
	assert.Equal(t, 2, props.GetVersion())
}

func TestIsFullEventTrackingEnabled(t *testing.T) {
	flag1 := ldbuilders.NewFlagBuilder("key").Build()
	assert.False(t, FlagEventProperties(flag1).IsFullEventTrackingEnabled())

	flag2 := ldbuilders.NewFlagBuilder("key").TrackEvents(true).Build()
	assert.True(t, FlagEventProperties(flag2).IsFullEventTrackingEnabled())
}

func TestGetDebugEventsUntilDate(t *testing.T) {
	flag1 := ldbuilders.NewFlagBuilder("key").Build()
	assert.Equal(t, ldtime.UnixMillisecondTime(0), FlagEventProperties(flag1).GetDebugEventsUntilDate())

	date := ldtime.UnixMillisecondTime(100000)
	flag2 := ldbuilders.NewFlagBuilder("key").DebugEventsUntilDate(date).Build()
	assert.Equal(t, date, FlagEventProperties(flag2).GetDebugEventsUntilDate())
}

func TestIsExperimentDefaultsToFalse(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").Build()
	assert.False(t, FlagEventProperties(flag).IsExperimentationEnabled(ldreason.NewEvalReasonOff()))
}

func TestIsExperimentReturnsFalseForFallthroughIfTrackEventsFallthroughIsFalse(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").Build()
	assert.False(t, FlagEventProperties(flag).IsExperimentationEnabled(ldreason.NewEvalReasonFallthrough()))
}

func TestIsExperimentReturnsTrueForFallthroughIfTrackEventsFallthroughIsTrue(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").TrackEventsFallthrough(true).Build()
	assert.True(t, FlagEventProperties(flag).IsExperimentationEnabled(ldreason.NewEvalReasonFallthrough()))
}

func TestIsExperimentReturnsFalseForRuleMatchIfTrackEventsIsFalseForThatRule(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").
		AddRule(ldbuilders.NewRuleBuilder().ID("rule0").TrackEvents(true)).
		AddRule(ldbuilders.NewRuleBuilder().ID("rule1").TrackEvents(false)).
		Build()
	reason := ldreason.NewEvalReasonRuleMatch(1, "rule1")
	assert.False(t, FlagEventProperties(flag).IsExperimentationEnabled(reason))
}

func TestIsExperimentReturnsTrueForRuleMatchIfTrackEventsIsTrueForThatRule(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").
		AddRule(ldbuilders.NewRuleBuilder().ID("rule0").TrackEvents(true)).
		AddRule(ldbuilders.NewRuleBuilder().ID("rule1").TrackEvents(false)).
		Build()
	reason := ldreason.NewEvalReasonRuleMatch(0, "rule0")
	assert.True(t, FlagEventProperties(flag).IsExperimentationEnabled(reason))
}

func TestIsExperimentReturnsFalseForRuleMatchIfRuleIndexIsNegative(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").
		AddRule(ldbuilders.NewRuleBuilder().ID("rule0").TrackEvents(true)).
		AddRule(ldbuilders.NewRuleBuilder().ID("rule1").TrackEvents(false)).
		Build()
	reason := ldreason.NewEvalReasonRuleMatch(-1, "rule1")
	assert.False(t, FlagEventProperties(flag).IsExperimentationEnabled(reason))
}

func TestIsExperimentReturnsFalseForRuleMatchIfRuleIndexIsTooHigh(t *testing.T) {
	flag := ldbuilders.NewFlagBuilder("key").
		AddRule(ldbuilders.NewRuleBuilder().ID("rule0").TrackEvents(true)).
		AddRule(ldbuilders.NewRuleBuilder().ID("rule1").TrackEvents(false)).
		Build()
	reason := ldreason.NewEvalReasonRuleMatch(2, "rule1")
	assert.False(t, FlagEventProperties(flag).IsExperimentationEnabled(reason))
}
