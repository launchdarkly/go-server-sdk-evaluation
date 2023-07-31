package ldmodel

import (
	"encoding/json"

	"github.com/launchdarkly/go-sdk-common/v3/ldtime"

	"github.com/launchdarkly/go-sdk-common/v3/ldattr"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
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

type clauseSerializationTestParams struct {
	name          string
	clause        Clause
	jsonString    string
	jsonAltInputs []string
}

type rolloutSerializationTestParams struct {
	name          string
	rollout       Rollout
	jsonString    string
	jsonAltInputs []string
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
	"excludeFromSummaries":   false,
}

var segmentTopLevelDefaultProperties = map[string]interface{}{
	"key":              "",
	"deleted":          false,
	"version":          0,
	"generation":       nil,
	"included":         []string{},
	"excluded":         []string{},
	"includedContexts": []interface{}{},
	"excludedContexts": []interface{}{},
	"rules":            []interface{}{},
	"salt":             "",
}

func makeFlagSerializationTestParams() []flagSerializationTestParams {
	ret := []flagSerializationTestParams{
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
					{ContextKind: "org", Variation: 1, Values: []string{"a", "b"}},
				},
			},
			jsonString: `{"contextTargets": [ {"contextKind": "org", "variation": 1, "values": ["a", "b"]} ]}`,
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
		{
			name:       "migration",
			flag:       FeatureFlag{Migration: nil},
			jsonString: `{}`,
		},
		{
			name:       "migration",
			flag:       FeatureFlag{Migration: &MigrationFlagParameters{CheckRatio: ldvalue.NewOptionalInt(2)}},
			jsonString: `{"migration": {"checkRatio": 2}}`,
		},
		{
			name:       "migration",
			flag:       FeatureFlag{Migration: &MigrationFlagParameters{CheckRatio: ldvalue.NewOptionalInt(1)}},
			jsonString: `{"migration": {"checkRatio": 1}}`,
		},
		{
			name:       "samplingRatio",
			flag:       FeatureFlag{},
			jsonString: `{}`,
		},
		{
			name:       "samplingRatio",
			flag:       FeatureFlag{SamplingRatio: ldvalue.NewOptionalInt(10)},
			jsonString: `{"samplingRatio": 10}`,
		},
		{
			name:       "samplingRatio",
			flag:       FeatureFlag{SamplingRatio: ldvalue.NewOptionalInt(1)},
			jsonString: `{"samplingRatio": 1}`,
		},
		{
			name:       "excludeFromSummaries",
			flag:       FeatureFlag{},
			jsonString: `{}`,
		},
		{
			name:       "excludeFromSummaries",
			flag:       FeatureFlag{ExcludeFromSummaries: true},
			jsonString: `{"excludeFromSummaries": true}`,
		},
		{
			name:       "excludeFromSummaries",
			flag:       FeatureFlag{ExcludeFromSummaries: false},
			jsonString: `{"excludeFromSummaries": false}`,
		},
		{
			name:       "excludeFromSummaries",
			flag:       FeatureFlag{},
			jsonString: `{"excludeFromSummaries": false}`,
		},
	}

	makeFlagJSONForClause := func(clauseJSON string) string {
		return `{"rules": [ {"variation": 1, "clauses": [` + clauseJSON + `], "trackEvents": false }]}`
	}
	for _, cp := range makeClauseSerializationTestParams() {
		fp := flagSerializationTestParams{
			name: "rule clause " + cp.name,
			flag: FeatureFlag{
				Rules: []FlagRule{
					{
						VariationOrRollout: VariationOrRollout{Variation: ldvalue.NewOptionalInt(1)},
						Clauses:            []Clause{cp.clause},
					},
				},
			},
			jsonString: makeFlagJSONForClause(cp.jsonString),
		}
		for _, alt := range cp.jsonAltInputs {
			fp.jsonAltInputs = append(fp.jsonAltInputs, makeFlagJSONForClause(alt))
		}
		ret = append(ret, fp)
	}

	makeFlagJSONForFallthroughRollout := func(rolloutJSON string) string {
		return `{"fallthrough": {"rollout": ` + rolloutJSON + `}}`
	}
	makeFlagJSONForRuleRollout := func(rolloutJSON string) string {
		return `{"rules": [ {"rollout": ` + rolloutJSON + `, "clauses": [], "trackEvents": false} ]}`
	}
	for _, rp := range makeRolloutSerializationTestParams() {
		fp1 := flagSerializationTestParams{
			name: "fallthrough rollout " + rp.name,
			flag: FeatureFlag{
				Fallthrough: VariationOrRollout{Rollout: rp.rollout},
			},
			jsonString: makeFlagJSONForFallthroughRollout(rp.jsonString),
		}
		for _, alt := range rp.jsonAltInputs {
			fp1.jsonAltInputs = append(fp1.jsonAltInputs, makeFlagJSONForFallthroughRollout(alt))
		}
		fp2 := flagSerializationTestParams{
			name: "rule rollout " + rp.name,
			flag: FeatureFlag{
				Rules: []FlagRule{{VariationOrRollout: VariationOrRollout{Rollout: rp.rollout}}},
			},
			jsonString: makeFlagJSONForRuleRollout(rp.jsonString),
		}
		for _, alt := range rp.jsonAltInputs {
			fp2.jsonAltInputs = append(fp2.jsonAltInputs, makeFlagJSONForRuleRollout(alt))
		}
		ret = append(ret, fp1, fp2)
	}

	return ret
}

