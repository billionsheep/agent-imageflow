package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/storage"
	"github.com/billionsheep/agent-imageflow/internal/store"
)

const defaultCleanupDryRunLimit = 100

func (s *Service) GetStorageGovernance(ctx context.Context, scope domain.Scope) (domain.StorageGovernanceResponse, error) {
	if err := s.store.CheckScope(ctx, scope); err != nil {
		return domain.StorageGovernanceResponse{}, err
	}
	knownPaths, err := s.store.ListKnownAssetFilePaths(ctx, 0)
	if err != nil {
		return domain.StorageGovernanceResponse{}, err
	}
	usage, err := s.storage.ScanUsage(ctx, storage.UsageScanOptions{
		Scope:               scope,
		KnownAssetFilePaths: knownPaths,
	})
	if err != nil {
		return domain.StorageGovernanceResponse{}, err
	}
	counts, err := s.store.GetStorageGovernanceCounts(ctx, scope)
	if err != nil {
		return domain.StorageGovernanceResponse{}, err
	}
	return domain.StorageGovernanceResponse{
		GeneratedAt: time.Now().UTC(),
		Scope:       scope,
		Usage:       usage,
		Counts:      counts,
	}, nil
}

func (s *Service) CleanupDryRun(ctx context.Context, opts domain.CleanupDryRunOptions) (domain.CleanupDryRunReport, error) {
	if err := s.store.CheckScope(ctx, opts.Scope); err != nil {
		return domain.CleanupDryRunReport{}, err
	}
	opts = normalizeCleanupDryRunOptions(opts)

	report := domain.CleanupDryRunReport{
		GeneratedAt: time.Now().UTC(),
		DryRun:      true,
		Scope:       opts.Scope,
		Summary: domain.CleanupDryRunSummary{
			ByReason: map[string]int{},
		},
	}

	counts, err := s.store.GetStorageGovernanceCounts(ctx, opts.Scope)
	if err != nil {
		return domain.CleanupDryRunReport{}, err
	}
	report.Protected = domain.CleanupProtectedStats{
		SelectedAssetCount:  counts.Campaign.SelectedAssetCount,
		PublishedAssetCount: counts.Campaign.PublishedAssetCount,
	}

	assetCandidates, err := s.store.ListCleanupAssetCandidates(ctx, opts.Scope, opts.IncludeRejected, opts.IncludeGenerated, opts.Limit)
	if err != nil {
		return domain.CleanupDryRunReport{}, err
	}
	for _, item := range assetCandidates {
		reason, ok := cleanupDryRunReasonForAssetStatus(item.Status)
		if !ok {
			continue
		}
		addCleanupCandidate(&report, s.assetCleanupCandidate(item, reason))
	}

	if opts.IncludeFailedTaskTmp || opts.IncludeOrphans {
		knownPaths, err := s.store.ListKnownAssetFilePaths(ctx, 0)
		if err != nil {
			return domain.CleanupDryRunReport{}, err
		}
		fileCandidates, err := s.storage.ListCleanupFileCandidates(ctx, storage.CleanupFileCandidateOptions{
			Scope:               opts.Scope,
			KnownAssetFilePaths: knownPaths,
			IncludeTmp:          opts.IncludeFailedTaskTmp,
			IncludeOrphans:      opts.IncludeOrphans,
			Limit:               opts.Limit,
		})
		if err != nil {
			return domain.CleanupDryRunReport{}, err
		}
		for _, candidate := range fileCandidates {
			addCleanupCandidate(&report, candidate)
		}
	}

	report.DryRunToken = cleanupDryRunToken(report)
	return report, nil
}

