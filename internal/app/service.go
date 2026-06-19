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
	cfg           config.Config
	store         *store.PostgresStore
	queue         *queue.RedisQueue
	storage       storage.LocalStorage
	providers     map[string]provider.Adapter
	bestOfScorers map[string]bestOfScorer
}

func NewService(cfg config.Config, st *store.PostgresStore, q *queue.RedisQueue, fs storage.LocalStorage) *Service {
	providers := map[string]provider.Adapter{
		provider.MockProviderID: provider.MockProvider{},
	}
	falProvider := provider.NewFalProvider(provider.FalConfig{
		BaseURL:             cfg.FalBaseURL,
		RestBaseURL:         cfg.FalRestBaseURL,
		APIKey:              cfg.FalAPIKey,
		Model:               cfg.FalModel,
		TimeoutSeconds:      cfg.ProviderTimeoutSeconds,
		PollIntervalMs:      cfg.FalPollIntervalMs,
		StartTimeoutSeconds: cfg.ProviderTimeoutSeconds,
	})
	if falProvider.Configured() {
		providers[provider.FalProviderID] = falProvider
	}
	openAICompatible := provider.NewOpenAICompatibleProvider(provider.OpenAICompatibleConfig{
		BaseURL:        cfg.OpenAICompatibleBaseURL,
		APIKey:         cfg.OpenAICompatibleAPIKey,
		Model:          cfg.OpenAICompatibleModel,
		TimeoutSeconds: cfg.ProviderTimeoutSeconds,
	})
	if openAICompatible.Configured() {
		providers[provider.OpenAICompatibleProviderID] = openAICompatible
	}
	bestOfScorers := map[string]bestOfScorer{
		domain.BestOfStrategyLocalMetadata: localMetadataBestOfScorer{},
	}
	httpJudge := newHTTPJudgeBestOfScorer(cfg.BestOfHTTPScorerURL, cfg.BestOfHTTPScorerAPIKey, cfg.BestOfHTTPScorerTimeout)
	if httpJudge.Configured() {
		bestOfScorers[domain.BestOfStrategyHTTPJudge] = httpJudge
	}
	return &Service{
		cfg:           cfg,
		store:         st,
		queue:         q,
		storage:       fs,
		providers:     providers,
		bestOfScorers: bestOfScorers,
	}
}

func (s *Service) Queue() *queue.RedisQueue {
	return s.queue
}

func (s *Service) CreateTask(ctx context.Context, scope domain.Scope, req domain.CreateTaskRequest) (domain.TaskResponse, error) {
	if err := s.store.CheckScope(ctx, scope); err != nil {
		return domain.TaskResponse{}, err
	}
	projectProfile := domain.QualityProfile{}
	if req.UseProjectQualityProfile {
		var err error
		projectProfile, err = s.store.GetProjectQualityProfile(ctx, scope.WorkspaceID, scope.ProjectID)
		if err != nil {
			return domain.TaskResponse{}, err
		}
	}
	normalized, structured, inputHash, err := s.normalizeTaskRequest(ctx, scope, req, projectProfile)
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
		SelectionMode:       normalized.SelectionMode,
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

func (s *Service) GetProjectQualityProfile(ctx context.Context, workspaceID, projectID string) (domain.ProjectQualityProfileResponse, error) {
	profile, err := s.store.GetProjectQualityProfile(ctx, workspaceID, projectID)
	if err != nil {
		return domain.ProjectQualityProfileResponse{}, err
	}
	profile, err = normalizeQualityProfile(profile)
	if err != nil {
		return domain.ProjectQualityProfileResponse{}, err
	}
	return domain.ProjectQualityProfileResponse{
		WorkspaceID:    workspaceID,
		ProjectID:      projectID,
		QualityProfile: profile,
	}, nil
}

func (s *Service) UpdateProjectQualityProfile(ctx context.Context, workspaceID, projectID string, profile domain.QualityProfile) (domain.ProjectQualityProfileResponse, error) {
	normalized, err := normalizeQualityProfile(profile)
	if err != nil {
		return domain.ProjectQualityProfileResponse{}, err
	}
	saved, err := s.store.UpdateProjectQualityProfile(ctx, workspaceID, projectID, normalized)
	if err != nil {
		return domain.ProjectQualityProfileResponse{}, err
	}
	return domain.ProjectQualityProfileResponse{
		WorkspaceID:    workspaceID,
		ProjectID:      projectID,
		QualityProfile: saved,
	}, nil
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
		return item.Version.ThumbnailPath, storage.MimeTypeForPath(item.Version.ThumbnailPath), nil
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

	attemptID, attemptNo, err := s.store.CreateAttempt(ctx, task)
	if err != nil {
		return err
	}
	started := time.Now()
	if err := s.store.UpdateTaskStatus(ctx, task.ID, domain.TaskRunning, nil, nil); err != nil {
		return err
	}
	task.Status = domain.TaskRunning

	adapter, ok := s.providers[task.Provider]
	if !ok {
		err := fmt.Errorf("provider %q is not enabled", task.Provider)
		code := "provider_not_enabled"
		msg := err.Error()
		_ = s.store.FinishAttempt(ctx, attemptID, domain.AttemptFailed, provider.Result{ErrorCode: code, ErrorMessage: msg}, started, &code, &msg, nil)
		_ = s.store.UpdateTaskStatus(ctx, task.ID, domain.TaskFailed, &code, &msg)
		return err
	}
	result, err := adapter.Generate(ctx, task)
	if err != nil {
		scheduled, retryErr := s.scheduleRetry(ctx, task.ID, attemptID, attemptNo, result, started, err)
		if retryErr != nil {
			return retryErr
		}
		if scheduled {
			return nil
		}
		code := "provider_error"
		msg := err.Error()
		_ = s.store.FinishAttempt(ctx, attemptID, domain.AttemptFailed, result, started, &code, &msg, nil)
		_ = s.store.UpdateTaskStatus(ctx, task.ID, domain.TaskFailed, &code, &msg)
		return err
	}

	successCount := 0
	var processErr error
	registered := make([]domain.AssetWithVersion, 0, len(result.Files))
	for _, file := range result.Files {
		assetID := domain.NewID("asset")
		versionID := domain.NewID("ver")
		stored, err := s.storage.StoreGeneratedFile(ctx, task, assetID, versionID, file)
		if err != nil {
			processErr = err
			continue
		}
		item, err := s.store.InsertAssetWithVersion(ctx, task, file, stored)
		if err != nil {
			processErr = err
			continue
		}
		registered = append(registered, item)
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
	if err := s.store.FinishAttempt(ctx, attemptID, attemptStatus, result, started, code, message, nil); err != nil {
		return err
	}
	if err := s.store.UpdateTaskStatus(ctx, task.ID, status, code, message); err != nil {
		return err
	}
	return s.autoSelectBestAsset(ctx, task, registered)
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
		MetadataJSON:   metadataJSONFromStructuredInput(item.TaskStructuredInputJSON),
		Delivery: domain.DeliveryInfo{
			LocalPath:    item.Version.FilePath,
			DownloadURL:  s.assetURL(item.ID, "original"),
			ThumbnailURL: s.assetURL(item.ID, "thumbnail"),
			MetadataURL:  s.assetURL(item.ID, ""),
		},
		CreatedAt: item.CreatedAt,
	}
}

func metadataJSONFromStructuredInput(raw json.RawMessage) json.RawMessage {
	var structured map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &structured) != nil {
		return json.RawMessage(`{}`)
	}
	metadata := structured["metadata_json"]
	if len(metadata) == 0 || !json.Valid(metadata) {
		return json.RawMessage(`{}`)
	}
	return metadata
}

