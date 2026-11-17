package ldmodel

import (
	"github.com/launchdarkly/go-sdk-common/v4/ldattr"
	"github.com/launchdarkly/go-sdk-common/v4/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v4/ldtime"
	"github.com/launchdarkly/go-sdk-common/v4/ldvalue"

	"github.com/launchdarkly/go-jsonstream/v4/jreader"
)

func unmarshalFeatureFlagFromBytes(data []byte) (FeatureFlag, error) {
	r := jreader.NewReader(data)
	parsed := unmarshalFeatureFlagFromReader(&r)
	if err := r.Error(); err != nil {
		return FeatureFlag{}, jreader.ToJSONError(err, &parsed)
	}
	return parsed, nil
}

func unmarshalFeatureFlagFromReader(r *jreader.Reader) FeatureFlag {
	var parsed FeatureFlag

	readFeatureFlag(r, &parsed)
	if r.Error() == nil {
		PreprocessFlag(&parsed)
	}
	return parsed
}

func unmarshalSegmentFromBytes(data []byte) (Segment, error) {
	r := jreader.NewReader(data)
	parsed := unmarshalSegmentFromReader(&r)
	if err := r.Error(); err != nil {
		return Segment{}, jreader.ToJSONError(err, &parsed)
	}
	return parsed, nil
}

func unmarshalSegmentFromReader(r *jreader.Reader) Segment {
	var parsed Segment
	readSegment(r, &parsed)
	if r.Error() == nil {
		PreprocessSegment(&parsed)
	}
	return parsed
}

// The readers use the OrNull methods for optional properties. A null value gets the same default
// as an omitted property.
func readFeatureFlag(r *jreader.Reader, flag *FeatureFlag) {
	deprecatedClientSide := false

	for obj := r.Object(); obj.Next(); {
		name := obj.Name()
		switch string(name) {
		case "key":
			flag.Key = r.String()
		case "on":
			flag.On = r.Bool()
		case "prerequisites":
			readPrerequisites(r, &flag.Prerequisites)
		case "targets":
			readTargets(r, &flag.Targets)
		case "contextTargets":
			readTargets(r, &flag.ContextTargets)
		case "rules":
			readFlagRules(r, &flag.Rules)
		case "fallthrough":
			readVariationOrRollout(r, &flag.Fallthrough)
		case "offVariation":
			flag.OffVariation.ReadFromJSONReader(r)
		case "variations":
			readValueList(r, &flag.Variations)
		case "clientSideAvailability":
			readClientSideAvailability(r, &flag.ClientSideAvailability)
		case "clientSide":
			deprecatedClientSide, _ = r.BoolOrNull()
		case "salt":
			flag.Salt, _ = r.StringOrNull()
		case "trackEvents":
			flag.TrackEvents, _ = r.BoolOrNull()
		case "trackEventsFallthrough":
			flag.TrackEventsFallthrough, _ = r.BoolOrNull()
		case "debugEventsUntilDate":
			val, _ := r.Float64OrNull() // val will be zero if null
			flag.DebugEventsUntilDate = ldtime.UnixMillisecondTime(val)
		case "version":
			flag.Version = r.Int()
		case "deleted":
			flag.Deleted, _ = r.BoolOrNull()
		case "excludeFromSummaries":
			flag.ExcludeFromSummaries, _ = r.BoolOrNull()
		case "samplingRatio":
			if n, ok := r.IntOrNull(); ok {
				flag.SamplingRatio = ldvalue.NewOptionalInt(n)
			}
		case "migration":
			readMigration(r, flag)
		}
	}

	if !flag.ClientSideAvailability.Explicit {
		flag.ClientSideAvailability = ClientSideAvailability{
			UsingMobileKey:     true, // always assumed to be true in the old schema
			UsingEnvironmentID: deprecatedClientSide,
			Explicit:           false,
		}
	}
}

func readPrerequisites(r *jreader.Reader, out *[]Prerequisite) {
	for arr := r.ArrayOrNull(); arr.Next(); {
		var prereq Prerequisite
		for obj := r.Object(); obj.Next(); {
			switch string(obj.Name()) {
			case "key":
				prereq.Key = r.String()
			case "variation":
				prereq.Variation = r.Int()
			}
		}
		*out = append(*out, prereq)
	}
}

