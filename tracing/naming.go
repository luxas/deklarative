package tracing

import (
	"fmt"
	"io"
	"os"
)

// TracerNamed is an interface that allows types to customize their
// name shown in traces and logs.
type TracerNamed interface {
	TracerName() string
}

func tracerName(obj interface{}) string {
	switch t := obj.(type) {
	case string:
		return t
	case TracerNamed:
		return t.TracerName()
	case nil:
		return ""
	}

	switch obj {
	case os.Stdin:
		return "os.Stdin"
	case os.Stdout:
		return "os.Stdout"
	case os.Stderr:
		return "os.Stderr"
	case io.Discard:
		return "io.Discard"
	default:
		return fmt.Sprintf("%T", obj)
	}
}

// fmtSpanName appends the name of the given function (spanName) to the tracer
// name, if set.
func fmtSpanName(tracerName, spanName string) string {
	if len(tracerName) != 0 && len(spanName) != 0 {
		return tracerName + "." + spanName
	}
	// As either (or both) o.Name and spanName are empty strings, we can add them together
	name := tracerName + spanName
	if len(name) != 0 {
		return name
	}
	return "<unnamed_span>"
}
