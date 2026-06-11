package logger_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Vivekkumar-IN/gotdbot/logger"
)

func capture() (logger.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	l := logger.NewWithOptions(
		logger.WithOutput(&buf),
		logger.WithColor(false),
		logger.WithLevel(logger.LevelDebug),
	)
	return l, &buf
}

func TestLevelFiltering(t *testing.T) {
	l, buf := capture()
	l.SetLevel(logger.LevelWarn)

	l.Debug("hidden")
	l.Info("hidden")
	l.Warn("visible")
	l.Error("visible")

	out := buf.String()
	if strings.Contains(out, "hidden") {
		t.Fatalf("filtered level leaked:\n%s", out)
	}
	if !strings.Contains(out, "visible") {
		t.Fatalf("expected warn/error lines missing:\n%s", out)
	}
}

func TestLevelStrings(t *testing.T) {
	cases := []struct {
		level logger.Level
		want  string
	}{
		{logger.LevelDebug, "DBG"},
		{logger.LevelInfo, "INF"},
		{logger.LevelWarn, "WRN"},
		{logger.LevelError, "ERR"},
		{logger.LevelFatal, "FTL"},
		{logger.LevelPanic, "PNC"},
	}
	for _, tc := range cases {
		if got := tc.level.String(); got != tc.want {
			t.Errorf("Level(%d).String() = %q, want %q", tc.level, got, tc.want)
		}
	}
}

func TestInfo(t *testing.T) {
	l, buf := capture()
	l.Info("hello world")
	out := buf.String()
	if !strings.Contains(out, "INF") || !strings.Contains(out, "hello world") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestInfof(t *testing.T) {
	l, buf := capture()
	l.Infof("user=%s id=%d", "vivek", 42)
	if !strings.Contains(buf.String(), "user=vivek id=42") {
		t.Errorf("formatted message missing: %s", buf.String())
	}
}

func TestAllLevelsEmit(t *testing.T) {
	cases := []struct {
		fn   func(logger.Logger)
		want string
	}{
		{func(l logger.Logger) { l.Debug("x") }, "DBG"},
		{func(l logger.Logger) { l.Info("x") }, "INF"},
		{func(l logger.Logger) { l.Warn("x") }, "WRN"},
		{func(l logger.Logger) { l.Error("x") }, "ERR"},
	}
	for _, tc := range cases {
		l, buf := capture()
		tc.fn(l)
		if !strings.Contains(buf.String(), tc.want) {
			t.Errorf("expected %q in output: %s", tc.want, buf.String())
		}
	}
}

func TestFormatVariants(t *testing.T) {
	l, buf := capture()
	l.Debugf("val=%d", 7)
	l.Warnf("oops %s", "!")
	out := buf.String()
	if !strings.Contains(out, "val=7") {
		t.Errorf("Debugf missing: %s", out)
	}
	if !strings.Contains(out, "oops !") {
		t.Errorf("Warnf missing: %s", out)
	}
}

func TestDebugNotShownAtInfoLevel(t *testing.T) {
	l, buf := capture()
	l.SetLevel(logger.LevelInfo)
	l.Debug("invisible")
	if buf.Len() != 0 {
		t.Errorf("debug should be silent at INFO: %s", buf.String())
	}
}

func TestGetSetLevel(t *testing.T) {
	l, _ := capture()
	l.SetLevel(logger.LevelError)
	if got := l.GetLevel(); got != logger.LevelError {
		t.Errorf("GetLevel() = %v, want LevelError", got)
	}
}

func TestSetOutput(t *testing.T) {
	var buf bytes.Buffer
	l := logger.New()
	l.SetOutput(&buf)
	l.Info("redirected")
	if !strings.Contains(buf.String(), "redirected") {
		t.Errorf("SetOutput did not redirect: %s", buf.String())
	}
}

func TestPanic(t *testing.T) {
	l, _ := capture()
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	l.Panic("boom")
}

func TestPanicf(t *testing.T) {
	l, _ := capture()
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	l.Panicf("boom %s", "now")
}

func TestTimestampAndCaller(t *testing.T) {
	l, buf := capture()
	l.Info("check")
	out := buf.String()
	if len(out) < 12 || out[2] != ':' || out[5] != ':' || out[8] != '.' {
		t.Errorf("bad timestamp in: %s", out)
	}
	if !strings.Contains(out, ".go:") {
		t.Errorf("caller info missing: %s", out)
	}
}

func TestGlobalFunctions(t *testing.T) {
	var buf bytes.Buffer
	logger.SetLogger(logger.NewWithOptions(
		logger.WithOutput(&buf),
		logger.WithColor(false),
		logger.WithLevel(logger.LevelDebug),
	))

	logger.Info("global info")
	logger.Debugf("global %s", "debug")

	out := buf.String()
	if !strings.Contains(out, "global info") {
		t.Errorf("global Info missing: %s", out)
	}
	if !strings.Contains(out, "global debug") {
		t.Errorf("global Debugf missing: %s", out)
	}
}
