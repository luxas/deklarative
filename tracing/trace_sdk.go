package tracing

import (
	"context"
	"time"
)

// SDKTracerProvider represents a TracerProvider that is generated from the OpenTelemetry
// SDK and hence can be force-flushed and shutdown (which in both cases flushes all async,
// batched traces before stopping).
type SDKTracerProvider interface {
	TracerProvider
	Shutdown(ctx context.Context) error
	ForceFlush(ctx context.Context) error
}

// SDKOperationOption represents an option for a SDK TracerProvider option.
type SDKOperationOption interface {
	applyToSDKOperation(target *sdkOperationOptions)
}

// WithTimeout returns a SDKOperationOption that gives a SDK operation a
// grace period (default timeout is 0).
//
// If timeout == 0, the operation will be done without a grace period.
// If timeout > 0, the operation will have a grace period of that period of time.
func WithTimeout(timeout time.Duration) SDKOperationOption {
	return sdkOperationOptionFunc(func(target *sdkOperationOptions) {
		target.timeout = timeout
	})
}

// Shutdown tries to convert the TracerProvider to a SDKTracerProvider to
// access its Shutdown method to make sure all traces have been flushed using the exporters
// before it's shutdown.
//
// Unless WithTracerProvider is specified, the global tracer provider is affected.
// WithTimeout can be used to give a grace period.
func Shutdown(ctx context.Context, opts ...SDKOperationOption) error {
	return callSDKProvider(ctx, opts, func(ctx context.Context, sp SDKTracerProvider) error {
		return sp.Shutdown(ctx)
	})
}

// ForceFlush tries to convert the TracerProvider to a SDKTracerProvider to
// access its ForceFlush method to make sure all traces have been flushed using the exporters.
//
// Unless WithTracerProvider is specified, the global tracer provider is affected.
// WithTimeout can be used to give a grace period.
//
// Unlike Shutdown, which also flushes the traces, the provider is still operation after this.
func ForceFlush(ctx context.Context, opts ...SDKOperationOption) error {
	return callSDKProvider(ctx, opts, func(ctx context.Context, sp SDKTracerProvider) error {
		return sp.ForceFlush(ctx)
	})
}

func callSDKProvider(ctx context.Context, opts []SDKOperationOption, fn func(context.Context, SDKTracerProvider) error) error {
	o := (&sdkOperationOptions{
		tp: GetGlobalTracerProvider(),
	}).applyOptions(opts)

	p, ok := o.tp.(SDKTracerProvider)
	if !ok {
		return nil
	}

	if o.timeout != 0 {
		// Do not make the application hang when it is shutdown.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}

	return fn(ctx, p)
}

type sdkOperationOptions struct {
	timeout time.Duration
	tp      TracerProvider
}

func (o *sdkOperationOptions) applyOptions(opts []SDKOperationOption) *sdkOperationOptions {
	for _, opt := range opts {
		opt.applyToSDKOperation(o)
	}
	return o
}

// sdkOperationOptionFunc implements the SDKOperationOption
// by mutating sdkOperationOptions.
type sdkOperationOptionFunc func(target *sdkOperationOptions)

func (f sdkOperationOptionFunc) applyToSDKOperation(target *sdkOperationOptions) {
	f(target)
}
