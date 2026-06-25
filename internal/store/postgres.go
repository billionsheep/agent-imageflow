package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
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

type scopedDeleteStatement struct {
	SQL  string
	Args []any
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

	if err := execScopedDeleteStatements(ctx, tx, workspaceCascadeDeleteStatements(workspaceID)); err != nil {
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

	if err := execScopedDeleteStatements(ctx, tx, projectCascadeDeleteStatements(workspaceID, projectID)); err != nil {
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

	if err := execScopedDeleteStatements(ctx, tx, campaignCascadeDeleteStatements(workspaceID, projectID, campaignID)); err != nil {
		return err
	}
	return tx.Commit()
}

func execScopedDeleteStatements(ctx context.Context, tx *sql.Tx, statements []scopedDeleteStatement) error {
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement.SQL, statement.Args...); err != nil {
			return err
		}
	}
	return nil
}

func workspaceCascadeDeleteStatements(workspaceID string) []scopedDeleteStatement {
	args := []any{workspaceID}
	return append(scopedAssetCascadeDeleteStatements("a.workspace_id = $1", "gt.workspace_id = $1", args),
		scopedDeleteStatement{SQL: `DELETE FROM campaign WHERE workspace_id = $1`, Args: args},
		scopedDeleteStatement{SQL: `DELETE FROM project WHERE workspace_id = $1`, Args: args},
		scopedDeleteStatement{SQL: `DELETE FROM workspace WHERE id = $1`, Args: args},
	)
}

func projectCascadeDeleteStatements(workspaceID, projectID string) []scopedDeleteStatement {
	args := []any{workspaceID, projectID}
	return append(scopedAssetCascadeDeleteStatements("a.workspace_id = $1 AND a.project_id = $2", "gt.workspace_id = $1 AND gt.project_id = $2", args),
		scopedDeleteStatement{SQL: `DELETE FROM campaign WHERE workspace_id = $1 AND project_id = $2`, Args: args},
		scopedDeleteStatement{SQL: `DELETE FROM project WHERE workspace_id = $1 AND id = $2`, Args: args},
	)
}

func campaignCascadeDeleteStatements(workspaceID, projectID, campaignID string) []scopedDeleteStatement {
	args := []any{workspaceID, projectID, campaignID}
	return append(scopedAssetCascadeDeleteStatements("a.workspace_id = $1 AND a.project_id = $2 AND a.campaign_id = $3", "gt.workspace_id = $1 AND gt.project_id = $2 AND gt.campaign_id = $3", args),
		scopedDeleteStatement{SQL: `DELETE FROM campaign WHERE workspace_id = $1 AND project_id = $2 AND id = $3`, Args: args},
	)
}

func scopedAssetCascadeDeleteStatements(assetPredicate, taskPredicate string, args []any) []scopedDeleteStatement {
	return []scopedDeleteStatement{
		{SQL: `DELETE FROM delivery_event de USING asset a WHERE de.asset_id = a.id AND ` + assetPredicate, Args: args},
		{SQL: `DELETE FROM review_event re USING asset a WHERE re.asset_id = a.id AND ` + assetPredicate, Args: args},
		{SQL: `DELETE FROM asset_version av USING asset a WHERE av.asset_id = a.id AND ` + assetPredicate, Args: args},
		{SQL: `DELETE FROM asset a WHERE ` + assetPredicate, Args: args},
		{SQL: `DELETE FROM task_attempt ta USING generation_task gt WHERE ta.task_id = gt.id AND ` + taskPredicate, Args: args},
		{SQL: `DELETE FROM generation_task gt WHERE ` + taskPredicate, Args: args},
	}
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

func (s *PostgresStore) GetProjectProviderProfile(ctx context.Context, workspaceID, projectID string) (domain.ProjectProviderProfile, error) {
	var metadataRaw []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT metadata_json
		FROM project
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, projectID).Scan(&metadataRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProjectProviderProfile{}, ErrNotFound
	}
	if err != nil {
		return domain.ProjectProviderProfile{}, err
	}
	return providerProfileFromProjectMetadata(metadataRaw)
}

func (s *PostgresStore) GetProjectVisualContext(ctx context.Context, workspaceID, projectID string) (domain.ProjectVisualContext, error) {
	var metadataRaw []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT metadata_json
		FROM project
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, projectID).Scan(&metadataRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProjectVisualContext{}, ErrNotFound
	}
	if err != nil {
		return domain.ProjectVisualContext{}, err
	}
	return visualContextFromProjectMetadata(metadataRaw)
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

func (s *PostgresStore) UpdateProjectProviderProfile(ctx context.Context, workspaceID, projectID string, profile domain.ProjectProviderProfile) (domain.ProjectProviderProfile, error) {
	profileRaw, err := json.Marshal(profile)
	if err != nil {
		return domain.ProjectProviderProfile{}, err
	}
	var metadataRaw []byte
	err = s.db.QueryRowContext(ctx, `
		UPDATE project
		SET metadata_json = jsonb_set(metadata_json, '{provider_profile}', $3::jsonb, true),
			updated_at = now()
		WHERE workspace_id = $1 AND id = $2
		RETURNING metadata_json
	`, workspaceID, projectID, profileRaw).Scan(&metadataRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProjectProviderProfile{}, ErrNotFound
	}
	if err != nil {
		return domain.ProjectProviderProfile{}, err
	}
	return providerProfileFromProjectMetadata(metadataRaw)
}

