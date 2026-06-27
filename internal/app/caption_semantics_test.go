package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestNormalizeCaptionLineageMetadataPromotesCanonicalNestedObject(t *testing.T) {
	metadata, lineage, err := normalizeCaptionLineageMetadata(json.RawMessage(`{
		"source": "mcp",
		"derived_from_asset_id": "asset_original",
		"derivation_type": "caption_edit",
		"caption_text": "因为喜欢你",
		"caption_style": "rounded bubble",
		"source_task_id": "task_original",
		"source_scene_id": "scene_001",
		"speaker_character_id": "dog_xiaobai",
		"bubble_anchor": "top_right",
		"tail_direction": "toward_left",
		"caption_intent": "confession",
		"avoid_covering_subjects": true
	}`))
	if err != nil {
		t.Fatalf("normalize caption lineage metadata: %v", err)
	}
	if lineage == nil {
		t.Fatal("expected caption lineage summary")
	}

	var payload map[string]any
	if err := json.Unmarshal(metadata, &payload); err != nil {
		t.Fatalf("metadata should remain valid json: %v", err)
	}
	nested, ok := payload["caption_lineage"].(map[string]any)
	if !ok {
		t.Fatalf("expected canonical nested caption_lineage: %#v", payload)
	}
	if nested["speaker_character_id"] != "dog_xiaobai" ||
		nested["bubble_anchor"] != "top_right" ||
		nested["tail_direction"] != "toward_left" ||
		nested["caption_intent"] != "confession" ||
		nested["avoid_covering_subjects"] != true {
		t.Fatalf("unexpected nested caption_lineage payload: %#v", nested)
	}
}

func TestAppendCaptionSemanticsPromptAddsSpeakerBubbleHints(t *testing.T) {
	avoid := true
	prompt := appendCaptionSemanticsPrompt("Keep the original composition recognizable.", &domain.CaptionLineageSummary{
		CaptionText:           "因为喜欢你",
		SpeakerCharacterID:    "dog_xiaobai",
		BubbleAnchor:          "top_right",
		TailDirection:         "toward_left",
		CaptionIntent:         "confession",
		AvoidCoveringSubjects: &avoid,
	})

	for _, expected := range []string{
		"Caption semantics:",
		`Add exactly one readable caption text block: "因为喜欢你".`,
		"Speaker character id: dog_xiaobai.",
		"Bubble anchor: top_right.",
		"Bubble tail direction: toward_left.",
		"Caption intent: confession.",
		"Do not cover the main characters, their faces, or key props.",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected prompt to contain %q, got %q", expected, prompt)
		}
	}
}
