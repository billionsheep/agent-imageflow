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
}
