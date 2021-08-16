package tracing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/go-logr/logr"
	"github.com/luxas/deklarative-api-runtime/tracing/zaplog"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/trace"
)

func TestTp(t *testing.T) {
	ctx := context.Background()
	span := trace.SpanFromContext(ctx)
	assert.True(t, span.TracerProvider() == noopProvider)
}

func TestLogDiscard(t *testing.T) {
	ctx := context.Background()
	log := logr.FromContextOrDiscard(ctx)
	assert.True(t, logr.Discard() == log)
}

func replaceLinePattern(substr string) string {
	return "(?m)^.*" + substr + ".*$[\r\n]+"
}

// used to replace the source code lines that are output in the stack trace.
//nolint:gochecknoglobals
var (
	spanIDReplacePattern  = regexp.MustCompile(replaceLinePattern(`"SpanID"`))
	traceIDReplacePattern = regexp.MustCompile(replaceLinePattern(`"TraceID"`))
	timeReplacePattern    = regexp.MustCompile(replaceLinePattern(`Time`))
	traceReplacePatterns  = []*regexp.Regexp{spanIDReplacePattern, traceIDReplacePattern, timeReplacePattern}
)

func TestTracer(t *testing.T) {
	// Register the global logging tracerprovider, capture "stdout" to traceBuf
	var traceBuf bytes.Buffer
	tp, err := Provider().
		WithStdoutExporter(stdouttrace.WithWriter(&traceBuf)).
		Build()
	assert.Nil(t, err)

	var logBuf bytes.Buffer
	log := NewZap().
		Console().
		NoTimestamps().
		AtLevel(1).
		LogTo(&logBuf).
		Build()

	log.Info("executing TestTracer")

	ctx := Context(WithLogger(log), WithTracerProvider(tp))
	err = doWork(ctx, t)
	assert.ErrorIs(t, err, errSomeOperation)

	wantLog, err := os.ReadFile("testdata/log.txt")
	assert.Nil(t, err)

	gotLog := zaplog.FilterStacktraceOrigins(logBuf.Bytes())
	assert.Equal(t, string(wantLog), string(gotLog))

	wantTrace, err := os.ReadFile("testdata/trace.txt")
	assert.Nil(t, err)

	bgCtx := context.Background()
	err = ForceFlush(bgCtx, WithTracerProvider(tp))
	assert.Nil(t, err)

	gotTrace := traceBuf.String()
	for _, replacePattern := range traceReplacePatterns {
		gotTrace = replacePattern.ReplaceAllString(gotTrace, "")
	}
	assert.Equal(t, string(wantTrace), gotTrace)

	err = Shutdown(bgCtx, WithTracerProvider(tp))
	assert.Nil(t, err)
}

func doWork(ctx context.Context, t *testing.T) (retErr error) { //nolint:thelper
	ctx, span, log := Tracer().
		Capture(&retErr).
		WithActor("worker").
		WithAttributes(attribute.Bool("hello", true)).
		AddLevel(1).
		Trace(ctx, "doWork")
	defer span.End()

	result := "result"
	span.SetAttributes(attribute.String("result", result))
	log.Info("hello from the other side", "hello", -1.2)

	assert.True(t, span.IsRecording())

	op, err := someOperation(ctx)
	log.Info("got operation result", "op-result", op)
	return err
}

var errSomeOperation = errors.New("some operation failed")

func someOperation(ctx context.Context) (_ int64, retErr error) {
	// You can have multiple operations within the same span
	_, span, _ := Tracer().Trace(ctx, "someOperationPre")
	span.SetAttributes(attribute.Array("arr", []string{"foo", "bar"}))
	span.End()

	// The default logger is configured at level 0, and we bumped to
	// level 1 in worker.doWork. The logger is configured above to log
	// levels 1 and below. If we now try to bump to level 2, it'll be
	// ignored.
	_, span, _ = Tracer().AddLevel(1).Trace(ctx, "ignoreMe")
	span.SetAttributes(attribute.Array("arr", []string{"foo", "bar"}))
	span.End()

	// Show that errors can be captured although we're returning two
	// variables and not using named returns.
	_, span, _ = Tracer().
		Capture(&retErr).
		ErrRegisterFunc(func(err error, span trace.Span, log Logger) {
			// just register an event
			if errors.Is(err, errSomeOperation) {
				span.AddEvent("SomeOperationError")
				log.Info("manual entry about some operation error")
			}
		}).
		WithActor("errorOperator").
		Trace(ctx, "")
	defer span.End()

	span.SetName("newname")
	span.SetStatus(codes.Ok, "description: status is ok")

	return -1, fmt.Errorf("%w: unexpected thing happened", errSomeOperation)
}
