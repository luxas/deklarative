/*
Package tracing includes higher-level tools for tracing your application's
functions.

TODO: Add more documentation here and an example.

TODO: Use this logging tracer provider to unit test the traces generated, and code executing generally.

TODO: Allow fine-grained logging levels.
*/
package tracing

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/luxas/deklarative-api-runtime/tracing/zaplog"
	"go.opentelemetry.io/otel/trace"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

type (
	// Span is a symbolic link to trace.Span.
	Span = trace.Span
	// Logger is a symbolic link to logr.Logger.
	Logger = logr.Logger
)

// TraceEnabler controls if a trace with a given config should be started
// or not. If Enabled returns false, a no-op span will be returned from
// TracerBuilder.Start() and TracerBuilder.Trace(). The TraceEnabler is
// checked after the log level has been increased.
type TraceEnabler interface {
	Enabled(ctx context.Context, cfg *TracerConfig) bool
}

// LogLevelIncreaser controls how much the verbosity of a Logger should be
// bumped for a given trace configuration before starting the trace. This
// is run for each started trace.
type LogLevelIncreaser interface {
	GetVIncrease(ctx context.Context, cfg *TracerConfig) int
}

// TracerProvider represents a TracerProvider that is generated from the OpenTelemetry
// SDK and hence can be force-flushed and shutdown (which in both cases flushes all async,
// batched traces before stopping). The TracerProvider also controls which traces shall
// be started and which should not.
type TracerProvider interface {
	trace.TracerProvider

	// SDK operations

	Shutdown(ctx context.Context) error
	ForceFlush(ctx context.Context) error

	// IsNoop returns whether this is a no-op TracerProvider that does nothing.
	IsNoop() bool

	// TraceEnabler lets the provider control what spans shall be started.
	TraceEnabler
}

// Depth means "how many parent spans do I have?" for a Span.
// If this is a root span, depth is zero.
type Depth uint64

// TracerConfig is a collection of all the data that is present before starting
// a span in TracerBuilder.Start() and TracerBuilder.Trace(). This information
// can be used to make policy decisions in for example TraceEnabler or LogLevelIncreaser.
type TracerConfig struct {
	*trace.TracerConfig
	*trace.SpanConfig

	TracerName string
	FuncName   string

	Provider TracerProvider
	Depth    Depth

	Logger            Logger
	LogLevelIncreaser LogLevelIncreaser
}

// SpanName combines the TracerName and FuncName to yield a span name.
func (tc *TracerConfig) SpanName() string {
	return fmtSpanName(tc.TracerName, tc.FuncName)
}

// ErrRegisterFunc can register the error captured at the end of a
// function using TracerBuilder.Capture(*error) with the span.
//
// Depending on the error, one might want to call span.RecordError,
// span.AddEvent, or just log the error.
type ErrRegisterFunc func(err error, span Span, log Logger)

// DefaultErrRegisterFunc registers the error with the span using span.RecordError(err)
// if the error is non-nil.
func DefaultErrRegisterFunc(err error, span Span, log Logger) {
	if err != nil {
		span.RecordError(err)
	}
}

// ZapLogger is a shorthand for zaplog.NewZap().
//
// Refer to the zaplog package for usage details and examples.
func ZapLogger() *zaplog.Builder { return zaplog.NewZap() }

// NoopTracerProvider returns a TracerProvider that returns IsNoop == true,
// and creates spans that do nothing.
func NoopTracerProvider() TracerProvider { return fromUpstream(noopProvider) }
