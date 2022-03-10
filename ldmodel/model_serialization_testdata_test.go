package ldmodel

import (
	"encoding/json"

	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldtime"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"
)

type flagSerializationTestParams struct {
	name          string
	flag          FeatureFlag
	jsonString    string   // for marshaling tests, jsonString doesn't need to include any of flagTopDefaultLevelProperties
	jsonAltInputs []string // if specified, unmarshaling test will verify that these parse to the same result

	isCustomClientSideAvailability bool
}

type segmentSerializationTestParams struct {
	name          string
	segment       Segment
	jsonString    string   // for marshaling tests, jsonString doesn't need to include any of segmentTopDefaultLevelProperties
	jsonAltInputs []string // if specified, unmarshaling test will verify that these parse to the same result
}

var flagTopLevelDefaultProperties = map[string]interface{}{
	"key":                    "",
	"deleted":                false,
	"variations":             []interface{}{},
	"on":                     false,
	"offVariation":           nil,
	"fallthrough":            map[string]interface{}{},
	"targets":                []interface{}{},
	"contextTargets":         []interface{}{},
	"prerequisites":          []interface{}{},
	"rules":                  []interface{}{},
	"clientSide":             false,
	"trackEvents":            false,
	"trackEventsFallthrough": false,
	"debugEventsUntilDate":   nil,
	"salt":                   "",
	"version":                0,
}

var segmentTopLevelDefaultProperties = map[string]interface{}{
	"key":        "",
	"deleted":    false,
	"version":    0,
	"generation": nil,
	"included":   []string{},
	"excluded":   []string{},
	"rules":      []interface{}{},
	"salt":       "",
}

var simpleRollout = Rollout{
	Variations: []WeightedVariation{{Variation: 0, Weight: 100000}},
}

const simpleRolloutJSON = `{"variations": [{"variation": 0, "weight": 100000}]}`

var rolloutWithContextKind = Rollout{
	ContextKind: ldcontext.Kind("org"),
	Variations:  []WeightedVariation{{Variation: 0, Weight: 100000}},
}

const rolloutWithContextKindJSON = `{"contextKind": "org", "variations": [{"variation": 0, "weight": 100000}]}`

var rolloutWithBucketBy = Rollout{
	BucketBy:   ldattr.NewNameRef("name"),
	Variations: []WeightedVariation{{Variation: 0, Weight: 100000}},
}

const rolloutWithBucketByJSON = `{"bucketBy": "name", "variations": [{"variation": 0, "weight": 100000}]}`

var simpleExperiment = Rollout{
	Kind:       RolloutKindExperiment,
	Variations: []WeightedVariation{{Variation: 0, Weight: 100000}},
}

const simpleExperimentJSON = `{"kind": "experiment", "variations": [{"variation": 0, "weight": 100000}]}`
const experimentWithSeedNullJSON = `{"kind": "experiment", "seed": null, "variations": [{"variation": 0, "weight": 100000}]}`

var experimentWithSeed = Rollout{
	Kind:       RolloutKindExperiment,
	Seed:       ldvalue.NewOptionalInt(12345),
	Variations: []WeightedVariation{{Variation: 0, Weight: 100000}},
}

const experimentWithSeedJSON = `{"kind": "experiment", "seed": 12345, "variations": [{"variation": 0, "weight": 100000}]}`

var experimentWithUntracked = Rollout{
	Kind: RolloutKindExperiment,
	Variations: []WeightedVariation{
		{Variation: 0, Weight: 75000},
		{Variation: 1, Weight: 25000, Untracked: true},
	},
}

const experimentWithUntrackedJSON = `{"kind": "experiment", "variations": [` +
	`{"variation": 0, "weight": 75000}, {"variation": 1, "weight": 25000, "untracked": true}]}`

var simpleClause = Clause{
	Attribute: ldattr.NewNameRef("key"),
	Op:        OperatorIn,
	Values:    []ldvalue.Value{ldvalue.String("a")},
}

