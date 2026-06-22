package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/app"
	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/store"
)

type Options struct {
	BasicAuthUsername            string
	BasicAuthPassword            string
	AdminUsername                string
	AdminPassword                string
	AdminSessionSecret           string
	AdminSessionTTL              time.Duration
	AuditSink                    HTTPAuditSink
	RateLimiter                  RateLimiter
	RateLimitWindow              time.Duration
	RateLimitInstanceMaxRequests int
	RateLimitProjectMaxRequests  int
}

type Server struct {
	service *app.Service
	options Options
}

const maxTaskInputUploadBytes int64 = 50 << 20

type requestAuthScope struct {
	WorkspaceID    string
	ProjectID      string
	CampaignID     string
	TaskID         string
	AssetID        string
	InputFileID    string
	AllowBasicOnly bool
	AllowAdmin     bool
	RequireAdmin   bool
}

func New(service *app.Service, options Options) *Server {
	return &Server{service: service, options: options}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.setCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.URL.Path == "/healthz" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	parts := splitPath(r.URL.Path)
	var auditWriter *auditResponseWriter
	auditMeta := requestAuditMetadata{
		RequestID: domain.NewID("req"),
		Route:     "unknown",
		Action:    "unknown",
		Actor:     defaultRequestAuditActor(r),
	}
	if len(parts) > 0 && parts[0] == "api" && s.options.AuditSink != nil {
		auditWriter = newAuditResponseWriter(w)
		w = auditWriter
		auditMeta.Route, auditMeta.Action = inferAuditRoute(parts, r.Method)
		started := time.Now()
		defer func() {
			s.emitHTTPAuditEvent(r, parts, started, auditWriter, auditMeta)
		}()
	}
	if len(parts) == 0 || parts[0] != "api" {
		writeError(w, http.StatusNotFound, "not_found", "route not found")
		return
	}
	if s.handleAdminSessionRoute(w, r, parts) {
		return
	}
	authorized, authScope, auditActor, err := s.authorizeRequest(w, r, parts)
	auditMeta.Scope = authScope
	auditMeta.Actor = auditActor
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if !authorized {
		return
	}
	if !s.enforceRateLimits(w, r, authScope) {
		return
	}

	isRead := r.Method == http.MethodGet || r.Method == http.MethodHead
	switch {
	case isRead && match(parts, "api", "workspaces"):
		s.handleListWorkspaces(w, r)
	case r.Method == http.MethodPost && match(parts, "api", "workspaces"):
		s.handleCreateWorkspace(w, r)
	case r.Method == http.MethodPatch && match(parts, "api", "workspaces", "*"):
		s.handleUpdateWorkspace(w, r, parts[2])
	case r.Method == http.MethodDelete && match(parts, "api", "workspaces", "*"):
		s.handleDeleteWorkspace(w, r, parts[2])
	case isRead && match(parts, "api", "workspaces", "*", "projects"):
		s.handleListProjects(w, r, parts[2])
	case r.Method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects"):
		s.handleCreateProject(w, r, parts[2])
	case r.Method == http.MethodPatch && match(parts, "api", "workspaces", "*", "projects", "*"):
		s.handleUpdateProject(w, r, parts[2], parts[4])
	case r.Method == http.MethodDelete && match(parts, "api", "workspaces", "*", "projects", "*"):
		s.handleDeleteProject(w, r, parts[2], parts[4])
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns"):
		s.handleListCampaigns(w, r, parts[2], parts[4])
	case r.Method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns"):
		s.handleCreateCampaign(w, r, parts[2], parts[4])
	case r.Method == http.MethodPatch && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*"):
		s.handleUpdateCampaign(w, r, parts[2], parts[4], parts[6])
	case r.Method == http.MethodDelete && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*"):
		s.handleDeleteCampaign(w, r, parts[2], parts[4], parts[6])
	case r.Method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "input-files"):
		s.handleUploadTaskInputFile(w, r, parts[2], parts[4], parts[6])
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "input-files", "*"):
		s.handleGetTaskInputFile(w, r, parts[2], parts[4], parts[6], parts[8])
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "input-files", "*", "content"):
		s.handleTaskInputFileContent(w, r, parts[2], parts[4], parts[6], parts[8])
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "storage-governance"):
		s.handleGetStorageGovernance(w, r, parts[2], parts[4], parts[6])
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "storage-integrity"):
		s.handleGetStorageIntegrity(w, r, parts[2], parts[4], parts[6])
	case r.Method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "tasks"):
		s.handleCreateTask(w, r, parts[2], parts[4], parts[6])
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "quality-profile"):
		s.handleGetProjectQualityProfile(w, r, parts[2], parts[4])
	case r.Method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "quality-profile"):
		s.handleUpdateProjectQualityProfile(w, r, parts[2], parts[4])
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "visual-context"):
		s.handleGetProjectVisualContext(w, r, parts[2], parts[4])
	case r.Method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "visual-context"):
		s.handleUpdateProjectVisualContext(w, r, parts[2], parts[4])
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "provider-profile"):
		s.handleGetProjectProviderProfile(w, r, parts[2], parts[4])
	case r.Method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "provider-profile"):
		s.handleUpdateProjectProviderProfile(w, r, parts[2], parts[4])
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "access-config"):
		s.handleGetProjectAccessConfig(w, r, parts[2], parts[4])
	case r.Method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "access-config"):
		s.handleUpdateProjectAccessConfig(w, r, parts[2], parts[4])
	case isRead && match(parts, "api", "tasks", "*", "attempts"):
		s.handleGetTaskAttempts(w, r, parts[2])
	case isRead && match(parts, "api", "tasks", "*"):
		s.handleGetTask(w, r, parts[2])
	case isRead && match(parts, "api", "projects", "*", "campaigns", "*", "assets"):
		s.handleListAssets(w, r, parts[2], parts[4])
	case isRead && match(parts, "api", "admin", "assets", "recent"):
		s.handleListRecentAssets(w, r)
	case isRead && match(parts, "api", "projects", "*", "campaigns", "*", "batch-progress"):
		s.handleGetBatchProgress(w, r, parts[2], parts[4])
	case isRead && match(parts, "api", "projects", "*", "campaigns", "*", "batch-summary"):
		s.handleGetBatchStorySummary(w, r, parts[2], parts[4])
	case isRead && match(parts, "api", "projects", "*", "campaigns", "*", "batch-manifest"):
		s.handleGetBatchManifest(w, r, parts[2], parts[4])
	case r.Method == http.MethodPost && match(parts, "api", "projects", "*", "campaigns", "*", "scene-regenerations"):
		s.handleRegenerateSceneTask(w, r, parts[2], parts[4])
	case isRead && match(parts, "api", "assets", "*"):
		s.handleGetAsset(w, r, parts[2])
	case isRead && match(parts, "api", "assets", "*", "metadata"):
		s.handleGetAssetMetadata(w, r, parts[2])
	case r.Method == http.MethodPost && match(parts, "api", "assets", "*", "approve"):
		s.handleReviewAsset(w, r, parts[2], "approve")
	case r.Method == http.MethodPost && match(parts, "api", "assets", "*", "reject"):
		s.handleReviewAsset(w, r, parts[2], "reject")
	case isRead && match(parts, "api", "assets", "*", "original"):
		s.handleAssetFile(w, r, parts[2], "original")
	case isRead && match(parts, "api", "assets", "*", "thumbnail"):
		s.handleAssetFile(w, r, parts[2], "thumbnail")
	default:
		writeError(w, http.StatusNotFound, "not_found", "route not found")
	}
}

