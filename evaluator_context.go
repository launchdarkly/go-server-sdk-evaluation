package evaluation

import (
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-server-sdk-evaluation/v3/ldmodel"
)

// ContextEvaluator evaluates any number of feature flags for one evaluation context.
//
// Between evaluations it keeps what it has read from the context (its individual contexts by kind and
// their attribute values) and the result of every segment whose result depends only on the segment and
// the context: one with no big segment and no rule that references another segment. Evaluating many
// flags for one context through a ContextEvaluator therefore repeats less work than calling
// Evaluator.Evaluate for each, which is the case when a server evaluates a whole environment's flags
// for a client. Every result is identical to the one Evaluator.Evaluate returns for the same flag and
// context, and the evaluator's DataProvider and BigSegmentProvider are called exactly as Evaluate
// calls them.
//
// A ContextEvaluator is not safe for concurrent use. Because it remembers segment results by segment,
// the DataProvider must not change a segment's definition in place while the ContextEvaluator is in
// use (returning a different *ldmodel.Segment for an updated segment, as a data store does, is fine):
// create a ContextEvaluator for a batch of evaluations and discard it afterward.
type ContextEvaluator interface {
	// Evaluate evaluates a feature flag for the ContextEvaluator's context. It is otherwise the same as
	// Evaluator.Evaluate.
	Evaluate(flag *ldmodel.FeatureFlag, prerequisiteFlagEventRecorder PrerequisiteFlagEventRecorder) Result
}

// NewContextEvaluator returns a ContextEvaluator for the specified context, which uses the data
// provider, big segment provider and options of the specified Evaluator.
//
// The Evaluator should be one created by NewEvaluator or NewEvaluatorWithOptions; for any other
// implementation, the ContextEvaluator simply delegates each evaluation to it.
func NewContextEvaluator(e Evaluator, context ldcontext.Context) ContextEvaluator {
	owner, ok := e.(*evaluator)
	if !ok {
		return delegatingContextEvaluator{e: e, context: context}
	}
	ce := &contextEvaluator{owner: owner, context: context}
	ce.contexts = newContextCache(&ce.context)
	return ce
}

type contextEvaluator struct {
	owner    *evaluator
	context  ldcontext.Context
	contexts contextCache
}

func (ce *contextEvaluator) Evaluate(
	flag *ldmodel.FeatureFlag,
	prerequisiteFlagEventRecorder PrerequisiteFlagEventRecorder,
) Result {
	return ce.owner.evaluate(flag, ce.context, &ce.contexts, prerequisiteFlagEventRecorder)
}

type delegatingContextEvaluator struct {
	e       Evaluator
	context ldcontext.Context
}

func (d delegatingContextEvaluator) Evaluate(
	flag *ldmodel.FeatureFlag,
	prerequisiteFlagEventRecorder PrerequisiteFlagEventRecorder,
) Result {
	return d.e.Evaluate(flag, d.context, prerequisiteFlagEventRecorder)
}
