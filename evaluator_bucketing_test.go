package evaluation

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldbuilders"
)

var noSeed = ldvalue.OptionalInt{}

func userEvalScope(context ldcontext.Context) *evaluationScope {
	return &evaluationScope{context: context}
}

func TestVariationIndexForUser(t *testing.T) {
	vr := ldbuilders.Rollout(ldbuilders.Bucket(0, 60000), ldbuilders.Bucket(1, 40000))

	u1 := lduser.NewUser("userKeyA")
	variationIndex, inExperiment := userEvalScope(u1).variationIndexForUser(vr, "hashKey", "saltyA")
	assert.Equal(t, ldvalue.NewOptionalInt(0), variationIndex)
	assert.False(t, inExperiment)

	u2 := lduser.NewUser("userKeyB")
	variationIndex, inExperiment = userEvalScope(u2).variationIndexForUser(vr, "hashKey", "saltyA")
	assert.Equal(t, ldvalue.NewOptionalInt(1), variationIndex)
	assert.False(t, inExperiment)

	u3 := lduser.NewUser("userKeyC")
	variationIndex, inExperiment = userEvalScope(u3).variationIndexForUser(vr, "hashKey", "saltyA")
	assert.Equal(t, ldvalue.NewOptionalInt(0), variationIndex)
	assert.False(t, inExperiment)
}

func TestVariationIndexForUserWithCustomAttribute(t *testing.T) {
	vr := ldbuilders.Rollout(ldbuilders.Bucket(0, 60000), ldbuilders.Bucket(1, 40000))
	vr.Rollout.BucketBy = ldattr.NewNameRef("intAttr")

	u1 := lduser.NewUserBuilder("userKeyD").Custom("intAttr", ldvalue.Int(33333)).Build()
	variationIndex, inExperiment := userEvalScope(u1).variationIndexForUser(vr, "hashKey", "saltyA")
	assert.Equal(t, ldvalue.NewOptionalInt(0), variationIndex) // bucketValue = 0.54771423
	assert.False(t, inExperiment)

	u2 := lduser.NewUserBuilder("userKeyD").Custom("intAttr", ldvalue.Int(99999)).Build()
	variationIndex, inExperiment = userEvalScope(u2).variationIndexForUser(vr, "hashKey", "saltyA")
	assert.Equal(t, ldvalue.NewOptionalInt(1), variationIndex) // bucketValue = 0.7309658
	assert.False(t, inExperiment)
}

func TestVariationIndexForUserInExperiment(t *testing.T) {
	// seed here carefully chosen so users fall into different buckets
	vr := ldbuilders.Experiment(61, ldbuilders.Bucket(0, 10000), ldbuilders.Bucket(1, 20000), ldbuilders.BucketUntracked(0, 70000))

	u1 := lduser.NewUser("userKeyA")
	variationIndex, inExperiment := userEvalScope(u1).variationIndexForUser(vr, "hashKey", "saltyA")
	// bucketVal = 0.09801207
	assert.Equal(t, ldvalue.NewOptionalInt(0), variationIndex)
	assert.True(t, inExperiment)

	u2 := lduser.NewUser("userKeyB")
	variationIndex, inExperiment = userEvalScope(u2).variationIndexForUser(vr, "hashKey", "saltyA")
	// bucketVal = 0.14483777
	assert.Equal(t, ldvalue.NewOptionalInt(1), variationIndex)
	assert.True(t, inExperiment)

	u3 := lduser.NewUser("userKeyC")
	variationIndex, inExperiment = userEvalScope(u3).variationIndexForUser(vr, "hashKey", "saltyA")
	// bucketVal = 0.9242641
	assert.Equal(t, ldvalue.NewOptionalInt(0), variationIndex)
	assert.False(t, inExperiment)
}

