package telemetry

import (
	"bytes"
	"encoding/json"
	"net/http"
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

// SendEvent asynchronously sends an anonymous telemetry event to the OKF Hub proxy.
func SendEvent(eventName, bundleName string) {
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
		_, _ = client.Do(req)
	}()
}
