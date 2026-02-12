package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	f              *os.File
	consoleVerbose bool
}

func New(path string, consoleVerbose bool) (*Logger, error) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{f: f, consoleVerbose: consoleVerbose}, nil
}

func (l *Logger) Close() {
	if l.f != nil {
		_ = l.f.Close()
	}
}

func (l *Logger) write(level string, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s", time.Now().Format(time.RFC3339Nano), level, msg)

	if l.consoleVerbose {
		log.Println(line)
	}
	if l.f != nil {
		_, _ = l.f.WriteString(line + "\n")
	}
}

func (l *Logger) Info(format string, args ...any)  { l.write("INFO", format, args...) }
func (l *Logger) Warn(format string, args ...any)  { l.write("WARN", format, args...) }
func (l *Logger) Error(format string, args ...any) { l.write("ERROR", format, args...) }
func (l *Logger) Debug(format string, args ...any) {
	if !l.consoleVerbose {
		return
	}
	l.write("DEBUG", format, args...)
}
