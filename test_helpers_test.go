package evaluation

import (
	"testing"

	m "github.com/launchdarkly/go-test-helpers/v2/matchers"
	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
)

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

// EvalDetailProps is a shortcut for matching all three fields of a successful evaluation result.
func EvalDetailProps(variationIndex int, value ldvalue.Value, reason ldreason.EvaluationReason) m.Matcher {
	return EvalDetailEquals(ldreason.NewEvaluationDetail(value, variationIndex, reason))
}
