package evaluation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldlog"
)

func TestEvaluatorDefaultOptions(t *testing.T) {
	d := basicDataProvider()

	e1 := NewEvaluator(d).(*evaluator)
	assert.Equal(t, d, e1.dataProvider)
	assert.Nil(t, e1.bigSegmentProvider)
	assert.Nil(t, e1.errorLogger)

	e2 := NewEvaluatorWithOptions(d).(*evaluator)
	assert.Equal(t, d, e2.dataProvider)
	assert.Nil(t, e2.bigSegmentProvider)
	assert.Nil(t, e2.errorLogger)
}

func TestEvaluatorOptionBigSegmentProvider(t *testing.T) {
	d := basicDataProvider()
	b := basicBigSegmentsProvider()
	e := NewEvaluatorWithOptions(d, EvaluatorOptionBigSegmentProvider(b)).(*evaluator)
	assert.Equal(t, d, e.dataProvider)
	assert.Equal(t, b, e.bigSegmentProvider)
	assert.Nil(t, e.errorLogger)
}

func TestEvaluatorOptionErrorLogger(t *testing.T) {
	d := basicDataProvider()
	logger := ldlog.NewDefaultLoggers().ForLevel(ldlog.Error)
	e := NewEvaluatorWithOptions(d, EvaluatorOptionErrorLogger(logger)).(*evaluator)
	assert.Equal(t, d, e.dataProvider)
	assert.Nil(t, e.bigSegmentProvider)
	assert.Equal(t, logger, e.errorLogger)
}
