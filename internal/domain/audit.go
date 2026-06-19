package domain

import "time"

const (
	HTTPAuditSourceAPI = "http_api"
	HTTPAuditSourceCLI = "cli"
)

type HTTPAuditEvent struct {
	EventID           string    `json:"event_id"`
	Timestamp         time.Time `json:"timestamp"`
	Source            string    `json:"source"`
	RequestID         string    `json:"request_id"`
	Method            string    `json:"method"`
	Path              string    `json:"path"`
	Route             string    `json:"route"`
	Action            string    `json:"action"`
	StatusCode        int       `json:"status_code"`
	DurationMs        int64     `json:"duration_ms"`
	Success           bool      `json:"success"`
	AuthMode          string    `json:"auth_mode"`
	Actor             string    `json:"actor"`
	BasicAuthUser     string    `json:"basic_auth_user,omitempty"`
	ProjectAPIKeyName string    `json:"project_api_key_name,omitempty"`
	WorkspaceID       string    `json:"workspace_id,omitempty"`
	ProjectID         string    `json:"project_id,omitempty"`
	CampaignID        string    `json:"campaign_id,omitempty"`
	TaskID            string    `json:"task_id,omitempty"`
	AssetID           string    `json:"asset_id,omitempty"`
	InputFileID       string    `json:"input_file_id,omitempty"`
	ErrorCode         string    `json:"error_code,omitempty"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	RemoteAddr        string    `json:"remote_addr,omitempty"`
	UserAgent         string    `json:"user_agent,omitempty"`
}

type HTTPAuditQuery struct {
	Limit       int
	WorkspaceID string
	ProjectID   string
	CampaignID  string
	TaskID      string
	AssetID     string
	InputFileID string
	Action      string
	Actor       string
	StatusCode  int
}

type HTTPAuditListResponse struct {
	Events []HTTPAuditEvent `json:"events"`
}
