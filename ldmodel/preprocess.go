package ldmodel

import (
	"regexp"
	"time"

	"github.com/blang/semver"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

// PreprocessFlag precomputes internal data structures based on the flag configuration, to speed up
// evaluations.
//
// This is called once after a flag is deserialized from JSON, or is created with ldbuilders. If you
// construct a flag by some other means, you should call PreprocessFlag exactly once before making it
// available to any other code. The method is not safe for concurrent access across goroutines.
func PreprocessFlag(f *FeatureFlag) {
	for i, t := range f.Targets {
		f.Targets[i] = preprocessTarget(t)
	}
	for i, r := range f.Rules {
		for j, c := range r.Clauses {
			f.Rules[i].Clauses[j] = preprocessClause(c)
		}
	}
}

func preprocessTarget(t Target) Target {
	ret := t
	if len(t.Values) > 0 {
		m := make(map[string]bool, len(t.Values))
		for _, v := range t.Values {
			m[v] = true
		}
		ret.valuesMap = m
	}
	return ret
}

// clausePreprocessedValue contains data computed by preprocessClause() when appropriate to speed up
// clause evaluation.
type clausePreprocessedValue struct {
	computed     bool
	valid        bool
	parsedRegexp *regexp.Regexp // used for OperatorMatches
	parsedTime   time.Time      // used for OperatorAfter, OperatorBefore
	parsedSemver semver.Version // used for OperatorSemVerEqual, etc.
}

func preprocessClause(c Clause) Clause {
	ret := c
	switch c.Op {
	case OperatorMatches:
		ret.preprocessedValues = preprocessValues(c.Values, func(v ldvalue.Value) clausePreprocessedValue {
			r, ok := parseRegexp(v)
			return clausePreprocessedValue{valid: ok, parsedRegexp: r}
		})
	case OperatorBefore, OperatorAfter:
		ret.preprocessedValues = preprocessValues(c.Values, func(v ldvalue.Value) clausePreprocessedValue {
			t, ok := parseDateTime(v)
			return clausePreprocessedValue{valid: ok, parsedTime: t}
		})
	case OperatorSemVerEqual, OperatorSemVerGreaterThan, OperatorSemVerLessThan:
		ret.preprocessedValues = preprocessValues(c.Values, func(v ldvalue.Value) clausePreprocessedValue {
			s, ok := parseSemVer(v)
			return clausePreprocessedValue{valid: ok, parsedSemver: s}
		})
	default:
	}
	return ret
}

func preprocessValues(
	valuesIn []ldvalue.Value,
	fn func(ldvalue.Value) clausePreprocessedValue,
) []clausePreprocessedValue {
	ret := make([]clausePreprocessedValue, len(valuesIn))
	for i, v := range valuesIn {
		p := fn(v)
		p.computed = true
		ret[i] = p
	}
	return ret
}