func (s *Server) authorizeRequest(w http.ResponseWriter, r *http.Request, parts []string) (bool, requestAuthScope, requestAuditActor, error) {
	actor := defaultRequestAuditActor(r)
	if routeAllowsAdminSession(parts, r.Method) {
		if username, ok := s.adminSessionUsername(r); ok {
			scope, ok, err := s.resolveRequestAuthScope(r, parts)
			if err != nil {
				return false, requestAuthScope{}, actor, err
			}
			if ok {
				actor.AuthMode = "admin_session"
				actor.Actor = username
				return true, scope, actor, nil
			}
		}
	}
	if !s.authorizeBasicAuth(w, r) {
		return false, requestAuthScope{}, actor, nil
	}
	scope, ok, err := s.resolveRequestAuthScope(r, parts)
	if err != nil {
		return false, requestAuthScope{}, actor, err
	}
	if !ok || scope.ProjectID == "" {
		if scope.RequireAdmin && actor.BasicAuthUser == "" {
			writeUnauthorized(w, "admin_session_required", "admin session is required", false)
			return false, scope, actor, nil
		}
		if actor.BasicAuthUser != "" {
			actor.AuthMode = "basic_auth"
			actor.Actor = actor.BasicAuthUser
		}
		return true, scope, actor, nil
	}
	if scope.AllowBasicOnly {
		if actor.BasicAuthUser != "" {
			actor.AuthMode = "basic_auth"
			actor.Actor = actor.BasicAuthUser
		}
		return true, scope, actor, nil
	}
	if scope.RequireAdmin && actor.BasicAuthUser == "" {
		writeUnauthorized(w, "admin_session_required", "admin session is required", false)
		return false, scope, actor, nil
	}
	apiKey := readProjectAPIKey(r)
	var required bool
	var valid bool
	var matchedKey domain.ProjectAPIKeyView
	if scope.WorkspaceID != "" {
		required, valid, matchedKey, err = s.service.ValidateProjectAPIKey(r.Context(), scope.WorkspaceID, scope.ProjectID, apiKey)
	} else {
		required, valid, matchedKey, err = s.service.ValidateProjectAPIKeyByProjectID(r.Context(), scope.ProjectID, apiKey)
	}
	if err != nil {
		return false, requestAuthScope{}, actor, err
	}
	if required && !valid {
		writeUnauthorized(w, "project_api_key_invalid", "project api key is required or invalid", false)
		if strings.TrimSpace(apiKey) != "" {
			actor.AuthMode = "project_api_key"
			actor.Actor = "project_api_key"
			if actor.BasicAuthUser != "" {
				actor.AuthMode = "basic_auth+project_api_key"
				actor.Actor = actor.BasicAuthUser
			}
		}
		return false, scope, actor, nil
	}
	if required && strings.TrimSpace(apiKey) != "" {
		actor.AuthMode = "project_api_key"
		actor.Actor = "project_api_key"
		if actor.BasicAuthUser != "" {
			actor.AuthMode = "basic_auth+project_api_key"
			actor.Actor = actor.BasicAuthUser
		}
		if keyName := strings.TrimSpace(matchedKey.Name); keyName != "" {
			actor.ProjectAPIKeyName = keyName
			if actor.Actor == "project_api_key" {
				actor.Actor = keyName
			}
		} else if preview := strings.TrimSpace(matchedKey.Preview); preview != "" {
			actor.ProjectAPIKeyName = preview
			if actor.Actor == "project_api_key" {
				actor.Actor = preview
			}
		}
	} else if actor.BasicAuthUser != "" {
		actor.AuthMode = "basic_auth"
		actor.Actor = actor.BasicAuthUser
	}
	return true, scope, actor, nil
}

