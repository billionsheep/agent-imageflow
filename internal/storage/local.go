package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/provider"
)

type LocalStorage struct {
	root               string
	thumbnailMaxWidth  int
	thumbnailMaxHeight int
	cwebpBinary        string
}

type StoredAssetFile struct {
	AssetID       string
	VersionID     string
	Version       int
	FilePath      string
	ThumbnailPath string
	MetadataPath  string
	Hash          string
	MimeType      string
	Width         int
	Height        int
	StoreMs       int64
	ThumbnailMs   int64
}

func NewLocalStorage(root string, thumbnailMaxWidth, thumbnailMaxHeight int) LocalStorage {
	return LocalStorage{
		root:               root,
		thumbnailMaxWidth:  effectiveThumbnailBound(thumbnailMaxWidth, defaultThumbnailMaxWidth),
		thumbnailMaxHeight: effectiveThumbnailBound(thumbnailMaxHeight, defaultThumbnailMaxHeight),
		cwebpBinary:        defaultCWebPBinary,
	}
}

func (s LocalStorage) Root() string {
	return s.root
}

func (s LocalStorage) StoreGeneratedFile(ctx context.Context, task domain.Task, assetID string, versionID string, file provider.GeneratedFile) (StoredAssetFile, error) {
	started := time.Now()
	version := 1
	base := filepath.Join(s.root, "workspaces", task.WorkspaceID, "projects", task.ProjectID, "campaigns", task.CampaignID)
	tmpDir := filepath.Join(base, "tmp", assetID)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return StoredAssetFile{}, err
	}

	originalExt := fileExtensionForMimeType(file.MimeType)
	tmpOriginal := filepath.Join(tmpDir, "original"+originalExt)
	tmpThumbnail := filepath.Join(tmpDir, "thumbnail.webp")
	tmpMetadata := filepath.Join(tmpDir, "metadata.json")
	if err := os.WriteFile(tmpOriginal, file.Bytes, 0o644); err != nil {
		return StoredAssetFile{}, err
	}

	width := file.Width
	height := file.Height
	if width <= 0 || height <= 0 {
		detectedWidth, detectedHeight, err := detectImageDimensions(file.Bytes)
		if err != nil {
			return StoredAssetFile{}, err
		}
		width = detectedWidth
		height = detectedHeight
	}
	thumbnailStarted := time.Now()
	thumbnailWidth, thumbnailHeight, err := s.createThumbnail(ctx, tmpOriginal, tmpThumbnail, width, height)
	if err != nil {
		return StoredAssetFile{}, err
	}
	thumbnailMs := time.Since(thumbnailStarted).Milliseconds()

	hashBytes := sha256.Sum256(file.Bytes)
	hash := "sha256:" + hex.EncodeToString(hashBytes[:])
	metadata := map[string]any{
		"asset_id":         assetID,
		"version_id":       versionID,
		"version":          version,
		"task_id":          task.ID,
		"workspace_id":     task.WorkspaceID,
		"project_id":       task.ProjectID,
		"campaign_id":      task.CampaignID,
		"provider":         task.Provider,
		"model":            file.Model,
		"prompt":           task.Prompt,
		"hash":             hash,
		"width":            width,
		"height":           height,
		"thumbnail_width":  thumbnailWidth,
		"thumbnail_height": thumbnailHeight,
		"thumbnail_mime":   "image/webp",
	}
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return StoredAssetFile{}, err
	}
	if err := os.WriteFile(tmpMetadata, metaBytes, 0o644); err != nil {
		return StoredAssetFile{}, err
	}

	select {
	case <-ctx.Done():
		return StoredAssetFile{}, ctx.Err()
	default:
	}

	originalPath := filepath.Join(base, "originals", assetID, "1"+originalExt)
	thumbnailPath := filepath.Join(base, "thumbnails", assetID, "1.webp")
	metadataPath := filepath.Join(base, "metadata", assetID, "1.json")
	for _, path := range []string{originalPath, thumbnailPath, metadataPath} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return StoredAssetFile{}, err
		}
	}
	if err := os.Rename(tmpOriginal, originalPath); err != nil {
		return StoredAssetFile{}, err
	}
	if err := os.Rename(tmpThumbnail, thumbnailPath); err != nil {
		return StoredAssetFile{}, err
	}
	if err := os.Rename(tmpMetadata, metadataPath); err != nil {
		return StoredAssetFile{}, err
	}
	_ = os.RemoveAll(tmpDir)

	return StoredAssetFile{
		AssetID:       assetID,
		VersionID:     versionID,
		Version:       version,
		FilePath:      originalPath,
		ThumbnailPath: thumbnailPath,
		MetadataPath:  metadataPath,
		Hash:          hash,
		MimeType:      file.MimeType,
		Width:         width,
		Height:        height,
		StoreMs:       time.Since(started).Milliseconds(),
		ThumbnailMs:   thumbnailMs,
	}, nil
}
