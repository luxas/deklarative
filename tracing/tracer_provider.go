package tracing

import (
	"context"
	"io"
	"math/rand"
	"sync"

	"github.com/luxas/deklarative-api-runtime/tracing/filetest"
	"github.com/luxas/deklarative-api-runtime/tracing/traceyaml"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
)

// TODO: Figure out how to unit-test this creation flow, as one cannot compare the
// returned tracerProviders due to internal fields.

// CompositeTracerProviderFunc builds a composite TracerProvider from the given
// SDKTracerProvider. If the returned TracerProvider implements SDKTracerProvider,
// it'll be used as-is. If the returned TracerProvider doesn't implement Shutdown or
// ForceFlush, the "parent" SDKTracerProvider will be used.
type CompositeTracerProviderFunc func(TracerProvider) trace.TracerProvider

// Provider returns a new *TracerProviderBuilder instance.
func Provider() *TracerProviderBuilder {
	return &TracerProviderBuilder{}
}

// TracerProviderBuilder is an opinionated builder-pattern constructor for a
// TracerProvider that can export spans to stdout, the Jaeger HTTP API or an
// OpenTelemetry Collector gRPC proxy.
type TracerProviderBuilder struct {
	exporters    []tracesdk.SpanExporter
	errs         []error
	tpOpts       []tracesdk.TracerProviderOption
	attrs        []attribute.KeyValue
	sync         bool
	compositeFns []CompositeTracerProviderFunc
}

// WithInsecureOTelExporter registers an exporter to an OpenTelemetry Collector on the
// given address, which defaults to "localhost:55680" if addr is empty. The OpenTelemetry
// Collector speaks gRPC, hence, don't add any "http(s)://" prefix to addr. The OpenTelemetry
// Collector is just a proxy, it in turn can forward for example traces to Jaeger and metrics to
// Prometheus. Additional options can be supplied that can override the default behavior.
func (b *TracerProviderBuilder) WithInsecureOTelExporter(ctx context.Context, addr string, opts ...otlptracegrpc.Option) *TracerProviderBuilder {
	if len(addr) == 0 {
		addr = "localhost:55680"
	}

	defaultOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(addr),
		otlptracegrpc.WithInsecure(),
	}
	// Make sure to order the defaultOpts first, so opts can override the default ones
	opts = append(defaultOpts, opts...)
	// Run the main constructor for the otlptracegrpc exporter
	exp, err := otlptracegrpc.New(ctx, opts...)
	b.exporters = append(b.exporters, exp)
	b.errs = append(b.errs, err)
	return b
}

// WithInsecureJaegerExporter registers an exporter to Jaeger using Jaeger's own HTTP API.
// The default address is "http://localhost:14268/api/traces" if addr is left empty.
// Additional options can be supplied that can override the default behavior.
func (b *TracerProviderBuilder) WithInsecureJaegerExporter(addr string, opts ...jaeger.CollectorEndpointOption) *TracerProviderBuilder {
	defaultOpts := []jaeger.CollectorEndpointOption{}
	// Only override if addr is set. Default is "http://localhost:14268/api/traces"
	if len(addr) != 0 {
		defaultOpts = append(defaultOpts, jaeger.WithEndpoint(addr))
	}
	// Make sure to order the defaultOpts first, so opts can override the default ones
	opts = append(defaultOpts, opts...)
	// Run the main constructor for the jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(opts...))
	b.exporters = append(b.exporters, exp)
	b.errs = append(b.errs, err)
	return b
}

// WithStdoutExporter exports pretty-formatted telemetry data to os.Stdout, or another writer if
// stdouttrace.WithWriter(w) is supplied as an option. Note that stdouttrace.WithoutTimestamps() doesn't
// work due to an upstream bug in OpenTelemetry. TODO: Fix that issue upstream.
func (b *TracerProviderBuilder) WithStdoutExporter(opts ...stdouttrace.Option) *TracerProviderBuilder {
	defaultOpts := []stdouttrace.Option{
		stdouttrace.WithPrettyPrint(),
	}
	// Make sure to order the defaultOpts first, so opts can override the default ones
	opts = append(defaultOpts, opts...)
	// Run the main constructor for the stdout exporter
	exp, err := stdouttrace.New(opts...)
	b.exporters = append(b.exporters, exp)
	b.errs = append(b.errs, err)
	return b
}

