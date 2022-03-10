package evaluation

import (
	"fmt"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldreason"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v2/ldmodel"
)

func makeBigSegmentRef(s *ldmodel.Segment) string {
	// The format of big segment references is independent of what store implementation is being
	// used; the store implementation receives only this string and does not know the details of
	// the data model. The Relay Proxy will use the same format when writing to the store.
	return fmt.Sprintf("%s.g%d", s.Key, s.Generation.IntValue())
}

func (es *evaluationScope) segmentContainsUser(s *ldmodel.Segment) (bool, error) {
	userKey := es.context.Key()

	// Check if the user is specifically included in or excluded from the segment by key
	if s.Unbounded {
		if !s.Generation.IsDefined() {
			// Big segment queries can only be done if the generation is known. If it's unset,
			// that probably means the data store was populated by an older SDK that doesn't know
			// about the Generation property and therefore dropped it from the JSON data. We'll treat
			// that as a "not configured" condition.
			es.bigSegmentsReferenced = true
			es.bigSegmentsStatus = ldreason.BigSegmentsNotConfigured
			return false, nil
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
		if es.bigSegmentsMembership != nil {
			included := es.bigSegmentsMembership.CheckMembership(makeBigSegmentRef(s))
			if included.IsDefined() {
				return included.BoolValue(), nil
			}
		}
	} else {
		if ldmodel.EvaluatorAccessors.SegmentFindKeyInIncludes(s, userKey) {
			return true, nil
		}
		if ldmodel.EvaluatorAccessors.SegmentFindKeyInExcludes(s, userKey) {
			return false, nil
		}
	}

	// Check if any of the segment rules match
	for _, rule := range s.Rules {
		// Note, taking address of range variable here is OK because it's not used outside the loop
		match, err := es.segmentRuleMatchesUser(&rule, s.Key, s.Salt) //nolint:gosec // see comment above
		if err != nil {
			return false, malformedSegmentError{SegmentKey: s.Key, Err: err}
		}
		if match {
			return true, nil
		}
	}

	return false, nil
}

func (es *evaluationScope) segmentRuleMatchesUser(r *ldmodel.SegmentRule, key, salt string) (bool, error) {
	// Note that r is passed by reference only for efficiency; we do not modify it
	for _, clause := range r.Clauses {
		c := clause
		match, err := clauseMatchesContext(&c, &es.context)
		if !match || err != nil {
			return false, err
		}
	}

	// If the Weight is absent, this rule matches
	if !r.Weight.IsDefined() {
		return true, nil
	}

	// All of the clauses are met. Check to see if the user buckets in
	// TEMPORARY - instead of ldcontext.DefaultKind here, we will eventually have a Kind field in the segment
	bucket, err := es.computeBucketValue(ldvalue.OptionalInt{}, ldcontext.DefaultKind, key, r.BucketBy, salt)
	if err != nil {
		return false, err
	}
	weight := float32(r.Weight.IntValue()) / 100000.0

	return bucket < weight, nil
}
