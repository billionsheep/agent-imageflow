package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/provider"
	"github.com/billionsheep/agent-imageflow/internal/storage"
)

var ErrNotFound = errors.New("not found")

type PostgresStore struct {
	db *sql.DB
}

type RepairTaskCandidate struct {
	Task         domain.Task `json:"task"`
	IssueKind    string      `json:"issue_kind"`
	AttemptCount int         `json:"attempt_count"`
}

type InvalidCurrentVersionAsset struct {
	AssetID          string `json:"asset_id"`
	CurrentVersionID string `json:"current_version_id"`
	VersionStatus    string `json:"version_status,omitempty"`
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

func (s *PostgresStore) ListWorkspaces(ctx context.Context) ([]domain.WorkspaceSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, metadata_json
		FROM workspace
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.WorkspaceSummary{}
	for rows.Next() {
		var item domain.WorkspaceSummary
		var metadataRaw []byte
		if err := rows.Scan(&item.WorkspaceID, &item.Name, &metadataRaw); err != nil {
			return nil, err
		}
		metadata, err := parseScopeMetadata(metadataRaw)
		if err != nil {
			return nil, err
		}
		item.Archived = metadata.Archived()
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CreateWorkspace(ctx context.Context, req domain.CreateWorkspaceRequest) (domain.WorkspaceSummary, error) {
	var item domain.WorkspaceSummary
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO workspace (id, name, metadata_json)
		VALUES ($1, $2, '{}'::jsonb)
		RETURNING id, name
	`, req.WorkspaceID, req.Name).Scan(&item.WorkspaceID, &item.Name)
	return item, err
}

func (s *PostgresStore) ListProjects(ctx context.Context, workspaceID string) ([]domain.ProjectSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, name, description, metadata_json
		FROM project
		WHERE workspace_id = $1
		ORDER BY created_at ASC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.ProjectSummary{}
	for rows.Next() {
		var item domain.ProjectSummary
		var metadataRaw []byte
		if err := rows.Scan(&item.ProjectID, &item.WorkspaceID, &item.Name, &item.Description, &metadataRaw); err != nil {
			return nil, err
		}
		metadata, err := parseProjectMetadata(metadataRaw)
		if err != nil {
			return nil, err
		}
		item.Archived = metadata.Archived()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		exists, err := s.workspaceExists(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrNotFound
		}
	}
	return items, nil
}

func (s *PostgresStore) CreateProject(ctx context.Context, workspaceID string, req domain.CreateProjectRequest) (domain.ProjectSummary, error) {
	var item domain.ProjectSummary
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO project (id, workspace_id, name, description, style_preset, metadata_json)
		SELECT $1, w.id, $2, $3, '', '{}'::jsonb
		FROM workspace w
		WHERE w.id = $4
		RETURNING id, workspace_id, name, description
	`, req.ProjectID, req.Name, req.Description, workspaceID).Scan(&item.ProjectID, &item.WorkspaceID, &item.Name, &item.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProjectSummary{}, ErrNotFound
	}
	return item, err
}

func (s *PostgresStore) ListCampaigns(ctx context.Context, workspaceID, projectID string) ([]domain.CampaignSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, project_id, name, description, metadata_json
		FROM campaign
		WHERE workspace_id = $1 AND project_id = $2
		ORDER BY created_at ASC, id ASC
	`, workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.CampaignSummary{}
	for rows.Next() {
		var item domain.CampaignSummary
		var metadataRaw []byte
		if err := rows.Scan(&item.CampaignID, &item.WorkspaceID, &item.ProjectID, &item.Name, &item.Description, &metadataRaw); err != nil {
			return nil, err
		}
		metadata, err := parseScopeMetadata(metadataRaw)
		if err != nil {
			return nil, err
		}
		item.Archived = metadata.Archived()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		exists, err := s.projectExists(ctx, workspaceID, projectID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrNotFound
		}
	}
	return items, nil
}

func (s *PostgresStore) CreateCampaign(ctx context.Context, workspaceID, projectID string, req domain.CreateCampaignRequest) (domain.CampaignSummary, error) {
	var item domain.CampaignSummary
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO campaign (id, workspace_id, project_id, name, description, metadata_json)
		SELECT $1, p.workspace_id, p.id, $2, $3, '{}'::jsonb
		FROM project p
		WHERE p.workspace_id = $4 AND p.id = $5
		RETURNING id, workspace_id, project_id, name, description
	`, req.CampaignID, req.Name, req.Description, workspaceID, projectID).Scan(&item.CampaignID, &item.WorkspaceID, &item.ProjectID, &item.Name, &item.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.CampaignSummary{}, ErrNotFound
	}
	return item, err
}

func (s *PostgresStore) UpdateWorkspace(ctx context.Context, workspaceID string, req domain.UpdateWorkspaceRequest) (domain.WorkspaceSummary, error) {
	var currentName string
	var metadataRaw []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT name, metadata_json
		FROM workspace
		WHERE id = $1
	`, workspaceID).Scan(&currentName, &metadataRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.WorkspaceSummary{}, ErrNotFound
	}
	if err != nil {
		return domain.WorkspaceSummary{}, err
	}

	metadata, err := parseScopeMetadata(metadataRaw)
	if err != nil {
		return domain.WorkspaceSummary{}, err
	}
	if req.Archived != nil {
		metadata.SetArchived(*req.Archived)
	}

	nextName := currentName
	if req.Name != nil {
		nextName = *req.Name
	}
	updatedMetadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return domain.WorkspaceSummary{}, err
	}

	var item domain.WorkspaceSummary
	err = s.db.QueryRowContext(ctx, `
		UPDATE workspace
		SET name = $2, metadata_json = $3::jsonb, updated_at = now()
		WHERE id = $1
		RETURNING id, name
	`, workspaceID, nextName, updatedMetadataRaw).Scan(&item.WorkspaceID, &item.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.WorkspaceSummary{}, ErrNotFound
	}
	if err != nil {
		return domain.WorkspaceSummary{}, err
	}
	item.Archived = metadata.Archived()
	return item, nil
}

func (s *PostgresStore) UpdateProject(ctx context.Context, workspaceID, projectID string, req domain.UpdateProjectRequest) (domain.ProjectSummary, error) {
	var currentName string
	var currentDescription string
	var metadataRaw []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT name, description, metadata_json
		FROM project
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, projectID).Scan(&currentName, &currentDescription, &metadataRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProjectSummary{}, ErrNotFound
	}
	if err != nil {
		return domain.ProjectSummary{}, err
	}

	metadata, err := parseProjectMetadata(metadataRaw)
	if err != nil {
		return domain.ProjectSummary{}, err
	}
	if req.Archived != nil {
		metadata.SetArchived(*req.Archived)
	}

	nextName := currentName
	if req.Name != nil {
		nextName = *req.Name
	}
	updatedMetadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return domain.ProjectSummary{}, err
	}

	var item domain.ProjectSummary
	err = s.db.QueryRowContext(ctx, `
		UPDATE project
		SET name = $3, metadata_json = $4::jsonb, updated_at = now()
		WHERE workspace_id = $1 AND id = $2
		RETURNING id, workspace_id, name, description
	`, workspaceID, projectID, nextName, updatedMetadataRaw).Scan(&item.ProjectID, &item.WorkspaceID, &item.Name, &item.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProjectSummary{}, ErrNotFound
	}
	if err != nil {
		return domain.ProjectSummary{}, err
	}
	item.Description = currentDescription
	item.Archived = metadata.Archived()
	return item, nil
}

func (s *PostgresStore) UpdateCampaign(ctx context.Context, workspaceID, projectID, campaignID string, req domain.UpdateCampaignRequest) (domain.CampaignSummary, error) {
	var currentName string
	var currentDescription string
	var metadataRaw []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT name, description, metadata_json
		FROM campaign
		WHERE workspace_id = $1 AND project_id = $2 AND id = $3
	`, workspaceID, projectID, campaignID).Scan(&currentName, &currentDescription, &metadataRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.CampaignSummary{}, ErrNotFound
	}
	if err != nil {
		return domain.CampaignSummary{}, err
	}

	metadata, err := parseScopeMetadata(metadataRaw)
	if err != nil {
		return domain.CampaignSummary{}, err
	}
	if req.Archived != nil {
		metadata.SetArchived(*req.Archived)
	}

	nextName := currentName
	if req.Name != nil {
		nextName = *req.Name
	}
	updatedMetadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return domain.CampaignSummary{}, err
	}

	var item domain.CampaignSummary
	err = s.db.QueryRowContext(ctx, `
		UPDATE campaign
		SET name = $4, metadata_json = $5::jsonb, updated_at = now()
		WHERE workspace_id = $1 AND project_id = $2 AND id = $3
		RETURNING id, workspace_id, project_id, name, description
	`, workspaceID, projectID, campaignID, nextName, updatedMetadataRaw).Scan(&item.CampaignID, &item.WorkspaceID, &item.ProjectID, &item.Name, &item.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.CampaignSummary{}, ErrNotFound
	}
	if err != nil {
		return domain.CampaignSummary{}, err
	}
	item.Description = currentDescription
	item.Archived = metadata.Archived()
	return item, nil
}

