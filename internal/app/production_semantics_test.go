package app

import (
	"encoding/json"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestEnrichTaskRuntimeSemanticsMarksPartialSuccessFields(t *testing.T) {
	task := domain.Task{
		ID:             "task_partial",
		Status:         domain.TaskPartiallyCompleted,
		RequestedCount: 2,
	}
	message := "1 of 2 requested images are ready: openai-compatible provider failed: upstream timeout"
	task.ErrorMessage = &message

	enrichTaskRuntimeSemantics(&task, 1)

	if task.DeliveredCount != 1 {
		t.Fatalf("expected delivered_count=1, got %#v", task)
	}
	if task.PartialSuccessReason != "delivered_count_below_requested" {
		t.Fatalf("expected partial_success_reason=delivered_count_below_requested, got %#v", task)
	}
	if task.ProviderErrorSummary != "openai-compatible provider failed: upstream timeout" {
		t.Fatalf("expected provider_error_summary to keep only provider details, got %#v", task)
	}
}

func TestBuildAssetSummaryUsesStoryAndCaptionLineageSemantics(t *testing.T) {
	structured, err := json.Marshal(map[string]any{
		"provider_reference_participation": "resolved_input_files",
		"caption_lineage": map[string]any{
			"derived_from_asset_id":   "asset_base",
			"derivation_type":         "caption_edit",
			"caption_text":            "因为喜欢你",
			"auto_select_derivative":  true,
			"speaker_character_id":    "dog_xiaobai",
			"bubble_anchor":           "top_right",
			"avoid_covering_subjects": true,
		},
		"metadata_json": map[string]any{
			"story_id":                         "pet_story",
			"scene_id":                         "scene_002",
			"panel_index":                      2,
			"dialogue":                         "因为喜欢你",
			"previous_panel_asset_id":          "asset_panel_001",
			"provider_reference_participation": "resolved_input_files",
		},
	})
	if err != nil {
		t.Fatalf("marshal structured input: %v", err)
	}

	summary := buildAssetSummary(domain.AssetWithVersion{
		Asset: domain.Asset{
			ID:     "asset_caption",
			Status: domain.AssetApproved,
		},
		Version: domain.AssetVersion{
			Provider: "mock",
			Model:    "mock-image",
		},
		TaskStructuredInputJSON: structured,
	}, domain.DeliveryRoleFinalDelivery)
	if summary == nil {
		t.Fatal("expected asset_summary to be built")
	}
	if summary.StoryID != "pet_story" ||
		summary.SceneID != "scene_002" ||
		summary.PanelIndex != 2 ||
		summary.Dialogue != "因为喜欢你" ||
		summary.CaptionText != "因为喜欢你" ||
		summary.DerivedFromAssetID != "asset_base" ||
		summary.DerivationType != "caption_edit" ||
		summary.PreviousPanelAssetID != "asset_panel_001" ||
		summary.ProviderReferenceParticipation != "resolved_input_files" ||
		summary.Provider != "mock" ||
		summary.Model != "mock-image" ||
		summary.AssetStatus != domain.AssetApproved ||
		summary.DeliveryRole != domain.DeliveryRoleFinalDelivery {
		t.Fatalf("unexpected asset_summary: %#v", summary)
	}
}

func TestEnrichBatchStorySummaryDeliverySemanticsMarksTaskRuntimeFields(t *testing.T) {
	summary := enrichBatchStorySummaryDeliverySemantics(domain.BatchStorySummaryResponse{
		Scenes: []domain.BatchStorySummaryScene{{
			StoryID: "pet_story",
			SceneID: "scene_002",
			Tasks: []domain.BatchStorySummaryTask{{
				TaskID:         "task_partial",
				Status:         domain.TaskPartiallyCompleted,
				RequestedCount: 2,
				AssetCount:     1,
				ErrorMessage: func() *string {
					message := "1 of 2 requested images are ready: openai-compatible provider failed: upstream timeout"
					return &message
				}(),
			}},
		}},
	})

	task := summary.Scenes[0].Tasks[0]
	if task.DeliveredCount != 1 {
		t.Fatalf("expected delivered_count=1, got %#v", task)
	}
	if task.PartialSuccessReason != "delivered_count_below_requested" {
		t.Fatalf("expected partial_success_reason=delivered_count_below_requested, got %#v", task)
	}
	if task.ProviderErrorSummary != "openai-compatible provider failed: upstream timeout" {
		t.Fatalf("expected provider_error_summary to keep provider detail, got %#v", task)
	}
}
