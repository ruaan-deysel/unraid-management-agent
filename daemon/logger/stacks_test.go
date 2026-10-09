package logger

import (
	"strings"
	"testing"
)

func TestStackBuf(t *testing.T) {
	s := stackBuf()
	if s == "" {
		t.Fatal("stackBuf returned empty string")
	}
	if !strings.Contains(s, "goroutine") {
		t.Errorf("stackBuf output does not look like a stack trace: %q", s)
	}
}

func TestAllGoroutineStacks(t *testing.T) {
	s := AllGoroutineStacks()
	if s == "" {
		t.Fatal("AllGoroutineStacks returned empty string")
	}
	if !strings.Contains(s, "goroutine") {
		t.Errorf("AllGoroutineStacks output does not look like a stack dump: %q", s)
	}
	const maxOutput = 512 << 10
	// The output is capped; if capped it must carry the truncation marker.
	if len(s) > maxOutput && !strings.HasSuffix(s, "... (truncated)") {
		t.Errorf("output exceeds cap without truncation marker (len=%d)", len(s))
	}
}

func TestLogPanicWithStack(t *testing.T) {
	// Save and restore the global level so this test is isolated.
	original := GetLevel()
	t.Cleanup(func() { SetLevel(original) })

	SetLevel(LevelError)
	// Should not panic for any recovered value.
	LogPanicWithStack("test", "boom")
	LogPanicWithStack("test", nil)
	LogPanicWithStack("test", struct{ Code int }{Code: 500})
}
