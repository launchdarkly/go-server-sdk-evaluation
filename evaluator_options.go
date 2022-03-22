package evaluation

import "github.com/launchdarkly/go-sdk-common/v3/ldlog"

// EvaluatorOption is an optional parameter for NewEvaluator.
type EvaluatorOption interface {
	apply(e *evaluator)
}

type evaluatorOptionBigSegmentProvider struct{ bigSegmentProvider BigSegmentProvider }

// EvaluatorOptionBigSegmentProvider is an option for NewEvaluator that specifies a
// BigSegmentProvider for evaluating big segment membership. If the parameter is nil, it will
// be treated the same as a BigSegmentProvider that always returns a "store not configured"
// status.
func EvaluatorOptionBigSegmentProvider(bigSegmentProvider BigSegmentProvider) EvaluatorOption {
	return evaluatorOptionBigSegmentProvider{bigSegmentProvider: bigSegmentProvider}
}

func (o evaluatorOptionBigSegmentProvider) apply(e *evaluator) {
	e.bigSegmentProvider = o.bigSegmentProvider
}

type evaluatorOptionErrorLogger struct{ errorLogger ldlog.BaseLogger }

// EvaluatorOptionErrorLogger is an option for NewEvaluator that specifies a logger for
// error reporting. The Evaluator will only log errors for conditions that should not be
// possible and require investigation, such as a malformed flag or a code path that should
// not have been reached. If the parameter is nil, no logging is done.
func EvaluatorOptionErrorLogger(errorLogger ldlog.BaseLogger) EvaluatorOption {
	return evaluatorOptionErrorLogger{errorLogger: errorLogger}
}

func (o evaluatorOptionErrorLogger) apply(e *evaluator) {
	e.errorLogger = o.errorLogger
}
