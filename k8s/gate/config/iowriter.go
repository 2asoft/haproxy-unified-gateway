package config

import (
	"io"
	"log/slog"
)

// type Writer interface {
// 	Write(p []byte) (n int, err error)
// }

type LogConverter struct {
	log *slog.Logger
}

func (l *LogConverter) Write(p []byte) (n int, err error) {
	l.log.Debug(string(p))
	return len(p), nil
}

func NewIOWriter(logger *slog.Logger) io.Writer {
	return &LogConverter{log: logger}
}
