package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestNormalizeProjectVisualContextAssignsDefaults(t *testing.T) {
	now := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	visualContext, err := normalizeProjectVisualContext(domain.ProjectVisualContext{
		Characters: []domain.CharacterProfile{
			{
				ID:                " dog_1 ",
				Name:              "",
				Forbidden:         []string{" watermark ", "", "watermark"},
				ReferenceAssetIDs: []string{" asset_ref ", "asset_ref"},
			},
		},
		References: []domain.ProjectReferenceBinding{
			{ID: " style_ref ", AssetID: " asset_style ", Purpose: "", Weight: 9},
		},
		PromptRecipes: []domain.PromptRecipe{
			{
				ID:               " story_recipe ",
				PromptBlocks:     []domain.PromptBlock{{Text: " {{prompt}} in a cozy room "}, {Text: " "}},
				GenerationConfig: []byte(`{"quality":"high"}`),
			},
		},
	}, now)
	if err != nil {
		t.Fatalf("normalizeProjectVisualContext returned error: %v", err)
	}
	if visualContext.Characters[0].Name != "dog_1" || visualContext.Characters[0].Status != "active" {
		t.Fatalf("character defaults were not applied: %#v", visualContext.Characters[0])
	}
	if len(visualContext.Characters[0].Forbidden) != 1 || len(visualContext.Characters[0].ReferenceAssetIDs) != 1 {
		t.Fatalf("character lists were not normalized: %#v", visualContext.Characters[0])
	}
	if visualContext.References[0].Purpose != "style" || visualContext.References[0].Weight != 5 {
		t.Fatalf("reference defaults were not applied: %#v", visualContext.References[0])
	}
	if visualContext.PromptRecipes[0].Name != "story_recipe" || len(visualContext.PromptRecipes[0].PromptBlocks) != 1 {
		t.Fatalf("recipe defaults were not applied: %#v", visualContext.PromptRecipes[0])
	}
}

func TestNormalizeProjectVisualContextKeepsCharacterReferencePolicyAndLockNotes(t *testing.T) {
	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)
	visualContext, err := normalizeProjectVisualContext(domain.ProjectVisualContext{
		Characters: []domain.CharacterProfile{
			{
				ID:                  "dog_milo",
				Name:                "Milo",
				Status:              "active",
				ReferencePolicy:     " primary_plus_references ",
				AppearanceLockNotes: " keep white muzzle and red scarf ",
			},
		},
	}, now)
	if err != nil {
		t.Fatalf("normalizeProjectVisualContext returned error: %v", err)
	}
	character := visualContext.Characters[0]
	if character.ReferencePolicy != "primary_plus_references" {
		t.Fatalf("reference_policy was not normalized: %#v", character)
	}
	if character.AppearanceLockNotes != "keep white muzzle and red scarf" {
		t.Fatalf("appearance_lock_notes was not normalized: %#v", character)
	}
}

