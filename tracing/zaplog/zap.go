// Package zaplog provides a builder-pattern constructor for creating a
// logr.Logger implementation using Zap with some commonly-good defaults.
package zaplog

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/luxas/deklarative/tracing/filetest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	// Encoder is a symbolic link to zapcore.Encoder.
	Encoder = zapcore.Encoder
	// EncoderConfig is a symbolic link to zapcore.EncoderConfig.
	EncoderConfig = zapcore.EncoderConfig
	// LevelEncoder is a symbolic link to zapcore.LevelEncoder.
	LevelEncoder = zapcore.LevelEncoder

	// EncoderConfigOption represents a function that applies an option to the EncoderConfig.
	EncoderConfigOption func(*EncoderConfig)
	// EncoderCreator represents an Encoder constructor given a populated EncoderConfig.
	EncoderCreator func(EncoderConfig) Encoder
)

// JSONEncoderCreator is a symbolic link to zapcore.NewJSONEncoder.
func JSONEncoderCreator() EncoderCreator { return zapcore.NewJSONEncoder }

// ConsoleEncoderCreator is a symbolic link to zapcore.NewConsoleEncoder.
func ConsoleEncoderCreator() EncoderCreator { return zapcore.NewConsoleEncoder }

// ProductionEncoderConfig is a symbolic link to zap.NewProductionEncoderConfig().
func ProductionEncoderConfig() EncoderConfig { return zap.NewProductionEncoderConfig() }

// DevelopmentEncoderConfig is a symbolic link to zap.NewDevelopmentEncoderConfig().
func DevelopmentEncoderConfig() EncoderConfig { return zap.NewDevelopmentEncoderConfig() }

// LowercaseLevelEncoder is the default LevelEncoder; it extends the zapcore.LowercaseLevelEncoder
// by adding a "(v={V})" to all levels where {V} is the logr level.
//
// TODO: Once we can upgrade to logr v1.x, and https://github.com/go-logr/zapr/pull/37
// has landed, we can make log levels more easily a field.
func LowercaseLevelEncoder() LevelEncoder {
	return func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		str := l.String()
		if l < zap.DebugLevel {
			str = "debug"
		}
		if l <= zap.InfoLevel {
			str += "(v=" + strconv.Itoa(int(l*-1)) + ")"
		}
		enc.AppendString(str)
	}
}

// CapitalLevelEncoder extends the zapcore.CapitalLevelEncoder
// by adding a "(v={V})" to all levels where {V} is the logr level.
func CapitalLevelEncoder() LevelEncoder {
	return func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		str := l.CapitalString()
		if l < zap.DebugLevel {
			str = "DEBUG"
		}
		if l <= zap.InfoLevel {
			str += "(v=" + strconv.Itoa(int(l*-1)) + ")"
		}
		enc.AppendString(str)
	}
}

// NewZap returns a new *Builder using the default configuration.
func NewZap() *Builder {
	return (&Builder{
		outW:           os.Stdout,
		encoderCfg:     ProductionEncoderConfig(),
		encoderCreator: JSONEncoderCreator(),
	}).WithLevelEncoder(LowercaseLevelEncoder())
}

// Builder is a builder-pattern struct for building a logr.Logger
// using go.uber.org/zap.
//
// The default configuration uses the production encoder configuration,
// writes JSON, includes the V log levels in the level name, and logs to os.Stdout.
type Builder struct {
	outW              io.Writer
	encoderCfg        EncoderConfig
	encoderCfgOptions []EncoderConfigOption
	encoderCreator    EncoderCreator
	level             zapcore.Level
	opts              []zap.Option
}

// LogTo specifies where to write logs. If you want to write to multiple
// destinations, use io.MultiWriter or preferably, zapcore.NewMultiWriteSyncer.
//
// A zapcore.WriteSyncer shall be passed in if possible, otherwise a no-op Sync
// method will be used internally. The resulting WriteSyncer is automatically
// locked using zapcore.Lock, so it can be used in a thread-safe manner.
//
// Defaults to os.Stdout.
//
// A call to this function overwrites any previous value.
func (b *Builder) LogTo(w io.Writer) *Builder {
	b.outW = w
	return b
}

