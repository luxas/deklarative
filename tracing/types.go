/*
Package tracing includes higher-level tools for tracing your application's
functions.

TODO: Add more documentation here and an example.

TODO: Use this logging tracer provider to unit test the traces generated, and code executing generally.

TODO: Allow fine-grained logging levels.
*/
package tracing

import (
	"github.com/go-logr/logr"
	"github.com/luxas/deklarative-api-runtime/tracing/zaplog"
	"go.opentelemetry.io/otel/trace"
)

type (
	// TracerProvider is a symbolic link to trace.TracerProvider.
	TracerProvider = trace.TracerProvider
	// Span is a symbolic link to trace.Span.
	Span = trace.Span
	// Logger is a symbolic link to logr.Logger.
	Logger = logr.Logger
)

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

// NewZap is a shorthand for zaplog.NewZap().
//
// Refer to the zaplog package for usage details and examples.
func NewZap() *zaplog.Builder { return zaplog.NewZap() }