func (s *Server) authorizeBasicAuth(w http.ResponseWriter, r *http.Request) bool {
	if !s.basicAuthConfigured() {
		return true
	}
	username, password, ok := r.BasicAuth()
	if !ok {
		writeUnauthorized(w, "basic_auth_required", "basic auth is required", true)
		return false
	}
	if subtle.ConstantTimeCompare([]byte(username), []byte(s.options.BasicAuthUsername)) != 1 ||
		subtle.ConstantTimeCompare([]byte(password), []byte(s.options.BasicAuthPassword)) != 1 {
		writeUnauthorized(w, "basic_auth_invalid", "basic auth is invalid", true)
		return false
	}
	return true
}

func (s *Server) basicAuthConfigured() bool {
	return strings.TrimSpace(s.options.BasicAuthUsername) != "" || strings.TrimSpace(s.options.BasicAuthPassword) != ""
}

func (s *Server) rateLimitingEnabled() bool {
	return s.options.RateLimiter != nil &&
		s.options.RateLimitWindow > 0 &&
		(s.options.RateLimitInstanceMaxRequests > 0 || s.options.RateLimitProjectMaxRequests > 0)
}

func (s *Server) enforceRateLimits(w http.ResponseWriter, r *http.Request, scope requestAuthScope) bool {
	if !s.rateLimitingEnabled() {
		return true
	}

	if s.options.RateLimitInstanceMaxRequests > 0 {
		decision, err := s.options.RateLimiter.Allow(r.Context(), rateLimitInstanceKey(), s.options.RateLimitInstanceMaxRequests, s.options.RateLimitWindow)
		if err != nil {
			log.Printf("rate limit backend error (scope=instance): %v", err)
		} else if !decision.Allowed {
			writeRateLimited(w, "instance", decision)
			return false
		}
	}

	if s.options.RateLimitProjectMaxRequests > 0 && strings.TrimSpace(scope.ProjectID) != "" {
		decision, err := s.options.RateLimiter.Allow(r.Context(), rateLimitProjectKey(scope), s.options.RateLimitProjectMaxRequests, s.options.RateLimitWindow)
		if err != nil {
			log.Printf("rate limit backend error (scope=project project_id=%s workspace_id=%s): %v", scope.ProjectID, scope.WorkspaceID, err)
		} else if !decision.Allowed {
			writeRateLimited(w, "project", decision)
			return false
		}
	}

	return true
}

