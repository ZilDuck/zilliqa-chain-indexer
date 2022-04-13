package log

import (
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

func NewLogger(debug bool, file string) {
	logger := zap.New(zapcore.NewTee(fileCore(file), consoleCore(debug)))
	defer logger.Sync()

	zap.ReplaceGlobals(logger)
}

func fileCore(file string) zapcore.Core {
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.MessageKey = "message"
	cfg.TimeKey = "time"

	logFile, _ := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	return zapcore.NewCore(zapcore.NewJSONEncoder(cfg), zapcore.AddSync(logFile), zap.ErrorLevel)
}

func consoleCore(debug bool) zapcore.Core {
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.MessageKey = "message"
	cfg.TimeKey = "time"
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder

	level := zap.InfoLevel
	if debug {
		level = zap.DebugLevel
	}

	return zapcore.NewCore(zapcore.NewConsoleEncoder(cfg), zapcore.AddSync(colorable.NewColorableStdout()), level)
}

type Logger interface {
	Printf(format string, v ...interface{})
}
