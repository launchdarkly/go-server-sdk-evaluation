package ldmodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRolloutIsExperiment(t *testing.T) {
	r := Rollout{}
	assert.False(t, r.IsExperiment(), `rollout with undefined kind is not an experiment`)

	r = Rollout{Kind: RolloutKind("bogus")}
	assert.False(t, r.IsExperiment(), `rollout with bogus kind is not an experiment`)

	r = Rollout{Kind: RolloutKindRollout}
	assert.False(t, r.IsExperiment(), `rollout with kind "rollout" is not an experiment`)

	r = Rollout{Kind: RolloutKindExperiment}
	assert.True(t, r.IsExperiment())
}
