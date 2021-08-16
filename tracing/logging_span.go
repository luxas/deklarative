package tracing

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// loggingSpan is a span that's logging changes to it using the
// given Logger. It is a composite Span implementation.
type loggingSpan struct {
	log   Logger
	span  Span
	err   *error
	errFn ErrRegisterFunc
}

const (
	spanNameKey              = "span-name"
	spanEventKey             = "span-event"
	spanStatusCodeKey        = "span-status-code"
	spanStatusDescriptionKey = "span-status-description"
	// SpanAttributePrefix is the prefix used when logging an attribute registered
	// with a Span.
	SpanAttributePrefix = "span-attr-"
	// LogAttributePrefix is the prefix used when registering a logged attribute
	// with a Span.
	LogAttributePrefix = "log-attr-"
)

func (s *loggingSpan) IsRecording() bool                    { return s.span.IsRecording() }
func (s *loggingSpan) SpanContext() trace.SpanContext       { return s.span.SpanContext() }
func (s *loggingSpan) TracerProvider() trace.TracerProvider { return s.span.TracerProvider() }

func (s *loggingSpan) End(options ...trace.SpanEndOption) {
	// Register the error, if any
	if s.err != nil {
		errFn := s.errFn
		if errFn == nil {
			errFn = DefaultErrRegisterFunc
		}
		errFn(*s.err, s, s.log)
	}

	s.log.Info("ending span")
	s.span.End(options...)
}

func (s *loggingSpan) AddEvent(name string, options ...trace.EventOption) {
	s.log.Info("span event", spanEventKey, name)
	s.span.AddEvent(name, options...)
}

func (s *loggingSpan) RecordError(err error, options ...trace.EventOption) {
	s.log.Error(err, "span error")
	s.span.RecordError(err, options...)
}

func (s *loggingSpan) SetStatus(code codes.Code, description string) {
	s.log.Info("span status change",
		spanStatusCodeKey, code.String(),
		spanStatusDescriptionKey, description)
	s.span.SetStatus(code, description)
}

func (s *loggingSpan) SetName(name string) {
	s.log.Info("span name change", spanNameKey, name)
	s.span.SetName(name)
}

func (s *loggingSpan) SetAttributes(kv ...attribute.KeyValue) {
	s.log.Info("span attribute change", kvListToLogAttrs(kv)...)
	s.span.SetAttributes(kv...)
}

func kvListToLogAttrs(kv []attribute.KeyValue) []interface{} {
	attrs := make([]interface{}, 0, len(kv)*2)
	for _, item := range kv {
		attrs = append(attrs, SpanAttributePrefix+string(item.Key), item.Value.AsInterface())
	}
	return attrs
}
