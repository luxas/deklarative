package tracing

import (
	"context"

	"github.com/go-logr/logr"
)

type traceEnablerFunc func(ctx context.Context, opts *TracerConfig) bool

func (f traceEnablerFunc) Enabled(ctx context.Context, opts *TracerConfig) bool {
	return f(ctx, opts)
}

// MaxDepthEnabler is a TraceEnabler that allows all spans of trace depth below and
// equal to maxDepth. This is similar to how logr.Loggers are enabled upto a given
// log level.
func MaxDepthEnabler(maxDepth Depth) TraceEnabler {
	return traceEnablerFunc(func(_ context.Context, opts *TracerConfig) bool {
		return opts.Depth <= maxDepth
	})
}

// LoggerEnabler is a TraceEnabler that allows all spans as long as the Logger
// from the context is enabled. If the Logger is logr.Discard, any trace depth
// is allowed. This is useful when the Logger is the true source of verboseness
// allowance.
func LoggerEnabler() TraceEnabler {
	return traceEnablerFunc(func(_ context.Context, opts *TracerConfig) bool {
		return opts.Logger.Enabled() || isDiscard(opts.Logger)
	})
}

func isDiscard(log Logger) bool { return log == logr.Discard() }

type traceDepthKeyStruct struct{}

var traceDepthKey = traceDepthKeyStruct{} //nolint:gochecknoglobals

func getDepth(ctx context.Context, isNewRoot bool) Depth {
	if isNewRoot {
		return 0
	}
	d, ok := ctx.Value(traceDepthKey).(Depth)
	if !ok {
		return 0
	}
	return d + 1
}

func withDepth(ctx context.Context, depth Depth) context.Context {
	return context.WithValue(ctx, traceDepthKey, depth)
}

var _ TracerProvider = &enablerProvider{}

type enablerProvider struct {
	TracerProvider
	enabler TraceEnabler
}

func (tp *enablerProvider) Enabled(ctx context.Context, cfg *TracerConfig) bool {
	return tp.enabler.Enabled(ctx, cfg)
}

type logLevelIncreaserKeyStruct struct{}

var logLevelIncreaserKey = logLevelIncreaserKeyStruct{} //nolint:gochecknoglobals

func withLogLevelIncreaser(parent context.Context, lli LogLevelIncreaser) context.Context {
	return context.WithValue(parent, logLevelIncreaserKey, lli)
}

func getLogLevelIncreaser(ctx context.Context) LogLevelIncreaser {
	lli, ok := ctx.Value(logLevelIncreaserKey).(LogLevelIncreaser)
	if ok {
		return lli
	}
	return NthLogLevelIncrease(1)
}

type logLevelIncreaserFunc func(ctx context.Context, cfg *TracerConfig) int

func (f logLevelIncreaserFunc) GetVIncrease(ctx context.Context, cfg *TracerConfig) int {
	return f(ctx, cfg)
}

// NoLogLevelIncrease returns a LogLevelIncreaser that never bumps the verbosity,
// regardless of how deep traces there are.
func NoLogLevelIncrease() LogLevelIncreaser {
	return logLevelIncreaserFunc(func(ctx context.Context, cfg *TracerConfig) int {
		return 0
	})
}

// NthLogLevelIncrease returns a LogLevelIncreaser that increases the verbosity
// of the logger once every n traces of depth.
//
// The default LogLevelIncreaser is NthLogLevelIncrease(1), which essentially
// means log = log.V(1) for each child trace.
func NthLogLevelIncrease(n uint64) LogLevelIncreaser {
	return logLevelIncreaserFunc(func(ctx context.Context, cfg *TracerConfig) int {
		if cfg.Depth == 0 {
			return 0
		}
		if uint64(cfg.Depth)%n == 0 {
			return 1
		}
		return 0
	})
}
