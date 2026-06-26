package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/config"
	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/provider"
	"github.com/billionsheep/agent-imageflow/internal/queue"
	"github.com/billionsheep/agent-imageflow/internal/storage"
	"github.com/billionsheep/agent-imageflow/internal/store"
)

type Service struct {
	cfg              config.Config
	store            *store.PostgresStore
	queue            *queue.RedisQueue
	storage          storage.LocalStorage
	providers        map[string]provider.Adapter
	providerLimiters map[string]chan struct{}
	bestOfScorers    map[string]bestOfScorer
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
		BaseURL:                      cfg.OpenAICompatibleBaseURL,
		APIKey:                       cfg.OpenAICompatibleAPIKey,
		Model:                        cfg.OpenAICompatibleModel,
		TimeoutSeconds:               cfg.ProviderTimeoutSeconds,
		ConnectTimeoutSeconds:        cfg.OpenAICompatibleConnectTimeout,
		ResponseHeaderTimeoutSeconds: cfg.OpenAICompatibleHeaderTimeout,
		TotalTimeoutSeconds:          cfg.OpenAICompatibleTotalTimeout,
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
		cfg:              cfg,
		store:            st,
		queue:            q,
		storage:          fs,
		providers:        providers,
		providerLimiters: newProviderLimiters(cfg),
		bestOfScorers:    bestOfScorers,
	}
}

func newProviderLimiters(cfg config.Config) map[string]chan struct{} {
	limiters := map[string]chan struct{}{}
	if cfg.OpenAICompatibleMaxConcurrency > 0 {
		limiters[provider.OpenAICompatibleProviderID] = make(chan struct{}, cfg.OpenAICompatibleMaxConcurrency)
	}
	if cfg.FalMaxConcurrency > 0 {
		limiters[provider.FalProviderID] = make(chan struct{}, cfg.FalMaxConcurrency)
	}
	return limiters
}

func (s *Service) Queue() *queue.RedisQueue {
	return s.queue
}

