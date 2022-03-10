package evaluation

import (
	"strings"
	"time"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldmodel"
)

type opFn (func(clause *ldmodel.Clause, contextValue, clauseValue ldvalue.Value, clauseValueIndex int) bool)

var allOps = map[ldmodel.Operator]opFn{ //nolint:gochecknoglobals
	ldmodel.OperatorIn:                 operatorInFn,
	ldmodel.OperatorEndsWith:           operatorEndsWithFn,
	ldmodel.OperatorStartsWith:         operatorStartsWithFn,
	ldmodel.OperatorMatches:            operatorMatchesFn,
	ldmodel.OperatorContains:           operatorContainsFn,
	ldmodel.OperatorLessThan:           operatorLessThanFn,
	ldmodel.OperatorLessThanOrEqual:    operatorLessThanOrEqualFn,
	ldmodel.OperatorGreaterThan:        operatorGreaterThanFn,
	ldmodel.OperatorGreaterThanOrEqual: operatorGreaterThanOrEqualFn,
	ldmodel.OperatorBefore:             operatorBeforeFn,
	ldmodel.OperatorAfter:              operatorAfterFn,
	ldmodel.OperatorSemVerEqual:        operatorSemVerEqualFn,
	ldmodel.OperatorSemVerLessThan:     operatorSemVerLessThanFn,
	ldmodel.OperatorSemVerGreaterThan:  operatorSemVerGreaterThanFn,
}

func clauseMatchesContext(c *ldmodel.Clause, context *ldcontext.Context) (bool, error) {
	if !c.Attribute.IsDefined() {
		return false, emptyAttrRefError{}
	}
	if c.Attribute.Err() != nil {
		return false, badAttrRefError(c.Attribute.String())
	}
	if c.Attribute.String() == ldattr.KindAttr {
		return maybeNegate(c.Negate, clauseMatchByKind(c, context)), nil
	}
	kind := c.Kind
	if kind == "" {
		kind = ldcontext.DefaultKind
	}
	actualContext := *context
	if context.Multiple() {
		if individualContext, ok := context.MultiKindByName(kind); ok {
			actualContext = individualContext
		} else {
			return false, nil
		}
	} else {
		if context.Kind() != kind {
			return false, nil
		}
	}
	uValue, _ := actualContext.GetValueForRef(c.Attribute)
	if uValue.IsNull() {
		// if the user attribute is null/missing, it's an automatic non-match - regardless of c.Negate
		return false, nil
	}
	matchFn := operatorFn(c.Op)

	// If the user value is an array, see if the intersection is non-empty. If so, this clause matches
	if uValue.Type() == ldvalue.ArrayType {
		for i := 0; i < uValue.Count(); i++ {
			if matchAny(c, matchFn, uValue.GetByIndex(i)) {
				return maybeNegate(c.Negate, true), nil
			}
		}
		return maybeNegate(c.Negate, false), nil
	}

	return maybeNegate(c.Negate, matchAny(c, matchFn, uValue)), nil
}

func maybeNegate(negate, result bool) bool {
	if negate {
		return !result
	}
	return result
}

func matchAny(
	c *ldmodel.Clause,
	fn opFn,
	value ldvalue.Value,
) bool {
	if c.Op == ldmodel.OperatorIn {
		return ldmodel.EvaluatorAccessors.ClauseFindValue(c, value)
	}
	for i, v := range c.Values {
		if fn(c, value, v, i) {
			return true
		}
	}
	return false
}

func clauseMatchByKind(c *ldmodel.Clause, context *ldcontext.Context) bool {
	// If Attribute is "kind", then we treat Operator and Values as a match expression against a list
	// of all individual kinds in the context. That is, for a multi-kind context with kinds of "org"
	// and "user", it is a match if either of those strings is a match with Operator and Values.
	matchFn := operatorFn(c.Op)
	if context.Multiple() {
		for i := 0; i < context.MultiKindCount(); i++ {
			if individualContext, ok := context.MultiKindByIndex(i); ok {
				ctxValue := ldvalue.String(string(individualContext.Kind()))
				if matchAny(c, matchFn, ctxValue) {
					return true
				}
			}
		}
		return false
	}
	ctxValue := ldvalue.String(string(context.Kind()))
	return matchAny(c, matchFn, ctxValue)
}

