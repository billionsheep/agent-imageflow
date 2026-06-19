package app

import (
	"context"
	"fmt"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func (s *Service) GetStorageIntegrity(ctx context.Context, scope domain.Scope) (domain.StorageIntegrityResponse, error) {
	if err := s.store.CheckScope(ctx, scope); err != nil {
		return domain.StorageIntegrityResponse{}, err
	}
	checkedAt := time.Now().UTC()
	report := domain.StorageIntegrityResponse{
		CheckedAt: checkedAt,
		Scope:     scope,
		OK:        true,
		Summary: domain.StorageIntegritySummary{
			ByKind: map[string]int{},
		},
	}

	staleBefore := checkedAt.Add(-defaultRepairStaleAfter)
	tasks, err := s.store.ListRepairTaskCandidatesByScope(ctx, scope, staleBefore, defaultRepairLimit)
	if err != nil {
		return domain.StorageIntegrityResponse{}, err
	}
	for _, item := range tasks {
		message := fmt.Sprintf("task is %s and may need requeue", item.Task.Status)
		if item.IssueKind == "stale_running" || item.IssueKind == "stale_queued" {
			message = fmt.Sprintf("task has been %s since %s", item.Task.Status, item.Task.UpdatedAt.Format(time.RFC3339))
		}
		addStorageIntegrityIssue(&report, domain.StorageIntegrityIssue{
			Kind:       item.IssueKind,
			Severity:   "warning",
			TaskID:     item.Task.ID,
			Status:     item.Task.Status,
			Message:    message,
			RepairHint: "repair_scan_or_requeue_task",
		})
	}

	invalidAssets, err := s.store.ListInvalidCurrentVersionAssetsByScope(ctx, scope, defaultRepairLimit)
	if err != nil {
		return domain.StorageIntegrityResponse{}, err
	}
	for _, item := range invalidAssets {
		addStorageIntegrityIssue(&report, domain.StorageIntegrityIssue{
			Kind:       "invalid_current_version",
			Severity:   "error",
			AssetID:    item.AssetID,
			VersionID:  item.CurrentVersionID,
			Status:     item.VersionStatus,
			Message:    "asset current_version_id is empty, missing, or not ready",
			RepairHint: "inspect_asset_version",
		})
	}

	assets, err := s.store.ListCurrentAssetVersionsByScope(ctx, scope, defaultRepairLimit)
	if err != nil {
		return domain.StorageIntegrityResponse{}, err
	}
	for _, item := range assets {
		for _, check := range repairFileChecks(item.Version.FilePath, item.Version.ThumbnailPath, item.Version.MetadataPath) {
			if check.OK {
				continue
			}
			addStorageIntegrityIssue(&report, storageIntegrityFileIssue(item.ID, item.Version.ID, check))
		}
	}

	report.OK = len(report.Issues) == 0
	return report, nil
}

func addStorageIntegrityIssue(report *domain.StorageIntegrityResponse, issue domain.StorageIntegrityIssue) {
	report.Issues = append(report.Issues, issue)
	report.Summary.IssueCount++
	report.Summary.ByKind[issue.Kind]++
}

func storageIntegrityFileIssue(assetID, versionID string, check RepairFileCheck) domain.StorageIntegrityIssue {
	kind := "missing_file"
	message := fmt.Sprintf("%s file is missing", check.Kind)
	if check.Exists {
		kind = "empty_file"
		message = fmt.Sprintf("%s file is empty or not a regular file", check.Kind)
	}
	return domain.StorageIntegrityIssue{
		Kind:       kind,
		Severity:   "error",
		AssetID:    assetID,
		VersionID:  versionID,
		FileKind:   check.Kind,
		Message:    message,
		RepairHint: "regenerate_or_restore_file",
	}
}
