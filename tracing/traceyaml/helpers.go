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
	return EventConfig{Attributes: mapAttrs(ec.Attributes())}
}

func newSpanInfo(spanName string, opts ...trace.SpanStartOption) *SpanInfo {
	return &SpanInfo{
		Names:       []string{spanName},
		StartConfig: spanConfigFromStart(opts...),
		mu:          &sync.Mutex{},
	}
}

func mapAttrs(attrs []attribute.KeyValue) []Attribute {
	res := make([]Attribute, len(attrs))
	for i, attr := range attrs {
		res[i] = Attribute{
			Key:   string(attr.Key),
			Value: attr.Value.AsInterface(),
			Type:  attr.Value.Type().String(),
		}
	}
	return res
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
		Attributes: mapAttrs(sc.Attributes()),
		Links:      sc.Links(),
		NewRoot:    sc.NewRoot(),
		SpanKind:   sc.SpanKind(),
	}
}
