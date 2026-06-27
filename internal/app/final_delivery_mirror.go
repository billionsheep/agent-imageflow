package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

const (
	defaultFinalDeliveryMirrorLimit = 1000
	maxFinalDeliveryMirrorLimit     = 5000
	finalDeliveryMirrorFileName     = "manifest.final.json"
)

func (s *Service) MaterializeBatchFinalDeliveryMirror(ctx context.Context, scope domain.Scope, req domain.BatchFinalDeliveryMirrorRequest) (domain.BatchFinalDeliveryMirrorResponse, error) {
	scope.WorkspaceID = strings.TrimSpace(scope.WorkspaceID)
	scope.ProjectID = strings.TrimSpace(scope.ProjectID)
	scope.CampaignID = strings.TrimSpace(scope.CampaignID)
	if scope.WorkspaceID == "" || scope.ProjectID == "" || scope.CampaignID == "" {
		return domain.BatchFinalDeliveryMirrorResponse{}, fmt.Errorf("workspace_id, project_id and campaign_id are required")
	}
	if err := s.store.CheckScope(ctx, scope); err != nil {
		return domain.BatchFinalDeliveryMirrorResponse{}, err
	}
	batchID := strings.TrimSpace(req.BatchID)
	if batchID == "" {
		return domain.BatchFinalDeliveryMirrorResponse{}, fmt.Errorf("batch_id is required")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = defaultFinalDeliveryMirrorLimit
	}
	if limit > maxFinalDeliveryMirrorLimit {
		limit = maxFinalDeliveryMirrorLimit
	}
	manifest, err := s.GetBatchManifest(ctx, domain.BatchManifestQuery{
		BatchStorySummaryQuery: domain.BatchStorySummaryQuery{
			ProjectID:  scope.ProjectID,
			CampaignID: scope.CampaignID,
			SessionID:  strings.TrimSpace(req.SessionID),
			BatchID:    batchID,
			Limit:      limit,
		},
		SelectedOnly: true,
		View:         domain.BatchManifestViewFinalDelivery,
	})
	if err != nil {
		return domain.BatchFinalDeliveryMirrorResponse{}, err
	}
	return materializeFinalDeliveryMirror(s.cfg.FinalDeliveryMirrorRoot, scope, manifest, func(assetID string) (domain.AssetWithVersion, error) {
		item, err := s.store.GetAssetWithVersion(ctx, strings.TrimSpace(assetID))
		if err != nil {
			return domain.AssetWithVersion{}, err
		}
		if item.WorkspaceID != scope.WorkspaceID || item.ProjectID != scope.ProjectID || item.CampaignID != scope.CampaignID {
			return domain.AssetWithVersion{}, fmt.Errorf("asset %s does not belong to scope %s/%s/%s", assetID, scope.WorkspaceID, scope.ProjectID, scope.CampaignID)
		}
		return item, nil
	})
}

