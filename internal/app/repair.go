package app

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

const (
	defaultRepairLimit      = 100
	defaultRepairStaleAfter = 10 * time.Minute
)

type RepairScanOptions struct {
	Limit          int
	StaleAfter     time.Duration
	IncludeOrphans bool
}

type RepairReport struct {
	CheckedAt time.Time          `json:"checked_at"`
	OK        bool               `json:"ok"`
	Summary   RepairSummary      `json:"summary"`
	Issues    []RepairIssue      `json:"issues"`
	Options   RepairScanSettings `json:"options"`
}

type RepairScanSettings struct {
	Limit          int    `json:"limit"`
	StaleAfter     string `json:"stale_after"`
	IncludeOrphans bool   `json:"include_orphans"`
}

type RepairSummary struct {
	IssueCount int            `json:"issue_count"`
	ByKind     map[string]int `json:"by_kind"`
}

type RepairIssue struct {
	Kind       string `json:"kind"`
	Severity   string `json:"severity"`
	TaskID     string `json:"task_id,omitempty"`
	AssetID    string `json:"asset_id,omitempty"`
	VersionID  string `json:"version_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Path       string `json:"path,omitempty"`
	Message    string `json:"message"`
	RepairHint string `json:"repair_hint,omitempty"`
}

type RepairRequeueResult struct {
	TaskID         string `json:"task_id"`
	PreviousStatus string `json:"previous_status"`
	Status         string `json:"status"`
	DryRun         bool   `json:"dry_run"`
	Enqueued       bool   `json:"enqueued"`
	Message        string `json:"message"`
}

type RepairAssetVerifyResult struct {
	AssetID   string            `json:"asset_id"`
	VersionID string            `json:"version_id"`
	Status    string            `json:"status"`
	OK        bool              `json:"ok"`
	Files     []RepairFileCheck `json:"files"`
	Issues    []RepairIssue     `json:"issues"`
}

type RepairFileCheck struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Size   int64  `json:"size"`
	OK     bool   `json:"ok"`
}

func (s *Service) RepairScan(ctx context.Context, opts RepairScanOptions) (RepairReport, error) {
	opts = normalizeRepairScanOptions(opts)
	report := RepairReport{
		CheckedAt: time.Now().UTC(),
		OK:        true,
		Summary: RepairSummary{
			ByKind: map[string]int{},
		},
		Options: RepairScanSettings{
			Limit:          opts.Limit,
			StaleAfter:     opts.StaleAfter.String(),
			IncludeOrphans: opts.IncludeOrphans,
		},
	}

	staleBefore := report.CheckedAt.Add(-opts.StaleAfter)
	tasks, err := s.store.ListRepairTaskCandidates(ctx, staleBefore, opts.Limit)
	if err != nil {
		return RepairReport{}, err
	}
	for _, item := range tasks {
		message := fmt.Sprintf("task is %s and may need requeue", item.Task.Status)
		if item.IssueKind == "stale_running" || item.IssueKind == "stale_queued" {
			message = fmt.Sprintf("task has been %s since %s", item.Task.Status, item.Task.UpdatedAt.Format(time.RFC3339))
		}
		report.addIssue(RepairIssue{
			Kind:       item.IssueKind,
			Severity:   "warning",
			TaskID:     item.Task.ID,
			Status:     item.Task.Status,
			Message:    message,
			RepairHint: "requeue_task",
		})
	}

	invalidAssets, err := s.store.ListInvalidCurrentVersionAssets(ctx, opts.Limit)
	if err != nil {
		return RepairReport{}, err
	}
	for _, item := range invalidAssets {
		report.addIssue(RepairIssue{
			Kind:       "invalid_current_version",
			Severity:   "error",
			AssetID:    item.AssetID,
			VersionID:  item.CurrentVersionID,
			Status:     item.VersionStatus,
			Message:    "asset current_version_id is empty, missing, or not ready",
			RepairHint: "inspect_asset_version",
		})
	}

	assets, err := s.store.ListCurrentAssetVersions(ctx, opts.Limit)
	if err != nil {
		return RepairReport{}, err
	}
	for _, item := range assets {
		for _, check := range repairFileChecks(item.Version.FilePath, item.Version.ThumbnailPath, item.Version.MetadataPath) {
			if check.OK {
				continue
			}
			report.addIssue(fileCheckIssue(item.ID, item.Version.ID, check))
		}
	}

	if opts.IncludeOrphans {
		orphans, err := s.findOrphanFiles(ctx, opts.Limit)
		if err != nil {
			return RepairReport{}, err
		}
		for _, issue := range orphans {
			report.addIssue(issue)
		}
	}

	report.OK = len(report.Issues) == 0
	return report, nil
}

func (s *Service) RepairRequeueTask(ctx context.Context, taskID string, dryRun bool) (RepairRequeueResult, error) {
	task, err := s.store.GetTask(ctx, taskID)
	if err != nil {
		return RepairRequeueResult{}, err
	}
	result := RepairRequeueResult{
		TaskID:         task.ID,
		PreviousStatus: task.Status,
		Status:         task.Status,
		DryRun:         dryRun,
	}
	if !canRepairRequeue(task.Status) {
		return result, fmt.Errorf("task %s is %s and cannot be requeued by repair", task.ID, task.Status)
	}
	if dryRun {
		result.Message = "dry run: task can be requeued"
		return result, nil
	}
	if err := s.store.UpdateTaskStatus(ctx, task.ID, domain.TaskQueued, nil, nil); err != nil {
		return result, err
	}
	if err := s.queue.Enqueue(ctx, task.ID); err != nil {
		_ = s.store.MarkTaskEnqueueFailed(ctx, task.ID, err)
		return result, err
	}
	result.Status = domain.TaskQueued
	result.Enqueued = true
	result.Message = "task requeued"
	return result, nil
}

func (s *Service) RepairVerifyAsset(ctx context.Context, assetID string) (RepairAssetVerifyResult, error) {
	item, err := s.store.GetAssetWithVersion(ctx, assetID)
	if err != nil {
		return RepairAssetVerifyResult{}, err
	}
	result := RepairAssetVerifyResult{
		AssetID:   item.ID,
		VersionID: item.Version.ID,
		Status:    item.Version.Status,
		OK:        true,
		Files:     repairFileChecks(item.Version.FilePath, item.Version.ThumbnailPath, item.Version.MetadataPath),
	}
	if item.Version.Status != domain.VersionReady {
		result.Issues = append(result.Issues, RepairIssue{
			Kind:       "version_not_ready",
			Severity:   "error",
			AssetID:    item.ID,
			VersionID:  item.Version.ID,
			Status:     item.Version.Status,
			Message:    "asset current version is not ready",
			RepairHint: "inspect_asset_version",
		})
	}
	for _, check := range result.Files {
		if check.OK {
			continue
		}
		result.Issues = append(result.Issues, fileCheckIssue(item.ID, item.Version.ID, check))
	}
	result.OK = len(result.Issues) == 0
	return result, nil
}

func normalizeRepairScanOptions(opts RepairScanOptions) RepairScanOptions {
	if opts.Limit < 1 {
		opts.Limit = defaultRepairLimit
	}
	if opts.StaleAfter <= 0 {
		opts.StaleAfter = defaultRepairStaleAfter
	}
	return opts
}

func (r *RepairReport) addIssue(issue RepairIssue) {
	r.Issues = append(r.Issues, issue)
	r.Summary.IssueCount++
	r.Summary.ByKind[issue.Kind]++
}

func canRepairRequeue(status string) bool {
	switch status {
	case domain.TaskEnqueueFailed, domain.TaskQueued, domain.TaskRunning:
		return true
	default:
		return false
	}
}

func repairFileChecks(originalPath, thumbnailPath, metadataPath string) []RepairFileCheck {
	return []RepairFileCheck{
		checkRepairFile("original", originalPath),
		checkRepairFile("thumbnail", thumbnailPath),
		checkRepairFile("metadata", metadataPath),
	}
}

func checkRepairFile(kind, path string) RepairFileCheck {
	check := RepairFileCheck{Kind: kind, Path: path}
	info, err := os.Stat(path)
	if err != nil {
		return check
	}
	check.Exists = true
	check.Size = info.Size()
	check.OK = !info.IsDir() && info.Size() > 0
	return check
}

func fileCheckIssue(assetID, versionID string, check RepairFileCheck) RepairIssue {
	kind := "missing_file"
	message := fmt.Sprintf("%s file is missing", check.Kind)
	if check.Exists {
		kind = "empty_file"
		message = fmt.Sprintf("%s file is empty or not a regular file", check.Kind)
	}
	return RepairIssue{
		Kind:       kind,
		Severity:   "error",
		AssetID:    assetID,
		VersionID:  versionID,
		Path:       check.Path,
		Message:    message,
		RepairHint: "regenerate_or_restore_file",
	}
}

func (s *Service) findOrphanFiles(ctx context.Context, limit int) ([]RepairIssue, error) {
	root := s.storage.Root()
	if strings.TrimSpace(root) == "" {
		return nil, nil
	}
	known, err := s.store.ListKnownAssetFilePaths(ctx, limit)
	if err != nil {
		return nil, err
	}
	knownSet := map[string]bool{}
	for _, path := range known {
		knownSet[filepath.Clean(path)] = true
	}

	issues := []RepairIssue{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if len(issues) >= limit {
			return fs.SkipAll
		}
		if entry.IsDir() {
			if entry.Name() == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		clean := filepath.Clean(path)
		if !isFinalAssetFilePath(clean) || knownSet[clean] {
			return nil
		}
		issues = append(issues, RepairIssue{
			Kind:       "orphan_file",
			Severity:   "warning",
			Path:       clean,
			Message:    "file exists in final asset storage but no asset_version references it",
			RepairHint: "inspect_or_remove_file",
		})
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return issues, nil
}

func isFinalAssetFilePath(path string) bool {
	separator := string(filepath.Separator)
	return strings.Contains(path, separator+"originals"+separator) ||
		strings.Contains(path, separator+"thumbnails"+separator) ||
		strings.Contains(path, separator+"metadata"+separator)
}
