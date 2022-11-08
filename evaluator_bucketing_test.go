package evaluation

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/launchdarkly/go-server-sdk-evaluation/v2/ldbuilders"
	"github.com/launchdarkly/go-server-sdk-evaluation/v2/ldmodel"

	"github.com/launchdarkly/go-sdk-common/v3/ldattr"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// See evaluator_bucketing_testdata_test.go for the values used in parameterized tests here.

var noSeed = ldvalue.OptionalInt{}

func makeEvalScope(context ldcontext.Context, evalOptions ...EvaluatorOption) *evaluationScope {
	evaluator := NewEvaluatorWithOptions(basicDataProvider(), evalOptions...).(*evaluator)
	return &evaluationScope{context: context, owner: evaluator}
}

func makeUserContextWithSecondaryKey(t *testing.T, key, secondary string) ldcontext.Context {
	// It is deliberately not possible to set a secondary key via the regular context builder API.
	// The secondary attribute can only be present when a context is parsed from user JSON in the
	// old schema.
	userJSON := ldvalue.ObjectBuild().SetString("key", key).SetString("secondary", secondary).Build().JSONString()
	var context ldcontext.Context
	err := json.Unmarshal([]byte(userJSON), &context)
	require.NoError(t, err)
	return context
}

func findBucketValueInVariationList(bucketValue float32, buckets []ldmodel.WeightedVariation) int {
	// This partially replicates logic in variationOrRolloutResult-- that's deliberate since we
	// want to make sure that logic doesn't change unintentionally
	bucketValueInt := int(bucketValue * 100000)
	cumulativeWeight := 0
	for i, bucket := range buckets {
		cumulativeWeight += bucket.Weight
		if bucketValueInt < cumulativeWeight {
			return i
		}
	}
	return len(buckets) - 1
}

