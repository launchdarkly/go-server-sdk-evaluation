package internal

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalBuffer(t *testing.T) {
	for _, preallocated := range []bool{false, true} {
		t.Run(fmt.Sprintf("preallocated=%t", preallocated), func(t *testing.T) {
			var buf LocalBuffer
			if preallocated {
				buf.Data = make([]byte, 0, 100)
			}

			buf.AppendString("abc")
			buf.AppendByte('.')
			buf.Append([]byte("def"))
			buf.AppendInt(123)
			assert.Equal(t, "abc.def123", string(buf.Data))

			if preallocated {
				assert.Equal(t, 100, cap(buf.Data))
			}
		})
	}

	t.Run("reallocation", func(t *testing.T) {
		buf := LocalBuffer{Data: make([]byte, 0, 5)}

		// by default, the capacity doubles when we reallocate...
		buf.AppendString("abc")
		buf.AppendString("def")
		assert.Equal(t, "abcdef", string(buf.Data))
		assert.Equal(t, 10, cap(buf.Data)) // capacity was doubled

		// ...unless that would not be enough to hold the new data, in which case we allocate
		// double the total new length
		buf.AppendString("ghijklmnopqrstuvwxyz")
		assert.Equal(t, "abcdefghijklmnopqrstuvwxyz", string(buf.Data))
		assert.Equal(t, 52, cap(buf.Data)) //
	})
}
