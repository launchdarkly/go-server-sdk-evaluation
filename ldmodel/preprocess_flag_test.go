package ldmodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreprocessFlagBuildsTargetMap(t *testing.T) {
	f := FeatureFlag{
		Targets: []Target{
			{
				Variation: 0,
				Values:    nil,
			},
			{
				Variation: 1,
				Values:    []string{"a", "b"},
			},
		},
	}

	assert.Nil(t, f.Targets[0].valuesMap)
	assert.Nil(t, f.Targets[1].valuesMap)

	f.Preprocess()

	assert.Nil(t, f.Targets[0].valuesMap)

	assert.Len(t, f.Targets[1].valuesMap, 2)
	assert.True(t, f.Targets[1].valuesMap["a"])
	assert.True(t, f.Targets[1].valuesMap["b"])
}
