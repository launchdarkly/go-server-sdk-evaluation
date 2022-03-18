package evaluation

import (
	"testing"

	"github.com/launchdarkly/go-sdk-common/v3/ldattr"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
)

func BenchmarkComputeBucketValueNoAlloc(b *testing.B) {
	for _, p := range []struct {
		name       string
		secondary  string
		customAttr ldvalue.Value
		seed       ldvalue.OptionalInt
	}{
		{
			name: "simple",
		},
		{
			name:      "with secondary",
			secondary: "123",
		},
		{
			name: "with seed",
			seed: ldvalue.NewOptionalInt(42),
		},
		{
			name:       "bucket by custom attr string",
			customAttr: ldvalue.String("xyz"),
		},
		{
			name:       "bucket by custom attr int",
			customAttr: ldvalue.Int(123),
		},
	} {
		b.Run(p.name, func(b *testing.B) {
			builder := ldcontext.NewBuilder("userKey")
			if p.secondary != "" {
				builder.Secondary(p.secondary)
			}
			if p.customAttr.IsDefined() {
				builder.SetValue("attr1", p.customAttr)
			}
			context := builder.Build()
			evalScope := makeEvalScope(context)

			var bucketBy ldattr.Ref
			if p.customAttr.IsDefined() {
				bucketBy = ldattr.NewNameRef("attr1")
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _, evalBenchmarkErr = evalScope.computeBucketValue(false, p.seed, "", "hashKey", bucketBy, "saltyA")
			}
		})
	}
}
