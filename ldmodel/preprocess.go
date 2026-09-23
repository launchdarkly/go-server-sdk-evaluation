package ldmodel

import (
	"regexp"
	"time"

	"github.com/launchdarkly/go-semver"

	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
)

type targetPreprocessedData struct {
	valuesMap map[string]struct{}
}

type segmentPreprocessedData struct {
	includeMap map[string]struct{}
	excludeMap map[string]struct{}
}

type clausePreprocessedData struct {
	values    []clausePreprocessedValue
	valuesMap map[jsonPrimitiveValueKey]struct{}
}

type clausePreprocessedValue struct {
	computed     bool
	valid        bool
	parsedRegexp *regexp.Regexp // used for OperatorMatches
	parsedTime   time.Time      // used for OperatorAfter, OperatorBefore
	parsedSemver semver.Version // used for OperatorSemVerEqual, etc.
}

type jsonPrimitiveValueKey struct {
	valueType    ldvalue.ValueType
	booleanValue bool
	numberValue  float64
	stringValue  string
}

func (j jsonPrimitiveValueKey) isValid() bool {
	return j.valueType != ldvalue.NullType
}

// PreprocessOptions configures memory optimizations for flag preprocessing.
type PreprocessOptions struct {
	// DiscardRedundantClauseValues, if true, frees Clause.Values by setting it to nil
	// for OperatorIn clauses where an internal valuesMap has been successfully constructed.
	// This significantly reduces in-memory footprint in environments with large targeting
	// lists (e.g., thousands of user/account keys) by avoiding storing both the slice
	// and the hash map simultaneously.
	//
	// Note: If enabled, flags should not be re-serialized to JSON via model_marshal.
	DiscardRedundantClauseValues bool
}

// PreprocessFlagWithOptions precomputes internal data structures based on the flag
// configuration with configurable optimization options.
func PreprocessFlagWithOptions(f *FeatureFlag, opts PreprocessOptions) {
	for i, t := range f.Targets {
		f.Targets[i].preprocessed.valuesMap = preprocessStringSet(t.Values)
	}
	for i, r := range f.Rules {
		for j, c := range r.Clauses {
			preprocessed := preprocessClause(c)
			if opts.DiscardRedundantClauseValues && preprocessed.valuesMap != nil {
				f.Rules[i].Clauses[j].Values = nil
			}
			f.Rules[i].Clauses[j].preprocessed = preprocessed
		}
	}
}

// PreprocessFlag precomputes internal data structures based on the flag configuration, to speed up
// evaluations.
//
// This is called once after a flag is deserialized from JSON, or is created with ldbuilders. If you
// construct a flag by some other means, you should call PreprocessFlag exactly once before making it
// available to any other code. The method is not safe for concurrent access across goroutines.
func PreprocessFlag(f *FeatureFlag) {
	PreprocessFlagWithOptions(f, PreprocessOptions{})
}

// PreprocessSegmentWithOptions precomputes internal data structures based on the segment
// configuration with configurable optimization options.
func PreprocessSegmentWithOptions(s *Segment, opts PreprocessOptions) {
	p := segmentPreprocessedData{}
	p.includeMap = preprocessStringSet(s.Included)
	p.excludeMap = preprocessStringSet(s.Excluded)
	for i, t := range s.IncludedContexts {
		s.IncludedContexts[i].preprocessed.valuesMap = preprocessStringSet(t.Values)
	}
	for i, t := range s.ExcludedContexts {
		s.ExcludedContexts[i].preprocessed.valuesMap = preprocessStringSet(t.Values)
	}
	s.preprocessed = p

	for i, r := range s.Rules {
		for j, c := range r.Clauses {
			preprocessed := preprocessClause(c)
			if opts.DiscardRedundantClauseValues && preprocessed.valuesMap != nil {
				s.Rules[i].Clauses[j].Values = nil
			}
			s.Rules[i].Clauses[j].preprocessed = preprocessed
		}
	}
}

// PreprocessSegment precomputes internal data structures based on the segment configuration, to speed up
// evaluations.
//
// This is called once after a segment is deserialized from JSON, or is created with ldbuilders. If you
// construct a segment by some other means, you should call PreprocessSegment exactly once before making
// it available to any other code. The method is not safe for concurrent access across goroutines.
func PreprocessSegment(s *Segment) {
	PreprocessSegmentWithOptions(s, PreprocessOptions{})
}

func preprocessClause(c Clause) clausePreprocessedData {
	ret := clausePreprocessedData{}
	switch c.Op {
	case OperatorIn:
		// This is a special case where the clause is testing for an exact match against any of the
		// clause values. As long as the values are primitives, we can use them in a map key (map
		// keys just can't contain slices or maps), and we can convert this test from a linear search
		// to a map lookup.
		if len(c.Values) > 1 { // don't bother if it's empty or has a single value
			valid := true
			m := make(map[jsonPrimitiveValueKey]struct{}, len(c.Values))
			for _, v := range c.Values {
				if key := asPrimitiveValueKey(v); key.isValid() {
					m[key] = struct{}{}
				} else {
					valid = false
					break
				}
			}
			if valid {
				ret.valuesMap = m
			}
		}
	case OperatorMatches:
		ret.values = preprocessValues(c.Values, func(v ldvalue.Value) clausePreprocessedValue {
			r := parseRegexp(v)
			return clausePreprocessedValue{valid: r != nil, parsedRegexp: r}
		})
	case OperatorBefore, OperatorAfter:
		ret.values = preprocessValues(c.Values, func(v ldvalue.Value) clausePreprocessedValue {
			t, ok := parseDateTime(v)
			return clausePreprocessedValue{valid: ok, parsedTime: t}
		})
	case OperatorSemVerEqual, OperatorSemVerGreaterThan, OperatorSemVerLessThan:
		ret.values = preprocessValues(c.Values, func(v ldvalue.Value) clausePreprocessedValue {
			s, ok := parseSemVer(v)
			return clausePreprocessedValue{valid: ok, parsedSemver: s}
		})
	default:
	}
	return ret
}

func asPrimitiveValueKey(v ldvalue.Value) jsonPrimitiveValueKey {
	switch v.Type() {
	case ldvalue.BoolType:
		return jsonPrimitiveValueKey{valueType: ldvalue.BoolType, booleanValue: v.BoolValue()}
	case ldvalue.NumberType:
		return jsonPrimitiveValueKey{valueType: ldvalue.NumberType, numberValue: v.Float64Value()}
	case ldvalue.StringType:
		return jsonPrimitiveValueKey{valueType: ldvalue.StringType, stringValue: v.StringValue()}
	default:
		return jsonPrimitiveValueKey{}
	}
}

func preprocessStringSet(valuesIn []string) map[string]struct{} {
	if len(valuesIn) == 0 {
		return nil
	}
	ret := make(map[string]struct{}, len(valuesIn))
	for _, value := range valuesIn {
		ret[value] = struct{}{}
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
