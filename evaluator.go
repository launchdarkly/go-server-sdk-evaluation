package evaluation

import (
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/internal"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldmodel"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldlog"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
)

type evaluator struct {
	dataProvider       DataProvider
	bigSegmentProvider BigSegmentProvider
	errorLogger        ldlog.BaseLogger
}

// NewEvaluator creates an Evaluator, specifying a DataProvider that it will use if it needs to
// query additional feature flags or user segments during an evaluation.
//
// To support big segments, you must use NewEvaluatorWithOptions and EvaluatorOptionBigSegmentProvider.
func NewEvaluator(dataProvider DataProvider) Evaluator {
	return NewEvaluatorWithOptions(dataProvider)
}

// NewEvaluatorWithOptions creates an Evaluator, specifying a DataProvider that it will use if it
// needs to query additional feature flags or user segments during an evaluation, and also
// any number of EvaluatorOption modifiers.
func NewEvaluatorWithOptions(dataProvider DataProvider, options ...EvaluatorOption) Evaluator {
	e := &evaluator{
		dataProvider: dataProvider,
	}
	for _, o := range options {
		if o != nil {
			o.apply(e)
		}
	}
	return e
}

// Used internally to hold the parameters of an evaluation, to avoid repetitive parameter passing.
// Its methods use a pointer receiver for efficiency, even though it is allocated on the stack and
// its fields are never modified.
type evaluationScope struct {
	owner                         *evaluator
	flag                          *ldmodel.FeatureFlag
	context                       ldcontext.Context
	prerequisiteFlagEventRecorder PrerequisiteFlagEventRecorder
	// These bigSegments properties start out unset, and will be set only once during an
	// evaluation the first time we query a big segment, if any.
	bigSegmentsReferenced bool
	bigSegmentsMembership BigSegmentMembership
	bigSegmentsStatus     ldreason.BigSegmentsStatus
}

// Implementation of the Evaluator interface.
func (e *evaluator) Evaluate(
	flag *ldmodel.FeatureFlag,
	context ldcontext.Context,
	prerequisiteFlagEventRecorder PrerequisiteFlagEventRecorder,
) ldreason.EvaluationDetail {
	es := evaluationScope{
		owner:                         e,
		flag:                          flag,
		context:                       context,
		prerequisiteFlagEventRecorder: prerequisiteFlagEventRecorder,
	}

	// Preallocate some space for prerequisiteChain on the stack. We can get up to that many levels
	// of nested prerequisites before appending to the slice will cause a heap allocation.
	prerequisiteChain := make([]string, 0, 20)

	result, _ := es.evaluate(prerequisiteChain)
	if es.bigSegmentsReferenced {
		result.Reason = ldreason.NewEvalReasonFromReasonWithBigSegmentsStatus(result.Reason,
			es.bigSegmentsStatus)
	}
	return result
}

// Entry point for evaluating a flag which could be either the original flag or a prerequisite.
// The second return value is normally true. If it is false, it means we should immediately
// terminate the whole current stack of evaluations and not do any more checking or recursing.
func (es *evaluationScope) evaluate(prerequisiteChain []string) (ldreason.EvaluationDetail, bool) {
	if !es.flag.On {
		return es.getOffValue(ldreason.NewEvalReasonOff()), true
	}

	// Note that all of our internal methods operate on pointers (*User, *FeatureFlag, *Clause, etc.);
	// this is done to avoid the overhead of repeatedly copying these structs by value. We know that
	// the pointers cannot be nil, since the entry point is always Evaluate which does receive its
	// parameters by value; mutability is not a concern, since User is immutable and the evaluation
	// code will never modify anything in the data model. Taking the address of these structs will not
	// cause heap escaping because we are never *returning* pointers (and never passing them to
	// external code such as prerequisiteFlagEventRecorder).

	prereqErrorReason, ok := es.checkPrerequisites(prerequisiteChain)
	if !ok {
		// Is this an actual error, like a malformed flag? Then return an error with default value.
		if prereqErrorReason.GetKind() == ldreason.EvalReasonError {
			return ldreason.NewEvaluationDetailForError(prereqErrorReason.GetErrorKind(), ldvalue.Null()), false
		}
		// No, it's presumably just "prerequisite failed", which gets the off value.
		return es.getOffValue(prereqErrorReason), true
	}

	key := es.context.Key()

	// Check to see if targets match
	for _, target := range es.flag.Targets {
		// Note, taking address of range variable here is OK because it's not used outside the loop
		if ldmodel.TargetContainsKey(&target, key) { //nolint:gosec // see comment above
			return es.getVariation(target.Variation, ldreason.NewEvalReasonTargetMatch()), true
		}
	}

	// Now walk through the rules and see if any match
	for ruleIndex, rule := range es.flag.Rules {
		// Note, taking address of range variable here is OK because it's not used outside the loop
		match, err := es.ruleMatchesUser(&rule) //nolint:gosec // see comment above
		if err != nil {
			es.logEvaluationError(err)
			return ldreason.NewEvaluationDetailForError(internal.ErrorKindForError(err), ldvalue.Null()), false
		}
		if match {
			reason := ldreason.NewEvalReasonRuleMatch(ruleIndex, rule.ID)
			return es.getValueForVariationOrRollout(rule.VariationOrRollout, reason), true
		}
	}

	return es.getValueForVariationOrRollout(es.flag.Fallthrough, ldreason.NewEvalReasonFallthrough()), true
}

