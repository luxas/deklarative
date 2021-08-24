module github.com/luxas/deklarative/tracing

go 1.16

replace (
	// TODO: Fix this log level propagation issue upstream.
	github.com/go-logr/zapr => github.com/luxas/zapr v0.4.1
	// TODO: Remove this once https://github.com/open-telemetry/opentelemetry-go/pull/2196
	// is merged.
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace => github.com/luxas/opentelemetry-go/exporters/stdout/stdouttrace v1.0.0-RC2-fix-timestamps
)

require (
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/stdr v0.4.0
	github.com/go-logr/zapr v0.4.0
	github.com/maxbrunsfeld/counterfeiter/v6 v6.4.1
	github.com/sebdah/goldie/v2 v2.5.3
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/otel v1.0.0-RC2
	go.opentelemetry.io/otel/exporters/jaeger v1.0.0-RC2
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.0.0-RC2
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.0.0-RC2
	go.opentelemetry.io/otel/sdk v1.0.0-RC2
	go.opentelemetry.io/otel/trace v1.0.0-RC2
	go.uber.org/multierr v1.7.0
	go.uber.org/zap v1.19.0
	gopkg.in/yaml.v2 v2.4.0
)
