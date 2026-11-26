// Package telemetry dispatches opt-in anonymous OKF CLI usage events.
package telemetry

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// TelemetryPayload represents the structure expected by the backend proxy
type TelemetryPayload struct {
	ClientID string  `json:"client_id"`
	Events   []Event `json:"events"`
}

// Event represents a single telemetry event
type Event struct {
	Name   string            `json:"name"`
	Params map[string]string `json:"params"`
}

// IsDisabled reports whether telemetry is disabled via OKF_TELEMETRY_DISABLED or DO_NOT_TRACK.
func IsDisabled() bool {
	if v := strings.TrimSpace(os.Getenv("OKF_TELEMETRY_DISABLED")); v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes") {
		return true
	}
	if v := strings.TrimSpace(os.Getenv("DO_NOT_TRACK")); v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes") {
		return true
	}
	return false
}

// SendEvent asynchronously sends an anonymous telemetry event to the OKF Hub proxy.
func SendEvent(eventName, bundleName string) {
	if IsDisabled() {
		return
	}
	go func() {
		payload := TelemetryPayload{
			ClientID: "anonymous_cli_user",
			Events: []Event{
				{
					Name: eventName,
					Params: map[string]string{
						"bundle": bundleName,
					},
				},
			},
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return
		}

		req, err := http.NewRequest("POST", "https://okfgo.dev/api/telemetry", bytes.NewBuffer(data))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
		}
	}()
}
