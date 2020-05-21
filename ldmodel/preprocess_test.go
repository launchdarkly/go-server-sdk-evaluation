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

	assert.Nil(t, f.Targets[0].preprocessed.valuesMap)
	assert.Nil(t, f.Targets[1].preprocessed.valuesMap)

	PreprocessFlag(&f)

	assert.Nil(t, f.Targets[0].preprocessed.valuesMap)

	assert.Len(t, f.Targets[1].preprocessed.valuesMap, 2)
	assert.True(t, f.Targets[1].preprocessed.valuesMap["a"])
	assert.True(t, f.Targets[1].preprocessed.valuesMap["b"])
}
