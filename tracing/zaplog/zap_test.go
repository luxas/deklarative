package zaplog

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/luxas/deklarative/tracing/filetest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func ExampleBuilder_json() {
	// Build an example logger called bar that logs levels <= 1.
	log := NewZap().Example().LogUpto(1).Build().WithName("bar")

	// Sample info usage
	log.Info("some message", "foo", true)
	log.WithValues("bar", 1).V(1).Info("hello")

	// Sample error usage
	err := errors.New("unexpected error") //nolint:goerr113
	log.Error(err, "I don't know what happened here", "duration", time.Minute)

	// Verify that v=2 is disabled (i.e. discarded), but v=1 is enabled
	log.V(1).Info("am I enabled?", "enabled", log.V(1).Enabled())
	log.V(2).Info("am I enabled?", "enabled", log.V(2).Enabled())

	// Output:
	// {"level":"info(v=0)","logger":"bar","msg":"some message","foo":true}
	// {"level":"debug(v=1)","logger":"bar","msg":"hello","bar":1}
	// {"level":"error","logger":"bar","msg":"I don't know what happened here","duration":"1m0s","error":"unexpected error"}
	// {"level":"debug(v=1)","logger":"bar","msg":"am I enabled?","enabled":true}
}

func ExampleBuilder_console() {
	// Build an example logger called bar that logs levels <= 1.
	log := NewZap().Example().Console().LogUpto(1).Build().WithName("bar")

	// Sample info usage
	log.Info("some message", "foo", true)
	log.WithValues("bar", 1).V(1).Info("hello")

	// Sample error usage
	err := errors.New("unexpected error") //nolint:goerr113
	log.Error(err, "I don't know what happened here", "duration", time.Minute)

	// Verify that v=2 is disabled (i.e. discarded), but v=1 is enabled
	log.V(1).Info("am I enabled?", "enabled", log.V(1).Enabled())
	log.V(2).Info("am I enabled?", "enabled", log.V(2).Enabled())

	// Output:
	// INFO(v=0)	bar	some message	{"foo": true}
	// DEBUG(v=1)	bar	hello	{"bar": 1}
	// ERROR	bar	I don't know what happened here	{"duration": "1m0s", "error": "unexpected error"}
	// DEBUG(v=1)	bar	am I enabled?	{"enabled": true}
}

func ExampleBuilder_custom() {
	// Build an example logger called bar that logs levels <= 1.
	var buf bytes.Buffer
	log := NewZap().
		Example().
		LogUpto(1).
		WithEncoderConfig(DevelopmentEncoderConfig()).
		LogTo(&buf).
		Build().
		WithName("bar")

	// Sample info usage
	log.Info("some message", "foo", true)
	log.WithValues("bar", 1).V(1).Info("hello")

	// Sample error usage
	err := errors.New("unexpected error") //nolint:goerr113
	log.Error(err, "I don't know what happened here", "duration", time.Minute)

	// Verify that v=2 is disabled (i.e. discarded), but v=1 is enabled
	log.V(1).Info("am I enabled?", "enabled", log.V(1).Enabled())
	log.V(2).Info("am I enabled?", "enabled", log.V(2).Enabled())

	fmt.Println(buf.String())
	// Output:
	// {"L":"info(v=0)","N":"bar","M":"some message","foo":true}
	// {"L":"debug(v=1)","N":"bar","M":"hello","bar":1}
	// {"L":"error","N":"bar","M":"I don't know what happened here","duration":"1m0s","error":"unexpected error"}
	// {"L":"debug(v=1)","N":"bar","M":"am I enabled?","enabled":true}
}

func ExampleBuilder_calldepth() {
	// Build an example logger called bar that logs levels <= 1.
	var buf bytes.Buffer
	log := NewZap().
		NoTimestamps().
		LogTo(&buf).
		Console().
		LogUpto(1).
		Build().
		WithName("bar")

	// Sample info usage
	log.Info("some message", "foo", true)

	// This is literally meant to cause a DPANIC; one must not give zap
	// fields to a logr.Logger. Provoke a call stack here in the output.
	log.WithValues("bar", 1).V(1).Info("hello", zap.Float32("foo", 23.2))

	// Sample error usage. See the call stack in action.
	err := errors.New("unexpected error") //nolint:goerr113
	log.Error(err, "I don't know what happened here", "duration", time.Minute)

	// Verify that v=2 is disabled (i.e. discarded), but v=1 is enabled
	log.V(1).Info("am I enabled?", "enabled", log.V(1).Enabled())
	log.V(2).Info("am I enabled?", "enabled", log.V(2).Enabled())

	// Filter the call stack before outputting
	fmt.Println(string(FilterStacktraceOrigins(buf.Bytes())))

	// Output:
	// INFO(v=0)	bar	some message	{"foo": true}
	// DPANIC	bar	strongly-typed Zap Field passed to logr	{"bar": 1, "zap field": {"Key":"foo","Type":10,"Integer":1102682522,"String":"","Interface":null}}
	// github.com/go-logr/zapr.(*zapLogger).Info
	// github.com/luxas/deklarative/tracing/zaplog.ExampleBuilder_calldepth
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
	// main.main
	// runtime.main
	// DEBUG(v=1)	bar	hello	{"bar": 1}
	// ERROR	bar	I don't know what happened here	{"duration": "1m0s", "error": "unexpected error"}
	// github.com/luxas/deklarative/tracing/zaplog.ExampleBuilder_calldepth
	// testing.runExample
	// testing.runExamples
	// testing.(*M).Run
	// main.main
	// runtime.main
	// DEBUG(v=1)	bar	am I enabled?	{"enabled": true}
}