func TestRolloutBucketing(t *testing.T) {
	buckets := []ldmodel.WeightedVariation{
		// The variation indices here are deliberately out of order so we can be sure it's really using that
		// field, rather than assuming it is the same as the index of the WeightedVariation array element.
		{Variation: 3, Weight: 20000}, // [0,20000)
		{Variation: 2, Weight: 20000}, // [20000,40000)
		{Variation: 1, Weight: 20000}, // [40000,60000)
		{Variation: 0, Weight: 40000}, // [60000,100000]
	}

	for _, defaultContextKind := range []bool{true, false} {
		// The purpose of running everything twice with defaultContext=true or false is to prove that
		// comnputeBucketValue is always checking the desired context kind whenever it gets attributes
		// from the context.

		t.Run(fmt.Sprintf("defaultContextKind=%t", defaultContextKind), func(t *testing.T) {
			baseRollout := ldmodel.Rollout{Variations: buckets}
			contextKind := ldcontext.DefaultKind

			checkResult := func(t *testing.T, p bucketingTestParams, context ldcontext.Context, rollout ldmodel.Rollout) {
				// For each of these test cases, we're doing two tests. First, we test the lower-level method
				// computeBucketValue, which tells the actual bucket value. Then, we test variationOrRolloutResult--
				// which also calls computeBucketValue, but we are verifying that variationOrRolloutResult then
				// applies the right logic to pick the result variation.

				if !defaultContextKind {
					context = ldcontext.NewMulti(
						ldcontext.NewWithKind("irrelevantKind", "irrelevantKey"),
						ldcontext.NewBuilderFromContext(context).Kind(contextKind).Build(),
					)
					rollout.ContextKind = contextKind
				}

				bucketValue, failReason, err := makeEvalScope(context).computeBucketValue(false, noSeed,
					rollout.ContextKind, p.flagOrSegmentKey, rollout.BucketBy, p.salt)
				assert.NoError(t, err)
				assert.Equal(t, bucketingFailureReason(0), failReason)
				assert.InEpsilon(t, p.expectedBucketValue, bucketValue, 0.0000001)

				variationIndex, inExperiment, err := makeEvalScope(context).variationOrRolloutResult(
					ldmodel.VariationOrRollout{Rollout: rollout},
					p.flagOrSegmentKey,
					p.salt,
				)
				assert.NoError(t, err)
				expectedBucket := findBucketValueInVariationList(p.expectedBucketValue, buckets)
				assert.Equal(t, buckets[expectedBucket].Variation, variationIndex)
				assert.False(t, inExperiment)
			}

			t.Run("by key", func(t *testing.T) {
				for _, p := range makeBucketingTestParams() {
					t.Run(p.description(), func(t *testing.T) {
						context := ldcontext.New(p.contextValue)
						checkResult(t, p, context, baseRollout)
					})
				}
			})

			t.Run("by custom string attribute", func(t *testing.T) {
				rollout := baseRollout
				rollout.BucketBy = ldattr.NewLiteralRef("attr1")

				for _, p := range makeBucketingTestParams() {
					t.Run(p.description(), func(t *testing.T) {
						context := ldcontext.NewBuilder(p.contextValue).SetString("attr1", p.contextValue).Build()
						checkResult(t, p, context, rollout)
					})
				}
			})

			t.Run("by custom int attribute", func(t *testing.T) {
				rollout := baseRollout
				rollout.BucketBy = ldattr.NewLiteralRef("attr1")

				for _, p := range makeBucketingTestParamsWithNumericStringValues() {
					t.Run(p.description(), func(t *testing.T) {
						n, err := strconv.Atoi(p.contextValue)
						require.NoError(t, err)
						context := ldcontext.NewBuilder(p.contextValue).SetInt("attr1", n).Build()
						checkResult(t, p, context, rollout)
					})
				}
			})

			t.Run("secondary key changes result if secondary is explicitly enabled", func(t *testing.T) {
				for _, p := range makeBucketingTestParams() {
					t.Run(p.description(), func(t *testing.T) {
						context1 := ldcontext.New(p.contextValue)
						context2 := makeUserContextWithSecondaryKey(t, p.contextValue, "some-secondary-key")

						evalScope1 := makeEvalScope(context1, EvaluatorOptionEnableSecondaryKey(true))
						bucketValue1, failReason, err := evalScope1.computeBucketValue(false, noSeed,
							"", p.flagOrSegmentKey, ldattr.Ref{}, p.salt)
						assert.NoError(t, err)
						assert.Equal(t, bucketingFailureReason(0), failReason)
						assert.InEpsilon(t, p.expectedBucketValue, bucketValue1, 0.0000001)

						evalScope2 := makeEvalScope(context2, EvaluatorOptionEnableSecondaryKey(true))
						bucketValue2, failReason, err := evalScope2.computeBucketValue(false, noSeed,
							"", p.flagOrSegmentKey, ldattr.Ref{}, p.salt)
						assert.NoError(t, err)
						assert.Equal(t, bucketingFailureReason(0), failReason)
						assert.NotEqual(t, bucketValue1, bucketValue2)
					})
				}
			})

			t.Run("secondary key does not change result if secondary is not explicitly enabled", func(t *testing.T) {
				for _, p := range makeBucketingTestParams() {
					t.Run(p.description(), func(t *testing.T) {
						context1 := ldcontext.New(p.contextValue)
						context2 := makeUserContextWithSecondaryKey(t, p.contextValue, "some-secondary-key")

						evalScope1 := makeEvalScope(context1)
						bucketValue1, failReason, err := evalScope1.computeBucketValue(false, noSeed,
							"", p.flagOrSegmentKey, ldattr.Ref{}, p.salt)
						assert.NoError(t, err)
						assert.Equal(t, bucketingFailureReason(0), failReason)
						assert.InEpsilon(t, p.expectedBucketValue, bucketValue1, 0.0000001)

						evalScope2 := makeEvalScope(context2)
						bucketValue2, failReason, err := evalScope2.computeBucketValue(false, noSeed,
							"", p.flagOrSegmentKey, ldattr.Ref{}, p.salt)
						assert.NoError(t, err)
						assert.Equal(t, bucketingFailureReason(0), failReason)
						assert.Equal(t, bucketValue1, bucketValue2)
					})
				}
			})
		})
	}
}