func (s *Service) CleanupExecute(ctx context.Context, opts domain.CleanupExecuteOptions) (domain.CleanupExecutionReport, error) {
	dryRun, err := s.CleanupDryRun(ctx, cleanupDryRunOptionsFromExecuteOptions(opts))
	if err != nil {
		return domain.CleanupExecutionReport{}, err
	}
	report := domain.CleanupExecutionReport{
		GeneratedAt: time.Now().UTC(),
		DryRun:      false,
		Scope:       dryRun.Scope,
		DryRunToken: dryRun.DryRunToken,
		Protected:   dryRun.Protected,
		Summary: domain.CleanupExecutionSummary{
			CandidateCount: len(dryRun.Candidates),
			FileCount:      dryRun.Summary.FileCount,
			Bytes:          dryRun.Summary.Bytes,
			ByReason:       copyCleanupReasonCounts(dryRun.Summary.ByReason),
		},
	}

	if err := validateCleanupExecutionConfirmation(opts, dryRun.DryRunToken); err != nil {
		report.AuditEventID = s.appendCleanupExecutionAudit(ctx, report.Scope, opts.Actor, false, err.Error())
		return report, err
	}

	report.Executed = true
	var failed []string
	for _, candidate := range dryRun.Candidates {
		result := s.executeCleanupCandidate(ctx, dryRun.Scope, candidate)
		report.Results = append(report.Results, result)
		switch result.Action {
		case "deleted":
			report.Summary.DeletedCandidateCount++
		case "skipped":
			report.Summary.SkippedCandidateCount++
		default:
			report.Summary.FailedCandidateCount++
			if result.Error != "" {
				failed = append(failed, result.Error)
			}
		}
		for _, file := range result.Files {
			if file.Action == "deleted" {
				report.Summary.DeletedFileCount++
				report.Summary.DeletedBytes += file.Bytes
			} else if file.Action == "failed" && file.Error != "" {
				failed = append(failed, file.Error)
			}
		}
	}

	if len(failed) > 0 {
		err := fmt.Errorf("cleanup execution completed with %d failed operation(s): %s", len(failed), strings.Join(failed, "; "))
		report.AuditEventID = s.appendCleanupExecutionAudit(ctx, report.Scope, opts.Actor, false, err.Error())
		return report, err
	}
	report.AuditEventID = s.appendCleanupExecutionAudit(ctx, report.Scope, opts.Actor, true, "")
	return report, nil
}

func normalizeCleanupDryRunOptions(opts domain.CleanupDryRunOptions) domain.CleanupDryRunOptions {
	if opts.Limit < 1 {
		opts.Limit = defaultCleanupDryRunLimit
	}
	if !opts.IncludeRejected && !opts.IncludeGenerated && !opts.IncludeFailedTaskTmp && !opts.IncludeOrphans {
		opts.IncludeRejected = true
		opts.IncludeGenerated = true
		opts.IncludeFailedTaskTmp = true
		opts.IncludeOrphans = true
	}
	return opts
}

func cleanupDryRunOptionsFromExecuteOptions(opts domain.CleanupExecuteOptions) domain.CleanupDryRunOptions {
	return normalizeCleanupDryRunOptions(domain.CleanupDryRunOptions{
		Scope:                opts.Scope,
		IncludeRejected:      opts.IncludeRejected,
		IncludeGenerated:     opts.IncludeGenerated,
		IncludeFailedTaskTmp: opts.IncludeFailedTaskTmp,
		IncludeOrphans:       opts.IncludeOrphans,
		Limit:                opts.Limit,
	})
}

func validateCleanupExecutionConfirmation(opts domain.CleanupExecuteOptions, expectedToken string) error {
	if !opts.Execute {
		return fmt.Errorf("cleanup execution requires --execute")
	}
	token := strings.TrimSpace(opts.DryRunToken)
	if token != "" {
		if token != expectedToken {
			return fmt.Errorf("cleanup dry-run token mismatch")
		}
		return nil
	}
	if opts.Confirm {
		return nil
	}
	return fmt.Errorf("cleanup execution requires a matching dry-run token or --confirm")
}

func cleanupDryRunReasonForAssetStatus(status string) (string, bool) {
	switch status {
	case domain.AssetRejected:
		return "rejected_asset", true
	case domain.AssetDraft:
		return "generated_unselected_asset", true
	default:
		return "", false
	}
}

func cleanupAllowedAssetStatuses() []string {
	return []string{domain.AssetRejected, domain.AssetDraft}
}