func readTargets(r *jreader.Reader, out *[]Target) {
	for arr := r.ArrayOrNull(); arr.Next(); {
		var t Target
		for obj := r.Object(); obj.Next(); {
			switch string(obj.Name()) {
			case "contextKind":
				kind, _ := r.StringOrNull()
				t.ContextKind = ldcontext.Kind(kind)
			case "values":
				readStringList(r, &t.Values)
			case "variation":
				t.Variation = r.Int()
			}
		}
		*out = append(*out, t)
	}
}

func readFlagRules(r *jreader.Reader, out *[]FlagRule) {
	for arr := r.ArrayOrNull(); arr.Next(); {
		rule := FlagRule{}
		for obj := r.Object(); obj.Next(); {
			switch string(obj.Name()) {
			case "id":
				rule.ID, _ = r.StringOrNull()
			case "variation":
				rule.Variation.ReadFromJSONReader(r)
			case "rollout":
				readRollout(r, &rule.Rollout)
			case "clauses":
				readClauses(r, &rule.Clauses)
			case "trackEvents":
				rule.TrackEvents, _ = r.BoolOrNull()
			}
		}
		*out = append(*out, rule)
	}
}

func readClauses(r *jreader.Reader, out *[]Clause) {
	for arr := r.ArrayOrNull(); arr.Next(); {
		var clause Clause
		var attrStr string
		for obj := r.Object(); obj.Next(); {
			switch string(obj.Name()) {
			case "contextKind":
				kind, _ := r.StringOrNull()
				clause.ContextKind = ldcontext.Kind(kind)
			case "attribute":
				attrStr, _ = r.StringOrNull()
			case "op":
				clause.Op = Operator(r.String())
			case "values":
				readValueList(r, &clause.Values)
			case "negate":
				clause.Negate, _ = r.BoolOrNull()
			}
		}
		setAttrNameOrRef(attrStr, clause.ContextKind, &clause.Attribute)
		*out = append(*out, clause)
	}
}

func readVariationOrRollout(r *jreader.Reader, out *VariationOrRollout) {
	for obj := r.Object(); obj.Next(); {
		switch string(obj.Name()) {
		case "variation":
			out.Variation.ReadFromJSONReader(r)
		case "rollout":
			readRollout(r, &out.Rollout)
		}
	}
}

func readRollout(r *jreader.Reader, out *Rollout) {
	obj := r.ObjectOrNull()
	if !obj.IsDefined() {
		*out = Rollout{}
		return
	}
	var bucketByStr string
	for obj.Next() {
		switch string(obj.Name()) {
		case "kind":
			kind, _ := r.StringOrNull()
			out.Kind = RolloutKind(kind)
		case "contextKind":
			kind, _ := r.StringOrNull()
			out.ContextKind = ldcontext.Kind(kind)
		case "variations":
			for arr := r.Array(); arr.Next(); {
				var wv WeightedVariation
				for wrObj := r.Object(); wrObj.Next(); {
					switch string(wrObj.Name()) {
					case "variation":
						wv.Variation = r.Int()
					case "weight":
						wv.Weight = r.Int()
					case "untracked":
						wv.Untracked, _ = r.BoolOrNull()
					}
				}
				out.Variations = append(out.Variations, wv)
			}
		case "bucketBy":
			bucketByStr, _ = r.StringOrNull()
		case "seed":
			if n, ok := r.IntOrNull(); ok {
				out.Seed = ldvalue.NewOptionalInt(n)
			}
		}
	}
	setAttrNameOrRef(bucketByStr, out.ContextKind, &out.BucketBy)
}

func readClientSideAvailability(r *jreader.Reader, out *ClientSideAvailability) {
	obj := r.ObjectOrNull()
	out.Explicit = obj.IsDefined()
	for obj.Next() {
		switch string(obj.Name()) {
		case "usingEnvironmentId":
			out.UsingEnvironmentID, _ = r.BoolOrNull()
		case "usingMobileKey":
			out.UsingMobileKey, _ = r.BoolOrNull()
		}
	}
}