func (s *Server) resolveRequestAuthScope(r *http.Request, parts []string) (requestAuthScope, bool, error) {
	switch {
	case match(parts, "api", "workspaces"),
		match(parts, "api", "workspaces", "*"),
		match(parts, "api", "workspaces", "*", "projects"),
		match(parts, "api", "workspaces", "*", "projects", "*"),
		match(parts, "api", "workspaces", "*", "projects", "*", "campaigns"):
		return requestAuthScope{WorkspaceID: valueAt(parts, 2), ProjectID: valueAt(parts, 4), AllowBasicOnly: true, AllowAdmin: true, RequireAdmin: s.adminConfigured()}, true, nil
	case match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*"):
		return requestAuthScope{
			WorkspaceID:    valueAt(parts, 2),
			ProjectID:      valueAt(parts, 4),
			CampaignID:     valueAt(parts, 6),
			AllowBasicOnly: true,
			AllowAdmin:     true,
			RequireAdmin:   s.adminConfigured(),
		}, true, nil
	case match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "tasks"):
		return requestAuthScope{WorkspaceID: parts[2], ProjectID: parts[4], CampaignID: parts[6]}, true, nil
	case match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "input-files"),
		match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "input-files", "*"),
		match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "input-files", "*", "content"):
		return requestAuthScope{
			WorkspaceID: parts[2],
			ProjectID:   parts[4],
			CampaignID:  parts[6],
			InputFileID: valueAt(parts, 8),
		}, true, nil
	case match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "storage-governance"):
		return requestAuthScope{
			WorkspaceID: parts[2],
			ProjectID:   parts[4],
			CampaignID:  parts[6],
		}, true, nil
	case match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "storage-integrity"):
		return requestAuthScope{
			WorkspaceID: parts[2],
			ProjectID:   parts[4],
			CampaignID:  parts[6],
		}, true, nil
	case match(parts, "api", "workspaces", "*", "projects", "*", "quality-profile"):
		return requestAuthScope{WorkspaceID: parts[2], ProjectID: parts[4]}, true, nil
	case match(parts, "api", "workspaces", "*", "projects", "*", "visual-context"):
		return requestAuthScope{WorkspaceID: parts[2], ProjectID: parts[4]}, true, nil
	case match(parts, "api", "workspaces", "*", "projects", "*", "provider-profile"):
		return requestAuthScope{WorkspaceID: parts[2], ProjectID: parts[4]}, true, nil
	case match(parts, "api", "workspaces", "*", "projects", "*", "access-config"):
		return requestAuthScope{WorkspaceID: parts[2], ProjectID: parts[4], AllowBasicOnly: true, AllowAdmin: true, RequireAdmin: s.adminConfigured()}, true, nil
	case match(parts, "api", "tasks", "*"),
		match(parts, "api", "tasks", "*", "attempts"):
		scope, err := s.service.GetTaskScope(r.Context(), parts[2])
		if err != nil {
			return requestAuthScope{}, false, err
		}
		return requestAuthScope{
			WorkspaceID: scope.WorkspaceID,
			ProjectID:   scope.ProjectID,
			CampaignID:  scope.CampaignID,
			TaskID:      parts[2],
			AllowAdmin:  true,
		}, true, nil
	case match(parts, "api", "projects", "*", "campaigns", "*", "assets"),
		match(parts, "api", "projects", "*", "campaigns", "*", "batch-progress"),
		match(parts, "api", "projects", "*", "campaigns", "*", "batch-summary"),
		match(parts, "api", "projects", "*", "campaigns", "*", "batch-manifest"),
		match(parts, "api", "projects", "*", "campaigns", "*", "scene-regenerations"):
		return requestAuthScope{ProjectID: parts[2], CampaignID: parts[4], AllowAdmin: true}, true, nil
	case match(parts, "api", "admin", "assets", "recent"):
		return requestAuthScope{AllowAdmin: true, RequireAdmin: true}, true, nil
	case match(parts, "api", "assets", "*"),
		match(parts, "api", "assets", "*", "metadata"),
		match(parts, "api", "assets", "*", "approve"),
		match(parts, "api", "assets", "*", "reject"),
		match(parts, "api", "assets", "*", "original"),
		match(parts, "api", "assets", "*", "thumbnail"):
		scope, err := s.service.GetAssetScope(r.Context(), parts[2])
		if err != nil {
			return requestAuthScope{}, false, err
		}
		return requestAuthScope{
			WorkspaceID: scope.WorkspaceID,
			ProjectID:   scope.ProjectID,
			CampaignID:  scope.CampaignID,
			AssetID:     parts[2],
			AllowAdmin:  true,
		}, true, nil
	default:
		return requestAuthScope{}, false, nil
	}
}

