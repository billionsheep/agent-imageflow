package app

import (
	"encoding/json"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestBuildSceneRegenerationCreateTaskRequestCopiesSourceAndAppliesOverrides(t *testing.T) {
	sourceStructured := json.RawMessage(`{
		"title": "Scene 2 original",
		"purpose": "story scene",
		"prompt": "old prompt",
		"negative_prompt": "old negative",
		"style_preset": "storybook",
		"prompt_template": "{{prompt}}",
		"template_variables": {"channel":"story"},
		"reference_images": [{"asset_id":"asset_old_ref","role":"style"}],
		"character_ids": ["cat_old"],
		"reference_asset_ids": ["asset_old_ref"],
		"prompt_recipe_id": "recipe_old",
		"use_project_visual_context": true,
		"generation_config": {"quality":"medium","seed":111,"model":"old-model"},
		"use_project_quality_profile": true,
		"aspect_ratio": "1:1",
		"output_format": "png",
		"requested_count": 1,
		"provider": "mock",
		"selection_mode": "manual_optional",
		"review_required": false,
		"metadata_json": {
			"source": "codex",
			"session_id": "session_1",
			"batch_id": "batch_1",
			"story_id": "story_1",
			"scene_id": "scene_002",
			"target_path": "stories/story-1/scene-002.png",
			"scene_order": 2
		}
	}`)
	source := domain.Task{
		ID:                  "task_source",
		WorkspaceID:         "ws_default",
		ProjectID:           "prj_demo",
		CampaignID:          "cmp_demo",
		Title:               "Scene 2 original",
		Purpose:             "story scene",
		Prompt:              "old prompt",
		NegativePrompt:      "old negative",
		StylePreset:         "storybook",
		AspectRatio:         "1:1",
		OutputFormat:        "png",
		StructuredInputJSON: sourceStructured,
		Provider:            "mock",
		SelectionMode:       domain.SelectionManualOptional,
		RequestedCount:      1,
	}
	requestedCount := 2
	overridePrompt := "new prompt"
	overrideSelectionMode := domain.SelectionBestOf
	overrideAspectRatio := "3:4"
	overrideModel := "new-model"
	overrideGenerationConfig := json.RawMessage(`{"quality":"high","seed":222}`)

	result, err := buildSceneRegenerationCreateTaskRequest(source, domain.SceneRegenerateRequest{
		SourceTaskID:       "task_source",
		RegenerateReason:   "better composition",
		CreatedBy:          "codex",
		RequestSource:      "rest",
		SceneIdentity:      &domain.SceneIdentity{SessionID: "session_1", BatchID: "batch_1", StoryID: "story_1", SceneID: "scene_002", Source: "codex", TaskSelector: "latest"},
		RegenerationNumber: 3,
		Overrides: domain.SceneRegenerateOverrides{
			Prompt:            &overridePrompt,
			CharacterIDs:      []string{"cat_new", "dog_new"},
			ReferenceAssetIDs: []string{"asset_new_ref"},
			GenerationConfig:  overrideGenerationConfig,
			RequestedCount:    &requestedCount,
			SelectionMode:     &overrideSelectionMode,
			AspectRatio:       &overrideAspectRatio,
			Model:             &overrideModel,
		},
	})
	if err != nil {
		t.Fatalf("buildSceneRegenerationCreateTaskRequest returned error: %v", err)
	}
	if result.Request.Prompt != "new prompt" || result.Request.NegativePrompt != "old negative" {
		t.Fatalf("prompt fields were not copied/overridden: %#v", result.Request)
	}
	if result.Request.RequestedCount != 2 || result.Request.SelectionMode != domain.SelectionBestOf || result.Request.AspectRatio != "3:4" {
		t.Fatalf("basic overrides were not applied: %#v", result.Request)
	}
	if len(result.Request.CharacterIDs) != 2 || result.Request.CharacterIDs[0] != "cat_new" {
		t.Fatalf("character_ids override missing: %#v", result.Request.CharacterIDs)
	}
	if len(result.Request.ReferenceAssetIDs) != 1 || result.Request.ReferenceAssetIDs[0] != "asset_new_ref" {
		t.Fatalf("reference_asset_ids override missing: %#v", result.Request.ReferenceAssetIDs)
	}

	var generationConfig map[string]any
	if err := json.Unmarshal(result.Request.GenerationConfig, &generationConfig); err != nil {
		t.Fatalf("generation_config invalid: %v", err)
	}
	if generationConfig["quality"] != "high" || generationConfig["model"] != "new-model" {
		t.Fatalf("generation_config/model override mismatch: %#v", generationConfig)
	}

	var metadata map[string]any
	if err := json.Unmarshal(result.Request.MetadataJSON, &metadata); err != nil {
		t.Fatalf("metadata invalid: %v", err)
	}
	for key, want := range map[string]string{
		"source":                    "codex",
		"session_id":                "session_1",
		"batch_id":                  "batch_1",
		"story_id":                  "story_1",
		"scene_id":                  "scene_002",
		"target_path":               "stories/story-1/scene-002.png",
		"regenerated_from_task_id":  "task_source",
		"regenerate_reason":         "better composition",
		"regenerate_request_source": "rest",
		"regenerated_by":            "codex",
	} {
		if got := metadata[key]; got != want {
			t.Fatalf("metadata[%s] = %v, want %q; metadata=%#v", key, got, want, metadata)
		}
	}
	if got := int(metadata["regenerate_no"].(float64)); got != 3 {
		t.Fatalf("regenerate_no = %d, want 3", got)
	}
	overrides := metadata["regeneration_overrides"].(map[string]any)
	if overrides["prompt"] != "new prompt" || overrides["model"] != "new-model" {
		t.Fatalf("regeneration_overrides missing safe values: %#v", overrides)
	}
	if result.SceneIdentity.SessionID != "session_1" || result.SceneIdentity.SceneID != "scene_002" {
		t.Fatalf("scene identity not extracted: %#v", result.SceneIdentity)
	}
	if result.CopiedVisualContextSnapshot.CharacterCount != 2 || !result.CopiedVisualContextSnapshot.HasPromptRecipe {
		t.Fatalf("visual context snapshot mismatch: %#v", result.CopiedVisualContextSnapshot)
	}
}

func TestBuildSceneRegenerationCreateTaskRequestRequiresSceneMetadata(t *testing.T) {
	_, err := buildSceneRegenerationCreateTaskRequest(domain.Task{
		ID:                  "task_source",
		ProjectID:           "prj_demo",
		CampaignID:          "cmp_demo",
		StructuredInputJSON: json.RawMessage(`{"prompt":"hello","metadata_json":{"session_id":"session_1"}}`),
		Prompt:              "hello",
		Provider:            "mock",
		RequestedCount:      1,
	}, domain.SceneRegenerateRequest{SourceTaskID: "task_source", RegenerationNumber: 1})
	if err == nil {
		t.Fatal("expected missing scene metadata to fail")
	}
	if got := err.Error(); got != "source task metadata_json must include session_id, batch_id, story_id and scene_id" {
		t.Fatalf("unexpected error %q", got)
	}
}