func (s *PostgresStore) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var exists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM workspace
			WHERE id = $1
		)
	`, workspaceID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	var projectCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM project
		WHERE workspace_id = $1
	`, workspaceID).Scan(&projectCount); err != nil {
		return err
	}
	if projectCount > 0 {
		return fmt.Errorf("workspace %s is not empty", workspaceID)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM workspace
		WHERE id = $1
	`, workspaceID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PostgresStore) DeleteProject(ctx context.Context, workspaceID, projectID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var exists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM project
			WHERE workspace_id = $1 AND id = $2
		)
	`, workspaceID, projectID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	var campaignCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM campaign
		WHERE workspace_id = $1 AND project_id = $2
	`, workspaceID, projectID).Scan(&campaignCount); err != nil {
		return err
	}
	if campaignCount > 0 {
		return fmt.Errorf("project %s is not empty", projectID)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM project
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, projectID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PostgresStore) DeleteCampaign(ctx context.Context, workspaceID, projectID, campaignID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var exists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM campaign
			WHERE workspace_id = $1 AND project_id = $2 AND id = $3
		)
	`, workspaceID, projectID, campaignID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	var taskCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM generation_task
		WHERE workspace_id = $1 AND project_id = $2 AND campaign_id = $3
	`, workspaceID, projectID, campaignID).Scan(&taskCount); err != nil {
		return err
	}
	if taskCount > 0 {
		return fmt.Errorf("campaign %s is not empty", campaignID)
	}

	var assetCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM asset
		WHERE workspace_id = $1 AND project_id = $2 AND campaign_id = $3
	`, workspaceID, projectID, campaignID).Scan(&assetCount); err != nil {
		return err
	}
	if assetCount > 0 {
		return fmt.Errorf("campaign %s is not empty", campaignID)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM campaign
		WHERE workspace_id = $1 AND project_id = $2 AND id = $3
	`, workspaceID, projectID, campaignID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PostgresStore) GetProjectQualityProfile(ctx context.Context, workspaceID, projectID string) (domain.QualityProfile, error) {
	var stylePreset string
	var metadataRaw []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT style_preset, metadata_json
		FROM project
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, projectID).Scan(&stylePreset, &metadataRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.QualityProfile{}, ErrNotFound
	}
	if err != nil {
		return domain.QualityProfile{}, err
	}
	return qualityProfileFromProjectMetadata(metadataRaw, stylePreset)
}

func (s *PostgresStore) GetProjectAccessConfig(ctx context.Context, workspaceID, projectID string) (domain.ProjectAccessConfig, error) {
	var metadataRaw []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT metadata_json
		FROM project
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, projectID).Scan(&metadataRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProjectAccessConfig{}, ErrNotFound
	}
	if err != nil {
		return domain.ProjectAccessConfig{}, err
	}
	return accessConfigFromProjectMetadata(metadataRaw)
}

func (s *PostgresStore) GetProjectAccessConfigByProjectID(ctx context.Context, projectID string) (domain.ProjectAccessConfig, error) {
	var metadataRaw []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT metadata_json
		FROM project
		WHERE id = $1
	`, projectID).Scan(&metadataRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProjectAccessConfig{}, ErrNotFound
	}
	if err != nil {
		return domain.ProjectAccessConfig{}, err
	}
	return accessConfigFromProjectMetadata(metadataRaw)
}

