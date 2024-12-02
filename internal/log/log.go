package log

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var logger = zap.NewNop().Sugar()

func Logger() *zap.SugaredLogger {
	return logger
}

func InitLogger() error {
	noSugarLogger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("logger build error: %w", err)
	}

	logger = noSugarLogger.Sugar()

	return nil
}

func InitTestLogger(t *testing.T) {
	require.NoError(t, InitLogger())
}
