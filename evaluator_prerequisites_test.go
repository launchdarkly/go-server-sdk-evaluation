package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldbuilders"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldlog"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldlogtest"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlagReturnsOffVariationIfPrerequisiteIsNotFound(t *testing.T) {
	f0 := ldbuilders.NewFlagBuilder("feature0").
		On(true).
		OffVariation(1).
		AddPrerequisite("feature1", 1).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()
	evaluator := NewEvaluator(basicDataProvider().withNonexistentFlag("feature1"))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(&f0, flagTestContext, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(offValue, 1, ldreason.NewEvalReasonPrerequisiteFailed("feature1")), result)
	assert.Equal(t, 0, len(eventSink.events))
}

func TestFlagReturnsOffVariationAndEventIfPrerequisiteIsOff(t *testing.T) {
	f0 := ldbuilders.NewFlagBuilder("feature0").
		On(true).
		OffVariation(1).
		AddPrerequisite("feature1", 1).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()
	f1 := ldbuilders.NewFlagBuilder("feature1").
		On(false).
		OffVariation(1).
		// note that even though it returns the desired variation, it is still off and therefore not a match
		FallthroughVariation(0).
		Variations(ldvalue.String("nogo"), ldvalue.String("go")).
		Build()
	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(f1))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(&f0, flagTestContext, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(offValue, 1, ldreason.NewEvalReasonPrerequisiteFailed("feature1")), result)

	assert.Equal(t, 1, len(eventSink.events))
	e := eventSink.events[0]
	assert.Equal(t, f0.Key, e.TargetFlagKey)
	assert.Equal(t, flagTestContext, e.Context)
	assert.Equal(t, &f1, e.PrerequisiteFlag)
	assert.Equal(t, ldreason.NewEvaluationDetail(ldvalue.String("go"), 1, ldreason.NewEvalReasonOff()), e.PrerequisiteResult)
}

func TestFlagReturnsOffVariationAndEventIfPrerequisiteIsNotMet(t *testing.T) {
	f0 := ldbuilders.NewFlagBuilder("feature0").
		On(true).
		OffVariation(1).
		AddPrerequisite("feature1", 1).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()
	f1 := ldbuilders.NewFlagBuilder("feature1").
		On(true).
		FallthroughVariation(0).
		Variations(ldvalue.String("nogo"), ldvalue.String("go")).
		Build()
	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(f1))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(&f0, flagTestContext, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(offValue, 1, ldreason.NewEvalReasonPrerequisiteFailed("feature1")), result)

	assert.Equal(t, 1, len(eventSink.events))
	e := eventSink.events[0]
	assert.Equal(t, f0.Key, e.TargetFlagKey)
	assert.Equal(t, flagTestContext, e.Context)
	assert.Equal(t, &f1, e.PrerequisiteFlag)
	assert.Equal(t, ldreason.NewEvaluationDetail(ldvalue.String("nogo"), 0, ldreason.NewEvalReasonFallthrough()), e.PrerequisiteResult)
}

func TestFlagReturnsFallthroughVariationAndEventIfPrerequisiteIsMetAndThereAreNoRules(t *testing.T) {
	f0 := ldbuilders.NewFlagBuilder("feature0").
		On(true).
		OffVariation(1).
		AddPrerequisite("feature1", 1).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()
	f1 := ldbuilders.NewFlagBuilder("feature1").
		On(true).
		FallthroughVariation(1). // this 1 matches the 1 in f0's prerequisites
		Variations(ldvalue.String("nogo"), ldvalue.String("go")).
		Build()
	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(f1))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(&f0, flagTestContext, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(fallthroughValue, 0, ldreason.NewEvalReasonFallthrough()), result)

	assert.Equal(t, 1, len(eventSink.events))
	e := eventSink.events[0]
	assert.Equal(t, f0.Key, e.TargetFlagKey)
	assert.Equal(t, flagTestContext, e.Context)
	assert.Equal(t, &f1, e.PrerequisiteFlag)
	assert.Equal(t, ldreason.NewEvaluationDetail(ldvalue.String("go"), 1, ldreason.NewEvalReasonFallthrough()), e.PrerequisiteResult)
}

func TestPrerequisiteCanMatchWithNonScalarValue(t *testing.T) {
	f0 := ldbuilders.NewFlagBuilder("feature0").
		On(true).
		OffVariation(1).
		AddPrerequisite("feature1", 1).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()
	prereqVar0 := ldvalue.ArrayOf(ldvalue.String("000"))
	prereqVar1 := ldvalue.ArrayOf(ldvalue.String("001"))
	f1 := ldbuilders.NewFlagBuilder("feature1").
		On(true).
		FallthroughVariation(1). // this 1 matches the 1 in f0's prerequisites
		Variations(prereqVar0, prereqVar1).
		Build()
	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(f1))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(&f0, flagTestContext, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(fallthroughValue, 0, ldreason.NewEvalReasonFallthrough()), result)

	assert.Equal(t, 1, len(eventSink.events))
	e := eventSink.events[0]
	assert.Equal(t, f0.Key, e.TargetFlagKey)
	assert.Equal(t, flagTestContext, e.Context)
	assert.Equal(t, &f1, e.PrerequisiteFlag)
	assert.Equal(t, ldreason.NewEvaluationDetail(prereqVar1, 1, ldreason.NewEvalReasonFallthrough()), e.PrerequisiteResult)
}