func readMigration(r *jreader.Reader, flag *FeatureFlag) {
	flag.Migration = &MigrationFlagParameters{}

	for obj := r.ObjectOrNull(); obj.Next(); {
		if string(obj.Name()) == "checkRatio" {
			if n, ok := r.IntOrNull(); ok {
				flag.Migration.CheckRatio = ldvalue.NewOptionalInt(n)
			}
		}
	}
}

func readSegment(r *jreader.Reader, segment *Segment) {
	for obj := r.Object(); obj.Next(); {
		switch string(obj.Name()) {
		case "key":
			segment.Key = r.String()
		case "version":
			segment.Version = r.Int()
		case "generation":
			segment.Generation.ReadFromJSONReader(r)
		case "deleted":
			segment.Deleted, _ = r.BoolOrNull()
		case "included":
			readStringList(r, &segment.Included)
		case "excluded":
			readStringList(r, &segment.Excluded)
		case "includedContexts":
			readSegmentTargets(r, &segment.IncludedContexts)
		case "excludedContexts":
			readSegmentTargets(r, &segment.ExcludedContexts)
		case "rules":
			for rulesArr := r.ArrayOrNull(); rulesArr.Next(); {
				rule := SegmentRule{}
				var bucketByStr string
				for ruleObj := r.Object(); ruleObj.Next(); {
					switch string(ruleObj.Name()) {
					case "id":
						rule.ID, _ = r.StringOrNull()
					case "clauses":
						readClauses(r, &rule.Clauses)
					case "weight":
						if v, ok := r.IntOrNull(); ok {
							rule.Weight = ldvalue.NewOptionalInt(v)
						}
					case "bucketBy":
						bucketByStr, _ = r.StringOrNull()
					case "rolloutContextKind":
						kind, _ := r.StringOrNull()
						rule.RolloutContextKind = ldcontext.Kind(kind)
					}
				}
				setAttrNameOrRef(bucketByStr, rule.RolloutContextKind, &rule.BucketBy)
				segment.Rules = append(segment.Rules, rule)
			}
		case "salt":
			segment.Salt, _ = r.StringOrNull()
		case "unbounded":
			segment.Unbounded, _ = r.BoolOrNull()
		case "unboundedContextKind":
			kind, _ := r.StringOrNull()
			segment.UnboundedContextKind = ldcontext.Kind(kind)
		}
	}
}

func readSegmentTargets(r *jreader.Reader, out *[]SegmentTarget) {
	for arr := r.ArrayOrNull(); arr.Next(); {
		var t SegmentTarget
		for obj := r.Object(); obj.Next(); {
			switch string(obj.Name()) {
			case "contextKind":
				t.ContextKind = ldcontext.Kind(r.String())
			case "values":
				readStringList(r, &t.Values)
			}
		}
		*out = append(*out, t)
	}
}

func readStringList(r *jreader.Reader, out *[]string) {
	for arr := r.ArrayOrNull(); arr.Next(); {
		*out = append(*out, r.String())
	}
}

func readValueList(r *jreader.Reader, out *[]ldvalue.Value) {
	for arr := r.ArrayOrNull(); arr.Next(); {
		var v ldvalue.Value
		v.ReadFromJSONReader(r)
		*out = append(*out, v)
	}
}

func setAttrNameOrRef(value string, contextKind ldcontext.Kind, out *ldattr.Ref) {
	switch {
	case value == "":
		*out = ldattr.Ref{}
		// Note that we're not distinguishing here between an empty string and an omitted property;
		// "" would not be valid parameter to NewRef, and historically been a value LD may send for
		// these fields so we are treating either "" or null as "undefined".

	case contextKind == "":
		// If the context kind was not specified in this clause/rollout/etc., then this is old-style
		// data and we must interpret the attribute property as a plain attribute name, not an attribute
		// reference (in other words, a leading slash would be just part of the name).
		*out = ldattr.NewLiteralRef(value)

	default:
		*out = ldattr.NewRef(value)
	}
}
