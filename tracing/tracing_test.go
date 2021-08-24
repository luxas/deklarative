package tracing

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/luxas/deklarative/tracing/filetest"
	"github.com/luxas/deklarative/tracing/zaplog"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func TestIsNoop(t *testing.T) {
	ctx := context.Background()
	span := trace.SpanFromContext(ctx)
	assert.True(t, fromUpstream(span.TracerProvider()).IsNoop())
}

func TestIsDiscard(t *testing.T) {
	ctx := context.Background()
	log := logr.FromContextOrDiscard(ctx)
	assert.True(t, isDiscard(log))
}

func TestTracer(t *testing.T) {
	tests := []struct {
		name       string
		traceBuild func(g *filetest.Tester) *TracerProviderBuilder
		logBuild   func(g *filetest.Tester) *zaplog.Builder
		lli        LogLevelIncreaser
	}{
		{
			name: "TraceUptoLogger1",
			traceBuild: func(g *filetest.Tester) *TracerProviderBuilder {
				return Provider().TestJSON(g).TestYAML(g).TraceUptoLogger()
			},
			logBuild: func(g *filetest.Tester) *zaplog.Builder {
				return ZapLogger().Console().NoTimestamps().LogUpto(1).Test(g)
			},
		},
		{
			name: "Trace0DepthNoLogger",
			traceBuild: func(g *filetest.Tester) *TracerProviderBuilder {
				return Provider().TestJSON(g).TestYAML(g).TraceUpto(0)
			},
		},
		{
			name: "TraceAnyDepthLogger0",
			traceBuild: func(g *filetest.Tester) *TracerProviderBuilder {
				return Provider().TestJSON(g).TestYAML(g)
			},
			logBuild: func(g *filetest.Tester) *zaplog.Builder {
				return ZapLogger().Console().NoTimestamps().LogUpto(0).Test(g)
			},
		},
		{
			name: "NoTraceLogger2",
			logBuild: func(g *filetest.Tester) *zaplog.Builder {
				return ZapLogger().Console().NoTimestamps().LogUpto(2).Test(g)
			},
		},
		{
			name: "NoTraceLoggerNoIncrease",
			logBuild: func(g *filetest.Tester) *zaplog.Builder {
				return ZapLogger().Console().NoTimestamps().LogUpto(2).Test(g)
			},
			lli: NoLogLevelIncrease(),
		},
		{
			name: "NoTraceLoggerEverySecondTrace",
			logBuild: func(g *filetest.Tester) *zaplog.Builder {
				return ZapLogger().Console().NoTimestamps().LogUpto(2).Test(g)
			},
			lli: NthLogLevelIncrease(2),
		},
	}

	for _, rt := range tests {
		t.Run(rt.name, func(t *testing.T) {
			g := filetest.New(t, goldie.WithNameSuffix(""))
			defer g.Assert()

			tp := NoopTracerProvider()
			if rt.traceBuild != nil {
				var err error
				tp, err = rt.traceBuild(g).Build()
				require.Nil(t, err)
			}

			log := logr.Discard()
			if rt.logBuild != nil {
				log = rt.logBuild(g).Build()
			}

			ctx := Context().
				WithTracerProvider(tp).
				WithLogger(log).
				WithLogLevelIncreaser(rt.lli).
				Build()

			testCore(ctx, t, tp, log)

			assert.Nil(t, tp.ForceFlush(context.Background()))
			assert.Nil(t, tp.Shutdown(context.Background()))

			// After tp.Shutdown; there should be no more JSON output
			_, span, _ := Tracer().Trace(ctx, "afterShutdown")
			span.End()
		})
	}
}

func testCore(ctx context.Context, t *testing.T, tp TracerProvider, log Logger) { //nolint:thelper
	log.Info("executing TestTracer")

	err := doWork(ctx, t, tp.IsNoop())
	assert.ErrorIs(t, err, errSomeOperation)

	_, err = someOperation(ctx)
	assert.ErrorIs(t, err, errSomeOperation)
}

func doWork(ctx context.Context, t *testing.T, isNoop bool) (retErr error) { //nolint:thelper
	ctx, span, log := Tracer().
		Capture(&retErr).
		WithActor("worker").
		WithAttributes(attribute.Bool("hello", true)).
		Trace(ctx, "doWork")
	defer span.End()

	result := "result"
	span.SetAttributes(attribute.String("result", result))
	span.SetName("foo")
	// the description will be ignored, because the status is not Error.
	// This per the documentation of Span.SetStatus.
	span.SetStatus(codes.Ok, "this will be ignored")
	log.Info("hello from the other side", "hello", -1.2)

	// If isNoop == false, we should be recording, and vice versa
	assert.Equal(t, !isNoop, span.IsRecording())

	op, err := someOperation(ctx)
	log.Info("got operation result", "op-result", op)
	return err
}

var errSomeOperation = errors.New("some operation failed")

func someOperation(ctx context.Context) (_ int64, retErr error) {
	// You can have multiple operations within the same span
	someOpCtx, span := Tracer().Start(ctx, "someOperationPre")
	span.SetAttributes(attribute.Array("arr", []string{"foo", "bar"}))
	span.SetStatus(codes.Error, "this will be visible")

	// The default logger is configured at level 0, and we bumped to
	// level 1 in worker.doWork. The logger is configured above to log
	// levels 1 and below. If we now try to bump to level 2, it'll be
	// ignored.
	_, ignoreMeSpan, _ := Tracer().Trace(someOpCtx, "ignoreMe")
	ignoreMeSpan.SetAttributes(attribute.Array("arr", []string{"foo", "bar"}))
	ignoreMeSpan.End()

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
