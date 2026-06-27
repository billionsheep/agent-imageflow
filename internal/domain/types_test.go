package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCaptionLineageFromMetadataJSONMergesNestedSpeakerBubbleFields(t *testing.T) {
	raw := json.RawMessage(`{
		"caption_text": "根字段文案",
		"auto_select_derivative": false,
		"caption_lineage": {
			"derived_from_asset_id": "asset_original",
			"derivation_type": "caption_edit",
			"caption_text": "嵌套文案",
			"caption_style": "rounded speech bubble",
			"source_task_id": "task_original",
			"source_scene_id": "scene_001",
			"speaker_character_id": "dog_jimao",
			"bubble_anchor": "top_right",
			"tail_direction": "toward_left",
			"caption_intent": "comfort",
			"avoid_covering_subjects": true,
			"auto_select_derivative": true
		}
	}`)

	lineage := CaptionLineageFromMetadataJSON(raw)
	if lineage == nil {
		t.Fatal("expected caption lineage to be extracted")
	}
	if lineage.DerivedFromAssetID != "asset_original" ||
		lineage.DerivationType != "caption_edit" ||
		lineage.CaptionText != "根字段文案" ||
		lineage.CaptionStyle != "rounded speech bubble" ||
		lineage.SourceTaskID != "task_original" ||
		lineage.SourceSceneID != "scene_001" ||
		lineage.SpeakerCharacterID != "dog_jimao" ||
		lineage.BubbleAnchor != "top_right" ||
		lineage.TailDirection != "toward_left" ||
		lineage.CaptionIntent != "comfort" ||
		lineage.AutoSelectDerivative == nil ||
		!*lineage.AutoSelectDerivative ||
		lineage.AvoidCoveringSubjects == nil ||
		!*lineage.AvoidCoveringSubjects {
		t.Fatalf("unexpected caption lineage: %#v", lineage)
	}
}

func TestCaptionLineageFromStructuredInputPrefersStructuredSpeakerBubbleFields(t *testing.T) {
	raw := json.RawMessage(`{
		"caption_lineage": {
			"derived_from_asset_id": "asset_structured",
			"caption_text": "结构化文案",
			"speaker_character_id": "dog_xiaobai",
			"bubble_anchor": "above_speaker",
			"tail_direction": "toward_speaker",
			"caption_intent": "confession",
			"avoid_covering_subjects": false,
			"auto_select_derivative": false
		},
		"metadata_json": {
			"caption_lineage": {
				"derived_from_asset_id": "asset_metadata",
				"caption_text": "元数据文案",
				"speaker_character_id": "dog_jimao",
				"bubble_anchor": "top_left",
				"tail_direction": "toward_left",
				"caption_intent": "tease",
				"avoid_covering_subjects": true,
				"auto_select_derivative": true
			}
		}
	}`)

	lineage := CaptionLineageFromStructuredInput(raw)
	if lineage == nil {
		t.Fatal("expected structured caption lineage to be extracted")
	}
	if lineage.DerivedFromAssetID != "asset_structured" ||
		lineage.CaptionText != "结构化文案" ||
		lineage.SpeakerCharacterID != "dog_xiaobai" ||
		lineage.BubbleAnchor != "above_speaker" ||
		lineage.TailDirection != "toward_speaker" ||
		lineage.CaptionIntent != "confession" ||
		lineage.AutoSelectDerivative == nil ||
		*lineage.AutoSelectDerivative ||
		lineage.AvoidCoveringSubjects == nil ||
		*lineage.AvoidCoveringSubjects {
		t.Fatalf("structured caption lineage should win: %#v", lineage)
	}
}

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
