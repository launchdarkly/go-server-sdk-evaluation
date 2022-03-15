package evaluation

import (
	"fmt"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
)

// These parameters are used in evaluator_bucketing_test.go. In each case, we have precomputed
// the expected bucketing value using a version of the code that was known to be correct at the
// time.

type bucketingTestParams struct {
	flagOrSegmentKey    string
	salt                string
	seed                ldvalue.OptionalInt
	contextValue        string // i.e. the context key, or whatever other attribute we might be bucketing by
	secondaryKey        string
	expectedBucketValue float32
}

func (p bucketingTestParams) description() string { return fmt.Sprintf("%+v", p) }

func makeBucketingTestParams() []bucketingTestParams {
	return []bucketingTestParams{
		{
			flagOrSegmentKey:    "hashKey",
			salt:                "saltyA",
			contextValue:        "userKeyA",
			expectedBucketValue: 0.42157587,
		},
		{
			flagOrSegmentKey:    "hashKey",
			salt:                "saltyA",
			contextValue:        "userKeyB",
			expectedBucketValue: 0.6708485,
		},
		{
			flagOrSegmentKey:    "hashKey",
			salt:                "saltyA",
			contextValue:        "userKeyC",
			expectedBucketValue: 0.10343106,
		},
	}
}

func makeBucketingTestParamsWithNumericStringValues() []bucketingTestParams {
	return []bucketingTestParams{
		{
			flagOrSegmentKey:    "hashKey",
			salt:                "saltyA",
			contextValue:        "33333",
			expectedBucketValue: 0.54771423,
		},
		{
			flagOrSegmentKey:    "hashKey",
			salt:                "saltyA",
			contextValue:        "99999",
			expectedBucketValue: 0.7309658,
		},
	}
}

func makeBucketingTestParamsForExperiments() []bucketingTestParams {
	ret := makeBucketingTestParams()
	ret = append(ret, []bucketingTestParams{
		{
			flagOrSegmentKey:    "hashKey",
			salt:                "saltyA",
			contextValue:        "userKeyA",
			expectedBucketValue: 0.42157587,
		},
		{
			flagOrSegmentKey:    "hashKey",
			salt:                "saltyA",
			contextValue:        "userKeyB",
			expectedBucketValue: 0.6708485,
		},
		{
			flagOrSegmentKey:    "hashKey",
			salt:                "saltyA",
			contextValue:        "userKeyC",
			expectedBucketValue: 0.10343106,
		},
		{
			flagOrSegmentKey:    "hashKey",
			salt:                "saltyA",
			contextValue:        "userKeyA",
			seed:                ldvalue.NewOptionalInt(61),
			expectedBucketValue: 0.09801207,
		},
		{
			flagOrSegmentKey:    "hashKey",
			salt:                "saltyA",
			contextValue:        "userKeyB",
			seed:                ldvalue.NewOptionalInt(61),
			expectedBucketValue: 0.14483777,
		},
		{
			flagOrSegmentKey:    "hashKey",
			salt:                "saltyA",
			contextValue:        "userKeyC",
			seed:                ldvalue.NewOptionalInt(61),
			expectedBucketValue: 0.9242641,
		},
	}...)
	return ret
}