// Do a nested evaluation for a prerequisite of the current scope's flag. The second return value is
// normally true; it is false only in the case where we've detected a circular reference, in which
// case we want the entire evaluation to fail with a MalformedFlag error.
func (es *evaluationScope) evaluatePrerequisite(
	prereqFlag *ldmodel.FeatureFlag,
	prerequisiteChain []string,
) (ldreason.EvaluationDetail, bool) {
	for _, p := range prerequisiteChain {
		if prereqFlag.Key == p {
			err := internal.CircularPrereqReferenceError(prereqFlag.Key)
			es.logEvaluationError(err)
			return ldreason.EvaluationDetail{}, false
		}
	}
	subScope := *es
	subScope.flag = prereqFlag
	result, ok := subScope.evaluate(prerequisiteChain)
	if !es.bigSegmentsReferenced && subScope.bigSegmentsReferenced {
		es.bigSegmentsReferenced = true
		es.bigSegmentsStatus = subScope.bigSegmentsStatus
	}
	return result, ok
}

// Returns an empty reason if all prerequisites are OK, otherwise constructs an error reason that describes the failure
func (es *evaluationScope) checkPrerequisites(prerequisiteChain []string) (ldreason.EvaluationReason, bool) {
	if len(es.flag.Prerequisites) == 0 {
		return ldreason.EvaluationReason{}, true
	}

	prerequisiteChain = append(prerequisiteChain, es.flag.Key)
	// Note that the change to prerequisiteChain does not persist after returning from this method.
	// That introduces a potential edge-case inefficiency with deeply nested prerequisites: if the
	// original slice had a capacity of 20, and then the 20th prerequisite has 5 prerequisites of
	// its own, when checkPrerequisites is called for each of those it will end up hitting the
	// capacity of the slice each time and allocating a new backing array each time. The way
	// around that would be to pass a *pointer* to the slice, so the backing array would be
	// retained. However, doing so appears to defeat Go's escape analysis and cause heap escaping
	// of the slice every time, which would be worse in more typical use cases.

	for _, prereq := range es.flag.Prerequisites {
		prereqFeatureFlag := es.owner.dataProvider.GetFeatureFlag(prereq.Key)
		if prereqFeatureFlag == nil {
			return ldreason.NewEvalReasonPrerequisiteFailed(prereq.Key), false
		}
		prereqOK := true

		prereqResult, prereqValid := es.evaluatePrerequisite(prereqFeatureFlag, prerequisiteChain)
		if !prereqValid {
			// In this case we want to immediately exit with an error and not check any more prereqs
			return ldreason.NewEvalReasonError(ldreason.EvalErrorMalformedFlag), false
		}
		if !prereqFeatureFlag.On || prereqResult.IsDefaultValue() ||
			prereqResult.VariationIndex.IntValue() != prereq.Variation {
			// Note that if the prerequisite flag is off, we don't consider it a match no matter what its
			// off variation was. But we still need to evaluate it in order to generate an event.
			prereqOK = false
		}

		if es.prerequisiteFlagEventRecorder != nil {
			event := PrerequisiteFlagEvent{es.flag.Key, es.context, prereqFeatureFlag, prereqResult}
			es.prerequisiteFlagEventRecorder(event)
		}

		if !prereqOK {
			return ldreason.NewEvalReasonPrerequisiteFailed(prereq.Key), false
		}
	}
	return ldreason.EvaluationReason{}, true
}

func (es *evaluationScope) getVariation(index int, reason ldreason.EvaluationReason) ldreason.EvaluationDetail {
	if index < 0 || index >= len(es.flag.Variations) {
		err := internal.BadVariationError(index)
		es.logEvaluationError(err)
		return ldreason.NewEvaluationDetailForError(err.ErrorKind(), ldvalue.Null())
	}
	return ldreason.NewEvaluationDetail(es.flag.Variations[index], index, reason)
}

