package tracing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	golog "log"

	"go.opentelemetry.io/otel/attribute"
)

func Example_loggingAndYAMLTrace() {
	// Make a TracerProvider writing YAML about what's happening to
	// yamlTrace.
	var yamlTrace bytes.Buffer
	tp, err := Provider().TestYAMLTo(&yamlTrace).Build()
	if err != nil {
		golog.Fatal(err)
	}

	// Make an example logger logging to os.Stdout directly.
	fmt.Println("Log representation:")
	log := ZapLogger().Example().LogUpto(1).Build()

	// Define an "inner" function called by outer. Due to this being
	// an example, this function needs to be inline, and so does outer.
	// In the real world, they would be normal functions.
	inner := func(ctx context.Context, i int) {
		ctx, span := Tracer().Start(ctx, fmt.Sprintf("doSth-%d", i)) //nolint
		defer span.End()

		span.AddEvent("DoSTH")
		span.SetAttributes(attribute.Int("i", i))
	}
	sampleErr := errors.New("sample error") //nolint:goerr113
	outer := func(ctx context.Context) (retErr error) {
		ctx, span, log := Tracer().Capture(&retErr).Trace(ctx, "op1")
		defer span.End()

		log.V(1).Info("found a message", "hello", "from the other side")

		for i := 0; i < 2; i++ {
			inner(ctx, i)
		}

		return fmt.Errorf("unexpected: %w", sampleErr)
	}

	// Specifically use the given Logger and TracerProvider when performing this
	// trace operation by crafting a dedicated context.
	ctx := Context().WithLogger(log).WithTracerProvider(tp).Build()
	err = outer(ctx)
	log.Info("error is sampleErr", "is-sampleErr", errors.Is(err, sampleErr))

	// Shutdown the TracerProvider, and output the YAML it yielded to os.Stdout.
	if err := tp.Shutdown(ctx); err != nil {
		golog.Fatal(err)
	}
	fmt.Printf("\nYAML trace representation:\n%s", yamlTrace.String())

	// Output:
	// Log representation:
	// {"level":"info(v=0)","logger":"op1","msg":"starting span"}
	// {"level":"debug(v=1)","logger":"op1","msg":"found a message","hello":"from the other side"}
	// {"level":"debug(v=1)","logger":"doSth-0","msg":"starting span"}
	// {"level":"debug(v=1)","logger":"doSth-0","msg":"span event","span-event":"DoSTH"}
	// {"level":"debug(v=1)","logger":"doSth-0","msg":"span attribute change","span-attr-i":0}
	// {"level":"debug(v=1)","logger":"doSth-0","msg":"ending span"}
	// {"level":"debug(v=1)","logger":"doSth-1","msg":"starting span"}
	// {"level":"debug(v=1)","logger":"doSth-1","msg":"span event","span-event":"DoSTH"}
	// {"level":"debug(v=1)","logger":"doSth-1","msg":"span attribute change","span-attr-i":1}
	// {"level":"debug(v=1)","logger":"doSth-1","msg":"ending span"}
	// {"level":"error","logger":"op1","msg":"span error","error":"unexpected: sample error"}
	// {"level":"info(v=0)","logger":"op1","msg":"ending span"}
	// {"level":"info(v=0)","msg":"error is sampleErr","is-sampleErr":true}
	//
	// YAML trace representation:
	// # op1
	// - attributes:
	//   - key: log-attr-hello
	//     type: STRING
	//     value: from the other side
	//   children:
	//   - attributes:
	//     - key: i
	//       type: INT64
	//       value: 0
	//     events:
	//     - name: DoSTH
	//     names:
	//     - doSth-0
	//   - attributes:
	//     - key: i
	//       type: INT64
	//       value: 1
	//     events:
	//     - name: DoSTH
	//     names:
	//     - doSth-1
	//   errors:
	//   - error: 'unexpected: sample error'
	//   names:
	//   - op1
}
