package ldmodel

import (
	"github.com/launchdarkly/go-jsonstream/v3/jwriter"
	"github.com/launchdarkly/go-sdk-common/v3/ldattr"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
)

// There are two JSON encodings: the full encoding and the compact encoding.
//
// Full encoding
//
// For backward compatibility, we are only allowed to drop out properties that have default values if
// Go SDK 4.x would also have done so (since some SDKs are not tolerant of missing properties in
// general). This is true of all properties that have OptionalInt-like semantics (having either a
// numeric value or "undefined"), and properties that could be either a JSON object or null (like
// VariationOrRollout.Rollout), and the BucketBy property which has optional-string-like behavior.
// Array properties should not be dropped even if nil.
//
// Properties that did not exist prior to Go SDK v5 are always safe to drop if they have default
// values, since older SDKs will never look for them. These are:
//
// - FeatureFlag.ClientSideAvailability
// - FeatureFlag.Migration
// - FeatureFlag.Migration.CheckRatio
// - FeatureFlag.SamplingRatio
// - FeatureFlag.ExcludeFromSummaries
//
// - Segment.Unbounded
//
// Compact encoding
//
// The compact encoding also drops every property whose absence the unmarshaler reads as the same
// value. This includes zero integers, false booleans, empty strings, empty arrays, and undefined
// optional values. Only SDKs that can read this encoding can use it, so LaunchDarkly sends it only
// to SDKs that request it. These properties keep a different meaning when absent, so the compact
// encoding writes them as the full encoding does:
//
// - FeatureFlag.ClientSideAvailability, if it is explicit. If it is absent, the unmarshaler sets
//   UsingMobileKey to true.
// - FeatureFlag.Migration, if it is not nil. If it is absent, the unmarshaler sets it to nil.
// - Optional integers that have a value, even if the value is zero. If an optional integer is
//   absent, the unmarshaler reads it as undefined.

// modelWriter writes data model objects as JSON in the full or the compact encoding.
type modelWriter struct {
	w       *jwriter.Writer
	compact bool
}

// include tells if a property must be written. The full encoding writes every property that it
// calls include for. The compact encoding does not write a property that has its default value.
func (m modelWriter) include(isDefault bool) bool {
	return !m.compact || !isDefault
}

func marshalFeatureFlag(flag FeatureFlag) ([]byte, error) {
	w := jwriter.NewWriter()
	marshalFeatureFlagToWriter(flag, &w)
	return w.Bytes(), w.Error()
}

func marshalFeatureFlagCompact(flag FeatureFlag) ([]byte, error) {
	w := jwriter.NewWriter()
	marshalFeatureFlagToWriterCompact(flag, &w)
	return w.Bytes(), w.Error()
}

func marshalFeatureFlagToWriter(flag FeatureFlag, w *jwriter.Writer) {
	modelWriter{w: w}.writeFeatureFlag(flag)
}

func marshalFeatureFlagToWriterCompact(flag FeatureFlag, w *jwriter.Writer) {
	modelWriter{w: w, compact: true}.writeFeatureFlag(flag)
}