// WithEncoderConfig lets the user fine-tune how to encode/format logs.
//
// Defaults to zap.NewProductionEncoderConfig().
//
// A call to this function overwrites any previous value.
func (b *Builder) WithEncoderConfig(cfg EncoderConfig) *Builder {
	b.encoderCfg = cfg
	return b
}

// WithEncoderConfigOption registers a function that mutates the registered
// EncoderConfig from WithEncoderConfig at Build() time. This is useful
// for "patching" an individual part of the EncoderConfig, instead of
// overwriting everything.
//
// A call to this function appends to the list of previous values.
func (b *Builder) WithEncoderConfigOption(opts ...EncoderConfigOption) *Builder {
	b.encoderCfgOptions = append(b.encoderCfgOptions, opts...)
	return b
}

// WithEncoderCreator uses a specific EncoderCreator to create the encoder.
//
// Defaults to JSONEncoderCreator().
//
// A call to this function overwrites any previous value.
func (b *Builder) WithEncoderCreator(encoderCreator EncoderCreator) *Builder {
	b.encoderCreator = encoderCreator
	return b
}

// LogUpto specifies the logr level that shall be used. All log messages from
// a logr.Logger with a log level _less than or equal to_ logrLevel will be output.
//
// To convert between zap and logr log levels, multiply by -1 like follows:
//
// 	Level	Zap		Logr
//		-N		N
//	Debug	-1		1
//	Info	0		0		(default)
//	Warn	1		N/A
// 	Error	2		N/A
//
// The default level of 0 means that logr.Info and logr.Error calls will
// be output, unless logr.Logger.V() is used to raise the level.
//
// According to logr.Logger, "it's illegal to pass a log
// level less than zero.", hence, negative logrLevel values are disallowed.
//
// A call to this function overwrites any previous value.
func (b *Builder) LogUpto(logrLevel int8) *Builder {
	if b.level >= 0 {
		b.level = zapcore.Level(-1 * logrLevel)
	}
	return b
}

// WithOptions appends options for configuring zap.
//
// Options by default applied in Build() are:
//
//	zap.AddStacktrace(zap.ErrorLevel)
//	zap.ErrorOutput(sink)
//
// It is possible to overwrite these default using this method.
//
// A call to this function appends to the list of previous values.
func (b *Builder) WithOptions(opts ...zap.Option) *Builder {
	b.opts = append(b.opts, opts...)
	return b
}

// Console is a shorthand for:
//
//	WithEncoder(ConsoleEncoderCreator()).
//	HumanFriendlyTime().
//	WithLevelEncoder(CapitalLevelEncoder())
//
// A call to this function overwrites any previous value.
func (b *Builder) Console() *Builder {
	return b.WithEncoderCreator(ConsoleEncoderCreator()).
		HumanFriendlyTime().
		WithLevelEncoder(CapitalLevelEncoder())
}

// Example is a shorthand for
//
//	HumanFriendlyTime().
//	NoTimestamps().
//	NoStacktraceOnError()
//
// A call to this function overwrites any previous value.
func (b *Builder) Example() *Builder {
	return b.HumanFriendlyTime().
		NoTimestamps().
		NoStacktraceOnError()
}

// Test is a shorthand for verifying log output in a test with the help of the
// filetest package. Given a filetest.Tester, this will make the logger log to
// a file under testdata/ with the name of the test + the ".log" suffix.
//
// FilterStacktraceOrigins is applied before verifying the output such that
// in console mode the stack trace is filtered.
func (b *Builder) Test(g *filetest.Tester) *Builder {
	return b.LogTo(g.Add(g.T.Name() + ".log").Filter(FilterStacktraceOrigins).Writer())
}