func makeSegmentSerializationTestParams() []segmentSerializationTestParams {
	ret := []segmentSerializationTestParams{
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
			name: "includedContexts",
			segment: Segment{
				IncludedContexts: []SegmentTarget{
					{ContextKind: "org", Values: []string{"a", "b"}},
				}},
			jsonString: `{"includedContexts": [ {"contextKind": "org", "values": ["a", "b"]} ]}`,
		},
		{
			name: "excludedContexts",
			segment: Segment{
				ExcludedContexts: []SegmentTarget{
					{ContextKind: "org", Values: []string{"a", "b"}},
				}},
			jsonString: `{"excludedContexts": [ {"contextKind": "org", "values": ["a", "b"]} ]}`,
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
					{Weight: ldvalue.NewOptionalInt(100000), BucketBy: ldattr.NewLiteralRef("name")},
				},
			},
			jsonString: `{"rules": [ {"id": "", "weight": 100000, "bucketBy": "name", "clauses": []} ]}`,
		},
		{
			name: "rule bucketBy invalid ref",
			// Here we verify that an invalid attribute ref doesn't make parsing fail and is preserved in reserialization
			segment: Segment{
				Rules: []SegmentRule{
					{
						RolloutContextKind: ldcontext.Kind("user"),
						Weight:             ldvalue.NewOptionalInt(100000),
						BucketBy:           ldattr.NewRef("///"),
					},
				},
			},
			jsonString: `{"rules": [ {"id": "", "weight": 100000, "rolloutContextKind": "user", "bucketBy": "///", "clauses": []} ]}`,
		},
		{
			name: "rule rolloutContextKind",
			segment: Segment{
				Rules: []SegmentRule{
					{Weight: ldvalue.NewOptionalInt(100000), RolloutContextKind: ldcontext.Kind("org")},
				},
			},
			jsonString: `{"rules": [ {"id": "", "weight": 100000, "rolloutContextKind": "org", "clauses": []} ]}`,
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
			name:       "unbounded and generation",
			segment:    Segment{Unbounded: true, Generation: ldvalue.NewOptionalInt(1)},
			jsonString: `{"unbounded": true, "generation": 1}`,
		},
		{
			name:       "unbounded and generation and unboundedContextKind",
			segment:    Segment{Unbounded: true, UnboundedContextKind: "org", Generation: ldvalue.NewOptionalInt(1)},
			jsonString: `{"unbounded": true, "unboundedContextKind": "org", "generation": 1}`,
		},
		{
			name:       "salt",
			segment:    Segment{Salt: "segment-salt"},
			jsonString: `{"salt": "segment-salt"}`,
		},
	}

	makeSegmentJSONForClause := func(clauseJSON string) string {
		return `{"rules": [ {"id": "", "clauses": [` + clauseJSON + `]} ]}`
	}
	for _, cp := range makeClauseSerializationTestParams() {
		sp := segmentSerializationTestParams{
			name: "segment rule clause " + cp.name,
			segment: Segment{
				Rules: []SegmentRule{
					{
						Clauses: []Clause{cp.clause},
					},
				},
			},
			jsonString: makeSegmentJSONForClause(cp.jsonString),
		}
		for _, alt := range cp.jsonAltInputs {
			sp.jsonAltInputs = append(sp.jsonAltInputs, makeSegmentJSONForClause(alt))
		}
		ret = append(ret, sp)
	}

	return ret
}