func (s *PostgresStore) UpdateProjectVisualContext(ctx context.Context, workspaceID, projectID string, visualContext domain.ProjectVisualContext) (domain.ProjectVisualContext, error) {
	contextRaw, err := json.Marshal(visualContext)
	if err != nil {
		return domain.ProjectVisualContext{}, err
	}
	var metadataRaw []byte
	err = s.db.QueryRowContext(ctx, `
		UPDATE project
		SET metadata_json = jsonb_set(metadata_json, '{visual_context}', $3::jsonb, true),
			updated_at = now()
		WHERE workspace_id = $1 AND id = $2
		RETURNING metadata_json
	`, workspaceID, projectID, contextRaw).Scan(&metadataRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ProjectVisualContext{}, ErrNotFound
	}
	if err != nil {
		return domain.ProjectVisualContext{}, err
	}
	return visualContextFromProjectMetadata(metadataRaw)
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

func (s *PostgresStore) GetSceneRegenerationSourceTask(ctx context.Context, taskID string) (domain.Task, error) {
	return s.GetTask(ctx, taskID)
}

func (s *PostgresStore) ResolveLatestSceneTask(ctx context.Context, projectID, campaignID string, identity domain.SceneIdentity) (domain.Task, error) {
	built := buildResolveLatestSceneTaskQuery(identity, projectID, campaignID)
	row := s.db.QueryRowContext(ctx, built.SQL, built.Args...)
	task, _, err := scanTaskWithHash(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Task{}, ErrNotFound
	}
	return task, err
}

func (s *PostgresStore) CountSceneRegenerations(ctx context.Context, projectID, campaignID string, identity domain.SceneIdentity) (int, error) {
	built := buildCountSceneRegenerationsQuery(identity, projectID, campaignID)
	var count int
	if err := s.db.QueryRowContext(ctx, built.SQL, built.Args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
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

func (s *PostgresStore) ListAssetsByCampaign(ctx context.Context, query domain.AssetListQuery) ([]domain.AssetWithVersion, error) {
	sqlText, args := buildListAssetsByCampaignQuery(query)
	rows, err := s.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssetRows(rows)
}

func (s *PostgresStore) ListRecentAssets(ctx context.Context, query domain.AssetListQuery) ([]domain.AssetWithVersion, error) {
	sqlText, args := buildListRecentAssetsQuery(query)
	rows, err := s.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAssetRows(rows)
}

func buildListAssetsByCampaignQuery(query domain.AssetListQuery) (string, []any) {
	limit := query.Limit
	if limit <= 0 {
		limit = domain.DefaultAssetListLimit
	}
	if limit > domain.MaxAssetListLimit {
		limit = domain.MaxAssetListLimit
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	args := []any{query.ProjectID, query.CampaignID}
	conditions := []string{"a.project_id = $1", "a.campaign_id = $2"}
	addStringCondition := func(sqlExpr string, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(sqlExpr, len(args)))
	}
	addStringCondition("a.status = $%d", query.Status)
	addStringCondition("v.provider = $%d", query.Provider)
	addStringCondition("v.model = $%d", query.Model)
	addStringCondition("t.structured_input_json->'metadata_json'->>'source' = $%d", query.Source)
	addStringCondition("t.structured_input_json->'metadata_json'->>'session_id' = $%d", query.SessionID)
	addStringCondition("t.structured_input_json->'metadata_json'->>'batch_id' = $%d", query.BatchID)
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		args = append(args, "%"+strings.ToLower(keyword)+"%")
		placeholder := len(args)
		conditions = append(conditions, fmt.Sprintf(`(
			LOWER(a.id) LIKE $%[1]d OR
			LOWER(a.task_id) LIKE $%[1]d OR
			LOWER(a.name) LIKE $%[1]d OR
			LOWER(v.prompt) LIKE $%[1]d OR
			LOWER(t.title) LIKE $%[1]d
		)`, placeholder))
	}
	if query.CreatedFrom != nil {
		args = append(args, *query.CreatedFrom)
		conditions = append(conditions, fmt.Sprintf("a.created_at >= $%d", len(args)))
	}
	if query.CreatedTo != nil {
		args = append(args, *query.CreatedTo)
		conditions = append(conditions, fmt.Sprintf("a.created_at <= $%d", len(args)))
	}
	args = append(args, limit)
	limitPlaceholder := len(args)
	args = append(args, offset)
	offsetPlaceholder := len(args)

	return assetWithVersionSelect() + fmt.Sprintf(`
		WHERE %s
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $%d OFFSET $%d
	`, strings.Join(conditions, " AND "), limitPlaceholder, offsetPlaceholder), args
}

func buildListRecentAssetsQuery(query domain.AssetListQuery) (string, []any) {
	limit := query.Limit
	if limit <= 0 {
		limit = domain.DefaultAssetListLimit
	}
	if limit > domain.MaxAssetListLimit {
		limit = domain.MaxAssetListLimit
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	args := []any{}
	conditions := []string{}
	addStringCondition := func(sqlExpr string, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(sqlExpr, len(args)))
	}
	addStringCondition("a.status = $%d", query.Status)
	addStringCondition("v.provider = $%d", query.Provider)
	addStringCondition("v.model = $%d", query.Model)
	addStringCondition("t.structured_input_json->'metadata_json'->>'source' = $%d", query.Source)
	addStringCondition("t.structured_input_json->'metadata_json'->>'session_id' = $%d", query.SessionID)
	addStringCondition("t.structured_input_json->'metadata_json'->>'batch_id' = $%d", query.BatchID)
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		args = append(args, "%"+strings.ToLower(keyword)+"%")
		placeholder := len(args)
		conditions = append(conditions, fmt.Sprintf(`(
			LOWER(a.id) LIKE $%[1]d OR
			LOWER(a.task_id) LIKE $%[1]d OR
			LOWER(a.name) LIKE $%[1]d OR
			LOWER(v.prompt) LIKE $%[1]d OR
			LOWER(t.title) LIKE $%[1]d
		)`, placeholder))
	}
	if query.CreatedFrom != nil {
		args = append(args, *query.CreatedFrom)
		conditions = append(conditions, fmt.Sprintf("a.created_at >= $%d", len(args)))
	}
	if query.CreatedTo != nil {
		args = append(args, *query.CreatedTo)
		conditions = append(conditions, fmt.Sprintf("a.created_at <= $%d", len(args)))
	}
	args = append(args, limit)
	limitPlaceholder := len(args)
	args = append(args, offset)
	offsetPlaceholder := len(args)

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	return assetWithVersionSelect() + fmt.Sprintf(`
		%s
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $%d OFFSET $%d
	`, where, limitPlaceholder, offsetPlaceholder), args
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

func (s *PostgresStore) ListTaskAttempts(ctx context.Context, taskID string) ([]domain.TaskAttempt, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, attempt_no, status, provider, provider_request_id,
			started_at, finished_at, latency_ms, queue_wait_ms, provider_first_byte_ms,
			provider_total_ms, response_download_ms, store_ms, thumbnail_ms, retry_count,
			error_stage, response_bytes, retry_after, error_code, error_message, raw_response_json
		FROM task_attempt
		WHERE task_id = $1
		ORDER BY attempt_no ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.TaskAttempt{}
	for rows.Next() {
		var item domain.TaskAttempt
		var providerRequestID sql.NullString
		var finishedAt sql.NullTime
		var latencyMs sql.NullInt64
		var queueWaitMs sql.NullInt64
		var providerFirstByteMs sql.NullInt64
		var providerTotalMs sql.NullInt64
		var responseDownloadMs sql.NullInt64
		var storeMs sql.NullInt64
		var thumbnailMs sql.NullInt64
		var retryCount sql.NullInt64
		var errorStage sql.NullString
		var responseBytes sql.NullInt64
		var retryAfter sql.NullTime
		var errorCode sql.NullString
		var errorMessage sql.NullString
		var rawResponse []byte
		if err := rows.Scan(
			&item.ID,
			&item.TaskID,
			&item.AttemptNo,
			&item.Status,
			&item.Provider,
			&providerRequestID,
			&item.StartedAt,
			&finishedAt,
			&latencyMs,
			&queueWaitMs,
			&providerFirstByteMs,
			&providerTotalMs,
			&responseDownloadMs,
			&storeMs,
			&thumbnailMs,
			&retryCount,
			&errorStage,
			&responseBytes,
			&retryAfter,
			&errorCode,
			&errorMessage,
			&rawResponse,
		); err != nil {
			return nil, err
		}
		if providerRequestID.Valid {
			item.ProviderRequestID = providerRequestID.String
		}
		if finishedAt.Valid {
			item.FinishedAt = &finishedAt.Time
		}
		if latencyMs.Valid {
			value := int(latencyMs.Int64)
			item.LatencyMs = &value
		}
		if queueWaitMs.Valid {
			value := int(queueWaitMs.Int64)
			item.QueueWaitMs = &value
		}
		if providerFirstByteMs.Valid {
			value := int(providerFirstByteMs.Int64)
			item.ProviderFirstByteMs = &value
		}
		if providerTotalMs.Valid {
			value := int(providerTotalMs.Int64)
			item.ProviderTotalMs = &value
		}
		if responseDownloadMs.Valid {
			value := int(responseDownloadMs.Int64)
			item.ResponseDownloadMs = &value
		}
		if storeMs.Valid {
			value := int(storeMs.Int64)
			item.StoreMs = &value
		}
		if thumbnailMs.Valid {
			value := int(thumbnailMs.Int64)
			item.ThumbnailMs = &value
		}
		if retryCount.Valid {
			item.RetryCount = int(retryCount.Int64)
		}
		if errorStage.Valid {
			item.ErrorStage = errorStage.String
		}
		if responseBytes.Valid {
			item.ResponseBytes = responseBytes.Int64
		}
		if retryAfter.Valid {
			item.RetryAfter = &retryAfter.Time
		}
		if errorCode.Valid {
			item.ErrorCode = &errorCode.String
		}
		if errorMessage.Valid {
			item.ErrorMessage = &errorMessage.String
		}
		applyAttemptResponseSummary(&item, rawResponse)
		items = append(items, item)
	}
	return items, rows.Err()
}

func applyAttemptResponseSummary(item *domain.TaskAttempt, raw []byte) {
	if item == nil || len(raw) == 0 {
		return
	}
	var summary struct {
		APIMode           string `json:"api_mode"`
		RequestMode       string `json:"request_mode"`
		PartialImageCount int    `json:"partial_image_count"`
	}
	if json.Unmarshal(raw, &summary) != nil {
		return
	}
	item.APIMode = strings.TrimSpace(summary.APIMode)
	item.RequestMode = strings.TrimSpace(summary.RequestMode)
	if summary.PartialImageCount > 0 {
		item.Stream = true
		item.PartialImageCount = summary.PartialImageCount
	}
	if strings.Contains(item.RequestMode, "stream") {
		item.Stream = true
	}
}

func (s *PostgresStore) GetBatchProgress(ctx context.Context, query domain.BatchProgressQuery) (domain.BatchProgressResponse, error) {
	query.ProjectID = strings.TrimSpace(query.ProjectID)
	query.CampaignID = strings.TrimSpace(query.CampaignID)
	query.SessionID = strings.TrimSpace(query.SessionID)
	query.BatchID = strings.TrimSpace(query.BatchID)
	if query.Limit <= 0 {
		query.Limit = domain.DefaultBatchProgressLimit
	}
	if query.Limit > domain.MaxBatchProgressLimit {
		query.Limit = domain.MaxBatchProgressLimit
	}
	if query.ProjectID == "" || query.CampaignID == "" {
		return domain.BatchProgressResponse{}, fmt.Errorf("project_id and campaign_id are required")
	}
	if query.SessionID == "" && query.BatchID == "" {
		return domain.BatchProgressResponse{}, fmt.Errorf("session_id or batch_id is required")
	}

	args := []any{query.ProjectID, query.CampaignID}
	conditions := []string{"gt.project_id = $1", "gt.campaign_id = $2"}
	if query.SessionID != "" {
		args = append(args, query.SessionID)
		conditions = append(conditions, fmt.Sprintf("gt.structured_input_json->'metadata_json'->>'session_id' = $%d", len(args)))
	}
	if query.BatchID != "" {
		args = append(args, query.BatchID)
		conditions = append(conditions, fmt.Sprintf("gt.structured_input_json->'metadata_json'->>'batch_id' = $%d", len(args)))
	}
	args = append(args, query.Limit)
	limitPlaceholder := len(args)

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT gt.id, gt.status, gt.created_at, gt.updated_at, gt.error_code, gt.error_message,
			COUNT(DISTINCT a.id) AS asset_count,
			COUNT(DISTINCT ta.id) AS attempt_count,
			COALESCE(BOOL_OR(ta.retry_after > now()), false) AS retrying,
			COALESCE(MAX(NULLIF(ta.error_stage, '')), '') AS error_stage
		FROM generation_task gt
		LEFT JOIN asset a ON a.task_id = gt.id
		LEFT JOIN task_attempt ta ON ta.task_id = gt.id
		WHERE %s
		GROUP BY gt.id, gt.status, gt.created_at, gt.updated_at, gt.error_code, gt.error_message
		ORDER BY gt.created_at DESC, gt.id DESC
		LIMIT $%d
	`, strings.Join(conditions, " AND "), limitPlaceholder), args...)
	if err != nil {
		return domain.BatchProgressResponse{}, err
	}
	defer rows.Close()

	response := domain.BatchProgressResponse{
		GeneratedAt: time.Now().UTC(),
		ProjectID:   query.ProjectID,
		CampaignID:  query.CampaignID,
		SessionID:   query.SessionID,
		BatchID:     query.BatchID,
		Tasks:       []domain.BatchProgressTask{},
	}
	for rows.Next() {
		var item domain.BatchProgressTask
		var assetCount int64
		var attemptCount int64
		var errorStage string
		var errorCode sql.NullString
		var errorMessage sql.NullString
		if err := rows.Scan(
			&item.TaskID,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
			&errorCode,
			&errorMessage,
			&assetCount,
			&attemptCount,
			&item.Retrying,
			&errorStage,
		); err != nil {
			return domain.BatchProgressResponse{}, err
		}
		item.AssetCount = int(assetCount)
		item.AttemptCount = int(attemptCount)
		item.ErrorStage = strings.TrimSpace(errorStage)
		if errorCode.Valid {
			item.ErrorCode = &errorCode.String
		}
		if errorMessage.Valid {
			item.ErrorMessage = &errorMessage.String
		}
		response.Tasks = append(response.Tasks, item)
		addBatchProgressTaskCounts(&response.Counts, item)
	}
	if err := rows.Err(); err != nil {
		return domain.BatchProgressResponse{}, err
	}
	return response, nil
}

type batchStorySummarySQLQuery struct {
	SQL  string
	Args []any
}

func buildBatchStorySummaryBaseConditions(query domain.BatchStorySummaryQuery) ([]string, []any) {
	args := []any{query.ProjectID, query.CampaignID}
	conditions := []string{"gt.project_id = $1", "gt.campaign_id = $2"}
	addStringCondition := func(sqlExpr string, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(sqlExpr, len(args)))
	}
	addStringCondition("gt.structured_input_json->'metadata_json'->>'session_id' = $%d", query.SessionID)
	addStringCondition("gt.structured_input_json->'metadata_json'->>'batch_id' = $%d", query.BatchID)
	addStringCondition("gt.structured_input_json->'metadata_json'->>'story_id' = $%d", query.StoryID)
	addStringCondition("gt.structured_input_json->'metadata_json'->>'source' = $%d", query.Source)
	addStringCondition("gt.status = $%d", query.Status)
	return conditions, args
}

func batchStorySummaryExclusionCondition(alias string) string {
	metadata := alias + ".structured_input_json->'metadata_json'"
	return fmt.Sprintf(`(
		COALESCE(NULLIF(%[1]s->>'scene_id', ''), '') = '' OR
		COALESCE(%[1]s->>'task_role', '') IN ('setup', 'reference_setup', 'visual_context_setup', 'calibration') OR
		LOWER(COALESCE(%[1]s->>'exclude_from_story_summary', 'false')) = 'true'
	)`, metadata)
}

func buildBatchStorySummaryTasksQuery(query domain.BatchStorySummaryQuery) batchStorySummarySQLQuery {
	conditions, args := buildBatchStorySummaryBaseConditions(query)
	if !query.IncludeSetup {
		conditions = append(conditions, "NOT "+batchStorySummaryExclusionCondition("gt"))
	}
	args = append(args, query.Limit)
	limitPlaceholder := len(args)

	return batchStorySummarySQLQuery{
		SQL: fmt.Sprintf(`
		WITH limited_tasks AS (
			SELECT gt.*
			FROM generation_task gt
			WHERE %s
			ORDER BY gt.created_at ASC, gt.id ASC
			LIMIT $%d
		)
		SELECT gt.id, gt.status, gt.created_at, gt.updated_at, gt.error_code, gt.error_message,
			gt.structured_input_json,
			COALESCE(gt.structured_input_json->'metadata_json'->>'source', '') AS source,
			COALESCE(gt.structured_input_json->'metadata_json'->>'session_id', '') AS session_id,
			COALESCE(gt.structured_input_json->'metadata_json'->>'batch_id', '') AS batch_id,
			COALESCE(gt.structured_input_json->'metadata_json'->>'story_id', '') AS story_id,
			COALESCE(gt.structured_input_json->'metadata_json'->>'scene_id', '') AS scene_id,
			COALESCE(gt.structured_input_json->'metadata_json'->>'target_path', '') AS task_target_path,
			COALESCE(gt.structured_input_json->'metadata_json'->>'scene_order', '') AS scene_order,
			COALESCE(gt.structured_input_json->'metadata_json'->>'regenerated_from_task_id', '') AS regenerated_from_task_id,
			COUNT(a.id) OVER (PARTITION BY gt.id) AS task_asset_count,
			COALESCE(attempts.attempt_count, 0) AS attempt_count,
			COALESCE(attempts.retrying, false) AS retrying,
			COALESCE(attempts.error_stage, '') AS error_stage,
			a.id AS asset_id, a.status AS asset_status, a.created_at AS asset_created_at,
			COALESCE(v.provider, '') AS asset_provider,
			COALESCE(v.model, '') AS asset_model,
			COALESCE(v.prompt, '') AS asset_prompt
		FROM limited_tasks gt
		LEFT JOIN (
			SELECT task_id,
				COUNT(*) AS attempt_count,
				BOOL_OR(retry_after > now()) AS retrying,
				MAX(NULLIF(error_stage, '')) AS error_stage
			FROM task_attempt
			GROUP BY task_id
		) attempts ON attempts.task_id = gt.id
		LEFT JOIN asset a ON a.task_id = gt.id
		LEFT JOIN asset_version v ON v.id = a.current_version_id
		ORDER BY gt.created_at ASC, gt.id ASC, a.created_at ASC, a.id ASC
	`, strings.Join(conditions, " AND "), limitPlaceholder),
		Args: args,
	}
}

func buildBatchStorySummaryExcludedCountQuery(query domain.BatchStorySummaryQuery) batchStorySummarySQLQuery {
	conditions, args := buildBatchStorySummaryBaseConditions(query)
	conditions = append(conditions, batchStorySummaryExclusionCondition("gt"))
	return batchStorySummarySQLQuery{
		SQL: fmt.Sprintf(`
		SELECT COUNT(*)
		FROM generation_task gt
		WHERE %s
	`, strings.Join(conditions, " AND ")),
		Args: args,
	}
}

func buildSceneIdentityConditions(identity domain.SceneIdentity, projectID, campaignID string) ([]string, []any) {
	conditions := []string{}
	args := []any{}
	addStringCondition := func(format string, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(format, len(args)))
	}
	addStringCondition("project_id = $%d", projectID)
	addStringCondition("campaign_id = $%d", campaignID)
	addStringCondition("structured_input_json->'metadata_json'->>'session_id' = $%d", identity.SessionID)
	addStringCondition("structured_input_json->'metadata_json'->>'batch_id' = $%d", identity.BatchID)
	addStringCondition("structured_input_json->'metadata_json'->>'story_id' = $%d", identity.StoryID)
	addStringCondition("structured_input_json->'metadata_json'->>'scene_id' = $%d", identity.SceneID)
	addStringCondition("structured_input_json->'metadata_json'->>'source' = $%d", identity.Source)
	if len(conditions) == 0 {
		conditions = append(conditions, "1=0")
	}
	return conditions, args
}

func buildResolveLatestSceneTaskQuery(identity domain.SceneIdentity, projectID, campaignID string) batchStorySummarySQLQuery {
	conditions, args := buildSceneIdentityConditions(identity, projectID, campaignID)
	return batchStorySummarySQLQuery{
		SQL: taskSelect() + `
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY created_at DESC, id DESC
		LIMIT 1`,
		Args: args,
	}
}

func buildCountSceneRegenerationsQuery(identity domain.SceneIdentity, projectID, campaignID string) batchStorySummarySQLQuery {
	conditions, args := buildSceneIdentityConditions(identity, projectID, campaignID)
	conditions = append(conditions, "COALESCE(structured_input_json->'metadata_json'->>'regenerated_from_task_id', '') <> ''")
	return batchStorySummarySQLQuery{
		SQL: `
		SELECT COUNT(*)
		FROM generation_task
		WHERE ` + strings.Join(conditions, " AND "),
		Args: args,
	}
}

func (s *PostgresStore) GetBatchStorySummary(ctx context.Context, query domain.BatchStorySummaryQuery) (domain.BatchStorySummaryResponse, error) {
	query.ProjectID = strings.TrimSpace(query.ProjectID)
	query.CampaignID = strings.TrimSpace(query.CampaignID)
	query.SessionID = strings.TrimSpace(query.SessionID)
	query.BatchID = strings.TrimSpace(query.BatchID)
	query.StoryID = strings.TrimSpace(query.StoryID)
	query.Source = strings.TrimSpace(query.Source)
	query.Status = strings.TrimSpace(query.Status)
	if query.Limit <= 0 {
		query.Limit = domain.DefaultBatchProgressLimit
	}
	if query.Limit > domain.MaxBatchProgressLimit {
		query.Limit = domain.MaxBatchProgressLimit
	}
	if query.ProjectID == "" || query.CampaignID == "" {
		return domain.BatchStorySummaryResponse{}, fmt.Errorf("project_id and campaign_id are required")
	}
	if query.SessionID == "" && query.BatchID == "" {
		return domain.BatchStorySummaryResponse{}, fmt.Errorf("session_id or batch_id is required")
	}

	excludedQuery := buildBatchStorySummaryExcludedCountQuery(query)
	var excludedSetupTaskCount int
	if err := s.db.QueryRowContext(ctx, excludedQuery.SQL, excludedQuery.Args...).Scan(&excludedSetupTaskCount); err != nil {
		return domain.BatchStorySummaryResponse{}, err
	}

	tasksQuery := buildBatchStorySummaryTasksQuery(query)
	rows, err := s.db.QueryContext(ctx, tasksQuery.SQL, tasksQuery.Args...)
	if err != nil {
		return domain.BatchStorySummaryResponse{}, err
	}
	defer rows.Close()

	response := domain.BatchStorySummaryResponse{
		GeneratedAt: time.Now().UTC(),
		ProjectID:   query.ProjectID,
		CampaignID:  query.CampaignID,
		SessionID:   query.SessionID,
		BatchID:     query.BatchID,
		Source:      query.Source,
		StoryID:     query.StoryID,
		Stories:     []domain.BatchStorySummaryStory{},
		Scenes:      []domain.BatchStorySummaryScene{},
	}
	response.Counts.ExcludedSetupTaskCount = excludedSetupTaskCount

	sceneByKey := map[string]*domain.BatchStorySummaryScene{}
	taskSeen := map[string]bool{}
	for rows.Next() {
		row, err := scanBatchStorySummaryRow(rows)
		if err != nil {
			return domain.BatchStorySummaryResponse{}, err
		}
		if strings.TrimSpace(row.SceneID) == "" {
			continue
		}
		key := row.StoryID + "\x00" + row.SceneID
		scene := sceneByKey[key]
		if scene == nil {
			scene = &domain.BatchStorySummaryScene{
				StoryID:       row.StoryID,
				SceneID:       row.SceneID,
				SceneOrder:    deriveSceneOrder(row.SceneID, row.SceneOrder),
				TargetPath:    strings.TrimSpace(row.TaskTargetPath),
				Status:        "empty",
				VisualContext: extractBatchStoryVisualContext(row.StructuredInputJSON),
				Tasks:         []domain.BatchStorySummaryTask{},
				Assets:        []domain.BatchStorySummaryAsset{},
			}
			sceneByKey[key] = scene
		}
		if !taskSeen[row.TaskID] {
			taskSeen[row.TaskID] = true
			previousLatestUpdatedAt := latestTaskUpdatedAt(scene.Tasks)
			task := domain.BatchStorySummaryTask{
				TaskID:       row.TaskID,
				Status:       row.Status,
				AssetCount:   row.TaskAssetCount,
				AttemptCount: row.AttemptCount,
				Retrying:     row.Retrying,
				ErrorStage:   strings.TrimSpace(row.ErrorStage),
				CreatedAt:    row.CreatedAt,
				UpdatedAt:    row.UpdatedAt,
			}
			if row.ErrorCode.Valid {
				task.ErrorCode = &row.ErrorCode.String
			}
			if row.ErrorMessage.Valid {
				task.ErrorMessage = &row.ErrorMessage.String
			}
			scene.Tasks = append(scene.Tasks, task)
			scene.Counts.TaskCount++
			scene.Counts.AttemptCount += task.AttemptCount
			addBatchStorySummaryTaskCounts(&response.Counts, task)
			if row.RegeneratedFromTaskID != "" {
				scene.RegeneratedFromTaskID = row.RegeneratedFromTaskID
				scene.RegenerationCount++
			}
			if scene.LatestTaskID == "" || row.UpdatedAt.After(previousLatestUpdatedAt) || (row.UpdatedAt.Equal(previousLatestUpdatedAt) && row.TaskID > scene.LatestTaskID) {
				scene.LatestTaskID = row.TaskID
			}
			if scene.TargetPath == "" && strings.TrimSpace(row.TaskTargetPath) != "" {
				scene.TargetPath = strings.TrimSpace(row.TaskTargetPath)
			}
			if isBatchStoryVisualContextEmpty(scene.VisualContext) {
				scene.VisualContext = extractBatchStoryVisualContext(row.StructuredInputJSON)
			}
		}
		if row.AssetID.Valid {
			status := publicAssetStatus(row.AssetStatus.String)
			asset := domain.BatchStorySummaryAsset{
				AssetID:      row.AssetID.String,
				TaskID:       row.TaskID,
				Status:       status,
				Provider:     strings.TrimSpace(row.AssetProvider),
				Model:        strings.TrimSpace(row.AssetModel),
				Prompt:       strings.TrimSpace(row.AssetPrompt),
				DownloadURL:  "/api/assets/" + row.AssetID.String + "/original",
				ThumbnailURL: "/api/assets/" + row.AssetID.String + "/thumbnail",
				MetadataURL:  "/api/assets/" + row.AssetID.String + "/metadata",
				TargetPath:   firstNonEmpty(scene.TargetPath, strings.TrimSpace(row.TaskTargetPath)),
				CreatedAt:    row.AssetCreatedAt.Time,
			}
			scene.Assets = append(scene.Assets, asset)
			addBatchStorySummaryAssetCounts(&response.Counts, &scene.Counts, asset)
			if status == "selected" {
				if scene.PrimarySelectedAssetID == "" || row.AssetCreatedAt.Time.After(primarySelectedAssetTime(scene.Assets, scene.PrimarySelectedAssetID)) || (row.AssetCreatedAt.Time.Equal(primarySelectedAssetTime(scene.Assets, scene.PrimarySelectedAssetID)) && row.AssetID.String > scene.PrimarySelectedAssetID) {
					scene.PrimarySelectedAssetID = row.AssetID.String
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return domain.BatchStorySummaryResponse{}, err
	}

	for _, scene := range sceneByKey {
		scene.Status = deriveBatchStorySceneStatus(*scene)
		response.Scenes = append(response.Scenes, *scene)
	}
	sort.Slice(response.Scenes, func(i, j int) bool {
		left, right := response.Scenes[i], response.Scenes[j]
		if left.StoryID != right.StoryID {
			return left.StoryID < right.StoryID
		}
		if left.SceneOrder != right.SceneOrder {
			if left.SceneOrder == 0 {
				return false
			}
			if right.SceneOrder == 0 {
				return true
			}
			return left.SceneOrder < right.SceneOrder
		}
		return left.SceneID < right.SceneID
	})
	addBatchStorySummarySceneCounts(&response)
	response.Stories = buildBatchStorySummaryStories(response.Scenes)
	response.Counts.StoryCount = len(response.Stories)
	return response, nil
}

type batchStorySummaryRow struct {
	TaskID                string
	Status                string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	ErrorCode             sql.NullString
	ErrorMessage          sql.NullString
	StructuredInputJSON   json.RawMessage
	StoryID               string
	SceneID               string
	TaskTargetPath        string
	SceneOrder            string
	RegeneratedFromTaskID string
	TaskAssetCount        int
	AttemptCount          int
	Retrying              bool
	ErrorStage            string
	AssetID               sql.NullString
	AssetStatus           sql.NullString
	AssetCreatedAt        sql.NullTime
	AssetProvider         string
	AssetModel            string
	AssetPrompt           string
}

func scanBatchStorySummaryRow(rows *sql.Rows) (batchStorySummaryRow, error) {
	var row batchStorySummaryRow
	var source, sessionID, batchID string
	err := rows.Scan(
		&row.TaskID,
		&row.Status,
		&row.CreatedAt,
		&row.UpdatedAt,
		&row.ErrorCode,
		&row.ErrorMessage,
		&row.StructuredInputJSON,
		&source,
		&sessionID,
		&batchID,
		&row.StoryID,
		&row.SceneID,
		&row.TaskTargetPath,
		&row.SceneOrder,
		&row.RegeneratedFromTaskID,
		&row.TaskAssetCount,
		&row.AttemptCount,
		&row.Retrying,
		&row.ErrorStage,
		&row.AssetID,
		&row.AssetStatus,
		&row.AssetCreatedAt,
		&row.AssetProvider,
		&row.AssetModel,
		&row.AssetPrompt,
	)
	return row, err
}

func addBatchStorySummaryTaskCounts(counts *domain.BatchStorySummaryCounts, item domain.BatchStorySummaryTask) {
	counts.TaskCount++
	counts.AttemptCount += item.AttemptCount
	if item.Retrying {
		counts.RetryingCount++
	}
	switch item.Status {
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

func addBatchStorySummaryAssetCounts(total *domain.BatchStorySummaryCounts, scene *domain.BatchStorySceneCounts, item domain.BatchStorySummaryAsset) {
	total.AssetCount++
	scene.AssetCount++
	switch item.Status {
	case "generated":
		total.GeneratedAssetCount++
		scene.GeneratedAssetCount++
	case "selected":
		total.SelectedAssetCount++
		scene.SelectedAssetCount++
	case domain.AssetRejected:
		total.RejectedAssetCount++
		scene.RejectedAssetCount++
	}
}

func addBatchStorySummarySceneCounts(response *domain.BatchStorySummaryResponse) {
	response.Counts.SceneCount = len(response.Scenes)
	for i := range response.Scenes {
		scene := &response.Scenes[i]
		for _, task := range scene.Tasks {
			switch task.Status {
			case domain.TaskCompleted:
				scene.Counts.SucceededCount++
			case domain.TaskFailed, domain.TaskEnqueueFailed:
				scene.Counts.FailedCount++
			}
		}
		if scene.PrimarySelectedAssetID != "" {
			response.Counts.SceneWithSelectedCount++
		} else {
			response.Counts.SceneMissingSelectedCount++
		}
	}
}

func buildBatchStorySummaryStories(scenes []domain.BatchStorySummaryScene) []domain.BatchStorySummaryStory {
	storyMap := map[string]*domain.BatchStorySummaryStory{}
	order := []string{}
	for _, scene := range scenes {
		story := storyMap[scene.StoryID]
		if story == nil {
			story = &domain.BatchStorySummaryStory{StoryID: scene.StoryID}
			storyMap[scene.StoryID] = story
			order = append(order, scene.StoryID)
		}
		story.SceneCount++
		if scene.PrimarySelectedAssetID != "" {
			story.SelectedSceneCount++
		}
		story.Scenes = append(story.Scenes, scene.SceneID)
	}
	sort.Strings(order)
	stories := make([]domain.BatchStorySummaryStory, 0, len(order))
	for _, storyID := range order {
		stories = append(stories, *storyMap[storyID])
	}
	return stories
}

func deriveBatchStorySceneStatus(scene domain.BatchStorySummaryScene) string {
	if len(scene.Tasks) == 0 {
		return "empty"
	}
	anyRetrying := false
	anyRunning := false
	anyQueued := false
	anyPartial := false
	allFailed := true
	allCompleted := true
	hasCompleted := false
	hasFailed := false
	for _, task := range scene.Tasks {
		if task.Retrying {
			anyRetrying = true
		}
		switch task.Status {
		case domain.TaskRunning:
			anyRunning = true
			allFailed = false
			allCompleted = false
		case domain.TaskQueued:
			anyQueued = true
			allFailed = false
			allCompleted = false
		case domain.TaskPartiallyCompleted:
			anyPartial = true
			allFailed = false
			allCompleted = false
		case domain.TaskFailed, domain.TaskEnqueueFailed:
			hasFailed = true
			allCompleted = false
		case domain.TaskCompleted:
			hasCompleted = true
			allFailed = false
		default:
			allFailed = false
			allCompleted = false
		}
	}
	switch {
	case anyRetrying:
		return "retrying"
	case anyRunning:
		return "running"
	case anyQueued:
		return "queued"
	case anyPartial || (hasCompleted && hasFailed):
		return "partial"
	case allFailed:
		return "failed"
	case allCompleted && len(scene.Assets) > 0:
		return "completed"
	default:
		return "empty"
	}
}

func publicAssetStatus(status string) string {
	switch status {
	case domain.AssetDraft:
		return "generated"
	case domain.AssetApproved:
		return "selected"
	default:
		return status
	}
}

var trailingSceneNumberPattern = regexp.MustCompile(`(\d+)$`)

func deriveSceneOrder(sceneID, rawOrder string) int {
	rawOrder = strings.TrimSpace(rawOrder)
	if rawOrder != "" {
		if parsed, err := strconv.Atoi(rawOrder); err == nil && parsed > 0 {
			return parsed
		}
	}
	matches := trailingSceneNumberPattern.FindStringSubmatch(strings.TrimSpace(sceneID))
	if len(matches) == 2 {
		if parsed, err := strconv.Atoi(matches[1]); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}

func extractBatchStoryVisualContext(raw json.RawMessage) domain.BatchStoryVisualContext {
	var payload struct {
		CharacterIDs      []string `json:"character_ids"`
		ReferenceAssetIDs []string `json:"reference_asset_ids"`
		PromptRecipeID    string   `json:"prompt_recipe_id"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &payload) != nil {
		return domain.BatchStoryVisualContext{}
	}
	return domain.BatchStoryVisualContext{
		CharacterIDs:      trimStringSlice(payload.CharacterIDs),
		ReferenceAssetIDs: trimStringSlice(payload.ReferenceAssetIDs),
		PromptRecipeID:    strings.TrimSpace(payload.PromptRecipeID),
	}
}

func trimStringSlice(values []string) []string {
	cleaned := []string{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func isBatchStoryVisualContextEmpty(value domain.BatchStoryVisualContext) bool {
	return len(value.CharacterIDs) == 0 && len(value.ReferenceAssetIDs) == 0 && strings.TrimSpace(value.PromptRecipeID) == ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func latestTaskUpdatedAt(tasks []domain.BatchStorySummaryTask) time.Time {
	var latest time.Time
	for _, task := range tasks {
		if task.UpdatedAt.After(latest) {
			latest = task.UpdatedAt
		}
	}
	return latest
}

func primarySelectedAssetTime(assets []domain.BatchStorySummaryAsset, assetID string) time.Time {
	for _, asset := range assets {
		if asset.AssetID == assetID {
			return asset.CreatedAt
		}
	}
	return time.Time{}
}

func addBatchProgressTaskCounts(counts *domain.BatchProgressCounts, item domain.BatchProgressTask) {
	counts.TaskCount++
	counts.AssetCount += item.AssetCount
	counts.AttemptCount += item.AttemptCount
	if item.Retrying {
		counts.RetryingCount++
	}
	switch item.Status {
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

func (s *PostgresStore) ListCleanupAssetCandidates(ctx context.Context, opts domain.CleanupDryRunOptions) ([]domain.AssetWithVersion, error) {
	statuses := []string{}
	if opts.IncludeRejected {
		statuses = append(statuses, domain.AssetRejected)
	}
	if opts.IncludeGenerated {
		statuses = append(statuses, domain.AssetDraft)
	}
	if opts.IncludeDeprecated {
		statuses = append(statuses, domain.AssetDeprecated)
	}
	if len(statuses) == 0 {
		return nil, nil
	}
	if opts.Limit < 1 {
		opts.Limit = 100
	}
	args := []any{opts.Scope.WorkspaceID, opts.Scope.ProjectID, opts.Scope.CampaignID}
	placeholders := make([]string, 0, len(statuses))
	for _, status := range statuses {
		args = append(args, status)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
	}
	conditions := []string{
		"a.workspace_id = $1",
		"a.project_id = $2",
		"a.campaign_id = $3",
		fmt.Sprintf("a.status IN (%s)", strings.Join(placeholders, ",")),
	}
	addStringCondition := func(sql string, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(sql, len(args)))
	}
	addStringCondition("a.id = $%d", opts.AssetID)
	addStringCondition("a.task_id = $%d", opts.TaskID)
	addStringCondition("t.structured_input_json->'metadata_json'->>'session_id' = $%d", opts.SessionID)
	addStringCondition("t.structured_input_json->'metadata_json'->>'batch_id' = $%d", opts.BatchID)
	addStringCondition("t.structured_input_json->'metadata_json'->>'story_id' = $%d", opts.StoryID)
	args = append(args, opts.Limit)
	rows, err := s.db.QueryContext(ctx, assetWithVersionSelect()+fmt.Sprintf(`
		WHERE %s
		ORDER BY a.updated_at ASC, a.id ASC
		LIMIT $%d
	`, strings.Join(conditions, " AND "), len(args)), args...)
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

func (s *PostgresStore) FinishAttempt(ctx context.Context, attemptID, status string, result provider.Result, started time.Time, metrics domain.AttemptMetrics, code, message *string, retryAfter *time.Time) error {
	latency := int(time.Since(started).Milliseconds())
	_, err := s.db.ExecContext(ctx, `
		UPDATE task_attempt
		SET status = $2, provider_request_id = $3, finished_at = now(), latency_ms = $4,
			error_code = $5, error_message = $6, raw_response_json = $7::jsonb, cost_json = $8::jsonb,
			retry_after = $9, queue_wait_ms = $10, provider_first_byte_ms = $11,
			provider_total_ms = $12, response_download_ms = $13, store_ms = $14,
			thumbnail_ms = $15, retry_count = $16, error_stage = $17, response_bytes = $18
		WHERE id = $1
	`, attemptID, status, result.ProviderRequestID, latency, code, message,
		jsonOrEmpty(result.RawResponse), jsonOrEmpty(result.CostRaw), retryAfter,
		nullablePositiveInt(metrics.QueueWaitMs), nullablePositiveInt(metrics.ProviderFirstByteMs),
		nullablePositiveInt(metrics.ProviderTotalMs), nullablePositiveInt(metrics.ResponseDownloadMs),
		nullablePositiveInt(metrics.StoreMs), nullablePositiveInt(metrics.ThumbnailMs),
		metrics.RetryCount, strings.TrimSpace(metrics.ErrorStage), metrics.ResponseBytes)
	return err
}

func nullablePositiveInt(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
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
	QualityProfile  domain.QualityProfile         `json:"quality_profile"`
	AccessConfig    domain.ProjectAccessConfig    `json:"access_config"`
	ProviderProfile domain.ProjectProviderProfile `json:"provider_profile"`
	VisualContext   domain.ProjectVisualContext   `json:"visual_context"`
	ArchivedAt      *time.Time                    `json:"archived_at,omitempty"`
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

func providerProfileFromProjectMetadata(raw []byte) (domain.ProjectProviderProfile, error) {
	metadata, err := parseProjectMetadata(raw)
	if err != nil {
		return domain.ProjectProviderProfile{}, err
	}
	return metadata.ProviderProfile, nil
}

func visualContextFromProjectMetadata(raw []byte) (domain.ProjectVisualContext, error) {
	metadata, err := parseProjectMetadata(raw)
	if err != nil {
		return domain.ProjectVisualContext{}, err
	}
	return metadata.VisualContext, nil
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
