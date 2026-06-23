package app

import (
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestCleanupDryRunReasonForAssetStatusProtectsSelectedAndPublished(t *testing.T) {
	tests := []struct {
		status string
		wantOK bool
	}{
		{status: domain.AssetRejected, wantOK: true},
		{status: domain.AssetDraft, wantOK: true},
		{status: domain.AssetApproved, wantOK: false},
		{status: domain.AssetPublished, wantOK: false},
		{status: domain.AssetDeprecated, wantOK: true},
	}

	for _, tc := range tests {
		t.Run(tc.status, func(t *testing.T) {
			_, ok := cleanupDryRunReasonForAssetStatus(tc.status)
			if ok != tc.wantOK {
				t.Fatalf("cleanupDryRunReasonForAssetStatus(%q) ok=%v, want %v", tc.status, ok, tc.wantOK)
			}
		})
	}
}

func TestCleanupDryRunOptionsFromExecuteOptionsCarriesTargetFilters(t *testing.T) {
	got := cleanupDryRunOptionsFromExecuteOptions(domain.CleanupExecuteOptions{
		Scope:     domain.Scope{WorkspaceID: "ws_1", ProjectID: "prj_1", CampaignID: "cmp_1"},
		AssetID:   "asset_1",
		TaskID:    "task_1",
		SessionID: "session_1",
		BatchID:   "batch_1",
		StoryID:   "story_1",
		Limit:     25,
	})

	if got.AssetID != "asset_1" || got.TaskID != "task_1" || got.SessionID != "session_1" || got.BatchID != "batch_1" || got.StoryID != "story_1" {
		t.Fatalf("target filters were not preserved: %#v", got)
	}
}

func TestNormalizeCleanupDryRunOptionsDefaultsToAllReadOnlyCandidateClasses(t *testing.T) {
	opts := normalizeCleanupDryRunOptions(domain.CleanupDryRunOptions{})
	if !opts.IncludeRejected || !opts.IncludeGenerated || !opts.IncludeFailedTaskTmp || !opts.IncludeOrphans {
		t.Fatalf("expected all candidate classes to default on, got %#v", opts)
	}
	if opts.Limit != defaultCleanupDryRunLimit {
		t.Fatalf("expected default limit %d, got %d", defaultCleanupDryRunLimit, opts.Limit)
	}
}

func TestCleanupExecutionConfirmationRequiresExecute(t *testing.T) {
	err := validateCleanupExecutionConfirmation(domain.CleanupExecuteOptions{
		DryRunToken: "cleanup_token",
	}, "cleanup_token")
	if err == nil {
		t.Fatal("expected missing --execute to be rejected")
	}
}

func TestCleanupExecutionConfirmationAcceptsMatchingToken(t *testing.T) {
	err := validateCleanupExecutionConfirmation(domain.CleanupExecuteOptions{
		Execute:     true,
		DryRunToken: "cleanup_token",
	}, "cleanup_token")
	if err != nil {
		t.Fatalf("expected matching token to be accepted: %v", err)
	}
}

func TestCleanupExecutionConfirmationRejectsTokenMismatch(t *testing.T) {
	err := validateCleanupExecutionConfirmation(domain.CleanupExecuteOptions{
		Execute:     true,
		DryRunToken: "cleanup_other",
		Confirm:     true,
	}, "cleanup_token")
	if err == nil {
		t.Fatal("expected token mismatch to be rejected even with --confirm")
	}
}

func TestCleanupDryRunTokenIgnoresGeneratedAt(t *testing.T) {
	report := domain.CleanupDryRunReport{
		Scope: domain.Scope{WorkspaceID: "ws", ProjectID: "prj", CampaignID: "cmp"},
		Candidates: []domain.CleanupCandidate{
			{
				Kind:      "asset",
				Reason:    "rejected_asset",
				AssetID:   "asset_1",
				TaskID:    "task_1",
				Status:    domain.AssetRejected,
				FileCount: 1,
				Bytes:     11,
				Files: []domain.CleanupCandidateFile{
					{Kind: "original", StorageKey: "workspaces/ws/projects/prj/campaigns/cmp/originals/asset_1/1.png", Bytes: 11},
				},
			},
		},
	}
	first := cleanupDryRunToken(report)
	report.GeneratedAt = report.GeneratedAt.AddDate(0, 0, 1)
	second := cleanupDryRunToken(report)
	if first == "" || first != second {
		t.Fatalf("expected stable token, got %q and %q", first, second)
	}
}
