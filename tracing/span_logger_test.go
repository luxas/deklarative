package tracing

import (
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/luxas/deklarative/tracing/filetest"
	"github.com/luxas/deklarative/tracing/tracingfakes"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

// TODO: Make sure keysAndValues aren't modified when passed to Info/Error.

func Test_spanLogger_WithValues(t *testing.T) {
	log := (&spanLogger{Logger: logr.Discard()}).
		WithValues("foo", "bar")
	assert.Equal(t, log.(*spanLogger).keysAndValues, []interface{}{"foo", "bar"})

	newlog := log.WithValues("private", true)
	// newlog shouldn't modify the earlier assertion, verify it again
	assert.Equal(t, log.(*spanLogger).keysAndValues, []interface{}{"foo", "bar"})
	// newlog should now have more keys and values
	assert.Equal(t, newlog.(*spanLogger).keysAndValues, []interface{}{
		"foo", "bar", "private", true,
	})
}

//counterfeiter:generate go.opentelemetry.io/otel/trace.Span

var errSample = errors.New("sample error")

func Test_spanLogger_args(t *testing.T) {
	g := filetest.New(t, goldie.WithNameSuffix(""))
	defer g.Assert()

	zapLogger := ZapLogger().Console().Example().Test(g).Build()
	zapLogger = zapLogger.WithName("foo")
	s := &tracingfakes.FakeSpan{}

	log := &spanLogger{Logger: zapLogger, span: s}
	log.Info("good, no args")
	log.Info("good", "hello-1", 123)
	log.Info("odd number of arguments are ignored", "hello-2")
	log.V(1).Info("too verbose, ignored", "hello-3", 123)
	log.Info("non-string key invocations ignored", "hello-4", true, 123, false)

	log.WithName("log").WithValues("array", []string{"one", "two"}).Info("bar")

	log.Error(errSample, "good, no args")
	log.Error(errSample, "good", "hello-5", false, "sample-float", 1.2)
	log.Error(errSample, "odd number of arguments are ignored", "hello-6")
	log.V(1).Error(errSample, "too verbose, ignored", "hello-7", 123)
	log.Error(errSample, "non-string key invocations ignored", "hello-8", true, 123, false)

	assert.Equal(t, 3, s.SetAttributesCallCount())
	assert.Equal(t,
		[]attribute.KeyValue{attribute.Int64("log-attr-hello-1", 123)},
		s.SetAttributesArgsForCall(0))
	assert.Equal(t,
		[]attribute.KeyValue{
			attribute.Array("log-attr-array", []string{"one", "two"}),
		},
		s.SetAttributesArgsForCall(1))
	assert.Equal(t,
		[]attribute.KeyValue{
			attribute.Bool("log-attr-hello-5", false),
			attribute.Float64("log-attr-sample-float", 1.2),
		},
		s.SetAttributesArgsForCall(2))
}
