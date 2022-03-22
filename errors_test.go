package evaluation

import (
	"errors"
	"fmt"
	"testing"

	"github.com/launchdarkly/go-sdk-common/v3/ldreason"

	"github.com/stretchr/testify/assert"
)

func TestErrorKindForError(t *testing.T) {
	for _, err := range []error{
		badAttrRefError("x"),
		badVariationError(1),
		circularPrereqReferenceError("x"),
		emptyRolloutError{},
		malformedSegmentError{"x", nil},
	} {
		t.Run(fmt.Sprintf("%+v", err), func(t *testing.T) {
			assert.Equal(t, ldreason.EvalErrorMalformedFlag, errorKindForError(err))
		})
	}

	assert.Equal(t, ldreason.EvalErrorException, errorKindForError(errors.New("some other error")))
}
