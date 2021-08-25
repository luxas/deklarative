/*
Package tracing includes high-level tools for instrumenting your application
(and library) code using OpenTelemetry and go-logr.

This is done by interconnecting logs and traces; such that critical operations
that need to be instrumented start a tracing span using the *TracerBuilder
builder. Upon starting a span, the user gives it the context which it is operating
in. If the context contains a parent span, the new "child" span and the parent
are connected together. To the span various types of metadata can be registered,
for example attributes, status information, and potential errors. Spans always need
to be ended; most commonly using a defer statement right after creation.

The context given to the *TracerBuilder might carry a TracerProvider to
use for exporting span data, e.g. to Jaeger for visualization, or a logr.Logger,
to which logs are sent. The context can also carry a LogLevelIncreaser, which
correlates log levels to trace depth.

The core idea of interconnecting logs and traces is that when some metadata is
registered with a span (for example, it starts, ends, or has attributes or errors
registered), information about this is also logged. And upon logging something in
a function that is executing within a span, it is also registered with the span.

This means you have dual ways of looking at your application's execution; the
"waterfall" visualization of spans in a trace in an OpenTelemetry-compliant UI like
Jaeger, or through pluggable logging using logr. Additionally, there is a way to
output semi-human-readable YAML data based on the trace information, which is useful
when you want to unit-test a function based on its output trace data using a "golden
file" in a testdata/ directory.

Let's talk about trace depth and log levels. Consider this example trace (tree of spans):

	|A (d=0)                               |
	 -----> |B (d=1)          | |D (d=1) |
	         ----> |C (d=2) |

Span A is at depth 0, as this is a "root span". Inside of span A, span B starts, at
depth 1 (span B has exactly 1 parent span). Span B spawns span C at depth 2. Span B
ends, but after this span D starts at depth 1, as a child of span A. After D is done
executing, span A also ends after a while.

Using the TraceEnabler interface, the user can decide what spans are "enabled"
and hence sent to the TracerProvider backend, for example, Jaeger. By default, spans
of any depth are sent to the backing TracerProvider, but this is often not desireable
in production. The TraceEnabler can decide whether a span should be enabled based on
all data in tracing.TracerConfig, which includes e.g. span name, trace depth and so on.

For example, MaxDepthEnabler(maxDepth) allows all traces with depth maxDepth or less,
but LoggerEnabler() allows traces as long as the given Logger is enabled. With that,
lets take a look at how trace depth correlates with log levels.

The LogLevelIncreaser interface, possibly attached to a context, correlates how much
the log level (verboseness) should increase as an effect of the trace depth increasing.
The NoLogLevelIncrease() implementation, for example, never increases the log level
although the trace depth gets arbitrarily deep. However, that is most often not desired,
so there is also a NthLogLevelIncrease(n) implementation that raises the log level
every n-th increase of trace depth. For example, given the earlier example, log level
(often shortened "v") is increased like follows for NthLogLevelIncrease(2):

	|A (d=0, v=0)                                      |
	 -----> |B (d=1, v=0)           | |D (d=1, v=0) |
	         ----> |C (d=2, v=1) |

As per how logr.Loggers work, log levels can never be decreased, i.e. become less
verbose, they can only be increased. The logr.Logger backend enables log levels
up to a given maximum, configured by the user, similar to how MaxDepthEnabler works.

Log output for the example above would looks something like:

	{"level":"info(v=0)","logger":"A","msg":"starting span"}
	{"level":"info(v=0)","logger":"B","msg":"starting span"}
	{"level":"debug(v=1)","logger":"C","msg":"starting span"}
	{"level":"debug(v=1)","logger":"C","msg":"ending span"}
	{"level":"info(v=0)","logger":"B","msg":"ending span"}
	{"level":"info(v=0)","logger":"D","msg":"starting span"}
	{"level":"info(v=0)","logger":"D","msg":"ending span"}
	{"level":"info(v=0)","logger":"A","msg":"ending span"}

This is of course a bit dull example, because only the start/end span events are
logged, but it shows the spirit. If span operations like
span.Set{Name,Attributes,Status} are executed within the instrumented function, e.g.
to record errors, important return values, arbitrary attributes, or a decision, this
information will be logged automatically, without a need to call log.Info() separately.

At the same time, all trace data is nicely visualized in Jaeger :). For convenience,
a builder-pattern constructor for the zap logger, compliant with the Logger interface
is provided through the ZapLogger() function and zaplog sub-directory.

In package traceyaml there are utilities for unit testing the traces. In package
filetest there are utilities for using "golden" testdata/ files for comparing actual
output of loggers, tracers, and general writers against expected output. Both the
TracerProviderBuilder and zaplog.Builder support deterministic output for unit tests
and examples.

The philosophy behind this package is that instrumentable code (functions, structs,
and so on), should use the TracerBuilder to start spans; and will from there get a
Span and Logger implementation to use. It is safe for libraries used by other
consumers to use the TracerBuilder as well, if the user didn't want or request
tracing nor logging, all calls to the Span and Logger will be discarded!

The application owner wanting to (maybe conditionally) enable tracing and logging,
creates "backend" implementations of TracerProvider and Logger, e.g. using the
TracerProviderBuilder and/or zaplog.Builder. These backends control where the
telemetry data is sent, and how much of it is enabled. These "backend" implementations
are either attached specifically to a context, or registered globally. Using this
setup, telemetry can be enabled even on the fly, using e.g. a HTTP endpoint for
debugging a production system.

Have fun using this library and happy tracing!
*/
package tracing
