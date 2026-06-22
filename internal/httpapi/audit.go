package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

const maxAuditBodyCaptureBytes = 8 << 10

type HTTPAuditSink interface {
	AppendHTTPAuditEvent(ctx context.Context, event domain.HTTPAuditEvent) error
}

type requestAuditActor struct {
	AuthMode          string
	Actor             string
	BasicAuthUser     string
	ProjectAPIKeyName string
}

type requestAuditMetadata struct {
	RequestID string
	Route     string
	Action    string
	Scope     requestAuthScope
	Actor     requestAuditActor
}

type auditResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
	body        bytes.Buffer
}

func newAuditResponseWriter(w http.ResponseWriter) *auditResponseWriter {
	return &auditResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (w *auditResponseWriter) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		w.statusCode = statusCode
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *auditResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.body.Len() < maxAuditBodyCaptureBytes {
		remaining := maxAuditBodyCaptureBytes - w.body.Len()
		if remaining > len(p) {
			remaining = len(p)
		}
		_, _ = w.body.Write(p[:remaining])
	}
	return w.ResponseWriter.Write(p)
}

func defaultRequestAuditActor(r *http.Request) requestAuditActor {
	actor := requestAuditActor{
		AuthMode: "anonymous",
		Actor:    "anonymous",
	}
	if username, _, ok := r.BasicAuth(); ok {
		trimmed := strings.TrimSpace(username)
		if trimmed != "" {
			actor.AuthMode = "basic_auth"
			actor.Actor = trimmed
			actor.BasicAuthUser = trimmed
		}
	} else if strings.TrimSpace(readProjectAPIKey(r)) != "" {
		actor.AuthMode = "project_api_key"
		actor.Actor = "project_api_key"
	}
	return actor
}

func (s *Server) emitHTTPAuditEvent(r *http.Request, parts []string, started time.Time, writer *auditResponseWriter, metadata requestAuditMetadata) {
	if writer == nil || s.options.AuditSink == nil {
		return
	}
	event := domain.HTTPAuditEvent{
		EventID:           domain.NewID("audit"),
		Timestamp:         time.Now().UTC(),
		Source:            domain.HTTPAuditSourceAPI,
		RequestID:         metadata.RequestID,
		Method:            r.Method,
		Path:              r.URL.Path,
		Route:             metadata.Route,
		Action:            metadata.Action,
		StatusCode:        writer.statusCode,
		DurationMs:        time.Since(started).Milliseconds(),
		Success:           writer.statusCode >= 200 && writer.statusCode < 400,
		AuthMode:          metadata.Actor.AuthMode,
		Actor:             metadata.Actor.Actor,
		BasicAuthUser:     metadata.Actor.BasicAuthUser,
		ProjectAPIKeyName: metadata.Actor.ProjectAPIKeyName,
		WorkspaceID:       metadata.Scope.WorkspaceID,
		ProjectID:         metadata.Scope.ProjectID,
		CampaignID:        metadata.Scope.CampaignID,
		TaskID:            metadata.Scope.TaskID,
		AssetID:           metadata.Scope.AssetID,
		InputFileID:       metadata.Scope.InputFileID,
		RemoteAddr:        strings.TrimSpace(r.RemoteAddr),
		UserAgent:         strings.TrimSpace(r.UserAgent()),
	}
	enrichHTTPAuditEvent(&event, parts, writer.body.Bytes())
	if event.Actor == "" {
		event.Actor = "anonymous"
	}
	if event.AuthMode == "" {
		event.AuthMode = "anonymous"
	}
	if err := s.options.AuditSink.AppendHTTPAuditEvent(context.WithoutCancel(r.Context()), event); err != nil {
		log.Printf("write http audit event failed: %v", err)
	}
}

func enrichHTTPAuditEvent(event *domain.HTTPAuditEvent, parts []string, responseBody []byte) {
	switch {
	case match(parts, "api", "tasks", "*", "attempts"):
		if event.TaskID == "" {
			event.TaskID = parts[2]
		}
	case match(parts, "api", "tasks", "*"):
		if event.TaskID == "" {
			event.TaskID = parts[2]
		}
	case match(parts, "api", "assets", "*"),
		match(parts, "api", "assets", "*", "approve"),
		match(parts, "api", "assets", "*", "reject"),
		match(parts, "api", "assets", "*", "original"),
		match(parts, "api", "assets", "*", "thumbnail"):
		if event.AssetID == "" {
			event.AssetID = parts[2]
		}
	case match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "input-files", "*"),
		match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "input-files", "*", "content"):
		if event.InputFileID == "" {
			event.InputFileID = parts[8]
		}
	}

	var payload map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(responseBody), &payload); err != nil {
		return
	}
	assignAuditStringIfEmpty(&event.WorkspaceID, payload, "workspace_id")
	assignAuditStringIfEmpty(&event.ProjectID, payload, "project_id")
	assignAuditStringIfEmpty(&event.CampaignID, payload, "campaign_id")
	assignAuditStringIfEmpty(&event.TaskID, payload, "task_id")
	assignAuditStringIfEmpty(&event.AssetID, payload, "asset_id")
	assignAuditStringIfEmpty(&event.InputFileID, payload, "input_file_id")
	assignAuditStringIfEmpty(&event.ErrorCode, payload, "error_code")
	assignAuditStringIfEmpty(&event.ErrorMessage, payload, "error_message")
}