func addCleanupCandidate(r *domain.CleanupDryRunReport, candidate domain.CleanupCandidate) {
	r.Candidates = append(r.Candidates, candidate)
	r.Summary.CandidateCount++
	r.Summary.FileCount += candidate.FileCount
	r.Summary.Bytes += candidate.Bytes
	r.Summary.ByReason[candidate.Reason]++
}

func (s *Service) executeCleanupCandidate(ctx context.Context, scope domain.Scope, candidate domain.CleanupCandidate) domain.CleanupExecutionResult {
	result := domain.CleanupExecutionResult{
		Kind:    candidate.Kind,
		Reason:  candidate.Reason,
		AssetID: candidate.AssetID,
		TaskID:  candidate.TaskID,
		Status:  candidate.Status,
	}
	switch candidate.Kind {
	case "asset":
		return s.executeCleanupAssetCandidate(ctx, scope, candidate, result)
	case "file":
		result.Files = s.deleteCleanupCandidateFiles(candidate.Files)
		if cleanupFilesHaveFailure(result.Files) {
			result.Action = "failed"
			result.Error = "one or more cleanup files could not be deleted"
			return result
		}
		if cleanupFilesDeletedAny(result.Files) {
			result.Action = "deleted"
		} else {
			result.Action = "skipped"
		}
		return result
	default:
		result.Action = "failed"
		result.Error = fmt.Sprintf("unknown cleanup candidate kind %q", candidate.Kind)
		return result
	}
}

func (s *Service) executeCleanupAssetCandidate(ctx context.Context, scope domain.Scope, candidate domain.CleanupCandidate, result domain.CleanupExecutionResult) domain.CleanupExecutionResult {
	if _, ok := cleanupDryRunReasonForAssetStatus(candidate.Status); !ok {
		result.Action = "skipped"
		result.Error = fmt.Sprintf("asset %s is %s and is protected from cleanup", candidate.AssetID, candidate.Status)
		return result
	}
	deleted, err := s.store.DeleteCleanupAssetCandidate(ctx, scope, candidate.AssetID, cleanupAllowedAssetStatuses())
	if err != nil {
		result.Action = cleanupAssetErrorAction(err)
		result.Error = err.Error()
		return result
	}
	result.Status = deleted.Status
	result.Files = s.deleteCleanupCandidateFiles(candidate.Files)
	if cleanupFilesHaveFailure(result.Files) {
		result.Action = "failed"
		result.Error = "asset database rows were removed, but one or more files could not be deleted"
		return result
	}
	result.Action = "deleted"
	return result
}

func cleanupAssetErrorAction(err error) string {
	if errors.Is(err, store.ErrNotFound) {
		return "skipped"
	}
	if strings.Contains(err.Error(), "protected from cleanup") {
		return "skipped"
	}
	return "failed"
}

func (s *Service) deleteCleanupCandidateFiles(files []domain.CleanupCandidateFile) []domain.CleanupExecutionFile {
	results := make([]domain.CleanupExecutionFile, 0, len(files))
	for _, file := range files {
		result := domain.CleanupExecutionFile{
			Kind:       file.Kind,
			StorageKey: file.StorageKey,
			Bytes:      file.Bytes,
		}
		deletedBytes, err := s.storage.DeleteStorageKey(file.StorageKey)
		if err == nil {
			result.Action = "deleted"
			result.Bytes = deletedBytes
			results = append(results, result)
			continue
		}
		if os.IsNotExist(err) {
			result.Action = "missing"
			results = append(results, result)
			continue
		}
		result.Action = "failed"
		result.Error = err.Error()
		results = append(results, result)
	}
	return results
}

func cleanupFilesHaveFailure(files []domain.CleanupExecutionFile) bool {
	for _, file := range files {
		if file.Action == "failed" {
			return true
		}
	}
	return false
}

func cleanupFilesDeletedAny(files []domain.CleanupExecutionFile) bool {
	for _, file := range files {
		if file.Action == "deleted" {
			return true
		}
	}
	return false
}

