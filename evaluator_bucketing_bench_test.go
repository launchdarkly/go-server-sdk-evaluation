package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
)

func BenchmarkBucketUser(b *testing.B) {
	benchCases := map[string]struct {
		withSeed bool
	}{
		"with seed":    {true},
		"without seed": {false},
	}

	for label, benchCase := range benchCases {
		b.Run(label, func(b *testing.B) {
			benchmarkBucketUser(b, benchCase.withSeed)
		})
	}
}

func benchmarkBucketUser(b *testing.B, withSeed bool) {
	u := lduser.NewUser("userKeyA")
	evalScope := userEvalScope(u)

	var seed ldvalue.OptionalInt
	if withSeed {
		seed = ldvalue.NewOptionalInt(42)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = evalScope.bucketUser(seed, "hashKey", ldattr.NewNameRef("key"), "saltyA")
	}
}
