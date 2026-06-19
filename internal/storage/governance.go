package storage

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

const (
	UsageCategoryOriginal   = "original"
	UsageCategoryThumbnail  = "thumbnail"
	UsageCategoryMetadata   = "metadata"
	UsageCategoryInputFiles = "input_files"
	UsageCategoryAudit      = "audit"
	UsageCategoryTmp        = "tmp"
	UsageCategoryOrphan     = "orphan"
	UsageCategoryOther      = "other"
)

type UsageScanOptions struct {
	Scope               domain.Scope
	KnownAssetFilePaths []string
}

type CleanupFileCandidateOptions struct {
	Scope               domain.Scope
	KnownAssetFilePaths []string
	IncludeTmp          bool
	IncludeOrphans      bool
	Limit               int
}

type usageFileClassification struct {
	Category    string
	FileKind    string
	WorkspaceID string
	ProjectID   string
	CampaignID  string
}

type usageAccumulator struct {
	scopeType   string
	workspaceID string
	projectID   string
	campaignID  string
	fileCount   int64
	bytes       int64
	categories  map[string]domain.StorageUsageCategoryStat
}

func (s LocalStorage) ScanUsage(ctx context.Context, opts UsageScanOptions) (domain.StorageUsageScopes, error) {
	result := domain.StorageUsageScopes{
		Instance:  newUsageAccumulator("instance", "", "", "").Snapshot(),
		Workspace: newUsageAccumulator("workspace", opts.Scope.WorkspaceID, "", "").Snapshot(),
		Project:   newUsageAccumulator("project", opts.Scope.WorkspaceID, opts.Scope.ProjectID, "").Snapshot(),
		Campaign:  newUsageAccumulator("campaign", opts.Scope.WorkspaceID, opts.Scope.ProjectID, opts.Scope.CampaignID).Snapshot(),
	}
	root := strings.TrimSpace(s.root)
	if root == "" {
		return result, nil
	}
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, err
	}

	knownSet := cleanPathSet(opts.KnownAssetFilePaths)
	useKnownSet := opts.KnownAssetFilePaths != nil
	instance := newUsageAccumulator("instance", "", "", "")
	workspace := newUsageAccumulator("workspace", opts.Scope.WorkspaceID, "", "")
	project := newUsageAccumulator("project", opts.Scope.WorkspaceID, opts.Scope.ProjectID, "")
	campaign := newUsageAccumulator("campaign", opts.Scope.WorkspaceID, opts.Scope.ProjectID, opts.Scope.CampaignID)

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		classification := classifyUsageFile(root, path, knownSet, useKnownSet)
		instance.Add(classification.Category, info.Size())
		if opts.Scope.WorkspaceID != "" && classification.WorkspaceID == opts.Scope.WorkspaceID {
			workspace.Add(classification.Category, info.Size())
		}
		if opts.Scope.ProjectID != "" &&
			classification.WorkspaceID == opts.Scope.WorkspaceID &&
			classification.ProjectID == opts.Scope.ProjectID {
			project.Add(classification.Category, info.Size())
		}
		if opts.Scope.CampaignID != "" &&
			classification.WorkspaceID == opts.Scope.WorkspaceID &&
			classification.ProjectID == opts.Scope.ProjectID &&
			classification.CampaignID == opts.Scope.CampaignID {
			campaign.Add(classification.Category, info.Size())
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return result, err
	}

	result.Instance = instance.Snapshot()
	result.Workspace = workspace.Snapshot()
	result.Project = project.Snapshot()
	result.Campaign = campaign.Snapshot()
	return result, nil
}

