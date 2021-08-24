package traceyaml

import (
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (td *SpanInfo) newChild(spanName string, opts ...trace.SpanStartOption) *SpanInfo {
	td.mu.Lock()
	defer td.mu.Unlock()

	child := newSpanInfo(spanName, opts...)
	child.isChild = true
	td.Children = append(td.Children, child)
	return child
}

func eventConfigFrom(opts ...trace.EventOption) EventConfig {
	ec := trace.NewEventConfig(opts...)
	return EventConfig{Attributes: newAttrs(ec.Attributes())}
}

func newSpanInfo(spanName string, opts ...trace.SpanStartOption) *SpanInfo {
	return &SpanInfo{
		SpanName:    spanName,
		StartConfig: spanConfigFromStart(opts...),
		Attributes:  make(Attributes),
		mu:          &sync.Mutex{},
	}
}

func newAttrs(attrList []attribute.KeyValue) Attributes {
	attrMap := make(Attributes, len(attrList))
	attrsInto(attrList, attrMap)
	return attrMap
}

func attrsInto(attrList []attribute.KeyValue, attrMap Attributes) {
	for _, attr := range attrList {
		attrMap[string(attr.Key)] = attr.Value.AsInterface()
	}
}

func spanConfigFromStart(opts ...trace.SpanStartOption) *SpanConfig {
	if len(opts) == 0 {
		return nil
	}
	return spanConfigFrom(trace.NewSpanStartConfig(opts...))
}

func spanConfigFromEnd(opts ...trace.SpanEndOption) *SpanConfig {
	if len(opts) == 0 {
		return nil
	}
	return spanConfigFrom(trace.NewSpanEndConfig(opts...))
}

func spanConfigFrom(sc *trace.SpanConfig) *SpanConfig {
	return &SpanConfig{
		Attributes: newAttrs(sc.Attributes()),
		Links:      sc.Links(),
		NewRoot:    sc.NewRoot(),
		SpanKind:   sc.SpanKind(),
	}
}
