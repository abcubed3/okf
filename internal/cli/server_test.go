package cli

import (
	"context"
	"os"
	"path/filepath"
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
	sampleDir, err := filepath.Abs("../../testdata/bundles/sample")
	if err != nil {
		t.Fatalf("failed to resolve testdata path: %v", err)
	}

	// proxyRemoteMCP blocks on context.Background().Done(). We can test that --remote parses cleanly
	// and invokes proxyRemoteMCP by testing in a goroutine with timeout or custom context cancel if needed.
	// For testing, run proxyRemoteMCP in goroutine and verify it starts without error.
	done := make(chan error, 1)
	go func() {
		done <- RunServer([]string{"--bundle", sampleDir, "--remote"})
	}()

	select {
	case err := <-done:
		t.Fatalf("unexpected early termination of remote proxy: %v", err)
	case <-time.After(100 * time.Millisecond):
		// Proxy is running in background as expected
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

func TestProxyRemoteMCP_Output(t *testing.T) {
	out := captureStderr(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		go func() {
			<-ctx.Done()
		}()

		// Run proxy in background
		go func() {
			_ = proxyRemoteMCP("sample-bundle")
		}()
		time.Sleep(100 * time.Millisecond)
	})

	if !strings.Contains(out, "Starting OKF MCP Remote Proxy for bundle: sample-bundle") {
		t.Errorf("expected remote proxy output, got: %s", out)
	}
}
