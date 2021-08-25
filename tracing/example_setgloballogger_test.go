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

func ExampleSetGlobalLogger() {
	// This is an example of how to use this framework with no TraceProvider
	// at all.

	// If not, default to the stdlib logging library with stdr wrapping
	// it such that it is logr-compliant. Log up to verbosity 1.
	stdr.SetVerbosity(1)
	log := stdr.New(golog.New(filetest.ExampleStdout, "FooLogger: ", 0))
	tracing.SetGlobalLogger(log)

	// Construct a zap logger, and a context using it.
	realLogger := tracing.ZapLogger().Example().LogUpto(1).Build()
	ctxWithLog := tracing.Context().WithLogger(realLogger).Build()

	// Call sampleInstrumentedFunc with the zap logger in the context.
	fmt.Println("realLogger (zapr) is used with ctxWithLog:")
	sampleInstrumentedFunc2(ctxWithLog, "ctxWithLog")

	// Call sampleInstrumentedFunc with no logger in the context.
	fmt.Println("Use the global stdr logger if there's no logger in the context:")
	sampleInstrumentedFunc2(context.Background(), "context.Background")

	tracing.SetGlobalLogger(logr.Discard())

	// Output:
	// realLogger (zapr) is used with ctxWithLog:
	// {"level":"info(v=0)","logger":"sampleInstrumentedFunc2","msg":"starting span"}
	// {"level":"debug(v=1)","logger":"sampleInstrumentedFunc2","msg":"got context name","context-name":"ctxWithLog"}
	// {"level":"info(v=0)","logger":"sampleInstrumentedFunc2","msg":"ending span"}
	// Use the global stdr logger if there's no logger in the context:
	// FooLogger: sampleInstrumentedFunc2 "level"=0 "msg"="starting span"
	// FooLogger: sampleInstrumentedFunc2 "level"=1 "msg"="got context name"  "context-name"="context.Background"
	// FooLogger: sampleInstrumentedFunc2 "level"=0 "msg"="ending span"
}

func sampleInstrumentedFunc2(ctx context.Context, contextName string) {
	_, span, log := tracing.Tracer().Trace(ctx, "sampleInstrumentedFunc2")
	defer span.End()

	log.V(1).Info("got context name", "context-name", contextName)
}