func TestTestdata(t *testing.T) {
	g := filetest.New(t)
	defer g.Assert()

	// Build an example logger called bar that logs levels <= 1.
	log := NewZap().
		NoTimestamps().
		Test(g).
		Console().
		LogUpto(1).
		Build().
		WithName("bar")

	// Sample info usage
	log.Info("some message", "foo", true)

	// This is literally meant to cause a DPANIC; one must not give zap
	// fields to a logr.Logger. Provoke a call stack here in the output.
	log.WithValues("bar", 1).V(1).Info("hello", zap.Float32("foo", 23.2))

	// Sample error usage. See the call stack in action.
	err := errors.New("unexpected error") //nolint:goerr113
	log.Error(err, "I don't know what happened here", "duration", time.Minute)

	// Verify that v=2 is disabled (i.e. discarded), but v=1 is enabled
	log.V(1).Info("am I enabled?", "enabled", log.V(1).Enabled())
	log.V(2).Info("am I enabled?", "enabled", log.V(2).Enabled())
}

func TestLevelEncoders(t *testing.T) {
	tests := []struct {
		enc   LevelEncoder
		level zapcore.Level
		want  string
	}{
		// Capital case
		{CapitalLevelEncoder(), zapcore.FatalLevel, "FATAL"},
		{CapitalLevelEncoder(), zapcore.ErrorLevel, "ERROR"},
		{CapitalLevelEncoder(), zapcore.WarnLevel, "WARN"},
		{CapitalLevelEncoder(), zapcore.InfoLevel, "INFO(v=0)"},
		{CapitalLevelEncoder(), zapcore.DebugLevel, "DEBUG(v=1)"},
		{CapitalLevelEncoder(), -2, "DEBUG(v=2)"},
		{CapitalLevelEncoder(), -44, "DEBUG(v=44)"},
		// Lowercase
		{LowercaseLevelEncoder(), zapcore.FatalLevel, "fatal"},
		{LowercaseLevelEncoder(), zapcore.ErrorLevel, "error"},
		{LowercaseLevelEncoder(), zapcore.WarnLevel, "warn"},
		{LowercaseLevelEncoder(), zapcore.InfoLevel, "info(v=0)"},
		{LowercaseLevelEncoder(), zapcore.DebugLevel, "debug(v=1)"},
		{LowercaseLevelEncoder(), -2, "debug(v=2)"},
		{LowercaseLevelEncoder(), -44, "debug(v=44)"},
	}
	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			fpe := &fakePrimitiveEncoder{}
			tt.enc(tt.level, fpe)
			assert.Equal(t, tt.want, fpe.str)
		})
	}
}

type fakePrimitiveEncoder struct{ str string }

func (*fakePrimitiveEncoder) AppendBool(bool)             {}
func (*fakePrimitiveEncoder) AppendByteString([]byte)     {}
func (*fakePrimitiveEncoder) AppendComplex128(complex128) {}
func (*fakePrimitiveEncoder) AppendComplex64(complex64)   {}
func (*fakePrimitiveEncoder) AppendFloat64(float64)       {}
func (*fakePrimitiveEncoder) AppendFloat32(float32)       {}
func (*fakePrimitiveEncoder) AppendInt(int)               {}
func (*fakePrimitiveEncoder) AppendInt64(int64)           {}
func (*fakePrimitiveEncoder) AppendInt32(int32)           {}
func (*fakePrimitiveEncoder) AppendInt16(int16)           {}
func (*fakePrimitiveEncoder) AppendInt8(int8)             {}
func (e *fakePrimitiveEncoder) AppendString(in string)    { e.str += in }
func (*fakePrimitiveEncoder) AppendUint(uint)             {}
func (*fakePrimitiveEncoder) AppendUint64(uint64)         {}
func (*fakePrimitiveEncoder) AppendUint32(uint32)         {}
func (*fakePrimitiveEncoder) AppendUint16(uint16)         {}
func (*fakePrimitiveEncoder) AppendUint8(uint8)           {}
func (*fakePrimitiveEncoder) AppendUintptr(uintptr)       {}
