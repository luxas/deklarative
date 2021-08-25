package tracing_test

import (
	"context"
	"fmt"
	golog "log"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/luxas/deklarative/tracing"
	"github.com/luxas/deklarative/tracing/filetest"
)

func myAcquire(ctx context.Context) tracing.Logger {
	// If there is a logger in the context, use it
	if log := logr.FromContext(ctx); log != nil {
		return log
	}

	// If not, default to the stdlib logging library with stdr wrapping
	// it such that it is logr-compliant. Log up to verbosity 1.
	stdr.SetVerbosity(1)
	return stdr.New(golog.New(filetest.ExampleStdout, "FooLogger: ", 0))
}

func ExampleSetAcquireLoggerFunc() {
	// This is an example of how to use this framework with no TraceProvider
	// at all.

	// Use myAcquire to resolve a logger from the context
	tracing.SetAcquireLoggerFunc(myAcquire)

	// Construct a zap logger, and a context using it.
	realLogger := tracing.ZapLogger().Example().LogUpto(1).Build()
	ctxWithLog := tracing.Context().WithLogger(realLogger).Build()

	// Call sampleInstrumentedFunc with the zap logger in the context.
	fmt.Println("realLogger (zapr) is used with ctxWithLog:")
	sampleInstrumentedFunc(ctxWithLog, "ctxWithLog")

	// Call sampleInstrumentedFunc with no logger in the context.
	fmt.Println("myAcquire defaults to stdr if there's no logger in the context:")
	sampleInstrumentedFunc(context.Background(), "context.Background")

	// Output:
	// realLogger (zapr) is used with ctxWithLog:
	// {"level":"info(v=0)","logger":"sampleInstrumentedFunc","msg":"starting span"}
	// {"level":"debug(v=1)","logger":"sampleInstrumentedFunc","msg":"got context name","context-name":"ctxWithLog"}
	// {"level":"info(v=0)","logger":"sampleInstrumentedFunc","msg":"ending span"}
	// myAcquire defaults to stdr if there's no logger in the context:
	// FooLogger: sampleInstrumentedFunc "level"=0 "msg"="starting span"
	// FooLogger: sampleInstrumentedFunc "level"=1 "msg"="got context name"  "context-name"="context.Background"
	// FooLogger: sampleInstrumentedFunc "level"=0 "msg"="ending span"
}

func sampleInstrumentedFunc(ctx context.Context, contextName string) {
	_, span, log := tracing.Tracer().Trace(ctx, "sampleInstrumentedFunc")
	defer span.End()

	log.V(1).Info("got context name", "context-name", contextName)
}
