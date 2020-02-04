package evaluation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

func TestFlagReturnsOffVariationIfPrerequisiteIsNotFound(t *testing.T) {
	f0 := FeatureFlag{
		Key:           "feature0",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature1", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(0)},
		Variations:    []ldvalue.Value{fallthroughValue, offValue, onValue},
	}
	evaluator := NewEvaluator(basicDataProvider().withNonexistentFlag("feature1"))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(f0, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(offValue, 1, ldreason.NewEvalReasonPrerequisiteFailed("feature1")), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsOffVariationAndEventIfPrerequisiteIsOff(t *testing.T) {
	f0 := FeatureFlag{
		Key:           "feature0",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature1", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(0)},
		Variations:    []ldvalue.Value{fallthroughValue, offValue, onValue},
		Version:       1,
	}
	f1 := FeatureFlag{
		Key:          "feature1",
		On:           false,
		OffVariation: intPtr(1),
		// note that even though it returns the desired variation, it is still off and therefore not a match
		Fallthrough: VariationOrRollout{Variation: intPtr(0)},
		Variations:  []ldvalue.Value{ldvalue.String("nogo"), ldvalue.String("go")},
		Version:     2,
	}
	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(f1))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(f0, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(offValue, 1, ldreason.NewEvalReasonPrerequisiteFailed("feature1")), result)

	assert.Equal(t, 1, len(eventSink.events))
	e := eventSink.events[0]
	assert.Equal(t, f0.Key, e.TargetFlagKey)
	assert.Equal(t, f1, e.PrerequisiteFlag)
	assert.Equal(t, ldreason.NewEvaluationDetail(ldvalue.String("go"), 1, ldreason.NewEvalReasonOff()), e.PrerequisiteResult)
}

func TestFlagReturnsOffVariationAndEventIfPrerequisiteIsNotMet(t *testing.T) {
	f0 := FeatureFlag{
		Key:           "feature0",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature1", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(0)},
		Variations:    []ldvalue.Value{fallthroughValue, offValue, onValue},
		Version:       1,
	}
	f1 := FeatureFlag{
		Key:          "feature1",
		On:           true,
		OffVariation: intPtr(1),
		Fallthrough:  VariationOrRollout{Variation: intPtr(0)},
		Variations:   []ldvalue.Value{ldvalue.String("nogo"), ldvalue.String("go")},
		Version:      2,
	}
	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(f1))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(f0, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(offValue, 1, ldreason.NewEvalReasonPrerequisiteFailed("feature1")), result)

	assert.Equal(t, 1, len(eventSink.events))
	e := eventSink.events[0]
	assert.Equal(t, f0.Key, e.TargetFlagKey)
	assert.Equal(t, f1, e.PrerequisiteFlag)
	assert.Equal(t, ldreason.NewEvaluationDetail(ldvalue.String("nogo"), 0, ldreason.NewEvalReasonFallthrough()), e.PrerequisiteResult)
}

func TestFlagReturnsFallthroughVariationAndEventIfPrerequisiteIsMetAndThereAreNoRules(t *testing.T) {
	f0 := FeatureFlag{
		Key:           "feature0",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature1", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(0)},
		Variations:    []ldvalue.Value{fallthroughValue, offValue, onValue},
		Version:       1,
	}
	f1 := FeatureFlag{
		Key:          "feature1",
		On:           true,
		OffVariation: intPtr(1),
		Fallthrough:  VariationOrRollout{Variation: intPtr(1)}, // this 1 matches the 1 in the prerequisites array
		Variations:   []ldvalue.Value{ldvalue.String("nogo"), ldvalue.String("go")},
		Version:      2,
	}
	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(f1))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(f0, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(fallthroughValue, 0, ldreason.NewEvalReasonFallthrough()), result)

	assert.Equal(t, 1, len(eventSink.events))
	e := eventSink.events[0]
	assert.Equal(t, f0.Key, e.TargetFlagKey)
	assert.Equal(t, f1, e.PrerequisiteFlag)
	assert.Equal(t, ldreason.NewEvaluationDetail(ldvalue.String("go"), 1, ldreason.NewEvalReasonFallthrough()), e.PrerequisiteResult)
}

func TestPrerequisiteCanMatchWithNonScalarValue(t *testing.T) {
	f0 := FeatureFlag{
		Key:           "feature0",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature1", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(0)},
		Variations:    []ldvalue.Value{fallthroughValue, offValue, onValue},
		Version:       1,
	}
	prereqVar0 := ldvalue.ArrayOf(ldvalue.String("000"))
	prereqVar1 := ldvalue.ArrayOf(ldvalue.String("001"))
	f1 := FeatureFlag{
		Key:          "feature1",
		On:           true,
		OffVariation: intPtr(1),
		Fallthrough:  VariationOrRollout{Variation: intPtr(1)}, // this 1 matches the 1 in the prerequisites array
		Variations:   []ldvalue.Value{prereqVar0, prereqVar1},
		Version:      2,
	}
	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(f1))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(f0, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(fallthroughValue, 0, ldreason.NewEvalReasonFallthrough()), result)

	assert.Equal(t, 1, len(eventSink.events))
	e := eventSink.events[0]
	assert.Equal(t, f0.Key, e.TargetFlagKey)
	assert.Equal(t, f1, e.PrerequisiteFlag)
	assert.Equal(t, ldreason.NewEvaluationDetail(prereqVar1, 1, ldreason.NewEvalReasonFallthrough()), e.PrerequisiteResult)
}

func TestMultipleLevelsOfPrerequisiteProduceMultipleEvents(t *testing.T) {
	f0 := FeatureFlag{
		Key:           "feature0",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature1", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(0)},
		Variations:    []ldvalue.Value{fallthroughValue, offValue, onValue},
		Version:       1,
	}
	f1 := FeatureFlag{
		Key:           "feature1",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature2", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(1)}, // this 1 matches the 1 in the prerequisites array
		Variations:    []ldvalue.Value{ldvalue.String("nogo"), ldvalue.String("go")},
		Version:       2,
	}
	f2 := FeatureFlag{
		Key:         "feature2",
		On:          true,
		Fallthrough: VariationOrRollout{Variation: intPtr(1)},
		Variations:  []ldvalue.Value{ldvalue.String("nogo"), ldvalue.String("go")},
		Version:     3,
	}
	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(f1, f2))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(f0, flagUser, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(fallthroughValue, 0, ldreason.NewEvalReasonFallthrough()), result)

	assert.Equal(t, 2, len(eventSink.events))
	// events are generated recursively, so the deepest level of prerequisite appears first

	e0 := eventSink.events[0]
	assert.Equal(t, f1.Key, e0.TargetFlagKey)
	assert.Equal(t, f2, e0.PrerequisiteFlag)
	assert.Equal(t, ldreason.NewEvaluationDetail(ldvalue.String("go"), 1, ldreason.NewEvalReasonFallthrough()), e0.PrerequisiteResult)

	e1 := eventSink.events[1]
	assert.Equal(t, f0.Key, e1.TargetFlagKey)
	assert.Equal(t, f1, e1.PrerequisiteFlag)
	assert.Equal(t, ldreason.NewEvaluationDetail(ldvalue.String("go"), 1, ldreason.NewEvalReasonFallthrough()), e1.PrerequisiteResult)
}
