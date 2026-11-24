package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/abcubed3/okf/pkg/internal/links"
)

const bridgeMaxMessageBytes = 10 * 1024 * 1024 // 10MB

// RemoteBridgeConfig configures the stdio-to-Streamable-HTTP MCP relay.
type RemoteBridgeConfig struct {
	BundleID    string
	Host        string
	APIKey      string
	Traceparent string
	HTTPClient  *http.Client
	In          io.Reader
	Out         io.Writer
	ErrOut      io.Writer
}

// RunRemoteBridge relays newline-delimited JSON-RPC messages from cfg.In to the remote
// OKF Hub Streamable HTTP MCP endpoint and writes responses back to cfg.Out.
func RunRemoteBridge(ctx context.Context, cfg RemoteBridgeConfig) error {
	bundleID, err := links.ParseBundleID(cfg.BundleID)
	if err != nil {
		return err
	}

	endpoint, err := validateRemoteHostURL(cfg.Host, bundleID)
	if err != nil {
		return err
	}

	if cfg.In == nil {
		cfg.In = os.Stdin
	}
	if cfg.Out == nil {
		cfg.Out = os.Stdout
	}
	if cfg.ErrOut == nil {
		cfg.ErrOut = os.Stderr
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 60 * time.Second}
	}
	if cfg.Traceparent == "" {
		cfg.Traceparent = os.Getenv("TRACEPARENT")
	}

	b := &remoteBridge{
		cfg:      cfg,
		endpoint: endpoint,
	}

	_, _ = fmt.Fprintf(cfg.ErrOut, "OKF remote MCP bridge connected: bundle=%s endpoint=%s\n", bundleID, endpoint)

	scanner := bufio.NewScanner(cfg.In)
	scanner.Buffer(make([]byte, 0, 64*1024), bridgeMaxMessageBytes)

	for scanner.Scan() {
		if ctx.Err() != nil {
			break
		}
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		payload := make([]byte, len(line))
		copy(payload, line)
		b.recordInitLifecycle(payload)

		if err := b.forwardMessage(ctx, payload); err != nil {
			_, _ = fmt.Fprintf(cfg.ErrOut, "remote MCP bridge error: %v\n", err)
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		b.teardownSession()
		return fmt.Errorf("stdin read error: %w", err)
	}

	b.teardownSession()
	return nil
}

func validateRemoteHostURL(rawHost, bundleID string) (string, error) {
	rawHost = strings.TrimSpace(rawHost)
	if rawHost == "" {
		return "", fmt.Errorf("remote host URL cannot be empty")
	}
	u, err := url.Parse(rawHost)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid remote host URL %q", rawHost)
	}
	hostName := u.Hostname()
	isLoopback := hostName == "localhost" || hostName == "127.0.0.1" || hostName == "::1"
	if !isLoopback && u.Scheme != "https" {
		return "", fmt.Errorf("remote host %q must use https:// (http:// is only allowed for loopback hosts)", rawHost)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme %q in host %q", u.Scheme, rawHost)
	}

	base := strings.TrimRight(u.String(), "/")
	if ip := net.ParseIP(hostName); ip != nil && !ip.IsLoopback() && u.Scheme != "https" {
		return "", fmt.Errorf("remote host %q must use https://", rawHost)
	}
	return fmt.Sprintf("%s/mcp/http?bundle=%s", base, url.QueryEscape(bundleID)), nil
}

type remoteBridge struct {
	cfg      RemoteBridgeConfig
	endpoint string

	mu              sync.Mutex
	sessionID       string
	initReq         []byte
	initializedNote []byte
}

func (b *remoteBridge) recordInitLifecycle(payload []byte) {
	var msg struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	switch msg.Method {
	case "initialize":
		b.initReq = append([]byte(nil), payload...)
	case "notifications/initialized":
		b.initializedNote = append([]byte(nil), payload...)
	}
}

func (b *remoteBridge) forwardMessage(ctx context.Context, payload []byte) error {
	status, err := b.doPost(ctx, payload, true)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		// Session expired or evicted on server: re-initialize if we have the cached initialize request.
		b.mu.Lock()
		initReq := append([]byte(nil), b.initReq...)
		initNote := append([]byte(nil), b.initializedNote...)
		b.sessionID = ""
		b.mu.Unlock()

		if len(initReq) == 0 {
			return fmt.Errorf("remote MCP session expired (HTTP 404) and no initialize payload was cached")
		}
		_, _ = fmt.Fprintf(b.cfg.ErrOut, "remote MCP session expired (HTTP 404); re-initializing...\n")
		if _, err := b.doPost(ctx, initReq, false); err != nil {
			return fmt.Errorf("session re-initialization failed: %w", err)
		}
		if len(initNote) > 0 {
			if _, err := b.doPost(ctx, initNote, false); err != nil {
				return fmt.Errorf("session notifications/initialized failed: %w", err)
			}
		}
		retryStatus, err := b.doPost(ctx, payload, true)
		if err != nil {
			return err
		}
		if retryStatus >= 400 {
			return fmt.Errorf("remote MCP server returned HTTP %d after re-initialization", retryStatus)
		}
	}
	return nil
}

func (b *remoteBridge) doPost(ctx context.Context, payload []byte, writeOutput bool) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if b.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.cfg.APIKey)
	}
	if b.cfg.Traceparent != "" {
		req.Header.Set("traceparent", b.cfg.Traceparent)
	}

	b.mu.Lock()
	sid := b.sessionID
	b.mu.Unlock()
	if sid != "" {
		req.Header.Set("Mcp-Session-Id", sid)
	}

	resp, err := b.cfg.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if newSID := strings.TrimSpace(resp.Header.Get("Mcp-Session-Id")); newSID != "" {
		b.mu.Lock()
		b.sessionID = newSID
		b.mu.Unlock()
	}

	if resp.StatusCode == http.StatusNotFound {
		_, _ = io.Copy(io.Discard, resp.Body)
		return http.StatusNotFound, nil
	}
	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return resp.StatusCode, fmt.Errorf("remote MCP HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if !writeOutput {
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode, nil
	}

	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(ct, "text/event-stream") {
		return resp.StatusCode, b.writeSSEResponses(resp.Body)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, bridgeMaxMessageBytes))
	if err != nil {
		return resp.StatusCode, err
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 {
		if _, err := fmt.Fprintf(b.cfg.Out, "%s\n", trimmed); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

func (b *remoteBridge) writeSSEResponses(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), bridgeMaxMessageBytes)
	var dataLines []string
	flush := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		joined := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = nil
		if joined == "" || joined == "[DONE]" {
			return nil
		}
		_, err := fmt.Fprintf(b.cfg.Out, "%s\n", joined)
		return err
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return flush()
}

func (b *remoteBridge) teardownSession() {
	b.mu.Lock()
	sid := b.sessionID
	b.mu.Unlock()
	if sid == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, b.endpoint, nil)
	if err != nil {
		return
	}
	req.Header.Set("Mcp-Session-Id", sid)
	if b.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.cfg.APIKey)
	}
	if b.cfg.Traceparent != "" {
		req.Header.Set("traceparent", b.cfg.Traceparent)
	}
	resp, err := b.cfg.HTTPClient.Do(req)
	if err == nil && resp != nil {
		_ = resp.Body.Close()
	}
}