func (s *Service) CreateTask(ctx context.Context, scope domain.Scope, req domain.CreateTaskRequest) (domain.TaskResponse, error) {
	if err := s.store.CheckScope(ctx, scope); err != nil {
		return domain.TaskResponse{}, err
	}
	providerProfile, err := s.store.GetProjectProviderProfile(ctx, scope.WorkspaceID, scope.ProjectID)
	if err != nil {
		return domain.TaskResponse{}, err
	}
	projectProfile := domain.QualityProfile{}
	if req.UseProjectQualityProfile {
		projectProfile, err = s.store.GetProjectQualityProfile(ctx, scope.WorkspaceID, scope.ProjectID)
		if err != nil {
			return domain.TaskResponse{}, err
		}
	}
	normalized, structured, inputHash, err := s.normalizeTaskRequest(ctx, scope, req, projectProfile, providerProfile)
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

func (s *Service) ListTaskAttempts(ctx context.Context, taskID string) (domain.TaskAttemptsResponse, error) {
	if _, err := s.store.GetTask(ctx, taskID); err != nil {
		return domain.TaskAttemptsResponse{}, err
	}
	attempts, err := s.store.ListTaskAttempts(ctx, taskID)
	if err != nil {
		return domain.TaskAttemptsResponse{}, err
	}
	return domain.TaskAttemptsResponse{
		TaskID:   taskID,
		Attempts: attempts,
	}, nil
}

func (s *Service) GetBatchProgress(ctx context.Context, query domain.BatchProgressQuery) (domain.BatchProgressResponse, error) {
	return s.store.GetBatchProgress(ctx, query)
}

func (s *Service) GetBatchStorySummary(ctx context.Context, query domain.BatchStorySummaryQuery) (domain.BatchStorySummaryResponse, error) {
	return s.store.GetBatchStorySummary(ctx, query)
}

func (s *Service) GetBatchManifest(ctx context.Context, query domain.BatchManifestQuery) (domain.BatchManifestResponse, error) {
	summary, err := s.GetBatchStorySummary(ctx, query.BatchStorySummaryQuery)
	if err != nil {
		return domain.BatchManifestResponse{}, err
	}
	return buildBatchManifest(summary, query), nil
}

func buildBatchManifest(summary domain.BatchStorySummaryResponse, query domain.BatchManifestQuery) domain.BatchManifestResponse {
	manifest := domain.BatchManifestResponse{
		GeneratedAt:     time.Now().UTC(),
		ProjectID:       summary.ProjectID,
		CampaignID:      summary.CampaignID,
		SessionID:       summary.SessionID,
		BatchID:         summary.BatchID,
		Source:          summary.Source,
		StoryID:         summary.StoryID,
		SelectedOnly:    query.SelectedOnly,
		IncludeRejected: query.IncludeRejected,
		Stories:         summary.Stories,
		Tasks:           []domain.BatchManifestTask{},
		Assets:          []domain.BatchManifestAsset{},
		Scenes:          []domain.BatchManifestScene{},
	}
	manifest.Counts = summary.Counts
	manifest.Counts.AssetCount = 0
	manifest.Counts.GeneratedAssetCount = 0
	manifest.Counts.SelectedAssetCount = 0
	manifest.Counts.RejectedAssetCount = 0
	manifest.Counts.StoryCount = len(summary.Stories)
	manifest.Counts.SceneCount = len(summary.Scenes)
	manifest.Counts.SceneWithSelectedCount = 0
	manifest.Counts.SceneMissingSelectedCount = 0
	manifest.Counts.TaskCount = 0
	manifest.Counts.QueuedCount = 0
	manifest.Counts.RunningCount = 0
	manifest.Counts.SucceededCount = 0
	manifest.Counts.PartialCount = 0
	manifest.Counts.FailedCount = 0
	manifest.Counts.RetryingCount = 0
	manifest.Counts.AttemptCount = 0

	for _, scene := range summary.Scenes {
		if scene.PrimarySelectedAssetID != "" {
			manifest.Counts.SceneWithSelectedCount++
		} else {
			manifest.Counts.SceneMissingSelectedCount++
		}
		manifestScene := domain.BatchManifestScene{
			StoryID:                scene.StoryID,
			SceneID:                scene.SceneID,
			Status:                 scene.Status,
			TargetPath:             scene.TargetPath,
			LatestTaskID:           scene.LatestTaskID,
			PrimarySelectedAssetID: scene.PrimarySelectedAssetID,
			RegenerationCount:      scene.RegenerationCount,
			AssetIDs:               []string{},
			SelectedAssetIDs:       []string{},
			TaskIDs:                []string{},
			Continuity:             scene.Continuity,
			VisualContext:          scene.VisualContext,
		}
		for _, task := range scene.Tasks {
			manifest.Tasks = append(manifest.Tasks, domain.BatchManifestTask{
				TaskID:                task.TaskID,
				StoryID:               scene.StoryID,
				SceneID:               scene.SceneID,
				Status:                task.Status,
				AssetCount:            task.AssetCount,
				AttemptCount:          task.AttemptCount,
				Retrying:              task.Retrying,
				ErrorStage:            task.ErrorStage,
				ErrorCode:             task.ErrorCode,
				ErrorMessage:          task.ErrorMessage,
				CreatedAt:             task.CreatedAt,
				UpdatedAt:             task.UpdatedAt,
				RegeneratedFromTaskID: firstNonEmpty(task.RegeneratedFromTaskID, scene.RegeneratedFromTaskID),
				RegenerateNo:          task.RegenerateNo,
			})
			manifestScene.TaskIDs = append(manifestScene.TaskIDs, task.TaskID)
			addBatchManifestTaskCounts(&manifest.Counts, task)
		}
		for _, asset := range scene.Assets {
			if !batchManifestIncludesAsset(asset.Status, query.SelectedOnly, query.IncludeRejected) {
				continue
			}
			manifestAsset := domain.BatchManifestAsset{
				AssetID:       asset.AssetID,
				TaskID:        asset.TaskID,
				StoryID:       scene.StoryID,
				SceneID:       scene.SceneID,
				Status:        asset.Status,
				Provider:      asset.Provider,
				Model:         asset.Model,
				Prompt:        asset.Prompt,
				DownloadURL:   asset.DownloadURL,
				ThumbnailURL:  asset.ThumbnailURL,
				MetadataURL:   asset.MetadataURL,
				TargetPath:    firstNonEmpty(asset.TargetPath, scene.TargetPath),
				CreatedAt:     asset.CreatedAt,
				Continuity:    scene.Continuity,
				VisualContext: scene.VisualContext,
			}
			manifest.Assets = append(manifest.Assets, manifestAsset)
			manifestScene.AssetIDs = append(manifestScene.AssetIDs, asset.AssetID)
			if asset.Status == "selected" {
				manifestScene.SelectedAssetIDs = append(manifestScene.SelectedAssetIDs, asset.AssetID)
			}
			addBatchManifestAssetCounts(&manifest.Counts, asset.Status)
		}
		manifest.Scenes = append(manifest.Scenes, manifestScene)
	}
	return manifest
}

func addBatchManifestTaskCounts(counts *domain.BatchManifestCounts, task domain.BatchStorySummaryTask) {
	counts.TaskCount++
	counts.AttemptCount += task.AttemptCount
	if task.Retrying {
		counts.RetryingCount++
	}
	switch task.Status {
	case domain.TaskQueued:
		counts.QueuedCount++
	case domain.TaskRunning:
		counts.RunningCount++
	case domain.TaskCompleted:
		counts.SucceededCount++
	case domain.TaskPartiallyCompleted:
		counts.PartialCount++
	case domain.TaskFailed, domain.TaskEnqueueFailed:
		counts.FailedCount++
	}
}

func batchManifestIncludesAsset(status string, selectedOnly, includeRejected bool) bool {
	switch {
	case selectedOnly:
		return status == "selected"
	case includeRejected:
		return status == "generated" || status == "selected" || status == domain.AssetRejected
	default:
		return status == "generated" || status == "selected"
	}
}

func addBatchManifestAssetCounts(counts *domain.BatchManifestCounts, status string) {
	counts.AssetCount++
	switch status {
	case "generated":
		counts.GeneratedAssetCount++
	case "selected":
		counts.SelectedAssetCount++
	case domain.AssetRejected:
		counts.RejectedAssetCount++
	}
}

func (s *Service) RegenerateSceneTask(ctx context.Context, scope domain.Scope, req domain.SceneRegenerateRequest) (domain.SceneRegenerateResponse, error) {
	scope.ProjectID = strings.TrimSpace(scope.ProjectID)
	scope.CampaignID = strings.TrimSpace(scope.CampaignID)
	if scope.ProjectID == "" || scope.CampaignID == "" {
		return domain.SceneRegenerateResponse{}, fmt.Errorf("project_id and campaign_id are required")
	}
	if strings.TrimSpace(req.ProjectID) != "" && strings.TrimSpace(req.ProjectID) != scope.ProjectID {
		return domain.SceneRegenerateResponse{}, fmt.Errorf("request project_id does not match route project_id")
	}
	if strings.TrimSpace(req.CampaignID) != "" && strings.TrimSpace(req.CampaignID) != scope.CampaignID {
		return domain.SceneRegenerateResponse{}, fmt.Errorf("request campaign_id does not match route campaign_id")
	}

	warnings := []domain.SceneRegenerateWarning{{
		Code:    "selected_asset_preserved",
		Message: "Existing selected and rejected assets were not changed.",
	}}
	sourceTaskID := strings.TrimSpace(req.SourceTaskID)
	resolvedTaskSelector := "source_task_id"
	var source domain.Task
	var err error
	if sourceTaskID != "" {
		source, err = s.store.GetSceneRegenerationSourceTask(ctx, sourceTaskID)
	} else {
		if req.SceneIdentity == nil {
			return domain.SceneRegenerateResponse{}, fmt.Errorf("source_task_id or scene_identity is required")
		}
		identity := normalizeSceneIdentity(*req.SceneIdentity)
		if err := validateSceneIdentity(identity); err != nil {
			return domain.SceneRegenerateResponse{}, err
		}
		if identity.TaskSelector == "" {
			identity.TaskSelector = "latest"
		}
		if identity.TaskSelector != "latest" {
			return domain.SceneRegenerateResponse{}, fmt.Errorf("unsupported scene task_selector %q", identity.TaskSelector)
		}
		resolvedTaskSelector = identity.TaskSelector
		source, err = s.store.ResolveLatestSceneTask(ctx, scope.ProjectID, scope.CampaignID, identity)
		if err == nil {
			warnings = append(warnings, domain.SceneRegenerateWarning{
				Code:    "scene_identity_resolved",
				Message: "scene_identity was resolved to latest task " + source.ID + ".",
			})
		}
	}
	if err != nil {
		return domain.SceneRegenerateResponse{}, err
	}
	if source.ProjectID != scope.ProjectID || source.CampaignID != scope.CampaignID {
		return domain.SceneRegenerateResponse{}, fmt.Errorf("source task does not belong to project_id=%s campaign_id=%s", scope.ProjectID, scope.CampaignID)
	}
	scope.WorkspaceID = source.WorkspaceID

	sourceIdentity, err := sceneIdentityFromTask(source)
	if err != nil {
		return domain.SceneRegenerateResponse{}, err
	}
	sourceIdentity.TaskSelector = resolvedTaskSelector
	count, err := s.store.CountSceneRegenerations(ctx, scope.ProjectID, scope.CampaignID, sourceIdentity)
	if err != nil {
		return domain.SceneRegenerateResponse{}, err
	}
	req.SourceTaskID = source.ID
	req.ProjectID = scope.ProjectID
	req.CampaignID = scope.CampaignID
	req.SceneIdentity = &sourceIdentity
	req.RegenerationNumber = count + 1
	if strings.TrimSpace(req.RequestSource) == "" {
		req.RequestSource = "rest"
	}

	built, err := buildSceneRegenerationCreateTaskRequest(source, req)
	if err != nil {
		return domain.SceneRegenerateResponse{}, err
	}
	task, err := s.CreateTask(ctx, scope, built.Request)
	if err != nil {
		return domain.SceneRegenerateResponse{}, err
	}
	return domain.SceneRegenerateResponse{
		TaskID:                      task.ID,
		Status:                      task.Status,
		RegeneratedFromTaskID:       source.ID,
		RegenerateNo:                req.RegenerationNumber,
		ProjectID:                   scope.ProjectID,
		CampaignID:                  scope.CampaignID,
		SessionID:                   built.SceneIdentity.SessionID,
		BatchID:                     built.SceneIdentity.BatchID,
		StoryID:                     built.SceneIdentity.StoryID,
		SceneID:                     built.SceneIdentity.SceneID,
		CopiedVisualContextSnapshot: built.CopiedVisualContextSnapshot,
		Warnings:                    warnings,
	}, nil
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

func (s *Service) GetProjectProviderProfile(ctx context.Context, workspaceID, projectID string) (domain.ProjectProviderProfileResponse, error) {
	profile, err := s.store.GetProjectProviderProfile(ctx, workspaceID, projectID)
	if err != nil {
		return domain.ProjectProviderProfileResponse{}, err
	}
	return domain.ProjectProviderProfileResponse{
		WorkspaceID:     workspaceID,
		ProjectID:       projectID,
		ProviderProfile: normalizeProjectProviderProfile(profile),
	}, nil
}

func (s *Service) UpdateProjectProviderProfile(ctx context.Context, workspaceID, projectID string, profile domain.ProjectProviderProfile) (domain.ProjectProviderProfileResponse, error) {
	normalized := normalizeProjectProviderProfile(profile)
	saved, err := s.store.UpdateProjectProviderProfile(ctx, workspaceID, projectID, normalized)
	if err != nil {
		return domain.ProjectProviderProfileResponse{}, err
	}
	return domain.ProjectProviderProfileResponse{
		WorkspaceID:     workspaceID,
		ProjectID:       projectID,
		ProviderProfile: normalizeProjectProviderProfile(saved),
	}, nil
}

func (s *Service) ListAssets(ctx context.Context, query domain.AssetListQuery) ([]domain.AssetResponse, error) {
	query = normalizeAssetListQuery(query)
	items, err := s.store.ListAssetsByCampaign(ctx, query)
	if err != nil {
		return nil, err
	}
	responses := make([]domain.AssetResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, s.assetResponse(item))
	}
	return responses, nil
}

func (s *Service) ListRecentAssets(ctx context.Context, query domain.AssetListQuery) ([]domain.AssetResponse, error) {
	query = normalizeAssetListQuery(query)
	items, err := s.store.ListRecentAssets(ctx, query)
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

func (s *Service) GetAssetMetadata(ctx context.Context, assetID string) (domain.AssetMetadataResponse, error) {
	item, err := s.store.GetAssetWithVersion(ctx, assetID)
	if err != nil {
		return domain.AssetMetadataResponse{}, err
	}
	return s.assetMetadataResponse(item), nil
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

func (s *Service) ArchiveAsset(ctx context.Context, assetID string) (domain.AssetResponse, error) {
	item, err := s.store.LifecycleAsset(ctx, assetID, "archive", "local-user", "archive asset")
	if err != nil {
		return domain.AssetResponse{}, err
	}
	return s.assetResponse(item), nil
}

func (s *Service) RestoreAsset(ctx context.Context, assetID string) (domain.AssetResponse, error) {
	item, err := s.store.LifecycleAsset(ctx, assetID, "restore", "local-user", "restore archived asset")
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
	metrics := attemptBaseMetrics(task, started, attemptNo)
	if err := s.store.UpdateTaskStatus(ctx, task.ID, domain.TaskRunning, nil, nil); err != nil {
		return err
	}
	task.Status = domain.TaskRunning

	adapter, ok := s.providers[task.Provider]
	if !ok {
		err := fmt.Errorf("provider %q is not enabled", task.Provider)
		code := "provider_not_enabled"
		msg := err.Error()
		metrics.ErrorStage = "provider_lookup"
		_ = s.store.FinishAttempt(ctx, attemptID, domain.AttemptFailed, provider.Result{ErrorCode: code, ErrorMessage: msg}, started, metrics, &code, &msg, nil)
		_ = s.store.UpdateTaskStatus(ctx, task.ID, domain.TaskFailed, &code, &msg)
		return err
	}
	result, err := s.generateWithProviderLimit(ctx, task, adapter)
	metrics = mergeAttemptMetrics(metrics, result.Metrics)
	if err != nil {
		if len(result.Files) == 0 {
			scheduled, retryErr := s.scheduleRetry(ctx, task.ID, attemptID, attemptNo, result, started, metrics, err)
			if retryErr != nil {
				return retryErr
			}
			if scheduled {
				return nil
			}
			code := "provider_error"
			msg := err.Error()
			_ = s.store.FinishAttempt(ctx, attemptID, domain.AttemptFailed, result, started, metrics, &code, &msg, nil)
			_ = s.store.UpdateTaskStatus(ctx, task.ID, domain.TaskFailed, &code, &msg)
			return err
		}
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
			if metrics.ErrorStage == "" {
				metrics.ErrorStage = "store"
			}
			continue
		}
		metrics.StoreMs += stored.StoreMs
		metrics.ThumbnailMs += stored.ThumbnailMs
		item, err := s.store.InsertAssetWithVersion(ctx, task, file, stored)
		if err != nil {
			processErr = err
			if metrics.ErrorStage == "" {
				metrics.ErrorStage = "store"
			}
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
		} else if err != nil {
			c = firstNonEmpty(result.ErrorCode, "provider_error")
			m = err.Error()
		}
		code = &c
		message = &m
	} else if successCount < task.RequestedCount || processErr != nil || err != nil {
		status = domain.TaskPartiallyCompleted
		c := firstNonEmpty(result.ErrorCode, "partial_success")
		m := fmt.Sprintf("%d of %d requested images are ready", successCount, task.RequestedCount)
		if processErr != nil {
			m += ": " + processErr.Error()
		} else if err != nil {
			m += ": " + err.Error()
		}
		code = &c
		message = &m
	}

	attemptStatus := domain.AttemptCompleted
	if status == domain.TaskFailed {
		attemptStatus = domain.AttemptFailed
	}
	if err := s.store.FinishAttempt(ctx, attemptID, attemptStatus, result, started, metrics, code, message, nil); err != nil {
		return err
	}
	if err := s.store.UpdateTaskStatus(ctx, task.ID, status, code, message); err != nil {
		return err
	}
	return s.autoSelectBestAsset(ctx, task, registered)
}

func (s *Service) generateWithProviderLimit(ctx context.Context, task domain.Task, adapter provider.Adapter) (provider.Result, error) {
	maxPerRequest := effectiveTaskProviderMaxN(task)
	if task.RequestedCount > maxPerRequest {
		return s.generateSplitProviderRequests(ctx, task, adapter, maxPerRequest)
	}
	return s.generateOneProviderRequest(ctx, task, adapter)
}

func (s *Service) generateSplitProviderRequests(ctx context.Context, task domain.Task, adapter provider.Adapter, maxPerRequest int) (provider.Result, error) {
	splitCounts := providerSplitCounts(task.RequestedCount, maxPerRequest)
	type splitResult struct {
		result provider.Result
		err    error
	}
	results := make([]splitResult, len(splitCounts))
	var wg sync.WaitGroup
	for index, count := range splitCounts {
		index := index
		count := count
		wg.Add(1)
		go func() {
			defer wg.Done()
			subtask := task
			subtask.RequestedCount = count
			result, err := s.generateOneProviderRequest(ctx, subtask, adapter)
			results[index] = splitResult{result: result, err: err}
		}()
	}
	wg.Wait()

	slotOffset := 0
	var firstErr error
	var combined provider.Result
	combined.Status = "received"
	for _, item := range results {
		combined = combineProviderResults(combined, item.result, slotOffset)
		slotOffset += len(item.result.Files)
		if item.err != nil && firstErr == nil {
			firstErr = item.err
		}
	}
	if firstErr != nil {
		if len(combined.Files) > 0 {
			combined.Status = "partially_succeeded"
		} else {
			combined.Status = "failed"
		}
		return combined, firstErr
	}
	combined.Status = "succeeded"
	return combined, nil
}

func providerSplitCounts(requestedCount, maxPerRequest int) []int {
	if requestedCount < 1 {
		return nil
	}
	if maxPerRequest < 1 {
		maxPerRequest = 1
	}
	counts := make([]int, 0, (requestedCount+maxPerRequest-1)/maxPerRequest)
	remaining := requestedCount
	for remaining > 0 {
		count := maxPerRequest
		if count > remaining {
			count = remaining
		}
		counts = append(counts, count)
		remaining -= count
	}
	return counts
}

func (s *Service) generateOneProviderRequest(ctx context.Context, task domain.Task, adapter provider.Adapter) (provider.Result, error) {
	limiter := s.providerLimiters[task.Provider]
	if limiter == nil {
		return generateOneProviderRequest(ctx, task, adapter)
	}
	select {
	case limiter <- struct{}{}:
		defer func() { <-limiter }()
		return generateOneProviderRequest(ctx, task, adapter)
	case <-ctx.Done():
		return provider.Result{
			Status:       "failed",
			ErrorCode:    "provider_backpressure_canceled",
			ErrorMessage: ctx.Err().Error(),
			Metrics: domain.AttemptMetrics{
				ErrorStage: "provider_backpressure",
			},
		}, ctx.Err()
	}
}

func generateOneProviderRequest(ctx context.Context, task domain.Task, adapter provider.Adapter) (provider.Result, error) {
	started := time.Now()
	result, err := adapter.Generate(ctx, task)
	if result.Metrics.ProviderTotalMs <= 0 {
		result.Metrics.ProviderTotalMs = time.Since(started).Milliseconds()
	}
	if err != nil && result.Metrics.ErrorStage == "" {
		result.Metrics.ErrorStage = "provider_request"
	}
	return result, err
}

func combineProviderResults(combined provider.Result, next provider.Result, slotOffset int) provider.Result {
	if combined.ProviderRequestID == "" {
		combined.ProviderRequestID = next.ProviderRequestID
	} else if next.ProviderRequestID != "" && !strings.Contains(combined.ProviderRequestID, next.ProviderRequestID) {
		combined.ProviderRequestID += "," + next.ProviderRequestID
	}
	if len(next.RawResponse) > 0 {
		combined.RawResponse = next.RawResponse
	}
	if len(next.CostRaw) > 0 {
		combined.CostRaw = next.CostRaw
	}
	if next.ErrorCode != "" {
		combined.ErrorCode = next.ErrorCode
	}
	if next.ErrorMessage != "" {
		combined.ErrorMessage = next.ErrorMessage
	}
	for _, file := range next.Files {
		file.Slot += slotOffset
		combined.Files = append(combined.Files, file)
	}
	combined.Metrics = mergeAttemptMetrics(combined.Metrics, next.Metrics)
	return combined
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
			MetadataURL:  s.assetURL(item.ID, "metadata"),
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
		MetadataJSON:   domain.NormalizeMetadataJSON(metadataJSONFromStructuredInput(item.TaskStructuredInputJSON)),
		Delivery: domain.DeliveryInfo{
			LocalPath:    item.Version.FilePath,
			DownloadURL:  s.assetURL(item.ID, "original"),
			ThumbnailURL: s.assetURL(item.ID, "thumbnail"),
			MetadataURL:  s.assetURL(item.ID, "metadata"),
		},
		CreatedAt: item.CreatedAt,
	}
}

func (s *Service) assetMetadataResponse(item domain.AssetWithVersion) domain.AssetMetadataResponse {
	return domain.AssetMetadataResponse{
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
		MetadataJSON:   domain.NormalizeMetadataJSON(metadataJSONFromStructuredInput(item.TaskStructuredInputJSON)),
		Delivery: domain.PublicDeliveryInfo{
			DownloadURL:  s.assetURL(item.ID, "original"),
			ThumbnailURL: s.assetURL(item.ID, "thumbnail"),
			MetadataURL:  s.assetURL(item.ID, "metadata"),
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

func storyContextV1FromStructuredInput(raw json.RawMessage) json.RawMessage {
	var structured map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &structured) != nil {
		return nil
	}
	story := structured[storyContextV1MetadataKey]
	if len(story) == 0 || !json.Valid(story) {
		return nil
	}
	return story
}

func (s *Service) storyContinuityAssetSnapshot(ctx context.Context, assetID string) (storyContinuityAssetSnapshot, error) {
	item, err := s.store.GetAssetWithVersion(ctx, strings.TrimSpace(assetID))
	if err != nil {
		return storyContinuityAssetSnapshot{}, err
	}
	return storyContinuityAssetSnapshot{
		Scope: domain.Scope{
			WorkspaceID: item.WorkspaceID,
			ProjectID:   item.ProjectID,
			CampaignID:  item.CampaignID,
		},
		AssetStatus:  item.Status,
		MetadataJSON: domain.NormalizeMetadataJSON(metadataJSONFromStructuredInput(item.TaskStructuredInputJSON)),
	}, nil
}

func (s *Service) assetURL(assetID, suffix string) string {
	if suffix == "" {
		return s.cfg.PublicBaseURL + "/api/assets/" + assetID
	}
	return s.cfg.PublicBaseURL + "/api/assets/" + assetID + "/" + suffix
}

func normalizeAssetListQuery(query domain.AssetListQuery) domain.AssetListQuery {
	query.ProjectID = strings.TrimSpace(query.ProjectID)
	query.CampaignID = strings.TrimSpace(query.CampaignID)
	query.Status = normalizeAssetQueryStatus(query.Status)
	query.Provider = strings.TrimSpace(query.Provider)
	query.Model = strings.TrimSpace(query.Model)
	query.Source = strings.TrimSpace(query.Source)
	query.SessionID = strings.TrimSpace(query.SessionID)
	query.BatchID = strings.TrimSpace(query.BatchID)
	query.Keyword = strings.TrimSpace(query.Keyword)
	if query.Limit <= 0 {
		query.Limit = domain.DefaultAssetListLimit
	}
	if query.Limit > domain.MaxAssetListLimit {
		query.Limit = domain.MaxAssetListLimit
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	return query
}

func normalizeAssetQueryStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "generated":
		return domain.AssetDraft
	case "selected":
		return domain.AssetApproved
	default:
		return strings.TrimSpace(status)
	}
}

func normalizeProjectProviderProfile(profile domain.ProjectProviderProfile) domain.ProjectProviderProfile {
	profile.Provider = strings.TrimSpace(profile.Provider)
	profile.Model = strings.TrimSpace(profile.Model)
	profile.BaseURL = strings.TrimRight(strings.TrimSpace(profile.BaseURL), "/")
	profile.GenerationConfig = jsonOrEmptyObject(profile.GenerationConfig)
	switch strings.TrimSpace(profile.APIMode) {
	case "responses":
		profile.APIMode = "responses"
	default:
		profile.APIMode = "images"
	}
	if profile.PartialImages != nil {
		value := *profile.PartialImages
		if value < 0 {
			value = 0
		}
		if value > 3 {
			value = 3
		}
		profile.PartialImages = &value
	}
	if profile.MaxN <= 0 {
		profile.MaxN = 4
	}
	if profile.MaxN > 10 {
		profile.MaxN = 10
	}
	profile.PreferredResponseFormat = strings.TrimSpace(profile.PreferredResponseFormat)
	switch profile.PreferredResponseFormat {
	case "url":
		profile.PreferredResponseFormat = "url"
	case "b64_json":
		profile.PreferredResponseFormat = "b64_json"
	default:
		profile.PreferredResponseFormat = "url"
	}
	if profile.MaxConcurrency < 0 {
		profile.MaxConcurrency = 0
	}
	if profile.TimeoutSeconds < 0 {
		profile.TimeoutSeconds = 0
	}
	return profile
}

func jsonOrEmptyObject(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || !json.Valid(raw) {
		return json.RawMessage(`{}`)
	}
	var object map[string]any
	if json.Unmarshal(raw, &object) != nil || object == nil {
		return json.RawMessage(`{}`)
	}
	if len(object) == 0 {
		return json.RawMessage(`{}`)
	}
	normalized, err := json.Marshal(object)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return normalized
}

func attemptBaseMetrics(task domain.Task, started time.Time, attemptNo int) domain.AttemptMetrics {
	queueWait := started.Sub(task.CreatedAt).Milliseconds()
	if queueWait < 0 {
		queueWait = 0
	}
	retryCount := attemptNo - 1
	if retryCount < 0 {
		retryCount = 0
	}
	return domain.AttemptMetrics{
		QueueWaitMs: queueWait,
		RetryCount:  retryCount,
	}
}

func mergeAttemptMetrics(base domain.AttemptMetrics, next domain.AttemptMetrics) domain.AttemptMetrics {
	if next.QueueWaitMs > 0 {
		base.QueueWaitMs = next.QueueWaitMs
	}
	if next.ProviderFirstByteMs > 0 && (base.ProviderFirstByteMs == 0 || next.ProviderFirstByteMs < base.ProviderFirstByteMs) {
		base.ProviderFirstByteMs = next.ProviderFirstByteMs
	}
	if next.ProviderTotalMs > 0 {
		base.ProviderTotalMs += next.ProviderTotalMs
	}
	if next.ResponseDownloadMs > 0 {
		base.ResponseDownloadMs += next.ResponseDownloadMs
	}
	if next.StoreMs > 0 {
		base.StoreMs += next.StoreMs
	}
	if next.ThumbnailMs > 0 {
		base.ThumbnailMs += next.ThumbnailMs
	}
	if next.RetryCount > base.RetryCount {
		base.RetryCount = next.RetryCount
	}
	if strings.TrimSpace(next.ErrorStage) != "" {
		base.ErrorStage = strings.TrimSpace(next.ErrorStage)
	}
	if next.ResponseBytes > 0 {
		base.ResponseBytes += next.ResponseBytes
	}
	return base
}

func effectiveTaskProviderMaxN(task domain.Task) int {
	maxN := 4
	if strings.TrimSpace(task.Provider) == provider.OpenAICompatibleProviderID {
		maxN = 1
	}
	var input struct {
		GenerationConfig json.RawMessage               `json:"generation_config"`
		ProviderProfile  domain.ProjectProviderProfile `json:"provider_profile"`
	}
	if len(task.StructuredInputJSON) > 0 && json.Unmarshal(task.StructuredInputJSON, &input) == nil {
		if input.ProviderProfile.Enabled &&
			strings.TrimSpace(input.ProviderProfile.Provider) == task.Provider &&
			input.ProviderProfile.MaxN > 0 {
			maxN = input.ProviderProfile.MaxN
		}
		if value, ok := generationConfigInt(input.GenerationConfig, "max_n", 1, 10); ok {
			maxN = value
		}
	}
	if maxN < 1 {
		return 1
	}
	if maxN > 10 {
		return 10
	}
	return maxN
}

func generationConfigInt(raw json.RawMessage, key string, minValue, maxValue int) (int, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var config map[string]any
	if json.Unmarshal(raw, &config) != nil {
		return 0, false
	}
	value, ok := config[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		intValue := int(typed)
		if typed != float64(intValue) || intValue < minValue || intValue > maxValue {
			return 0, false
		}
		return intValue, true
	case int:
		if typed < minValue || typed > maxValue {
			return 0, false
		}
		return typed, true
	default:
		return 0, false
	}
}

func (s *Service) normalizeTaskRequest(ctx context.Context, scope domain.Scope, req domain.CreateTaskRequest, projectProfile domain.QualityProfile, providerProfile domain.ProjectProviderProfile) (domain.CreateTaskRequest, []byte, string, error) {
	providerProfile = normalizeProjectProviderProfile(providerProfile)
	req.Title = strings.TrimSpace(req.Title)
	req.Purpose = strings.TrimSpace(req.Purpose)
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.NegativePrompt = strings.TrimSpace(req.NegativePrompt)
	req.StylePreset = strings.TrimSpace(req.StylePreset)
	req.PromptTemplate = strings.TrimSpace(req.PromptTemplate)
	req.CharacterIDs = normalizeStringList(req.CharacterIDs)
	req.ReferenceAssetIDs = normalizeStringList(req.ReferenceAssetIDs)
	req.PromptRecipeID = strings.TrimSpace(req.PromptRecipeID)
	req.AspectRatio = strings.TrimSpace(req.AspectRatio)
	req.OutputFormat = strings.TrimSpace(req.OutputFormat)
	req.Provider = strings.TrimSpace(req.Provider)
	req.SelectionMode = strings.TrimSpace(req.SelectionMode)
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	req.MetadataJSON = domain.NormalizeMetadataJSON(req.MetadataJSON)
	if req.Title == "" {
		req.Title = "Untitled image task"
	}
	storyContext, _, err := storyContextV1FromMetadata(req.MetadataJSON)
	if err != nil {
		return req, nil, "", err
	}
	if storyContext != nil {
		visualContext, err := s.store.GetProjectVisualContext(ctx, scope.WorkspaceID, scope.ProjectID)
		if err != nil {
			return req, nil, "", err
		}
		visualContext, err = normalizeProjectVisualContext(visualContext, time.Now().UTC())
		if err != nil {
			return req, nil, "", err
		}
		req, err = applyStoryContextBindingsToRequest(req, visualContext, storyContext)
		if err != nil {
			return req, nil, "", err
		}
	}
	visualContext, err := s.expandProjectVisualContext(ctx, scope, req)
	if err != nil {
		return req, nil, "", err
	}
	req = visualContext.Request
	if req.AspectRatio == "" {
		req.AspectRatio = "1:1"
	}
	if req.OutputFormat == "" {
		req.OutputFormat = "png"
	}
	if req.Provider == "" && providerProfile.Enabled && strings.TrimSpace(providerProfile.Provider) != "" {
		req.Provider = providerProfile.Provider
	}
	if len(req.GenerationConfig) == 0 && providerProfile.Enabled && len(providerProfile.GenerationConfig) > 0 && json.Valid(providerProfile.GenerationConfig) {
		req.GenerationConfig = providerProfile.GenerationConfig
	}
	if !req.UseProjectQualityProfile && providerProfile.Enabled && providerProfile.UseProjectQualityProfile {
		req.UseProjectQualityProfile = true
		projectProfileFromStore, err := s.store.GetProjectQualityProfile(ctx, scope.WorkspaceID, scope.ProjectID)
		if err != nil {
			return req, nil, "", err
		}
		projectProfile = projectProfileFromStore
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
	referenceDiagnostics := buildReferenceParticipationDiagnostics(req, resolvedInputFiles)
	if _, ok := s.providers[req.Provider]; !ok {
		return req, nil, "", fmt.Errorf("provider %q is not enabled; configure it or use %q", req.Provider, provider.MockProviderID)
	}
	if req.RequestedCount < 1 {
		req.RequestedCount = 1
	}
	if req.RequestedCount > 10 {
		req.RequestedCount = 10
	}
	if storyContext != nil {
		updatedStory, updatedMetadata, err := prepareStoryContextV1ForTask(
			scope,
			req.MetadataJSON,
			req,
			resolvedInputFiles,
			referenceDiagnostics,
			func(assetID string) (storyContinuityAssetSnapshot, error) {
				return s.storyContinuityAssetSnapshot(ctx, assetID)
			},
		)
		if err != nil {
			return req, nil, "", err
		}
		if updatedStory != nil {
			storyContext = updatedStory
		}
		req.MetadataJSON = domain.NormalizeMetadataJSON(updatedMetadata)
	}

	structured, err := json.Marshal(map[string]any{
		"workspace_id":                     scope.WorkspaceID,
		"project_id":                       scope.ProjectID,
		"campaign_id":                      scope.CampaignID,
		"idempotency_key":                  req.IdempotencyKey,
		"title":                            req.Title,
		"purpose":                          req.Purpose,
		"prompt":                           req.Prompt,
		"negative_prompt":                  req.NegativePrompt,
		"style_preset":                     req.StylePreset,
		"prompt_template":                  req.PromptTemplate,
		"template_variables":               req.TemplateVariables,
		"reference_images":                 req.ReferenceImages,
		"character_ids":                    req.CharacterIDs,
		"reference_asset_ids":              req.ReferenceAssetIDs,
		"prompt_recipe_id":                 req.PromptRecipeID,
		"use_project_visual_context":       req.UseProjectVisualContext,
		"visual_context_snapshot":          visualContext.Snapshot,
		"mask_image":                       req.MaskImage,
		"best_of_config":                   req.BestOfConfig,
		"resolved_input_files":             resolvedInputFiles,
		"reference_asset_count":            referenceDiagnostics.ReferenceAssetCount,
		"reference_input_file_count":       referenceDiagnostics.ReferenceInputFileCount,
		"provider_reference_participation": referenceDiagnostics.ProviderReferenceParticipation,
		"provider_reference_sources":       referenceDiagnostics.ProviderReferenceSources,
		"provider_reference_mime_types":    referenceDiagnostics.ProviderReferenceMIMETypes,
		"generation_config":                json.RawMessage(req.GenerationConfig),
		"use_project_quality_profile":      req.UseProjectQualityProfile,
		"aspect_ratio":                     req.AspectRatio,
		"output_format":                    req.OutputFormat,
		"requested_count":                  req.RequestedCount,
		"provider":                         req.Provider,
		"provider_profile":                 providerProfile,
		"selection_mode":                   req.SelectionMode,
		"review_required":                  req.ReviewRequired,
		"story_context_v1":                 storyContext,
		"metadata_json":                    json.RawMessage(req.MetadataJSON),
	})
	if err != nil {
		return req, nil, "", err
	}
	hash := sha256.Sum256(structured)
	return req, structured, hex.EncodeToString(hash[:]), nil
}

type sceneRegenerationBuildResult struct {
	Request                     domain.CreateTaskRequest
	SceneIdentity               domain.SceneIdentity
	CopiedVisualContextSnapshot domain.SceneRegenerateVisualContextSnapshot
}

type sceneRegenerationSourceInput struct {
	Title                    string                  `json:"title"`
	Purpose                  string                  `json:"purpose"`
	Prompt                   string                  `json:"prompt"`
	NegativePrompt           string                  `json:"negative_prompt"`
	StylePreset              string                  `json:"style_preset"`
	PromptTemplate           string                  `json:"prompt_template"`
	TemplateVariables        map[string]any          `json:"template_variables"`
	ReferenceImages          []domain.ReferenceImage `json:"reference_images"`
	CharacterIDs             []string                `json:"character_ids"`
	ReferenceAssetIDs        []string                `json:"reference_asset_ids"`
	PromptRecipeID           string                  `json:"prompt_recipe_id"`
	UseProjectVisualContext  bool                    `json:"use_project_visual_context"`
	MaskImage                *domain.MaskImage       `json:"mask_image"`
	BestOfConfig             *domain.BestOfConfig    `json:"best_of_config"`
	GenerationConfig         json.RawMessage         `json:"generation_config"`
	UseProjectQualityProfile bool                    `json:"use_project_quality_profile"`
	AspectRatio              string                  `json:"aspect_ratio"`
	OutputFormat             string                  `json:"output_format"`
	RequestedCount           int                     `json:"requested_count"`
	Provider                 string                  `json:"provider"`
	SelectionMode            string                  `json:"selection_mode"`
	ReviewRequired           bool                    `json:"review_required"`
	MetadataJSON             json.RawMessage         `json:"metadata_json"`
}

func buildSceneRegenerationCreateTaskRequest(source domain.Task, req domain.SceneRegenerateRequest) (sceneRegenerationBuildResult, error) {
	input := sceneRegenerationInputFromTask(source)
	metadata, err := sceneRegenerationMetadataMap(input.MetadataJSON)
	if err != nil {
		return sceneRegenerationBuildResult{}, err
	}
	identity, err := sceneIdentityFromMetadata(metadata)
	if err != nil {
		return sceneRegenerationBuildResult{}, err
	}

	createReq := domain.CreateTaskRequest{
		Title:                    input.Title,
		Purpose:                  input.Purpose,
		Prompt:                   input.Prompt,
		NegativePrompt:           input.NegativePrompt,
		StylePreset:              input.StylePreset,
		PromptTemplate:           input.PromptTemplate,
		TemplateVariables:        input.TemplateVariables,
		ReferenceImages:          cloneReferenceImages(input.ReferenceImages),
		CharacterIDs:             append([]string(nil), input.CharacterIDs...),
		ReferenceAssetIDs:        append([]string(nil), input.ReferenceAssetIDs...),
		PromptRecipeID:           input.PromptRecipeID,
		UseProjectVisualContext:  input.UseProjectVisualContext,
		MaskImage:                input.MaskImage,
		BestOfConfig:             input.BestOfConfig,
		GenerationConfig:         cloneRawMessage(input.GenerationConfig),
		UseProjectQualityProfile: input.UseProjectQualityProfile,
		AspectRatio:              input.AspectRatio,
		OutputFormat:             input.OutputFormat,
		RequestedCount:           input.RequestedCount,
		Provider:                 input.Provider,
		SelectionMode:            input.SelectionMode,
		ReviewRequired:           input.ReviewRequired,
	}
	overrides, err := applySceneRegenerationOverrides(&createReq, req.Overrides)
	if err != nil {
		return sceneRegenerationBuildResult{}, err
	}

	metadata["regenerated_from_task_id"] = source.ID
	metadata["regenerate_no"] = req.RegenerationNumber
	if reason := strings.TrimSpace(req.RegenerateReason); reason != "" {
		metadata["regenerate_reason"] = reason
	}
	if requestSource := strings.TrimSpace(req.RequestSource); requestSource != "" {
		metadata["regenerate_request_source"] = requestSource
	}
	if actor := strings.TrimSpace(req.CreatedBy); actor != "" {
		metadata["regenerated_by"] = actor
	}
	metadata["regenerated_at"] = time.Now().UTC().Format(time.RFC3339)
	metadata["regeneration_overrides"] = overrides
	metadata["source_scene_identity"] = map[string]any{
		"session_id":     identity.SessionID,
		"batch_id":       identity.BatchID,
		"story_id":       identity.StoryID,
		"scene_id":       identity.SceneID,
		"source":         identity.Source,
		"task_selector":  firstNonEmpty(sceneIdentityTaskSelector(req), "source_task_id"),
		"source_task_id": source.ID,
	}
	if root := regenerationRootTaskID(metadata); root != "" {
		metadata["regeneration_root_task_id"] = root
	}
	metadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return sceneRegenerationBuildResult{}, err
	}
	createReq.MetadataJSON = metadataRaw

	return sceneRegenerationBuildResult{
		Request:                     createReq,
		SceneIdentity:               identity,
		CopiedVisualContextSnapshot: sceneRegenerationVisualSnapshot(createReq),
	}, nil
}

func sceneIdentityTaskSelector(req domain.SceneRegenerateRequest) string {
	if req.SceneIdentity == nil {
		return ""
	}
	return strings.TrimSpace(req.SceneIdentity.TaskSelector)
}

func sceneRegenerationInputFromTask(task domain.Task) sceneRegenerationSourceInput {
	var input sceneRegenerationSourceInput
	if len(task.StructuredInputJSON) > 0 {
		_ = json.Unmarshal(task.StructuredInputJSON, &input)
	}
	if input.Title == "" {
		input.Title = task.Title
	}
	if input.Purpose == "" {
		input.Purpose = task.Purpose
	}
	if input.Prompt == "" {
		input.Prompt = task.Prompt
	}
	if input.NegativePrompt == "" {
		input.NegativePrompt = task.NegativePrompt
	}
	if input.StylePreset == "" {
		input.StylePreset = task.StylePreset
	}
	if input.AspectRatio == "" {
		input.AspectRatio = task.AspectRatio
	}
	if input.OutputFormat == "" {
		input.OutputFormat = task.OutputFormat
	}
	if input.Provider == "" {
		input.Provider = task.Provider
	}
	if input.SelectionMode == "" {
		input.SelectionMode = task.SelectionMode
	}
	if input.RequestedCount == 0 {
		input.RequestedCount = task.RequestedCount
	}
	if len(input.MetadataJSON) == 0 {
		input.MetadataJSON = metadataJSONFromStructuredInput(task.StructuredInputJSON)
	}
	return input
}

func sceneIdentityFromTask(task domain.Task) (domain.SceneIdentity, error) {
	input := sceneRegenerationInputFromTask(task)
	metadata, err := sceneRegenerationMetadataMap(input.MetadataJSON)
	if err != nil {
		return domain.SceneIdentity{}, err
	}
	return sceneIdentityFromMetadata(metadata)
}

func sceneIdentityFromMetadata(metadata map[string]any) (domain.SceneIdentity, error) {
	identity := domain.SceneIdentity{
		SessionID: stringFromMetadata(metadata, "session_id"),
		BatchID:   stringFromMetadata(metadata, "batch_id"),
		StoryID:   stringFromMetadata(metadata, "story_id"),
		SceneID:   stringFromMetadata(metadata, "scene_id"),
		Source:    stringFromMetadata(metadata, "source"),
	}
	if err := validateSceneIdentity(identity); err != nil {
		return domain.SceneIdentity{}, err
	}
	return identity, nil
}

func validateSceneIdentity(identity domain.SceneIdentity) error {
	if strings.TrimSpace(identity.SessionID) == "" ||
		strings.TrimSpace(identity.BatchID) == "" ||
		strings.TrimSpace(identity.StoryID) == "" ||
		strings.TrimSpace(identity.SceneID) == "" {
		return fmt.Errorf("source task metadata_json must include session_id, batch_id, story_id and scene_id")
	}
	return nil
}

func normalizeSceneIdentity(identity domain.SceneIdentity) domain.SceneIdentity {
	identity.SessionID = strings.TrimSpace(identity.SessionID)
	identity.BatchID = strings.TrimSpace(identity.BatchID)
	identity.StoryID = strings.TrimSpace(identity.StoryID)
	identity.SceneID = strings.TrimSpace(identity.SceneID)
	identity.Source = strings.TrimSpace(identity.Source)
	identity.TaskSelector = strings.TrimSpace(identity.TaskSelector)
	return identity
}

func sceneRegenerationMetadataMap(raw json.RawMessage) (map[string]any, error) {
	var metadata map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &metadata) != nil || metadata == nil {
		return nil, fmt.Errorf("source task metadata_json must include session_id, batch_id, story_id and scene_id")
	}
	return metadata, nil
}

func stringFromMetadata(metadata map[string]any, key string) string {
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func applySceneRegenerationOverrides(req *domain.CreateTaskRequest, overrides domain.SceneRegenerateOverrides) (map[string]any, error) {
	applied := map[string]any{}
	if overrides.Prompt != nil {
		req.Prompt = strings.TrimSpace(*overrides.Prompt)
		applied["prompt"] = req.Prompt
	}
	if overrides.NegativePrompt != nil {
		req.NegativePrompt = strings.TrimSpace(*overrides.NegativePrompt)
		applied["negative_prompt"] = req.NegativePrompt
	}
	if overrides.PromptRecipeID != nil {
		req.PromptRecipeID = strings.TrimSpace(*overrides.PromptRecipeID)
		applied["prompt_recipe_id"] = req.PromptRecipeID
	}
	if overrides.CharacterIDs != nil {
		req.CharacterIDs = append([]string(nil), overrides.CharacterIDs...)
		applied["character_ids"] = append([]string(nil), overrides.CharacterIDs...)
	}
	if overrides.ReferenceAssetIDs != nil {
		req.ReferenceAssetIDs = append([]string(nil), overrides.ReferenceAssetIDs...)
		applied["reference_asset_ids"] = append([]string(nil), overrides.ReferenceAssetIDs...)
	}
	if overrides.ReferenceImages != nil {
		if err := validateSafeReferenceImages(overrides.ReferenceImages); err != nil {
			return nil, err
		}
		req.ReferenceImages = cloneReferenceImages(overrides.ReferenceImages)
		applied["reference_images"] = cloneReferenceImages(overrides.ReferenceImages)
	}
	if overrides.QualityProfileID != nil {
		qualityProfileID := strings.TrimSpace(*overrides.QualityProfileID)
		if qualityProfileID != "" {
			req.UseProjectQualityProfile = true
		}
		applied["quality_profile_id"] = qualityProfileID
	}
	if len(overrides.GenerationConfig) > 0 {
		merged, err := mergeGenerationConfig(req.GenerationConfig, overrides.GenerationConfig)
		if err != nil {
			return nil, err
		}
		req.GenerationConfig = merged
		var safe map[string]any
		_ = json.Unmarshal(overrides.GenerationConfig, &safe)
		applied["generation_config"] = safe
	}
	if overrides.RequestedCount != nil {
		req.RequestedCount = *overrides.RequestedCount
		applied["requested_count"] = req.RequestedCount
	}
	if overrides.SelectionMode != nil {
		req.SelectionMode = strings.TrimSpace(*overrides.SelectionMode)
		applied["selection_mode"] = req.SelectionMode
	}
	if overrides.AspectRatio != nil {
		req.AspectRatio = strings.TrimSpace(*overrides.AspectRatio)
		applied["aspect_ratio"] = req.AspectRatio
	}
	if overrides.OutputFormat != nil {
		req.OutputFormat = strings.TrimSpace(*overrides.OutputFormat)
		applied["output_format"] = req.OutputFormat
	}
	if overrides.Provider != nil {
		req.Provider = strings.TrimSpace(*overrides.Provider)
		applied["provider"] = req.Provider
	}
	if overrides.Model != nil {
		model := strings.TrimSpace(*overrides.Model)
		merged, err := setGenerationConfigString(req.GenerationConfig, "model", model)
		if err != nil {
			return nil, err
		}
		req.GenerationConfig = merged
		applied["model"] = model
	}
	return applied, nil
}

func mergeGenerationConfig(base, override json.RawMessage) (json.RawMessage, error) {
	baseMap, err := generationConfigMap(base)
	if err != nil {
		return nil, err
	}
	overrideMap, err := generationConfigMap(override)
	if err != nil {
		return nil, err
	}
	if err := validateSafeJSONMap(overrideMap); err != nil {
		return nil, err
	}
	for key, value := range overrideMap {
		baseMap[key] = value
	}
	return json.Marshal(baseMap)
}

func setGenerationConfigString(raw json.RawMessage, key, value string) (json.RawMessage, error) {
	config, err := generationConfigMap(raw)
	if err != nil {
		return nil, err
	}
	if value == "" {
		delete(config, key)
	} else {
		config[key] = value
	}
	return json.Marshal(config)
}

func generationConfigMap(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var config map[string]any
	if err := json.Unmarshal(raw, &config); err != nil || config == nil {
		return nil, fmt.Errorf("generation_config must be a JSON object")
	}
	return config, nil
}

func validateSafeJSONMap(values map[string]any) error {
	for key, value := range values {
		if isSensitiveOverrideKey(key) {
			return fmt.Errorf("generation_config contains unsupported sensitive field %q", key)
		}
		if err := validateSafeJSONValue(key, value); err != nil {
			return err
		}
	}
	return nil
}

func validateSafeJSONValue(key string, value any) error {
	switch typed := value.(type) {
	case string:
		if isLocalAbsolutePathLike(typed) {
			return fmt.Errorf("generation_config field %q contains a local absolute path", key)
		}
	case []any:
		for _, item := range typed {
			if err := validateSafeJSONValue(key, item); err != nil {
				return err
			}
		}
	case map[string]any:
		return validateSafeJSONMap(typed)
	}
	return nil
}

func isSensitiveOverrideKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, marker := range []string{"api_key", "provider_key", "secret", "token", "cookie", "authorization", "password"} {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return false
}

func validateSafeReferenceImages(images []domain.ReferenceImage) error {
	for _, image := range images {
		if isLocalAbsolutePathLike(image.URL) {
			return fmt.Errorf("reference_images may not contain local absolute paths")
		}
	}
	return nil
}

func isLocalAbsolutePathLike(value string) bool {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "file:") || strings.HasPrefix(value, "~/") || strings.HasPrefix(value, "/") {
		return true
	}
	if len(value) >= 3 && value[1] == ':' && (value[2] == '\\' || value[2] == '/') {
		return true
	}
	return false
}

func regenerationRootTaskID(metadata map[string]any) string {
	if root := stringFromMetadata(metadata, "regeneration_root_task_id"); root != "" {
		return root
	}
	return stringFromMetadata(metadata, "regenerated_from_task_id")
}

func sceneRegenerationVisualSnapshot(req domain.CreateTaskRequest) domain.SceneRegenerateVisualContextSnapshot {
	snapshot := domain.SceneRegenerateVisualContextSnapshot{
		CharacterIDs:      append([]string(nil), req.CharacterIDs...),
		ReferenceAssetIDs: append([]string(nil), req.ReferenceAssetIDs...),
		PromptRecipeID:    strings.TrimSpace(req.PromptRecipeID),
		CharacterCount:    len(req.CharacterIDs),
		ReferenceCount:    len(req.ReferenceAssetIDs),
	}
	snapshot.HasPromptRecipe = snapshot.PromptRecipeID != ""
	return snapshot
}

func cloneReferenceImages(images []domain.ReferenceImage) []domain.ReferenceImage {
	if images == nil {
		return nil
	}
	copied := make([]domain.ReferenceImage, len(images))
	copy(copied, images)
	return copied
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	copied := make([]byte, len(raw))
	copy(copied, raw)
	return json.RawMessage(copied)
}
