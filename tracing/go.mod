module github.com/weaveworks/deklarative-api-runtime/tracing

go 1.16

require (
	github.com/go-logr/logr v1.0.0-rc1
	github.com/go-logr/zapr v1.0.0
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
