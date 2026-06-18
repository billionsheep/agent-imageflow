package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/config"
	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/provider"
	"github.com/billionsheep/agent-imageflow/internal/queue"
	"github.com/billionsheep/agent-imageflow/internal/storage"
	"github.com/billionsheep/agent-imageflow/internal/store"
)

type Service struct {
	cfg      config.Config
	store    *store.PostgresStore
	queue    *queue.RedisQueue
	storage  storage.LocalStorage
	provider provider.MockProvider
}

func NewService(cfg config.Config, st *store.PostgresStore, q *queue.RedisQueue, fs storage.LocalStorage) *Service {
	return &Service{
		cfg:      cfg,
		store:    st,
		queue:    q,
		storage:  fs,
		provider: provider.MockProvider{},
	}
}

func (s *Service) Queue() *queue.RedisQueue {
	return s.queue
}

func (s *Service) CreateTask(ctx context.Context, scope domain.Scope, req domain.CreateTaskRequest) (domain.TaskResponse, error) {
	if err := s.store.CheckScope(ctx, scope); err != nil {
		return domain.TaskResponse{}, err
	}
	normalized, structured, inputHash, err := s.normalizeTaskRequest(scope, req)
	if err != nil {
		return domain.TaskResponse{}, err
	}

	if existing, existingHash, found, err := s.store.FindTaskByIdempotency(ctx, scope, normalized.IdempotencyKey); err != nil {
		return domain.TaskResponse{}, err
	} else if found {
		if existingHash != inputHash {
			return domain.TaskResponse{}, fmt.Errorf("idempotency_conflict: same key was used with different input")
		}
		return s.GetTask(ctx, existing.ID)
	}

	now := time.Now().UTC()
	task := domain.Task{
		ID:                  domain.NewID("task"),
		WorkspaceID:         scope.WorkspaceID,
		ProjectID:           scope.ProjectID,
		CampaignID:          scope.CampaignID,
		IdempotencyKey:      normalized.IdempotencyKey,
		Title:               normalized.Title,
		Purpose:             normalized.Purpose,
		Prompt:              normalized.Prompt,
		NegativePrompt:      normalized.NegativePrompt,
		StylePreset:         normalized.StylePreset,
		AspectRatio:         normalized.AspectRatio,
		OutputFormat:        normalized.OutputFormat,
		StructuredInputJSON: structured,
		Provider:            normalized.Provider,
		Status:              domain.TaskQueued,
		RequestedCount:      normalized.RequestedCount,
		CreatedBy:           "local-user",
		TraceID:             domain.NewID("trace"),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := s.store.InsertTask(ctx, task, inputHash, normalized.ReviewRequired); err != nil {
		return domain.TaskResponse{}, err
	}
	if err := s.queue.Enqueue(ctx, task.ID); err != nil {
		_ = s.store.MarkTaskEnqueueFailed(ctx, task.ID, err)
		task.Status = domain.TaskEnqueueFailed
		message := err.Error()
		code := "enqueue_failed"
		task.ErrorCode = &code
		task.ErrorMessage = &message
	}
	return s.taskResponse(ctx, task)
}

func (s *Service) GetTask(ctx context.Context, taskID string) (domain.TaskResponse, error) {
	task, err := s.store.GetTask(ctx, taskID)
	if err != nil {
		return domain.TaskResponse{}, err
	}
	return s.taskResponse(ctx, task)
}

func (s *Service) ListAssets(ctx context.Context, projectID, campaignID string) ([]domain.AssetResponse, error) {
	items, err := s.store.ListAssetsByCampaign(ctx, projectID, campaignID)
	if err != nil {
		return nil, err
	}
	responses := make([]domain.AssetResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, s.assetResponse(item))
	}
	return responses, nil
}

func (s *Service) GetAsset(ctx context.Context, assetID string) (domain.AssetResponse, error) {
	item, err := s.store.GetAssetWithVersion(ctx, assetID)
	if err != nil {
		return domain.AssetResponse{}, err
	}
	return s.assetResponse(item), nil
}

