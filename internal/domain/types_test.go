package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeMetadataJSON(t *testing.T) {
	raw := json.RawMessage(`{
		"source": " mcp ",
		"session_id": " ",
		"batch_id": 42,
		"content_day": 1,
		"nested": {"ok": true}
	}`)
	normalized := NormalizeMetadataJSON(raw)

	var metadata map[string]any
	if err := json.Unmarshal(normalized, &metadata); err != nil {
		t.Fatalf("normalized metadata should be valid JSON: %v", err)
	}
	if got := metadata["source"]; got != "mcp" {
		t.Fatalf("expected trimmed source=mcp, got %v", got)
	}
	if _, exists := metadata["session_id"]; exists {
		t.Fatal("expected empty session_id to be removed")
	}
	if _, exists := metadata["batch_id"]; exists {
		t.Fatal("expected non-string standard batch_id to be removed")
	}
	if got := metadata["content_day"]; got != float64(1) {
		t.Fatalf("expected unknown numeric field to be preserved, got %v", got)
	}
	if _, exists := metadata["nested"]; !exists {
		t.Fatal("expected unknown nested field to be preserved")
	}
}

func TestNormalizeMetadataJSONFallsBackToEmptyObject(t *testing.T) {
	for _, raw := range []json.RawMessage{
		nil,
		json.RawMessage(``),
		json.RawMessage(`not-json`),
		json.RawMessage(`[]`),
	} {
		if got := string(NormalizeMetadataJSON(raw)); got != `{}` {
			t.Fatalf("expected empty object for %q, got %s", string(raw), got)
		}
	}
}

func TestTaskAttemptPublicJSONDoesNotExposeRawProviderPayload(t *testing.T) {
	latency := 1200
	response := TaskAttemptsResponse{
		TaskID: "task_1",
		Attempts: []TaskAttempt{{
			ID:            "attempt_1",
			TaskID:        "task_1",
			AttemptNo:     1,
			Status:        AttemptFailed,
			Provider:      "openai-compatible",
			LatencyMs:     &latency,
			ErrorStage:    "provider_first_byte",
			ResponseBytes: 42,
		}},
	}
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal attempt response: %v", err)
	}
	payload := string(raw)
	for _, forbidden := range []string{"raw_response", "raw_response_json", "Authorization", "api_key"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("public attempt JSON should not contain %q: %s", forbidden, payload)
		}
	}
	for _, required := range []string{"error_stage", "response_bytes"} {
		if !strings.Contains(payload, required) {
			t.Fatalf("public attempt JSON should contain %q metrics: %s", required, payload)
		}
	}
}