const simpleClauseJSON = `{"attribute": "key", "op": "in", "values": ["a"], "negate": false}`
const simpleClauseMinimalJSON = `{"attribute": "key", "op": "in", "values": ["a"]}`

var clauseWithKind = Clause{
	Kind:      ldcontext.Kind("org"),
	Attribute: ldattr.NewNameRef("key"),
	Op:        OperatorIn,
	Values:    []ldvalue.Value{ldvalue.String("a")},
}

const clauseWithKindJSON = `{"kind": "org", "attribute": "key", "op": "in", "values": ["a"], "negate": false}`

var negatedClause = Clause{
	Attribute: ldattr.NewNameRef("key"),
	Op:        OperatorIn,
	Values:    []ldvalue.Value{ldvalue.String("a")},
	Negate:    true,
}

const negatedClauseJSON = `{"attribute": "key", "op": "in", "values": ["a"], "negate": true}`

func makeFlagSerializationTestParams() []flagSerializationTestParams {
	return []flagSerializationTestParams{
		{
			name:       "defaults",
			flag:       FeatureFlag{},
			jsonString: `{}`,
			jsonAltInputs: []string{
				`{"deleted": false}`,
				`{"on": false}`,
				`{"offVariation": null}`,
				`{"fallthrough": {"variation": null}}`,
				`{"prerequisites": []}`,
				`{"targets": []}`,
				`{"rules": []}`,
			},
		},
		{
			name:       "key",
			flag:       FeatureFlag{Key: "flag-key"},
			jsonString: `{"key": "flag-key"}`,
		},
		{
			name:       "version",
			flag:       FeatureFlag{Version: 99},
			jsonString: `{"version": 99}`,
		},
		{
			name:       "deleted",
			flag:       FeatureFlag{Deleted: true},
			jsonString: `{"deleted": true}`,
		},
		{
			name: "variations",
			flag: FeatureFlag{Variations: []ldvalue.Value{
				ldvalue.Bool(true),
				ldvalue.Int(1),
				ldvalue.Float64(1.5),
				ldvalue.String("x"),
				ldvalue.ArrayOf(),
				ldvalue.ObjectBuild().Build(),
			},
			},
			jsonString: `{"variations": [true, 1, 1.5, "x", [], {}]}`,
		},
		{
			name:       "on",
			flag:       FeatureFlag{On: true},
			jsonString: `{"on": true}`,
		},
		{
			name:       "offVariation",
			flag:       FeatureFlag{OffVariation: ldvalue.NewOptionalInt(1)},
			jsonString: `{"offVariation": 1}`,
		},
		{
			name:          "fallthrough variation",
			flag:          FeatureFlag{Fallthrough: VariationOrRollout{Variation: ldvalue.NewOptionalInt(1)}},
			jsonString:    `{"fallthrough": {"variation": 1}}`,
			jsonAltInputs: []string{`{"fallthrough": {"variation": 1, "rollout": null}}`},
		},
		{
			name:          "fallthrough rollout",
			flag:          FeatureFlag{Fallthrough: VariationOrRollout{Rollout: simpleRollout}},
			jsonString:    `{"fallthrough": {"rollout": ` + simpleRolloutJSON + `}}`,
			jsonAltInputs: []string{`{"fallthrough": {"variation": null, "rollout": ` + simpleRolloutJSON + `}}`},
		},
		{
			name:          "fallthrough experiment",
			flag:          FeatureFlag{Fallthrough: VariationOrRollout{Rollout: simpleExperiment}},
			jsonString:    `{"fallthrough": {"rollout": ` + simpleExperimentJSON + `}}`,
			jsonAltInputs: []string{`{"fallthrough": {"rollout": ` + experimentWithSeedNullJSON + `}}`},
		},
		{
			name:       "fallthrough experiment with seed",
			flag:       FeatureFlag{Fallthrough: VariationOrRollout{Rollout: experimentWithSeed}},
			jsonString: `{"fallthrough": {"rollout": ` + experimentWithSeedJSON + `}}`,
		},
		{
			name:       "fallthrough experiment with untracked",
			flag:       FeatureFlag{Fallthrough: VariationOrRollout{Rollout: experimentWithUntracked}},
			jsonString: `{"fallthrough": {"rollout": ` + experimentWithUntrackedJSON + `}}`,
		},
		{
			name:       "fallthrough rollout contextKind",
			flag:       FeatureFlag{Fallthrough: VariationOrRollout{Rollout: rolloutWithContextKind}},
			jsonString: `{"fallthrough": {"rollout": ` + rolloutWithContextKindJSON + `}}`,
		},
		{
			name:       "fallthrough rollout bucketBy",
			flag:       FeatureFlag{Fallthrough: VariationOrRollout{Rollout: rolloutWithBucketBy}},
			jsonString: `{"fallthrough": {"rollout": ` + rolloutWithBucketByJSON + `}}`,
		},
		{
			name: "fallthrough rollout bucketBy invalid ref",
			// Here we verify that an invalid attribute ref doesn't make parsing fail and is preserved in reserialization
			flag: FeatureFlag{Fallthrough: VariationOrRollout{Rollout: Rollout{
				BucketBy:   ldattr.NewRef("///"),
				Variations: []WeightedVariation{{Variation: 0, Weight: 100000}},
			}}},
			jsonString: `{"fallthrough": {"rollout": {"bucketBy": "///", "variations": [{"variation": 0, "weight": 100000}]}}}`,
		},
		{
			name: "prerequisites",
			flag: FeatureFlag{
				Prerequisites: []Prerequisite{
					{Variation: 1, Key: "pre-key"},
				},
			},
			jsonString: `{"prerequisites": [ {"variation": 1, "key": "pre-key"} ]}`,
		},
		{
			name: "targets",
			flag: FeatureFlag{
				Targets: []Target{
					{Variation: 1, Values: []string{"a", "b"}},
				},
			},
			jsonString: `{"targets": [ {"variation": 1, "values": ["a", "b"]} ]}`,
		},
		{
			name: "contextTargets",
			flag: FeatureFlag{
				ContextTargets: []Target{
					{Kind: "org", Variation: 1, Values: []string{"a", "b"}},
				},
			},
			jsonString: `{"contextTargets": [ {"kind": "org", "variation": 1, "values": ["a", "b"]} ]}`,
		},
		{
			name: "minimal rule with variation",
			flag: FeatureFlag{
				Rules: []FlagRule{
					{VariationOrRollout: VariationOrRollout{Variation: ldvalue.NewOptionalInt(1)}},
				},
			},
			jsonString:    `{"rules": [ {"variation": 1, "clauses": [], "trackEvents": false} ]}`,
			jsonAltInputs: []string{`{"rules": [ {"variation": 1} ]}`},
		},
		{
			name: "minimal rule with rollout",
			flag: FeatureFlag{
				Rules: []FlagRule{
					{VariationOrRollout: VariationOrRollout{Rollout: simpleRollout}},
				},
			},
			jsonString:    `{"rules": [ {"rollout": ` + simpleRolloutJSON + `, "clauses": [], "trackEvents": false} ]}`,
			jsonAltInputs: []string{`{"rules": [ {"rollout": ` + simpleRolloutJSON + `} ]}`},
		},
		{
			name: "rule ID",
			flag: FeatureFlag{
				Rules: []FlagRule{
					{ID: "a", VariationOrRollout: VariationOrRollout{Variation: ldvalue.NewOptionalInt(1)}},
				},
			},
			jsonString: `{"rules": [ {"id": "a", "variation": 1, "clauses": [], "trackEvents": false} ]}`,
		},
		{
			name: "rule trackEvents",
			flag: FeatureFlag{
				Rules: []FlagRule{
					{VariationOrRollout: VariationOrRollout{Variation: ldvalue.NewOptionalInt(1)}, TrackEvents: true},
				},
			},
			jsonString: `{"rules": [ {"variation": 1, "clauses": [], "trackEvents": true} ]}`,
		},
		{
			name: "minimal rule clause",
			flag: FeatureFlag{
				Rules: []FlagRule{
					{
						VariationOrRollout: VariationOrRollout{Variation: ldvalue.NewOptionalInt(1)},
						Clauses:            []Clause{simpleClause},
					},
				},
			},
			jsonString:    `{"rules": [ {"variation": 1, "clauses": [` + simpleClauseJSON + `], "trackEvents": false }]}`,
			jsonAltInputs: []string{`{"rules": [ {"variation": 1, "clauses": [` + simpleClauseMinimalJSON + `] }]}`},
		},
		{
			name: "rule clause with kind",
			flag: FeatureFlag{
				Rules: []FlagRule{
					{
						VariationOrRollout: VariationOrRollout{Variation: ldvalue.NewOptionalInt(1)},
						Clauses:            []Clause{clauseWithKind},
					},
				},
			},
			jsonString: `{"rules": [ {"variation": 1, "clauses": [` + clauseWithKindJSON + `], "trackEvents": false }]}`,
		},
		{
			name: "rule clause negate",
			flag: FeatureFlag{
				Rules: []FlagRule{
					{
						VariationOrRollout: VariationOrRollout{Variation: ldvalue.NewOptionalInt(1)},
						Clauses:            []Clause{negatedClause},
					},
				},
			},
			jsonString: `{"rules": [ {"variation": 1, "clauses": [` + negatedClauseJSON + `], "trackEvents": false} ]}`,
		},
		{
			name: "rule clause with segmentMatch",
			flag: FeatureFlag{
				Rules: []FlagRule{
					{
						VariationOrRollout: VariationOrRollout{Variation: ldvalue.NewOptionalInt(1)},
						Clauses: []Clause{
							{Op: OperatorSegmentMatch, Values: []ldvalue.Value{ldvalue.String("a")}},
						},
					},
				},
			},
			jsonString: `{"rules": [ {"variation": 1, "clauses": ` +
				`[ {"attribute": "", "op": "segmentMatch", "values": ["a"], "negate": false} ]` +
				`, "trackEvents": false} ]}`, // note, attribute is serialized as "" in this case, not omitted
		},
		{
			name: "rule clause with invalid attribute ref",
			// Here we verify that an invalid attribute ref doesn't make parsing fail and is preserved in reserialization
			flag: FeatureFlag{
				Rules: []FlagRule{
					{
						VariationOrRollout: VariationOrRollout{Variation: ldvalue.NewOptionalInt(1)},
						Clauses: []Clause{
							{Attribute: ldattr.NewRef("///"), Op: OperatorIn, Values: []ldvalue.Value{ldvalue.String("a")}},
						},
					},
				},
			},
			jsonString: `{"rules": [ {"variation": 1, "clauses": ` +
				`[ {"attribute": "///", "op": "in", "values": ["a"], "negate": false} ]` +
				`, "trackEvents": false} ]}`,
		},
		{
			name: "clientSide",
			flag: FeatureFlag{
				ClientSideAvailability: ClientSideAvailability{
					UsingMobileKey:     true,
					UsingEnvironmentID: true,
					Explicit:           false,
				},
			},
			jsonString:                     `{"clientSide": true}`,
			isCustomClientSideAvailability: true,
		},
		{
			name: "clientSide explicitly false",
			flag: FeatureFlag{
				ClientSideAvailability: ClientSideAvailability{
					UsingMobileKey:     true,
					UsingEnvironmentID: false,
					Explicit:           false,
				},
			},
			jsonString:                     `{"clientSide": false}`,
			isCustomClientSideAvailability: true,
		},
		{
			name: "clientSideAvailability both false",
			flag: FeatureFlag{
				ClientSideAvailability: ClientSideAvailability{
					Explicit: true,
				},
			},
			jsonString:                     `{"clientSideAvailability": {"usingMobileKey": false, "usingEnvironmentId": false}}`,
			isCustomClientSideAvailability: true,
		},
		{
			name: "clientSideAvailability both true",
			flag: FeatureFlag{
				ClientSideAvailability: ClientSideAvailability{
					UsingMobileKey:     true,
					UsingEnvironmentID: true,
					Explicit:           true,
				},
			},
			jsonString:                     `{"clientSide": true, "clientSideAvailability": {"usingMobileKey": true, "usingEnvironmentId": true}}`,
			isCustomClientSideAvailability: true,
		},
		{
			name: "clientSideAvailability usingMobileKey only",
			flag: FeatureFlag{
				ClientSideAvailability: ClientSideAvailability{
					UsingMobileKey: true,
					Explicit:       true,
				},
			},
			jsonString:                     `{"clientSideAvailability": {"usingMobileKey": true, "usingEnvironmentId": false}}`,
			isCustomClientSideAvailability: true,
		},
		{
			name: "clientSideAvailability usingEnvironmentId only",
			flag: FeatureFlag{
				ClientSideAvailability: ClientSideAvailability{
					UsingEnvironmentID: true,
					Explicit:           true,
				},
			},
			jsonString:                     `{"clientSide": true, "clientSideAvailability": {"usingMobileKey": false, "usingEnvironmentId": true}}`,
			isCustomClientSideAvailability: true,
		},
		{
			name:       "salt",
			flag:       FeatureFlag{Salt: "flag-salt"},
			jsonString: `{"salt": "flag-salt"}`,
		},
		{
			name:       "trackEvents",
			flag:       FeatureFlag{TrackEvents: true},
			jsonString: `{"trackEvents": true}`,
		},
		{
			name:       "trackEventsFallthrough",
			flag:       FeatureFlag{TrackEventsFallthrough: true},
			jsonString: `{"trackEventsFallthrough": true}`,
		},
		{
			name:       "debugEventsUntilDate",
			flag:       FeatureFlag{DebugEventsUntilDate: ldtime.UnixMillisecondTime(1000)},
			jsonString: `{"debugEventsUntilDate": 1000}`,
		},
	}
}

