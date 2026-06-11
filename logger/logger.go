package logger

import (
	"io"
	"os"
)

var std Logger = New()

// SetLogger replaces the global default logger.
func SetLogger(l Logger) { std = l }

// GetLogger returns the global default logger.
func GetLogger() Logger { return std }

// SetLevel sets the global logger's minimum level.
func SetLevel(level Level) { std.SetLevel(level) }

// SetOutput sets the global logger's output writer.
func SetOutput(w io.Writer) { std.SetOutput(w) }

func Debug(v ...any) { std.Debug(v...) }
func Info(v ...any)  { std.Info(v...) }
func Warn(v ...any)  { std.Warn(v...) }
func Error(v ...any) { std.Error(v...) }
func Fatal(v ...any) { std.Fatal(v...) }
func Panic(v ...any) { std.Panic(v...) }

func Debugf(format string, v ...any) { std.Debugf(format, v...) }
func Infof(format string, v ...any)  { std.Infof(format, v...) }
func Warnf(format string, v ...any)  { std.Warnf(format, v...) }
func Errorf(format string, v ...any) { std.Errorf(format, v...) }
func Fatalf(format string, v ...any) { std.Fatalf(format, v...) }
func Panicf(format string, v ...any) { std.Panicf(format, v...) }

// isTerminal returns true if w is a real terminal (supports ANSI color).
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
