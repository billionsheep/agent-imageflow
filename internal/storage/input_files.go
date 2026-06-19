package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

type StoredTaskInputFile struct {
	InputFileID      string
	WorkspaceID      string
	ProjectID        string
	CampaignID       string
	Kind             string
	OriginalFilename string
	FilePath         string
	MetadataPath     string
	MimeType         string
	Width            int
	Height           int
	SizeBytes        int64
}

type taskInputFileMetadata struct {
	InputFileID      string `json:"input_file_id"`
	WorkspaceID      string `json:"workspace_id"`
	ProjectID        string `json:"project_id"`
	CampaignID       string `json:"campaign_id"`
	Kind             string `json:"kind"`
	OriginalFilename string `json:"original_filename"`
	FilePath         string `json:"file_path"`
	MimeType         string `json:"mime_type"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	SizeBytes        int64  `json:"size_bytes"`
}

func (s LocalStorage) StoreTaskInputFile(ctx context.Context, scope domain.Scope, inputFileID, kind, originalFilename, mimeType string, raw []byte) (StoredTaskInputFile, error) {
	normalizedKind, ok := domain.NormalizeInputFileKind(kind)
	if !ok {
		return StoredTaskInputFile{}, fmt.Errorf("unknown input file kind %q", kind)
	}
	if inputFileID == "" {
		return StoredTaskInputFile{}, fmt.Errorf("input file id is required")
	}
	if len(raw) == 0 {
		return StoredTaskInputFile{}, fmt.Errorf("input file is empty")
	}

	mimeType = normalizeInputImageMimeType(mimeType, raw)
	if !strings.HasPrefix(mimeType, "image/") {
		return StoredTaskInputFile{}, fmt.Errorf("unsupported input file mime type %q", mimeType)
	}
	width, height, err := detectImageDimensions(raw)
	if err != nil {
		return StoredTaskInputFile{}, fmt.Errorf("decode input image: %w", err)
	}

	base := filepath.Join(s.root, "workspaces", scope.WorkspaceID, "projects", scope.ProjectID, "campaigns", scope.CampaignID)
	tmpDir := filepath.Join(base, "tmp-inputs", inputFileID)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return StoredTaskInputFile{}, err
	}

	fileExt := fileExtensionForMimeType(mimeType)
	tmpOriginalPath := filepath.Join(tmpDir, "original"+fileExt)
	tmpMetadataPath := filepath.Join(tmpDir, "metadata.json")
	if err := os.WriteFile(tmpOriginalPath, raw, 0o644); err != nil {
		return StoredTaskInputFile{}, err
	}

	finalDir := filepath.Join(base, "input-files", inputFileID)
	finalOriginalPath := filepath.Join(finalDir, "original"+fileExt)
	finalMetadataPath := filepath.Join(finalDir, "metadata.json")
	if err := os.MkdirAll(finalDir, 0o755); err != nil {
		return StoredTaskInputFile{}, err
	}

	metadata := taskInputFileMetadata{
		InputFileID:      inputFileID,
		WorkspaceID:      scope.WorkspaceID,
		ProjectID:        scope.ProjectID,
		CampaignID:       scope.CampaignID,
		Kind:             normalizedKind,
		OriginalFilename: strings.TrimSpace(originalFilename),
		FilePath:         finalOriginalPath,
		MimeType:         mimeType,
		Width:            width,
		Height:           height,
		SizeBytes:        int64(len(raw)),
	}
	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return StoredTaskInputFile{}, err
	}
	if err := os.WriteFile(tmpMetadataPath, metadataBytes, 0o644); err != nil {
		return StoredTaskInputFile{}, err
	}

	select {
	case <-ctx.Done():
		return StoredTaskInputFile{}, ctx.Err()
	default:
	}

	if err := os.Rename(tmpOriginalPath, finalOriginalPath); err != nil {
		return StoredTaskInputFile{}, err
	}
	if err := os.Rename(tmpMetadataPath, finalMetadataPath); err != nil {
		return StoredTaskInputFile{}, err
	}
	_ = os.RemoveAll(tmpDir)

	return StoredTaskInputFile{
		InputFileID:      inputFileID,
		WorkspaceID:      scope.WorkspaceID,
		ProjectID:        scope.ProjectID,
		CampaignID:       scope.CampaignID,
		Kind:             normalizedKind,
		OriginalFilename: metadata.OriginalFilename,
		FilePath:         finalOriginalPath,
		MetadataPath:     finalMetadataPath,
		MimeType:         mimeType,
		Width:            width,
		Height:           height,
		SizeBytes:        int64(len(raw)),
	}, nil
}

func (s LocalStorage) GetTaskInputFile(scope domain.Scope, inputFileID string) (StoredTaskInputFile, error) {
	metadataPath := filepath.Join(
		s.root,
		"workspaces", scope.WorkspaceID,
		"projects", scope.ProjectID,
		"campaigns", scope.CampaignID,
		"input-files", inputFileID,
		"metadata.json",
	)
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return StoredTaskInputFile{}, err
	}
	var metadata taskInputFileMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return StoredTaskInputFile{}, err
	}
	if metadata.InputFileID != inputFileID ||
		metadata.WorkspaceID != scope.WorkspaceID ||
		metadata.ProjectID != scope.ProjectID ||
		metadata.CampaignID != scope.CampaignID {
		return StoredTaskInputFile{}, fmt.Errorf("input file scope mismatch")
	}
	if _, err := os.Stat(metadata.FilePath); err != nil {
		return StoredTaskInputFile{}, err
	}
	return StoredTaskInputFile{
		InputFileID:      metadata.InputFileID,
		WorkspaceID:      metadata.WorkspaceID,
		ProjectID:        metadata.ProjectID,
		CampaignID:       metadata.CampaignID,
		Kind:             metadata.Kind,
		OriginalFilename: metadata.OriginalFilename,
		FilePath:         metadata.FilePath,
		MetadataPath:     metadataPath,
		MimeType:         metadata.MimeType,
		Width:            metadata.Width,
		Height:           metadata.Height,
		SizeBytes:        metadata.SizeBytes,
	}, nil
}

func (s LocalStorage) DeleteWorkspaceScopeData(workspaceID string) error {
	return os.RemoveAll(filepath.Join(s.root, "workspaces", workspaceID))
}

func (s LocalStorage) DeleteProjectScopeData(workspaceID, projectID string) error {
	return os.RemoveAll(filepath.Join(
		s.root,
		"workspaces", workspaceID,
		"projects", projectID,
	))
}

func (s LocalStorage) DeleteCampaignScopeData(scope domain.Scope) error {
	return os.RemoveAll(filepath.Join(
		s.root,
		"workspaces", scope.WorkspaceID,
		"projects", scope.ProjectID,
		"campaigns", scope.CampaignID,
	))
}

func normalizeInputImageMimeType(mimeType string, raw []byte) string {
	normalized := strings.ToLower(strings.TrimSpace(mimeType))
	if normalized != "" && normalized != "application/octet-stream" {
		return normalized
	}
	detected := strings.ToLower(strings.TrimSpace(http.DetectContentType(raw)))
	if strings.HasPrefix(detected, "image/") {
		return detected
	}
	return "application/octet-stream"
}
