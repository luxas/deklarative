package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

func fromUpstream(upstream trace.TracerProvider) TracerProvider {
	return composite(upstream, nil)
}

func composite(upstream trace.TracerProvider, underlying TracerProvider) TracerProvider {
	if tp, ok := upstream.(TracerProvider); ok {
		return tp
	}
	return &upstreamConverter{upstream, underlying}
}

var _ TracerProvider = &upstreamConverter{}

type upstreamConverter struct {
	trace.TracerProvider
	underlying TracerProvider
}

func (c *upstreamConverter) Shutdown(ctx context.Context) error {
	if shutdownable, ok := c.TracerProvider.(interface {
		Shutdown(ctx context.Context) error
	}); ok {
		return shutdownable.Shutdown(ctx)
	}
	if c.underlying != nil {
		return c.underlying.Shutdown(ctx)
	}
	return nil
}

func (c *upstreamConverter) ForceFlush(ctx context.Context) error {
	if flushable, ok := c.TracerProvider.(interface {
		ForceFlush(ctx context.Context) error
	}); ok {
		return flushable.ForceFlush(ctx)
	}
	if c.underlying != nil {
		return c.underlying.ForceFlush(ctx)
	}
	return nil
}

func (c *upstreamConverter) Enabled(ctx context.Context, cfg *TracerConfig) bool {
	if enabler, ok := c.TracerProvider.(interface {
		Enabled(ctx context.Context, cfg *TracerConfig) bool
	}); ok {
		return enabler.Enabled(ctx, cfg)
	}
	if c.underlying != nil {
		return c.underlying.Enabled(ctx, cfg)
	}
	return true
}

func (c *upstreamConverter) IsNoop() bool {
	if noopable, ok := c.TracerProvider.(interface {
		IsNoop() bool
	}); ok {
		return noopable.IsNoop()
	}
	if c.underlying != nil {
		return c.underlying.IsNoop()
	}
	return c.TracerProvider == noopProvider
}
