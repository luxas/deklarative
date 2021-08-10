package tracing

import (
	"context"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// LogOption applies an option to the target LogOptions struct.
type LogOption interface {
	ApplyToLog(target *LogOptions)
}

// AquireLoggerFunc represents a function that can resolve
// a logr.Logger from the given context. Two common implementations
// are logr.FromContextOrDiscard and
// "sigs.k8s.io/controller-runtime/pkg/log".FromContext.
type AquireLoggerFunc func(context.Context) logr.Logger

var _ LogOption = AquireLoggerFunc(nil)

// ApplyToLog implements LogOption.
func (f AquireLoggerFunc) ApplyToLog(target *LogOptions) { target.AquireFunc = f }

// LogOptions store options about the auto-tracing logger.
type LogOptions struct {
	// AquireFunc resolves a logger from the given context.
	// By default it uses the logr.FromContextOrDiscard
	// function. This function is used by the
	//
	// However, another good alternative would be:
	// "sigs.k8s.io/controller-runtime/pkg/log".FromContext
	// if controller-runtime is used.
	AquireFunc AquireLoggerFunc
}

func (o *LogOptions) applyOptions(opts []LogOption) *LogOptions {
	for _, opt := range opts {
		opt.ApplyToLog(o)
	}
	return o
}

// NewLoggingTracerProvider is a composite TracerProvider which automatically logs trace events
// created by trace spans using a logger given to the context using logr, or as configured by controller
// runtime.
func NewLoggingTracerProvider(tp trace.TracerProvider, opts ...LogOption) SDKTracerProvider {
	o := (&LogOptions{
		AquireFunc: logr.FromContextOrDiscard,
	}).applyOptions(opts)
	return &loggingTracerProvider{tp, o}
}

type loggingTracerProvider struct {
	tp   trace.TracerProvider
	opts *LogOptions
}

func (tp *loggingTracerProvider) Tracer(instrumentationName string, opts ...trace.TracerOption) trace.Tracer {
	tracer := tp.tp.Tracer(instrumentationName, opts...)
	return &loggingTracer{
		provider: tp,
		tracer:   tracer,
		opts:     tp.opts,
	}
}

func (tp *loggingTracerProvider) Shutdown(ctx context.Context) error {
	p, ok := tp.tp.(SDKTracerProvider)
	if !ok {
		return nil
	}
	return p.Shutdown(ctx)
}

func (tp *loggingTracerProvider) ForceFlush(ctx context.Context) error {
	p, ok := tp.tp.(SDKTracerProvider)
	if !ok {
		return nil
	}
	return p.ForceFlush(ctx)
}

type loggingTracer struct {
	provider trace.TracerProvider
	tracer   trace.Tracer
	opts     *LogOptions
}

func (t *loggingTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// Acquire the logger from either the context
	log := t.opts.AquireFunc(ctx).WithName(spanName)

	// When starting up, log all given attributes.
	spanCfg := trace.NewSpanStartConfig(opts...)
	startLog := log
	if len(spanCfg.Attributes()) != 0 {
		startLog = startLog.WithValues(kvListToLogAttrs(spanCfg.Attributes())...)
	}
	startLog.Info("starting span")

	// Call the composite tracer, but swap out the returned span for ours, both in the
	// return value and context.
	ctx, span := t.tracer.Start(ctx, spanName, opts...)
	logSpan := &loggingSpan{t.provider, log, span, spanName}
	ctx = trace.ContextWithSpan(ctx, logSpan)
	return ctx, logSpan
}

type loggingSpan struct {
	provider trace.TracerProvider
	log      logr.Logger
	span     trace.Span
	spanName string
}

const (
	spanNameKey              = "span-name"
	spanEventKey             = "span-event"
	spanStatusCodeKey        = "span-status-code"
	spanStatusDescriptionKey = "span-status-description"
	spanAttributePrefix      = "span-attr-"
)

func (s *loggingSpan) End(options ...trace.SpanEndOption) {
	s.log.Info("ending span")
	s.span.End(options...)
}

func (s *loggingSpan) AddEvent(name string, options ...trace.EventOption) {
	s.log.Info("span event", spanEventKey, name)
	s.span.AddEvent(name, options...)
}

func (s *loggingSpan) IsRecording() bool { return s.span.IsRecording() }

func (s *loggingSpan) RecordError(err error, options ...trace.EventOption) {
	s.log.Error(err, "span error")
	s.span.RecordError(err, options...)
}

func (s *loggingSpan) SpanContext() trace.SpanContext { return s.span.SpanContext() }

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

func kvListToLogAttrs(kv []attribute.KeyValue) []interface{} {
	attrs := make([]interface{}, 0, len(kv)*2)
	for _, item := range kv {
		attrs = append(attrs, spanAttributePrefix+string(item.Key), item.Value.AsInterface())
	}
	return attrs
}

func (s *loggingSpan) SetAttributes(kv ...attribute.KeyValue) {
	s.log.Info("span attribute change", kvListToLogAttrs(kv)...)
	s.span.SetAttributes(kv...)
}

func (s *loggingSpan) TracerProvider() trace.TracerProvider { return s.provider }
