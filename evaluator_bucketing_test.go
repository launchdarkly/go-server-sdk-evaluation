package evaluation

import (
	"testing"

	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldmodel"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"

	"github.com/stretchr/testify/assert"
)

var noSeed = ldvalue.OptionalInt{}

func userEvalScope(context ldcontext.Context) *evaluationScope {
	return &evaluationScope{context: context}
}

func TestVariationIndexForUser(t *testing.T) {
	vr := ldbuilders.Rollout(ldbuilders.Bucket(0, 60000), ldbuilders.Bucket(1, 40000))

	c1 := ldcontext.New("userKeyA")
	variationIndex, inExperiment, err := userEvalScope(c1).variationOrRolloutResult(vr, "hashKey", "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, 0, variationIndex)
	assert.False(t, inExperiment)

	c2 := ldcontext.New("userKeyB")
	variationIndex, inExperiment, err = userEvalScope(c2).variationOrRolloutResult(vr, "hashKey", "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, 1, variationIndex)
	assert.False(t, inExperiment)

	c3 := ldcontext.New("userKeyC")
	variationIndex, inExperiment, err = userEvalScope(c3).variationOrRolloutResult(vr, "hashKey", "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, 0, variationIndex)
	assert.False(t, inExperiment)
}

func TestVariationIndexForUserWithCustomAttribute(t *testing.T) {
	vr := ldbuilders.Rollout(ldbuilders.Bucket(0, 60000), ldbuilders.Bucket(1, 40000))
	vr.Rollout.BucketBy = ldattr.NewNameRef("intAttr")

	c1 := ldcontext.NewBuilder("userKeyD").SetInt("intAttr", 33333).Build()
	variationIndex, inExperiment, err := userEvalScope(c1).variationOrRolloutResult(vr, "hashKey", "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, 0, variationIndex) // bucketValue = 0.54771423
	assert.False(t, inExperiment)

	c2 := ldcontext.NewBuilder("userKeyD").SetInt("intAttr", 99999).Build()
	variationIndex, inExperiment, err = userEvalScope(c2).variationOrRolloutResult(vr, "hashKey", "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, 1, variationIndex) // bucketValue = 0.7309658
	assert.False(t, inExperiment)
}

func TestVariationIndexForUserInExperiment(t *testing.T) {
	// seed here carefully chosen so users fall into different buckets
	vr := ldbuilders.Experiment(61, ldbuilders.Bucket(0, 10000), ldbuilders.Bucket(1, 20000), ldbuilders.BucketUntracked(0, 70000))

	c1 := ldcontext.New("userKeyA")
	variationIndex, inExperiment, err := userEvalScope(c1).variationOrRolloutResult(vr, "hashKey", "saltyA")
	// bucketVal = 0.09801207
	assert.NoError(t, err)
	assert.Equal(t, 0, variationIndex)
	assert.True(t, inExperiment)

	c2 := ldcontext.New("userKeyB")
	variationIndex, inExperiment, err = userEvalScope(c2).variationOrRolloutResult(vr, "hashKey", "saltyA")
	// bucketVal = 0.14483777
	assert.NoError(t, err)
	assert.Equal(t, 1, variationIndex)
	assert.True(t, inExperiment)

	c3 := ldcontext.New("userKeyC")
	variationIndex, inExperiment, err = userEvalScope(c3).variationOrRolloutResult(vr, "hashKey", "saltyA")
	// bucketVal = 0.9242641
	assert.NoError(t, err)
	assert.Equal(t, 0, variationIndex)
	assert.False(t, inExperiment)
}

func TestVariationIndexForUserErrorConditions(t *testing.T) {
	user := ldcontext.New("key")

	vr1 := ldmodel.VariationOrRollout{
		Rollout: ldmodel.Rollout{},
	}
	_, _, err := userEvalScope(user).variationOrRolloutResult(vr1, "hashKey", "salt")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "rollout or experiment with no variations")
	}

	vr2 := ldmodel.VariationOrRollout{
		Rollout: ldmodel.Rollout{
			BucketBy:   ldattr.NewRef("///"),
			Variations: []ldmodel.WeightedVariation{{Variation: 0, Weight: 100000}},
		},
	}
	_, _, err = userEvalScope(user).variationOrRolloutResult(vr2, "hashKey", "salt")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "attribute reference")
	}
}

func TestComputeBucketValueByKey(t *testing.T) {
	c1 := ldcontext.New("userKeyA")
	bucket1, err := userEvalScope(c1).computeBucketValue(noSeed, "", "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.NoError(t, err)
	assert.InEpsilon(t, 0.42157587, bucket1, 0.0000001)

	bucket1a, err := userEvalScope(c1).computeBucketValue(noSeed, "", "hashKey", ldattr.Ref{}, "saltyA") // defaults to key
	assert.NoError(t, err)
	assert.Equal(t, bucket1, bucket1a)

	c2 := ldcontext.New("userKeyB")
	bucket2, err := userEvalScope(c2).computeBucketValue(noSeed, "", "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.NoError(t, err)
	assert.InEpsilon(t, 0.6708485, bucket2, 0.0000001)

	c3 := ldcontext.New("userKeyC")
	bucket3, err := userEvalScope(c3).computeBucketValue(noSeed, "", "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.NoError(t, err)
	assert.InEpsilon(t, 0.10343106, bucket3, 0.0000001)
}

func TestComputeBucketValueByKeyForSpecificKind(t *testing.T) {
	user := ldcontext.New("otherKey")
	org := ldcontext.NewWithKind("org", "userKeyA")
	multi := ldcontext.NewMulti(user, org)

	bucket1, err := userEvalScope(org).computeBucketValue(noSeed, "org", "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.NoError(t, err)
	assert.InEpsilon(t, 0.42157587, bucket1, 0.0000001) // should match answer for userKeyA in previous test

	bucket2, err := userEvalScope(multi).computeBucketValue(noSeed, "org", "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, bucket1, bucket2)
}

func TestComputeBucketValueWithSeed(t *testing.T) {
	seed := ldvalue.NewOptionalInt(61)

	c1 := ldcontext.New("userKeyA")
	bucket1, err := userEvalScope(c1).computeBucketValue(seed, "", "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.NoError(t, err)
	assert.InEpsilon(t, 0.09801207, bucket1, 0.0000001)

	c2 := ldcontext.New("userKeyB")
	bucket2, err := userEvalScope(c2).computeBucketValue(seed, "", "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.NoError(t, err)
	assert.InEpsilon(t, 0.14483777, bucket2, 0.0000001)

	c3 := ldcontext.New("userKeyC")
	bucket3, err := userEvalScope(c3).computeBucketValue(seed, "", "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.NoError(t, err)
	assert.InEpsilon(t, 0.9242641, bucket3, 0.0000001)

	t.Run("changing hashKey and salt has no effect when seed is specified", func(t *testing.T) {
		bucket1DifferentHashKeySalt, err := userEvalScope(c1).computeBucketValue(seed, "", "otherHashKey", ldattr.NewNameRef("key"), "otherSaltyA")
		assert.NoError(t, err)
		assert.InEpsilon(t, bucket1, bucket1DifferentHashKeySalt, 0.0000001)
	})

	t.Run("changing seed produces different bucket value", func(t *testing.T) {
		otherSeed := ldvalue.NewOptionalInt(60)
		bucket1DifferentSeed, err := userEvalScope(c1).computeBucketValue(otherSeed, "", "hashKey", ldattr.NewNameRef("key"), "saltyA")
		assert.NoError(t, err)
		assert.InEpsilon(t, 0.7008816, bucket1DifferentSeed, 0.0000001)
	})
}

func TestComputeBucketValueWithSecondaryKey(t *testing.T) {
	c1 := ldcontext.New("userKey")
	c2 := ldcontext.NewBuilder("userKey").Secondary("mySecondaryKey").Build()
	bucket1, err := userEvalScope(c1).computeBucketValue(noSeed, "", "hashKey", ldattr.NewNameRef("key"), "salt")
	assert.NoError(t, err)
	bucket2, err := userEvalScope(c2).computeBucketValue(noSeed, "", "hashKey", ldattr.NewNameRef("key"), "salt")
	assert.NoError(t, err)
	assert.NotEqual(t, bucket1, bucket2)
}

func TestComputeBucketValueWithSecondaryKeyForSpecificKind(t *testing.T) {
	other := ldcontext.NewWithKind("other", "someKey")
	org := ldcontext.NewBuilder("someKey").Kind("org").Secondary("mySecondaryKey").Build()
	multi := ldcontext.NewMulti(other, org)
	bucket1, err := userEvalScope(org).computeBucketValue(noSeed, "org", "hashKey", ldattr.NewNameRef("key"), "salt")
	assert.NoError(t, err)
	bucket2, err := userEvalScope(multi).computeBucketValue(noSeed, "org", "hashKey", ldattr.NewNameRef("key"), "salt")
	assert.NoError(t, err)
	assert.Equal(t, bucket1, bucket2)
}

func TestComputeBucketValueByIntAttr(t *testing.T) {
	user := ldcontext.NewBuilder("userKeyD").SetInt("intAttr", 33333).Build()
	bucket, err := userEvalScope(user).computeBucketValue(noSeed, "", "hashKey", ldattr.NewNameRef("intAttr"), "saltyA")
	assert.NoError(t, err)
	assert.InEpsilon(t, 0.54771423, bucket, 0.0000001)

	user = ldcontext.NewBuilder("userKeyD").SetString("stringAttr", "33333").Build()
	bucket2, err := userEvalScope(user).computeBucketValue(noSeed, "", "hashKey", ldattr.NewNameRef("stringAttr"), "saltyA")
	assert.NoError(t, err)
	assert.InEpsilon(t, bucket, bucket2, 0.0000001)
}

func TestComputeBucketValueByFloatAttrNotAllowed(t *testing.T) {
	user := ldcontext.NewBuilder("userKeyE").SetFloat64("floatAttr", 999.999).Build()
	bucket, err := userEvalScope(user).computeBucketValue(noSeed, "", "hashKey", ldattr.NewNameRef("floatAttr"), "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, float32(0), bucket)
}

func TestComputeBucketValueByFloatAttrThatIsReallyAnIntIsAllowed(t *testing.T) {
	user := ldcontext.NewBuilder("userKeyE").SetFloat64("floatAttr", 33333).Build()
	bucket, err := userEvalScope(user).computeBucketValue(noSeed, "", "hashKey", ldattr.NewNameRef("floatAttr"), "saltyA")
	assert.NoError(t, err)
	assert.InEpsilon(t, 0.54771423, bucket, 0.0000001)
}

func TestComputeBucketValueByUnknownAttr(t *testing.T) {
	user := ldcontext.NewBuilder("userKeyE").Build()
	bucket, err := userEvalScope(user).computeBucketValue(noSeed, "", "hashKey", ldattr.NewNameRef("unknownAttr"), "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, float32(0), bucket)
}

func TestComputeBucketValueWithUnknownKind(t *testing.T) {
	user := ldcontext.New("otherKey")
	org := ldcontext.NewWithKind("org", "userKeyA")
	multi := ldcontext.NewMulti(user, org)

	bucket1, err := userEvalScope(org).computeBucketValue(noSeed, "user", "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, float32(0), bucket1)

	bucket2, err := userEvalScope(multi).computeBucketValue(noSeed, "other", "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, float32(0), bucket2)
}

func TestComputeBucketValueInvalidRef(t *testing.T) {
	c1 := ldcontext.New("userKeyA")
	_, err := userEvalScope(c1).computeBucketValue(noSeed, "", "hashKey", ldattr.NewRef("///"), "saltyA")
	assert.Error(t, err)
}

func TestBucketValueBeyondLastBucketIsPinnedToLastBucket(t *testing.T) {
	vr := ldbuilders.Rollout(ldbuilders.Bucket(0, 5000), ldbuilders.Bucket(1, 5000))
	user := ldcontext.NewBuilder("userKeyD").SetInt("intAttr", 99999).Build()
	variationIndex, inExperiment, err := userEvalScope(user).variationOrRolloutResult(vr, "hashKey", "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, 1, variationIndex)
	assert.False(t, inExperiment)
}

func TestBucketValueBeyondLastBucketIsPinnedToLastBucketForExperiment(t *testing.T) {
	vr := ldbuilders.Experiment(42, ldbuilders.Bucket(0, 5000), ldbuilders.Bucket(1, 5000))
	user := ldcontext.NewBuilder("userKeyD").SetInt("intAttr", 99999).Build()
	variationIndex, inExperiment, err := userEvalScope(user).variationOrRolloutResult(vr, "hashKey", "saltyA")
	assert.NoError(t, err)
	assert.Equal(t, 1, variationIndex)
	assert.True(t, inExperiment)
}

func TestEmptyExperimenttIsError(t *testing.T) {
	vr := ldbuilders.Experiment(42)
	user := ldcontext.NewBuilder("userKeyD").SetInt("intAttr", 99999).Build()
	_, _, err := userEvalScope(user).variationOrRolloutResult(vr, "hashKey", "saltyA")
	assert.Error(t, err)
}
