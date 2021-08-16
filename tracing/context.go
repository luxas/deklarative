package tracing

import (
	"context"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/trace"
)

// SpanFromContext retrieves the currently-executing Span stored in the
// context, if any, or a no-op Span.
func SpanFromContext(ctx context.Context) Span { return trace.SpanFromContext(ctx) }

// TracerProviderFromContext retrieves the TracerProvider from the
// context. If the current Span's TracerProvider() is not the no-op
// TracerProvider returned by trace.NewNoopTracerProvider(), it is
// used, or otherwise the global from GetGlobalTracerProvider().
func TracerProviderFromContext(ctx context.Context) TracerProvider {
	if spanTp := SpanFromContext(ctx).TracerProvider(); !isNoop(spanTp) {
		return spanTp
	}
	return GetGlobalTracerProvider()
}

// ContextWithLogger injects the given Logger into a new context
// descending from parent.
func ContextWithLogger(parent context.Context, log Logger) context.Context {
	return logr.NewContext(parent, log)
}

// ContextWithTracerProvider injects the given TracerProvider into a new context
// descending from parent.
func ContextWithTracerProvider(parent context.Context, tp TracerProvider) context.Context {
	return trace.ContextWithSpan(parent, &tracerProviderSpan{
		Span: SpanFromContext(parent),
		tp:   tp,
	})
}

// tracerProviderSpan is a composite Span just returning a static TracerProvider.
// This trick allows us to register a TracerProvider with a context, without a
// new context.Context.WithValue key.
type tracerProviderSpan struct {
	Span
	tp TracerProvider
}

func (s *tracerProviderSpan) TracerProvider() TracerProvider { return s.tp }

// Context returns context.Background(), optionally with a specific Logger and
// TracerProvider registered using the WithLogger and WithTracerProvider methods.
//
// This is useful at the very root of your application, or in tests, where you
// need a starting point, a "root" context that can be passed into instrumentable
// functions, using a specific Logger or TracerProvider.
func Context(opts ...ContextOption) context.Context {
	ctx := context.Background()
	o := (&contextOptions{}).applyOptions(opts)

	if o.log != nil {
		ctx = ContextWithLogger(ctx, o.log)
	}
	if o.tp != nil {
		ctx = ContextWithTracerProvider(ctx, o.tp)
	}
	return ctx
}

// ContextOption represents an option to the Context() method.
type ContextOption interface {
	applyToContext(target *contextOptions)
}

// WithLogger registers the given Logger with Context() return value.
func WithLogger(log Logger) ContextOption {
	return contextOptionFunc(func(target *contextOptions) {
		target.log = log
	})
}

// WithTracerProvider registers the given TracerProvider with Context() return value.
func WithTracerProvider(tp TracerProvider) WithTracerProviderOption {
	return &withTracerProvider{tp}
}

// WithTracerProviderOption is a union of the ContextOption and
// SDKOperationOption interfaces, as WithTracerProvider applies to both.
type WithTracerProviderOption interface {
	ContextOption
	SDKOperationOption
}

type withTracerProvider struct{ tp TracerProvider }

func (w *withTracerProvider) applyToContext(target *contextOptions) {
	target.tp = w.tp
}

func (w withTracerProvider) applyToSDKOperation(target *sdkOperationOptions) {
	target.tp = w.tp
}

// contextOptions collects all options fields available for Context().
type contextOptions struct {
	log Logger
	tp  TracerProvider
}

func (o *contextOptions) applyOptions(opts []ContextOption) *contextOptions {
	for _, opt := range opts {
		opt.applyToContext(o)
	}
	return o
}

// contextOptionFunc implements the ContextOption by mutating contextOptions.
type contextOptionFunc func(target *contextOptions)

func (f contextOptionFunc) applyToContext(target *contextOptions) { f(target) }
