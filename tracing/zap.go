package tracing

import (
	"io"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// HumanReadableLogger returns a human-readable logr.Logger implementation
// using zap.
func HumanReadableLogger(w io.Writer, lvl zapcore.LevelEnabler) logr.Logger {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = ""
	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	sink := zapcore.AddSync(w)
	errLevel := zap.NewAtomicLevelAt(zap.ErrorLevel)
	log := zap.New(zapcore.NewCore(encoder, sink, lvl)).
		WithOptions(
			zap.AddCallerSkip(1),
			zap.ErrorOutput(sink),
			zap.AddStacktrace(errLevel),
		)
	return zapr.NewLogger(log)
}
