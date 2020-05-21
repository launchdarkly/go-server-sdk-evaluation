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

func TestPreprocessSegmentBuildsIncludeAndExcludeMaps(t *testing.T) {
	s := Segment{
		Included: []string{"a", "b"},
		Excluded: []string{"c"},
	}

	assert.Nil(t, s.preprocessed.includeMap)
	assert.Nil(t, s.preprocessed.excludeMap)

	PreprocessSegment(&s)

	assert.Len(t, s.preprocessed.includeMap, 2)
	assert.True(t, s.preprocessed.includeMap["a"])
	assert.True(t, s.preprocessed.includeMap["b"])

	assert.Len(t, s.preprocessed.excludeMap, 1)
	assert.True(t, s.preprocessed.excludeMap["c"])
}
