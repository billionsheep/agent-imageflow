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

func TestBuildBatchStorySummaryTasksQueryAddsContractFilters(t *testing.T) {
	built := buildBatchStorySummaryTasksQuery(domain.BatchStorySummaryQuery{
		ProjectID:  "prj_demo",
		CampaignID: "cmp_demo",
		SessionID:  "session_1",
		BatchID:    "batch_1",
		StoryID:    "story_1",
		Source:     "codex",
		Status:     domain.TaskCompleted,
		Limit:      999,
	})

	for _, fragment := range []string{
		"gt.project_id = $1",
		"gt.campaign_id = $2",
		"gt.structured_input_json->'metadata_json'->>'session_id' = $3",
		"gt.structured_input_json->'metadata_json'->>'batch_id' = $4",
		"gt.structured_input_json->'metadata_json'->>'story_id' = $5",
		"gt.structured_input_json->'metadata_json'->>'source' = $6",
		"gt.status = $7",
		"NOT",
		"task_role",
		"exclude_from_story_summary",
		"WITH limited_tasks AS",
		"LIMIT $8",
		"LEFT JOIN asset a ON a.task_id = gt.id",
		"LEFT JOIN asset_version v ON v.id = a.current_version_id",
	} {
		if !strings.Contains(built.SQL, fragment) {
			t.Fatalf("expected batch story summary query to contain %q:\n%s", fragment, built.SQL)
		}
	}
	if got := built.Args[len(built.Args)-1]; got != 999 {
		t.Fatalf("expected raw limit arg to be preserved for store normalization, got %v", got)
	}
}

func TestBuildBatchStorySummaryTasksQueryCanIncludeSetupRoles(t *testing.T) {
	built := buildBatchStorySummaryTasksQuery(domain.BatchStorySummaryQuery{
		ProjectID:    "prj_demo",
		CampaignID:   "cmp_demo",
		SessionID:    "session_1",
		IncludeSetup: true,
		Limit:        10,
	})
	if strings.Contains(built.SQL, "NOT") && strings.Contains(built.SQL, "task_role") {
		t.Fatalf("include_setup=true should not add setup exclusion to task query:\n%s", built.SQL)
	}

	excluded := buildBatchStorySummaryExcludedCountQuery(domain.BatchStorySummaryQuery{
		ProjectID:  "prj_demo",
		CampaignID: "cmp_demo",
		SessionID:  "session_1",
	})
	if !strings.Contains(excluded.SQL, "task_role") || !strings.Contains(excluded.SQL, "scene_id") {
		t.Fatalf("excluded count query should count setup-like tasks:\n%s", excluded.SQL)
	}
}

func TestBuildResolveLatestSceneTaskQueryAddsSceneIdentityFilters(t *testing.T) {
	built := buildResolveLatestSceneTaskQuery(domain.SceneIdentity{
		SessionID: "session_1",
		BatchID:   "batch_1",
		StoryID:   "story_1",
		SceneID:   "scene_002",
		Source:    "codex",
	}, "prj_demo", "cmp_demo")

	for _, fragment := range []string{
		"project_id = $1",
		"campaign_id = $2",
		"structured_input_json->'metadata_json'->>'session_id' = $3",
		"structured_input_json->'metadata_json'->>'batch_id' = $4",
		"structured_input_json->'metadata_json'->>'story_id' = $5",
		"structured_input_json->'metadata_json'->>'scene_id' = $6",
		"structured_input_json->'metadata_json'->>'source' = $7",
		"ORDER BY created_at DESC, id DESC",
		"LIMIT 1",
	} {
		if !strings.Contains(built.SQL, fragment) {
			t.Fatalf("expected latest scene task query to contain %q:\n%s", fragment, built.SQL)
		}
	}
	if len(built.Args) != 7 || built.Args[6] != "codex" {
		t.Fatalf("unexpected latest query args: %#v", built.Args)
	}
}

func TestBuildCountSceneRegenerationsQueryCountsMetadataLineage(t *testing.T) {
	built := buildCountSceneRegenerationsQuery(domain.SceneIdentity{
		SessionID: "session_1",
		BatchID:   "batch_1",
		StoryID:   "story_1",
		SceneID:   "scene_002",
	}, "prj_demo", "cmp_demo")

	for _, fragment := range []string{
		"project_id = $1",
		"campaign_id = $2",
		"structured_input_json->'metadata_json'->>'session_id' = $3",
		"structured_input_json->'metadata_json'->>'batch_id' = $4",
		"structured_input_json->'metadata_json'->>'story_id' = $5",
		"structured_input_json->'metadata_json'->>'scene_id' = $6",
		"COALESCE(structured_input_json->'metadata_json'->>'regenerated_from_task_id', '') <> ''",
		"COUNT(*)",
	} {
		if !strings.Contains(built.SQL, fragment) {
			t.Fatalf("expected regeneration count query to contain %q:\n%s", fragment, built.SQL)
		}
	}
	if len(built.Args) != 6 {
		t.Fatalf("unexpected count query args: %#v", built.Args)
	}
}

func TestScopedCascadeDeleteStatementsOrderDependentTables(t *testing.T) {
	tests := []struct {
		name       string
		statements []scopedDeleteStatement
		wantOrder  []string
	}{
		{
			name:       "campaign",
			statements: campaignCascadeDeleteStatements("ws_1", "prj_1", "cmp_1"),
			wantOrder: []string{
				"DELETE FROM delivery_event",
				"DELETE FROM review_event",
				"DELETE FROM asset_version",
				"DELETE FROM asset",
				"DELETE FROM task_attempt",
				"DELETE FROM generation_task",
				"DELETE FROM campaign",
			},
		},
		{
			name:       "project",
			statements: projectCascadeDeleteStatements("ws_1", "prj_1"),
			wantOrder: []string{
				"DELETE FROM delivery_event",
				"DELETE FROM review_event",
				"DELETE FROM asset_version",
				"DELETE FROM asset",
				"DELETE FROM task_attempt",
				"DELETE FROM generation_task",
				"DELETE FROM campaign",
				"DELETE FROM project",
			},
		},
		{
			name:       "workspace",
			statements: workspaceCascadeDeleteStatements("ws_1"),
			wantOrder: []string{
				"DELETE FROM delivery_event",
				"DELETE FROM review_event",
				"DELETE FROM asset_version",
				"DELETE FROM asset",
				"DELETE FROM task_attempt",
				"DELETE FROM generation_task",
				"DELETE FROM campaign",
				"DELETE FROM project",
				"DELETE FROM workspace",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.statements) != len(tt.wantOrder) {
				t.Fatalf("expected %d statements, got %d", len(tt.wantOrder), len(tt.statements))
			}
			for index, fragment := range tt.wantOrder {
				if !strings.HasPrefix(tt.statements[index].SQL, fragment) {
					t.Fatalf("expected statement %d to start with %q, got:\n%s", index, fragment, tt.statements[index].SQL)
				}
			}
		})
	}
}
