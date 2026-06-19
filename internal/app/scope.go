package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func (s *Service) ListWorkspaces(ctx context.Context) (domain.WorkspaceListResponse, error) {
	items, err := s.store.ListWorkspaces(ctx)
	if err != nil {
		return domain.WorkspaceListResponse{}, err
	}
	return domain.WorkspaceListResponse{Workspaces: items}, nil
}

func (s *Service) CreateWorkspace(ctx context.Context, req domain.CreateWorkspaceRequest) (domain.WorkspaceSummary, error) {
	req.WorkspaceID = normalizeRequiredField(req.WorkspaceID)
	req.Name = normalizeRequiredField(req.Name)
	if req.WorkspaceID == "" {
		return domain.WorkspaceSummary{}, fmt.Errorf("workspace_id is required")
	}
	if req.Name == "" {
		return domain.WorkspaceSummary{}, fmt.Errorf("name is required")
	}
	return s.store.CreateWorkspace(ctx, req)
}

func (s *Service) ListProjects(ctx context.Context, workspaceID string) (domain.ProjectListResponse, error) {
	workspaceID = normalizeRequiredField(workspaceID)
	if workspaceID == "" {
		return domain.ProjectListResponse{}, fmt.Errorf("workspace_id is required")
	}
	items, err := s.store.ListProjects(ctx, workspaceID)
	if err != nil {
		return domain.ProjectListResponse{}, err
	}
	return domain.ProjectListResponse{
		WorkspaceID: workspaceID,
		Projects:    items,
	}, nil
}

