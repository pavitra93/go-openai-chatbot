package logger

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"
)

func SetupLogger() {
	logDir := ensureLogDir()
	logFile := filepath.Join(logDir, "app.log")

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}

	// JSON handler
	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger) // set as global default
}

func ensureLogDir() string {
	logDir := "logs"
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		log.Fatalf("failed to create log dir: %v", err)
	}
	return logDir
}
