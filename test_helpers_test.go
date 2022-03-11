package evaluation

import (
	"testing"

	m "github.com/launchdarkly/go-test-helpers/v2/matchers"
	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
)

// FailOnAnyPrereqEvent can be used as the prerequisiteFlagEventRecorder parameter to Evaluator.Evaluate()
// in any test where we do not expect any prerequisite events to be generated. It causes an automaitc test
// failure if it receives any events.
func FailOnAnyPrereqEvent(t *testing.T) func(PrerequisiteFlagEvent) {
	return func(e PrerequisiteFlagEvent) {
		assert.Fail(t, "did not expect any prerequisite events", "got event: %+v", e)
	}
}

// EvalDetailEquals is a custom matcher that works the same as m.Equals, but produces better failure
// output by treating the value as a JSON object rather than a struct.
func EvalDetailEquals(expected ldreason.EvaluationDetail) m.Matcher {
	return m.JSONEqual(expected)
}

// EvalDetailProps is a shortcut for matching all three fields of an EvaluationDetail.
func EvalDetailProps(variationIndex int, value ldvalue.Value, reason ldreason.EvaluationReason) m.Matcher {
	return EvalDetailEquals(ldreason.NewEvaluationDetail(value, variationIndex, reason))
}

// EvalDetailError is a shortcut for matching the fields of an EvaluationDetail for a failed evaluation.
func EvalDetailError(errorKind ldreason.EvalErrorKind) m.Matcher {
	return EvalDetailEquals(ldreason.NewEvaluationDetailForError(errorKind, ldvalue.Null()))
}

// EvalResultDetail is a shortcut for matching all three fields of a successful evaluation result.
func ResultDetailProps(variationIndex int, value ldvalue.Value, reason ldreason.EvaluationReason) m.Matcher {
	return ResultDetail().Should(EvalDetailProps(variationIndex, value, reason))
}

// EvalResultDetail is a shortcut for matching a Result for a failed evaluation.
func ResultDetailError(errorKind ldreason.EvalErrorKind) m.Matcher {
	return ResultDetail().Should(EvalDetailError(errorKind))
}

func ResultDetail() m.MatcherTransform {
	return m.Transform("result detail", func(value interface{}) (interface{}, error) { return value.(Result).Detail, nil }).
		EnsureInputValueType(Result{})
}
