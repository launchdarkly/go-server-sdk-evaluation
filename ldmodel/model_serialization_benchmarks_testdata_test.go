package ldmodel

import (
	"encoding/json"
	"fmt"

	"github.com/launchdarkly/go-sdk-common/v3/ldattr"
	"github.com/launchdarkly/go-sdk-common/v3/ldtime"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
)

var flagWithAllProperties = FeatureFlag{
	Key: "flag-key",
	On:  true,
	Prerequisites: []Prerequisite{
		{
			Key:       "prereq-key",
			Variation: 1,
		},
	},
	Targets: []Target{
		{
			Values:       []string{"user-key"},
			Variation:    2,
			preprocessed: targetPreprocessedData{valuesMap: map[string]struct{}{"user-key": {}}}, // this is set by PreprocessFlag()
		},
	},
	Rules: []FlagRule{
		{
			ID: "rule-id1",
			Clauses: []Clause{
				{
					Attribute: ldattr.NewLiteralRef("name"),
					Op:        OperatorIn,
					Values:    []ldvalue.Value{ldvalue.String("clause-value")},
					Negate:    true,
				},
			},
			VariationOrRollout: VariationOrRollout{
				Variation: ldvalue.NewOptionalInt(1),
			},
			TrackEvents: true,
		},
		{
			ID:      "rule-id2",
			Clauses: []Clause{},
			VariationOrRollout: VariationOrRollout{
				Rollout: Rollout{
					Kind: RolloutKindRollout,
					Variations: []WeightedVariation{
						{
							Weight:    100000,
							Variation: 3,
						},
					},
					BucketBy: ldattr.NewLiteralRef("name"),
				},
			},
		},
		{
			ID:      "rule-id3",
			Clauses: []Clause{},
			VariationOrRollout: VariationOrRollout{
				Rollout: Rollout{
					Kind: RolloutKindExperiment,
					Variations: []WeightedVariation{
						{
							Weight:    10000,
							Variation: 1,
						},
						{
							Weight:    10000,
							Variation: 2,
						},
						{
							Weight:    80000,
							Variation: 3,
							Untracked: true,
						},
					},
					BucketBy: ldattr.NewLiteralRef("name"),
					Seed:     ldvalue.NewOptionalInt(42),
				},
			},
			TrackEvents: true,
		},
	},
	Fallthrough: VariationOrRollout{
		Rollout: Rollout{
			Variations: []WeightedVariation{
				{
					Weight:    100000,
					Variation: 3,
				},
			},
		},
	},
	OffVariation: ldvalue.NewOptionalInt(3),
	Variations:   []ldvalue.Value{ldvalue.Bool(false), ldvalue.Int(9), ldvalue.String("other")},
	ClientSideAvailability: ClientSideAvailability{
		UsingEnvironmentID: true,
		UsingMobileKey:     true,
		Explicit:           true,
	},
	Salt:                   "flag-salt",
	TrackEvents:            true,
	TrackEventsFallthrough: true,
	DebugEventsUntilDate:   ldtime.UnixMillisecondTime(1000),
	Version:                99,
	Deleted:                true,
}

var flagWithAllPropertiesJSON = map[string]interface{}{
	"key": "flag-key",
	"on":  true,
	"prerequisites": []interface{}{
		map[string]interface{}{
			"key":       "prereq-key",
			"variation": float64(1),
		},
	},
	"targets": []interface{}{
		map[string]interface{}{
			"values":    []interface{}{"user-key"},
			"variation": float64(2),
		},
	},
	"rules": []interface{}{
		map[string]interface{}{
			"id": "rule-id1",
			"clauses": []interface{}{
				map[string]interface{}{
					"attribute": "name",
					"op":        "in",
					"values":    []interface{}{"clause-value"},
					"negate":    true,
				},
			},
			"variation":   float64(1),
			"trackEvents": true,
		},
		map[string]interface{}{
			"id":      "rule-id2",
			"clauses": []interface{}{},
			"rollout": map[string]interface{}{
				"kind": "rollout",
				"variations": []interface{}{
					map[string]interface{}{
						"weight":    float64(100000),
						"variation": float64(3),
					},
				},
				"bucketBy": "name",
			},
			"trackEvents": false,
		},
		map[string]interface{}{
			"id":      "rule-id3",
			"clauses": []interface{}{},
			"rollout": map[string]interface{}{
				"kind":     "experiment",
				"bucketBy": "name",
				"variations": []interface{}{
					map[string]interface{}{
						"weight":    float64(10000),
						"variation": float64(1),
					},
					map[string]interface{}{
						"weight":    float64(10000),
						"variation": float64(2),
					},
					map[string]interface{}{
						"weight":    float64(80000),
						"variation": float64(3),
						"untracked": true,
					},
				},
				"seed": float64(42),
			},
			"trackEvents": true,
		},
	},
	"fallthrough": map[string]interface{}{
		"rollout": map[string]interface{}{
			"variations": []interface{}{
				map[string]interface{}{
					"weight":    float64(100000),
					"variation": float64(3),
				},
			},
		},
	},
	"offVariation": float64(3),
	"variations":   []interface{}{false, float64(9), "other"},
	"clientSideAvailability": map[string]interface{}{
		"usingEnvironmentId": true,
		"usingMobileKey":     true,
	},
	"clientSide":             true,
	"salt":                   "flag-salt",
	"trackEvents":            true,
	"trackEventsFallthrough": true,
	"debugEventsUntilDate":   float64(1000),
	"version":                float64(99),
	"deleted":                true,
}