func (s *PostgresStore) UpdateProjectQualityProfile(ctx context.Context, workspaceID, projectID string, profile domain.QualityProfile) (domain.QualityProfile, error) {
	profileRaw, err := json.Marshal(profile)
	if err != nil {
		return domain.QualityProfile{}, err
	}
	var stylePreset string
	var metadataRaw []byte
	err = s.db.QueryRowContext(ctx, `
		UPDATE project
		SET metadata_json = jsonb_set(metadata_json, '{quality_profile}', $3::jsonb, true),
			style_preset = CASE WHEN $4 <> '' THEN $4 ELSE style_preset END,
			updated_at = now()
		WHERE workspace_id = $1 AND id = $2
		RETURNING style_preset, metadata_json
	`, workspaceID, projectID, profileRaw, profile.StylePreset).Scan(&stylePreset, &metadataRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.QualityProfile{}, ErrNotFound
	}
	if err != nil {
		return domain.QualityProfile{}, err
	}
	return qualityProfileFromProjectMetadata(metadataRaw, stylePreset)
}

func (s *PostgresStore) UpdateProjectAccessConfig(ctx context.Context, workspaceID, projectID string, config domain.ProjectAccessConfig) (domain.ProjectAccessConfig, error) {
	configRaw, err := json.Marshal(config)
	if err != nil {
		return domain.ProjectAccessConfig{}, err
	}
	var metadataRaw []byte
	err = s.db.QueryRowContext(ctx, `
		UPDATE project
		SET metadata_json = jsonb_set(metadata_json, '{access_config}', $3::jsonb, true),
			updated_at = now()
		WHERE workspace_id = $1 AND id = $2
		RETURNING metadata_json
	`, workspaceID, projectID, configRaw).Scan(&metadataRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProjectAccessConfig{}, ErrNotFound
	}
	if err != nil {
		return domain.ProjectAccessConfig{}, err
	}
	return accessConfigFromProjectMetadata(metadataRaw)
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

func (s *PostgresStore) GetTaskScope(ctx context.Context, taskID string) (domain.Scope, error) {
	var scope domain.Scope
	err := s.db.QueryRowContext(ctx, `
		SELECT workspace_id, project_id, campaign_id
		FROM generation_task
		WHERE id = $1
	`, taskID).Scan(&scope.WorkspaceID, &scope.ProjectID, &scope.CampaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Scope{}, ErrNotFound
	}
	return scope, err
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

func (s *PostgresStore) GetAssetScope(ctx context.Context, assetID string) (domain.Scope, error) {
	var scope domain.Scope
	err := s.db.QueryRowContext(ctx, `
		SELECT workspace_id, project_id, campaign_id
		FROM asset
		WHERE id = $1
	`, assetID).Scan(&scope.WorkspaceID, &scope.ProjectID, &scope.CampaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Scope{}, ErrNotFound
	}
	return scope, err
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

func (s *PostgresStore) ListRepairTaskCandidates(ctx context.Context, staleBefore time.Time, limit int) ([]RepairTaskCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+taskSelectColumns("gt")+`,
			COALESCE(attempts.attempt_count, 0),
			CASE
				WHEN gt.status = $1 THEN 'enqueue_failed'
				WHEN gt.status = $2 THEN 'stale_running'
				WHEN gt.status = $3 THEN 'stale_queued'
				ELSE 'unknown'
			END
		FROM generation_task gt
		LEFT JOIN (
			SELECT task_id, COUNT(*) AS attempt_count
			FROM task_attempt
			GROUP BY task_id
		) attempts ON attempts.task_id = gt.id
		WHERE gt.status = $1
			OR (gt.status = $2 AND gt.updated_at < $4)
			OR (gt.status = $3 AND gt.updated_at < $4)
		ORDER BY gt.updated_at ASC
		LIMIT $5
	`, domain.TaskEnqueueFailed, domain.TaskRunning, domain.TaskQueued, staleBefore, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRepairTaskCandidateRows(rows)
}

func (s *PostgresStore) ListRepairTaskCandidatesByScope(ctx context.Context, scope domain.Scope, staleBefore time.Time, limit int) ([]RepairTaskCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+taskSelectColumns("gt")+`,
			COALESCE(attempts.attempt_count, 0),
			CASE
				WHEN gt.status = $1 THEN 'enqueue_failed'
				WHEN gt.status = $2 THEN 'stale_running'
				WHEN gt.status = $3 THEN 'stale_queued'
				ELSE 'unknown'
			END
		FROM generation_task gt
		LEFT JOIN (
			SELECT task_id, COUNT(*) AS attempt_count
			FROM task_attempt
			GROUP BY task_id
		) attempts ON attempts.task_id = gt.id
		WHERE gt.workspace_id = $5 AND gt.project_id = $6 AND gt.campaign_id = $7
			AND (
				gt.status = $1
				OR (gt.status = $2 AND gt.updated_at < $4)
				OR (gt.status = $3 AND gt.updated_at < $4)
			)
		ORDER BY gt.updated_at ASC
		LIMIT $8
	`, domain.TaskEnqueueFailed, domain.TaskRunning, domain.TaskQueued, staleBefore, scope.WorkspaceID, scope.ProjectID, scope.CampaignID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRepairTaskCandidateRows(rows)
}

func scanRepairTaskCandidateRows(rows *sql.Rows) ([]RepairTaskCandidate, error) {
	items := []RepairTaskCandidate{}
	for rows.Next() {
		var item RepairTaskCandidate
		var inputHash string
		var errorCode sql.NullString
		var errorMessage sql.NullString
		err := rows.Scan(&item.Task.ID, &item.Task.WorkspaceID, &item.Task.ProjectID, &item.Task.CampaignID,
			&item.Task.IdempotencyKey, &inputHash, &item.Task.Title, &item.Task.Purpose, &item.Task.Prompt,
			&item.Task.NegativePrompt, &item.Task.StylePreset, &item.Task.AspectRatio, &item.Task.OutputFormat,
			&item.Task.StructuredInputJSON, &item.Task.Provider, &item.Task.Status, &item.Task.RequestedCount,
			&item.Task.CreatedBy, &item.Task.TraceID, &item.Task.CreatedAt, &item.Task.UpdatedAt,
			&errorCode, &errorMessage, &item.AttemptCount, &item.IssueKind)
		if err != nil {
			return nil, err
		}
		item.Task.SelectionMode = selectionModeFromStructuredInput(item.Task.StructuredInputJSON)
		if errorCode.Valid {
			item.Task.ErrorCode = &errorCode.String
		}
		if errorMessage.Valid {
			item.Task.ErrorMessage = &errorMessage.String
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) ListCurrentAssetVersions(ctx context.Context, limit int) ([]domain.AssetWithVersion, error) {
	rows, err := s.db.QueryContext(ctx, assetWithVersionSelect()+`
		ORDER BY a.updated_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssetRows(rows)
}

func (s *PostgresStore) ListCurrentAssetVersionsByScope(ctx context.Context, scope domain.Scope, limit int) ([]domain.AssetWithVersion, error) {
	rows, err := s.db.QueryContext(ctx, assetWithVersionSelect()+`
		WHERE a.workspace_id = $1 AND a.project_id = $2 AND a.campaign_id = $3
		ORDER BY a.updated_at DESC
		LIMIT $4
	`, scope.WorkspaceID, scope.ProjectID, scope.CampaignID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssetRows(rows)
}

func (s *PostgresStore) ListInvalidCurrentVersionAssets(ctx context.Context, limit int) ([]InvalidCurrentVersionAsset, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.current_version_id, COALESCE(v.status, '')
		FROM asset a
		LEFT JOIN asset_version v ON v.id = a.current_version_id
		WHERE a.current_version_id = '' OR v.id IS NULL OR v.status <> $1
		ORDER BY a.updated_at DESC
		LIMIT $2
	`, domain.VersionReady, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []InvalidCurrentVersionAsset{}
	for rows.Next() {
		var item InvalidCurrentVersionAsset
		if err := rows.Scan(&item.AssetID, &item.CurrentVersionID, &item.VersionStatus); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) ListInvalidCurrentVersionAssetsByScope(ctx context.Context, scope domain.Scope, limit int) ([]InvalidCurrentVersionAsset, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.current_version_id, COALESCE(v.status, '')
		FROM asset a
		LEFT JOIN asset_version v ON v.id = a.current_version_id
		WHERE a.workspace_id = $2 AND a.project_id = $3 AND a.campaign_id = $4
			AND (a.current_version_id = '' OR v.id IS NULL OR v.status <> $1)
		ORDER BY a.updated_at DESC
		LIMIT $5
	`, domain.VersionReady, scope.WorkspaceID, scope.ProjectID, scope.CampaignID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []InvalidCurrentVersionAsset{}
	for rows.Next() {
		var item InvalidCurrentVersionAsset
		if err := rows.Scan(&item.AssetID, &item.CurrentVersionID, &item.VersionStatus); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) ListKnownAssetFilePaths(ctx context.Context, limit int) ([]string, error) {
	query := `
		SELECT file_path, thumbnail_path, metadata_path
		FROM asset_version
		ORDER BY created_at DESC`
	args := []any{}
	if limit > 0 {
		query += ` LIMIT $1`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	paths := []string{}
	for rows.Next() {
		var filePath, thumbnailPath, metadataPath string
		if err := rows.Scan(&filePath, &thumbnailPath, &metadataPath); err != nil {
			return nil, err
		}
		paths = append(paths, filePath, thumbnailPath, metadataPath)
	}
	return paths, rows.Err()
}

func (s *PostgresStore) GetStorageGovernanceCounts(ctx context.Context, scope domain.Scope) (domain.StorageGovernanceCounts, error) {
	instance, err := s.storageGovernanceCountSnapshot(ctx, "", nil)
	if err != nil {
		return domain.StorageGovernanceCounts{}, err
	}
	workspace, err := s.storageGovernanceCountSnapshot(ctx, "workspace_id = $1", []any{scope.WorkspaceID})
	if err != nil {
		return domain.StorageGovernanceCounts{}, err
	}
	project, err := s.storageGovernanceCountSnapshot(ctx, "workspace_id = $1 AND project_id = $2", []any{scope.WorkspaceID, scope.ProjectID})
	if err != nil {
		return domain.StorageGovernanceCounts{}, err
	}
	campaign, err := s.storageGovernanceCountSnapshot(ctx, "workspace_id = $1 AND project_id = $2 AND campaign_id = $3", []any{scope.WorkspaceID, scope.ProjectID, scope.CampaignID})
	if err != nil {
		return domain.StorageGovernanceCounts{}, err
	}
	return domain.StorageGovernanceCounts{
		Instance:  instance,
		Workspace: workspace,
		Project:   project,
		Campaign:  campaign,
	}, nil
}

func (s *PostgresStore) storageGovernanceCountSnapshot(ctx context.Context, whereClause string, args []any) (domain.StorageGovernanceCountSnapshot, error) {
	whereSQL := ""
	if strings.TrimSpace(whereClause) != "" {
		whereSQL = " WHERE " + whereClause
	}
	taskArgs := append([]any{}, args...)
	failedStart := len(taskArgs) + 1
	taskArgs = append(taskArgs, domain.TaskFailed, domain.TaskEnqueueFailed)
	taskQuery := fmt.Sprintf(`
		SELECT COUNT(*), COUNT(*) FILTER (WHERE status IN ($%d, $%d))
		FROM generation_task%s
	`, failedStart, failedStart+1, whereSQL)

	var snapshot domain.StorageGovernanceCountSnapshot
	if err := s.db.QueryRowContext(ctx, taskQuery, taskArgs...).Scan(&snapshot.TaskCount, &snapshot.FailedTaskCount); err != nil {
		return domain.StorageGovernanceCountSnapshot{}, err
	}

	assetArgs := append([]any{}, args...)
	statusStart := len(assetArgs) + 1
	assetArgs = append(assetArgs, domain.AssetDraft, domain.AssetApproved, domain.AssetRejected, domain.AssetPublished)
	assetQuery := fmt.Sprintf(`
		SELECT COUNT(*),
			COUNT(*) FILTER (WHERE status = $%d),
			COUNT(*) FILTER (WHERE status = $%d),
			COUNT(*) FILTER (WHERE status = $%d),
			COUNT(*) FILTER (WHERE status = $%d)
		FROM asset%s
	`, statusStart, statusStart+1, statusStart+2, statusStart+3, whereSQL)
	if err := s.db.QueryRowContext(ctx, assetQuery, assetArgs...).Scan(
		&snapshot.AssetCount,
		&snapshot.GeneratedAssetCount,
		&snapshot.SelectedAssetCount,
		&snapshot.RejectedAssetCount,
		&snapshot.PublishedAssetCount,
	); err != nil {
		return domain.StorageGovernanceCountSnapshot{}, err
	}
	return snapshot, nil
}

func (s *PostgresStore) ListCleanupAssetCandidates(ctx context.Context, scope domain.Scope, includeRejected, includeGenerated bool, limit int) ([]domain.AssetWithVersion, error) {
	statuses := []string{}
	if includeRejected {
		statuses = append(statuses, domain.AssetRejected)
	}
	if includeGenerated {
		statuses = append(statuses, domain.AssetDraft)
	}
	if len(statuses) == 0 {
		return nil, nil
	}
	if limit < 1 {
		limit = 100
	}
	args := []any{scope.WorkspaceID, scope.ProjectID, scope.CampaignID}
	placeholders := make([]string, 0, len(statuses))
	for _, status := range statuses {
		args = append(args, status)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, assetWithVersionSelect()+fmt.Sprintf(`
		WHERE a.workspace_id = $1 AND a.project_id = $2 AND a.campaign_id = $3
			AND a.status IN (%s)
		ORDER BY a.updated_at ASC, a.id ASC
		LIMIT $%d
	`, strings.Join(placeholders, ","), len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssetRows(rows)
}

func (s *PostgresStore) DeleteCleanupAssetCandidate(ctx context.Context, scope domain.Scope, assetID string, allowedStatuses []string) (domain.AssetWithVersion, error) {
	allowed := map[string]bool{}
	for _, status := range allowedStatuses {
		status = strings.TrimSpace(status)
		if status != "" {
			allowed[status] = true
		}
	}
	if len(allowed) == 0 {
		return domain.AssetWithVersion{}, fmt.Errorf("at least one cleanup asset status must be allowed")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.AssetWithVersion{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	row := tx.QueryRowContext(ctx, assetWithVersionSelect()+`
		WHERE a.id = $1 AND a.workspace_id = $2 AND a.project_id = $3 AND a.campaign_id = $4
		FOR UPDATE OF a, v
	`, assetID, scope.WorkspaceID, scope.ProjectID, scope.CampaignID)
	item, err := scanAssetWithVersion(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AssetWithVersion{}, ErrNotFound
	}
	if err != nil {
		return domain.AssetWithVersion{}, err
	}
	if !allowed[item.Status] {
		return item, fmt.Errorf("asset %s is %s and is protected from cleanup", item.ID, item.Status)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM delivery_event WHERE asset_id = $1`, item.ID); err != nil {
		return domain.AssetWithVersion{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM review_event WHERE asset_id = $1`, item.ID); err != nil {
		return domain.AssetWithVersion{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM asset_version WHERE asset_id = $1`, item.ID); err != nil {
		return domain.AssetWithVersion{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM asset WHERE id = $1`, item.ID); err != nil {
		return domain.AssetWithVersion{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.AssetWithVersion{}, err
	}
	return item, nil
}

func (s *PostgresStore) FinishAttempt(ctx context.Context, attemptID, status string, result provider.Result, started time.Time, code, message *string, retryAfter *time.Time) error {
	latency := int(time.Since(started).Milliseconds())
	_, err := s.db.ExecContext(ctx, `
		UPDATE task_attempt
		SET status = $2, provider_request_id = $3, finished_at = now(), latency_ms = $4,
			error_code = $5, error_message = $6, raw_response_json = $7::jsonb, cost_json = $8::jsonb,
			retry_after = $9
		WHERE id = $1
	`, attemptID, status, result.ProviderRequestID, latency, code, message,
		jsonOrEmpty(result.RawResponse), jsonOrEmpty(result.CostRaw), retryAfter)
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.AssetWithVersion{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	item, err = reviewAssetTx(ctx, tx, item, action, reviewer, note)
	if err != nil {
		return domain.AssetWithVersion{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.AssetWithVersion{}, err
	}
	return item, nil
}

func (s *PostgresStore) ApplyBestOfSelection(ctx context.Context, selectedAssetID string, rejectedAssetIDs []string, reviewer, selectedNote, rejectedNote string) (domain.AssetWithVersion, error) {
	selectedItem, err := s.GetAssetWithVersion(ctx, selectedAssetID)
	if err != nil {
		return domain.AssetWithVersion{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.AssetWithVersion{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	selectedItem, err = reviewAssetTx(ctx, tx, selectedItem, "approve", reviewer, selectedNote)
	if err != nil {
		return domain.AssetWithVersion{}, err
	}

	seen := map[string]struct{}{
		selectedAssetID: {},
	}
	for _, assetID := range rejectedAssetIDs {
		assetID = strings.TrimSpace(assetID)
		if assetID == "" {
			continue
		}
		if _, exists := seen[assetID]; exists {
			continue
		}
		seen[assetID] = struct{}{}

		item, err := getAssetWithVersionTx(ctx, tx, assetID)
		if err != nil {
			return domain.AssetWithVersion{}, err
		}
		if item.TaskID != selectedItem.TaskID {
			return domain.AssetWithVersion{}, fmt.Errorf("asset %s does not belong to task %s", assetID, selectedItem.TaskID)
		}
		if _, err := reviewAssetTx(ctx, tx, item, "reject", reviewer, rejectedNote); err != nil {
			return domain.AssetWithVersion{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.AssetWithVersion{}, err
	}
	return selectedItem, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func taskSelect() string {
	return `SELECT ` + taskSelectColumns("") + ` FROM generation_task`
}

func taskSelectColumns(alias string) string {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	return prefix + `id, ` + prefix + `workspace_id, ` + prefix + `project_id, ` + prefix + `campaign_id, ` + prefix + `idempotency_key, ` + prefix + `input_hash,
		` + prefix + `title, ` + prefix + `purpose, ` + prefix + `prompt, ` + prefix + `negative_prompt, ` + prefix + `style_preset, ` + prefix + `aspect_ratio, ` + prefix + `output_format,
		` + prefix + `structured_input_json, ` + prefix + `provider, ` + prefix + `status, ` + prefix + `requested_count, ` + prefix + `created_by, ` + prefix + `trace_id,
		` + prefix + `created_at, ` + prefix + `updated_at, ` + prefix + `error_code, ` + prefix + `error_message`
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
	if err != nil {
		return task, inputHash, err
	}
	task.SelectionMode = selectionModeFromStructuredInput(task.StructuredInputJSON)
	if errorCode.Valid {
		task.ErrorCode = &errorCode.String
	}
	if errorMessage.Valid {
		task.ErrorMessage = &errorMessage.String
	}
	return task, inputHash, nil
}

func assetWithVersionSelect() string {
	return `SELECT a.id, a.workspace_id, a.project_id, a.campaign_id, a.task_id,
		a.name, a.type, a.current_version_id, a.status, a.created_at, a.updated_at,
		v.id, v.asset_id, v.version, v.status, v.file_path, v.thumbnail_path, v.metadata_path,
		v.object_key, v.public_url, v.mime_type, v.width, v.height, v.hash, v.provider, v.model,
		v.prompt, v.parameters_json, v.cost_json, v.created_at, t.structured_input_json
		FROM asset a
		JOIN asset_version v ON v.id = a.current_version_id
		JOIN generation_task t ON t.id = a.task_id`
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
		&item.Version.Prompt, &item.Version.ParametersJSON, &item.Version.CostJSON, &item.Version.CreatedAt,
		&item.TaskStructuredInputJSON)
	return item, err
}

func getAssetWithVersionTx(ctx context.Context, tx *sql.Tx, assetID string) (domain.AssetWithVersion, error) {
	row := tx.QueryRowContext(ctx, assetWithVersionSelect()+` WHERE a.id = $1`, assetID)
	item, err := scanAssetWithVersion(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.AssetWithVersion{}, ErrNotFound
	}
	return item, err
}

func reviewAssetTx(ctx context.Context, tx *sql.Tx, item domain.AssetWithVersion, action, reviewer, note string) (domain.AssetWithVersion, error) {
	nextStatus, err := nextStatusForReviewAction(action)
	if err != nil {
		return domain.AssetWithVersion{}, err
	}
	if item.Status == nextStatus {
		return item, nil
	}
	if !canReviewAssetTransition(item.Status, action) {
		return domain.AssetWithVersion{}, fmt.Errorf("asset %s is %s and cannot transition to %s", item.ID, item.Status, nextStatus)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE asset SET status = $2, updated_at = now() WHERE id = $1
	`, item.ID, nextStatus); err != nil {
		return domain.AssetWithVersion{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO review_event (id, asset_id, version_id, action, reviewer, note)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, domain.NewID("rev"), item.ID, item.Version.ID, action, reviewer, note); err != nil {
		return domain.AssetWithVersion{}, err
	}
	item.Status = nextStatus
	return item, nil
}

func nextStatusForReviewAction(action string) (string, error) {
	switch action {
	case "approve":
		return domain.AssetApproved, nil
	case "reject":
		return domain.AssetRejected, nil
	default:
		return "", fmt.Errorf("unknown review action %q", action)
	}
}

func canReviewAssetTransition(currentStatus, action string) bool {
	switch action {
	case "approve":
		return currentStatus == domain.AssetDraft || currentStatus == domain.AssetRejected
	case "reject":
		return currentStatus == domain.AssetDraft || currentStatus == domain.AssetApproved
	default:
		return false
	}
}

func jsonOrEmpty(raw []byte) []byte {
	if len(raw) == 0 || !json.Valid(raw) {
		return []byte(`{}`)
	}
	return raw
}

func selectionModeFromStructuredInput(raw []byte) string {
	var input struct {
		SelectionMode string `json:"selection_mode"`
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &input)
	}
	mode, ok := domain.NormalizeSelectionMode(input.SelectionMode)
	if !ok {
		return domain.SelectionManualOptional
	}
	return mode
}

type projectMetadata struct {
	QualityProfile domain.QualityProfile      `json:"quality_profile"`
	AccessConfig   domain.ProjectAccessConfig `json:"access_config"`
	ArchivedAt     *time.Time                 `json:"archived_at,omitempty"`
}

type scopeMetadata struct {
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
}

func parseScopeMetadata(raw []byte) (scopeMetadata, error) {
	var metadata scopeMetadata
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &metadata); err != nil {
			return scopeMetadata{}, err
		}
	}
	return metadata, nil
}

func (m scopeMetadata) Archived() bool {
	return m.ArchivedAt != nil
}

func (m *scopeMetadata) SetArchived(archived bool) {
	if !archived {
		m.ArchivedAt = nil
		return
	}
	now := time.Now().UTC()
	m.ArchivedAt = &now
}

func parseProjectMetadata(raw []byte) (projectMetadata, error) {
	var metadata projectMetadata
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &metadata); err != nil {
			return projectMetadata{}, err
		}
	}
	return metadata, nil
}

func qualityProfileFromProjectMetadata(raw []byte, stylePreset string) (domain.QualityProfile, error) {
	metadata, err := parseProjectMetadata(raw)
	if err != nil {
		return domain.QualityProfile{}, err
	}
	profile := metadata.QualityProfile
	if profile.StylePreset == "" {
		profile.StylePreset = stylePreset
	}
	return profile, nil
}

func accessConfigFromProjectMetadata(raw []byte) (domain.ProjectAccessConfig, error) {
	metadata, err := parseProjectMetadata(raw)
	if err != nil {
		return domain.ProjectAccessConfig{}, err
	}
	return metadata.AccessConfig.Normalize(), nil
}

func (m projectMetadata) Archived() bool {
	return m.ArchivedAt != nil
}

func (m *projectMetadata) SetArchived(archived bool) {
	if !archived {
		m.ArchivedAt = nil
		return
	}
	now := time.Now().UTC()
	m.ArchivedAt = &now
}

func (s *PostgresStore) workspaceExists(ctx context.Context, workspaceID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM workspace
			WHERE id = $1
		)
	`, workspaceID).Scan(&exists)
	return exists, err
}

func (s *PostgresStore) projectExists(ctx context.Context, workspaceID, projectID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM project
			WHERE workspace_id = $1 AND id = $2
		)
	`, workspaceID, projectID).Scan(&exists)
	return exists, err
}
