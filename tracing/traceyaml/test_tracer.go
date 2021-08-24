// Package traceyaml provides a means to unit test a trace flow, using a YAML file
// structure that is representative and as close to human-readable as it gets.
//
// This package is tested by unit tests in the above tracing package.
package traceyaml

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

// New returns a composite TracerProvider that captures all data written into
// spans created. The recursively captured span/trace data is gathered into a
// SpanInfo struct, marshalled into YAML, and written to w. Writer w can optionally
// implement the zapcore.WriteSyncer interface; if so it'll be used.
// As soon as a span ends; its list item of YAML will be output to w, as:
//
//	# Trace1
//	- {Trace1 data}
//
// 	# Trace2
//	- {Trace2 data}
func New(tp trace.TracerProvider, w io.Writer) trace.TracerProvider {
	return &testTracerProvider{tp, zapcore.Lock(zapcore.AddSync(w))}
}

type testTracerProvider struct {
	// embedding is important; this automatically exposes all inherited functionality from the
	// underlying resource.
	trace.TracerProvider
	// ws is a race-free writer
	ws zapcore.WriteSyncer
}

func (tp *testTracerProvider) Tracer(instrumentationName string, opts ...trace.TracerOption) trace.Tracer {
	tracer := tp.TracerProvider.Tracer(instrumentationName, opts...)
	return &testTracer{tracer, tp}
}

type testTracer struct {
	// embedding is important; this automatically exposes all inherited functionality from the
	// underlying resource.
	trace.Tracer

	provider *testTracerProvider
}

func (t *testTracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	ctx, span := t.Tracer.Start(ctx, spanName, opts...)
	newSpan := &testSpan{span, t.provider, nil}

	cfg := trace.NewSpanStartConfig(opts...)

	if parentData := getSpanInfo(ctx); parentData != nil && !cfg.NewRoot() {
		newSpan.data = parentData.newChild(spanName, opts...)
	} else {
		newSpan.data = newSpanInfo(spanName, opts...)
	}
	ctx = withSpanInfo(ctx, newSpan.data)

	return trace.ContextWithSpan(ctx, newSpan), newSpan
}

type testSpan struct {
	// embedding is important; this automatically exposes all inherited functionality from the
	// underlying resource.
	trace.Span

	provider *testTracerProvider
	data     *SpanInfo
}

func (s *testSpan) End(options ...trace.SpanEndOption) {
	s.data.mu.Lock()
	defer s.data.mu.Unlock()

	s.data.EndConfig = spanConfigFromEnd(options...)

	if !s.data.isChild {
		listItem := []*SpanInfo{s.data}
		// Deliberately use yaml.v2 here as it marshals lists on the same
		// indentation level as the list key.
		// TODO: When "our own" YAML library is ready, use that.
		out, err := yaml.Marshal(listItem)
		if err == nil {
			header := fmt.Sprintf("# %s", s.data.SpanName)
			out = bytes.Join([][]byte{[]byte(header), out, nil}, []byte{'\n'})
			err = multierr.Combine(err, writeNoLength(s.provider.ws, out))
		}
		if err != nil {
			s.Span.RecordError(err)
		}
	}

	s.Span.End(options...)
}

func writeNoLength(w io.Writer, p []byte) error {
	_, err := w.Write(p)
	return err
}

func (s *testSpan) AddEvent(name string, options ...trace.EventOption) {
	s.data.mu.Lock()
	defer s.data.mu.Unlock()

	s.data.Events = append(s.data.Events, Event{
		Name:        name,
		EventConfig: eventConfigFrom(options...),
	})

	s.Span.AddEvent(name, options...)
}

func (s *testSpan) RecordError(err error, options ...trace.EventOption) {
	s.data.mu.Lock()
	defer s.data.mu.Unlock()

	s.data.Errors = append(s.data.Errors, Error{
		Error:       fmt.Sprintf("%v", err),
		EventConfig: eventConfigFrom(options...),
	})

	s.Span.RecordError(err, options...)
}

func (s *testSpan) SetStatus(code codes.Code, description string) {
	s.data.mu.Lock()
	defer s.data.mu.Unlock()

	sc := Status{
		Code: code,
	}
	// Set description only if codes.Error
	if code == codes.Error {
		sc.Description = description
	}
	s.data.StatusChanges = append(s.data.StatusChanges, sc)

	s.Span.SetStatus(code, description)
}

func (s *testSpan) SetName(name string) {
	s.data.mu.Lock()
	defer s.data.mu.Unlock()

	s.data.NameChanges = append(s.data.NameChanges, name)
	s.Span.SetName(name)
}

func (s *testSpan) SetAttributes(kv ...attribute.KeyValue) {
	s.data.mu.Lock()
	defer s.data.mu.Unlock()

	attrsInto(kv, s.data.Attributes)
	s.Span.SetAttributes(kv...)
}

func (s *testSpan) TracerProvider() trace.TracerProvider { return s.provider }

type traceDataCtxKeyStruct struct{}

//nolint:gochecknoglobals
var traceDataCtxKey = traceDataCtxKeyStruct{}

func withSpanInfo(ctx context.Context, traceData *SpanInfo) context.Context {
	return context.WithValue(ctx, traceDataCtxKey, traceData)
}

func getSpanInfo(ctx context.Context) *SpanInfo {
	td, _ := ctx.Value(traceDataCtxKey).(*SpanInfo)
	return td
}
