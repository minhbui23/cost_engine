// internal/utils/utils.go

package utils

import (
	"log/slog"
	"os"
)

// *** THÊM: Hàm thiết lập logger ***
func SetupLogger(debug bool) *slog.Logger {
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	return logger
}
