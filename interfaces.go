package evaluation

import (
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldmodel"
)

// Evaluator is the engine for evaluating feature flags.
type Evaluator interface {
	// Evaluate evaluates a feature flag for the specified user.
	//
	// The flag is passed by reference only for efficiency; the evaluator will never modify any flag
	// properties. Passing a nil flag will result in a panic.
	//
	// The evaluator does not know anything about analytics events; generating any appropriate analytics
	// events is the responsibility of the caller, who can also provide a callback in prerequisiteFlagEventRecorder
	// to be notified if any additional evaluations were done due to prerequisites. The prerequisiteFlagEventRecorder
	// parameter can be nil if you do not need to track prerequisite evaluations.
	Evaluate(
		flag *ldmodel.FeatureFlag,
		user lduser.User,
		prerequisiteFlagEventRecorder PrerequisiteFlagEventRecorder,
	) ldreason.EvaluationDetail
}

// PrerequisiteFlagEventRecorder is a function that Evaluator.Evaluate() will call to record the
// result of a prerequisite flag evaluation.
type PrerequisiteFlagEventRecorder func(PrerequisiteFlagEvent)

// PrerequisiteFlagEvent is the parameter data passed to PrerequisiteFlagEventRecorder.
type PrerequisiteFlagEvent struct {
	// TargetFlagKey is the key of the feature flag that had a prerequisite.
	TargetFlagKey string
	// PrerequisiteFlag is the full configuration of the prerequisite flag. This is passed by
	// reference for efficiency only, and will never be nil; the PrerequisiteFlagEventRecorder
	// must not modify the flag's properties.
	PrerequisiteFlag *ldmodel.FeatureFlag
	// PrerequisiteResult is the result of evaluating the prerequisite flag.
	PrerequisiteResult ldreason.EvaluationDetail
}

// DataProvider is an abstraction for querying feature flags and user segments from a data store.
// The caller provides an implementation of this interface to NewEvaluator.
//
// Flags and segments are returned by reference for efficiency only (on the assumption that the
// caller already has these objects in memory); the evaluator will never modify their properties.
type DataProvider interface {
	// GetFeatureFlag attempts to retrieve a feature flag from the data store by key.
	//
	// The evaluator calls this method if a flag contains a prerequisite condition referencing
	// another flag.
	//
	// The method returns nil if the flag was not found. The DataProvider should treat any deleted
	// flag as "not found" even if the data store contains a deleted flag placeholder for it.
	GetFeatureFlag(key string) *ldmodel.FeatureFlag
	// GetSegment attempts to retrieve a user segment from the data store by key.
	//
	// The evaluator calls this method if a clause in a flag rule uses the OperatorSegmentMatch
	// test.
	//
	// The method returns nil if the segment was not found. The DataProvider should treat any deleted
	// segment as "not found" even if the data store contains a deleted segment placeholder for it.
	GetSegment(key string) *ldmodel.Segment
}