func operatorFn(operator ldmodel.Operator) opFn {
	if op, ok := allOps[operator]; ok {
		return op
	}
	return operatorNoneFn
}

func operatorInFn(
	clause *ldmodel.Clause,
	ctxValue, clValue ldvalue.Value,
	clValueIndex int,
) bool {
	return ctxValue.Equal(clValue)
}

func stringOperator(
	ctxValue, clValue ldvalue.Value,
	stringTestFn func(string, string) bool,
) bool {
	if ctxValue.IsString() && clValue.IsString() {
		return stringTestFn(ctxValue.StringValue(), clValue.StringValue())
	}
	return false
}

func operatorStartsWithFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	return stringOperator(ctxValue, clValue, strings.HasPrefix)
}

func operatorEndsWithFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	return stringOperator(ctxValue, clValue, strings.HasSuffix)
}

func operatorMatchesFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	if ctxValue.IsString() {
		r := ldmodel.EvaluatorAccessors.ClauseGetValueAsRegexp(c, clValueIndex)
		if r != nil {
			return r.MatchString(ctxValue.StringValue())
		}
	}
	return false
}

func operatorContainsFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	return stringOperator(ctxValue, clValue, strings.Contains)
}

func numericOperator(ctxValue, clValue ldvalue.Value, fn func(float64, float64) bool) bool {
	if ctxValue.IsNumber() && clValue.IsNumber() {
		return fn(ctxValue.Float64Value(), clValue.Float64Value())
	}
	return false
}

func operatorLessThanFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	return numericOperator(ctxValue, clValue, func(a float64, b float64) bool { return a < b })
}

func operatorLessThanOrEqualFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	return numericOperator(ctxValue, clValue, func(a float64, b float64) bool { return a <= b })
}

func operatorGreaterThanFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	return numericOperator(ctxValue, clValue, func(a float64, b float64) bool { return a > b })
}

func operatorGreaterThanOrEqualFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	return numericOperator(ctxValue, clValue, func(a float64, b float64) bool { return a >= b })
}

func dateOperator(
	c *ldmodel.Clause,
	ctxValue ldvalue.Value,
	clValueIndex int,
	fn func(time.Time, time.Time) bool,
) bool {
	if clValueTime, ok := ldmodel.EvaluatorAccessors.ClauseGetValueAsTimestamp(c, clValueIndex); ok {
		if ctxValueTime, ok := ldmodel.TypeConversions.ValueToTimestamp(ctxValue); ok {
			return fn(ctxValueTime, clValueTime)
		}
	}
	return false
}

func operatorBeforeFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	return dateOperator(c, ctxValue, clValueIndex, time.Time.Before)
}

func operatorAfterFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	return dateOperator(c, ctxValue, clValueIndex, time.Time.After)
}

func semVerOperator(
	c *ldmodel.Clause,
	ctxValue ldvalue.Value,
	clValueIndex int,
	expectedComparisonResult int,
) bool {
	if clValueVer, ok := ldmodel.EvaluatorAccessors.ClauseGetValueAsSemanticVersion(c, clValueIndex); ok {
		if ctxValueVer, ok := ldmodel.TypeConversions.ValueToSemanticVersion(ctxValue); ok {
			return ctxValueVer.ComparePrecedence(clValueVer) == expectedComparisonResult
		}
	}
	return false
}

func operatorSemVerEqualFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	return semVerOperator(c, ctxValue, clValueIndex, 0)
}

func operatorSemVerLessThanFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	return semVerOperator(c, ctxValue, clValueIndex, -1)
}

func operatorSemVerGreaterThanFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	return semVerOperator(c, ctxValue, clValueIndex, 1)
}

func operatorNoneFn(c *ldmodel.Clause, ctxValue, clValue ldvalue.Value, clValueIndex int) bool {
	return false
}
