package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/storage"
	"github.com/billionsheep/agent-imageflow/internal/store"
)

const (
	maxRemoteTaskInputBytes = 50 << 20

	taskInputSourceUpload     = "agent-imageflow-upload"
	taskInputSourceRemoteURL  = "agent-imageflow-remote-url"
	taskInputSourceAssetReuse = "agent-imageflow-asset-reuse"
)

type resolvedTaskInputFiles struct {
	ReferenceImages []resolvedTaskInputFile `json:"reference_images,omitempty"`
	MaskImage       *resolvedTaskInputFile  `json:"mask_image,omitempty"`
}

type resolvedTaskInputFile struct {
	InputFileID   string `json:"input_file_id"`
	Kind          string `json:"kind"`
	FilePath      string `json:"file_path"`
	MimeType      string `json:"mime_type"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	Role          string `json:"role,omitempty"`
	TargetImageID string `json:"target_image_id,omitempty"`
}

func (s *Service) UploadTaskInputFile(ctx context.Context, scope domain.Scope, kind, originalFilename, mimeType string, raw []byte) (domain.InputFileResponse, error) {
	if err := s.store.CheckScope(ctx, scope); err != nil {
		return domain.InputFileResponse{}, err
	}
	normalizedKind, ok := domain.NormalizeInputFileKind(kind)
	if !ok {
		return domain.InputFileResponse{}, fmt.Errorf("unknown input file kind %q", kind)
	}
	inputFileID := domain.NewID("inp")
	stored, err := s.storage.StoreTaskInputFile(ctx, scope, inputFileID, normalizedKind, originalFilename, mimeType, raw)
	if err != nil {
		return domain.InputFileResponse{}, err
	}
	return s.taskInputFileResponse(scope, stored), nil
}

func (s *Service) GetTaskInputFile(ctx context.Context, scope domain.Scope, inputFileID string) (domain.InputFileResponse, error) {
	if err := s.store.CheckScope(ctx, scope); err != nil {
		return domain.InputFileResponse{}, err
	}
	stored, err := s.storage.GetTaskInputFile(scope, strings.TrimSpace(inputFileID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.InputFileResponse{}, fmt.Errorf("%w: input file %s", store.ErrNotFound, strings.TrimSpace(inputFileID))
		}
		return domain.InputFileResponse{}, err
	}
	return s.taskInputFileResponse(scope, stored), nil
}

func (s *Service) GetTaskInputFileContent(ctx context.Context, scope domain.Scope, inputFileID string) (string, string, error) {
	if err := s.store.CheckScope(ctx, scope); err != nil {
		return "", "", err
	}
	stored, err := s.storage.GetTaskInputFile(scope, strings.TrimSpace(inputFileID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", fmt.Errorf("%w: input file %s", store.ErrNotFound, strings.TrimSpace(inputFileID))
		}
		return "", "", err
	}
	return stored.FilePath, stored.MimeType, nil
}

func (s *Service) taskInputFileResponse(scope domain.Scope, stored storage.StoredTaskInputFile) domain.InputFileResponse {
	return domain.InputFileResponse{
		InputFileID:      stored.InputFileID,
		WorkspaceID:      scope.WorkspaceID,
		ProjectID:        scope.ProjectID,
		CampaignID:       scope.CampaignID,
		Kind:             stored.Kind,
		OriginalFilename: stored.OriginalFilename,
		MimeType:         stored.MimeType,
		Width:            stored.Width,
		Height:           stored.Height,
		SizeBytes:        stored.SizeBytes,
		DownloadURL:      s.taskInputFileContentURL(scope, stored.InputFileID),
		MetadataURL:      s.taskInputFileMetadataURL(scope, stored.InputFileID),
	}
}

func (s *Service) taskInputFileMetadataURL(scope domain.Scope, inputFileID string) string {
	return s.cfg.PublicBaseURL + "/api/workspaces/" + scope.WorkspaceID +
		"/projects/" + scope.ProjectID +
		"/campaigns/" + scope.CampaignID +
		"/input-files/" + inputFileID
}

func (s *Service) taskInputFileContentURL(scope domain.Scope, inputFileID string) string {
	return s.taskInputFileMetadataURL(scope, inputFileID) + "/content"
}

func (s *Service) resolveTaskInputFiles(ctx context.Context, scope domain.Scope, req domain.CreateTaskRequest) (domain.CreateTaskRequest, *resolvedTaskInputFiles, error) {
	resolved := &resolvedTaskInputFiles{}
	for i := range req.ReferenceImages {
		req.ReferenceImages[i].ID = strings.TrimSpace(req.ReferenceImages[i].ID)
		req.ReferenceImages[i].URL = strings.TrimSpace(req.ReferenceImages[i].URL)
		req.ReferenceImages[i].AssetID = strings.TrimSpace(req.ReferenceImages[i].AssetID)
		req.ReferenceImages[i].InputFileID = strings.TrimSpace(req.ReferenceImages[i].InputFileID)
		req.ReferenceImages[i].Role = strings.TrimSpace(req.ReferenceImages[i].Role)
		req.ReferenceImages[i].Source = strings.TrimSpace(req.ReferenceImages[i].Source)
		req.ReferenceImages[i].MimeType = strings.TrimSpace(req.ReferenceImages[i].MimeType)
		resolvedReference, err := s.resolveReferenceImage(ctx, scope, &req.ReferenceImages[i])
		if err != nil {
			return req, nil, err
		}
		if resolvedReference != nil {
			resolved.ReferenceImages = append(resolved.ReferenceImages, *resolvedReference)
		}
	}

	if req.MaskImage != nil {
		req.MaskImage.ID = strings.TrimSpace(req.MaskImage.ID)
		req.MaskImage.URL = strings.TrimSpace(req.MaskImage.URL)
		req.MaskImage.AssetID = strings.TrimSpace(req.MaskImage.AssetID)
		req.MaskImage.InputFileID = strings.TrimSpace(req.MaskImage.InputFileID)
		req.MaskImage.TargetImageID = strings.TrimSpace(req.MaskImage.TargetImageID)
		req.MaskImage.Source = strings.TrimSpace(req.MaskImage.Source)
		req.MaskImage.MimeType = strings.TrimSpace(req.MaskImage.MimeType)
		resolvedMask, err := s.resolveMaskImage(ctx, scope, req.MaskImage, len(resolved.ReferenceImages) > 0)
		if err != nil {
			return req, nil, err
		}
		if resolvedMask != nil {
			resolved.MaskImage = resolvedMask
		}
	}

	if len(resolved.ReferenceImages) == 0 && resolved.MaskImage == nil {
		return req, nil, nil
	}
	return req, resolved, nil
}

func (s *Service) resolveReferenceImage(ctx context.Context, scope domain.Scope, ref *domain.ReferenceImage) (*resolvedTaskInputFile, error) {
	switch {
	case ref.InputFileID != "":
		stored, err := s.getStoredTaskInputFile(scope, ref.InputFileID, domain.InputFileKindReference, "reference image")
		if err != nil {
			return nil, err
		}
		applyStoredReferenceImage(ref, s.taskInputFileContentURL(scope, stored.InputFileID), stored, true, taskInputSourceUpload)
		resolved := resolvedTaskInputFileFromStored(stored)
		resolved.Role = ref.Role
		return &resolved, nil
	case ref.AssetID != "":
		item, err := s.getReusableAsset(ctx, scope, ref.AssetID, "reference image")
		if err != nil {
			return nil, err
		}
		applyAssetReferenceImage(ref, item, s.assetURL(item.ID, "original"))
		resolved := resolvedTaskInputFileFromAsset(item, domain.InputFileKindReference)
		resolved.Role = ref.Role
		return &resolved, nil
	case ref.URL != "":
		stored, err := s.materializeRemoteTaskInputFile(ctx, scope, domain.InputFileKindReference, ref.URL)
		if err != nil {
			return nil, fmt.Errorf("resolve reference image url %q: %w", ref.URL, err)
		}
		applyStoredReferenceImage(ref, s.taskInputFileContentURL(scope, stored.InputFileID), stored, false, taskInputSourceRemoteURL)
		resolved := resolvedTaskInputFileFromStored(stored)
		resolved.Role = ref.Role
		return &resolved, nil
	default:
		return nil, nil
	}
}

func (s *Service) resolveMaskImage(ctx context.Context, scope domain.Scope, mask *domain.MaskImage, hasReferenceImage bool) (*resolvedTaskInputFile, error) {
	switch {
	case mask.InputFileID != "":
		stored, err := s.getStoredTaskInputFile(scope, mask.InputFileID, domain.InputFileKindMask, "mask image")
		if err != nil {
			return nil, err
		}
		if !hasReferenceImage {
			return nil, fmt.Errorf("mask image requires at least one resolved input image")
		}
		applyStoredMaskImage(mask, s.taskInputFileContentURL(scope, stored.InputFileID), stored, true, taskInputSourceUpload)
		resolved := resolvedTaskInputFileFromStored(stored)
		resolved.TargetImageID = mask.TargetImageID
		return &resolved, nil
	case mask.AssetID != "":
		item, err := s.getReusableAsset(ctx, scope, mask.AssetID, "mask image")
		if err != nil {
			return nil, err
		}
		if !hasReferenceImage {
			return nil, fmt.Errorf("mask image requires at least one resolved input image")
		}
		applyAssetMaskImage(mask, item, s.assetURL(item.ID, "original"))
		resolved := resolvedTaskInputFileFromAsset(item, domain.InputFileKindMask)
		resolved.TargetImageID = mask.TargetImageID
		return &resolved, nil
	case mask.URL != "":
		stored, err := s.materializeRemoteTaskInputFile(ctx, scope, domain.InputFileKindMask, mask.URL)
		if err != nil {
			return nil, fmt.Errorf("resolve mask image url %q: %w", mask.URL, err)
		}
		if !hasReferenceImage {
			return nil, fmt.Errorf("mask image requires at least one resolved input image")
		}
		applyStoredMaskImage(mask, s.taskInputFileContentURL(scope, stored.InputFileID), stored, false, taskInputSourceRemoteURL)
		resolved := resolvedTaskInputFileFromStored(stored)
		resolved.TargetImageID = mask.TargetImageID
		return &resolved, nil
	default:
		return nil, nil
	}
}

func (s *Service) getStoredTaskInputFile(scope domain.Scope, inputFileID, expectedKind, label string) (storage.StoredTaskInputFile, error) {
	stored, err := s.storage.GetTaskInputFile(scope, inputFileID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return storage.StoredTaskInputFile{}, fmt.Errorf("resolve %s %q: %w", label, inputFileID, store.ErrNotFound)
		}
		return storage.StoredTaskInputFile{}, fmt.Errorf("resolve %s %q: %w", label, inputFileID, err)
	}
	if stored.Kind != expectedKind {
		return storage.StoredTaskInputFile{}, fmt.Errorf("input file %q is not a %s", stored.InputFileID, expectedKind)
	}
	return stored, nil
}

func (s *Service) getReusableAsset(ctx context.Context, scope domain.Scope, assetID, label string) (domain.AssetWithVersion, error) {
	item, err := s.store.GetAssetWithVersion(ctx, assetID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return domain.AssetWithVersion{}, fmt.Errorf("resolve %s asset %q: %w", label, assetID, store.ErrNotFound)
		}
		return domain.AssetWithVersion{}, fmt.Errorf("resolve %s asset %q: %w", label, assetID, err)
	}
	if err := validateReusableAssetScope(scope, item); err != nil {
		return domain.AssetWithVersion{}, fmt.Errorf("resolve %s asset %q: %w", label, assetID, err)
	}
	return item, nil
}

func (s *Service) materializeRemoteTaskInputFile(ctx context.Context, scope domain.Scope, kind, remoteURL string) (storage.StoredTaskInputFile, error) {
	download, err := fetchRemoteTaskInputFile(ctx, taskInputHTTPClient(s.cfg.ProviderTimeoutSeconds), remoteURL)
	if err != nil {
		return storage.StoredTaskInputFile{}, err
	}
	inputFileID := domain.NewID("inp")
	stored, err := s.storage.StoreTaskInputFile(ctx, scope, inputFileID, kind, download.OriginalFilename, download.MimeType, download.Raw)
	if err != nil {
		return storage.StoredTaskInputFile{}, err
	}
	return stored, nil
}

func taskInputHTTPClient(timeoutSeconds int) *http.Client {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

type remoteTaskInputDownload struct {
	OriginalFilename string
	MimeType         string
	Raw              []byte
}

func fetchRemoteTaskInputFile(ctx context.Context, client *http.Client, rawURL string) (remoteTaskInputDownload, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return remoteTaskInputDownload{}, fmt.Errorf("remote input url is required")
	}
	parsed, err := neturl.Parse(trimmed)
	if err != nil {
		return remoteTaskInputDownload{}, fmt.Errorf("invalid remote input url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return remoteTaskInputDownload{}, fmt.Errorf("remote input url must use http or https")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return remoteTaskInputDownload{}, fmt.Errorf("remote input url host is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return remoteTaskInputDownload{}, err
	}
	req.Header.Set("Accept", "image/*")
	req.Header.Set("User-Agent", "agent-imageflow/0.1")

	resp, err := client.Do(req)
	if err != nil {
		return remoteTaskInputDownload{}, fmt.Errorf("download remote input: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return remoteTaskInputDownload{}, fmt.Errorf("download remote input: unexpected status %s", resp.Status)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxRemoteTaskInputBytes+1))
	if err != nil {
		return remoteTaskInputDownload{}, fmt.Errorf("download remote input: %w", err)
	}
	if len(raw) == 0 {
		return remoteTaskInputDownload{}, fmt.Errorf("remote input is empty")
	}
	if len(raw) > maxRemoteTaskInputBytes {
		return remoteTaskInputDownload{}, fmt.Errorf("remote input exceeds %d bytes", maxRemoteTaskInputBytes)
	}

	return remoteTaskInputDownload{
		OriginalFilename: remoteTaskInputFilename(parsed),
		MimeType:         strings.TrimSpace(resp.Header.Get("Content-Type")),
		Raw:              raw,
	}, nil
}

func remoteTaskInputFilename(parsed *neturl.URL) string {
	base := path.Base(parsed.Path)
	if base == "" || base == "." || base == "/" {
		return "remote-input"
	}
	if unescaped, err := neturl.PathUnescape(base); err == nil && strings.TrimSpace(unescaped) != "" {
		return unescaped
	}
	return base
}

func validateReusableAssetScope(scope domain.Scope, item domain.AssetWithVersion) error {
	if item.WorkspaceID != scope.WorkspaceID || item.ProjectID != scope.ProjectID {
		return fmt.Errorf("asset belongs to a different workspace/project")
	}
	if item.Version.Status != domain.VersionReady {
		return fmt.Errorf("asset current version is not ready")
	}
	if strings.TrimSpace(item.Version.FilePath) == "" {
		return fmt.Errorf("asset current version has no original file path")
	}
	if _, err := os.Stat(item.Version.FilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("asset original file is missing")
		}
		return fmt.Errorf("asset original file is not readable: %w", err)
	}
	return nil
}

func resolvedTaskInputFileFromStored(stored storage.StoredTaskInputFile) resolvedTaskInputFile {
	return resolvedTaskInputFile{
		InputFileID: stored.InputFileID,
		Kind:        stored.Kind,
		FilePath:    stored.FilePath,
		MimeType:    stored.MimeType,
		Width:       stored.Width,
		Height:      stored.Height,
	}
}

func resolvedTaskInputFileFromAsset(item domain.AssetWithVersion, kind string) resolvedTaskInputFile {
	return resolvedTaskInputFile{
		Kind:     kind,
		FilePath: item.Version.FilePath,
		MimeType: item.Version.MimeType,
		Width:    item.Version.Width,
		Height:   item.Version.Height,
	}
}

func applyStoredReferenceImage(ref *domain.ReferenceImage, downloadURL string, stored storage.StoredTaskInputFile, updateURL bool, source string) {
	if ref.ID == "" {
		ref.ID = stored.InputFileID
	}
	ref.InputFileID = stored.InputFileID
	if updateURL || ref.URL == "" {
		ref.URL = downloadURL
	}
	if ref.Source == "" {
		ref.Source = source
	}
	ref.MimeType = stored.MimeType
	ref.Width = stored.Width
	ref.Height = stored.Height
}

func applyAssetReferenceImage(ref *domain.ReferenceImage, item domain.AssetWithVersion, assetURL string) {
	if ref.ID == "" {
		ref.ID = item.ID
	}
	if ref.URL == "" {
		ref.URL = assetURL
	}
	if ref.Source == "" {
		ref.Source = taskInputSourceAssetReuse
	}
	ref.MimeType = item.Version.MimeType
	ref.Width = item.Version.Width
	ref.Height = item.Version.Height
}

func applyStoredMaskImage(mask *domain.MaskImage, downloadURL string, stored storage.StoredTaskInputFile, updateURL bool, source string) {
	if mask.ID == "" {
		mask.ID = stored.InputFileID
	}
	mask.InputFileID = stored.InputFileID
	if updateURL || mask.URL == "" {
		mask.URL = downloadURL
	}
	if mask.Source == "" {
		mask.Source = source
	}
	mask.MimeType = stored.MimeType
	mask.Width = stored.Width
	mask.Height = stored.Height
}

func applyAssetMaskImage(mask *domain.MaskImage, item domain.AssetWithVersion, assetURL string) {
	if mask.ID == "" {
		mask.ID = item.ID
	}
	if mask.URL == "" {
		mask.URL = assetURL
	}
	if mask.Source == "" {
		mask.Source = taskInputSourceAssetReuse
	}
	mask.MimeType = item.Version.MimeType
	mask.Width = item.Version.Width
	mask.Height = item.Version.Height
}
