package evaluation

import (
	"fmt"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldmodel"
)

func makeBigSegmentRef(s *ldmodel.Segment) string {
	// The format of big segment references is independent of what store implementation is being
	// used; the store implementation receives only this string and does not know the details of
	// the data model. The Relay Proxy will use the same format when writing to the store.
	return fmt.Sprintf("%s.g%d", s.Key, s.Generation.IntValue())
}

func (es *evaluationScope) segmentContainsUser(s *ldmodel.Segment) bool {
	userKey := es.user.GetKey()

	// Check if the user is specifically included in or excluded from the segment by key
	if s.Unbounded {
		if !s.Generation.IsDefined() {
			// Big segment queries can only be done if the generation is known. If it's unset,
			// that probably means the data store was populated by an older SDK that doesn't know
			// about the Generation property and therefore dropped it from the JSON data. We'll treat
			// that as a "not configured" condition.
			es.bigSegmentsReferenced = true
			es.bigSegmentsStatus = ldreason.BigSegmentsNotConfigured
			return false
		}
		// Even if multiple big segments are referenced within a single flag evaluation,
		// we only need to do this query once, since it returns *all* of the user's segment
		// memberships.
		if !es.bigSegmentsReferenced {
			es.bigSegmentsReferenced = true
			if es.owner.bigSegmentProvider == nil {
				// If the provider is nil, that means the SDK hasn't been configured to be able to
				// use big segments.
				es.bigSegmentsStatus = ldreason.BigSegmentsNotConfigured
			} else {
				es.bigSegmentsMembership, es.bigSegmentsStatus =
					es.owner.bigSegmentProvider.GetUserMembership(userKey)
			}
		}
		if es.bigSegmentsMembership == nil {
			return false
		}
		included := es.bigSegmentsMembership.CheckMembership(makeBigSegmentRef(s))
		if included.IsDefined() {
			return included.BoolValue()
		}
	} else if included, found := ldmodel.SegmentIncludesOrExcludesKey(s, userKey); found {
		return included
	}

	// Check if any of the segment rules match
	for _, rule := range s.Rules {
		// Note, taking address of range variable here is OK because it's not used outside the loop
		if es.segmentRuleMatchesUser(&rule, s.Key, s.Salt) { //nolint:gosec // see comment above
			return true
		}
	}

	return false
}

func (es *evaluationScope) segmentRuleMatchesUser(r *ldmodel.SegmentRule, key, salt string) bool {
	// Note that r is passed by reference only for efficiency; we do not modify it
	for _, clause := range r.Clauses {
		c := clause
		if !ldmodel.ClauseMatchesUser(&c, &es.user) {
			return false
		}
	}

	// If the Weight is absent, this rule matches
	if r.Weight < 0 {
		return true
	}

	// All of the clauses are met. Check to see if the user buckets in
	bucketBy := lduser.KeyAttribute
	if r.BucketBy != "" {
		bucketBy = r.BucketBy
	}

	// Check whether the user buckets into the segment
	bucket := es.bucketUser(ldvalue.OptionalInt{}, key, bucketBy, salt)
	weight := float32(r.Weight) / 100000.0

	return bucket < weight
}
