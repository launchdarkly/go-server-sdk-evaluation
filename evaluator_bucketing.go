package evaluation

import (
	"crypto/sha1" //nolint:gosec // SHA1 is cryptographically weak but we are not using it to hash any credentials
	"encoding/hex"
	"io"
	"strconv"

	"github.com/launchdarkly/go-sdk-common/v3/ldattr"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
)

const (
	longScale = float32(0xFFFFFFFFFFFFFFF)
)

type bucketingFailureReason int

const (
	bucketingFailureInvalidAttrRef bucketingFailureReason = iota + 1 // 0 means no failure
	bucketingFailureContextLacksDesiredKind
	bucketingFailureAttributeNotFound
	bucketingFailureAttributeValueWrongType
)

// computeBucketValue is used for rollouts and experiments in flag rules, flag fallthroughs, and segment rules--
// anywhere a rollout/experiment can be. It implements the logic in the flag evaluation spec for computing a
// one-way hash from some combination of inputs related to the context and the flag or segment, and converting
// that hash into a percentage represented as a floating-point value in the range [0,1].
//
// The isExperiment parameter is true if this is an experiment rather than a plain rollout. Experiments can use
// the seed parameter in place of the context key and flag key; rollouts cannot. Rollouts can use the attr
// parameter to specify a context attribute other than the key, and can include a context's "secondary" key in
// the inputs; experiments cannot. Parameters that are irrelevant in either case are simply ignored.
//
// There are several conditions that could cause this computation to fail. The only one that causes an actual
// error value to be returned is if there is an invalid attribute reference, since that indicates malformed
// flag/segment data. For all other failure conditions, the method returns a zero bucket value, plus an enum
// indicating the type of failure (since these may have somewhat different consequences in different areas of
// evaluations).
func (es *evaluationScope) computeBucketValue(
	isExperiment bool,
	seed ldvalue.OptionalInt,
	contextKind ldcontext.Kind,
	key string,
	attr ldattr.Ref,
	salt string,
) (float32, bucketingFailureReason, error) {
	var prefix string
	if seed.IsDefined() {
		prefix = strconv.Itoa(seed.IntValue())
	} else {
		prefix = key + "." + salt
	}

	if isExperiment || !attr.IsDefined() { // always bucket by key in an experiment
		attr = ldattr.NewNameRef(ldattr.KeyAttr)
	} else if attr.Err() != nil {
		return 0, bucketingFailureInvalidAttrRef, attr.Err()
	}
	selectedContext, ok := getApplicableContextByKind(&es.context, contextKind)
	if !ok {
		return 0, bucketingFailureContextLacksDesiredKind, nil
	}
	uValue, ok := selectedContext.GetValueForRef(attr)
	if !ok {
		return 0, bucketingFailureAttributeNotFound, nil
	}
	idHash, ok := bucketableStringValue(uValue)
	if !ok {
		return 0, bucketingFailureAttributeValueWrongType, nil
	}

	if !isExperiment { // secondary key is not supported in experiments
		if secondary := selectedContext.Secondary(); secondary.IsDefined() {
			idHash = idHash + "." + secondary.StringValue()
		}
	}

	h := sha1.New() // nolint:gas // just used for insecure hashing
	_, _ = io.WriteString(h, prefix+"."+idHash)
	hash := hex.EncodeToString(h.Sum(nil))[:15]

	intVal, _ := strconv.ParseInt(hash, 16, 64)

	bucket := float32(intVal) / longScale

	return bucket, 0, nil
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