func (m modelWriter) writeFeatureFlag(flag FeatureFlag) {
	obj := m.w.Object()

	if m.include(flag.Key == "") {
		obj.Name("key").String(flag.Key)
	}

	if m.include(!flag.On) {
		obj.Name("on").Bool(flag.On)
	}

	if m.include(len(flag.Prerequisites) == 0) {
		prereqsArr := obj.Name("prerequisites").Array()
		for _, p := range flag.Prerequisites {
			prereqObj := prereqsArr.Object()
			if m.include(p.Key == "") {
				prereqObj.Name("key").String(p.Key)
			}
			if m.include(p.Variation == 0) {
				prereqObj.Name("variation").Int(p.Variation)
			}
			prereqObj.End()
		}
		prereqsArr.End()
	}

	m.writeTargets(&obj, flag.Targets, "targets")
	m.writeTargets(&obj, flag.ContextTargets, "contextTargets")

	if m.include(len(flag.Rules) == 0) {
		rulesArr := obj.Name("rules").Array()
		for _, r := range flag.Rules {
			ruleObj := rulesArr.Object()
			m.writeVariationOrRolloutProperties(&ruleObj, r.VariationOrRollout)
			ruleObj.Maybe("id", r.ID != "").String(r.ID)
			m.writeClauses(&ruleObj, r.Clauses)
			if m.include(!r.TrackEvents) {
				ruleObj.Name("trackEvents").Bool(r.TrackEvents)
			}
			ruleObj.End()
		}
		rulesArr.End()
	}

	if m.include(isDefaultVariationOrRollout(flag.Fallthrough)) {
		fallthroughObj := obj.Name("fallthrough").Object()
		m.writeVariationOrRolloutProperties(&fallthroughObj, flag.Fallthrough)
		fallthroughObj.End()
	}

	if m.include(!flag.OffVariation.IsDefined()) {
		flag.OffVariation.WriteToJSONWriter(obj.Name("offVariation"))
	}

	if m.include(len(flag.Variations) == 0) {
		variationsArr := obj.Name("variations").Array()
		for _, v := range flag.Variations {
			v.WriteToJSONWriter(m.w)
		}
		variationsArr.End()
	}

	// In the older JSON schema, ClientSideAvailability.UsingEnvironmentID was in "clientSide", and
	// ClientSideAvailability.UsingMobileKey was assumed to be true. In the newer schema, those are
	// both in a "clientSideAvailability" object.
	//
	// If ClientSideAvailability.Explicit is true, then this flag used the newer schema and should be
	// reserialized the same way. If it is false, we will reserialize with the old schema, which
	// does not include UsingMobileKey; note that in that case UsingMobileKey is assumed to be true.
	//
	// For backward compatibility with older SDKs that might be reading a flag that was serialized by
	// this SDK, we always include the older "clientSide" property if it would be true. Some SDKs read
	// only "clientSide", so the compact encoding also writes it if it is true.
	if flag.ClientSideAvailability.Explicit {
		csaObj := obj.Name("clientSideAvailability").Object()
		csaObj.Name("usingMobileKey").Bool(flag.ClientSideAvailability.UsingMobileKey)
		csaObj.Name("usingEnvironmentId").Bool(flag.ClientSideAvailability.UsingEnvironmentID)
		csaObj.End()
	}
	if m.include(!flag.ClientSideAvailability.UsingEnvironmentID) {
		obj.Name("clientSide").Bool(flag.ClientSideAvailability.UsingEnvironmentID)
	}

	if m.include(flag.Salt == "") {
		obj.Name("salt").String(flag.Salt)
	}

	if m.include(!flag.TrackEvents) {
		obj.Name("trackEvents").Bool(flag.TrackEvents)
	}
	if m.include(!flag.TrackEventsFallthrough) {
		obj.Name("trackEventsFallthrough").Bool(flag.TrackEventsFallthrough)
	}

	if m.include(flag.DebugEventsUntilDate == 0) {
		obj.Name("debugEventsUntilDate").Float64OrNull(flag.DebugEventsUntilDate != 0, float64(flag.DebugEventsUntilDate))
	}

	if m.include(flag.Version == 0) {
		obj.Name("version").Int(flag.Version)
	}

	if m.include(!flag.Deleted) {
		obj.Name("deleted").Bool(flag.Deleted)
	}

	if flag.Migration != nil {
		migrationObj := obj.Name("migration").Object()

		if checkRatio, ok := flag.Migration.CheckRatio.Get(); ok {
			migrationObj.Name("checkRatio").Int(checkRatio)
		}

		migrationObj.End()
	}

	if weight, ok := flag.SamplingRatio.Get(); ok {
		obj.Name("samplingRatio").Int(weight)
	}

	if flag.ExcludeFromSummaries {
		obj.Name("excludeFromSummaries").Bool(flag.ExcludeFromSummaries)
	}

	obj.End()
}