func TestValidateProjectVisualContextAssetScopesRejectsCrossProjectAsset(t *testing.T) {
	scope := domain.Scope{WorkspaceID: "ws_a", ProjectID: "prj_a"}
	err := validateProjectVisualContextAssetScopes(context.Background(), scope, domain.ProjectVisualContext{
		Characters: []domain.CharacterProfile{{ID: "char_1", Name: "Milo", Status: "active", PrimaryAssetID: "asset_cross"}},
	}, func(ctx context.Context, assetID string) (domain.Scope, error) {
		return domain.Scope{WorkspaceID: "ws_a", ProjectID: "prj_b", CampaignID: "cmp_b"}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "not ws_a/prj_a") {
		t.Fatalf("expected cross project asset rejection, got %v", err)
	}
}

func TestBuildReferenceParticipationDiagnosticsCountsAssetsAndInputFiles(t *testing.T) {
	diagnostics := buildReferenceParticipationDiagnostics(domain.CreateTaskRequest{
		ReferenceImages: []domain.ReferenceImage{
			{AssetID: "asset_primary", Source: "project_visual_context", MimeType: "image/png"},
			{InputFileID: "inp_manual", Source: "agent-imageflow-upload", MimeType: "image/webp"},
			{URL: "https://example.test/ref.png", Source: "agent-imageflow-remote-url", MimeType: "image/png"},
		},
	}, &resolvedTaskInputFiles{
		ReferenceImages: []resolvedTaskInputFile{
			{Kind: domain.InputFileKindReference, FilePath: "/tmp/asset_primary.png", MimeType: "image/png"},
			{InputFileID: "inp_manual", Kind: domain.InputFileKindReference, FilePath: "/tmp/manual.webp", MimeType: "image/webp"},
			{InputFileID: "inp_remote", Kind: domain.InputFileKindReference, FilePath: "/tmp/remote.png", MimeType: "image/png"},
		},
	})
	if diagnostics.ReferenceAssetCount != 1 {
		t.Fatalf("expected 1 asset reference, got %#v", diagnostics)
	}
	if diagnostics.ReferenceInputFileCount != 2 {
		t.Fatalf("expected 2 input-file references, got %#v", diagnostics)
	}
	if diagnostics.ProviderReferenceParticipation != "resolved_input_files" {
		t.Fatalf("expected resolved input participation, got %#v", diagnostics)
	}
	if !containsString(diagnostics.ProviderReferenceSources, "agent-imageflow-upload") ||
		!containsString(diagnostics.ProviderReferenceMIMETypes, "image/webp") {
		t.Fatalf("source/mime diagnostics missing: %#v", diagnostics)
	}
}

func TestBuildProjectVisualContextReferenceDiagnosticsFlagsTextConstrainedAndWeakSpeciesLock(t *testing.T) {
	diagnostics := buildProjectVisualContextReferenceDiagnostics(domain.ProjectVisualContext{
		Characters: []domain.CharacterProfile{
			{
				ID:         "dog_jimao",
				Name:       "鸡毛",
				Status:     "active",
				Role:       "mascot",
				Appearance: "warm beige mascot with rounded ears",
			},
		},
		PromptRecipes: []domain.PromptRecipe{
			{
				ID:             "pet_card",
				Name:           "Pet card",
				Status:         "active",
				NegativePrompt: "watermark, unreadable text",
			},
		},
	})

	if diagnostics.PrimaryReadiness != "text_constrained" {
		t.Fatalf("primary readiness = %q, want text_constrained; value=%#v", diagnostics.PrimaryReadiness, diagnostics)
	}
	for _, want := range []string{"text_constrained", "missing_environment_reference", "weak_species_lock"} {
		if !containsString(diagnostics.Labels, want) {
			t.Fatalf("expected diagnostics labels to contain %q: %#v", want, diagnostics)
		}
	}
	if diagnostics.ProviderReferenceParticipationRisk != "descriptor_only_risk" {
		t.Fatalf("provider reference participation risk = %q, want descriptor_only_risk; value=%#v", diagnostics.ProviderReferenceParticipationRisk, diagnostics)
	}
	if diagnostics.MissingCharacterImageCount != 1 || !containsString(diagnostics.MissingCharacterIDs, "dog_jimao") {
		t.Fatalf("missing character image diagnostics incorrect: %#v", diagnostics)
	}
	if diagnostics.NegativePromptCoversSpeciesDrift {
		t.Fatalf("negative prompt should not look species-safe: %#v", diagnostics)
	}
	if diagnostics.IdentitySignalPresent {
		t.Fatalf("identity signal should stay weak for mascot-only wording: %#v", diagnostics)
	}
}

func TestApplyProjectVisualContextExpandsRecipeReferencesAndSnapshot(t *testing.T) {
	req := domain.CreateTaskRequest{
		Title:                   "Scene 1",
		Prompt:                  "Milo and Orange Nap share a moon cake",
		CharacterIDs:            []string{"dog_milo"},
		PromptRecipeID:          "pet_story",
		UseProjectVisualContext: true,
		MetadataJSON:            []byte(`{"story_id":"story_moon","scene_id":"scene_001"}`),
	}
	visualContext := domain.ProjectVisualContext{
		Characters: []domain.CharacterProfile{
			{
				ID:                "dog_milo",
				Name:              "Milo",
				Status:            "active",
				Appearance:        "fluffy white dog",
				PrimaryAssetID:    "asset_milo_primary",
				ReferenceAssetIDs: []string{"asset_milo_side"},
			},
		},
		References: []domain.ProjectReferenceBinding{
			{ID: "style_soft", AssetID: "asset_soft_style", Purpose: "style", Status: "active", Weight: 1.5},
			{ID: "scene_archived", AssetID: "asset_old_scene", Purpose: "scene", Status: "archived"},
		},
		PromptRecipes: []domain.PromptRecipe{
			{
				ID:                  "pet_story",
				Name:                "Cute pet story",
				Status:              "active",
				PromptBlocks:        []domain.PromptBlock{{Role: "scene", Text: "Scene: {{prompt}}"}, {Role: "camera", Text: "warm close-up, children-book mood"}},
				NegativePrompt:      "watermark, scary face",
				DefaultAspectRatio:  "3:4",
				DefaultOutputFormat: "png",
				DefaultProvider:     "mock",
				DefaultModel:        "mock-image",
				GenerationConfig:    []byte(`{"quality":"high"}`),
			},
		},
	}

	expanded, snapshot, err := applyProjectVisualContext(req, visualContext)
	if err != nil {
		t.Fatalf("applyProjectVisualContext returned error: %v", err)
	}
	if expanded.Prompt != "Scene: Milo and Orange Nap share a moon cake\n\nwarm close-up, children-book mood" {
		t.Fatalf("unexpected expanded prompt: %q", expanded.Prompt)
	}
	if expanded.NegativePrompt != "watermark, scary face" || expanded.AspectRatio != "3:4" || expanded.Provider != "mock" {
		t.Fatalf("recipe defaults were not applied: %#v", expanded)
	}
	var generationConfig map[string]any
	if err := json.Unmarshal(expanded.GenerationConfig, &generationConfig); err != nil {
		t.Fatalf("generation_config invalid: %v", err)
	}
	if generationConfig["quality"] != "high" || generationConfig["model"] != "mock-image" {
		t.Fatalf("generation_config defaults missing: %#v", generationConfig)
	}
	referenceAssets := []string{}
	for _, ref := range expanded.ReferenceImages {
		referenceAssets = append(referenceAssets, ref.AssetID)
	}
	for _, want := range []string{"asset_milo_primary", "asset_milo_side", "asset_soft_style"} {
		if !containsString(referenceAssets, want) {
			t.Fatalf("expanded references missing %s: %#v", want, expanded.ReferenceImages)
		}
	}
	if containsString(referenceAssets, "asset_old_scene") {
		t.Fatalf("archived reference should not be expanded: %#v", expanded.ReferenceImages)
	}
	if snapshot == nil || snapshot.PromptRecipe == nil || snapshot.PromptRecipe.ID != "pet_story" {
		t.Fatalf("snapshot missing recipe: %#v", snapshot)
	}
	if snapshot == nil || snapshot.ReferenceDiagnostics == nil {
		t.Fatalf("snapshot missing reference diagnostics: %#v", snapshot)
	}
	if snapshot.ReferenceDiagnostics.PrimaryReadiness != "image_backed" {
		t.Fatalf("snapshot primary readiness = %q, want image_backed; value=%#v", snapshot.ReferenceDiagnostics.PrimaryReadiness, snapshot.ReferenceDiagnostics)
	}
	for _, want := range []string{"image_backed", "missing_environment_reference", "weak_species_lock"} {
		if !containsString(snapshot.ReferenceDiagnostics.Labels, want) {
			t.Fatalf("expected snapshot diagnostics label %q in %#v", want, snapshot.ReferenceDiagnostics)
		}
	}
	var metadata map[string]any
	if err := json.Unmarshal(expanded.MetadataJSON, &metadata); err != nil {
		t.Fatalf("metadata invalid: %v", err)
	}
	if _, ok := metadata[visualContextSnapshotKey]; !ok {
		t.Fatalf("visual context snapshot missing from metadata: %#v", metadata)
	}
	diagnostics, ok := metadata[projectVisualContextDiagnosticsKey].(map[string]any)
	if !ok {
		t.Fatalf("project visual context diagnostics missing from metadata: %#v", metadata)
	}
	if diagnostics["primary_readiness"] != "image_backed" {
		t.Fatalf("unexpected metadata diagnostics primary readiness: %#v", diagnostics)
	}
}

func TestApplyPromptRecipeKeepsExplicitTaskFields(t *testing.T) {
	req := domain.CreateTaskRequest{
		Prompt:           "base scene",
		NegativePrompt:   "explicit negative",
		AspectRatio:      "16:9",
		OutputFormat:     "webp",
		Provider:         "mock",
		GenerationConfig: []byte(`{"quality":"low","model":"explicit-model"}`),
	}
	recipe := domain.PromptRecipe{
		PromptBlocks:        []domain.PromptBlock{{Text: "style block"}},
		NegativePrompt:      "recipe negative",
		DefaultAspectRatio:  "3:4",
		DefaultOutputFormat: "png",
		DefaultProvider:     "openai-compatible",
		DefaultModel:        "recipe-model",
		GenerationConfig:    []byte(`{"quality":"high","background":"soft"}`),
	}
	expanded, err := applyPromptRecipe(req, recipe)
	if err != nil {
		t.Fatalf("applyPromptRecipe returned error: %v", err)
	}
	if expanded.Prompt != "base scene\n\nstyle block" {
		t.Fatalf("unexpected prompt: %q", expanded.Prompt)
	}
	if expanded.NegativePrompt != "explicit negative" || expanded.AspectRatio != "16:9" || expanded.OutputFormat != "webp" || expanded.Provider != "mock" {
		t.Fatalf("explicit task fields were overwritten: %#v", expanded)
	}
	var generationConfig map[string]any
	if err := json.Unmarshal(expanded.GenerationConfig, &generationConfig); err != nil {
		t.Fatalf("generation_config invalid: %v", err)
	}
	if generationConfig["quality"] != "low" || generationConfig["model"] != "explicit-model" || generationConfig["background"] != "soft" {
		t.Fatalf("generation_config precedence mismatch: %#v", generationConfig)
	}
}
