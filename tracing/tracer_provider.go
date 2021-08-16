package tracing

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.uber.org/multierr"
)

// TODO: Figure out how to unit-test this creation flow, as one cannot compare the
// returned tracerProviders due to internal fields.

// ErrNoExportersProvided describes that no exporters where provided when building.
var ErrNoExportersProvided = errors.New("no exporters provided")

// Provider returns a new *TracerProviderBuilder instance.
func Provider() *TracerProviderBuilder {
	//nolint:exhaustivestruct
	return &TracerProviderBuilder{}
}

// TracerProviderBuilder is an opinionated builder-pattern constructor for a
// TracerProvider that can export spans to stdout, the Jaeger HTTP API or an
// OpenTelemetry Collector gRPC proxy.
type TracerProviderBuilder struct {
	exporters []tracesdk.SpanExporter
	errs      []error
	tpOpts    []tracesdk.TracerProviderOption
	attrs     []attribute.KeyValue
	sync      bool
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

// WithSynchronousExports allows configuring whether the exporters should export in synchronous mode
// (which must be used ONLY for testing) or (by default) the batching mode.
func (b *TracerProviderBuilder) WithSynchronousExports(sync bool) *TracerProviderBuilder {
	b.sync = sync
	return b
}

// Build builds the SDKTracerProvider.
func (b *TracerProviderBuilder) Build() (SDKTracerProvider, error) {
	// Combine and filter the errors from the exporter building
	if err := multierr.Combine(b.errs...); err != nil {
		return nil, err
	}
	if len(b.exporters) == 0 {
		return nil, ErrNoExportersProvided
	}

	// By default, set the service name to "libgitops".
	// This can be overridden through WithAttributes
	defaultAttrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String("libgitops"),
	}
	// Make sure to order the defaultAttrs first, so b.attrs can override the default ones
	//nolint:gocritic
	attrs := append(defaultAttrs, b.attrs...)

	// By default, register a resource with the given attributes
	defaultTpOpts := []tracesdk.TracerProviderOption{
		// Record information about this application in an Resource.
		tracesdk.WithResource(resource.NewWithAttributes(semconv.SchemaURL, attrs...)),
	}

	// Register all exporters with the options list
	for _, exporter := range b.exporters {
		// The non-syncing mode shall only be used in testing. The batching mode must be used in production.
		if b.sync {
			defaultTpOpts = append(defaultTpOpts, tracesdk.WithSyncer(exporter))
			continue
		}

		defaultTpOpts = append(defaultTpOpts, tracesdk.WithBatcher(exporter))
	}

	// Make sure to order the defaultTpOpts first, so b.tpOpts can override the default ones
	//nolint:gocritic
	opts := append(defaultTpOpts, b.tpOpts...)
	// Build the tracing provider
	return tracesdk.NewTracerProvider(opts...), nil
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
