package tracing

import (
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// loggingSpan is a span that's logging changes to it using the
// given Logger. It is a composite Span implementation.
type loggingSpan struct {
	// embedding is important; this automatically exposes all inherited functionality from the
	// underlying resource.
	Span

	provider TracerProvider
	log      Logger
	err      *error
	errFn    ErrRegisterFunc
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

func (s *loggingSpan) TracerProvider() trace.TracerProvider { return s.provider }

func (s *loggingSpan) End(options ...trace.SpanEndOption) {
	// Register the error, if any
	log := logr.WithCallDepth(s.log, 1)
	if s.err != nil {
		s2 := *s
		s2.log = logr.WithCallDepth(log, 1)
		s.errFn(*s.err, &s2, log)
	}

	log.Info("ending span")
	s.Span.End(options...)
}

func (s *loggingSpan) AddEvent(name string, options ...trace.EventOption) {
	log := logr.WithCallDepth(s.log, 1)
	log.Info("span event", spanEventKey, name)
	s.Span.AddEvent(name, options...)
}

func (s *loggingSpan) RecordError(err error, options ...trace.EventOption) {
	log := logr.WithCallDepth(s.log, 1)
	log.Error(err, "span error")
	s.Span.RecordError(err, options...)
}

func (s *loggingSpan) SetStatus(code codes.Code, description string) {
	log := logr.WithCallDepth(s.log, 1)
	// The description is only included when there's an error, as per the
	// spec of Span.SetStatus.
	args := []interface{}{spanStatusCodeKey, code.String()}
	if code == codes.Error {
		args = append(args, spanStatusDescriptionKey, description)
	}
	log.Info("span status change", args...)

	s.Span.SetStatus(code, description)
}

func (s *loggingSpan) SetName(name string) {
	log := logr.WithCallDepth(s.log, 1)
	log.Info("span name change", spanNameKey, name)
	s.Span.SetName(name)
}

func (s *loggingSpan) SetAttributes(kv ...attribute.KeyValue) {
	log := logr.WithCallDepth(s.log, 1)
	log.Info("span attribute change", kvListToLogAttrs(kv)...)
	s.Span.SetAttributes(kv...)
}

func kvListToLogAttrs(kv []attribute.KeyValue) []interface{} {
	attrs := make([]interface{}, 0, len(kv)*2)
	for _, item := range kv {
		attrs = append(attrs, SpanAttributePrefix+string(item.Key), item.Value.AsInterface())
	}
	return attrs
}
