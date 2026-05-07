package logger

import (
	"log"
	"strings"

	"go.uber.org/zap"
)

// Logger is the package-level zap logger. Call Init before using it.
var Logger *zap.Logger

// Init initialises the package-level Logger based on the provided log level.
// "debug" → development (colored, stack traces on warns).
// Anything else → production (JSON, sampling, no dev extras).
func Init(logLevel string) {
	var (
		l   *zap.Logger
		err error
	)

	if strings.ToLower(logLevel) == "debug" {
		l, err = zap.NewDevelopment()
	} else {
		l, err = zap.NewProduction()
	}

	if err != nil {
		log.Fatalf("logger: failed to initialise: %v", err)
	}

	Logger = l
}

// Sync flushes any buffered log entries. Call it as `defer logger.Sync()`.
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}
