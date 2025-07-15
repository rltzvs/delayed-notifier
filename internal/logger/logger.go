package logger

import (
	"log/slog"
	"os"
	"strings"
	"time"
)

func NewLogger(logLevel string) *slog.Logger {
	var slogLevel slog.Level
	switch logLevel {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slogLevel,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				a.Value = slog.StringValue(time.Now().Format("2006-01-02 15:04:05"))
			case slog.LevelKey:
				a.Value = slog.StringValue(a.Value.String())
			case slog.SourceKey:
				a.Value = slog.StringValue(shortenSource(a.Value.String()))
			}
			return a
		},
	})

	return slog.New(handler)
}

func shortenSource(source string) string {
	if len(source) > 0 {
		parts := strings.Split(source, "/")
		return parts[len(parts)-1]
	}
	return source
}
