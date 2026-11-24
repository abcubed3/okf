package cli

import (
	"os"
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
