# telemetry

## Overview
Package `telemetry` dispatches fire-and-forget anonymous CLI usage events to the OKF telemetry endpoint (`https://okfgo.dev/api/telemetry`).

## Interface
- `[telemetry.go:TelemetryPayload]` — ``type TelemetryPayload struct { ClientID string `json:"client_id"`; Events []Event `json:"events"` }``
- `[telemetry.go:Event]` — ``type Event struct { Name string `json:"name"`; Params map[string]string `json:"params"` }``
- `[telemetry.go:SendEvent]` — `func SendEvent(eventName, bundleName string)`

## Invariants
- `[telemetry.go:SendEvent]` — Always sets `TelemetryPayload.ClientID` to `"anonymous_cli_user"` and wraps a single `Event` with `Name: eventName` and `Params: map[string]string{"bundle": bundleName}`.
- `[telemetry.go:SendEvent]` — Uses a 5-second HTTP client timeout (`http.Client{Timeout: 5 * time.Second}`) and silently ignores all JSON marshaling, request construction, and HTTP transport errors.

## Side Effects
- `[telemetry.go:SendEvent]` — Spawns an untracked background goroutine on each call.
- `[telemetry.go:SendEvent]` — Sends an outbound HTTP `POST` request with `Content-Type: application/json` to `"https://okfgo.dev/api/telemetry"` and closes `resp.Body` when `client.Do` returns a non-nil response.

## Verification
- Run: `go test ./...`
