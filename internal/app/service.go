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
		MetadataJSON:   domain.NormalizeMetadataJSON(metadataJSONFromStructuredInput(item.TaskStructuredInputJSON)),
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
	if req.Title == "" {
		req.Title = "Untitled image task"
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
	if _, ok := s.providers[req.Provider]; !ok {
		return req, nil, "", fmt.Errorf("provider %q is not enabled; configure it or use %q", req.Provider, provider.MockProviderID)
	}
	if req.RequestedCount < 1 {
		req.RequestedCount = 1
	}
	if req.RequestedCount > 10 {
		req.RequestedCount = 10
	}
	req.MetadataJSON = domain.NormalizeMetadataJSON(req.MetadataJSON)

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
		"character_ids":               req.CharacterIDs,
		"reference_asset_ids":         req.ReferenceAssetIDs,
		"prompt_recipe_id":            req.PromptRecipeID,
		"use_project_visual_context":  req.UseProjectVisualContext,
		"visual_context_snapshot":     visualContext.Snapshot,
		"mask_image":                  req.MaskImage,
		"best_of_config":              req.BestOfConfig,
		"resolved_input_files":        resolvedInputFiles,
		"generation_config":           json.RawMessage(req.GenerationConfig),
		"use_project_quality_profile": req.UseProjectQualityProfile,
		"aspect_ratio":                req.AspectRatio,
		"output_format":               req.OutputFormat,
		"requested_count":             req.RequestedCount,
		"provider":                    req.Provider,
		"provider_profile":            providerProfile,
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
