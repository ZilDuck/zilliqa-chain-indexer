package log

import (
	"github.com/TheZeroSlave/zapsentry"
	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(debug bool, sentryDsn string) {
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

	if sentryDsn != "" {
		logger = modifyToSentryLogger(logger, sentryDsn)
	}

	zap.ReplaceGlobals(logger)
}

type Logger interface {
	Printf(format string, v ...interface{})
}

func modifyToSentryLogger(log *zap.Logger, DSN string) *zap.Logger {
	cfg := zapsentry.Configuration{
		Level:             zapcore.ErrorLevel, //when to send message to sentry
		EnableBreadcrumbs: true,               // enable sending breadcrumbs to Sentry
		BreadcrumbLevel:   zapcore.InfoLevel,  // at what level should we sent breadcrumbs to sentry
		Tags: map[string]string{
			"component": "system",
		},
	}
	core, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromDSN(DSN))

	// to use breadcrumbs feature - create new scope explicitly
	log = log.With(zapsentry.NewScope())

	//in case of err it will return noop core. so we can safely attach it
	if err != nil {
		log.Warn("failed to init zap", zap.Error(err))
	}
	return zapsentry.AttachCoreToLogger(core, log)
}
