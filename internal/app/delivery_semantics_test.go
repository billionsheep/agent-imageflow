package app

import (
	"encoding/json"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestCaptionDerivativeAutomaticSelectionDecisionSelectsFirstDerivativeWhenRequested(t *testing.T) {
	structured, err := json.Marshal(map[string]any{
		"caption_lineage": map[string]any{
			"derived_from_asset_id":   "asset_base",
			"derivation_type":         "caption_edit",
			"auto_select_derivative":  true,
			"speaker_character_id":    "dog_xiaobai",
			"bubble_anchor":           "top_right",
			"avoid_covering_subjects": true,
		},
	})
	if err != nil {
		t.Fatalf("marshal structured input: %v", err)
	}

	decision := captionDerivativeAutomaticSelectionDecision(domain.Task{
		ID:                  "task_caption",
		SelectionMode:       domain.SelectionManualOptional,
		StructuredInputJSON: structured,
	}, []domain.AssetWithVersion{{
		Asset: domain.Asset{ID: "asset_caption_1"},
	}})
	if decision == nil {
		t.Fatal("expected caption derivative automatic selection decision")
	}
	if decision.AssetID != "asset_caption_1" {
		t.Fatalf("expected asset_caption_1 to be auto-selected, got %#v", decision)
	}
}

func TestCaptionDerivativeAutomaticSelectionDecisionSkipsWhenManualConfirmationIsPreferred(t *testing.T) {
	structured, err := json.Marshal(map[string]any{
		"caption_lineage": map[string]any{
			"derived_from_asset_id":  "asset_base",
			"derivation_type":        "caption_edit",
			"auto_select_derivative": false,
		},
	})
	if err != nil {
		t.Fatalf("marshal structured input: %v", err)
	}

	decision := captionDerivativeAutomaticSelectionDecision(domain.Task{
		ID:                  "task_caption",
		SelectionMode:       domain.SelectionManualOptional,
		StructuredInputJSON: structured,
	}, []domain.AssetWithVersion{{
		Asset: domain.Asset{ID: "asset_caption_1"},
	}})
	if decision != nil {
		t.Fatalf("expected no automatic selection when manual confirmation is preferred, got %#v", decision)
	}
}

func TestCaptionDerivativeAutomaticSelectionDecisionSkipsWhenTaskAlreadyUsesAutoSelection(t *testing.T) {
	structured, err := json.Marshal(map[string]any{
		"caption_lineage": map[string]any{
			"derived_from_asset_id":  "asset_base",
			"derivation_type":        "caption_edit",
			"auto_select_derivative": true,
		},
	})
	if err != nil {
		t.Fatalf("marshal structured input: %v", err)
	}

	decision := captionDerivativeAutomaticSelectionDecision(domain.Task{
		ID:                  "task_caption",
		SelectionMode:       domain.SelectionAuto,
		StructuredInputJSON: structured,
	}, []domain.AssetWithVersion{{
		Asset: domain.Asset{ID: "asset_caption_1"},
	}})
	if decision != nil {
		t.Fatalf("selection_mode=auto should stay on the existing auto-select path, got %#v", decision)
	}
}
