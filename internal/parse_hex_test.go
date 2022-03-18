package internal

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
)

func TestParseHexUint64IsEquivalentToStrconvParseUint(t *testing.T) {
	validate := func(t *testing.T, s string) bool {
		expectedResult, expectedError := strconv.ParseUint(s, 16, 64)
		expectedOK := expectedError == nil
		result, ok := ParseHexUint64([]byte(s))

		if expectedResult != result || ok != expectedOK {
			t.Errorf("For input %q, strconv returned (%v, %v) but ParseHexUint64 returned (%v, %v)",
				s, expectedResult, expectedError, result, ok)
			return false
		}
		return true
	}

	t.Run("for empty string", func(t *testing.T) { // quick.Check isn't guaranteed to produce an empty string
		assert.True(t, validate(t, ""))
	})

	t.Run("for arbitrary strings", func(t *testing.T) {
		randomStrings := func(s string) bool {
			return validate(t, s)
		}
		if err := quick.Check(randomStrings, nil); err != nil {
			t.Error(err)
		}
	})

	t.Run("for hex representations of arbitrary uint64 values", func(t *testing.T) {
		hexStrings := func(value uint64) bool {
			hexString := fmt.Sprintf("%x", value)
			return validate(t, strings.ToLower(hexString)) && validate(t, strings.ToUpper(hexString))
		}
		if err := quick.Check(hexStrings, nil); err != nil {
			t.Error(err)
		}
	})
}