var flagWithMinimalProperties = FeatureFlag{
	Key:         "flag-key",
	Fallthrough: VariationOrRollout{Variation: ldvalue.NewOptionalInt(1)},
	Variations:  []ldvalue.Value{ldvalue.Bool(false), ldvalue.Int(9), ldvalue.String("other")},
	ClientSideAvailability: ClientSideAvailability{
		UsingMobileKey: true,
		Explicit:       false,
	},
	Salt:    "flag-salt",
	Version: 99,
}

var flagWithMinimalPropertiesJSON = map[string]interface{}{
	"key":          "flag-key",
	"on":           false,
	"offVariation": nil,
	"fallthrough": map[string]interface{}{
		"variation": float64(1),
	},
	"variations":             []interface{}{false, float64(9), "other"},
	"targets":                []interface{}{},
	"rules":                  []interface{}{},
	"prerequisites":          []interface{}{},
	"clientSide":             false,
	"salt":                   "flag-salt",
	"trackEvents":            false,
	"trackEventsFallthrough": false,
	"debugEventsUntilDate":   nil,
	"version":                float64(99),
	"deleted":                false,
}

var segmentWithAllProperties = Segment{
	Key:      "segment-key",
	Included: []string{"user1"},
	Excluded: []string{"user2"},
	preprocessed: segmentPreprocessedData{
		includeMap: map[string]struct{}{"user1": {}},
		excludeMap: map[string]struct{}{"user2": {}},
	},
	Rules: []SegmentRule{
		{
			ID: "rule-id",
			Clauses: []Clause{
				{
					Attribute: ldattr.NewLiteralRef("name"),
					Op:        OperatorIn,
					Values:    []ldvalue.Value{ldvalue.String("clause-value")},
					Negate:    true,
				},
			},
		},
		{
			Weight:   ldvalue.NewOptionalInt(50000),
			BucketBy: ldattr.NewLiteralRef("name"),
		},
	},
	Salt:       "segment-salt",
	Unbounded:  true,
	Version:    99,
	Generation: ldvalue.NewOptionalInt(51),
	Deleted:    true,
}

var segmentWithAllPropertiesJSON = map[string]interface{}{
	"key":      "segment-key",
	"included": []interface{}{"user1"},
	"excluded": []interface{}{"user2"},
	"rules": []interface{}{
		map[string]interface{}{
			"id": "rule-id",
			"clauses": []interface{}{
				map[string]interface{}{
					"attribute": "name",
					"op":        "in",
					"values":    []interface{}{"clause-value"},
					"negate":    true,
				},
			},
		},
		map[string]interface{}{
			"id":       "",
			"clauses":  []interface{}{},
			"weight":   float64(50000),
			"bucketBy": "name",
		},
	},
	"salt":       "segment-salt",
	"unbounded":  true,
	"version":    float64(99),
	"generation": float64(51),
	"deleted":    true,
}

var segmentWithMinimalProperties = Segment{
	Key:     "segment-key",
	Salt:    "segment-salt",
	Version: 99,
}

var segmentWithMinimalPropertiesJSON = map[string]interface{}{
	"key":        "segment-key",
	"included":   []interface{}{},
	"excluded":   []interface{}{},
	"rules":      []interface{}{},
	"salt":       "segment-salt",
	"version":    float64(99),
	"generation": nil,
	"deleted":    false,
}

