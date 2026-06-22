package store

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestCanReviewAssetTransitionAllowsApproveFromRejected(t *testing.T) {
	if !canReviewAssetTransition(domain.AssetRejected, "approve") {
		t.Fatal("expected rejected asset to be selectable again")
	}
	if !canReviewAssetTransition(domain.AssetApproved, "reject") {
		t.Fatal("expected selected asset to remain rejectable")
	}
	if canReviewAssetTransition(domain.AssetPublished, "approve") {
		t.Fatal("published asset should not transition back to approve through review flow")
	}
}

func TestVisualContextFromProjectMetadataPreservesExistingMetadataShape(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"quality_profile": map[string]any{
			"style_preset": "storybook",
		},
		"provider_profile": map[string]any{
			"enabled":  true,
			"provider": "mock",
		},
		"visual_context": map[string]any{
			"characters": []map[string]any{
				{"id": "dog_milo", "name": "Milo", "status": "active", "primary_asset_id": "asset_milo_primary"},
			},
			"prompt_recipes": []map[string]any{
				{"id": "pet_story", "name": "Pet story", "status": "active"},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	visualContext, err := visualContextFromProjectMetadata(raw)
	if err != nil {
		t.Fatalf("visualContextFromProjectMetadata returned error: %v", err)
	}
	if len(visualContext.Characters) != 1 || visualContext.Characters[0].ID != "dog_milo" {
		t.Fatalf("visual context characters were not parsed: %#v", visualContext)
	}
	if len(visualContext.PromptRecipes) != 1 || visualContext.PromptRecipes[0].ID != "pet_story" {
		t.Fatalf("visual context recipes were not parsed: %#v", visualContext)
	}
}

func TestBuildListAssetsByCampaignQueryAddsFiltersAndDefaultLimit(t *testing.T) {
	from := time.Date(2026, 6, 19, 1, 2, 3, 0, time.UTC)
	sqlText, args := buildListAssetsByCampaignQuery(domain.AssetListQuery{
		ProjectID:   "prj_demo",
		CampaignID:  "cmp_demo",
		Status:      domain.AssetApproved,
		Provider:    "mock",
		Model:       "mock-image",
		Source:      "mcp",
		SessionID:   "session_1",
		BatchID:     "batch_1",
		Keyword:     "hero",
		CreatedFrom: &from,
		Limit:       500,
		Offset:      10,
	})

	for _, fragment := range []string{
		"a.project_id = $1",
		"a.campaign_id = $2",
		"a.status = $3",
		"v.provider = $4",
		"v.model = $5",
		"t.structured_input_json->'metadata_json'->>'source' = $6",
		"t.structured_input_json->'metadata_json'->>'session_id' = $7",
		"t.structured_input_json->'metadata_json'->>'batch_id' = $8",
		"LOWER(v.prompt) LIKE $9",
		"a.created_at >= $10",
		"ORDER BY a.created_at DESC, a.id DESC",
		"LIMIT $11 OFFSET $12",
	} {
		if !strings.Contains(sqlText, fragment) {
			t.Fatalf("expected query to contain %q:\n%s", fragment, sqlText)
		}
	}
	if got := args[len(args)-2]; got != domain.MaxAssetListLimit {
		t.Fatalf("expected limit capped to %d, got %v", domain.MaxAssetListLimit, got)
	}
	if got := args[len(args)-1]; got != 10 {
		t.Fatalf("expected offset=10, got %v", got)
	}
}

func TestBuildListRecentAssetsQueryDoesNotRequireScopeAndKeepsFilters(t *testing.T) {
	sqlText, args := buildListRecentAssetsQuery(domain.AssetListQuery{
		Status:    domain.AssetDraft,
		Provider:  "mock",
		Source:    "web",
		SessionID: "session_recent",
		Keyword:   "night",
		Limit:     24,
		Offset:    48,
	})

	for _, fragment := range []string{
		"a.status = $1",
		"v.provider = $2",
		"t.structured_input_json->'metadata_json'->>'source' = $3",
		"t.structured_input_json->'metadata_json'->>'session_id' = $4",
		"LOWER(v.prompt) LIKE $5",
		"ORDER BY a.created_at DESC, a.id DESC",
		"LIMIT $6 OFFSET $7",
	} {
		if !strings.Contains(sqlText, fragment) {
			t.Fatalf("expected recent query to contain %q:\n%s", fragment, sqlText)
		}
	}
	for _, forbidden := range []string{
		"a.project_id = $1",
		"a.campaign_id = $2",
	} {
		if strings.Contains(sqlText, forbidden) {
			t.Fatalf("recent query should not require current scope condition %q:\n%s", forbidden, sqlText)
		}
	}
	if got := args[len(args)-2]; got != 24 {
		t.Fatalf("expected limit=24, got %v", got)
	}
	if got := args[len(args)-1]; got != 48 {
		t.Fatalf("expected offset=48, got %v", got)
	}
}
