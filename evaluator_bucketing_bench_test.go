package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
)

func BenchmarkComputeBucketValue(b *testing.B) {
	benchCases := map[string]struct {
		withSeed bool
	}{
		"with seed":    {true},
		"without seed": {false},
	}

	for label, benchCase := range benchCases {
		b.Run(label, func(b *testing.B) {
			benchmarkComputeBucketValue(b, benchCase.withSeed)
		})
	}
}

func benchmarkComputeBucketValue(b *testing.B, withSeed bool) {
	context := ldcontext.New("userKeyA")
	evalScope := makeEvalScope(context)

	var seed ldvalue.OptionalInt
	if withSeed {
		seed = ldvalue.NewOptionalInt(42)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, evalBenchmarkErr = evalScope.computeBucketValue(false, seed, "", "hashKey", ldattr.NewNameRef("key"), "saltyA")
	}
}
