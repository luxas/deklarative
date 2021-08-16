package tracing

import (
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
)

// spanLogger is a composite logr.Logger implementation that registers
// keysAndValues arguments of Logger.Info and Logger.Error calls with
// the span.
type spanLogger struct {
	log           Logger
	span          Span
	keysAndValues []interface{}
}

func (l *spanLogger) Enabled() bool { return l.log.Enabled() }
func (l *spanLogger) Info(msg string, keysAndValues ...interface{}) {
	if !l.Enabled() {
		return
	}

	attrs := keysAndValuesToAttrs(append(l.keysAndValues, keysAndValues...))
	l.span.SetAttributes(attrs...)

	l.log.Info(msg, keysAndValues...)
}

func (l *spanLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	if !l.Enabled() {
		return
	}

	attrs := keysAndValuesToAttrs(append(l.keysAndValues, keysAndValues...))
	l.span.SetAttributes(attrs...)
	l.span.RecordError(err)

	l.log.Error(err, msg, keysAndValues...)
}

func (l *spanLogger) V(level int) Logger {
	return &spanLogger{
		log:           l.log.V(level),
		span:          l.span,
		keysAndValues: l.keysAndValues,
	}
}

func (l *spanLogger) WithValues(keysAndValues ...interface{}) Logger {
	return &spanLogger{
		log:           l.log.WithValues(keysAndValues...),
		span:          l.span,
		keysAndValues: append(l.keysAndValues, keysAndValues...),
	}
}

func (l *spanLogger) WithName(name string) Logger {
	return &spanLogger{
		log:           l.log.WithName(name),
		span:          l.span,
		keysAndValues: l.keysAndValues,
	}
}

func (l *spanLogger) WithCallDepth(depth int) Logger {
	if depthLog, ok := l.log.(logr.CallDepthLogger); ok {
		return depthLog.WithCallDepth(depth)
	}
	return l.log
}

func keysAndValuesToAttrs(keysAndValues []interface{}) []attribute.KeyValue {
	keyValLen := len(keysAndValues)
	if keyValLen%2 != 0 {
		return nil
	}
	attrLen := keyValLen / 2
	attrs := make([]attribute.KeyValue, attrLen)
	for i := 0; i < attrLen; i++ {
		k := keysAndValues[i*2]
		v := keysAndValues[i*2+1]

		key, ok := k.(string)
		if !ok {
			continue
		}
		attrs[i] = attribute.Any(LogAttributePrefix+key, v)
	}
	return attrs
}
