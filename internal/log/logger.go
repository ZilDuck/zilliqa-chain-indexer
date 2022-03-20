package log

import (
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(debug bool) {
	pe := zap.NewProductionEncoderConfig()
	pe.EncodeTime = zapcore.ISO8601TimeEncoder
	pe.MessageKey = "message"
	pe.TimeKey = "time"

	pe.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(pe)

	level := zap.InfoLevel
	if debug {
		level = zap.DebugLevel
	}

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(colorable.NewColorableStdout()), level),
	)

	logger := zap.New(core)
	defer logger.Sync()

	zap.ReplaceGlobals(logger)
}

type Logger interface {
	Printf(format string, v ...interface{})
}
