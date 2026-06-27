package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestApplyStoryContextBindingsToRequestExpandsVisualContextAndPreviousPanelBindings(t *testing.T) {
	req := domain.CreateTaskRequest{}
	visualContext := domain.ProjectVisualContext{
		Characters: []domain.CharacterProfile{
			{
				ID:                "dog_xiaobai",
				Name:              "小白",
				Status:            "active",
				PrimaryAssetID:    "asset_xiaobai_primary",
				ReferenceAssetIDs: []string{"asset_xiaobai_side"},
			},
		},
		References: []domain.ProjectReferenceBinding{
			{
				ID:      "warm_pink_living_room",
				AssetID: "asset_room",
				Purpose: "scene",
				Status:  "active",
			},
		},
	}
	story := &domain.StoryContextV1{
		ReferenceBindings: domain.StoryReferenceBindings{
			"character_reference":      {"dog_xiaobai"},
			"environment_reference":    {"warm_pink_living_room"},
			"previous_panel_reference": {"asset_panel_001_selected"},
		},
	}

	expanded, err := applyStoryContextBindingsToRequest(req, visualContext, story)
	if err != nil {
		t.Fatalf("applyStoryContextBindingsToRequest returned error: %v", err)
	}
	if len(expanded.CharacterIDs) != 1 || expanded.CharacterIDs[0] != "dog_xiaobai" {
		t.Fatalf("character ids were not expanded from story_context_v1: %#v", expanded.CharacterIDs)
	}
	if len(expanded.ReferenceAssetIDs) != 1 || expanded.ReferenceAssetIDs[0] != "asset_room" {
		t.Fatalf("reference asset ids were not expanded from environment binding: %#v", expanded.ReferenceAssetIDs)
	}
	if len(expanded.ReferenceImages) != 1 {
		t.Fatalf("previous panel reference was not injected as direct reference image: %#v", expanded.ReferenceImages)
	}
	if expanded.ReferenceImages[0].AssetID != "asset_panel_001_selected" || expanded.ReferenceImages[0].Role != "previous_panel_reference" {
		t.Fatalf("unexpected injected previous panel reference: %#v", expanded.ReferenceImages[0])
	}
}

