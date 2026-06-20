package store

import (
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
