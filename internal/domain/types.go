package domain

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

const (
	TaskQueued             = "queued"
	TaskRunning            = "running"
	TaskCompleted          = "completed"
	TaskPartiallyCompleted = "partially_completed"
	TaskFailed             = "failed"
	TaskEnqueueFailed      = "enqueue_failed"

	AttemptRunning   = "running"
	AttemptCompleted = "completed"
	AttemptFailed    = "failed"

	AssetDraft      = "draft"
	AssetApproved   = "approved"
	AssetRejected   = "rejected"
	AssetPublished  = "published"
	AssetDeprecated = "deprecated"

	VersionReady  = "ready"
	VersionFailed = "failed"
)

type Scope struct {
	WorkspaceID string
	ProjectID   string
	CampaignID  string
}

type CreateTaskRequest struct {
	IdempotencyKey string          `json:"idempotency_key"`
	Title          string          `json:"title"`
	Purpose        string          `json:"purpose"`
	Prompt         string          `json:"prompt"`
	NegativePrompt string          `json:"negative_prompt"`
	StylePreset    string          `json:"style_preset"`
	AspectRatio    string          `json:"aspect_ratio"`
	OutputFormat   string          `json:"output_format"`
	RequestedCount int             `json:"requested_count"`
	Provider       string          `json:"provider"`
	ReviewRequired bool            `json:"review_required"`
	MetadataJSON   json.RawMessage `json:"metadata_json"`
}

type Task struct {
	ID                  string          `json:"task_id"`
	WorkspaceID         string          `json:"workspace_id"`
	ProjectID           string          `json:"project_id"`
	CampaignID          string          `json:"campaign_id"`
	IdempotencyKey      string          `json:"idempotency_key,omitempty"`
	Title               string          `json:"title"`
	Purpose             string          `json:"purpose"`
	Prompt              string          `json:"prompt"`
	NegativePrompt      string          `json:"negative_prompt,omitempty"`
	StylePreset         string          `json:"style_preset,omitempty"`
	AspectRatio         string          `json:"aspect_ratio"`
	OutputFormat        string          `json:"output_format"`
	StructuredInputJSON json.RawMessage `json:"structured_input_json,omitempty"`
	Provider            string          `json:"provider"`
	Status              string          `json:"status"`
	RequestedCount      int             `json:"requested_count"`
	CreatedBy           string          `json:"created_by"`
	TraceID             string          `json:"trace_id,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	ErrorCode           *string         `json:"error_code"`
	ErrorMessage        *string         `json:"error_message"`
}

type Asset struct {
	ID               string    `json:"asset_id"`
	WorkspaceID      string    `json:"workspace_id"`
	ProjectID        string    `json:"project_id"`
	CampaignID       string    `json:"campaign_id"`
	TaskID           string    `json:"task_id"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	CurrentVersionID string    `json:"current_version_id,omitempty"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type AssetVersion struct {
	ID             string          `json:"asset_version_id"`
	AssetID        string          `json:"asset_id"`
	Version        int             `json:"version"`
	Status         string          `json:"status"`
	FilePath       string          `json:"file_path"`
	ThumbnailPath  string          `json:"thumbnail_path"`
	MetadataPath   string          `json:"metadata_path"`
	ObjectKey      string          `json:"object_key,omitempty"`
	PublicURL      string          `json:"public_url,omitempty"`
	MimeType       string          `json:"mime_type"`
	Width          int             `json:"width"`
	Height         int             `json:"height"`
	Hash           string          `json:"hash"`
	Provider       string          `json:"provider"`
	Model          string          `json:"model"`
	Prompt         string          `json:"prompt"`
	ParametersJSON json.RawMessage `json:"parameters_json,omitempty"`
	CostJSON       json.RawMessage `json:"cost_json,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

type AssetWithVersion struct {
	Asset
	Version AssetVersion `json:"version"`
}

type TaskResponse struct {
	Task
	AssetIDs []string         `json:"asset_ids"`
	Assets   []AssetListEntry `json:"assets"`
}

type AssetListEntry struct {
	AssetID      string `json:"asset_id"`
	Status       string `json:"status"`
	ThumbnailURL string `json:"thumbnail_url"`
	MetadataURL  string `json:"metadata_url"`
}

type AssetResponse struct {
	AssetID        string          `json:"asset_id"`
	WorkspaceID    string          `json:"workspace_id"`
	ProjectID      string          `json:"project_id"`
	CampaignID     string          `json:"campaign_id"`
	TaskID         string          `json:"task_id"`
	CurrentVersion int             `json:"current_version"`
	Status         string          `json:"status"`
	Hash           string          `json:"hash"`
	Provider       string          `json:"provider"`
	Model          string          `json:"model"`
	Prompt         string          `json:"prompt"`
	ParametersJSON json.RawMessage `json:"parameters_json"`
	Delivery       DeliveryInfo    `json:"delivery"`
	CreatedAt      time.Time       `json:"created_at"`
}

type DeliveryInfo struct {
	LocalPath    string `json:"local_path"`
	DownloadURL  string `json:"download_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	MetadataURL  string `json:"metadata_url"`
}

func NewID(prefix string) string {
	var buf [10]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buf[:]))
}