func (s *Service) assetURL(assetID, suffix string) string {
	if suffix == "" {
		return s.cfg.PublicBaseURL + "/api/assets/" + assetID
	}
	return s.cfg.PublicBaseURL + "/api/assets/" + assetID + "/" + suffix
}

func (s *Service) normalizeTaskRequest(ctx context.Context, scope domain.Scope, req domain.CreateTaskRequest, projectProfile domain.QualityProfile) (domain.CreateTaskRequest, []byte, string, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Purpose = strings.TrimSpace(req.Purpose)
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.NegativePrompt = strings.TrimSpace(req.NegativePrompt)
	req.StylePreset = strings.TrimSpace(req.StylePreset)
	req.PromptTemplate = strings.TrimSpace(req.PromptTemplate)
	req.AspectRatio = strings.TrimSpace(req.AspectRatio)
	req.OutputFormat = strings.TrimSpace(req.OutputFormat)
	req.Provider = strings.TrimSpace(req.Provider)
	req.SelectionMode = strings.TrimSpace(req.SelectionMode)
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if req.Title == "" {
		req.Title = "Untitled image task"
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
	selectionMode, ok := domain.NormalizeSelectionMode(req.SelectionMode)
	if !ok {
		return req, nil, "", fmt.Errorf("unknown selection_mode %q", req.SelectionMode)
	}
	req.SelectionMode = selectionMode
	quality, err := applyQualityProfile(req, projectProfile)
	if err != nil {
		return req, nil, "", err
	}
	req = quality.Request
	if req.Prompt == "" {
		return req, nil, "", fmt.Errorf("prompt is required")
	}
	req.MaskImage = normalizeMaskImage(req.MaskImage)
	if err := s.validateBestOfConfig(req.SelectionMode, req.BestOfConfig); err != nil {
		return req, nil, "", err
	}
	req, resolvedInputFiles, err := s.resolveTaskInputFiles(ctx, scope, req)
	if err != nil {
		return req, nil, "", err
	}
	if _, ok := s.providers[req.Provider]; !ok {
		return req, nil, "", fmt.Errorf("provider %q is not enabled; configure it or use %q", req.Provider, provider.MockProviderID)
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
		"workspace_id":                scope.WorkspaceID,
		"project_id":                  scope.ProjectID,
		"campaign_id":                 scope.CampaignID,
		"idempotency_key":             req.IdempotencyKey,
		"title":                       req.Title,
		"purpose":                     req.Purpose,
		"prompt":                      req.Prompt,
		"negative_prompt":             req.NegativePrompt,
		"style_preset":                req.StylePreset,
		"prompt_template":             req.PromptTemplate,
		"template_variables":          req.TemplateVariables,
		"reference_images":            req.ReferenceImages,
		"mask_image":                  req.MaskImage,
		"best_of_config":              req.BestOfConfig,
		"resolved_input_files":        resolvedInputFiles,
		"generation_config":           json.RawMessage(req.GenerationConfig),
		"use_project_quality_profile": req.UseProjectQualityProfile,
		"aspect_ratio":                req.AspectRatio,
		"output_format":               req.OutputFormat,
		"requested_count":             req.RequestedCount,
		"provider":                    req.Provider,
		"selection_mode":              req.SelectionMode,
		"review_required":             req.ReviewRequired,
		"metadata_json":               json.RawMessage(req.MetadataJSON),
	})
	if err != nil {
		return req, nil, "", err
	}
	hash := sha256.Sum256(structured)
	return req, structured, hex.EncodeToString(hash[:]), nil
}
