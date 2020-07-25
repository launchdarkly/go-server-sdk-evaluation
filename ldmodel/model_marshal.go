package ldmodel

import (
	"gopkg.in/launchdarkly/go-sdk-common.v2/jsonstream"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

func marshalFeatureFlag(flag FeatureFlag) ([]byte, error) {
	var b jsonstream.JSONBuffer
	b.Grow(200)

	b.BeginObject()

	writeString(&b, "key", flag.Key)

	writePropIfNotNull(&b, "on", trueValueOrNull(flag.On))

	if len(flag.Prerequisites) > 0 {
		b.WriteName("prerequisites")
		b.BeginArray()
		for _, p := range flag.Prerequisites {
			b.BeginObject()
			writeString(&b, "key", p.Key)
			writeInt(&b, "variation", p.Variation)
			b.EndObject()
		}
		b.EndArray()
	}

	if len(flag.Targets) > 0 {
		b.WriteName("targets")
		b.BeginArray()
		for _, t := range flag.Targets {
			b.BeginObject()
			writeInt(&b, "variation", t.Variation)
			writeStringArray(&b, "values", t.Values)
			b.EndObject()
		}
		b.EndArray()
	}

	if len(flag.Rules) > 0 {
		b.WriteName("rules")
		b.BeginArray()
		for _, r := range flag.Rules {
			b.BeginObject()
			writeVariationOrRolloutProperties(&b, r.VariationOrRollout)
			if r.ID != "" {
				writeString(&b, "id", r.ID)
			}
			writeClauses(&b, r.Clauses)
			writePropIfNotNull(&b, "trackEvents", trueValueOrNull(r.TrackEvents))
			b.EndObject()
		}
		b.EndArray()
	}

	b.WriteName("fallthrough")
	b.BeginObject()
	writeVariationOrRolloutProperties(&b, flag.Fallthrough)
	b.EndObject()

	if flag.OffVariation != NoVariation {
		writeInt(&b, "offVariation", flag.OffVariation)
	}

	b.WriteName("variations")
	b.BeginArray()
	for _, v := range flag.Variations {
		v.WriteToJSONBuffer(&b)
	}
	b.EndArray()

	writeClientSideAvailability(&b, flag.ClientSideAvailability)

	writeString(&b, "salt", flag.Salt)

	writePropIfNotNull(&b, "trackEvents", trueValueOrNull(flag.TrackEvents))
	writePropIfNotNull(&b, "trackEventsFallthrough", trueValueOrNull(flag.TrackEventsFallthrough))

	if flag.DebugEventsUntilDate != 0 {
		b.WriteName("debugEventsUntilDate")
		b.WriteUint64(uint64(flag.DebugEventsUntilDate))
	}

	writeInt(&b, "version", flag.Version)

	writePropIfNotNull(&b, "deleted", trueValueOrNull(flag.Deleted))

	b.EndObject()

	return b.Get()
}

func marshalSegment(segment Segment) ([]byte, error) {
	var b jsonstream.JSONBuffer
	b.Grow(200)

	b.BeginObject()

	writeString(&b, "key", segment.Key)

	if len(segment.Included) > 0 {
		writeStringArray(&b, "included", segment.Included)
	}

	if len(segment.Excluded) > 0 {
		writeStringArray(&b, "excluded", segment.Excluded)
	}

	writeString(&b, "salt", segment.Salt)

	if len(segment.Rules) > 0 {
		b.WriteName("rules")
		b.BeginArray()
		for _, r := range segment.Rules {
			b.BeginObject()
			if r.ID != "" {
				writeString(&b, "id", r.ID)
			}
			writeClauses(&b, r.Clauses)
			if r.Weight >= 0 {
				writeInt(&b, "weight", r.Weight)
			}
			if r.BucketBy != "" {
				writeString(&b, "bucketBy", string(r.BucketBy))
			}
			b.EndObject()
		}
		b.EndArray()
	}

	writeInt(&b, "version", segment.Version)

	writePropIfNotNull(&b, "deleted", trueValueOrNull(segment.Deleted))

	b.EndObject()
	return b.Get()
}

func writeProp(b *jsonstream.JSONBuffer, name string, value ldvalue.Value) {
	b.WriteName(name)
	value.WriteToJSONBuffer(b)
}

func writePropIfNotNull(b *jsonstream.JSONBuffer, name string, value ldvalue.Value) {
	if !value.IsNull() {
		writeProp(b, name, value)
	}
}

func trueValueOrNull(value bool) ldvalue.Value {
	if value {
		return ldvalue.Bool(true)
	}
	return ldvalue.Null()
}

func writeInt(b *jsonstream.JSONBuffer, name string, value int) {
	b.WriteName(name)
	b.WriteInt(value)
}

func writeString(b *jsonstream.JSONBuffer, name string, value string) {
	b.WriteName(name)
	b.WriteString(value)
}

func writeStringArray(b *jsonstream.JSONBuffer, name string, values []string) {
	b.WriteName(name)
	b.BeginArray()
	for _, v := range values {
		b.WriteString(v)
	}
	b.EndArray()
}

func writeVariationOrRolloutProperties(b *jsonstream.JSONBuffer, vr VariationOrRollout) {
	if vr.Variation != NoVariation {
		writeInt(b, "variation", vr.Variation)
	}
	if len(vr.Rollout.Variations) > 0 {
		b.WriteName("rollout")
		b.BeginObject()
		b.WriteName("variations")
		b.BeginArray()
		for _, wv := range vr.Rollout.Variations {
			b.BeginObject()
			writeInt(b, "variation", wv.Variation)
			writeInt(b, "weight", wv.Weight)
			b.EndObject()
		}
		b.EndArray()
		if vr.Rollout.BucketBy != "" {
			writeString(b, "bucketBy", string(vr.Rollout.BucketBy))
		}
		b.EndObject()
	}
}

func writeClauses(b *jsonstream.JSONBuffer, clauses []Clause) {
	b.WriteName("clauses")
	b.BeginArray()
	for _, c := range clauses {
		b.BeginObject()
		writeString(b, "attribute", string(c.Attribute))
		writeString(b, "op", string(c.Op))
		b.WriteName("values")
		b.BeginArray()
		for _, v := range c.Values {
			v.WriteToJSONBuffer(b)
		}
		b.EndArray()
		writePropIfNotNull(b, "negate", trueValueOrNull(c.Negate))
		b.EndObject()
	}
	b.EndArray()
}

func writeClientSideAvailability(b *jsonstream.JSONBuffer, availability ClientSideAvailability) {
	usingEnvironmentID := trueValueOrNull(availability.UsingEnvironmentID)
	usingMobileKey := trueValueOrNull(availability.UsingMobileKey)
	if isAnyNotNull(usingEnvironmentID, usingMobileKey) {
		b.WriteName("clientSideAvailability")
		b.BeginObject()
		writePropIfNotNull(b, "usingEnvironmentId", usingEnvironmentID)
		writePropIfNotNull(b, "usingMobileKey", usingMobileKey)
		b.EndObject()
	}
	// continue to include the legacy clientSide field for use in older sdks
	writePropIfNotNull(b, "clientSide", trueValueOrNull(availability.UsingEnvironmentID))
}

func isAnyNotNull(values ...ldvalue.Value) bool {
	for _, value := range values {
		if !value.IsNull() {
			return true
		}
	}
	return false
}