func (es *evaluationScope) getOffValue(reason ldreason.EvaluationReason) ldreason.EvaluationDetail {
	if !es.flag.OffVariation.IsDefined() {
		return ldreason.EvaluationDetail{Reason: reason}
	}
	return es.getVariation(es.flag.OffVariation.IntValue(), reason)
}

func (es *evaluationScope) getValueForVariationOrRollout(
	vr ldmodel.VariationOrRollout,
	reason ldreason.EvaluationReason,
) ldreason.EvaluationDetail {
	index, inExperiment, err := es.variationIndexForUser(vr, es.flag.Key, es.flag.Salt)
	if err != nil {
		es.logEvaluationError(err)
		return ldreason.NewEvaluationDetailForError(internal.ErrorKindForError(err), ldvalue.Null())
	}
	if inExperiment {
		reason = reasonToExperimentReason(reason)
	}
	return es.getVariation(index, reason)
}

func (es *evaluationScope) ruleMatchesUser(rule *ldmodel.FlagRule) (bool, error) {
	// Note that rule is passed by reference only for efficiency; we do not modify it
	for _, clause := range rule.Clauses {
		// Note, taking address of range variable here is OK because it's not used outside the loop
		match, err := es.clauseMatchesUser(&clause) //nolint:gosec // see comment above
		if !match || err != nil {
			return match, err
		}
	}
	return true, nil
}

func (es *evaluationScope) clauseMatchesUser(clause *ldmodel.Clause) (bool, error) {
	// Note that clause is passed by reference only for efficiency; we do not modify it
	// In the case of a segment match operator, we check if the user is in any of the segments,
	// and possibly negate
	if clause.Op == ldmodel.OperatorSegmentMatch {
		for _, value := range clause.Values {
			if value.Type() == ldvalue.StringType {
				if segment := es.owner.dataProvider.GetSegment(value.StringValue()); segment != nil {
					if es.segmentContainsUser(segment) {
						return !clause.Negate, nil // match - true unless negated
					}
				}
			}
		}
		return clause.Negate, nil // non-match - false unless negated
	}

	return ldmodel.ClauseMatchesContext(clause, &es.context)
}

func (es *evaluationScope) variationIndexForUser(
	r ldmodel.VariationOrRollout, key, salt string) (variationIndex int, inExperiment bool, err error) {
	if r.Variation.IsDefined() {
		return r.Variation.IntValue(), false, nil
	}
	if len(r.Rollout.Variations) == 0 {
		// This is an error (malformed flag); either Variation or Rollout must be non-nil.
		return -1, false, internal.EmptyRolloutError{}
	}

	bucketVal, err := es.bucketUser(r.Rollout.Seed, key, r.Rollout.BucketBy, salt)
	if err != nil {
		return -1, false, err
	}
	var sum float32

	isExperiment := r.Rollout.IsExperiment()

	for _, bucket := range r.Rollout.Variations {
		sum += float32(bucket.Weight) / 100000.0
		if bucketVal < sum {
			return bucket.Variation, isExperiment && !bucket.Untracked, nil
		}
	}

	// The user's bucket value was greater than or equal to the end of the last bucket. This could happen due
	// to a rounding error, or due to the fact that we are scaling to 100000 rather than 99999, or the flag
	// data could contain buckets that don't actually add up to 100000. Rather than returning an error in
	// this case (or changing the scaling, which would potentially change the results for *all* users), we
	// will simply put the user in the last bucket.
	lastBucket := r.Rollout.Variations[len(r.Rollout.Variations)-1]
	return lastBucket.Variation, isExperiment && !lastBucket.Untracked, nil
}

func (es *evaluationScope) logEvaluationError(err error) {
	if err == nil || es.owner.errorLogger == nil {
		return
	}
	if internal.ErrorKindForError(err) == ldreason.EvalErrorMalformedFlag {
		es.owner.errorLogger.Printf("Invalid flag configuration detected in flag %q: %s",
			es.flag.Key,
			err,
		)
	} else { // COVERAGE: currently no way to cause this in unit tests
		es.owner.errorLogger.Printf("Error in evaluating flag %q: %s",
			es.flag.Key,
			err,
		)
	}
}

func reasonToExperimentReason(reason ldreason.EvaluationReason) ldreason.EvaluationReason {
	switch reason.GetKind() {
	case ldreason.EvalReasonFallthrough:
		return ldreason.NewEvalReasonFallthroughExperiment(true)
	case ldreason.EvalReasonRuleMatch:
		return ldreason.NewEvalReasonRuleMatchExperiment(reason.GetRuleIndex(), reason.GetRuleID(), true)
	default:
		return reason // COVERAGE: unreachable
	}
}