func (m modelWriter) writeTargets(obj *jwriter.ObjectState, targets []Target, name string) {
	if !m.include(len(targets) == 0) {
		return
	}
	targetsArr := obj.Name(name).Array()
	for _, t := range targets {
		targetObj := targetsArr.Object()
		if t.ContextKind != "" {
			targetObj.Name("contextKind").String(string(t.ContextKind))
		}
		if m.include(t.Variation == 0) {
			targetObj.Name("variation").Int(t.Variation)
		}
		m.writeStringArray(&targetObj, "values", t.Values)
		targetObj.End()
	}
	targetsArr.End()
}

func marshalSegment(segment Segment) ([]byte, error) {
	w := jwriter.NewWriter()
	marshalSegmentToWriter(segment, &w)
	return w.Bytes(), w.Error()
}

func marshalSegmentCompact(segment Segment) ([]byte, error) {
	w := jwriter.NewWriter()
	marshalSegmentToWriterCompact(segment, &w)
	return w.Bytes(), w.Error()
}

func marshalSegmentToWriter(segment Segment, w *jwriter.Writer) {
	modelWriter{w: w}.writeSegment(segment)
}

func marshalSegmentToWriterCompact(segment Segment, w *jwriter.Writer) {
	modelWriter{w: w, compact: true}.writeSegment(segment)
}

func (m modelWriter) writeSegment(segment Segment) {
	obj := m.w.Object()

	if m.include(segment.Key == "") {
		obj.Name("key").String(segment.Key)
	}
	m.writeStringArray(&obj, "included", segment.Included)
	m.writeStringArray(&obj, "excluded", segment.Excluded)
	m.writeSegmentTargets(&obj, segment.IncludedContexts, "includedContexts")
	m.writeSegmentTargets(&obj, segment.ExcludedContexts, "excludedContexts")
	if m.include(segment.Salt == "") {
		obj.Name("salt").String(segment.Salt)
	}

	if m.include(len(segment.Rules) == 0) {
		rulesArr := obj.Name("rules").Array()
		for i, r := range segment.Rules {
			ruleObj := rulesArr.Object()
			if m.include(r.ID == "") {
				ruleObj.Name("id").String(r.ID)
			}
			m.writeClauses(&ruleObj, r.Clauses)
			ruleObj.Maybe("weight", r.Weight.IsDefined()).Int(r.Weight.IntValue())
			writeAttrRef(ruleObj.Maybe("bucketBy", r.BucketBy.IsDefined()), &segment.Rules[i].BucketBy, r.RolloutContextKind)
			ruleObj.Maybe("rolloutContextKind", r.RolloutContextKind != "").String(string(r.RolloutContextKind))
			ruleObj.End()
		}
		rulesArr.End()
	}

	obj.Maybe("unbounded", segment.Unbounded).Bool(segment.Unbounded)
	obj.Maybe("unboundedContextKind", segment.UnboundedContextKind != "").String(string(segment.UnboundedContextKind))

	if m.include(segment.Version == 0) {
		obj.Name("version").Int(segment.Version)
	}
	if m.include(!segment.Generation.IsDefined()) {
		segment.Generation.WriteToJSONWriter(obj.Name("generation"))
	}
	if m.include(!segment.Deleted) {
		obj.Name("deleted").Bool(segment.Deleted)
	}

	obj.End()
}

func (m modelWriter) writeSegmentTargets(obj *jwriter.ObjectState, targets []SegmentTarget, name string) {
	if !m.include(len(targets) == 0) {
		return
	}
	targetsArr := obj.Name(name).Array()
	for _, t := range targets {
		targetObj := targetsArr.Object()
		if t.ContextKind != "" {
			targetObj.Name("contextKind").String(string(t.ContextKind))
		}
		m.writeStringArray(&targetObj, "values", t.Values)
		targetObj.End()
	}
	targetsArr.End()
}