func makeLargeFlagJSON() []byte {
	makeManyStrings := func() []string {
		ret := []string{}
		for i := 0; i < 200; i++ {
			ret = append(ret, fmt.Sprintf("string%d", i))
		}
		return ret
	}
	makeRules := func() []map[string]interface{} {
		ret := []map[string]interface{}{}
		for i := 0; i < 20; i++ {
			ret = append(ret, map[string]interface{}{
				"id": fmt.Sprintf("rule-id%d", i),
				"clauses": []interface{}{
					map[string]interface{}{
						"attribute": "name",
						"op":        "in",
						"values":    []interface{}{"clause-value"},
						"negate":    true,
					},
				},
				"variation":   float64(1),
				"trackEvents": true,
			})
		}
		return ret
	}
	data := map[string]interface{}{
		"key": "large-flag-key",
		"on":  true,
		"prerequisites": []interface{}{
			map[string]interface{}{
				"key":       "prereq-key",
				"variation": float64(1),
			},
		},
		"targets": []interface{}{
			map[string]interface{}{
				"values":    makeManyStrings(),
				"variation": float64(2),
			},
		},
		"rules": makeRules(),
		"fallthrough": map[string]interface{}{
			"rollout": map[string]interface{}{
				"variations": []interface{}{
					map[string]interface{}{
						"weight":    float64(100000),
						"variation": float64(3),
					},
				},
			},
		},
		"offVariation": float64(3),
		"variations":   []interface{}{false, float64(9), "other"},
		"clientSideAvailability": map[string]interface{}{
			"usingEnvironmentId": true,
			"usingMobileKey":     true,
		},
		"clientSide":             true,
		"salt":                   "flag-salt",
		"trackEvents":            true,
		"trackEventsFallthrough": true,
		"debugEventsUntilDate":   float64(1000),
		"version":                float64(99),
		"deleted":                true,
	}
	bytes, _ := json.Marshal(data)
	return bytes
}

func makeLargeSegmentJSON() []byte {
	makeManyStrings := func() []string {
		ret := []string{}
		for i := 0; i < 200; i++ {
			ret = append(ret, fmt.Sprintf("string%d", i))
		}
		return ret
	}
	makeRules := func() []map[string]interface{} {
		ret := []map[string]interface{}{}
		for i := 0; i < 20; i++ {
			ret = append(ret, map[string]interface{}{
				"id": fmt.Sprintf("rule-id%d", i),
				"clauses": []interface{}{
					map[string]interface{}{
						"attribute": "name",
						"op":        "in",
						"values":    []interface{}{"clause-value"},
						"negate":    true,
					},
				},
				"weight":   float64(50000),
				"bucketBy": "name",
			})
		}
		return ret
	}
	data := map[string]interface{}{
		"key":       "large-segment-key",
		"included":  makeManyStrings(),
		"excluded":  makeManyStrings(),
		"rules":     makeRules(),
		"salt":      "segment-salt",
		"unbounded": true,
		"version":   float64(99),
		"deleted":   true,
	}
	bytes, _ := json.Marshal(data)
	return bytes
}

type featureFlagEquivalentStruct struct {
	Key           string `json:"key"`
	On            bool   `json:"on"`
	Prerequisites []struct {
		Key       string `json:"key"`
		Variation int    `json:"variation"`
	} `json:"prerequisites"`
	Targets []struct {
		Values    []string `json:"values"`
		Variation int      `json:"variation"`
	} `json:"targets"`
	Rules []struct {
		Variation *int `json:"variation"`
		Rollout   *struct {
			Variations []struct {
				Variation int `json:"variation"`
				Weight    int `json:"weight"`
			} `json:"variations"`
			BucketBy *string `json:"bucketBy"`
		} `json:"rollout"`
		ID      string `json:"id"`
		Clauses []struct {
			Attribute string          `json:"attribute"`
			Op        string          `json:"op"`
			Values    []ldvalue.Value `json:"values"`
			Negate    bool            `json:"negate"`
		} `json:"clauses"`
		TrackEvents bool `json:"trackEvents"`
	} `json:"rules"`
	Fallthrough struct {
		Variation *int `json:"variation"`
		Rollout   struct {
			Variations []struct {
			} `json:"variations"`
			BucketBy *string `json:"bucketBy"`
		} `json:"rollout"`
	} `json:"fallthrough"`
	OffVariation           *int            `json:"offVariation"`
	Variations             []ldvalue.Value `json:"variations"`
	ClientSideAvailability *struct {
		UsingMobileKey     bool `json:"usingMobileKey"`
		UsingEnvironmentID bool `json:"usingEnvironmentId"`
	} `json:"clientSideAvailability"`
	Salt                   string  `json:"salt"`
	TrackEvents            bool    `json:"trackEvents"`
	TrackEventsFallthrough bool    `json:"trackEventsFallthrough"`
	DebugEventsUntilDate   *uint64 `json:"debugEventsUntilDate"`
	Version                int     `json:"version"`
	Deleted                bool    `json:"deleted"`
}

type segmentEquivalentStruct struct {
	Key      string   `json:"key"`
	Included []string `json:"included"`
	Excluded []string `json:"excluded"`
	Salt     string   `json:"salt"`
	Rules    []struct {
		ID      string `json:"id"`
		Clauses []struct {
			Attribute string          `json:"attribute"`
			Op        string          `json:"op"`
			Values    []ldvalue.Value `json:"values"`
			Negate    bool            `json:"negate"`
		} `json:"clauses"`
		Weight   *int    `json:"weight"`
		BucketBy *string `json:"bucketBy"`
	} `json:"rules"`
	Unbounded bool `json:"unbounded"`
	Version   int  `json:"version"`
	Deleted   bool `json:"deleted"`
}