func cleanupDryRunToken(report domain.CleanupDryRunReport) string {
	builder := strings.Builder{}
	builder.WriteString(report.Scope.WorkspaceID)
	builder.WriteByte('/')
	builder.WriteString(report.Scope.ProjectID)
	builder.WriteByte('/')
	builder.WriteString(report.Scope.CampaignID)
	builder.WriteByte('\n')

	entries := make([]string, 0, len(report.Candidates))
	for _, candidate := range report.Candidates {
		fileKeys := make([]string, 0, len(candidate.Files))
		for _, file := range candidate.Files {
			fileKeys = append(fileKeys, fmt.Sprintf("%s:%s:%d", file.Kind, file.StorageKey, file.Bytes))
		}
		sort.Strings(fileKeys)
		entries = append(entries, fmt.Sprintf("%s|%s|%s|%s|%s|%d|%d|%s",
			candidate.Kind,
			candidate.Reason,
			candidate.AssetID,
			candidate.TaskID,
			candidate.Status,
			candidate.FileCount,
			candidate.Bytes,
			strings.Join(fileKeys, ","),
		))
	}
	sort.Strings(entries)
	for _, entry := range entries {
		builder.WriteString(entry)
		builder.WriteByte('\n')
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return "cleanup_" + hex.EncodeToString(sum[:16])
}

func copyCleanupReasonCounts(source map[string]int) map[string]int {
	copied := map[string]int{}
	for key, value := range source {
		copied[key] = value
	}
	return copied
}

func (s *Service) appendCleanupExecutionAudit(ctx context.Context, scope domain.Scope, actor string, success bool, message string) string {
	eventID := domain.NewID("audit")
	statusCode := 200
	errorCode := ""
	if !success {
		statusCode = 500
		errorCode = "cleanup_execute_failed"
	}
	if strings.TrimSpace(actor) == "" {
		actor = "vag"
	}
	event := domain.HTTPAuditEvent{
		EventID:      eventID,
		Timestamp:    time.Now().UTC(),
		Source:       domain.HTTPAuditSourceCLI,
		Method:       "CLI",
		Path:         "vag storage cleanup-execute",
		Route:        "cli:vag storage cleanup-execute",
		Action:       "storage_cleanup_execute",
		StatusCode:   statusCode,
		Success:      success,
		AuthMode:     "local_cli",
		Actor:        actor,
		WorkspaceID:  scope.WorkspaceID,
		ProjectID:    scope.ProjectID,
		CampaignID:   scope.CampaignID,
		ErrorCode:    errorCode,
		ErrorMessage: truncateAuditMessage(message),
	}
	if err := s.storage.AppendHTTPAuditEvent(context.WithoutCancel(ctx), event); err != nil {
		return ""
	}
	return eventID
}

func truncateAuditMessage(message string) string {
	message = strings.TrimSpace(message)
	if len(message) <= 500 {
		return message
	}
	return message[:500]
}

func (s *Service) assetCleanupCandidate(item domain.AssetWithVersion, reason string) domain.CleanupCandidate {
	files := []domain.CleanupCandidateFile{
		s.cleanupCandidateFile("original", item.Version.FilePath),
		s.cleanupCandidateFile("thumbnail", item.Version.ThumbnailPath),
		s.cleanupCandidateFile("metadata", item.Version.MetadataPath),
	}
	var bytes int64
	var fileCount int64
	for _, file := range files {
		bytes += file.Bytes
		fileCount++
	}
	return domain.CleanupCandidate{
		Kind:      "asset",
		Reason:    reason,
		AssetID:   item.ID,
		TaskID:    item.TaskID,
		Status:    item.Status,
		FileCount: fileCount,
		Bytes:     bytes,
		Files:     files,
	}
}

func (s *Service) cleanupCandidateFile(kind, path string) domain.CleanupCandidateFile {
	var size int64
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		size = info.Size()
	}
	return domain.CleanupCandidateFile{
		Kind:       kind,
		StorageKey: s.storage.StorageKey(path),
		Bytes:      size,
	}
}