func makeSegmentSerializationTestParams() []segmentSerializationTestParams {
	return []segmentSerializationTestParams{
		{
			name:       "defaults",
			segment:    Segment{},
			jsonString: `{}`,
			jsonAltInputs: []string{
				`{"deleted": false}`,
				`{"included": []}`,
				`{"excluded": []}`,
				`{"rules": []}`,
				`{"unbounded": false}`,
				`{"generation": null}`,
			},
		},
		{
			name:       "key",
			segment:    Segment{Key: "segment-key"},
			jsonString: `{"key": "segment-key"}`,
		},
		{
			name:       "version",
			segment:    Segment{Version: 99},
			jsonString: `{"version": 99}`,
		},
		{
			name:       "deleted",
			segment:    Segment{Deleted: true},
			jsonString: `{"deleted": true}`,
		},
		{
			name:       "included",
			segment:    Segment{Included: []string{"a", "b"}},
			jsonString: `{"included": ["a", "b"]}`,
		},
		{
			name:       "excluded",
			segment:    Segment{Excluded: []string{"a", "b"}},
			jsonString: `{"excluded": ["a", "b"]}`,
		},
		{
			name: "minimal rule",
			segment: Segment{
				Rules: []SegmentRule{
					{},
				},
			},
			jsonString:    `{"rules": [ {"id": "", "clauses": []} ]}`,
			jsonAltInputs: []string{`{"rules": [ {} ]}`},
		},
		{
			name: "minimal rule with weight",
			segment: Segment{
				Rules: []SegmentRule{
					{Weight: ldvalue.NewOptionalInt(100000)},
				},
			},
			jsonString: `{"rules": [ {"id": "", "weight": 100000, "clauses": []} ]}`,
		},
		{
			name: "rule bucketBy",
			segment: Segment{
				Rules: []SegmentRule{
					{Weight: ldvalue.NewOptionalInt(100000), BucketBy: ldattr.NewNameRef("name")},
				},
			},
			jsonString: `{"rules": [ {"id": "", "weight": 100000, "bucketBy": "name", "clauses": []} ]}`,
		},
		{
			name: "rule bucketBy invalid ref",
			// Here we verify that an invalid attribute ref doesn't make parsing fail and is preserved in reserialization
			segment: Segment{
				Rules: []SegmentRule{
					{Weight: ldvalue.NewOptionalInt(100000), BucketBy: ldattr.NewRef("///")},
				},
			},
			jsonString: `{"rules": [ {"id": "", "weight": 100000, "bucketBy": "///", "clauses": []} ]}`,
		},
		{
			name: "rule ID",
			segment: Segment{
				Rules: []SegmentRule{
					{ID: "a"},
				},
			},
			jsonString: `{"rules": [ {"id": "a", "clauses": []} ]}`,
		},
		{
			name: "minimal rule clause",
			segment: Segment{
				Rules: []SegmentRule{
					{Clauses: []Clause{simpleClause}},
				},
			},
			jsonString:    `{"rules": [ {"id": "", "clauses": [` + simpleClauseJSON + `]} ]}`,
			jsonAltInputs: []string{`{"rules": [ {"id": "", "clauses": [` + simpleClauseMinimalJSON + `]} ]}`},
		},
		{
			name: "rule clause with kind",
			segment: Segment{
				Rules: []SegmentRule{
					{Clauses: []Clause{clauseWithKind}},
				},
			},
			jsonString: `{"rules": [ {"id": "", "clauses": [` + clauseWithKindJSON + `]} ]}`,
		},
		{
			name: "rule clause negated",
			segment: Segment{
				Rules: []SegmentRule{
					{Clauses: []Clause{negatedClause}},
				},
			},
			jsonString: `{"rules": [ {"id": "", "clauses": [` + negatedClauseJSON + `]} ]}`,
		},
		{
			name: "rule clause with invalid attribute ref",
			// Here we verify that an invalid attribute ref doesn't make parsing fail and is preserved in reserialization
			segment: Segment{
				Rules: []SegmentRule{
					{
						Clauses: []Clause{
							{Attribute: ldattr.NewRef("///"), Op: OperatorIn, Values: []ldvalue.Value{ldvalue.String("a")}},
						},
					},
				},
			},
			jsonString: `{"rules": [ {"id": "", "clauses": ` +
				`[ {"attribute": "///", "op": "in", "values": ["a"], "negate": false} ]` +
				`} ]}`,
		},
		{
			name:       "unbounded and generation",
			segment:    Segment{Unbounded: true, Generation: ldvalue.NewOptionalInt(1)},
			jsonString: `{"unbounded": true, "generation": 1}`,
		},
		{
			name:       "salt",
			segment:    Segment{Salt: "segment-salt"},
			jsonString: `{"salt": "segment-salt"}`,
		},
	}
}

func mergeDefaultProperties(output json.RawMessage, defaults map[string]interface{}) json.RawMessage {
	var parsedOutput map[string]interface{}
	if err := json.Unmarshal(output, &parsedOutput); err != nil {
		return output
	}
	outMap := make(map[string]interface{})
	for k, v := range parsedOutput {
		outMap[k] = v
	}
	for k, v := range defaults {
		if _, found := outMap[k]; !found {
			outMap[k] = v
		}
	}
	data, err := json.Marshal(outMap)
	if err != nil {
		return output
	}
	return data
}
