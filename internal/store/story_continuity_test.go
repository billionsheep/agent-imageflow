package store

import (
	"encoding/json"
	"testing"
)

func TestExtractBatchStoryContinuitySummaryReadsStoryContextV1(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"provider_reference_participation": "resolved_input_files",
		"metadata_json": map[string]any{
			"story_id":                "pet_story",
			"scene_id":                "scene_002",
			"panel_index":             2,
			"previous_panel_asset_id": "asset_panel_001_selected",
			"story_context_v1": map[string]any{
				"story_revision":  "rev_001",
				"story_plan_hash": "sha256:story-plan",
				"generation_mode": "sequential_previous_panel",
				"resolved_reference_assets": []map[string]any{
					{"role": "previous_panel_reference", "asset_id": "asset_panel_001_selected"},
					{"role": "character_reference", "asset_id": "asset_xiaobai_primary"},
				},
				"continuity_warnings": []map[string]any{
					{"code": "mock_only", "message": "mock only validates data flow"},
				},
				"panel_plan": []map[string]any{
					{"panel_index": 1, "scene_id": "scene_001", "dialogue": "才没有等你"},
					{
						"panel_index":     2,
						"scene_id":        "scene_002",
						"narrative_role":  "arrival",
						"dialogue":        "牛奶来啦",
						"dialogue_intent": "温柔回应小白。",
						"previous_state":  "小白坐在沙发左侧。",
						"trigger_event":   "鸡毛端着牛奶出现。",
						"visible_action":  "鸡毛从右侧出现。",
						"resulting_state": "两只小狗进入同一空间。",
						"must_keep_props": []string{"粉色沙发", "热牛奶"},
						"allowed_changes": []string{"鸡毛进入画面"},
						"target_path":     "stories/pet_story/scene_002.png",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal structured input: %v", err)
	}

	summary := extractBatchStoryContinuitySummary(raw)
	if summary.PanelIndex != 2 || summary.NarrativeRole != "arrival" {
		t.Fatalf("unexpected continuity summary panel fields: %#v", summary)
	}
	if summary.PreviousPanelAssetID != "asset_panel_001_selected" {
		t.Fatalf("previous panel asset was not extracted: %#v", summary)
	}
	if summary.ProviderReferenceParticipation != "resolved_input_files" {
		t.Fatalf("provider reference participation missing: %#v", summary)
	}
	if len(summary.ResolvedReferenceAssets) != 2 {
		t.Fatalf("resolved reference assets length = %d, want 2; value=%#v", len(summary.ResolvedReferenceAssets), summary.ResolvedReferenceAssets)
	}
	if len(summary.ContinuityWarnings) != 1 || summary.ContinuityWarnings[0].Code != "mock_only" {
		t.Fatalf("continuity warnings missing: %#v", summary.ContinuityWarnings)
	}
	if len(summary.MustKeepProps) != 2 || len(summary.AllowedChanges) != 1 {
		t.Fatalf("panel causality arrays missing: %#v", summary)
	}
}
