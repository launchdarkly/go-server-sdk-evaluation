package ldmodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSegmentPropertyGetters(t *testing.T) {
	segment := Segment{
		Key:     "segment-key",
		Version: 99,
	}
	assert.Equal(t, "segment-key", segment.GetKey())
	assert.Equal(t, 99, segment.GetVersion())
	assert.False(t, segment.IsDeleted())

	assert.True(t, (&Segment{Deleted: true}).IsDeleted())
}