// WithOptions allows configuring the TracerProvider in various ways, for example tracesdk.WithSpanProcessor(sp)
// or tracesdk.WithIDGenerator().
func (b *TracerProviderBuilder) WithOptions(opts ...tracesdk.TracerProviderOption) *TracerProviderBuilder {
	b.tpOpts = append(b.tpOpts, opts...)
	return b
}

// WithAttributes allows registering more default attributes for traces created by this TracerProvider.
// By default semantic conventions of version v1.4.0 are used, with "service.name" => "libgitops".
func (b *TracerProviderBuilder) WithAttributes(attrs ...attribute.KeyValue) *TracerProviderBuilder {
	b.attrs = append(b.attrs, attrs...)
	return b
}

// Synchronous allows configuring whether the exporters should export in synchronous mode,
// which is useful for avoiding flakes in unit tests. The default mode is batching.
// DO NOT use in production.
func (b *TracerProviderBuilder) Synchronous() *TracerProviderBuilder {
	b.sync = true
	return b
}

// Composite builds a composite TracerProvider from the resulting SDKTracerProvider
// when Build() is called. If the returned TracerProvider implements SDKTracerProvider,
// it'll be used as-is. If the returned TracerProvider doesn't implement Shutdown or
// ForceFlush, the "parent" SDKTracerProvider will be used. It is possible to build a
// chain of composite TracerProviders by calling this function repeatedly.
func (b *TracerProviderBuilder) Composite(fn CompositeTracerProviderFunc) *TracerProviderBuilder {
	b.compositeFns = append(b.compositeFns, fn)
	return b
}

// TestYAMLTo builds a composite TracerProvider that uses traceyaml.New() to write
// trace testing YAML to writer w. See traceyaml.New for more information about how
// it works.
//
// This is useful for unit tests.
func (b *TracerProviderBuilder) TestYAMLTo(w io.Writer) *TracerProviderBuilder {
	return b.Composite(func(tp TracerProvider) trace.TracerProvider {
		return traceyaml.New(tp, w)
	})
}

// WithTraceEnabler registers a TraceEnabler that determines if tracing shall
// be enabled for a given TracerConfig.
func (b *TracerProviderBuilder) WithTraceEnabler(te TraceEnabler) *TracerProviderBuilder {
	return b.Composite(func(tp TracerProvider) trace.TracerProvider {
		return &enablerProvider{tp, te}
	})
}

// TraceUpto includes traces with depth less than or equal to the given depth
// argument.
func (b *TracerProviderBuilder) TraceUpto(depth Depth) *TracerProviderBuilder {
	return b.WithTraceEnabler(maxDepthEnabler(depth))
}

// TraceUptoLogger includes trace data as long as the logger is enabled.
// If a logger is not provided in the context (that is, it is logr.Discard),
// then there's no depth limit for the tracing.
func (b *TracerProviderBuilder) TraceUptoLogger() *TracerProviderBuilder {
	return b.WithTraceEnabler(loggerEnabler())
}

// TestYAML is a shorthand for TestYAMLTo, that writes to a testdata/ file
// with the name of the test + the ".yaml" suffix.
//
// This is useful for unit tests.
func (b *TracerProviderBuilder) TestYAML(g *filetest.Tester) *TracerProviderBuilder {
	return b.TestYAMLTo(g.Add(g.T.Name() + ".yaml").Writer())
}

