package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorGray   = "\033[90m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorBold   = "\033[1m"
)

var levelColors = [...]string{
	colorCyan,   // Debug
	colorGreen,  // Info
	colorYellow, // Warn
	colorRed,    // Error
	colorRed,    // Fatal
	colorRed,    // Panic
}

type defaultLogger struct {
	mu    sync.Mutex
	out   io.Writer
	level Level
	color bool
}

var _ Logger = (*defaultLogger)(nil)

// New returns a logger writing to stderr at INFO level with color auto-detected.
func New() Logger {
	return &defaultLogger{
		out:   os.Stderr,
		level: LevelInfo,
		color: isTerminal(os.Stderr),
	}
}

// NewWithOptions returns a logger with the given options applied.
func NewWithOptions(opts ...Option) Logger {
	l := &defaultLogger{
		out:   os.Stderr,
		level: LevelInfo,
		color: isTerminal(os.Stderr),
	}
	for _, o := range opts {
		o(l)
	}
	return l
}

// Option configures a defaultLogger.
type Option func(*defaultLogger)

// WithLevel sets the minimum log level.
func WithLevel(level Level) Option {
	return func(l *defaultLogger) { l.level = level }
}

// WithOutput sets the writer; color is auto-detected from the new writer.
func WithOutput(w io.Writer) Option {
	return func(l *defaultLogger) {
		l.out = w
		l.color = isTerminal(w)
	}
}

// WithColor forces color on or off regardless of terminal detection.
func WithColor(enabled bool) Option {
	return func(l *defaultLogger) { l.color = enabled }
}

func (l *defaultLogger) SetLevel(level Level) {
	l.mu.Lock()
	l.level = level
	l.mu.Unlock()
}

func (l *defaultLogger) GetLevel() Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

func (l *defaultLogger) SetOutput(w io.Writer) {
	l.mu.Lock()
	l.out = w
	l.color = isTerminal(w)
	l.mu.Unlock()
}

func (l *defaultLogger) emit(level Level, msg string) {
	if level < l.level {
		return
	}

	_, file, line, ok := runtime.Caller(3)
	caller := "???"
	if ok {
		caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	var b strings.Builder
	b.Grow(128)

	if l.color {
		b.WriteString(colorGray)
	}
	b.WriteString(time.Now().Format("15:04:05.000"))
	if l.color {
		b.WriteString(colorReset)
	}
	b.WriteByte(' ')

	if l.color {
		b.WriteString(levelColors[level])
		b.WriteString(colorBold)
	}
	b.WriteString(level.String())
	if l.color {
		b.WriteString(colorReset)
	}
	b.WriteByte(' ')

	if l.color {
		b.WriteString(colorGray)
	}
	b.WriteString(caller)
	if l.color {
		b.WriteString(colorReset)
	}
	b.WriteByte(' ')

	b.WriteString(msg)
	b.WriteByte('\n')

	l.mu.Lock()
	_, _ = io.WriteString(l.out, b.String())
	l.mu.Unlock()

	if level == LevelFatal {
		os.Exit(1)
	}
	if level == LevelPanic {
		panic(msg)
	}
}

func (l *defaultLogger) log(level Level, v []any)                    { l.emit(level, fmt.Sprint(v...)) }
func (l *defaultLogger) logf(level Level, format string, v []any)    { l.emit(level, fmt.Sprintf(format, v...)) }

func (l *defaultLogger) Debug(v ...any)  { l.log(LevelDebug, v) }
func (l *defaultLogger) Info(v ...any)   { l.log(LevelInfo, v) }
func (l *defaultLogger) Warn(v ...any)   { l.log(LevelWarn, v) }
func (l *defaultLogger) Error(v ...any)  { l.log(LevelError, v) }
func (l *defaultLogger) Fatal(v ...any)  { l.log(LevelFatal, v) }
func (l *defaultLogger) Panic(v ...any)  { l.log(LevelPanic, v) }

func (l *defaultLogger) Debugf(format string, v ...any) { l.logf(LevelDebug, format, v) }
func (l *defaultLogger) Infof(format string, v ...any)  { l.logf(LevelInfo, format, v) }
func (l *defaultLogger) Warnf(format string, v ...any)  { l.logf(LevelWarn, format, v) }
func (l *defaultLogger) Errorf(format string, v ...any) { l.logf(LevelError, format, v) }
func (l *defaultLogger) Fatalf(format string, v ...any) { l.logf(LevelFatal, format, v) }
func (l *defaultLogger) Panicf(format string, v ...any) { l.logf(LevelPanic, format, v) }