func TestExperimentBucketing(t *testing.T) {
	// seed here carefully chosen so users fall into different buckets
	buckets := []ldmodel.WeightedVariation{
		ldbuilders.Bucket(1, 10000),
		ldbuilders.Bucket(0, 20000),
		ldbuilders.BucketUntracked(0, 70000),
	}
	baseExperiment := ldmodel.Rollout{Kind: ldmodel.RolloutKindExperiment, Variations: buckets}

	// We won't check every permutation that was covered in TestRolloutBucketing - mostly areas where
	// the behavior of experiments is expected to be different from the behavior of rollouts.

	checkResult := func(t *testing.T, p bucketingTestParams, context ldcontext.Context, experiment ldmodel.Rollout) {
		// Here we enable the secondary key behavior just to verify that it will still *not*
		// be used in an experiment
		evalScope := makeEvalScope(context, EvaluatorOptionEnableSecondaryKey(true))

		bucketValue, failReason, err := evalScope.computeBucketValue(true, p.seed,
			experiment.ContextKind, p.flagOrSegmentKey, experiment.BucketBy, p.salt)
		assert.NoError(t, err)
		assert.Equal(t, bucketingFailureReason(0), failReason)
		assert.InEpsilon(t, p.expectedBucketValue, bucketValue, 0.0000001)

		experiment.Seed = p.seed
		variationIndex, inExperiment, err := evalScope.variationOrRolloutResult(
			ldmodel.VariationOrRollout{Rollout: experiment},
			p.flagOrSegmentKey,
			p.salt,
		)
		assert.NoError(t, err)
		expectedBucket := findBucketValueInVariationList(p.expectedBucketValue, buckets)
		assert.Equal(t, buckets[expectedBucket].Variation, variationIndex)
		assert.Equal(t, !buckets[expectedBucket].Untracked, inExperiment)
	}

	t.Run("by key", func(t *testing.T) {
		for _, p := range makeBucketingTestParamsForExperiments() {
			t.Run(p.description(), func(t *testing.T) {
				context := ldcontext.New(p.contextValue)
				checkResult(t, p, context, baseExperiment)
			})
		}
	})

	t.Run("changing hashKey and salt has no effect when seed is specified", func(t *testing.T) {
		for _, p := range makeBucketingTestParamsForExperiments() {
			if !p.seed.IsDefined() {
				continue
			}
			t.Run(p.description(), func(t *testing.T) {
				context := ldcontext.New(p.contextValue)

				modifiedParams1 := p
				modifiedParams1.flagOrSegmentKey += "xxx"
				checkResult(t, modifiedParams1, context, baseExperiment) // did not change expectedBucketValue, still passes

				modifiedParams2 := p
				modifiedParams2.salt += "yyy"
				checkResult(t, modifiedParams2, context, baseExperiment) // did not change expectedBucketValue, still passes
			})
		}
	})

	t.Run("changing seed produces different bucket value", func(t *testing.T) {
		for _, p := range makeBucketingTestParamsForExperiments() {
			t.Run(p.description(), func(t *testing.T) {
				context := ldcontext.New(p.contextValue)

				bucketValue1, failReason, err := makeEvalScope(context).computeBucketValue(true, p.seed,
					"", p.flagOrSegmentKey, ldattr.Ref{}, p.salt)
				assert.NoError(t, err)
				assert.Equal(t, bucketingFailureReason(0), failReason)

				var modifiedSeed ldvalue.OptionalInt
				if p.seed.IsDefined() {
					modifiedSeed = ldvalue.NewOptionalInt(p.seed.IntValue() + 1)
				} else {
					modifiedSeed = ldvalue.NewOptionalInt(999)
				}
				bucketValue2, failReason, err := makeEvalScope(context).computeBucketValue(true, modifiedSeed,
					"", p.flagOrSegmentKey, ldattr.Ref{}, p.salt)
				assert.NoError(t, err)
				assert.Equal(t, bucketingFailureReason(0), failReason)

				assert.NotEqual(t, bucketValue1, bucketValue2)
			})
		}
	})

	t.Run("when context kind is not found, first bucket is chosen but inExperiment is false", func(t *testing.T) {
		context := ldcontext.NewWithKind("rightkind", "key")
		experiment := baseExperiment

		variationIndex, inExperiment, err := makeEvalScope(context).variationOrRolloutResult(
			ldmodel.VariationOrRollout{Rollout: experiment},
			"flagkey",
			"salt",
		)
		assert.NoError(t, err)
		assert.Equal(t, baseExperiment.Variations[0].Variation, variationIndex)
		assert.False(t, inExperiment)
	})

	t.Run("secondary key is ignored, even if enabled in the evaluator", func(t *testing.T) {
		for _, p := range makeBucketingTestParamsForExperiments() {
			t.Run(p.description(), func(t *testing.T) {
				context := makeUserContextWithSecondaryKey(t, p.contextValue, "shouldbeignored")
				checkResult(t, p, context, baseExperiment) // did not change expectedBucketValue, still passes
			})
		}
	})

	t.Run("bucketBy is ignored", func(t *testing.T) {
		for _, p := range makeBucketingTestParamsForExperiments() {
			t.Run(p.description(), func(t *testing.T) {
				context := ldcontext.NewBuilder(p.contextValue).SetString("attr1", p.contextValue+"xyz").Build()
				experiment := baseExperiment
				experiment.BucketBy = ldattr.NewLiteralRef("attr1")

				checkResult(t, p, context, experiment) // did not change expectedBucketValue, still passes
			})
		}
	})
}