func (s *Service) GetAssetFile(ctx context.Context, assetID, kind string) (string, string, error) {
	item, err := s.store.GetAssetWithVersion(ctx, assetID)
	if err != nil {
		return "", "", err
	}
	if item.Version.Status != domain.VersionReady {
		return "", "", fmt.Errorf("asset version is not ready")
	}
	switch kind {
	case "original":
		return item.Version.FilePath, item.Version.MimeType, nil
	case "thumbnail":
		return item.Version.ThumbnailPath, "image/png", nil
	default:
		return "", "", fmt.Errorf("unknown file kind %q", kind)
	}
}

func (s *Service) ReviewAsset(ctx context.Context, assetID, action string) (domain.AssetResponse, error) {
	item, err := s.store.ReviewAsset(ctx, assetID, action, "local-user", "")
	if err != nil {
		return domain.AssetResponse{}, err
	}
	return s.assetResponse(item), nil
}

func (s *Service) ProcessTask(ctx context.Context, taskID string) error {
	task, err := s.store.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	if task.Status == domain.TaskCompleted || task.Status == domain.TaskFailed || task.Status == domain.TaskPartiallyCompleted {
		return nil
	}

	attemptID, _, err := s.store.CreateAttempt(ctx, task)
	if err != nil {
		return err
	}
	started := time.Now()
	if err := s.store.UpdateTaskStatus(ctx, task.ID, domain.TaskRunning, nil, nil); err != nil {
		return err
	}
	task.Status = domain.TaskRunning

	result, err := s.provider.Generate(ctx, task)
	if err != nil {
		code := "provider_error"
		msg := err.Error()
		_ = s.store.FinishAttempt(ctx, attemptID, domain.AttemptFailed, result, started, &code, &msg)
		_ = s.store.UpdateTaskStatus(ctx, task.ID, domain.TaskFailed, &code, &msg)
		return err
	}

	successCount := 0
	var processErr error
	for _, file := range result.Files {
		assetID := domain.NewID("asset")
		versionID := domain.NewID("ver")
		stored, err := s.storage.StoreGeneratedFile(ctx, task, assetID, versionID, file)
		if err != nil {
			processErr = err
			continue
		}
		if _, err := s.store.InsertAssetWithVersion(ctx, task, file, stored); err != nil {
			processErr = err
			continue
		}
		successCount++
	}

	status := domain.TaskCompleted
	var code *string
	var message *string
	if successCount == 0 {
		status = domain.TaskFailed
		c := "asset_processing_failed"
		m := "no generated files were registered"
		if processErr != nil {
			m = processErr.Error()
		}
		code = &c
		message = &m
	} else if successCount < task.RequestedCount || processErr != nil {
		status = domain.TaskPartiallyCompleted
		c := "partial_success"
		m := fmt.Sprintf("%d of %d requested images are ready", successCount, task.RequestedCount)
		if processErr != nil {
			m += ": " + processErr.Error()
		}
		code = &c
		message = &m
	}

	attemptStatus := domain.AttemptCompleted
	if status == domain.TaskFailed {
		attemptStatus = domain.AttemptFailed
	}
	if err := s.store.FinishAttempt(ctx, attemptID, attemptStatus, result, started, code, message); err != nil {
		return err
	}
	return s.store.UpdateTaskStatus(ctx, task.ID, status, code, message)
}

func (s *Service) taskResponse(ctx context.Context, task domain.Task) (domain.TaskResponse, error) {
	assets, err := s.store.ListAssetsByTask(ctx, task.ID)
	if err != nil {
		return domain.TaskResponse{}, err
	}
	response := domain.TaskResponse{
		Task:     task,
		AssetIDs: make([]string, 0, len(assets)),
		Assets:   make([]domain.AssetListEntry, 0, len(assets)),
	}
	for _, item := range assets {
		response.AssetIDs = append(response.AssetIDs, item.ID)
		response.Assets = append(response.Assets, domain.AssetListEntry{
			AssetID:      item.ID,
			Status:       item.Status,
			ThumbnailURL: s.assetURL(item.ID, "thumbnail"),
			MetadataURL:  s.assetURL(item.ID, ""),
		})
	}
	return response, nil
}

