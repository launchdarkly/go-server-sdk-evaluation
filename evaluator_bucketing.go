package evaluation

import (
	"crypto/sha1" //nolint:gosec // SHA1 is cryptographically weak but we are not using it to hash any credentials
	"encoding/hex"
	"io"
	"strconv"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
)

const (
	longScale = float32(0xFFFFFFFFFFFFFFF)
)

func (es *evaluationScope) computeBucketValue(
	seed ldvalue.OptionalInt, contextKind ldcontext.Kind, key string, attr ldattr.Ref, salt string) (float32, error) {
	var prefix string
	if seed.IsDefined() {
		prefix = strconv.Itoa(seed.IntValue())
	} else {
		prefix = key + "." + salt
	}

	if !attr.IsDefined() {
		attr = ldattr.NewNameRef(ldattr.KeyAttr)
	} else if attr.Err() != nil {
		return 0, attr.Err()
	}
	context, ok := es.getApplicableContextByKind(contextKind)
	if !ok {
		return 0, nil
	}
	uValue, ok := context.GetValueForRef(attr)
	if !ok {
		return 0, nil
	}
	idHash, ok := bucketableStringValue(uValue)
	if !ok {
		return 0, nil
	}

	if secondary := es.context.Secondary(); secondary.IsDefined() {
		idHash = idHash + "." + secondary.StringValue()
	}

	h := sha1.New() // nolint:gas // just used for insecure hashing
	_, _ = io.WriteString(h, prefix+"."+idHash)
	hash := hex.EncodeToString(h.Sum(nil))[:15]

	intVal, _ := strconv.ParseInt(hash, 16, 64)

	bucket := float32(intVal) / longScale

	return bucket, nil
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