func (m modelWriter) writeStringArray(obj *jwriter.ObjectState, name string, values []string) {
	if !m.include(len(values) == 0) {
		return
	}
	arr := obj.Name(name).Array()
	for _, v := range values {
		arr.String(v)
	}
	arr.End()
}

// isDefaultVariationOrRollout tells if vr has no properties to write. The unmarshaler reads an
// absent VariationOrRollout as the same value.
func isDefaultVariationOrRollout(vr VariationOrRollout) bool {
	return !vr.Variation.IsDefined() && len(vr.Rollout.Variations) == 0
}

func (m modelWriter) writeVariationOrRolloutProperties(obj *jwriter.ObjectState, vr VariationOrRollout) {
	obj.Maybe("variation", vr.Variation.IsDefined()).Int(vr.Variation.IntValue())
	if len(vr.Rollout.Variations) > 0 {
		rolloutObj := obj.Name("rollout").Object()
		rolloutObj.Maybe("kind", vr.Rollout.Kind != "").String(string(vr.Rollout.Kind))
		rolloutObj.Maybe("contextKind", vr.Rollout.ContextKind != "").String(string(vr.Rollout.ContextKind))
		variationsArr := rolloutObj.Name("variations").Array()
		for _, wv := range vr.Rollout.Variations {
			variationObj := variationsArr.Object()
			if m.include(wv.Variation == 0) {
				variationObj.Name("variation").Int(wv.Variation)
			}
			if m.include(wv.Weight == 0) {
				variationObj.Name("weight").Int(wv.Weight)
			}
			variationObj.Maybe("untracked", wv.Untracked).Bool(wv.Untracked)
			variationObj.End()
		}
		variationsArr.End()
		rolloutObj.Maybe("seed", vr.Rollout.Seed.IsDefined()).Int(vr.Rollout.Seed.IntValue())
		writeAttrRef(rolloutObj.Maybe("bucketBy", vr.Rollout.BucketBy.IsDefined()),
			&vr.Rollout.BucketBy, vr.Rollout.ContextKind)
		rolloutObj.End()
	}
}

func (m modelWriter) writeClauses(obj *jwriter.ObjectState, clauses []Clause) {
	if !m.include(len(clauses) == 0) {
		return
	}
	clausesArr := obj.Name("clauses").Array()
	for i, c := range clauses {
		clauseObj := clausesArr.Object()
		if c.ContextKind != "" {
			clauseObj.Name("contextKind").String(string(c.ContextKind))
		}

		if !c.Attribute.IsDefined() {
			// See comments in unmarshaling logic - for consistency with LD service behavior, we serialize
			// this as an empty string even if it was undefined. The unmarshaler reads an absent attribute
			// as undefined, so the compact encoding does not write it.
			if m.include(true) {
				clauseObj.Name("attribute").String("")
			}
		} else {
			writeAttrRef(clauseObj.Name("attribute"), &clauses[i].Attribute, c.ContextKind)
		}

		if m.include(c.Op == "") {
			clauseObj.Name("op").String(string(c.Op))
		}
		if m.include(!clauses[i].hasValues()) {
			valuesArr := clauseObj.Name("values").Array()
			for v := range clauses[i].yieldValues {
				v.WriteToJSONWriter(m.w)
			}
			valuesArr.End()
		}
		if m.include(!c.Negate) {
			clauseObj.Name("negate").Bool(c.Negate)
		}
		clauseObj.End()
	}
	clausesArr.End()
}

func writeAttrRef(w *jwriter.Writer, ref *ldattr.Ref, contextKind ldcontext.Kind) {
	if contextKind == "" {
		// If there was no context kind, then we received this as old-style data in which the attribute is
		// interpreted as a plain name rather than an attribute reference. However, ref.String() will always
		// return an attribute reference which could contain escape characters. We should instead get the
		// unescaped attribute name, which AttrRef represents as the first element in a single-element path.
		w.String(ref.Component(0))
	} else {
		w.String(ref.String())
	}
}