func TestVariationOrRolloutResultErrorConditions(t *testing.T) {
	context := ldcontext.New("key")

	for _, p := range []struct {
		rollout              ldmodel.Rollout
		expectedErrorMessage string
	}{
		{
			rollout:              ldmodel.Rollout{},
			expectedErrorMessage: "rollout or experiment with no variations",
		},
		{
			rollout:              ldmodel.Rollout{Kind: ldmodel.RolloutKindExperiment},
			expectedErrorMessage: "rollout or experiment with no variations",
		},
		{
			rollout: ldmodel.Rollout{
				BucketBy:   ldattr.NewRef("///"),
				Variations: []ldmodel.WeightedVariation{{Variation: 0, Weight: 100000}},
			},
			expectedErrorMessage: "attribute reference",
		},
	} {
		t.Run(p.expectedErrorMessage, func(t *testing.T) {
			vr := ldmodel.VariationOrRollout{Rollout: p.rollout}
			_, _, err := makeEvalScope(context).variationOrRolloutResult(vr, "hashKey", "salt")
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), p.expectedErrorMessage)
			}
		})
	}
}

func TestComputeBucketValueInvalidConditions(t *testing.T) {
	flagKey, salt := "flagKey", "saltyA" // irrelevant to these tests

	t.Run("single-kind context does not match desired kind", func(t *testing.T) {
		context := ldcontext.New("key")
		desiredKind := ldcontext.Kind("org")
		bucket, failReason, err := makeEvalScope(context).computeBucketValue(false, noSeed, desiredKind, flagKey, ldattr.Ref{}, "saltyA")
		assert.NoError(t, err)
		assert.Equal(t, bucketingFailureContextLacksDesiredKind, failReason)
		assert.Equal(t, float32(0), bucket)
	})

	t.Run("multi-kind context does not match desired kind", func(t *testing.T) {
		context := ldcontext.NewMulti(ldcontext.New("irrelevantKey1"), ldcontext.NewWithKind("irrelevantKind", "irrelevantKey2"))
		desiredKind := ldcontext.Kind("org")
		bucket, failReason, err := makeEvalScope(context).computeBucketValue(false, noSeed, desiredKind, flagKey, ldattr.Ref{}, "saltyA")
		assert.NoError(t, err)
		assert.Equal(t, bucketingFailureContextLacksDesiredKind, failReason)
		assert.Equal(t, float32(0), bucket)
	})

	t.Run("bucket by nonexistent attribute", func(t *testing.T) {
		context := ldcontext.New("key")
		bucket, failReason, err := makeEvalScope(context).computeBucketValue(false, noSeed, "", flagKey, ldattr.NewLiteralRef("unknownAttr"), salt)
		assert.NoError(t, err)
		assert.Equal(t, bucketingFailureAttributeNotFound, failReason)
		assert.Equal(t, float32(0), bucket)
	})

	t.Run("bucket by non-integer numeric attribute", func(t *testing.T) {
		context := ldcontext.NewBuilder("key").SetFloat64("floatAttr", 999.999).Build()
		bucket, failReason, err := makeEvalScope(context).computeBucketValue(false, noSeed, "", flagKey, ldattr.NewLiteralRef("floatAttr"), salt)
		assert.NoError(t, err)
		assert.Equal(t, bucketingFailureAttributeValueWrongType, failReason)
		assert.Equal(t, float32(0), bucket)
	})

	t.Run("bucket by invalid attribute reference", func(t *testing.T) {
		context := ldcontext.New("key")
		badAttr := ldattr.NewRef("///")
		_, failReason, err := makeEvalScope(context).computeBucketValue(false, noSeed, "", flagKey, badAttr, salt)
		assert.Error(t, err) // Unlike the other invalid conditions, we treat this one as a malformed flag error
		assert.Equal(t, bucketingFailureInvalidAttrRef, failReason)
	})
}

func TestBucketValueBeyondLastBucketIsPinnedToLastBucket(t *testing.T) {
	vr := ldbuilders.Rollout(ldbuilders.Bucket(0, 5000), ldbuilders.Bucket(1, 5000))
	user := ldcontext.NewBuilder("userKeyD").SetInt("intAttr", 99999).Build()
	variationIndex, inExperiment, err := makeEvalScope(user).variationOrRolloutResult(vr, "hashKey", "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, 1, variationIndex)
	assert.False(t, inExperiment)
}

func TestBucketValueBeyondLastBucketIsPinnedToLastBucketForExperiment(t *testing.T) {
	vr := ldbuilders.Experiment(ldvalue.NewOptionalInt(42), ldbuilders.Bucket(0, 5000), ldbuilders.Bucket(1, 5000))
	user := ldcontext.NewBuilder("userKeyD").SetInt("intAttr", 99999).Build()
	variationIndex, inExperiment, err := makeEvalScope(user).variationOrRolloutResult(vr, "hashKey", "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, 1, variationIndex)
	assert.True(t, inExperiment)
}
