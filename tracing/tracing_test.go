package tracing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

func replaceLinePattern(substr string) string {
	return "(?m)^.*" + substr + ".*$[\r\n]+"
}

// used to replace the source code lines that are output in the stack trace.
//nolint:gochecknoglobals
var (
	logReplacePattern     = regexp.MustCompile(replaceLinePattern("deklarative-api-runtime/tracing/tracing"))
	spanIDReplacePattern  = regexp.MustCompile(replaceLinePattern(`"SpanID"`))
	traceIDReplacePattern = regexp.MustCompile(replaceLinePattern(`"TraceID"`))
	timeReplacePattern    = regexp.MustCompile(replaceLinePattern(`Time`))
	traceReplacePatterns  = []*regexp.Regexp{spanIDReplacePattern, traceIDReplacePattern, timeReplacePattern}
)

func TestFuncTracer(t *testing.T) {

	// Register the global logging tracerprovider, capture "stdout" to traceBuf
	var traceBuf bytes.Buffer
	err := NewBuilder().
		RegisterStdoutExporter(stdouttrace.WithWriter(&traceBuf)).
		WithLogging(true).
		InstallGlobally()
	assert.Nil(t, err)

	var logBuf bytes.Buffer
	log := HumanReadableLogger(&logBuf, zapcore.InfoLevel)

	ctx := Context(true)
	ctx = logr.NewContext(ctx, log)
	err = doWork(ctx, t)
	assert.ErrorIs(t, err, errSomeOperation)

	wantLog, err := os.ReadFile("testdata/log.txt")
	assert.Nil(t, err)

	gotLog := logReplacePattern.ReplaceAllString(logBuf.String(), "")
	assert.Equal(t, string(wantLog), gotLog)

	wantTrace, err := os.ReadFile("testdata/trace.txt")
	assert.Nil(t, err)

	err = ForceFlushGlobal(context.Background(), 0)
	assert.Nil(t, err)

	gotTrace := traceBuf.String()
	for _, replacePattern := range traceReplacePatterns {
		gotTrace = replacePattern.ReplaceAllString(gotTrace, "")
	}
	assert.Equal(t, string(wantTrace), gotTrace)

	err = ShutdownGlobal(context.Background(), 0)
	assert.Nil(t, err)
}

func doWork(ctx context.Context, t *testing.T) error { //nolint:thelper
	return FromContext(ctx, "worker").TraceFunc(ctx, "doWork",
		func(ctx context.Context, span trace.Span) error {
			result := "result"
			span.SetAttributes(attribute.String("result", result))

			assert.True(t, span.IsRecording())

			return someOperation(ctx)
		}, trace.WithAttributes(attribute.Bool("hello", true))).Register()
}

var errSomeOperation = errors.New("some operation failed")

func someOperation(ctx context.Context) error {
	_ = FromContextUnnamed(ctx).TraceFunc(ctx, "someOperationPre",
		func(ctx context.Context, span trace.Span) error {
			span.SetAttributes(attribute.Array("arr", []string{"foo", "bar"}))

			return nil
		}).Register()

	return FromContext(ctx, "errorOperator").TraceFunc(ctx, "",
		func(ctx context.Context, span trace.Span) error {

			span.SetName("newname")
			span.SetStatus(codes.Ok, "description: status is ok")

			return errSomeOperation
		}).RegisterCustom(func(span trace.Span, err error) {
		// just register an event
		span.AddEvent("SomeOperationError")
	})
}

func Test_tracerName(t *testing.T) {
	tests := []struct {
		obj  interface{}
		want string
	}{
		{"foo", "foo"},
		{trNamed{"bar"}, "bar"},
		{nil, ""},
		{bytes.NewBuffer(nil), "*bytes.Buffer"},
		{os.Stdin, "os.Stdin"},
		{os.Stdout, "os.Stdout"},
		{os.Stderr, "os.Stderr"},
		{io.Discard, "io.Discard"},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			assert.Equal(t, tt.want, tracerName(tt.obj))
		})
	}
}

type trNamed struct{ name string }

func (t trNamed) TracerName() string { return t.name }

func Test_funcTracer_fmtSpanName(t *testing.T) {
	tests := []struct {
		tracerName string
		fnName     string
		want       string
	}{
		{tracerName: "Tracer", fnName: "Func", want: "Tracer.Func"},
		{tracerName: "", fnName: "Func", want: "Func"},
		{tracerName: "Tracer", fnName: "", want: "Tracer"},
		{tracerName: "", fnName: "", want: "<unnamed_span>"},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			assert.Equal(t, tt.want, fmtSpanName(tt.tracerName, tt.fnName))
		})
	}
}
