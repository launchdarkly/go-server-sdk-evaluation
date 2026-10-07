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

// clauseInValuesSetMinSize is the smallest number of values for which an OperatorIn clause uses a
// set lookup. Benchmarks show that a linear scan is faster for shorter lists, with a hot or a cold
// CPU cache.
const clauseInValuesSetMinSize = 8

type clausePreprocessedData struct {
	values []clausePreprocessedValue
	// inValueSet is the set of values of an OperatorIn clause. It maps each distinct value to its
	// position among the distinct values, in the order they first appear in Clause.Values, so that a
	// released clause can write its values in their original order without keeping the list. It is
	// nil if the clause was not preprocessed, if it has fewer than clauseInValuesSetMinSize values,
	// or if a value is not a primitive. Evaluation uses Clause.Values if inValueSet is nil.
	inValueSet map[jsonPrimitiveValueKey]int32
}

// newClauseInValueSet returns a set of the values if there are at least clauseInValuesSetMinSize
// values and all of them are primitives. Otherwise it returns nil.
func newClauseInValueSet(values []ldvalue.Value) map[jsonPrimitiveValueKey]int32 {
	if len(values) < clauseInValuesSetMinSize {
		return nil
	}
	set := make(map[jsonPrimitiveValueKey]int32, len(values))
	for _, v := range values {
		key := asPrimitiveValueKey(v)
		if !key.isValid() {
			return nil
		}
		if _, seen := set[key]; !seen {
			set[key] = int32(len(set)) //nolint:gosec // a clause cannot hold 2^31 values
		}
	}
	return set
}

func listContainsValue(list []ldvalue.Value, value ldvalue.Value) bool {
	switch value.Type() {
	case ldvalue.BoolType, ldvalue.NumberType, ldvalue.StringType:
		for _, v := range list {
			if value.Equal(v) {
				return true
			}
		}
	default:
	}
	return false
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

func (j jsonPrimitiveValueKey) toValue() ldvalue.Value {
	switch j.valueType {
	case ldvalue.BoolType:
		return ldvalue.Bool(j.booleanValue)
	case ldvalue.NumberType:
		return ldvalue.Float64(j.numberValue)
	case ldvalue.StringType:
		return ldvalue.String(j.stringValue)
	default:
		return ldvalue.Null()
	}
}

// PreprocessFlag precomputes internal data structures based on the flag configuration, to speed up
// evaluations.
//
// This is called once after a flag is deserialized from JSON, or is created with ldbuilders. If you
// construct a flag by some other means, you should call PreprocessFlag exactly once before making it
// available to any other code. The method is not safe for concurrent access across goroutines.
func PreprocessFlag(f *FeatureFlag) {
	for i, t := range f.Targets {
		f.Targets[i].preprocessed.valuesMap = preprocessStringSet(t.Values)
	}
	for _, r := range f.Rules {
		preprocessClauses(r.Clauses)
	}
}

// PreprocessSegment precomputes internal data structures based on the segment configuration, to speed up
// evaluations.
//
// This is called once after a segment is deserialized from JSON, or is created with ldbuilders. If you
// construct a segment by some other means, you should call PreprocessSegment exactly once before making
// it available to any other code. The method is not safe for concurrent access across goroutines.
func PreprocessSegment(s *Segment) {
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

	for _, r := range s.Rules {
		preprocessClauses(r.Clauses)
	}
}

// preprocessClauses preprocesses each clause in place. A clause that ReleaseClauseValues released
// has no Values to preprocess, so it keeps its existing data.
func preprocessClauses(clauses []Clause) {
	for i := range clauses {
		c := &clauses[i]
		if c.valuesReleased() {
			continue
		}
		c.preprocessed = preprocessClause(*c)
	}
}

// ReleaseClauseValues reduces the memory use of a preprocessed flag. Each OperatorIn clause in the
// flag rules that uses a set lookup stops holding its Values list, and Clause.Values becomes nil.
// Evaluation and JSON serialization use the set for these clauses. Serialization writes the values
// in the order they first appeared, without duplicates, so a released clause serializes the same as
// the original unless its list repeated a value.
//
// Call this function after PreprocessFlag, and before the flag is available to other code. The
// function changes the flag in place, which includes copies of the flag that share its Rules. The
// function is not safe for concurrent access across goroutines.
func ReleaseClauseValues(f *FeatureFlag) {
	for _, r := range f.Rules {
		releaseClauseValues(r.Clauses)
	}
}

// ReleaseSegmentClauseValues reduces the memory use of a preprocessed segment, in the same way
// that ReleaseClauseValues does for a flag.
//
// Call this function after PreprocessSegment, and before the segment is available to other code.
// The function changes the segment in place, which includes copies of the segment that share its
// Rules. The function is not safe for concurrent access across goroutines.
func ReleaseSegmentClauseValues(s *Segment) {
	for _, r := range s.Rules {
		releaseClauseValues(r.Clauses)
	}
}

func releaseClauseValues(clauses []Clause) {
	for i := range clauses {
		c := &clauses[i]
		if c.preprocessed.inValueSet != nil {
			c.Values = nil
		}
	}
}

func preprocessClause(c Clause) clausePreprocessedData {
	ret := clausePreprocessedData{}
	switch c.Op {
	case OperatorIn:
		// This is a special case where the clause is testing for an exact match against any of the
		// clause values. As long as the values are primitives, we can use them in a map key (map
		// keys just can't contain slices or maps), and we can convert this test from a linear search
		// to a map lookup. A short list is faster to scan than a map is to look up, so the map is
		// built only for lists with at least clauseInValuesSetMinSize values.
		ret.inValueSet = newClauseInValueSet(c.Values)
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
