package evaluation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldbuilders"
)

func TestVariationIndexForUser(t *testing.T) {
	vr := ldbuilders.Rollout(ldbuilders.Bucket(0, 60000), ldbuilders.Bucket(1, 40000))

	u1 := lduser.NewUser("userKeyA")
	variationIndex := variationIndexForUser(vr, &u1, "hashKey", "saltyA")
	assert.NotNil(t, variationIndex)
	assert.Equal(t, 0, *variationIndex)

	u2 := lduser.NewUser("userKeyB")
	variationIndex = variationIndexForUser(vr, &u2, "hashKey", "saltyA")
	assert.NotNil(t, variationIndex)
	assert.Equal(t, 1, *variationIndex)

	u3 := lduser.NewUser("userKeyC")
	variationIndex = variationIndexForUser(vr, &u3, "hashKey", "saltyA")
	assert.NotNil(t, variationIndex)
	assert.Equal(t, 0, *variationIndex)
}

func TestBucketUserByKey(t *testing.T) {
	u1 := lduser.NewUser("userKeyA")
	bucket1 := bucketUser(&u1, "hashKey", "key", "saltyA")
	assert.InEpsilon(t, 0.42157587, bucket1, 0.0000001)

	u2 := lduser.NewUser("userKeyB")
	bucket2 := bucketUser(&u2, "hashKey", "key", "saltyA")
	assert.InEpsilon(t, 0.6708485, bucket2, 0.0000001)

	u3 := lduser.NewUser("userKeyC")
	bucket3 := bucketUser(&u3, "hashKey", "key", "saltyA")
	assert.InEpsilon(t, 0.10343106, bucket3, 0.0000001)
}

func TestBucketUserWithSecondaryKey(t *testing.T) {
	u1 := lduser.NewUser("userKey")
	u2 := lduser.NewUserBuilder("userKey").Secondary("mySecondaryKey").Build()
	bucket1 := bucketUser(&u1, "hashKey", lduser.KeyAttribute, "salt")
	bucket2 := bucketUser(&u2, "hashKey", lduser.KeyAttribute, "salt")
	assert.NotEqual(t, bucket1, bucket2)
}

func TestBucketUserByIntAttr(t *testing.T) {
	user := lduser.NewUserBuilder("userKeyD").Custom("intAttr", ldvalue.Int(33333)).Build()
	bucket := bucketUser(&user, "hashKey", "intAttr", "saltyA")
	assert.InEpsilon(t, 0.54771423, bucket, 0.0000001)

	user = lduser.NewUserBuilder("userKeyD").Custom("stringAttr", ldvalue.String("33333")).Build()
	bucket2 := bucketUser(&user, "hashKey", "stringAttr", "saltyA")
	assert.InEpsilon(t, bucket, bucket2, 0.0000001)
}

func TestBucketUserByFloatAttrNotAllowed(t *testing.T) {
	user := lduser.NewUserBuilder("userKeyE").Custom("floatAttr", ldvalue.Float64(999.999)).Build()
	bucket := bucketUser(&user, "hashKey", "floatAttr", "saltyA")
	assert.InDelta(t, 0.0, bucket, 0.0000001)
}

func TestBucketUserByFloatAttrThatIsReallyAnIntIsAllowed(t *testing.T) {
	user := lduser.NewUserBuilder("userKeyE").Custom("floatAttr", ldvalue.Float64(33333)).Build()
	bucket := bucketUser(&user, "hashKey", "floatAttr", "saltyA")
	assert.InEpsilon(t, 0.54771423, bucket, 0.0000001)
}
