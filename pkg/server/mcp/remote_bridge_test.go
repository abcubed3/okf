package mcp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

type bridgeRoundTripFunc func(*http.Request) (*http.Response, error)

func (f bridgeRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestRemoteBridgeLifecycleAndReinit(t *testing.T) {
	var callCount int32
	var deleted atomic.Bool
	var sawTraceparent atomic.Bool

	client := &http.Client{
		Transport: bridgeRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("traceparent") == "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
				sawTraceparent.Store(true)
			}
			if r.Method == http.MethodDelete {
				if r.Header.Get("Mcp-Session-Id") == "sess-2" {
					deleted.Store(true)
				}
				return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))}, nil
			}
			n := atomic.AddInt32(&callCount, 1)
			bodyBytes, _ := io.ReadAll(r.Body)
			bodyStr := string(bodyBytes)

			// 1. First initialize -> sess-1
			if n == 1 && strings.Contains(bodyStr, `"initialize"`) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type":   []string{"application/json"},
						"Mcp-Session-Id": []string{"sess-1"},
					},
					Body: io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`)),
				}, nil
			}
			// 2. First tools/list with sess-1 -> 404 triggers re-init
			if n == 2 && r.Header.Get("Mcp-Session-Id") == "sess-1" {
				return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, nil
			}
			// 3. Re-initialize -> sess-2
			if n == 3 && strings.Contains(bodyStr, `"initialize"`) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type":   []string{"application/json"},
						"Mcp-Session-Id": []string{"sess-2"},
					},
					Body: io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":{"reinit":true}}`)),
				}, nil
			}
			// 4. Retried tools/list with sess-2 -> SSE response
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader("event: message\ndata: {\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"tools\":[]}}\n\n")),
			}, nil
		}),
	}

	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n" + `{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n")
	var out, errOut bytes.Buffer

	err := RunRemoteBridge(context.Background(), RemoteBridgeConfig{
		BundleID:    "hub://acme/core",
		Host:        "https://hub.example.com",
		APIKey:      "secret",
		Traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		HTTPClient:  client,
		In:          in,
		Out:         &out,
		ErrOut:      &errOut,
	})
	if err != nil {
		t.Fatalf("RunRemoteBridge failed: %v", err)
	}
	if !strings.Contains(out.String(), `"tools":[]`) {
		t.Errorf("expected retried SSE response in stdout, got %q", out.String())
	}
	if !deleted.Load() {
		t.Errorf("expected HTTP DELETE teardown on stdin EOF")
	}
	if !sawTraceparent.Load() {
		t.Errorf("expected W3C traceparent header propagation")
	}
}

func TestRemoteBridgeRejectsInsecureAndMalformed(t *testing.T) {
	if _, err := validateRemoteHostURL("http://hub.example.com", "acme/core"); err == nil {
		t.Errorf("expected http:// on non-loopback host to be rejected")
	}
	if _, err := validateRemoteHostURL("http://127.0.0.1:8080", "acme/core"); err != nil {
		t.Errorf("expected http:// on 127.0.0.1 to be allowed, got %v", err)
	}
}
