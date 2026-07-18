package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout is a helper to capture stdout during tests
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		fmt.Printf("Error copying to buffer: %v\n", err)
	}
	return buf.String()
}

func TestExecute_NoArgs(t *testing.T) {
	out := captureStdout(func() {
		err := Execute([]string{}, "1.0", "abc", "2026")
		if err != nil {
			t.Errorf("Expected nil error for empty args, got %v", err)
		}
	})

	if !strings.Contains(out, "OKF CLI") {
		t.Errorf("Expected usage output for empty args, got %q", out)
	}
}

func TestExecute_Version(t *testing.T) {
	out := captureStdout(func() {
		err := Execute([]string{"version"}, "1.0.0", "abcdef", "today")
		if err != nil {
			t.Errorf("Expected nil error for version cmd, got %v", err)
		}
	})

	if !strings.Contains(out, "okf version 1.0.0") || !strings.Contains(out, "abcdef") {
		t.Errorf("Expected version output, got %q", out)
	}
}

func TestExecute_Help(t *testing.T) {
	out := captureStdout(func() {
		err := Execute([]string{"help"}, "1.0", "abc", "2026")
		if err != nil {
			t.Errorf("Expected nil error for help cmd, got %v", err)
		}
	})

	if !strings.Contains(out, "Available Commands:") {
		t.Errorf("Expected usage output for help cmd, got %q", out)
	}
}

func TestExecute_UnknownCommand(t *testing.T) {
	err := Execute([]string{"does-not-exist"}, "1.0", "abc", "2026")
	if err == nil {
		t.Errorf("Expected error for unknown command, got nil")
	}

	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Expected 'unknown command' error, got %v", err)
	}
}