func makeClauseSerializationTestParams() []clauseSerializationTestParams {
	return []clauseSerializationTestParams{
		{
			name: "simple",
			clause: Clause{
				Attribute: ldattr.NewLiteralRef("key"),
				Op:        OperatorIn,
				Values:    []ldvalue.Value{ldvalue.String("a")},
			},
			jsonString:    `{"attribute": "key", "op": "in", "values": ["a"], "negate": false}`,
			jsonAltInputs: []string{`{"attribute": "key", "op": "in", "values": ["a"]}`},
		},
		{
			name: "with kind",
			clause: Clause{
				ContextKind: ldcontext.Kind("org"),
				Attribute:   ldattr.NewLiteralRef("key"),
				Op:          OperatorIn,
				Values:      []ldvalue.Value{ldvalue.String("a")},
			},
			jsonString: `{"contextKind": "org", "attribute": "key", "op": "in", "values": ["a"], "negate": false}`,
		},
		{
			name: "with kind and complex attribute ref",
			clause: Clause{
				ContextKind: ldcontext.Kind("user"),
				Attribute:   ldattr.NewRef("/attr1/subprop"),
				Op:          OperatorIn,
				Values:      []ldvalue.Value{ldvalue.String("a")},
			},
			jsonString: `{"contextKind": "user", "attribute": "/attr1/subprop", "op": "in", "values": ["a"], "negate": false}`,
		},
		{
			name: "attribute is treated as plain name and not path when kind is omitted",
			clause: Clause{
				Attribute: ldattr.NewLiteralRef("/attr1/subprop"), // note NewLiteralRef, not NewRef
				Op:        OperatorIn,
				Values:    []ldvalue.Value{ldvalue.String("a")},
			},
			jsonString: `{"attribute": "/attr1/subprop", "op": "in", "values": ["a"], "negate": false}`,
		},
		{
			name: "invalid attribute ref",
			clause: Clause{
				ContextKind: ldcontext.Kind("user"),
				Attribute:   ldattr.NewRef("///"),
				Op:          OperatorIn,
				Values:      []ldvalue.Value{ldvalue.String("a")},
			},
			jsonString: `{"contextKind": "user", "attribute": "///", "op": "in", "values": ["a"], "negate": false}`,
		},
		{
			name: "with segmentMatch operator",
			clause: Clause{
				Op:     OperatorSegmentMatch,
				Values: []ldvalue.Value{ldvalue.String("a")},
			},
			jsonString: `{"attribute": "", "op": "segmentMatch", "values": ["a"], "negate": false}`,
			// note, attribute is serialized as "" in this case, not omitted
		},
		{
			name: "negated",
			clause: Clause{
				Attribute: ldattr.NewLiteralRef("key"),
				Op:        OperatorIn,
				Values:    []ldvalue.Value{ldvalue.String("a")},
				Negate:    true,
			},
			jsonString: `{"attribute": "key", "op": "in", "values": ["a"], "negate": true}`,
		},
	}
}

func makeRolloutSerializationTestParams() []rolloutSerializationTestParams {
	basicVariations := []WeightedVariation{{Variation: 1, Weight: 100000}}
	basicVariationsJSON := `[{"variation": 1, "weight": 100000}]`

	return []rolloutSerializationTestParams{
		{
			name:       "simple",
			rollout:    Rollout{Variations: basicVariations},
			jsonString: `{"variations": ` + basicVariationsJSON + `}`,
		},
		{
			name: "with context kind",
			rollout: Rollout{
				ContextKind: ldcontext.Kind("org"),
				Variations:  basicVariations,
			},
			jsonString: `{"contextKind": "org", "variations": ` + basicVariationsJSON + `}`,
		},
		{
			name: "with bucketBy",
			rollout: Rollout{
				BucketBy:   ldattr.NewLiteralRef("name"),
				Variations: basicVariations,
			},
			jsonString: `{"bucketBy": "name", "variations": ` + basicVariationsJSON + `}`,
		},
		{
			name: "with contextKind and complex bucketBy ref",
			rollout: Rollout{
				ContextKind: ldcontext.Kind("user"),
				BucketBy:    ldattr.NewRef("/attr1/subprop"),
				Variations:  basicVariations,
			},
			jsonString: `{"contextKind": "user", "bucketBy": "/attr1/subprop", "variations": ` + basicVariationsJSON + `}`,
		},
		{
			name: "bucketBy is treated as plain name and not path when kind is omitted",
			rollout: Rollout{
				BucketBy:   ldattr.NewLiteralRef("/attr1/subprop"), // note, NewLiteralRef rather than NewRef
				Variations: basicVariations,
			},
			jsonString: `{"bucketBy": "/attr1/subprop", "variations": ` + basicVariationsJSON + `}`,
		},
		{
			name: "invalid bucketBy ref",
			rollout: Rollout{
				ContextKind: ldcontext.Kind("user"),
				BucketBy:    ldattr.NewRef("///"),
				Variations:  basicVariations,
			},
			jsonString: `{"contextKind": "user", "bucketBy": "///", "variations": ` + basicVariationsJSON + `}`,
		},
		{
			name: "simple experiment",
			rollout: Rollout{
				Kind:       RolloutKindExperiment,
				Variations: basicVariations,
			},
			jsonString:    `{"kind": "experiment", "variations": ` + basicVariationsJSON + `}`,
			jsonAltInputs: []string{`{"kind": "experiment", "seed": null, "variations": ` + basicVariationsJSON + `}`},
		},
		{
			name: "experiment with seed",
			rollout: Rollout{
				Kind:       RolloutKindExperiment,
				Seed:       ldvalue.NewOptionalInt(12345),
				Variations: basicVariations,
			},
			jsonString: `{"kind": "experiment", "seed": 12345, "variations": ` + basicVariationsJSON + `}`,
		},
		{
			name: "experiment with untracked",
			rollout: Rollout{
				Kind: RolloutKindExperiment,
				Variations: []WeightedVariation{
					{Variation: 0, Weight: 75000},
					{Variation: 1, Weight: 25000, Untracked: true},
				},
			},
			jsonString: `{"kind": "experiment", "variations": [` +
				`{"variation": 0, "weight": 75000}, {"variation": 1, "weight": 25000, "untracked": true}]}`,
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
