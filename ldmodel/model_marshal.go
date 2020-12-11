package ldmodel

import (
	"gopkg.in/launchdarkly/go-jsonstream.v1/jwriter"
)

// For backward compatibility, we are only allowed to drop out properties that have default values if
// Go SDK 4.x would also have done so (since some SDKs are not tolerant of missing properties in
// general). This is true of all properties that have OptionalInt-like semantics (having either a
// numeric value or "undefined"), and properties that could be either a JSON object or null (like
// VariationOrRollout.Rollout), and the BucketBy property which has optional-string-like behavior.
// Array properties should not be dropped even if nil.
//
// Properties that did not exist prior to Go SDK v5 are always safe to drop if they have default
// values, since older SDKs will never look for them. These are:
// - FeatureFlag.ClientSideAvailability
// - Segment.Unbounded

func marshalFeatureFlag(flag FeatureFlag) ([]byte, error) {
	w := jwriter.NewWriter()
	marshalFeatureFlagToWriter(flag, &w)
	return w.Bytes(), w.Error()
}

func marshalFeatureFlagToWriter(flag FeatureFlag, w *jwriter.Writer) {
	obj := w.Object()

	obj.String("key", flag.Key)

	obj.Bool("on", flag.On)

	prereqsArr := obj.Array("prerequisites")
	for _, p := range flag.Prerequisites {
		prereqObj := prereqsArr.Object()
		prereqObj.String("key", p.Key)
		prereqObj.Int("variation", p.Variation)
		prereqObj.End()
	}
	prereqsArr.End()

	targetsArr := obj.Array("targets")
	for _, t := range flag.Targets {
		targetObj := targetsArr.Object()
		targetObj.Int("variation", t.Variation)
		writeStringArray(&targetObj, "values", t.Values)
		targetObj.End()
	}
	targetsArr.End()

	rulesArr := obj.Array("rules")
	for _, r := range flag.Rules {
		ruleObj := rulesArr.Object()
		writeVariationOrRolloutProperties(&ruleObj, r.VariationOrRollout)
		ruleObj.OptString("id", r.ID != "", r.ID)
		writeClauses(w, &ruleObj, r.Clauses)
		ruleObj.Bool("trackEvents", r.TrackEvents)
		ruleObj.End()
	}
	rulesArr.End()

	fallthroughObj := obj.Object("fallthrough")
	writeVariationOrRolloutProperties(&fallthroughObj, flag.Fallthrough)
	fallthroughObj.End()

	obj.Property("offVariation")
	flag.OffVariation.WriteToJSONWriter(w)

	variationsArr := obj.Array("variations")
	for _, v := range flag.Variations {
		v.WriteToJSONWriter(w)
	}
	variationsArr.End()

	// In the older JSON schema, ClientSideAvailability.UsingEnvironmentID was in "clientSide", and
	// ClientSideAvailability.UsingMobileKey was assumed to be true. In the newer schema, those are
	// both in a "clientSideAvailability" object.
	//
	// If ClientSideAvailability.Explicit is true, then this flag used the newer schema and should be
	// reserialized the same way. If it is false, we will reserialize with the old schema, which
	// does not include UsingMobileKey; note that in that case UsingMobileKey is assumed to be true.
	//
	// For backward compatibility with older SDKs that might be reading a flag that was serialized by
	// this SDK, we always include the older "clientSide" property if it would be true.
	if flag.ClientSideAvailability.Explicit {
		csaObj := obj.Object("clientSideAvailability")
		csaObj.Bool("usingMobileKey", flag.ClientSideAvailability.UsingMobileKey)
		csaObj.Bool("usingEnvironmentId", flag.ClientSideAvailability.UsingEnvironmentID)
		csaObj.End()
	}
	obj.Bool("clientSide", flag.ClientSideAvailability.UsingEnvironmentID)

	obj.String("salt", flag.Salt)

	obj.Bool("trackEvents", flag.TrackEvents)
	obj.Bool("trackEventsFallthrough", flag.TrackEventsFallthrough)

	obj.Property("debugEventsUntilDate")
	if flag.DebugEventsUntilDate == 0 {
		w.Null()
	} else {
		w.Float64(float64(flag.DebugEventsUntilDate))
	}

	obj.Int("version", flag.Version)

	obj.Bool("deleted", flag.Deleted)

	obj.End()
}

func marshalSegment(segment Segment) ([]byte, error) {
	w := jwriter.NewWriter()
	marshalSegmentToWriter(segment, &w)
	return w.Bytes(), w.Error()
}

func marshalSegmentToWriter(segment Segment, w *jwriter.Writer) {
	obj := w.Object()

	obj.String("key", segment.Key)
	writeStringArray(&obj, "included", segment.Included)
	writeStringArray(&obj, "excluded", segment.Excluded)
	obj.String("salt", segment.Salt)

	rulesArr := obj.Array("rules")
	for _, r := range segment.Rules {
		ruleObj := rulesArr.Object()
		ruleObj.String("id", r.ID)
		writeClauses(w, &ruleObj, r.Clauses)
		ruleObj.OptInt("weight", r.Weight >= 0, r.Weight)
		ruleObj.OptString("bucketBy", r.BucketBy != "", string(r.BucketBy))
		ruleObj.End()
	}
	rulesArr.End()

	obj.OptBool("unbounded", segment.Unbounded, segment.Unbounded)

	obj.Int("version", segment.Version)
	obj.Bool("deleted", segment.Deleted)

	obj.End()
}

func writeStringArray(obj *jwriter.ObjectState, name string, values []string) {
	arr := obj.Array(name)
	for _, v := range values {
		arr.String(v)
	}
	arr.End()
}

func writeVariationOrRolloutProperties(obj *jwriter.ObjectState, vr VariationOrRollout) {
	obj.OptInt("variation", vr.Variation.IsDefined(), vr.Variation.IntValue())
	if len(vr.Rollout.Variations) > 0 {
		rolloutObj := obj.Object("rollout")
		variationsArr := rolloutObj.Array("variations")
		for _, wv := range vr.Rollout.Variations {
			variationObj := variationsArr.Object()
			variationObj.Int("variation", wv.Variation)
			variationObj.Int("weight", wv.Weight)
			variationObj.End()
		}
		variationsArr.End()
		rolloutObj.OptString("bucketBy", vr.Rollout.BucketBy != "", string(vr.Rollout.BucketBy))
		rolloutObj.End()
	}
}

func writeClauses(w *jwriter.Writer, obj *jwriter.ObjectState, clauses []Clause) {
	clausesArr := obj.Array("clauses")
	for _, c := range clauses {
		clauseObj := clausesArr.Object()
		clauseObj.String("attribute", string(c.Attribute))
		clauseObj.String("op", string(c.Op))
		valuesArr := clauseObj.Array("values")
		for _, v := range c.Values {
			v.WriteToJSONWriter(w)
		}
		valuesArr.End()
		clauseObj.Bool("negate", c.Negate)
		clauseObj.End()
	}
	clausesArr.End()
}
