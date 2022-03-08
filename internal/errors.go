package internal

import (
	"fmt"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldreason"
)

// These error types are used only internally to distinguish between reasons an evaluation might fail.
// They are surfaced only in terms of the EvaluationReason/ErrorKind types.

// When possible, we define these types as renames of a simple type like string or int, rather than as
// a struct. This is a minor optimization to take advantage of the fact that a simple type that implements
// an interface does not need to be allocated on the heap.

// EvalError is an internal interface for an error that should cause evaluation to fail.
type EvalError interface {
	error
	ErrorKind() ldreason.EvalErrorKind
}

// ErrorKindForError returns the appropriate ldreason.EvalErrorKind value for an error.
func ErrorKindForError(err error) ldreason.EvalErrorKind {
	if e, ok := err.(EvalError); ok {
		return e.ErrorKind()
	}
	return ldreason.EvalErrorException
}

// BadVariationError means a variation index was out of range. The integer value is the index.
type BadVariationError int

func (e BadVariationError) Error() string {
	return fmt.Sprintf("rule, fallthrough, or target referenced a nonexistent variation index %d", int(e))
}

func (e BadVariationError) ErrorKind() ldreason.EvalErrorKind { //nolint:revive
	return ldreason.EvalErrorMalformedFlag
}

// EmptyAttrRefError means an attribute reference in a clause was undefined
type EmptyAttrRefError struct{}

func (e EmptyAttrRefError) Error() string {
	return "rule clause did not specify an attribute"
}

func (e EmptyAttrRefError) ErrorKind() ldreason.EvalErrorKind { //nolint:revive
	return ldreason.EvalErrorMalformedFlag
}

// BadAttrRefError means an attribute reference in a clause was syntactically invalid. The string value is the
// attribute reference.
type BadAttrRefError string

func (e BadAttrRefError) Error() string {
	return fmt.Sprintf("invalid context attribute reference %q", string(e))
}

func (e BadAttrRefError) ErrorKind() ldreason.EvalErrorKind { //nolint:revive
	return ldreason.EvalErrorMalformedFlag
}

// EmptyRolloutError means a rollout or experiment had no variations.
type EmptyRolloutError struct{}

func (e EmptyRolloutError) Error() string {
	return "rollout or experiment with no variations"
}

func (e EmptyRolloutError) ErrorKind() ldreason.EvalErrorKind { //nolint:revive
	return ldreason.EvalErrorMalformedFlag
}

// CircularPrereqReferenceError means there was a cycle in prerequisites. The string value is the key of the
// prerequisite.
type CircularPrereqReferenceError string

func (e CircularPrereqReferenceError) Error() string {
	return fmt.Sprintf("prerequisite relationship to %q caused a circular reference;"+
		" this is probably a temporary condition due to an incomplete update", string(e))
}

func (e CircularPrereqReferenceError) ErrorKind() ldreason.EvalErrorKind { //nolint:revive
	return ldreason.EvalErrorMalformedFlag
}
