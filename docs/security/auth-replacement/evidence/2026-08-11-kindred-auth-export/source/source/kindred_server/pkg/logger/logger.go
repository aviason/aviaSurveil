package logger

import (
	"encoding/json"
	"io"
	"log/slog"
)

type Logger struct {
	base *slog.Logger
}

func New(w io.Writer, env string) *Logger {
	return &Logger{base: slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{AddSource: env != "production"}))}
}

func (l *Logger) Info(message string, fields map[string]any) {
	l.base.Info(message, attrs(fields)...)
}

func (l *Logger) Error(message string, err error, fields map[string]any) {
	if fields == nil {
		fields = map[string]any{}
	}
	fields["error"] = err.Error()
	l.base.Error(message, attrs(fields)...)
}

func (l *Logger) With(fields map[string]any) *Logger {
	return &Logger{base: l.base.With(attrs(fields)...)}
}

func attrs(fields map[string]any) []any {
	values := make([]any, 0, len(fields)*2)
	for key, value := range fields {
		values = append(values, key, value)
	}
	return values
}

func JSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
