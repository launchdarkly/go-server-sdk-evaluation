package evaluation

import (
	"fmt"
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v3/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldmodel"

	"github.com/stretchr/testify/assert"
)

const dateStr1 = "2017-12-06T00:00:00.000-07:00"
const dateStr2 = "2017-12-06T00:01:01.000-07:00"
const dateMs1 = 10000000
const dateMs2 = 10000001
const invalidDate = "hey what's this?"

type opTestInfo struct {
	opName           ldmodel.Operator
	userValue        interface{}
	clauseValue      interface{}
	moreClauseValues []interface{}
	expected         bool
}

var operatorTests = []opTestInfo{
	// numeric operators
	{"in", int(99), int(99), nil, true},
	{"in", int(99), int(99), []interface{}{int(98), int(97), int(96)}, true},
	{"in", float64(99.0001), float64(99.0001), nil, true},
	{"in", float64(99.0001), float64(99.0001), []interface{}{float64(98), float64(97), float64(96)}, true},
	{"lessThan", int(1), float64(1.99999), nil, true},
	{"lessThan", float64(1.99999), int(1), nil, false},
	{"lessThan", int(1), uint(2), nil, true},
	{"lessThanOrEqual", int(1), float64(1), nil, true},
	{"greaterThan", int(2), float64(1.99999), nil, true},
	{"greaterThan", float64(1.99999), int(2), nil, false},
	{"greaterThan", int(2), uint(1), nil, true},
	{"greaterThanOrEqual", int(1), float64(1), nil, true},

	// string operators
	{"in", "x", "x", nil, true},
	{"in", "x", "x", []interface{}{"a", "b", "c"}, true},
	{"in", "x", "xyz", nil, false},
	{"startsWith", "xyz", "x", nil, true},
	{"startsWith", "x", "xyz", nil, false},
	{"endsWith", "xyz", "z", nil, true},
	{"endsWith", "z", "xyz", nil, false},
	{"contains", "xyz", "y", nil, true},
	{"contains", "y", "xyz", nil, false},

	// mixed strings and numbers
	{"in", "99", int(99), nil, false},
	{"in", int(99), "99", nil, false},
	{"contains", "99", int(99), nil, false},
	{"startsWith", "99", int(99), nil, false},
	{"endsWith", "99", int(99), nil, false},
	{"lessThanOrEqual", "99", int(99), nil, false},
	{"lessThanOrEqual", int(99), "99", nil, false},
	{"greaterThanOrEqual", "99", int(99), nil, false},
	{"greaterThanOrEqual", int(99), "99", nil, false},

	// equality of boolean values
	{"in", true, true, nil, true},
	{"in", false, false, nil, true},
	{"in", true, false, nil, false},
	{"in", false, true, nil, false},
	{"in", true, false, []interface{}{true}, true},

	// regex
	{"matches", "hello world", "hello.*rld", nil, true},
	{"matches", "hello world", "hello.*orl", nil, true},
	{"matches", "hello world", "l+", nil, true},
	{"matches", "hello world", "(world|planet)", nil, true},
	{"matches", "hello world", "aloha", nil, false},
	{"matches", "hello world", "***bad regex", nil, false},

	// date operators
	{"before", dateStr1, dateStr2, nil, true},
	{"before", dateMs1, dateMs2, nil, true},
	{"before", dateStr2, dateStr1, nil, false},
	{"before", dateMs2, dateMs1, nil, false},
	{"before", dateStr1, dateStr1, nil, false},
	{"before", dateMs1, dateMs1, nil, false},
	{"before", nil, dateStr1, nil, false},
	{"before", dateStr1, invalidDate, nil, false},
	{"after", dateStr2, dateStr1, nil, true},
	{"after", dateMs2, dateMs1, nil, true},
	{"after", dateStr1, dateStr2, nil, false},
	{"after", dateMs1, dateMs2, nil, false},
	{"after", dateStr1, dateStr1, nil, false},
	{"after", dateMs1, dateMs1, nil, false},
	{"after", nil, dateStr1, nil, false},
	{"after", dateStr1, invalidDate, nil, false},

	// semver operators
	{"semVerEqual", "2.0.0", "2.0.0", nil, true},
	{"semVerEqual", "2.0", "2.0.0", nil, true},
	{"semVerEqual", "2-rc1", "2.0.0-rc1", nil, true},
	{"semVerEqual", "2+build2", "2.0.0+build2", nil, true},
	{"semVerEqual", "2.0.0", "2.0.1", nil, false},
	{"semVerLessThan", "2.0.0", "2.0.1", nil, true},
	{"semVerLessThan", "2.0", "2.0.1", nil, true},
	{"semVerLessThan", "2.0.1", "2.0.0", nil, false},
	{"semVerLessThan", "2.0.1", "2.0", nil, false},
	{"semVerLessThan", "2.0.1", "xbad%ver", nil, false},
	{"semVerLessThan", "2.0.0-rc", "2.0.0-rc.beta", nil, true},
	{"semVerGreaterThan", "2.0.1", "2.0", nil, true},
	{"semVerGreaterThan", "10.0.1", "2.0", nil, true},
	{"semVerGreaterThan", "2.0.0", "2.0.1", nil, false},
	{"semVerGreaterThan", "2.0", "2.0.1", nil, false},
	{"semVerGreaterThan", "2.0.1", "xbad%ver", nil, false},
	{"semVerGreaterThan", "2.0.0-rc.1", "2.0.0-rc.0", nil, true},

	// invalid operator
	{"whatever", "x", "x", nil, false},
}

func TestAllOperators(t *testing.T) {
	userAttr := "attr"
	for _, ti := range operatorTests {
		for _, withPreprocessing := range []bool{false, true} {
			t.Run(
				fmt.Sprintf("%v %s %v should be %v (preprocess: %t)", ti.userValue, ti.opName, ti.clauseValue, ti.expected, withPreprocessing),
				func(t *testing.T) {
					uValue := ldvalue.CopyArbitraryValue(ti.userValue)
					cValue := ldvalue.CopyArbitraryValue(ti.clauseValue)
					c := ldbuilders.Clause(userAttr, ti.opName)
					for _, v := range ti.moreClauseValues {
						c.Values = append(c.Values, ldvalue.CopyArbitraryValue(v))
					}
					c.Values = append(c.Values, cValue)
					if withPreprocessing {
						flag := ldmodel.FeatureFlag{
							Rules: []ldmodel.FlagRule{
								{Clauses: []ldmodel.Clause{c}},
							},
						}
						ldmodel.PreprocessFlag(&flag)
						c = flag.Rules[0].Clauses[0]
					}
					context := lduser.NewUserBuilder("key").Custom(userAttr, uValue).Build()
					isMatch, err := clauseMatchesContext(&c, &context)
					assert.NoError(t, err)
					assert.Equal(t, ti.expected, isMatch)
				},
			)
		}
	}
}
