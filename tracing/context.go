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
	if spanTp := fromUpstream(SpanFromContext(ctx).TracerProvider()); !spanTp.IsNoop() {
		return spanTp
	}
	return GetGlobalTracerProvider()
}

// contextWithLogger injects the given Logger into a new context
// descending from parent.
func contextWithLogger(parent context.Context, log Logger) context.Context {
	return logr.NewContext(parent, log)
}

// contextWithTracerProvider injects the given TracerProvider into a new context
// descending from parent.
func contextWithTracerProvider(parent context.Context, tp TracerProvider) context.Context {
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

func (s *tracerProviderSpan) TracerProvider() trace.TracerProvider { return s.tp }

// Context returns a new *ContextBuilder.
func Context() *ContextBuilder { return &ContextBuilder{} }

// ContextBuilder is a builder-pattern constructor for a context.Context,
// that possibly includes a TracerProvider, Logger and/or LogLevelIncreaser.
type ContextBuilder struct {
	from context.Context
	tp   TracerProvider
	log  Logger
	lli  LogLevelIncreaser
}

// From sets the "base context" to start applying context.WithValue operations
// to. By default this is context.Background().
func (b *ContextBuilder) From(ctx context.Context) *ContextBuilder {
	b.from = ctx
	return b
}

// WithTracerProvider registers a TracerProvider with the context.
func (b *ContextBuilder) WithTracerProvider(tp TracerProvider) *ContextBuilder {
	b.tp = tp
	return b
}

// WithLogger registers a Logger with the context.
func (b *ContextBuilder) WithLogger(log Logger) *ContextBuilder {
	b.log = log
	return b
}

// WithLogLevelIncreaser registers a LogLevelIncreaser with the context.
func (b *ContextBuilder) WithLogLevelIncreaser(lli LogLevelIncreaser) *ContextBuilder {
	b.lli = lli
	return b
}

// Build builds the context.
func (b *ContextBuilder) Build() context.Context {
	ctx := b.from
	if ctx == nil {
		ctx = context.Background()
	}
	if b.tp != nil {
		ctx = contextWithTracerProvider(ctx, b.tp)
	}
	if b.log != nil {
		ctx = contextWithLogger(ctx, b.log)
	}
	if b.lli != nil {
		ctx = withLogLevelIncreaser(ctx, b.lli)
	}
	return ctx
}