// TestJSON enables Synchronous mode, exports using WithStdoutExporter without
// timestamps to a filetest.Tester file under testdata/ with the current test
// name and a ".json" suffix. Deterministic IDs are used with a static seed.
//
// This is useful for unit tests.
func (b *TracerProviderBuilder) TestJSON(g *filetest.Tester) *TracerProviderBuilder {
	return b.Synchronous().WithStdoutExporter(
		stdouttrace.WithWriter(g.Add(g.T.Name()+".json").Writer()),
		stdouttrace.WithoutTimestamps(),
	).DeterministicIDs(1234)
}

// DeterministicIDs enables deterministic trace and span IDs. Useful for unit tests.
// DO NOT use in production.
func (b *TracerProviderBuilder) DeterministicIDs(seed int64) *TracerProviderBuilder {
	return b.WithOptions(tracesdk.WithIDGenerator(deterministicWithSeed(seed)))
}

// Build builds the SDKTracerProvider.
func (b *TracerProviderBuilder) Build() (TracerProvider, error) {
	// Default to discard all trace output, if no exporter is configured
	if len(b.exporters) == 0 {
		b = b.WithStdoutExporter(stdouttrace.WithWriter(io.Discard))
	}
	// Combine and filter the errors from the exporter building
	if err := multierr.Combine(b.errs...); err != nil {
		return nil, err
	}

	// By default, set the service name to "libgitops".
	// This can be overridden through WithAttributes
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String("libgitops"),
	}
	// Make sure to order the default attrs first, so b.attrs can override the default ones
	attrs = append(attrs, b.attrs...)

	// By default, register a resource with the given attributes
	tpOpts := []tracesdk.TracerProviderOption{
		// Record information about this application in an Resource.
		tracesdk.WithResource(resource.NewWithAttributes(semconv.SchemaURL, attrs...)),
	}

	// Register all exporters with the options list
	for _, exporter := range b.exporters {
		// The non-syncing mode shall only be used in testing. The batching mode must be used in production.
		if b.sync {
			tpOpts = append(tpOpts, tracesdk.WithSyncer(exporter))
			continue
		}

		tpOpts = append(tpOpts, tracesdk.WithBatcher(exporter))
	}

	// Make sure to order the defaultTpOpts first, so b.tpOpts can override the default ones
	tpOpts = append(tpOpts, b.tpOpts...)
	// Build the tracing provider
	sdktp := tracesdk.NewTracerProvider(tpOpts...)

	// Compose a set of SDKTracerProviders on top of each other
	tp := fromUpstream(sdktp)
	for _, fn := range b.compositeFns {
		tp = composite(fn(tp), tp)
	}
	return tp, nil
}

// InstallGlobally builds the TracerProvider and registers it globally using otel.SetTracerProvider(tp).
func (b *TracerProviderBuilder) InstallGlobally() error {
	// First, build the tracing provider...
	tp, err := b.Build()
	if err != nil {
		return err
	}
	// ... and register it globally
	SetGlobalTracerProvider(tp)
	return nil
}

type deterministicIDGenerator struct {
	mu  *sync.Mutex
	rnd *rand.Rand
}

func (g *deterministicIDGenerator) NewSpanID(context.Context, trace.TraceID) trace.SpanID {
	g.mu.Lock()
	defer g.mu.Unlock()
	sid := trace.SpanID{}
	_, _ = g.rnd.Read(sid[:])
	return sid
}

func (g *deterministicIDGenerator) NewIDs(context.Context) (trace.TraceID, trace.SpanID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	tid := trace.TraceID{}
	_, _ = g.rnd.Read(tid[:])
	sid := trace.SpanID{}
	_, _ = g.rnd.Read(sid[:])
	return tid, sid
}

func deterministicWithSeed(seed int64) tracesdk.IDGenerator {
	return &deterministicIDGenerator{
		mu: &sync.Mutex{},
		// Use the "weak" random number generator math/rand, not the more secure
		// crypto/rand because we specifically don't want secure randomness but
		// deterministicness for unit tests.
		//nolint:gosec
		rnd: rand.New(rand.NewSource(seed)),
	}
}