func assignAuditStringIfEmpty(target *string, payload map[string]any, key string) {
	if target == nil || *target != "" {
		return
	}
	value, ok := payload[key]
	if !ok {
		return
	}
	text, ok := value.(string)
	if !ok {
		return
	}
	*target = strings.TrimSpace(text)
}

func inferAuditRoute(parts []string, method string) (string, string) {
	isRead := method == http.MethodGet || method == http.MethodHead
	switch {
	case method == http.MethodPost && match(parts, "api", "admin", "login"):
		return "/api/admin/login", "admin_login"
	case isRead && match(parts, "api", "admin", "me"):
		return "/api/admin/me", "admin_me"
	case method == http.MethodPost && match(parts, "api", "admin", "logout"):
		return "/api/admin/logout", "admin_logout"
	case isRead && match(parts, "api", "admin", "assets", "recent"):
		return "/api/admin/assets/recent", "list_recent_assets"
	case isRead && match(parts, "api", "workspaces"):
		return "/api/workspaces", "list_workspaces"
	case method == http.MethodPost && match(parts, "api", "workspaces"):
		return "/api/workspaces", "create_workspace"
	case method == http.MethodPatch && match(parts, "api", "workspaces", "*"):
		return "/api/workspaces/{workspace_id}", "update_workspace"
	case method == http.MethodDelete && match(parts, "api", "workspaces", "*"):
		return "/api/workspaces/{workspace_id}", "delete_workspace"
	case isRead && match(parts, "api", "workspaces", "*", "projects"):
		return "/api/workspaces/{workspace_id}/projects", "list_projects"
	case method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects"):
		return "/api/workspaces/{workspace_id}/projects", "create_project"
	case method == http.MethodPatch && match(parts, "api", "workspaces", "*", "projects", "*"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}", "update_project"
	case method == http.MethodDelete && match(parts, "api", "workspaces", "*", "projects", "*"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}", "delete_project"
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/campaigns", "list_campaigns"
	case method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/campaigns", "create_campaign"
	case method == http.MethodPatch && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}", "update_campaign"
	case method == http.MethodDelete && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}", "delete_campaign"
	case method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "input-files"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/input-files", "upload_input_file"
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "input-files", "*"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/input-files/{input_file_id}", "get_input_file"
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "input-files", "*", "content"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/input-files/{input_file_id}/content", "get_input_file_content"
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "storage-governance"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/storage-governance", "get_storage_governance"
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "storage-integrity"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/storage-integrity", "get_storage_integrity"
	case method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "campaigns", "*", "tasks"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/tasks", "create_task"
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "quality-profile"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/quality-profile", "get_quality_profile"
	case method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "quality-profile"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/quality-profile", "update_quality_profile"
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "visual-context"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/visual-context", "get_visual_context"
	case method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "visual-context"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/visual-context", "update_visual_context"
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "provider-profile"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/provider-profile", "get_provider_profile"
	case method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "provider-profile"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/provider-profile", "update_provider_profile"
	case isRead && match(parts, "api", "workspaces", "*", "projects", "*", "access-config"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/access-config", "get_access_config"
	case method == http.MethodPost && match(parts, "api", "workspaces", "*", "projects", "*", "access-config"):
		return "/api/workspaces/{workspace_id}/projects/{project_id}/access-config", "update_access_config"
	case isRead && match(parts, "api", "admin", "runtime-status"):
		return "/api/admin/runtime-status", "get_runtime_status"
	case isRead && match(parts, "api", "tasks", "*", "attempts"):
		return "/api/tasks/{task_id}/attempts", "list_task_attempts"
	case isRead && match(parts, "api", "tasks", "*"):
		return "/api/tasks/{task_id}", "get_task"
	case isRead && match(parts, "api", "projects", "*", "campaigns", "*", "assets"):
		return "/api/projects/{project_id}/campaigns/{campaign_id}/assets", "list_assets"
	case isRead && match(parts, "api", "projects", "*", "campaigns", "*", "batch-progress"):
		return "/api/projects/{project_id}/campaigns/{campaign_id}/batch-progress", "get_batch_progress"
	case isRead && match(parts, "api", "projects", "*", "campaigns", "*", "batch-summary"):
		return "/api/projects/{project_id}/campaigns/{campaign_id}/batch-summary", "get_batch_summary"
	case isRead && match(parts, "api", "projects", "*", "campaigns", "*", "batch-manifest"):
		return "/api/projects/{project_id}/campaigns/{campaign_id}/batch-manifest", "get_batch_manifest"
	case method == http.MethodPost && match(parts, "api", "projects", "*", "campaigns", "*", "scene-regenerations"):
		return "/api/projects/{project_id}/campaigns/{campaign_id}/scene-regenerations", "regenerate_scene"
	case isRead && match(parts, "api", "assets", "*"):
		return "/api/assets/{asset_id}", "get_asset"
	case isRead && match(parts, "api", "assets", "*", "metadata"):
		return "/api/assets/{asset_id}/metadata", "get_asset_metadata"
	case method == http.MethodPost && match(parts, "api", "assets", "*", "approve"):
		return "/api/assets/{asset_id}/approve", "approve_asset"
	case method == http.MethodPost && match(parts, "api", "assets", "*", "reject"):
		return "/api/assets/{asset_id}/reject", "reject_asset"
	case isRead && match(parts, "api", "assets", "*", "original"):
		return "/api/assets/{asset_id}/original", "get_asset_original"
	case isRead && match(parts, "api", "assets", "*", "thumbnail"):
		return "/api/assets/{asset_id}/thumbnail", "get_asset_thumbnail"
	default:
		return "unknown", "unknown"
	}
}
