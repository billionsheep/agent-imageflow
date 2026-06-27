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
						"panel_index":            2,
						"scene_id":               "scene_002",
						"narrative_role":         "arrival",
						"dialogue":               "牛奶来啦",
						"dialogue_intent":        "温柔回应小白。",
						"previous_state":         "小白坐在沙发左侧。",
						"trigger_event":          "鸡毛端着牛奶出现。",
						"visible_action":         "鸡毛从右侧出现。",
						"resulting_state":        "两只小狗进入同一空间。",
						"emotion_before":         "嘴硬但期待",
						"emotion_after":          "安心并开始靠近",
						"pose_change":            "从独坐抱书变成抬头看向鸡毛",
						"relationship_shift":     "从等待变成正式同框互动",
						"must_keep_props":        []string{"粉色沙发", "热牛奶"},
						"allowed_changes":        []string{"鸡毛进入画面"},
						"must_change":            []string{"鸡毛必须进入画面"},
						"must_not_keep":          []string{"不能继续只有小白单独在画面里"},
						"state_transition_notes": "必须看得出上一格导致这一格。",
						"target_path":            "stories/pet_story/scene_002.png",
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
	if summary.EmotionBefore != "嘴硬但期待" ||
		summary.EmotionAfter != "安心并开始靠近" ||
		summary.PoseChange != "从独坐抱书变成抬头看向鸡毛" ||
		summary.RelationshipShift != "从等待变成正式同框互动" ||
		len(summary.MustChange) != 1 ||
		len(summary.MustNotKeep) != 1 ||
		summary.StateTransitionNotes != "必须看得出上一格导致这一格。" {
		t.Fatalf("state transition semantics missing: %#v", summary)
	}
}

func TestExtractBatchStoryVisualContextReadsReferenceDiagnostics(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"character_ids":       []string{"dog_jimao", "dog_xiaobai"},
		"reference_asset_ids": []string{"asset_jimao_primary"},
		"prompt_recipe_id":    "pet_dialogue_card",
		"project_visual_context_diagnostics": map[string]any{
			"primary_readiness":                     "image_backed",
			"labels":                                []string{"image_backed", "missing_environment_reference"},
			"summary":                               "image-backed, missing environment reference",
			"active_character_count":                2,
			"character_with_image_count":            2,
			"missing_character_image_count":         0,
			"missing_character_ids":                 []string{},
			"active_reference_count":                1,
			"environment_reference_count":           0,
			"image_reference_count":                 1,
			"negative_prompt_covers_species_drift":  true,
			"identity_signal_present":               true,
			"provider_reference_participation_risk": "likely_resolved_input_files",
		},
	})
	if err != nil {
		t.Fatalf("marshal structured input: %v", err)
	}

	visualContext := extractBatchStoryVisualContext(raw)
	if visualContext.PromptRecipeID != "pet_dialogue_card" {
		t.Fatalf("prompt recipe id missing: %#v", visualContext)
	}
	if visualContext.ReferenceDiagnostics == nil {
		t.Fatalf("reference diagnostics missing: %#v", visualContext)
	}
	if visualContext.ReferenceDiagnostics.PrimaryReadiness != "image_backed" {
		t.Fatalf("primary readiness = %q, want image_backed; value=%#v", visualContext.ReferenceDiagnostics.PrimaryReadiness, visualContext.ReferenceDiagnostics)
	}
	if !visualContext.ReferenceDiagnostics.NegativePromptCoversSpeciesDrift {
		t.Fatalf("negative prompt coverage should be preserved: %#v", visualContext.ReferenceDiagnostics)
	}
	if visualContext.ReferenceDiagnostics.ProviderReferenceParticipationRisk != "likely_resolved_input_files" {
		t.Fatalf("provider risk missing: %#v", visualContext.ReferenceDiagnostics)
	}
}