func TestPrepareStoryContextV1ForTaskPopulatesResolvedReferencesAndPanelMetadata(t *testing.T) {
	story := &domain.StoryContextV1{
		SchemaVersion:  "1.0",
		StoryID:        "pet_story",
		StoryRevision:  "rev_001",
		StoryPlanHash:  "sha256:story-plan",
		GenerationMode: domain.StoryGenerationModeSequentialPreviousPanel,
		PanelPlan: []domain.StoryPanelPlanEntry{
			{
				PanelIndex:    1,
				SceneID:       "scene_001",
				NarrativeRole: "setup",
				Dialogue:      "才没有等你",
			},
			{
				PanelIndex:           2,
				SceneID:              "scene_002",
				NarrativeRole:        "arrival",
				PreviousState:        "小白坐在沙发左侧。",
				TriggerEvent:         "鸡毛端着牛奶出现。",
				VisibleAction:        "鸡毛从右侧出现。",
				ResultingState:       "两只小狗进入同一空间。",
				Dialogue:             "牛奶来啦",
				DialogueIntent:       "温柔回应小白。",
				EmotionBefore:        "嘴硬但期待",
				EmotionAfter:         "安心并开始靠近",
				PoseChange:           "从独坐抱书变成抬头看向鸡毛",
				RelationshipShift:    "从等待变成正式同框互动",
				MustKeepProps:        []string{"粉色沙发", "热牛奶"},
				AllowedChanges:       []string{"鸡毛进入画面"},
				MustChange:           []string{"鸡毛必须进入画面", "小白视线转向鸡毛"},
				MustNotKeep:          []string{"不能继续只有小白单独在画面里"},
				StateTransitionNotes: "重点是既保留同一客厅和沙发位置，又明确让关系往靠近推进。",
				TargetPath:           "stories/pet_story/scene_002.png",
			},
		},
		ReferenceBindings: domain.StoryReferenceBindings{
			"character_reference":      {"dog_xiaobai"},
			"environment_reference":    {"warm_pink_living_room"},
			"previous_panel_reference": {"asset_panel_001_selected"},
		},
		ContinuityPolicy: domain.StoryContinuityPolicy{
			Mode:                         domain.StoryGenerationModeSequentialPreviousPanel,
			RequirePreviousSelectedAsset: true,
			MaxCandidatesPerPanel:        2,
		},
	}
	req := domain.CreateTaskRequest{
		RequestedCount: 2,
		SelectionMode:  domain.SelectionManualOptional,
		ReferenceImages: []domain.ReferenceImage{
			{AssetID: "asset_panel_001_selected", Role: "previous_panel_reference", Source: "story_context_v1"},
			{AssetID: "asset_xiaobai_primary", Role: "character_reference", Source: "project_visual_context"},
			{AssetID: "asset_room", Role: "environment_reference", Source: "project_visual_context"},
		},
	}
	resolved := &resolvedTaskInputFiles{
		ReferenceImages: []resolvedTaskInputFile{
			{Kind: domain.InputFileKindReference, Role: "previous_panel_reference", FilePath: "/tmp/panel-001.png", MimeType: "image/png"},
			{Kind: domain.InputFileKindReference, Role: "character_reference", FilePath: "/tmp/character.png", MimeType: "image/png"},
			{Kind: domain.InputFileKindReference, Role: "environment_reference", FilePath: "/tmp/room.png", MimeType: "image/png"},
		},
	}
	metadataRaw, err := json.Marshal(map[string]any{
		"story_id":         "pet_story",
		"scene_id":         "scene_002",
		"story_context_v1": story,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	diagnostics := domain.ReferenceParticipationDiagnostics{
		ProviderReferenceParticipation: "resolved_input_files",
		ProviderReferenceSources:       []string{"story_context_v1", "project_visual_context"},
		ProviderReferenceMIMETypes:     []string{"image/png"},
	}

	updatedStory, updatedMetadataRaw, err := prepareStoryContextV1ForTask(
		domain.Scope{WorkspaceID: "ws_default", ProjectID: "prj_demo", CampaignID: "cmp_demo"},
		metadataRaw,
		req,
		resolved,
		diagnostics,
		func(assetID string) (storyContinuityAssetSnapshot, error) {
			if assetID != "asset_panel_001_selected" {
				return storyContinuityAssetSnapshot{
					Scope:       domain.Scope{WorkspaceID: "ws_default", ProjectID: "prj_demo", CampaignID: "cmp_demo"},
					AssetStatus: domain.AssetApproved,
				}, nil
			}
			return storyContinuityAssetSnapshot{
				Scope:        domain.Scope{WorkspaceID: "ws_default", ProjectID: "prj_demo", CampaignID: "cmp_demo"},
				AssetStatus:  domain.AssetApproved,
				MetadataJSON: json.RawMessage(`{"story_id":"pet_story","scene_id":"scene_001","panel_index":1}`),
			}, nil
		},
	)
	if err != nil {
		t.Fatalf("prepareStoryContextV1ForTask returned error: %v", err)
	}
	if len(updatedStory.ResolvedReferenceAssets) != 3 {
		t.Fatalf("resolved reference assets length = %d, want 3; value=%#v", len(updatedStory.ResolvedReferenceAssets), updatedStory.ResolvedReferenceAssets)
	}
	if updatedStory.ResolvedReferenceAssets[0].Role != "previous_panel_reference" || updatedStory.ResolvedReferenceAssets[0].AssetID != "asset_panel_001_selected" {
		t.Fatalf("previous panel reference was not preserved in resolved assets: %#v", updatedStory.ResolvedReferenceAssets)
	}
	var updatedMetadata map[string]any
	if err := json.Unmarshal(updatedMetadataRaw, &updatedMetadata); err != nil {
		t.Fatalf("updated metadata invalid: %v", err)
	}
	if got := int(updatedMetadata["panel_index"].(float64)); got != 2 {
		t.Fatalf("panel_index = %d, want 2; metadata=%#v", got, updatedMetadata)
	}
	if got := int(updatedMetadata["scene_order"].(float64)); got != 2 {
		t.Fatalf("scene_order = %d, want 2; metadata=%#v", got, updatedMetadata)
	}
	for key, want := range map[string]string{
		"narrative_role":          "arrival",
		"previous_panel_asset_id": "asset_panel_001_selected",
		"dialogue":                "牛奶来啦",
		"dialogue_intent":         "温柔回应小白。",
		"emotion_before":          "嘴硬但期待",
		"emotion_after":           "安心并开始靠近",
		"pose_change":             "从独坐抱书变成抬头看向鸡毛",
		"relationship_shift":      "从等待变成正式同框互动",
		"state_transition_notes":  "重点是既保留同一客厅和沙发位置，又明确让关系往靠近推进。",
		"target_path":             "stories/pet_story/scene_002.png",
	} {
		if got := strings.TrimSpace(updatedMetadata[key].(string)); got != want {
			t.Fatalf("metadata[%s] = %q, want %q; metadata=%#v", key, got, want, updatedMetadata)
		}
	}
	if got := strings.TrimSpace(updatedMetadata["provider_reference_participation"].(string)); got != "resolved_input_files" {
		t.Fatalf("provider_reference_participation = %q, want resolved_input_files", got)
	}
	if got := updatedMetadata["must_change"].([]any); len(got) != 2 {
		t.Fatalf("must_change length = %d, want 2; metadata=%#v", len(got), updatedMetadata)
	}
	if got := updatedMetadata["must_not_keep"].([]any); len(got) != 1 {
		t.Fatalf("must_not_keep length = %d, want 1; metadata=%#v", len(got), updatedMetadata)
	}
}

func TestPrepareStoryContextV1ForTaskRejectsNonSelectedPreviousPanelAsset(t *testing.T) {
	story := &domain.StoryContextV1{
		SchemaVersion:  "1.0",
		StoryID:        "pet_story",
		StoryRevision:  "rev_001",
		StoryPlanHash:  "sha256:story-plan",
		GenerationMode: domain.StoryGenerationModeSequentialPreviousPanel,
		PanelPlan: []domain.StoryPanelPlanEntry{
			{PanelIndex: 1, SceneID: "scene_001"},
			{PanelIndex: 2, SceneID: "scene_002"},
		},
		ReferenceBindings: domain.StoryReferenceBindings{
			"previous_panel_reference": {"asset_panel_001_generated"},
		},
		ContinuityPolicy: domain.StoryContinuityPolicy{
			Mode:                         domain.StoryGenerationModeSequentialPreviousPanel,
			RequirePreviousSelectedAsset: true,
			MaxCandidatesPerPanel:        2,
		},
	}
	metadataRaw, err := json.Marshal(map[string]any{
		"story_id":         "pet_story",
		"scene_id":         "scene_002",
		"story_context_v1": story,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	_, _, err = prepareStoryContextV1ForTask(
		domain.Scope{WorkspaceID: "ws_default", ProjectID: "prj_demo", CampaignID: "cmp_demo"},
		metadataRaw,
		domain.CreateTaskRequest{
			RequestedCount: 2,
			SelectionMode:  domain.SelectionAuto,
			ReferenceImages: []domain.ReferenceImage{
				{AssetID: "asset_panel_001_generated", Role: "previous_panel_reference", Source: "story_context_v1"},
			},
		},
		&resolvedTaskInputFiles{
			ReferenceImages: []resolvedTaskInputFile{
				{Kind: domain.InputFileKindReference, Role: "previous_panel_reference", FilePath: "/tmp/panel-001.png", MimeType: "image/png"},
			},
		},
		domain.ReferenceParticipationDiagnostics{ProviderReferenceParticipation: "resolved_input_files"},
		func(assetID string) (storyContinuityAssetSnapshot, error) {
			return storyContinuityAssetSnapshot{
				Scope:        domain.Scope{WorkspaceID: "ws_default", ProjectID: "prj_demo", CampaignID: "cmp_demo"},
				AssetStatus:  domain.AssetDraft,
				MetadataJSON: json.RawMessage(`{"story_id":"pet_story","scene_id":"scene_001","panel_index":1}`),
			}, nil
		},
	)
	if err == nil {
		t.Fatal("expected sequential preflight to fail")
	}
	if !strings.Contains(err.Error(), "previous panel selected asset") && !strings.Contains(err.Error(), "manual_optional") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppendStoryPanelTransitionPromptAddsProgressionHints(t *testing.T) {
	story := &domain.StoryContextV1{
		PanelPlan: []domain.StoryPanelPlanEntry{{
			PanelIndex:           2,
			SceneID:              "scene_002",
			EmotionBefore:        "嘴硬但期待",
			EmotionAfter:         "安心并开始靠近",
			PoseChange:           "从独坐抱书变成抬头看向鸡毛",
			RelationshipShift:    "从等待变成正式同框互动",
			MustChange:           []string{"鸡毛必须进入画面", "小白视线转向鸡毛"},
			MustNotKeep:          []string{"不能继续只有小白单独在画面里"},
			StateTransitionNotes: "必须看得出上一格导致这一格。",
		}},
	}
	metadata := json.RawMessage(`{"scene_id":"scene_002","panel_index":2}`)

	prompt := appendStoryPanelTransitionPrompt("Keep the same pink living room.", story, metadata)
	for _, expected := range []string{
		"State transition requirements:",
		"Emotion before: 嘴硬但期待.",
		"Emotion after: 安心并开始靠近.",
		"Pose change: 从独坐抱书变成抬头看向鸡毛.",
		"Relationship shift: 从等待变成正式同框互动.",
		"Must change: 鸡毛必须进入画面; 小白视线转向鸡毛.",
		"Must not keep: 不能继续只有小白单独在画面里.",
		"State transition notes: 必须看得出上一格导致这一格。.",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected prompt to contain %q, got %q", expected, prompt)
		}
	}
}
