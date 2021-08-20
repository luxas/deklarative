package tracing

import (
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
)

// spanLogger is a composite logr.Logger implementation that registers
// keysAndValues arguments of Logger.Info and Logger.Error calls with
// the span.
type spanLogger struct {
	// embedding is important; this automatically exposes all inherited functionality from the
	// underlying resource.
	Logger

	span          Span
	keysAndValues []interface{}
}

func (l *spanLogger) Enabled() bool { return l.Logger.Enabled() }
func (l *spanLogger) Info(msg string, keysAndValues ...interface{}) {
	if !l.Enabled() {
		return
	}

	attrs := keysAndValuesToAttrs(append(l.keysAndValues, keysAndValues...))
	if len(attrs) != 0 {
		l.span.SetAttributes(attrs...)
	}

	l.Logger.Info(msg, keysAndValues...)
}

func (l *spanLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	if !l.Enabled() {
		return
	}

	attrs := keysAndValuesToAttrs(append(l.keysAndValues, keysAndValues...))
	if len(attrs) != 0 {
		l.span.SetAttributes(attrs...)
	}
	l.span.RecordError(err)

	l.Logger.Error(err, msg, keysAndValues...)
}

func (l *spanLogger) V(level int) Logger {
	return &spanLogger{
		Logger:        l.Logger.V(level),
		span:          l.span,
		keysAndValues: l.keysAndValues,
	}
}

func (l *spanLogger) WithValues(keysAndValues ...interface{}) Logger {
	return &spanLogger{
		Logger:        l.Logger.WithValues(keysAndValues...),
		span:          l.span,
		keysAndValues: append(l.keysAndValues, keysAndValues...),
	}
}

func (l *spanLogger) WithName(name string) Logger {
	return &spanLogger{
		Logger:        l.Logger.WithName(name),
		span:          l.span,
		keysAndValues: l.keysAndValues,
	}
}

func (l *spanLogger) WithCallDepth(depth int) Logger {
	if depthLog, ok := l.Logger.(logr.CallDepthLogger); ok {
		return depthLog.WithCallDepth(depth)
	}
	return l.Logger
}

func keysAndValuesToAttrs(keysAndValues []interface{}) []attribute.KeyValue {
	keyValLen := len(keysAndValues)
	if keyValLen%2 != 0 {
		// match zap behavior of "odd number of arguments passed as key-value pairs for logging"
		return nil
	}
	attrLen := keyValLen / 2
	attrs := make([]attribute.KeyValue, attrLen)
	for i := 0; i < attrLen; i++ {
		k := keysAndValues[i*2]
		v := keysAndValues[i*2+1]

		key, ok := k.(string)
		if !ok {
			// match zap behavior of "non-string key argument passed to logging, ignoring all later arguments"
			return nil
		}
		attrs[i] = attribute.Any(LogAttributePrefix+key, v)
	}
	return attrs
}