// NoStacktraceOnError makes the logger not output a stack trace when
// an error is logged. This is done by moving the stack trace level
// to only be output for the DPanicLevel or higher (zap) levels.
//
// A call to this function overwrites any previous value.
func (b *Builder) NoStacktraceOnError() *Builder {
	return b.WithOptions(zap.AddStacktrace(zap.DPanicLevel))
}

// WithLevelEncoder customizes how the log level is encoded.
//
// The default is LowercaseLevelEncoder.
//
// A call to this function overwrites any previous value.
func (b *Builder) WithLevelEncoder(levelEnc LevelEncoder) *Builder {
	return b.WithEncoderConfigOption(func(ec *EncoderConfig) {
		ec.EncodeLevel = levelEnc
	})
}

// NoTimestamps omits timestamps in the logs. It's useful for deterministic
// output in examples and tests.
//
// It corresponds to setting EncoderConfig.TimeKey = zapcore.OmitKey.
//
// By default timestamps are included in the log output.
//
// A call to this function overwrites any previous value.
func (b *Builder) NoTimestamps() *Builder {
	return b.WithEncoderConfigOption(func(ec *EncoderConfig) {
		ec.TimeKey = zapcore.OmitKey
	})
}

// HumanFriendlyTime serializes time.Time and time.Duration in a human-friendly
// manner.
//
// It serializes a time.Time to an ISO8601-formatted string with millisecond precision.
// It serializes a time.Duration using its built-in String method.
//
// It corresponds to setting EncoderConfig fields as follows:
// 	.EncodeTime = zapcore.ISO8601TimeEncoder
// 	.EncodeDuration = zapcore.StringDurationEncoder
//
// A call to this function overwrites any previous value.
func (b *Builder) HumanFriendlyTime() *Builder {
	return b.WithEncoderConfigOption(func(ec *EncoderConfig) {
		ec.EncodeTime = zapcore.ISO8601TimeEncoder
		ec.EncodeDuration = zapcore.StringDurationEncoder
	})
}

// Build builds the logger with the configured options.
//
// By default the logger name is an empty string, and the log level is 0.
func (b *Builder) Build() logr.Logger {
	// Convert the io.Writer to a zapcore.WriteSyncer, if a zapcore.WriteSyncer wasn't already
	// provided, and lock the resulting zapcore.WriteSyncer to make it thread-safe. Locking is
	// needed, e.g. for *os.Files.
	sink := zapcore.Lock(zapcore.AddSync(b.outW))

	// Create the encoder
	encCfg := b.encoderCfg
	for _, mutFn := range b.encoderCfgOptions {
		mutFn(&encCfg)
	}
	encoder := b.encoderCreator(encCfg)

	// Pre-populate the options with opinionated defaults, such that internal errors are written to
	// the same sink as configured above, and that stack traces are output for all errors by default.
	// By prepending the defaults, the user can override them later.
	opts := []zap.Option{
		zap.AddStacktrace(zap.ErrorLevel),
		zap.ErrorOutput(sink),
	}
	opts = append(opts, b.opts...)

	// We know that the zapr Logger implements logr.CallDepthLogger, so this cast is safe.
	return zapr.NewLogger(
		zap.New(zapcore.NewCore(encoder, sink, b.level), opts...),
	)
}

// FilterStacktraceOrigins removes every line in content that
// starts with tab. It is meant to be used for filtering call
// stack output from for example a logger when testing (as the exact
// lines of caller origin might vary for instance across Go versions).
//
// TODO: Make this work with JSON output as well.
func FilterStacktraceOrigins(content []byte) []byte {
	s := bufio.NewScanner(bytes.NewReader(content))
	out := make([]byte, 0, len(content))
	for s.Scan() {
		line := s.Bytes()
		if bytes.HasPrefix(line, []byte("\t")) {
			continue
		}

		out = append(out, line...)
		out = append(out, '\n')
	}
	return out
}
