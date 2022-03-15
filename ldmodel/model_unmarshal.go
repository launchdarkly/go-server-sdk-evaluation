package ldmodel

import (
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldattr"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldcontext"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldtime"
	"gopkg.in/launchdarkly/go-sdk-common.v3/ldvalue"

	"gopkg.in/launchdarkly/go-jsonstream.v1/jreader"
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
			deprecatedClientSide = r.Bool()
		case "salt":
			flag.Salt = r.String()
		case "trackEvents":
			flag.TrackEvents = r.Bool()
		case "trackEventsFallthrough":
			flag.TrackEventsFallthrough = r.Bool()
		case "debugEventsUntilDate":
			val, _ := r.Float64OrNull() // val will be zero if null
			flag.DebugEventsUntilDate = ldtime.UnixMillisecondTime(val)
		case "version":
			flag.Version = r.Int()
		case "deleted":
			flag.Deleted = r.Bool()
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
				t.ContextKind = ldcontext.Kind(r.String())
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
				rule.ID = r.String()
			case "variation":
				rule.Variation.ReadFromJSONReader(r)
			case "rollout":
				readRollout(r, &rule.Rollout)
			case "clauses":
				readClauses(r, &rule.Clauses)
			case "trackEvents":
				rule.TrackEvents = r.Bool()
			}
		}
		*out = append(*out, rule)
	}
}

func readClauses(r *jreader.Reader, out *[]Clause) {
	for arr := r.ArrayOrNull(); arr.Next(); {
		var clause Clause
		for obj := r.Object(); obj.Next(); {
			switch string(obj.Name()) {
			case "contextKind":
				clause.ContextKind = ldcontext.Kind(r.String())
			case "attribute":
				readAttrRef(r, &clause.Attribute)
			case "op":
				clause.Op = Operator(r.String())
			case "values":
				readValueList(r, &clause.Values)
			case "negate":
				clause.Negate = r.Bool()
			}
		}
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
	for obj.Next() {
		switch string(obj.Name()) {
		case "kind":
			out.Kind = RolloutKind(r.String())
		case "contextKind":
			out.ContextKind = ldcontext.Kind(r.String())
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
						wv.Untracked = r.Bool()
					}
				}
				out.Variations = append(out.Variations, wv)
			}
		case "bucketBy":
			readAttrRef(r, &out.BucketBy)
		case "seed":
			if n, ok := r.IntOrNull(); ok {
				out.Seed = ldvalue.NewOptionalInt(n)
			}
		}
	}
}

func readClientSideAvailability(r *jreader.Reader, out *ClientSideAvailability) {
	obj := r.ObjectOrNull()
	out.Explicit = obj.IsDefined()
	for obj.Next() {
		switch string(obj.Name()) {
		case "usingEnvironmentId":
			out.UsingEnvironmentID = r.Bool()
		case "usingMobileKey":
			out.UsingMobileKey = r.Bool()
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
			segment.Deleted = r.Bool()
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
				for ruleObj := r.Object(); ruleObj.Next(); {
					switch string(ruleObj.Name()) {
					case "id":
						rule.ID = r.String()
					case "clauses":
						readClauses(r, &rule.Clauses)
					case "weight":
						if v, ok := r.IntOrNull(); ok {
							rule.Weight = ldvalue.NewOptionalInt(v)
						}
					case "bucketBy":
						readAttrRef(r, &rule.BucketBy)
					}
				}
				segment.Rules = append(segment.Rules, rule)
			}
		case "salt":
			segment.Salt = r.String()
		case "unbounded":
			segment.Unbounded = r.Bool()
		case "unboundedContextKind":
			segment.UnboundedContextKind = ldcontext.Kind(r.String())
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

func readAttrRef(r *jreader.Reader, out *ldattr.Ref) {
	if s, _ := r.StringOrNull(); s != "" {
		*out = ldattr.NewRef(s)
		// NewRef takes care of parsing and validating a string that could either be a simple attribute
		// name ("email") or a slash-delimited path reference ("/addresses/0/street"). Storing the
		// ldattr.Ref in the Clause, rather than just a string, saves us the work of having to do the
		// parsing and validation again each time we evaluate a flag.
		// If the string was invalid as an attribute reference (e.g. "///"), the result is that *out
		// retains the original string (so if we re-serialize it, we get what we started with), but
		// also retains state saying it is invalid (see ldattr.Ref.Err())-- so any attempt to use it
		// to look up a context value results in an immediate "not found".
	} else {
		*out = ldattr.Ref{}
		// "" is not a valid parameter to NewRef, but that has historically been a value LD may send
		// for these fields so we are treating "" as equivalent to null to mean "undefined".
	}
}