func materializeFinalDeliveryMirror(root string, scope domain.Scope, manifest domain.BatchManifestResponse, lookup func(assetID string) (domain.AssetWithVersion, error)) (domain.BatchFinalDeliveryMirrorResponse, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return domain.BatchFinalDeliveryMirrorResponse{}, fmt.Errorf("final delivery mirror root is not configured")
	}
	if lookup == nil {
		return domain.BatchFinalDeliveryMirrorResponse{}, fmt.Errorf("asset lookup is required")
	}
	batchID := strings.TrimSpace(manifest.BatchID)
	if batchID == "" {
		return domain.BatchFinalDeliveryMirrorResponse{}, fmt.Errorf("batch_id is required")
	}
	if manifest.FinalDelivery == nil {
		return domain.BatchFinalDeliveryMirrorResponse{}, fmt.Errorf("final_delivery manifest is required")
	}

	mirrorRelativePath := filepath.Join("workspaces", scope.WorkspaceID, "projects", scope.ProjectID, "batches", batchID)
	batchDir := filepath.Join(root, mirrorRelativePath)
	parentDir := filepath.Dir(batchDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return domain.BatchFinalDeliveryMirrorResponse{}, err
	}
	stageDir, err := os.MkdirTemp(parentDir, filepath.Base(batchDir)+".tmp-*")
	if err != nil {
		return domain.BatchFinalDeliveryMirrorResponse{}, err
	}
	defer os.RemoveAll(stageDir)

	finalDir := filepath.Join(stageDir, "final")
	thumbnailDir := filepath.Join(stageDir, "thumbnails")
	for _, dir := range []string{finalDir, thumbnailDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return domain.BatchFinalDeliveryMirrorResponse{}, err
		}
	}

	for _, asset := range manifest.FinalDelivery.FinalAssets {
		item, err := lookup(strings.TrimSpace(asset.AssetID))
		if err != nil {
			return domain.BatchFinalDeliveryMirrorResponse{}, err
		}
		finalRelative, thumbnailRelative, err := finalDeliveryMirrorTargetPaths(asset, item.Version.FilePath)
		if err != nil {
			return domain.BatchFinalDeliveryMirrorResponse{}, err
		}
		if err := copyFileIntoMirror(item.Version.FilePath, filepath.Join(finalDir, filepath.FromSlash(finalRelative))); err != nil {
			return domain.BatchFinalDeliveryMirrorResponse{}, err
		}
		if err := copyFileIntoMirror(item.Version.ThumbnailPath, filepath.Join(thumbnailDir, filepath.FromSlash(thumbnailRelative))); err != nil {
			return domain.BatchFinalDeliveryMirrorResponse{}, err
		}
	}

	manifestPath := filepath.Join(stageDir, finalDeliveryMirrorFileName)
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return domain.BatchFinalDeliveryMirrorResponse{}, err
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		return domain.BatchFinalDeliveryMirrorResponse{}, err
	}

	if err := os.RemoveAll(batchDir); err != nil {
		return domain.BatchFinalDeliveryMirrorResponse{}, err
	}
	if err := os.Rename(stageDir, batchDir); err != nil {
		return domain.BatchFinalDeliveryMirrorResponse{}, err
	}

	result := domain.BatchFinalDeliveryMirrorResponse{
		GeneratedAt:        manifest.GeneratedAt,
		WorkspaceID:        scope.WorkspaceID,
		ProjectID:          scope.ProjectID,
		CampaignID:         scope.CampaignID,
		SessionID:          manifest.SessionID,
		BatchID:            batchID,
		ManifestView:       manifest.ManifestView,
		SelectedOnly:       manifest.SelectedOnly,
		MirrorRelativePath: filepath.ToSlash(mirrorRelativePath),
		ManifestFile:       filepath.ToSlash(filepath.Join(mirrorRelativePath, finalDeliveryMirrorFileName)),
		FinalDir:           filepath.ToSlash(filepath.Join(mirrorRelativePath, "final")),
		ThumbnailDir:       filepath.ToSlash(filepath.Join(mirrorRelativePath, "thumbnails")),
		SceneCount:         manifest.FinalDelivery.Counts.SceneCount,
		FinalAssetCount:    len(manifest.FinalDelivery.FinalAssets),
	}
	return result, nil
}

func finalDeliveryMirrorTargetPaths(asset domain.BatchFinalDeliveryAsset, sourceOriginalPath string) (string, string, error) {
	if cleaned, ok, err := sanitizeMirrorRelativePath(strings.TrimSpace(asset.TargetPath)); err != nil {
		return "", "", err
	} else if ok {
		return cleaned, replaceExtension(cleaned, ".webp"), nil
	}
	storyID := strings.TrimSpace(asset.StoryID)
	sceneID := strings.TrimSpace(asset.SceneID)
	if sceneID == "" {
		sceneID = strings.TrimSpace(asset.AssetID)
	}
	fileExt := filepath.Ext(strings.TrimSpace(sourceOriginalPath))
	if fileExt == "" {
		fileExt = ".png"
	}
	fallback := sceneID + fileExt
	if storyID != "" {
		fallback = path.Join("stories", storyID, fallback)
	}
	return fallback, replaceExtension(fallback, ".webp"), nil
}

func sanitizeMirrorRelativePath(value string) (string, bool, error) {
	if value == "" {
		return "", false, nil
	}
	normalized := strings.ReplaceAll(value, "\\", "/")
	cleaned := path.Clean(normalized)
	if cleaned == "." || cleaned == "" {
		return "", false, nil
	}
	if strings.HasPrefix(cleaned, "/") {
		return "", false, fmt.Errorf("target_path must be relative: %s", value)
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if segment == ".." {
			return "", false, fmt.Errorf("target_path must not escape the mirror root: %s", value)
		}
	}
	return cleaned, true, nil
}

func replaceExtension(value, ext string) string {
	base := strings.TrimSuffix(value, path.Ext(value))
	return base + ext
}

func copyFileIntoMirror(src, dst string) error {
	if strings.TrimSpace(src) == "" {
		return fmt.Errorf("source file path is required")
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