func TestMultipleLevelsOfPrerequisiteProduceMultipleEvents(t *testing.T) {
	f0 := ldbuilders.NewFlagBuilder("feature0").
		On(true).
		OffVariation(1).
		AddPrerequisite("feature1", 1).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()
	f1 := ldbuilders.NewFlagBuilder("feature1").
		On(true).
		AddPrerequisite("feature2", 1).
		FallthroughVariation(1). // this 1 matches the 1 in f0's prerequisites
		Variations(ldvalue.String("nogo"), ldvalue.String("go")).
		Build()
	f2 := ldbuilders.NewFlagBuilder("feature2").
		On(true).
		FallthroughVariation(1). // this 1 matches the 1 in f1's prerequisites
		Variations(ldvalue.String("nogo"), ldvalue.String("go")).
		Build()
	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(f1, f2))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(&f0, flagTestContext, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetail(fallthroughValue, 0, ldreason.NewEvalReasonFallthrough()), result)

	assert.Equal(t, 2, len(eventSink.events))
	// events are generated recursively, so the deepest level of prerequisite appears first

	e0 := eventSink.events[0]
	assert.Equal(t, f1.Key, e0.TargetFlagKey)
	assert.Equal(t, flagTestContext, e0.Context)
	assert.Equal(t, &f2, e0.PrerequisiteFlag)
	assert.Equal(t, ldreason.NewEvaluationDetail(ldvalue.String("go"), 1, ldreason.NewEvalReasonFallthrough()), e0.PrerequisiteResult)

	e1 := eventSink.events[1]
	assert.Equal(t, f0.Key, e1.TargetFlagKey)
	assert.Equal(t, flagTestContext, e1.Context)
	assert.Equal(t, &f1, e1.PrerequisiteFlag)
	assert.Equal(t, ldreason.NewEvaluationDetail(ldvalue.String("go"), 1, ldreason.NewEvalReasonFallthrough()), e1.PrerequisiteResult)
}

func TestPrerequisiteCycleLeadingBackToOriginalFlagReturnsErrorAndDoesNotOverflow(t *testing.T) {
	f0 := ldbuilders.NewFlagBuilder("feature0").
		On(true).
		OffVariation(1).
		AddPrerequisite("feature1", 1).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()
	f1 := ldbuilders.NewFlagBuilder("feature1").
		On(true).
		AddPrerequisite("feature2", 1).
		FallthroughVariation(1). // this 1 matches the 1 in f0's prerequisites
		Variations(ldvalue.String("nogo"), ldvalue.String("go")).
		Build()
	f2 := ldbuilders.NewFlagBuilder("feature2").
		On(true).
		AddPrerequisite("feature0", 1). // deliberate error: this points back to the original flag
		FallthroughVariation(1).        // this 1 matches the 1 in f1's prerequisites
		Variations(ldvalue.String("nogo"), ldvalue.String("go")).
		Build()
	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(f0, f1, f2))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(&f0, flagTestContext, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)

	assert.Len(t, eventSink.events, 0)
}

func TestPrerequisiteCycleNotInvolvingOriginalFlagReturnsErrorAndDoesNotOverflow(t *testing.T) {
	f0 := ldbuilders.NewFlagBuilder("feature0").
		On(true).
		OffVariation(1).
		AddPrerequisite("feature1", 1).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()
	f1 := ldbuilders.NewFlagBuilder("feature1").
		On(true).
		AddPrerequisite("feature2", 1).
		FallthroughVariation(1). // this 1 matches the 1 in f0's prerequisites
		Variations(ldvalue.String("nogo"), ldvalue.String("go")).
		Build()
	f2 := ldbuilders.NewFlagBuilder("feature2").
		On(true).
		AddPrerequisite("feature1", 1). // deliberate error: this points back to a flag we've already visited
		FallthroughVariation(1).        // this 1 matches the 1 in f1's prerequisites
		Variations(ldvalue.String("nogo"), ldvalue.String("go")).
		Build()
	evaluator := NewEvaluator(basicDataProvider().withStoredFlags(f0, f1, f2))

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(&f0, flagTestContext, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)

	assert.Len(t, eventSink.events, 0)
}

func TestPrerequisiteCycleCausesErrorToBeLogged(t *testing.T) {
	f0 := ldbuilders.NewFlagBuilder("feature0").
		On(true).
		OffVariation(1).
		AddPrerequisite("feature1", 1).
		FallthroughVariation(0).
		Variations(fallthroughValue, offValue, onValue).
		Build()
	f1 := ldbuilders.NewFlagBuilder("feature1").
		On(true).
		AddPrerequisite("feature0", 1). // deliberate error
		FallthroughVariation(1).        // this 1 matches the 1 in f0's prerequisites
		Variations(ldvalue.String("nogo"), ldvalue.String("go")).
		Build()
	logCapture := ldlogtest.NewMockLog()
	evaluator := NewEvaluatorWithOptions(
		basicDataProvider().withStoredFlags(f0, f1),
		EvaluatorOptionErrorLogger(logCapture.Loggers.ForLevel(ldlog.Error)),
	)

	eventSink := prereqEventSink{}
	result := evaluator.Evaluate(&f0, flagTestContext, eventSink.record)
	assert.Equal(t, ldreason.NewEvaluationDetailForError(ldreason.EvalErrorMalformedFlag, ldvalue.Null()), result)

	assert.Len(t, eventSink.events, 0)

	errorLines := logCapture.GetOutput(ldlog.Error)
	require.Len(t, errorLines, 1)
	assert.Regexp(t, `"feature1".*prerequisite.*"feature0".*circular reference`, errorLines[0])
}
