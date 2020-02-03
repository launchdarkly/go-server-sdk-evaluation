package evaluation

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"strconv"

	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

func bucketUser(user *lduser.User, key string, attr lduser.UserAttribute, salt string) float32 {
	uValue := user.GetAttribute(attr)
	idHash, ok := bucketableStringValue(uValue)
	if !ok {
		return 0
	}

	if secondary := user.GetSecondaryKey(); secondary.IsDefined() {
		idHash = idHash + "." + secondary.StringValue()
	}

	h := sha1.New() // nolint:gas // just used for insecure hashing
	_, _ = io.WriteString(h, key+"."+salt+"."+idHash)
	hash := hex.EncodeToString(h.Sum(nil))[:15]

	intVal, _ := strconv.ParseInt(hash, 16, 64)

	bucket := float32(intVal) / longScale

	return bucket
}

func bucketableStringValue(uValue ldvalue.Value) (string, bool) {
	if uValue.Type() == ldvalue.StringType {
		return uValue.StringValue(), true
	}
	if uValue.IsInt() {
		return strconv.Itoa(uValue.IntValue()), true
	}
	return "", false
}
