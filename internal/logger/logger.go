package logger

import (
	"io"
	"log/slog"
	"os"
)

// creates a new structured logger (w/ specified debug level)
func New(debug bool) *slog.Logger {
	var handler slog.Handler

	if !debug {
		// create a handler that discards all log messages
		handler = slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelError, // Set to a high level to discard everything
		})
	} else {
		// create a text handler that outputs to stderr with debug level enabled
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	return slog.New(handler)
}
