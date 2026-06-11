package logger

import (
	"fmt"
	"io"
)

// Level defines the priority of a log message.
type Level int8

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelPanic
)

var levelStrings = [...]string{
	"DBG",
	"INF",
	"WRN",
	"ERR",
	"FTL",
	"PNC",
}

func (l Level) String() string {
	if l >= LevelDebug && l <= LevelPanic {
		return levelStrings[l]
	}
	return fmt.Sprintf("?%d", int(l))
}

// Logger is the single logging interface: plain, formatted, and key-value styles
// plus configuration methods.
type Logger interface {
	Debug(v ...any)
	Info(v ...any)
	Warn(v ...any)
	Error(v ...any)
	Fatal(v ...any)
	Panic(v ...any)

	Debugf(format string, v ...any)
	Infof(format string, v ...any)
	Warnf(format string, v ...any)
	Errorf(format string, v ...any)
	Fatalf(format string, v ...any)
	Panicf(format string, v ...any)

	SetLevel(level Level)
	SetOutput(w io.Writer)
	GetLevel() Level
}