func (s LocalStorage) ListCleanupFileCandidates(ctx context.Context, opts CleanupFileCandidateOptions) ([]domain.CleanupCandidate, error) {
	root := strings.TrimSpace(s.root)
	if root == "" {
		return nil, nil
	}
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	limit := opts.Limit
	if limit < 1 {
		limit = 100
	}
	knownSet := cleanPathSet(opts.KnownAssetFilePaths)
	candidates := []domain.CleanupCandidate{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return nil
		}
		if len(candidates) >= limit {
			return fs.SkipAll
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		classification := classifyUsageFile(root, path, knownSet, true)
		if !cleanupFileMatchesScope(classification, opts.Scope) {
			return nil
		}
		reason := ""
		switch classification.Category {
		case UsageCategoryOrphan:
			if opts.IncludeOrphans {
				reason = "orphan_file"
			}
		case UsageCategoryTmp:
			if opts.IncludeTmp {
				reason = "failed_task_tmp"
			}
		}
		if reason == "" {
			return nil
		}
		kind := classification.FileKind
		if kind == "" {
			kind = classification.Category
		}
		candidates = append(candidates, domain.CleanupCandidate{
			Kind:      "file",
			Reason:    reason,
			FileCount: 1,
			Bytes:     info.Size(),
			Files: []domain.CleanupCandidateFile{
				{
					Kind:       kind,
					StorageKey: s.StorageKey(path),
					Bytes:      info.Size(),
				},
			},
		})
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return candidates, nil
}

func (s LocalStorage) StorageKey(path string) string {
	root := strings.TrimSpace(s.root)
	if root == "" || strings.TrimSpace(path) == "" {
		return ""
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return ""
	}
	return filepath.ToSlash(rel)
}

func (s LocalStorage) DeleteStorageKey(storageKey string) (int64, error) {
	root := strings.TrimSpace(s.root)
	if root == "" {
		return 0, fmt.Errorf("storage root is empty")
	}
	key := strings.TrimSpace(storageKey)
	if key == "" {
		return 0, fmt.Errorf("storage key is required")
	}
	key = filepath.Clean(filepath.FromSlash(key))
	if key == "." || key == ".." || filepath.IsAbs(key) || strings.HasPrefix(key, ".."+string(filepath.Separator)) {
		return 0, fmt.Errorf("unsafe storage key %q", storageKey)
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return 0, err
	}
	pathAbs, err := filepath.Abs(filepath.Join(rootAbs, key))
	if err != nil {
		return 0, err
	}
	rel, err := filepath.Rel(rootAbs, pathAbs)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return 0, fmt.Errorf("storage key %q escapes storage root", storageKey)
	}

	info, err := os.Stat(pathAbs)
	if err != nil {
		return 0, err
	}
	if info.IsDir() {
		return 0, fmt.Errorf("storage key %q points to a directory", storageKey)
	}
	if err := os.Remove(pathAbs); err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func newUsageAccumulator(scopeType, workspaceID, projectID, campaignID string) *usageAccumulator {
	return &usageAccumulator{
		scopeType:   scopeType,
		workspaceID: workspaceID,
		projectID:   projectID,
		campaignID:  campaignID,
		categories:  map[string]domain.StorageUsageCategoryStat{},
	}
}

func (a *usageAccumulator) Add(category string, bytes int64) {
	if category == "" {
		category = UsageCategoryOther
	}
	item := a.categories[category]
	item.Category = category
	item.FileCount++
	item.Bytes += bytes
	a.categories[category] = item
	a.fileCount++
	a.bytes += bytes
}

func (a *usageAccumulator) Snapshot() domain.StorageUsageSnapshot {
	categories := make([]domain.StorageUsageCategoryStat, 0, len(a.categories))
	for _, item := range a.categories {
		categories = append(categories, item)
	}
	sort.Slice(categories, func(i, j int) bool {
		return usageCategoryRank(categories[i].Category) < usageCategoryRank(categories[j].Category)
	})
	return domain.StorageUsageSnapshot{
		ScopeType:   a.scopeType,
		WorkspaceID: a.workspaceID,
		ProjectID:   a.projectID,
		CampaignID:  a.campaignID,
		FileCount:   a.fileCount,
		Bytes:       a.bytes,
		Categories:  categories,
	}
}

func usageCategoryRank(category string) int {
	order := []string{
		UsageCategoryOriginal,
		UsageCategoryThumbnail,
		UsageCategoryMetadata,
		UsageCategoryInputFiles,
		UsageCategoryAudit,
		UsageCategoryTmp,
		UsageCategoryOrphan,
		UsageCategoryOther,
	}
	for index, item := range order {
		if item == category {
			return index
		}
	}
	return len(order)
}

func cleanPathSet(paths []string) map[string]bool {
	set := map[string]bool{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		set[filepath.Clean(path)] = true
	}
	return set
}

func classifyUsageFile(root, path string, knownSet map[string]bool, useKnownSet bool) usageFileClassification {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return usageFileClassification{Category: UsageCategoryOther, FileKind: UsageCategoryOther}
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) == 0 {
		return usageFileClassification{Category: UsageCategoryOther, FileKind: UsageCategoryOther}
	}
	if parts[0] == "audit" {
		return usageFileClassification{Category: UsageCategoryAudit, FileKind: UsageCategoryAudit}
	}
	if len(parts) < 7 ||
		parts[0] != "workspaces" ||
		parts[2] != "projects" ||
		parts[4] != "campaigns" {
		return usageFileClassification{Category: UsageCategoryOther, FileKind: UsageCategoryOther}
	}

	classification := usageFileClassification{
		WorkspaceID: parts[1],
		ProjectID:   parts[3],
		CampaignID:  parts[5],
		Category:    UsageCategoryOther,
		FileKind:    UsageCategoryOther,
	}
	switch parts[6] {
	case "originals":
		classification.Category = UsageCategoryOriginal
		classification.FileKind = UsageCategoryOriginal
	case "thumbnails":
		classification.Category = UsageCategoryThumbnail
		classification.FileKind = UsageCategoryThumbnail
	case "metadata":
		classification.Category = UsageCategoryMetadata
		classification.FileKind = UsageCategoryMetadata
	case "input-files":
		classification.Category = UsageCategoryInputFiles
		classification.FileKind = UsageCategoryInputFiles
	case "tmp", "tmp-inputs":
		classification.Category = UsageCategoryTmp
		classification.FileKind = UsageCategoryTmp
	}
	if useKnownSet && isFinalAssetUsageCategory(classification.Category) && !knownSet[filepath.Clean(path)] {
		classification.Category = UsageCategoryOrphan
	}
	return classification
}

func isFinalAssetUsageCategory(category string) bool {
	return category == UsageCategoryOriginal || category == UsageCategoryThumbnail || category == UsageCategoryMetadata
}

func cleanupFileMatchesScope(classification usageFileClassification, scope domain.Scope) bool {
	if scope.WorkspaceID != "" && classification.WorkspaceID != scope.WorkspaceID {
		return false
	}
	if scope.ProjectID != "" && classification.ProjectID != scope.ProjectID {
		return false
	}
	if scope.CampaignID != "" && classification.CampaignID != scope.CampaignID {
		return false
	}
	return true
}
