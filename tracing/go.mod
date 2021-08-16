module github.com/luxas/deklarative-api-runtime/tracing

go 1.16

// TODO: Fix this log level propagation issue upstream.
replace github.com/go-logr/zapr => github.com/luxas/zapr v0.4.1

require (
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v1.0.0-RC2
	go.opentelemetry.io/otel/exporters/jaeger v1.0.0-RC2
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.0.0-RC2
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.0.0-RC2
	go.opentelemetry.io/otel/sdk v1.0.0-RC2
	go.opentelemetry.io/otel/trace v1.0.0-RC2
	go.uber.org/multierr v1.7.0
	go.uber.org/zap v1.19.0
)