func TestBucketUserByKey(t *testing.T) {
	u1 := lduser.NewUser("userKeyA")
	bucket1 := userEvalScope(u1).bucketUser(noSeed, "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.InEpsilon(t, 0.42157587, bucket1, 0.0000001)

	u2 := lduser.NewUser("userKeyB")
	bucket2 := userEvalScope(u2).bucketUser(noSeed, "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.InEpsilon(t, 0.6708485, bucket2, 0.0000001)

	u3 := lduser.NewUser("userKeyC")
	bucket3 := userEvalScope(u3).bucketUser(noSeed, "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.InEpsilon(t, 0.10343106, bucket3, 0.0000001)
}

func TestBucketUserWithSeed(t *testing.T) {
	seed := ldvalue.NewOptionalInt(61)

	u1 := lduser.NewUser("userKeyA")
	bucket1 := userEvalScope(u1).bucketUser(seed, "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.InEpsilon(t, 0.09801207, bucket1, 0.0000001)

	u2 := lduser.NewUser("userKeyB")
	bucket2 := userEvalScope(u2).bucketUser(seed, "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.InEpsilon(t, 0.14483777, bucket2, 0.0000001)

	u3 := lduser.NewUser("userKeyC")
	bucket3 := userEvalScope(u3).bucketUser(seed, "hashKey", ldattr.NewNameRef("key"), "saltyA")
	assert.InEpsilon(t, 0.9242641, bucket3, 0.0000001)

	t.Run("changing hashKey and salt has no effect when seed is specified", func(t *testing.T) {
		bucket1DifferentHashKeySalt := userEvalScope(u1).bucketUser(seed, "otherHashKey", ldattr.NewNameRef("key"), "otherSaltyA")
		assert.InEpsilon(t, bucket1, bucket1DifferentHashKeySalt, 0.0000001)
	})

	t.Run("changing seed produces different bucket value", func(t *testing.T) {
		otherSeed := ldvalue.NewOptionalInt(60)
		bucket1DifferentSeed := userEvalScope(u1).bucketUser(otherSeed, "hashKey", ldattr.NewNameRef("key"), "saltyA")
		assert.InEpsilon(t, 0.7008816, bucket1DifferentSeed, 0.0000001)
	})
}

func TestBucketUserWithSecondaryKey(t *testing.T) {
	u1 := lduser.NewUser("userKey")
	u2 := lduser.NewUserBuilder("userKey").Secondary("mySecondaryKey").Build()
	bucket1 := userEvalScope(u1).bucketUser(noSeed, "hashKey", ldattr.NewNameRef("key"), "salt")
	bucket2 := userEvalScope(u2).bucketUser(noSeed, "hashKey", ldattr.NewNameRef("key"), "salt")
	assert.NotEqual(t, bucket1, bucket2)
}

func TestBucketUserByIntAttr(t *testing.T) {
	user := lduser.NewUserBuilder("userKeyD").Custom("intAttr", ldvalue.Int(33333)).Build()
	bucket := userEvalScope(user).bucketUser(noSeed, "hashKey", ldattr.NewNameRef("intAttr"), "saltyA")
	assert.InEpsilon(t, 0.54771423, bucket, 0.0000001)

	user = lduser.NewUserBuilder("userKeyD").Custom("stringAttr", ldvalue.String("33333")).Build()
	bucket2 := userEvalScope(user).bucketUser(noSeed, "hashKey", ldattr.NewNameRef("stringAttr"), "saltyA")
	assert.InEpsilon(t, bucket, bucket2, 0.0000001)
}

func TestBucketUserByFloatAttrNotAllowed(t *testing.T) {
	user := lduser.NewUserBuilder("userKeyE").Custom("floatAttr", ldvalue.Float64(999.999)).Build()
	bucket := userEvalScope(user).bucketUser(noSeed, "hashKey", ldattr.NewNameRef("floatAttr"), "saltyA")
	assert.InDelta(t, 0.0, bucket, 0.0000001)
}

func TestBucketUserByFloatAttrThatIsReallyAnIntIsAllowed(t *testing.T) {
	user := lduser.NewUserBuilder("userKeyE").Custom("floatAttr", ldvalue.Float64(33333)).Build()
	bucket := userEvalScope(user).bucketUser(noSeed, "hashKey", ldattr.NewNameRef("floatAttr"), "saltyA")
	assert.InEpsilon(t, 0.54771423, bucket, 0.0000001)
}

func TestBucketValueBeyondLastBucketIsPinnedToLastBucket(t *testing.T) {
	vr := ldbuilders.Rollout(ldbuilders.Bucket(0, 5000), ldbuilders.Bucket(1, 5000))
	user := lduser.NewUserBuilder("userKeyD").Custom("intAttr", ldvalue.Int(99999)).Build()
	variationIndex, inExperiment := userEvalScope(user).variationIndexForUser(vr, "hashKey", "saltyA")
	assert.Equal(t, ldvalue.NewOptionalInt(1), variationIndex)
	assert.False(t, inExperiment)
}

func TestBucketValueBeyondLastBucketIsPinnedToLastBucketForExperiment(t *testing.T) {
	vr := ldbuilders.Experiment(42, ldbuilders.Bucket(0, 5000), ldbuilders.Bucket(1, 5000))
	user := lduser.NewUserBuilder("userKeyD").Custom("intAttr", ldvalue.Int(99999)).Build()
	variationIndex, inExperiment := userEvalScope(user).variationIndexForUser(vr, "hashKey", "saltyA")
	assert.Equal(t, ldvalue.NewOptionalInt(1), variationIndex)
	assert.True(t, inExperiment)
}
