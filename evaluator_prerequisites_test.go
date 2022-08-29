package evaluation

import (
	"fmt"
	"testing"

	"github.com/launchdarkly/go-server-sdk-evaluation/v2/ldbuilders"

	"github.com/launchdarkly/go-sdk-common/v3/ldlog"
	"github.com/launchdarkly/go-sdk-common/v3/ldlogtest"
	"github.com/launchdarkly/go-sdk-common/v3/ldreason"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	m "github.com/launchdarkly/go-test-helpers/v3/matchers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type prereqEventSink struct {
	events []PrerequisiteFlagEvent
}

func (p *prereqEventSink) record(event PrerequisiteFlagEvent) {
	p.events = append(p.events, event)
}

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
	m.In(t).Assert(result, ResultDetailProps(1, offValue, ldreason.NewEvalReasonPrerequisiteFailed("feature1")))
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
	m.In(t).Assert(result, ResultDetailProps(1, offValue, ldreason.NewEvalReasonPrerequisiteFailed("feature1")))

	assert.Equal(t, 1, len(eventSink.events))
	e := eventSink.events[0]
	assert.Equal(t, f0.Key, e.TargetFlagKey)
	assert.Equal(t, flagTestContext, e.Context)
	assert.Equal(t, &f1, e.PrerequisiteFlag)
	m.In(t).Assert(e.PrerequisiteResult, ResultDetailProps(1, ldvalue.String("go"), ldreason.NewEvalReasonOff()))
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
	m.In(t).Assert(result, ResultDetailProps(1, offValue, ldreason.NewEvalReasonPrerequisiteFailed("feature1")))

	assert.Equal(t, 1, len(eventSink.events))
	e := eventSink.events[0]
	assert.Equal(t, f0.Key, e.TargetFlagKey)
	assert.Equal(t, flagTestContext, e.Context)
	assert.Equal(t, &f1, e.PrerequisiteFlag)
	m.In(t).Assert(e.PrerequisiteResult, ResultDetailProps(0, ldvalue.String("nogo"), ldreason.NewEvalReasonFallthrough()))
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
	m.In(t).Assert(result, ResultDetailProps(0, fallthroughValue, ldreason.NewEvalReasonFallthrough()))

	assert.Equal(t, 1, len(eventSink.events))
	e := eventSink.events[0]
	assert.Equal(t, f0.Key, e.TargetFlagKey)
	assert.Equal(t, flagTestContext, e.Context)
	assert.Equal(t, &f1, e.PrerequisiteFlag)
	m.In(t).Assert(e.PrerequisiteResult, ResultDetailProps(1, ldvalue.String("go"), ldreason.NewEvalReasonFallthrough()))
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
	m.In(t).Assert(result, ResultDetailProps(0, fallthroughValue, ldreason.NewEvalReasonFallthrough()))

	assert.Equal(t, 1, len(eventSink.events))
	e := eventSink.events[0]
	assert.Equal(t, f0.Key, e.TargetFlagKey)
	assert.Equal(t, flagTestContext, e.Context)
	assert.Equal(t, &f1, e.PrerequisiteFlag)
	m.In(t).Assert(e.PrerequisiteResult, ResultDetailProps(1, prereqVar1, ldreason.NewEvalReasonFallthrough()))
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
	m.In(t).Assert(result, ResultDetailProps(0, fallthroughValue, ldreason.NewEvalReasonFallthrough()))

	assert.Equal(t, 2, len(eventSink.events))
	// events are generated recursively, so the deepest level of prerequisite appears first

	e0 := eventSink.events[0]
	assert.Equal(t, f1.Key, e0.TargetFlagKey)
	assert.Equal(t, flagTestContext, e0.Context)
	assert.Equal(t, &f2, e0.PrerequisiteFlag)
	m.In(t).Assert(e0.PrerequisiteResult, ResultDetailProps(1, ldvalue.String("go"), ldreason.NewEvalReasonFallthrough()))

	e1 := eventSink.events[1]
	assert.Equal(t, f0.Key, e1.TargetFlagKey)
	assert.Equal(t, flagTestContext, e1.Context)
	assert.Equal(t, &f1, e1.PrerequisiteFlag)
	m.In(t).Assert(e1.PrerequisiteResult, ResultDetailProps(1, ldvalue.String("go"), ldreason.NewEvalReasonFallthrough()))
}

func TestPrerequisiteCycleDetection(t *testing.T) {
	for _, cycleGoesToOriginalFlag := range []bool{true, false} {
		t.Run(fmt.Sprintf("cycleGoesToOriginalFlag=%t", cycleGoesToOriginalFlag), func(t *testing.T) {
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
			cycleTargetKey := f1.Key
			if cycleGoesToOriginalFlag {
				cycleTargetKey = f0.Key
			}
			f2 := ldbuilders.NewFlagBuilder("feature2").
				On(true).
				AddPrerequisite(cycleTargetKey, 1). // deliberate error
				FallthroughVariation(1).
				Variations(ldvalue.String("nogo"), ldvalue.String("go")).
				Build()

			logCapture := ldlogtest.NewMockLog()
			evaluator := NewEvaluatorWithOptions(
				basicDataProvider().withStoredFlags(f0, f1, f2),
				EvaluatorOptionErrorLogger(logCapture.Loggers.ForLevel(ldlog.Error)),
			)

			result := evaluator.Evaluate(&f0, flagTestContext, FailOnAnyPrereqEvent(t))
			m.In(t).Assert(result, ResultDetailError(ldreason.EvalErrorMalformedFlag))

			// Note, we used FailOnAnyPrereqEvent because we would only generate a prerequisite event after
			// we *finish* evaluating a prerequisite-- and due to the cycle, none of these evaluations can
			// ever successfully finish.

			errorLines := logCapture.GetOutput(ldlog.Error)
			require.Len(t, errorLines, 1)
			assert.Regexp(t, `Invalid flag configuration.*prerequisite relationship.*circular reference`, errorLines[0])
		})
	}
}
