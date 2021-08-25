package tracing

import (
	"context"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

//nolint:gochecknoglobals
var (
	noopProvider = trace.NewNoopTracerProvider()
)

// TracerBuilder implements trace.Tracer.
type TracerBuilder struct {
	actor interface{}
	err   *error
	errFn ErrRegisterFunc // default: DefaultErrRegisterFunc

	spanStartOpts []trace.SpanStartOption
}

var _ trace.Tracer = &TracerBuilder{}

// Tracer returns a new *TracerBuilder.
func Tracer() *TracerBuilder {
	return &TracerBuilder{errFn: DefaultErrRegisterFunc}
}

// WithActor registers an "actor" for the given function that is
// instrumented.
//
// If the function instrumented is called e.g. Read and the struct
// implementing Read is *FooReader, then *FooReader is the actor.
//
// In order to make the span and logger name "*FooReader.Read", and
// not just an ambiguous "Read", pass the *FooReader as actor here.
//
// If the actor implements TracerNamed, the return value of that will
// be returned. If actor is a string, that name is used. If actor
// is a os.Std{in,out,err} or io.Discard, those human-friendly names
// are used. Otherwise, the type name is resolved by
// fmt.Sprintf("%T", actor), which automatically registers the package
// and type name.
func (b *TracerBuilder) WithActor(actor interface{}) *TracerBuilder {
	b.actor = actor
	return b
}

// WithAttributes registers attributes that are added as
// trace.SpanStartOptions automatically, but also logged in
// the beginning using the logger, if enabled.
//
// A call to this function appends to the list of previous values.
func (b *TracerBuilder) WithAttributes(attrs ...attribute.KeyValue) *TracerBuilder {
	b.spanStartOpts = append(b.spanStartOpts, trace.WithAttributes(attrs...))
	return b
}

// Capture is used to capture a named error return value from the
// function this TracerBuilder is executing in. It is possible to
// "expose" a return value like "func foo() (retErr error) {}"
// although named returns are never used.
//
// When the deferred span.End() is called at the end of the function,
// the ErrRegisterFunc will be run for whatever error value this error
// pointer points to, including if the error value is nil.
//
// This, in combination with ErrRegisterFunc allows for seamless error
// handling for traced functions; information about the error will
// propagate both to the Span and the Logger automatically.
//
// A call to this function overwrites any previous value.
func (b *TracerBuilder) Capture(err *error) *TracerBuilder {
	b.err = err
	return b
}

// ErrRegisterFunc allows configuring what ErrRegisterFunc shall be run
// when the traced function ends, if Capture has been called.
//
// By default this is DefaultErrRegisterFunc.
//
// A call to this function overwrites any previous value.
func (b *TracerBuilder) ErrRegisterFunc(fn ErrRegisterFunc) *TracerBuilder {
	b.errFn = fn
	return b
}

// Start implements trace.Tracer. See Trace for more information about how
// this trace.Tracer works. The only difference between this function and
// Trace is the signature; Trace also returns a Logger.
func (b *TracerBuilder) Start(ctx context.Context, fnName string, opts ...trace.SpanStartOption) (context.Context, Span) {
	ctx, span, _ := b.Trace(ctx, fnName, opts...)
	return ctx, span
}

// Trace creates a new Span, derived from the given context, with a Span and Logger
// name that is a combination of the string representation of the actor (described
// in WithActor) and fnName.
//
// If WithLogger isn't specified, the logger is retrieved using LoggerFromContext.
//
// If the Logger is logr.Discard(), no logs are output. However, if a Logger is specified,
// no tracing or logging will take place if it is disabled (in other words, if this span
// is "too verbose") for the Logger configuration.
//
// If opts contain any attributes, these will be logged when the span starts.
//
// If WithTracerProvider isn't specified, TracerProviderFromContext is used to get
// the TracerProvider.
//
// If the Logger is not logr.Discard(), updates registered with the span are automatically
// logged with the SpanAttributePrefix prefix. And vice versa, keysAndValues given to the
// returned Logger's Info or Error method are registered with the Span with the
// LogAttributePrefix prefix.
//
// If Capture and possibly ErrRegisterFunc are set, the error return value will be
// automatically registered to the Span.
func (b *TracerBuilder) Trace(ctx context.Context, fnName string, opts ...trace.SpanStartOption) (context.Context, Span, Logger) {
	// Prepend the options from the builder, such that the options
	// specified in the params have higher priority.
	opts = append(b.spanStartOpts, opts...)
	sc := trace.NewSpanStartConfig(opts...)

	cfg := TracerConfig{
		SpanConfig:   sc,
		TracerConfig: trace.NewTracerConfig(), // TODO

		TracerName: tracerName(b.actor), // TODO: Unify funcName, actorName, spanName and tracerName
		FuncName:   fnName,

		Provider:          TracerProviderFromContext(ctx),
		Depth:             getDepth(ctx, sc.NewRoot()),
		Logger:            LoggerFromContext(ctx),
		LogLevelIncreaser: getLogLevelIncreaser(ctx),
	}

	addLevel := cfg.LogLevelIncreaser.GetVIncrease(ctx, &cfg)
	if addLevel != 0 {
		cfg.Logger = cfg.Logger.V(addLevel)
		// Register the logger with the new level with the context
		// It's important to do this at this stage such that any child
		// users of this context will also be using the same logging level
		ctx = logr.NewContext(ctx, cfg.Logger)
	}

	// Register the depth
	ctx = withDepth(ctx, cfg.Depth)

	if !cfg.Provider.Enabled(ctx, &cfg) {
		cfg.Provider = NoopTracerProvider()
	}

	// Assign a name here before using the logger,
	// but don't propagate the name downwards.
	log := cfg.Logger.WithName(cfg.SpanName())

	// Send a "span start" log entry, together with the attributes in the beginning
	// These attributes won't be shown for every log entry in this
	startLog := log
	if attrs := cfg.SpanConfig.Attributes(); len(attrs) != 0 {
		startLog = startLog.WithValues(kvListToLogAttrs(attrs)...)
	}
	startLog.Info("starting span")

	// Acquire the TracerProvider; and construct a Tracer from there
	tracer := cfg.Provider.Tracer(cfg.TracerName) // TODO: Allow registering trace.TracerOptions?

	// Call the composite tracer, but swap out the returned span for ours, both in the
	// return value and context.
	ctx, span := tracer.Start(ctx, cfg.SpanName(), opts...)

	// Construct a composite Logger that also registers information
	// to the Span.
	spanLog := &spanLogger{
		Logger: log,
		span:   span,
	}
	// Construct a composite Span that also logs using the Logger.
	logSpan := &loggingSpan{
		Span:     span,
		provider: cfg.Provider,
		log:      log,
		err:      b.err,
		errFn:    b.errFn,
	}
	// The Span needs to be re-registered with the ctx to propagate
	// downwards. The Logger is already re-registered with the Span
	// after a potential log level increase above.
	return trace.ContextWithSpan(ctx, logSpan), logSpan, spanLog
}