func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	response, err := s.service.ListWorkspaces(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req domain.CreateWorkspaceRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	response, err := s.service.CreateWorkspace(r.Context(), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) handleUpdateWorkspace(w http.ResponseWriter, r *http.Request, workspaceID string) {
	defer r.Body.Close()
	var req domain.UpdateWorkspaceRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	response, err := s.service.UpdateWorkspace(r.Context(), workspaceID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request, workspaceID string) {
	if err := s.service.DeleteWorkspace(r.Context(), workspaceID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request, workspaceID string) {
	response, err := s.service.ListProjects(r.Context(), workspaceID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request, workspaceID string) {
	defer r.Body.Close()
	var req domain.CreateProjectRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	response, err := s.service.CreateProject(r.Context(), workspaceID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) {
	defer r.Body.Close()
	var req domain.UpdateProjectRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	response, err := s.service.UpdateProject(r.Context(), workspaceID, projectID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) {
	if err := s.service.DeleteProject(r.Context(), workspaceID, projectID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) {
	response, err := s.service.ListCampaigns(r.Context(), workspaceID, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) {
	defer r.Body.Close()
	var req domain.CreateCampaignRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	response, err := s.service.CreateCampaign(r.Context(), workspaceID, projectID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) handleUpdateCampaign(w http.ResponseWriter, r *http.Request, workspaceID, projectID, campaignID string) {
	defer r.Body.Close()
	var req domain.UpdateCampaignRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	response, err := s.service.UpdateCampaign(r.Context(), workspaceID, projectID, campaignID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleDeleteCampaign(w http.ResponseWriter, r *http.Request, workspaceID, projectID, campaignID string) {
	if err := s.service.DeleteCampaign(r.Context(), workspaceID, projectID, campaignID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUploadTaskInputFile(w http.ResponseWriter, r *http.Request, workspaceID, projectID, campaignID string) {
	r.Body = http.MaxBytesReader(w, r.Body, maxTaskInputUploadBytes+(1<<20))
	if err := r.ParseMultipartForm(maxTaskInputUploadBytes + (1 << 20)); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_multipart", err.Error())
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file_required", "multipart field \"file\" is required")
		return
	}
	defer file.Close()

	raw, err := io.ReadAll(io.LimitReader(file, maxTaskInputUploadBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "read_file_failed", err.Error())
		return
	}
	if int64(len(raw)) > maxTaskInputUploadBytes {
		writeError(w, http.StatusBadRequest, "file_too_large", "input file exceeds upload limit")
		return
	}

	kind := strings.TrimSpace(r.FormValue("kind"))
	mimeType := strings.TrimSpace(r.FormValue("mime_type"))
	if mimeType == "" {
		mimeType = strings.TrimSpace(header.Header.Get("Content-Type"))
	}
	response, err := s.service.UploadTaskInputFile(r.Context(), domain.Scope{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		CampaignID:  campaignID,
	}, kind, header.Filename, mimeType, raw)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) handleGetTaskInputFile(w http.ResponseWriter, r *http.Request, workspaceID, projectID, campaignID, inputFileID string) {
	response, err := s.service.GetTaskInputFile(r.Context(), domain.Scope{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		CampaignID:  campaignID,
	}, inputFileID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleTaskInputFileContent(w http.ResponseWriter, r *http.Request, workspaceID, projectID, campaignID, inputFileID string) {
	path, mimeType, err := s.service.GetTaskInputFileContent(r.Context(), domain.Scope{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		CampaignID:  campaignID,
	}, inputFileID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", mimeType)
	http.ServeFile(w, r, path)
}

func (s *Server) handleGetStorageGovernance(w http.ResponseWriter, r *http.Request, workspaceID, projectID, campaignID string) {
	response, err := s.service.GetStorageGovernance(r.Context(), domain.Scope{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		CampaignID:  campaignID,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetStorageIntegrity(w http.ResponseWriter, r *http.Request, workspaceID, projectID, campaignID string) {
	response, err := s.service.GetStorageIntegrity(r.Context(), domain.Scope{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		CampaignID:  campaignID,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request, workspaceID, projectID, campaignID string) {
	defer r.Body.Close()
	var req domain.CreateTaskRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	response, err := s.service.CreateTask(r.Context(), domain.Scope{
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		CampaignID:  campaignID,
	}, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create_task_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request, taskID string) {
	response, err := s.service.GetTask(r.Context(), taskID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetTaskAttempts(w http.ResponseWriter, r *http.Request, taskID string) {
	response, err := s.service.ListTaskAttempts(r.Context(), taskID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetProjectQualityProfile(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) {
	response, err := s.service.GetProjectQualityProfile(r.Context(), workspaceID, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleUpdateProjectQualityProfile(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) {
	defer r.Body.Close()
	var profile domain.QualityProfile
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&profile); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	response, err := s.service.UpdateProjectQualityProfile(r.Context(), workspaceID, projectID, profile)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetProjectVisualContext(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) {
	response, err := s.service.GetProjectVisualContext(r.Context(), workspaceID, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleUpdateProjectVisualContext(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) {
	defer r.Body.Close()
	var payload struct {
		VisualContext domain.ProjectVisualContext `json:"visual_context"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	response, err := s.service.UpdateProjectVisualContext(r.Context(), workspaceID, projectID, payload.VisualContext)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetProjectProviderProfile(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) {
	response, err := s.service.GetProjectProviderProfile(r.Context(), workspaceID, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleUpdateProjectProviderProfile(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) {
	defer r.Body.Close()
	var profile domain.ProjectProviderProfile
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&profile); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	response, err := s.service.UpdateProjectProviderProfile(r.Context(), workspaceID, projectID, profile)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetProjectAccessConfig(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) {
	response, err := s.service.GetProjectAccessConfig(r.Context(), workspaceID, projectID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleUpdateProjectAccessConfig(w http.ResponseWriter, r *http.Request, workspaceID, projectID string) {
	defer r.Body.Close()
	var req domain.ProjectAccessConfigUpdateRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	response, err := s.service.UpdateProjectAccessConfig(r.Context(), workspaceID, projectID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleListAssets(w http.ResponseWriter, r *http.Request, projectID, campaignID string) {
	query, err := parseAssetListQuery(r, projectID, campaignID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}
	response, err := s.service.ListAssets(r.Context(), query)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleListRecentAssets(w http.ResponseWriter, r *http.Request) {
	query, err := parseAssetListQuery(r, "", "")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}
	response, err := s.service.ListRecentAssets(r.Context(), query)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetBatchProgress(w http.ResponseWriter, r *http.Request, projectID, campaignID string) {
	query, err := parseBatchProgressQuery(r, projectID, campaignID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}
	response, err := s.service.GetBatchProgress(r.Context(), query)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetBatchStorySummary(w http.ResponseWriter, r *http.Request, projectID, campaignID string) {
	query, err := parseBatchStorySummaryQuery(r, projectID, campaignID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}
	response, err := s.service.GetBatchStorySummary(r.Context(), query)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetBatchManifest(w http.ResponseWriter, r *http.Request, projectID, campaignID string) {
	query, err := parseBatchManifestQuery(r, projectID, campaignID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}
	response, err := s.service.GetBatchManifest(r.Context(), query)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleRegenerateSceneTask(w http.ResponseWriter, r *http.Request, projectID, campaignID string) {
	defer r.Body.Close()
	var req domain.SceneRegenerateRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	response, err := s.service.RegenerateSceneTask(r.Context(), domain.Scope{
		ProjectID:  projectID,
		CampaignID: campaignID,
	}, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "scene_regeneration_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) handleGetAsset(w http.ResponseWriter, r *http.Request, assetID string) {
	response, err := s.service.GetAsset(r.Context(), assetID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetAssetMetadata(w http.ResponseWriter, r *http.Request, assetID string) {
	response, err := s.service.GetAssetMetadata(r.Context(), assetID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleReviewAsset(w http.ResponseWriter, r *http.Request, assetID, action string) {
	response, err := s.service.ReviewAsset(r.Context(), assetID, action)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleAssetFile(w http.ResponseWriter, r *http.Request, assetID, kind string) {
	path, mimeType, err := s.service.GetAssetFile(r.Context(), assetID, kind)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", mimeType)
	http.ServeFile(w, r, path)
}

func parseBatchProgressQuery(r *http.Request, projectID, campaignID string) (domain.BatchProgressQuery, error) {
	values := r.URL.Query()
	limit := domain.DefaultBatchProgressLimit
	if rawLimit := strings.TrimSpace(values.Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			return domain.BatchProgressQuery{}, err
		}
		limit = parsed
	}
	if limit <= 0 {
		limit = domain.DefaultBatchProgressLimit
	}
	if limit > domain.MaxBatchProgressLimit {
		limit = domain.MaxBatchProgressLimit
	}
	return domain.BatchProgressQuery{
		ProjectID:  projectID,
		CampaignID: campaignID,
		SessionID:  strings.TrimSpace(values.Get("session_id")),
		BatchID:    strings.TrimSpace(values.Get("batch_id")),
		Limit:      limit,
	}, nil
}

func parseBatchStorySummaryQuery(r *http.Request, projectID, campaignID string) (domain.BatchStorySummaryQuery, error) {
	values := r.URL.Query()
	limit := domain.DefaultBatchProgressLimit
	if rawLimit := strings.TrimSpace(values.Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			return domain.BatchStorySummaryQuery{}, err
		}
		limit = parsed
	}
	if limit <= 0 {
		limit = domain.DefaultBatchProgressLimit
	}
	if limit > domain.MaxBatchProgressLimit {
		limit = domain.MaxBatchProgressLimit
	}
	includeSetup := false
	if rawIncludeSetup := strings.TrimSpace(values.Get("include_setup")); rawIncludeSetup != "" {
		parsed, err := strconv.ParseBool(rawIncludeSetup)
		if err != nil {
			return domain.BatchStorySummaryQuery{}, err
		}
		includeSetup = parsed
	}
	return domain.BatchStorySummaryQuery{
		ProjectID:    projectID,
		CampaignID:   campaignID,
		SessionID:    strings.TrimSpace(values.Get("session_id")),
		BatchID:      strings.TrimSpace(values.Get("batch_id")),
		StoryID:      strings.TrimSpace(values.Get("story_id")),
		Source:       strings.TrimSpace(values.Get("source")),
		Status:       strings.TrimSpace(values.Get("status")),
		IncludeSetup: includeSetup,
		Limit:        limit,
	}, nil
}

func parseBatchManifestQuery(r *http.Request, projectID, campaignID string) (domain.BatchManifestQuery, error) {
	summaryQuery, err := parseBatchStorySummaryQuery(r, projectID, campaignID)
	if err != nil {
		return domain.BatchManifestQuery{}, err
	}
	if summaryQuery.SessionID == "" && summaryQuery.BatchID == "" {
		return domain.BatchManifestQuery{}, errors.New("session_id or batch_id is required")
	}
	values := r.URL.Query()
	selectedOnly := true
	if rawSelectedOnly := strings.TrimSpace(values.Get("selected_only")); rawSelectedOnly != "" {
		parsed, err := strconv.ParseBool(rawSelectedOnly)
		if err != nil {
			return domain.BatchManifestQuery{}, err
		}
		selectedOnly = parsed
	}
	includeRejected := false
	if rawIncludeRejected := strings.TrimSpace(values.Get("include_rejected")); rawIncludeRejected != "" {
		parsed, err := strconv.ParseBool(rawIncludeRejected)
		if err != nil {
			return domain.BatchManifestQuery{}, err
		}
		includeRejected = parsed
	}
	return domain.BatchManifestQuery{
		BatchStorySummaryQuery: summaryQuery,
		SelectedOnly:           selectedOnly,
		IncludeRejected:        includeRejected,
	}, nil
}

func parseAssetListQuery(r *http.Request, projectID, campaignID string) (domain.AssetListQuery, error) {
	values := r.URL.Query()
	query := domain.AssetListQuery{
		ProjectID:  projectID,
		CampaignID: campaignID,
		Status:     values.Get("status"),
		Provider:   values.Get("provider"),
		Model:      values.Get("model"),
		Source:     values.Get("source"),
		SessionID:  values.Get("session_id"),
		BatchID:    values.Get("batch_id"),
		Keyword:    values.Get("keyword"),
	}
	if limitRaw := strings.TrimSpace(values.Get("limit")); limitRaw != "" {
		limit, err := strconv.Atoi(limitRaw)
		if err != nil {
			return domain.AssetListQuery{}, err
		}
		query.Limit = limit
	}
	if offsetRaw := strings.TrimSpace(values.Get("offset")); offsetRaw != "" {
		offset, err := strconv.Atoi(offsetRaw)
		if err != nil {
			return domain.AssetListQuery{}, err
		}
		query.Offset = offset
	}
	if createdFromRaw := strings.TrimSpace(values.Get("created_from")); createdFromRaw != "" {
		createdFrom, err := time.Parse(time.RFC3339, createdFromRaw)
		if err != nil {
			return domain.AssetListQuery{}, err
		}
		query.CreatedFrom = &createdFrom
	}
	if createdToRaw := strings.TrimSpace(values.Get("created_to")); createdToRaw != "" {
		createdTo, err := time.Parse(time.RFC3339, createdToRaw)
		if err != nil {
			return domain.AssetListQuery{}, err
		}
		query.CreatedTo = &createdTo
	}
	return query, nil
}

func (s *Server) setCORS(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-API-Key")
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func valueAt(parts []string, index int) string {
	if index < 0 || index >= len(parts) {
		return ""
	}
	return parts[index]
}

func match(parts []string, pattern ...string) bool {
	if len(parts) != len(pattern) {
		return false
	}
	for i := range pattern {
		if pattern[i] == "*" {
			continue
		}
		if parts[i] != pattern[i] {
			return false
		}
	}
	return true
}

func readProjectAPIKey(r *http.Request) string {
	if apiKey := strings.TrimSpace(r.Header.Get("X-API-Key")); apiKey != "" {
		return apiKey
	}
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		return strings.TrimSpace(authorization[len("Bearer "):])
	}
	return ""
}

func rateLimitInstanceKey() string {
	return "rate_limit:instance:http_api"
}

func rateLimitProjectKey(scope requestAuthScope) string {
	workspaceID := strings.TrimSpace(scope.WorkspaceID)
	if workspaceID == "" {
		workspaceID = "_"
	}
	return "rate_limit:project:" + workspaceID + ":" + strings.TrimSpace(scope.ProjectID)
}

func retryAfterSeconds(delay time.Duration) int {
	if delay <= 0 {
		return 1
	}
	seconds := int((delay + time.Second - 1) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	log.Printf("request failed: %v", err)
	writeError(w, http.StatusBadRequest, "request_failed", err.Error())
}

func writeUnauthorized(w http.ResponseWriter, code, message string, basicChallenge bool) {
	if basicChallenge {
		w.Header().Set("WWW-Authenticate", `Basic realm="Agent ImageFlow"`)
	}
	writeError(w, http.StatusUnauthorized, code, message)
}

func writeRateLimited(w http.ResponseWriter, scope string, decision RateLimitDecision) {
	retryAfter := retryAfterSeconds(decision.RetryAfter)
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	writeJSON(w, http.StatusTooManyRequests, map[string]any{
		"error_code":          "rate_limited",
		"error_message":       scope + " rate limit exceeded; retry in " + strconv.Itoa(retryAfter) + " second(s)",
		"rate_limit_scope":    scope,
		"retry_after_seconds": retryAfter,
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error_code":    code,
		"error_message": message,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