func (s *Service) CreateProject(ctx context.Context, workspaceID string, req domain.CreateProjectRequest) (domain.ProjectSummary, error) {
	workspaceID = normalizeRequiredField(workspaceID)
	req.ProjectID = normalizeRequiredField(req.ProjectID)
	req.Name = normalizeRequiredField(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if workspaceID == "" {
		return domain.ProjectSummary{}, fmt.Errorf("workspace_id is required")
	}
	if req.ProjectID == "" {
		return domain.ProjectSummary{}, fmt.Errorf("project_id is required")
	}
	if req.Name == "" {
		return domain.ProjectSummary{}, fmt.Errorf("name is required")
	}
	return s.store.CreateProject(ctx, workspaceID, req)
}

func (s *Service) ListCampaigns(ctx context.Context, workspaceID, projectID string) (domain.CampaignListResponse, error) {
	workspaceID = normalizeRequiredField(workspaceID)
	projectID = normalizeRequiredField(projectID)
	if workspaceID == "" {
		return domain.CampaignListResponse{}, fmt.Errorf("workspace_id is required")
	}
	if projectID == "" {
		return domain.CampaignListResponse{}, fmt.Errorf("project_id is required")
	}
	items, err := s.store.ListCampaigns(ctx, workspaceID, projectID)
	if err != nil {
		return domain.CampaignListResponse{}, err
	}
	return domain.CampaignListResponse{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		Campaigns:   items,
	}, nil
}

func (s *Service) CreateCampaign(ctx context.Context, workspaceID, projectID string, req domain.CreateCampaignRequest) (domain.CampaignSummary, error) {
	workspaceID = normalizeRequiredField(workspaceID)
	projectID = normalizeRequiredField(projectID)
	req.CampaignID = normalizeRequiredField(req.CampaignID)
	req.Name = normalizeRequiredField(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if workspaceID == "" {
		return domain.CampaignSummary{}, fmt.Errorf("workspace_id is required")
	}
	if projectID == "" {
		return domain.CampaignSummary{}, fmt.Errorf("project_id is required")
	}
	if req.CampaignID == "" {
		return domain.CampaignSummary{}, fmt.Errorf("campaign_id is required")
	}
	if req.Name == "" {
		return domain.CampaignSummary{}, fmt.Errorf("name is required")
	}
	return s.store.CreateCampaign(ctx, workspaceID, projectID, req)
}

func (s *Service) UpdateWorkspace(ctx context.Context, workspaceID string, req domain.UpdateWorkspaceRequest) (domain.WorkspaceSummary, error) {
	workspaceID = normalizeRequiredField(workspaceID)
	if workspaceID == "" {
		return domain.WorkspaceSummary{}, fmt.Errorf("workspace_id is required")
	}
	normalized, err := normalizeWorkspaceUpdateRequest(req)
	if err != nil {
		return domain.WorkspaceSummary{}, err
	}
	return s.store.UpdateWorkspace(ctx, workspaceID, normalized)
}

func (s *Service) UpdateProject(ctx context.Context, workspaceID, projectID string, req domain.UpdateProjectRequest) (domain.ProjectSummary, error) {
	workspaceID = normalizeRequiredField(workspaceID)
	projectID = normalizeRequiredField(projectID)
	if workspaceID == "" {
		return domain.ProjectSummary{}, fmt.Errorf("workspace_id is required")
	}
	if projectID == "" {
		return domain.ProjectSummary{}, fmt.Errorf("project_id is required")
	}
	normalized, err := normalizeProjectUpdateRequest(req)
	if err != nil {
		return domain.ProjectSummary{}, err
	}
	return s.store.UpdateProject(ctx, workspaceID, projectID, normalized)
}

func (s *Service) UpdateCampaign(ctx context.Context, workspaceID, projectID, campaignID string, req domain.UpdateCampaignRequest) (domain.CampaignSummary, error) {
	workspaceID = normalizeRequiredField(workspaceID)
	projectID = normalizeRequiredField(projectID)
	campaignID = normalizeRequiredField(campaignID)
	if workspaceID == "" {
		return domain.CampaignSummary{}, fmt.Errorf("workspace_id is required")
	}
	if projectID == "" {
		return domain.CampaignSummary{}, fmt.Errorf("project_id is required")
	}
	if campaignID == "" {
		return domain.CampaignSummary{}, fmt.Errorf("campaign_id is required")
	}
	normalized, err := normalizeCampaignUpdateRequest(req)
	if err != nil {
		return domain.CampaignSummary{}, err
	}
	return s.store.UpdateCampaign(ctx, workspaceID, projectID, campaignID, normalized)
}

func (s *Service) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	workspaceID = normalizeRequiredField(workspaceID)
	if workspaceID == "" {
		return fmt.Errorf("workspace_id is required")
	}
	if err := s.store.DeleteWorkspace(ctx, workspaceID); err != nil {
		return err
	}
	return s.storage.DeleteWorkspaceScopeData(workspaceID)
}

func (s *Service) DeleteProject(ctx context.Context, workspaceID, projectID string) error {
	workspaceID = normalizeRequiredField(workspaceID)
	projectID = normalizeRequiredField(projectID)
	if workspaceID == "" {
		return fmt.Errorf("workspace_id is required")
	}
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if err := s.store.DeleteProject(ctx, workspaceID, projectID); err != nil {
		return err
	}
	return s.storage.DeleteProjectScopeData(workspaceID, projectID)
}

func (s *Service) DeleteCampaign(ctx context.Context, workspaceID, projectID, campaignID string) error {
	workspaceID = normalizeRequiredField(workspaceID)
	projectID = normalizeRequiredField(projectID)
	campaignID = normalizeRequiredField(campaignID)
	if workspaceID == "" {
		return fmt.Errorf("workspace_id is required")
	}
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if campaignID == "" {
		return fmt.Errorf("campaign_id is required")
	}
	if err := s.store.DeleteCampaign(ctx, workspaceID, projectID, campaignID); err != nil {
		return err
	}
	return s.storage.DeleteCampaignScopeData(domain.Scope{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		CampaignID:  campaignID,
	})
}

func normalizeRequiredField(value string) string {
	return strings.TrimSpace(value)
}

func normalizeWorkspaceUpdateRequest(req domain.UpdateWorkspaceRequest) (domain.UpdateWorkspaceRequest, error) {
	normalizedName, hasName, err := normalizeScopeUpdateName(req.Name)
	if err != nil {
		return domain.UpdateWorkspaceRequest{}, err
	}
	if !hasName && req.Archived == nil {
		return domain.UpdateWorkspaceRequest{}, fmt.Errorf("at least one field must be provided")
	}
	if hasName {
		req.Name = &normalizedName
	} else {
		req.Name = nil
	}
	return req, nil
}

func normalizeProjectUpdateRequest(req domain.UpdateProjectRequest) (domain.UpdateProjectRequest, error) {
	normalizedName, hasName, err := normalizeScopeUpdateName(req.Name)
	if err != nil {
		return domain.UpdateProjectRequest{}, err
	}
	if !hasName && req.Archived == nil {
		return domain.UpdateProjectRequest{}, fmt.Errorf("at least one field must be provided")
	}
	if hasName {
		req.Name = &normalizedName
	} else {
		req.Name = nil
	}
	return req, nil
}

func normalizeCampaignUpdateRequest(req domain.UpdateCampaignRequest) (domain.UpdateCampaignRequest, error) {
	normalizedName, hasName, err := normalizeScopeUpdateName(req.Name)
	if err != nil {
		return domain.UpdateCampaignRequest{}, err
	}
	if !hasName && req.Archived == nil {
		return domain.UpdateCampaignRequest{}, fmt.Errorf("at least one field must be provided")
	}
	if hasName {
		req.Name = &normalizedName
	} else {
		req.Name = nil
	}
	return req, nil
}

func normalizeScopeUpdateName(value *string) (string, bool, error) {
	if value == nil {
		return "", false, nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return "", false, fmt.Errorf("name cannot be empty")
	}
	return normalized, true, nil
}
