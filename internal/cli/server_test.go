package cli

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunServer_InvalidFlag(t *testing.T) {
	err := RunServer([]string{"--invalid-flag-123"})
	if err == nil {
		t.Error("expected error for invalid flag, got nil")
	}
}

func TestRunServer_InvalidBundlePath(t *testing.T) {
	err := RunServer([]string{"--bundle", "/nonexistent/path/for/bundle/test"})
	if err == nil {
		t.Error("expected error for nonexistent bundle path, got nil")
	}
}

func TestRunServer_RemoteFlag(t *testing.T) {
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		_ = r.Close()
		_ = w.Close()
	}()

	done := make(chan error, 1)
	go func() {
		done <- RunServer([]string{"--bundle", "acme/core", "--remote"})
	}()

	select {
	case err := <-done:
		t.Fatalf("unexpected early termination of remote proxy: %v", err)
	case <-time.After(50 * time.Millisecond):
		// Close stdin to signal EOF and verify clean exit
		_ = w.Close()
	}

	if err := <-done; err != nil {
		t.Fatalf("expected clean exit on stdin EOF, got: %v", err)
	}
}

func captureStderr(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = old
	var buf strings.Builder
	b := make([]byte, 1024)
	for {
		n, err := r.Read(b)
		if n > 0 {
			buf.Write(b[:n])
		}
		if err != nil {
			break
		}
	}
	return buf.String()
}

func TestRunServer_RemoteValidation(t *testing.T) {
	if err := RunServer([]string{"--remote", "--transport", "http", "--bundle", "acme/core"}); err == nil {
		t.Errorf("expected --remote with --transport http to fail")
	}
	if err := RunServer([]string{"--remote", "--port", "9090", "--bundle", "acme/core"}); err == nil {
		t.Errorf("expected --remote with --port to fail")
	}
	if err := RunServer([]string{"--remote", "--bundle", "single-segment"}); err == nil {
		t.Errorf("expected --remote with 1-segment bundle ID to fail")
	}
}
