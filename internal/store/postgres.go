package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/provider"
	"github.com/billionsheep/agent-imageflow/internal/storage"
)

var ErrNotFound = errors.New("not found")

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) CheckScope(ctx context.Context, scope domain.Scope) error {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM campaign c
			JOIN project p ON p.id = c.project_id
			JOIN workspace w ON w.id = c.workspace_id
			WHERE w.id = $1 AND p.id = $2 AND c.id = $3
		)
	`, scope.WorkspaceID, scope.ProjectID, scope.CampaignID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("invalid scope workspace=%s project=%s campaign=%s", scope.WorkspaceID, scope.ProjectID, scope.CampaignID)
	}
	return nil
}

func (s *PostgresStore) FindTaskByIdempotency(ctx context.Context, scope domain.Scope, key string) (domain.Task, string, bool, error) {
	if key == "" {
		return domain.Task{}, "", false, nil
	}
	row := s.db.QueryRowContext(ctx, taskSelect()+`
		WHERE workspace_id = $1 AND project_id = $2 AND idempotency_key = $3
	`, scope.WorkspaceID, scope.ProjectID, key)
	task, inputHash, err := scanTaskWithHash(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, "", false, nil
	}
	if err != nil {
		return domain.Task{}, "", false, err
	}
	return task, inputHash, true, nil
}

func (s *PostgresStore) InsertTask(ctx context.Context, task domain.Task, inputHash string, reviewRequired bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO generation_task (
			id, workspace_id, project_id, campaign_id, idempotency_key, input_hash,
			title, purpose, prompt, negative_prompt, style_preset, aspect_ratio,
			output_format, structured_input_json, provider, status, requested_count,
			review_required, created_by, trace_id
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::jsonb,$15,$16,$17,$18,$19,$20)
	`, task.ID, task.WorkspaceID, task.ProjectID, task.CampaignID, task.IdempotencyKey, inputHash,
		task.Title, task.Purpose, task.Prompt, task.NegativePrompt, task.StylePreset, task.AspectRatio,
		task.OutputFormat, jsonOrEmpty(task.StructuredInputJSON), task.Provider, task.Status, task.RequestedCount,
		reviewRequired, task.CreatedBy, task.TraceID)
	return err
}

func (s *PostgresStore) MarkTaskEnqueueFailed(ctx context.Context, taskID string, err error) error {
	_, updateErr := s.db.ExecContext(ctx, `
		UPDATE generation_task
		SET status = $2, error_code = 'enqueue_failed', error_message = $3, updated_at = now()
		WHERE id = $1
	`, taskID, domain.TaskEnqueueFailed, err.Error())
	return updateErr
}

func (s *PostgresStore) GetTask(ctx context.Context, taskID string) (domain.Task, error) {
	row := s.db.QueryRowContext(ctx, taskSelect()+` WHERE id = $1`, taskID)
	task, _, err := scanTaskWithHash(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, ErrNotFound
	}
	return task, err
}

func (s *PostgresStore) ListAssetsByTask(ctx context.Context, taskID string) ([]domain.AssetWithVersion, error) {
	rows, err := s.db.QueryContext(ctx, assetWithVersionSelect()+`
		WHERE a.task_id = $1
		ORDER BY a.created_at ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssetRows(rows)
}

func (s *PostgresStore) ListAssetsByCampaign(ctx context.Context, projectID, campaignID string) ([]domain.AssetWithVersion, error) {
	rows, err := s.db.QueryContext(ctx, assetWithVersionSelect()+`
		WHERE a.project_id = $1 AND a.campaign_id = $2
		ORDER BY a.created_at DESC
	`, projectID, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssetRows(rows)
}

func (s *PostgresStore) GetAssetWithVersion(ctx context.Context, assetID string) (domain.AssetWithVersion, error) {
	row := s.db.QueryRowContext(ctx, assetWithVersionSelect()+` WHERE a.id = $1`, assetID)
	item, err := scanAssetWithVersion(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AssetWithVersion{}, ErrNotFound
	}
	return item, err
}

func (s *PostgresStore) CreateAttempt(ctx context.Context, task domain.Task) (string, int, error) {
	var attemptNo int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(attempt_no), 0) + 1 FROM task_attempt WHERE task_id = $1
	`, task.ID).Scan(&attemptNo); err != nil {
		return "", 0, err
	}
	attemptID := domain.NewID("attempt")
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO task_attempt (id, task_id, attempt_no, status, provider)
		VALUES ($1, $2, $3, $4, $5)
	`, attemptID, task.ID, attemptNo, domain.AttemptRunning, task.Provider)
	return attemptID, attemptNo, err
}

func (s *PostgresStore) UpdateTaskStatus(ctx context.Context, taskID, status string, code, message *string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE generation_task
		SET status = $2, error_code = $3, error_message = $4, updated_at = now()
		WHERE id = $1
	`, taskID, status, code, message)
	return err
}

