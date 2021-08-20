package tracing

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
)

//nolint:gochecknoglobals
var (
	acquireLoggerFunc   AcquireLoggerFunc = DefaultAcquireLoggerFunc
	acquireLoggerFuncMu                   = &sync.Mutex{}

	logger   = logr.Discard()
	loggerMu = &sync.Mutex{}
)

// GetGlobalTracerProvider returns the global TracerProvider registered.
// The default TracerProvider is trace.NewNoopTracerProvider().
// This is a shorthand for otel.GetTracerProvider().
func GetGlobalTracerProvider() TracerProvider { return fromUpstream(otel.GetTracerProvider()) }

// SetGlobalTracerProvider sets globally-registered TracerProvider to tp.
// This is a shorthand for otel.SetTracerProvider(tp).
func SetGlobalTracerProvider(tp TracerProvider) { otel.SetTracerProvider(tp) }

// GetGlobalLogger gets the globally-registered Logger in this package.
// The default Logger implementation is logr.Discard().
func GetGlobalLogger() Logger {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	return logger
}

// SetGlobalLogger sets the globally-registered Logger in this package.
func SetGlobalLogger(log Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	logger = log
}

// AcquireLoggerFunc represents a function that can resolve
// a Logger from the given context. Two common implementations
// are DefaultAcquireLoggerFunc and
// "sigs.k8s.io/controller-runtime/pkg/log".FromContext.
type AcquireLoggerFunc func(context.Context) Logger

// DefaultAcquireLoggerFunc is the default AcquireLoggerFunc implementation.
// It tries to resolve a logger from the given context using logr.FromContext,
// but if no Logger is registered, it defaults to GetGlobalLogger().
func DefaultAcquireLoggerFunc(ctx context.Context) Logger {
	if log := logr.FromContext(ctx); log != nil {
		return log
	}
	return GetGlobalLogger()
}

// LoggerFromContext executes the globally-registered AcquireLoggerFunc in
// this package to resolve a Logger from the context. By default,
// DefaultAcquireLoggerFunc is used which uses the Logger in the context,
// if any, or falls back to GetGlobalLogger().
//
// If you want to customize this behavior, run SetAcquireLoggerFunc().
func LoggerFromContext(ctx context.Context) Logger {
	acquireLoggerFuncMu.Lock()
	defer acquireLoggerFuncMu.Unlock()

	return acquireLoggerFunc(ctx)
}

// SetAcquireLoggerFunc sets the globally-registered AcquireLoggerFunc
// in this package. For example, fn can be DefaultAcquireLoggerFunc
// (the default) or "sigs.k8s.io/controller-runtime/pkg/log".FromContext.
func SetAcquireLoggerFunc(fn AcquireLoggerFunc) {
	acquireLoggerFuncMu.Lock()
	defer acquireLoggerFuncMu.Unlock()

	acquireLoggerFunc = fn
}
