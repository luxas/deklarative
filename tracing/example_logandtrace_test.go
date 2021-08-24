package tracing_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	golog "log"

	"github.com/luxas/deklarative/tracing"
	"go.opentelemetry.io/otel/attribute"
)

func Example_loggingAndYAMLTrace() {
	// This example shows how tracing and logging is unified from the perspective
	// of instrumented functions myInstrumentedFunction and childInstrumentedFunction.
	//
	// When a function is traced using the tracer in this package, trace data is
	// also logged. When data is logged using the logger, the similar data is also
	// registered with the trace.

	// Make a TracerProvider writing YAML about what's happening to
	// the yamlTrace buffer.
	var yamlTrace bytes.Buffer
	tp, err := tracing.Provider().TestYAMLTo(&yamlTrace).Build()
	if err != nil {
		golog.Fatal(err)
	}

	// Make an example logger logging to os.Stdout directly.
	fmt.Println("Log representation:")
	log := tracing.ZapLogger().Example().LogUpto(1).Build()

	// Specifically use the given Logger and TracerProvider when performing this
	// trace operation by crafting a dedicated context.
	ctx := tracing.Context().WithLogger(log).WithTracerProvider(tp).Build()

	// Using the context that points to the wanted Logger and TracerProvider,
	// start instrumenting a function. myInstrumentedFunc might be a function
	// we at this point control, but it could also be a function we do not
	// control, e.g. coming from an external library.
	err = myInstrumentedFunc(ctx)
	log.Info("error is sampleErr", "is-sampleErr", errors.Is(err, errSample))

	// Shutdown the TracerProvider, and output the YAML it yielded to os.Stdout.
	if err := tp.Shutdown(ctx); err != nil {
		golog.Fatal(err)
	}
	fmt.Printf("\nYAML trace representation:\n%s", yamlTrace.String())

	// Output:
	// Log representation:
	// {"level":"info(v=0)","logger":"myInstrumentedFunc","msg":"starting span"}
	// {"level":"info(v=0)","logger":"myInstrumentedFunc","msg":"normal verbosity!"}
	// {"level":"debug(v=1)","logger":"myInstrumentedFunc","msg":"found a message","hello":"from the other side"}
	// {"level":"debug(v=1)","logger":"child-0","msg":"starting span"}
	// {"level":"debug(v=1)","logger":"child-0","msg":"span event","span-event":"DoSTH"}
	// {"level":"debug(v=1)","logger":"child-0","msg":"span attribute change","span-attr-i":0}
	// {"level":"debug(v=1)","logger":"child-0","msg":"ending span"}
	// {"level":"debug(v=1)","logger":"child-1","msg":"starting span"}
	// {"level":"debug(v=1)","logger":"child-1","msg":"span event","span-event":"DoSTH"}
	// {"level":"debug(v=1)","logger":"child-1","msg":"span attribute change","span-attr-i":1}
	// {"level":"debug(v=1)","logger":"child-1","msg":"ending span"}
	// {"level":"error","logger":"myInstrumentedFunc","msg":"span error","error":"unexpected: sample error"}
	// {"level":"info(v=0)","logger":"myInstrumentedFunc","msg":"ending span"}
	// {"level":"info(v=0)","msg":"error is sampleErr","is-sampleErr":true}
	//
	// YAML trace representation:
	// # myInstrumentedFunc
	// - spanName: myInstrumentedFunc
	//   attributes:
	//     log-attr-hello: from the other side
	//   errors:
	//   - error: 'unexpected: sample error'
	//   children:
	//   - spanName: child-0
	//     attributes:
	//       i: 0
	//     events:
	//     - name: DoSTH
	//   - spanName: child-1
	//     attributes:
	//       i: 1
	//     events:
	//     - name: DoSTH
}

var errSample = errors.New("sample error")

func myInstrumentedFunc(ctx context.Context) (retErr error) {
	// If an error is returned, capture retErr such that the error is
	// automatically logged and traced.
	ctx, span, log := tracing.Tracer().Capture(&retErr).Trace(ctx, "myInstrumentedFunc")
	// Always remember to end the span when the function or operation is done!
	defer span.End()

	// Try logging with different verbosities
	log.Info("normal verbosity!")
	// Key-value pairs given to the logger will also be added to the span,
	// with the "log-attr-" prefix, i.e. there is "log-attr-hello": "from the other side"
	// in the span.
	log.V(1).Info("found a message", "hello", "from the other side")

	// Run the child function twice. Notice how, when the trace depth increases,
	// the log level also automatically increases (myInstrumentedFunc logged the
	// span start event with v=0, but child logs this with v=1).
	for i := 0; i < 2; i++ {
		child(ctx, i)
	}

	// Return an error for demonstration of the capture feature. No need to use
	// named returns here; retErr is caught in the defer span.End() anyways!
	return fmt.Errorf("unexpected: %w", errSample)
}

func child(ctx context.Context, i int) {
	// Start a child span. As there is an ongoing trace registered in the context,
	// a child span will be created automatically.
	_, span := tracing.Tracer().Start(ctx, fmt.Sprintf("child-%d", i))
	defer span.End()

	// Register an event and an attribute. Notice these also showing up in the log.
	span.AddEvent("DoSTH")
	span.SetAttributes(attribute.Int("i", i))
}