func (s *PostgresStore) FinishAttempt(ctx context.Context, attemptID, status string, result provider.Result, started time.Time, code, message *string) error {
	latency := int(time.Since(started).Milliseconds())
	_, err := s.db.ExecContext(ctx, `
		UPDATE task_attempt
		SET status = $2, provider_request_id = $3, finished_at = now(), latency_ms = $4,
			error_code = $5, error_message = $6, raw_response_json = $7::jsonb, cost_json = $8::jsonb
		WHERE id = $1
	`, attemptID, status, result.ProviderRequestID, latency, code, message,
		jsonOrEmpty(result.RawResponse), jsonOrEmpty(result.CostRaw))
	return err
}

func (s *PostgresStore) InsertAssetWithVersion(ctx context.Context, task domain.Task, file provider.GeneratedFile, stored storage.StoredAssetFile) (domain.AssetWithVersion, error) {
	asset := domain.Asset{
		ID:          stored.AssetID,
		WorkspaceID: task.WorkspaceID,
		ProjectID:   task.ProjectID,
		CampaignID:  task.CampaignID,
		TaskID:      task.ID,
		Name:        fmt.Sprintf("%s candidate %d", task.Title, file.Slot+1),
		Type:        "image",
		Status:      domain.AssetDraft,
	}
	version := domain.AssetVersion{
		ID:             stored.VersionID,
		AssetID:        stored.AssetID,
		Version:        stored.Version,
		Status:         domain.VersionReady,
		FilePath:       stored.FilePath,
		ThumbnailPath:  stored.ThumbnailPath,
		MetadataPath:   stored.MetadataPath,
		MimeType:       stored.MimeType,
		Width:          stored.Width,
		Height:         stored.Height,
		Hash:           stored.Hash,
		Provider:       task.Provider,
		Model:          file.Model,
		Prompt:         task.Prompt,
		ParametersJSON: file.ParametersRaw,
		CostJSON:       file.CostRaw,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.AssetWithVersion{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO asset (id, workspace_id, project_id, campaign_id, task_id, name, type, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, asset.ID, asset.WorkspaceID, asset.ProjectID, asset.CampaignID, asset.TaskID, asset.Name, asset.Type, asset.Status); err != nil {
		return domain.AssetWithVersion{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO asset_version (
			id, asset_id, version, status, file_path, thumbnail_path, metadata_path,
			mime_type, width, height, hash, provider, model, prompt, parameters_json, cost_json
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::jsonb,$16::jsonb)
	`, version.ID, version.AssetID, version.Version, version.Status, version.FilePath, version.ThumbnailPath,
		version.MetadataPath, version.MimeType, version.Width, version.Height, version.Hash, version.Provider,
		version.Model, version.Prompt, jsonOrEmpty(version.ParametersJSON), jsonOrEmpty(version.CostJSON)); err != nil {
		return domain.AssetWithVersion{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE asset SET current_version_id = $2, updated_at = now() WHERE id = $1
	`, asset.ID, version.ID); err != nil {
		return domain.AssetWithVersion{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.AssetWithVersion{}, err
	}
	asset.CurrentVersionID = version.ID
	return domain.AssetWithVersion{Asset: asset, Version: version}, nil
}

func (s *PostgresStore) ReviewAsset(ctx context.Context, assetID, action, reviewer, note string) (domain.AssetWithVersion, error) {
	item, err := s.GetAssetWithVersion(ctx, assetID)
	if err != nil {
		return domain.AssetWithVersion{}, err
	}
	nextStatus := ""
	switch action {
	case "approve":
		nextStatus = domain.AssetApproved
	case "reject":
		nextStatus = domain.AssetRejected
	default:
		return domain.AssetWithVersion{}, fmt.Errorf("unknown review action %q", action)
	}
	if item.Status == nextStatus {
		return item, nil
	}
	if item.Status != domain.AssetDraft {
		return domain.AssetWithVersion{}, fmt.Errorf("asset %s is %s and cannot transition to %s", assetID, item.Status, nextStatus)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.AssetWithVersion{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err := tx.ExecContext(ctx, `
		UPDATE asset SET status = $2, updated_at = now() WHERE id = $1
	`, assetID, nextStatus); err != nil {
		return domain.AssetWithVersion{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO review_event (id, asset_id, version_id, action, reviewer, note)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, domain.NewID("rev"), assetID, item.Version.ID, action, reviewer, note); err != nil {
		return domain.AssetWithVersion{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.AssetWithVersion{}, err
	}
	item.Status = nextStatus
	return item, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func taskSelect() string {
	return `SELECT id, workspace_id, project_id, campaign_id, idempotency_key, input_hash,
		title, purpose, prompt, negative_prompt, style_preset, aspect_ratio, output_format,
		structured_input_json, provider, status, requested_count, created_by, trace_id,
		created_at, updated_at, error_code, error_message FROM generation_task`
}

func scanTaskWithHash(row rowScanner) (domain.Task, string, error) {
	var task domain.Task
	var inputHash string
	var errorCode sql.NullString
	var errorMessage sql.NullString
	err := row.Scan(&task.ID, &task.WorkspaceID, &task.ProjectID, &task.CampaignID,
		&task.IdempotencyKey, &inputHash, &task.Title, &task.Purpose, &task.Prompt,
		&task.NegativePrompt, &task.StylePreset, &task.AspectRatio, &task.OutputFormat,
		&task.StructuredInputJSON, &task.Provider, &task.Status, &task.RequestedCount,
		&task.CreatedBy, &task.TraceID, &task.CreatedAt, &task.UpdatedAt,
		&errorCode, &errorMessage)
	if errorCode.Valid {
		task.ErrorCode = &errorCode.String
	}
	if errorMessage.Valid {
		task.ErrorMessage = &errorMessage.String
	}
	return task, inputHash, err
}

func assetWithVersionSelect() string {
	return `SELECT a.id, a.workspace_id, a.project_id, a.campaign_id, a.task_id,
		a.name, a.type, a.current_version_id, a.status, a.created_at, a.updated_at,
		v.id, v.asset_id, v.version, v.status, v.file_path, v.thumbnail_path, v.metadata_path,
		v.object_key, v.public_url, v.mime_type, v.width, v.height, v.hash, v.provider, v.model,
		v.prompt, v.parameters_json, v.cost_json, v.created_at
		FROM asset a
		JOIN asset_version v ON v.id = a.current_version_id`
}

func scanAssetRows(rows *sql.Rows) ([]domain.AssetWithVersion, error) {
	items := []domain.AssetWithVersion{}
	for rows.Next() {
		item, err := scanAssetWithVersion(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanAssetWithVersion(row rowScanner) (domain.AssetWithVersion, error) {
	var item domain.AssetWithVersion
	err := row.Scan(&item.ID, &item.WorkspaceID, &item.ProjectID, &item.CampaignID, &item.TaskID,
		&item.Name, &item.Type, &item.CurrentVersionID, &item.Status, &item.CreatedAt, &item.UpdatedAt,
		&item.Version.ID, &item.Version.AssetID, &item.Version.Version, &item.Version.Status,
		&item.Version.FilePath, &item.Version.ThumbnailPath, &item.Version.MetadataPath,
		&item.Version.ObjectKey, &item.Version.PublicURL, &item.Version.MimeType, &item.Version.Width,
		&item.Version.Height, &item.Version.Hash, &item.Version.Provider, &item.Version.Model,
		&item.Version.Prompt, &item.Version.ParametersJSON, &item.Version.CostJSON, &item.Version.CreatedAt)
	return item, err
}

func jsonOrEmpty(raw []byte) []byte {
	if len(raw) == 0 || !json.Valid(raw) {
		return []byte(`{}`)
	}
	return raw
}
