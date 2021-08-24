package tracing_test

import (
	"context"
	"fmt"
	"os"

	"github.com/luxas/deklarative/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func requireNonNil(err error) {
	if err != nil {
		panic(err)
	}
}

func Example_globalTracing() {
	// Create a new TracerProvider that logs trace data to os.Stdout, and
	// is registered globally. It won't trace deeper than 2 child layers.
	err := tracing.Provider().
		TestYAMLTo(os.Stdout).
		TraceUpto(2).
		InstallGlobally()
	requireNonNil(err)

	// Start from a root, background context, that will be given to instrumented
	// functions from this example.
	ctx := context.Background()

	// Remember to shut down the global tracing provider in the end.
	defer requireNonNil(tracing.GetGlobalTracerProvider().Shutdown(ctx))

	// Create some operator struct with some important data.
	f := Foo{importantData: "very important!"}
	// Execute an instrumented operation on this struct.
	_, _ = f.Operation(ctx)

	// Output:
	// # doOperation-4
	// - spanName: doOperation-4
	//   attributes:
	//     i: 4
	//   errors:
	//   - error: oh no
	//   startConfig:
	//     newRoot: true
	//   children:
	//   - spanName: doOperation-5
	//     attributes:
	//       i: 5
	//     errors:
	//     - error: oh no
	//
	// # *tracing_test.Foo.Operation
	// - spanName: '*tracing_test.Foo.Operation'
	//   attributes:
	//     result: my error value
	//   errors:
	//   - error: 'operation got unexpected error: oh no'
	//   startConfig:
	//     attributes:
	//       important-data: very important!
	//   children:
	//   - spanName: doOperation-1
	//     attributes:
	//       i: 1
	//     errors:
	//     - error: oh no
	//     children:
	//     - spanName: doOperation-2
	//       attributes:
	//         i: 2
	//       errors:
	//       - error: oh no
}

type Foo struct {
	importantData string
}

func (f *Foo) Operation(ctx context.Context) (retStr string, retErr error) {
	// Start tracing; calculate the tracer name automatically by providing a
	// reference to the "actor" *Foo. This will fmt.Sprintf("%T") leading to
	// the prefix being "*tracing_test.Foo".
	//
	// Also register important data stored in the struct in an attribute.
	//
	// At the end of the function, when the span ends, automatically register
	// the return error with the trace, if non-nil.
	ctx, span := tracing.Tracer().
		WithActor(f).
		WithAttributes(attribute.String("important-data", f.importantData)).
		Capture(&retErr).
		Start(ctx, "Operation")
	// Always remember to end the span
	defer span.End()
	// Register the return value, the string, as an attribute in the span as well.
	defer func() { span.SetAttributes(attribute.String("result", retStr)) }()

	// Start a "child" trace using function doOperation, and conditionally return
	// the string.
	if err := doOperation(ctx, 1); err != nil {
		return "my error value", fmt.Errorf("operation got unexpected error: %w", err)
	}
	return "my normal value", nil
}

func doOperation(ctx context.Context, i int64) (retErr error) {
	// Just to show off that you don't have to inherit the parent span; it's
	// possible to also create a new "root" span at any point.
	var startOpts []trace.SpanStartOption
	if i == 4 {
		startOpts = append(startOpts, trace.WithNewRoot())
	}

	// Start the new span, and automatically register the error, if any.
	ctx, span := tracing.Tracer().
		Capture(&retErr).
		Start(ctx, fmt.Sprintf("doOperation-%d", i), startOpts...)
	defer span.End()

	span.SetAttributes(attribute.Int64("i", i))

	// Just to show off trace depth here, recursively call itself and
	// increase i until it is 5, and then return an error. This means that
	// the error returned at i==5 will be returned for all i < 5, too.
	//
	// This, in combination with the trace depth configured above with
	// TraceUpto(2), means that doOperation-3, although executed, won't
	// be shown in the output, because that is at depth 3.
	//
	// However, as doOperation-4 is a root span, it is at depth 0 and hence
	// comfortably within the allowed range.
	if i == 5 {
		return fmt.Errorf("oh no") //nolint:goerr113
	}
	return doOperation(ctx, i+1)
}