func (s *Service) assetResponse(item domain.AssetWithVersion) domain.AssetResponse {
	return domain.AssetResponse{
		AssetID:        item.ID,
		WorkspaceID:    item.WorkspaceID,
		ProjectID:      item.ProjectID,
		CampaignID:     item.CampaignID,
		TaskID:         item.TaskID,
		CurrentVersion: item.Version.Version,
		Status:         item.Status,
		Hash:           item.Version.Hash,
		Provider:       item.Version.Provider,
		Model:          item.Version.Model,
		Prompt:         item.Version.Prompt,
		ParametersJSON: item.Version.ParametersJSON,
		Delivery: domain.DeliveryInfo{
			LocalPath:    item.Version.FilePath,
			DownloadURL:  s.assetURL(item.ID, "original"),
			ThumbnailURL: s.assetURL(item.ID, "thumbnail"),
			MetadataURL:  s.assetURL(item.ID, ""),
		},
		CreatedAt: item.CreatedAt,
	}
}

func (s *Service) assetURL(assetID, suffix string) string {
	if suffix == "" {
		return s.cfg.PublicBaseURL + "/api/assets/" + assetID
	}
	return s.cfg.PublicBaseURL + "/api/assets/" + assetID + "/" + suffix
}

func (s *Service) normalizeTaskRequest(scope domain.Scope, req domain.CreateTaskRequest) (domain.CreateTaskRequest, []byte, string, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Purpose = strings.TrimSpace(req.Purpose)
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.NegativePrompt = strings.TrimSpace(req.NegativePrompt)
	req.StylePreset = strings.TrimSpace(req.StylePreset)
	req.AspectRatio = strings.TrimSpace(req.AspectRatio)
	req.OutputFormat = strings.TrimSpace(req.OutputFormat)
	req.Provider = strings.TrimSpace(req.Provider)
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if req.Title == "" {
		req.Title = "Untitled image task"
	}
	if req.Prompt == "" {
		return req, nil, "", fmt.Errorf("prompt is required")
	}
	if req.AspectRatio == "" {
		req.AspectRatio = "1:1"
	}
	if req.OutputFormat == "" {
		req.OutputFormat = "png"
	}
	if req.Provider == "" {
		req.Provider = s.cfg.DefaultProvider
	}
	if req.Provider != "mock" {
		return req, nil, "", fmt.Errorf("provider %q is not enabled in this slice", req.Provider)
	}
	if req.RequestedCount < 1 {
		req.RequestedCount = 1
	}
	if req.RequestedCount > 4 {
		req.RequestedCount = 4
	}
	if len(req.MetadataJSON) == 0 || !json.Valid(req.MetadataJSON) {
		req.MetadataJSON = []byte(`{}`)
	}

	structured, err := json.Marshal(map[string]any{
		"workspace_id":    scope.WorkspaceID,
		"project_id":      scope.ProjectID,
		"campaign_id":     scope.CampaignID,
		"idempotency_key": req.IdempotencyKey,
		"title":           req.Title,
		"purpose":         req.Purpose,
		"prompt":          req.Prompt,
		"negative_prompt": req.NegativePrompt,
		"style_preset":    req.StylePreset,
		"aspect_ratio":    req.AspectRatio,
		"output_format":   req.OutputFormat,
		"requested_count": req.RequestedCount,
		"provider":        req.Provider,
		"review_required": req.ReviewRequired,
		"metadata_json":   json.RawMessage(req.MetadataJSON),
	})
	if err != nil {
		return req, nil, "", err
	}
	hash := sha256.Sum256(structured)
	return req, structured, hex.EncodeToString(hash[:]), nil
}
