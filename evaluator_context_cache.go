package evaluation

import (
	"github.com/launchdarkly/go-sdk-common/v3/ldattr"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	"github.com/launchdarkly/go-server-sdk-evaluation/v3/ldmodel"
)

// contextCache holds what an evaluation reads from its context, so that each individual context
// and each attribute value is resolved once rather than at every clause, target, segment and
// rollout that reads it. The accessors of ldcontext.Context have value receivers, so every
// IndividualContextByKind, GetValueForRef or Key call copies the whole Context; for a flag with
// many clauses those copies dominate the cost of the evaluation.
//
// A cache belongs to one evaluation context and is only read and written by one goroutine. It is
// shared by an evaluation and its nested (prerequisite and segment) evaluations, which all read the
// same context.
type contextCache struct {
	context *ldcontext.Context
	kind    ldcontext.Kind // context.Kind(), read once

	nKinds int
	kinds  [cachedKinds]cachedKind
	// overflow is the slot for a kind that does not fit in kinds. Its attribute values are never
	// cached, and it is overwritten by the next lookup that overflows.
	overflow cachedKind

	nAttrs int
	attrs  [cachedAttrs]cachedAttr

	// Results of self-contained segments (see segmentIsSelfContained), by segment.
	nSegments int
	segments  [cachedSegments]cachedSegment
}

const (
	// A context rarely has more than a few kinds, and flags rarely reference more than a few
	// attributes; both tables are scanned linearly.
	cachedKinds    = 3
	cachedAttrs    = 8
	cachedSegments = 32
)

type cachedKind struct {
	kind    ldcontext.Kind
	context ldcontext.Context // IndividualContextByKind(kind): not IsDefined() when absent
	defined bool
	key     string
	cached  bool // in kinds (not the overflow slot), so its attribute values may be cached
}

type cachedSegment struct {
	segment  *ldmodel.Segment
	contains bool
}

type cachedAttr struct {
	kind  *cachedKind
	ref   string // ldattr.Ref.String(): references with equal strings read the same value
	value ldvalue.Value
}

func newContextCache(context *ldcontext.Context) contextCache {
	return contextCache{context: context, kind: context.Kind()}
}

// individual returns the individual context of the given kind, as IndividualContextByKind does
// (an empty kind means the default kind).
func (cc *contextCache) individual(kind ldcontext.Kind) *cachedKind {
	if kind == "" {
		kind = ldcontext.DefaultKind
	}
	for i := 0; i < cc.nKinds; i++ {
		if cc.kinds[i].kind == kind {
			return &cc.kinds[i]
		}
	}
	slot := &cc.overflow
	if cc.nKinds < cachedKinds {
		slot = &cc.kinds[cc.nKinds]
		cc.nKinds++
	}
	*slot = cachedKind{kind: kind, context: cc.context.IndividualContextByKind(kind), cached: slot != &cc.overflow}
	slot.defined = slot.context.IsDefined()
	if slot.defined {
		slot.key = slot.context.Key()
	}
	return slot
}

// key returns the key of the individual context of the given kind, as
// IndividualContextByKind(kind).Key() does, and whether that context exists.
func (cc *contextCache) key(kind ldcontext.Kind) (string, bool) {
	k := cc.individual(kind)
	return k.key, k.defined
}

// value returns k.context.GetValueForRef(ref), for a k returned by individual.
func (cc *contextCache) value(k *cachedKind, ref ldattr.Ref) ldvalue.Value {
	if !k.cached {
		return k.context.GetValueForRef(ref)
	}
	refString := ref.String()
	for i := 0; i < cc.nAttrs; i++ {
		if a := &cc.attrs[i]; a.kind == k && a.ref == refString {
			return a.value
		}
	}
	v := k.context.GetValueForRef(ref)
	if cc.nAttrs < cachedAttrs {
		cc.attrs[cc.nAttrs] = cachedAttr{kind: k, ref: refString, value: v}
		cc.nAttrs++
	}
	return v
}

// contextKey returns the key of the individual context of the given kind, as
// IndividualContextByKind(kind).Key() does, and whether that context exists.
func (es *evaluationScope) contextKey(kind ldcontext.Kind) (string, bool) {
	if es.contexts != nil {
		return es.contexts.key(kind)
	}
	if ic := es.context.IndividualContextByKind(kind); ic.IsDefined() {
		return ic.Key(), true
	}
	return "", false
}

// segmentResult returns the remembered result of a segment, if any.
func (cc *contextCache) segmentResult(s *ldmodel.Segment) (contains, ok bool) {
	for i := 0; i < cc.nSegments; i++ {
		if cc.segments[i].segment == s {
			return cc.segments[i].contains, true
		}
	}
	return false, false
}

// setSegmentResult remembers the result of a self-contained segment, while there is room.
func (cc *contextCache) setSegmentResult(s *ldmodel.Segment, contains bool) {
	if cc.nSegments < cachedSegments {
		cc.segments[cc.nSegments] = cachedSegment{segment: s, contains: contains}
		cc.nSegments++
	}
}
