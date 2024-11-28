package log

import (
	"fmt"
	"runtime"

	"go.uber.org/zap"
)

var logger = zap.NewNop().Sugar()

func Logger() *zap.SugaredLogger {
	return logger
}

func IsOutputsToConsole() bool {
	// for now, we only use `zap.NewDevelopment`, which uses stderr as output
	return true
}

func IsColorOutputSupported() bool {
	if !IsOutputsToConsole() {
		return false
	}

	if runtime.GOOS == "windows" {
		return false
	}

	return true
}

func InitLogger() error {
	noSugarLogger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("logger build error: %w", err)
	}

	logger = noSugarLogger.Sugar()

	return nil
}
