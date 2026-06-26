package provider

import (
	"encoding/json"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestTaskProviderParametersIncludesAdvancedInputDescriptors(t *testing.T) {
	structured, err := json.Marshal(map[string]any{
		"reference_images": []domain.ReferenceImage{
			{ID: "ref_local", Role: "edit_target", Source: "web-indexeddb", MimeType: "image/png"},
		},
		"mask_image": domain.MaskImage{
			TargetImageID: "ref_local",
			Source:        "web-mask-draft",
			MimeType:      "image/png",
			HasMask:       true,
		},
		"generation_config": map[string]any{"quality": "high"},
		"visual_context_snapshot": domain.VisualContextSnapshot{
			Source:            "project",
			CharacterIDs:      []string{"dog_milo"},
			ReferenceAssetIDs: []string{"asset_milo_primary"},
			PromptRecipeID:    "pet_story",
		},
		"reference_asset_count":            1,
		"reference_input_file_count":       1,
		"provider_reference_participation": "resolved_input_files",
		"provider_reference_sources":       []string{"project_visual_context"},
		"provider_reference_mime_types":    []string{"image/png"},
		"story_context_v1": map[string]any{
			"story_id":        "pet_story",
			"story_revision":  "rev_001",
			"story_plan_hash": "sha256:story-plan",
			"generation_mode": "sequential_previous_panel",
			"resolved_reference_assets": []map[string]any{
				{"role": "character_reference", "asset_id": "asset_milo_primary"},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal structured input: %v", err)
	}

	raw := taskProviderParameters(domain.Task{StructuredInputJSON: structured}, map[string]any{"slot": 0})
	var parameters map[string]any
	if err := json.Unmarshal(raw, &parameters); err != nil {
		t.Fatalf("parameters JSON invalid: %v", err)
	}

	references := parameters["reference_images"].([]any)
	reference := references[0].(map[string]any)
	if reference["id"] != "ref_local" || reference["source"] != "web-indexeddb" {
		t.Fatalf("reference descriptor missing from parameters: %#v", parameters)
	}
	mask := parameters["mask_image"].(map[string]any)
	if mask["target_image_id"] != "ref_local" || mask["source"] != "web-mask-draft" || mask["has_mask"] != true {
		t.Fatalf("mask descriptor missing from parameters: %#v", parameters)
	}
	generationConfig := parameters["generation_config"].(map[string]any)
	if generationConfig["quality"] != "high" {
		t.Fatalf("generation_config missing from parameters: %#v", parameters)
	}
	visualContext := parameters["visual_context_snapshot"].(map[string]any)
	if visualContext["prompt_recipe_id"] != "pet_story" {
		t.Fatalf("visual_context_snapshot missing from parameters: %#v", parameters)
	}
	if parameters["reference_asset_count"] != float64(1) ||
		parameters["reference_input_file_count"] != float64(1) ||
		parameters["provider_reference_participation"] != "resolved_input_files" {
		t.Fatalf("reference participation diagnostics missing from parameters: %#v", parameters)
	}
	storyContext := parameters["story_context_v1"].(map[string]any)
	if storyContext["story_id"] != "pet_story" || storyContext["generation_mode"] != "sequential_previous_panel" {
		t.Fatalf("story_context_v1 missing from parameters: %#v", parameters)
	}
}

func TestTaskProviderParametersIncludesCaptionLineageSummary(t *testing.T) {
	structured, err := json.Marshal(map[string]any{
		"metadata_json": map[string]any{
			"derived_from_asset_id": "asset_original",
			"derivation_type":       "caption_edit",
			"caption_text":          "今天也要可爱",
			"caption_style":         "rounded speech bubble, handwritten",
			"source_task_id":        "task_original",
			"source_scene_id":       "scene_001",
		},
	})
	if err != nil {
		t.Fatalf("marshal structured input: %v", err)
	}

	raw := taskProviderParameters(domain.Task{StructuredInputJSON: structured}, map[string]any{"slot": 0})
	var parameters map[string]any
	if err := json.Unmarshal(raw, &parameters); err != nil {
		t.Fatalf("parameters JSON invalid: %v", err)
	}

	lineage := parameters["caption_lineage"].(map[string]any)
	if lineage["derived_from_asset_id"] != "asset_original" ||
		lineage["derivation_type"] != "caption_edit" ||
		lineage["caption_text"] != "今天也要可爱" ||
		lineage["caption_style"] != "rounded speech bubble, handwritten" ||
		lineage["source_task_id"] != "task_original" ||
		lineage["source_scene_id"] != "scene_001" {
		t.Fatalf("caption lineage missing from provider parameters: %#v", parameters)
	}
}
