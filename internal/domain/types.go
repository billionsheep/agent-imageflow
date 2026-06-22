package domain

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
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

	SelectionManualOptional = "manual_optional"
	SelectionAuto           = "auto"
	SelectionBestOf         = "best_of"

	BestOfStrategyLocalMetadata = "local_metadata_v1"
	BestOfStrategyHTTPJudge     = "http_judge_v1"

	InputFileKindReference = "reference"
	InputFileKindMask      = "mask"

	ProjectAccessActionAddKey    = "add_key"
	ProjectAccessActionUpdateKey = "update_key"
	ProjectAccessActionDeleteKey = "delete_key"

	ProjectAPIKeyDefaultID   = "default"
	ProjectAPIKeyDefaultName = "default"
)

type Scope struct {
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	CampaignID  string `json:"campaign_id"`
}

type CreateTaskRequest struct {
	IdempotencyKey           string           `json:"idempotency_key"`
	Title                    string           `json:"title"`
	Purpose                  string           `json:"purpose"`
	Prompt                   string           `json:"prompt"`
	NegativePrompt           string           `json:"negative_prompt"`
	StylePreset              string           `json:"style_preset"`
	PromptTemplate           string           `json:"prompt_template"`
	TemplateVariables        map[string]any   `json:"template_variables"`
	ReferenceImages          []ReferenceImage `json:"reference_images"`
	CharacterIDs             []string         `json:"character_ids,omitempty"`
	ReferenceAssetIDs        []string         `json:"reference_asset_ids,omitempty"`
	PromptRecipeID           string           `json:"prompt_recipe_id,omitempty"`
	UseProjectVisualContext  bool             `json:"use_project_visual_context,omitempty"`
	MaskImage                *MaskImage       `json:"mask_image,omitempty"`
	BestOfConfig             *BestOfConfig    `json:"best_of_config,omitempty"`
	GenerationConfig         json.RawMessage  `json:"generation_config"`
	UseProjectQualityProfile bool             `json:"use_project_quality_profile"`
	AspectRatio              string           `json:"aspect_ratio"`
	OutputFormat             string           `json:"output_format"`
	RequestedCount           int              `json:"requested_count"`
	Provider                 string           `json:"provider"`
	SelectionMode            string           `json:"selection_mode"`
	ReviewRequired           bool             `json:"review_required"`
	MetadataJSON             json.RawMessage  `json:"metadata_json"`
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
	SelectionMode       string          `json:"selection_mode,omitempty"`
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
	Version                 AssetVersion    `json:"version"`
	TaskStructuredInputJSON json.RawMessage `json:"task_structured_input_json,omitempty"`
}

type TaskResponse struct {
	Task
	AssetIDs []string         `json:"asset_ids"`
	Assets   []AssetListEntry `json:"assets"`
}

type TaskAttempt struct {
	ID                  string     `json:"attempt_id"`
	TaskID              string     `json:"task_id"`
	AttemptNo           int        `json:"attempt_no"`
	Status              string     `json:"status"`
	Provider            string     `json:"provider"`
	ProviderRequestID   string     `json:"provider_request_id,omitempty"`
	RequestMode         string     `json:"request_mode,omitempty"`
	APIMode             string     `json:"api_mode,omitempty"`
	Stream              bool       `json:"stream,omitempty"`
	PartialImageCount   int        `json:"partial_image_count,omitempty"`
	StartedAt           time.Time  `json:"started_at"`
	FinishedAt          *time.Time `json:"finished_at,omitempty"`
	LatencyMs           *int       `json:"latency_ms,omitempty"`
	QueueWaitMs         *int       `json:"queue_wait_ms,omitempty"`
	ProviderFirstByteMs *int       `json:"provider_first_byte_ms,omitempty"`
	ProviderTotalMs     *int       `json:"provider_total_ms,omitempty"`
	ResponseDownloadMs  *int       `json:"response_download_ms,omitempty"`
	StoreMs             *int       `json:"store_ms,omitempty"`
	ThumbnailMs         *int       `json:"thumbnail_ms,omitempty"`
	RetryCount          int        `json:"retry_count"`
	ErrorStage          string     `json:"error_stage,omitempty"`
	ResponseBytes       int64      `json:"response_bytes,omitempty"`
	RetryAfter          *time.Time `json:"retry_after,omitempty"`
	ErrorCode           *string    `json:"error_code,omitempty"`
	ErrorMessage        *string    `json:"error_message,omitempty"`
}

type TaskAttemptsResponse struct {
	TaskID   string        `json:"task_id"`
	Attempts []TaskAttempt `json:"attempts"`
}

type AttemptMetrics struct {
	QueueWaitMs         int64  `json:"queue_wait_ms,omitempty"`
	ProviderFirstByteMs int64  `json:"provider_first_byte_ms,omitempty"`
	ProviderTotalMs     int64  `json:"provider_total_ms,omitempty"`
	ResponseDownloadMs  int64  `json:"response_download_ms,omitempty"`
	StoreMs             int64  `json:"store_ms,omitempty"`
	ThumbnailMs         int64  `json:"thumbnail_ms,omitempty"`
	RetryCount          int    `json:"retry_count,omitempty"`
	ErrorStage          string `json:"error_stage,omitempty"`
	ResponseBytes       int64  `json:"response_bytes,omitempty"`
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
	MetadataJSON   json.RawMessage `json:"metadata_json"`
	Delivery       DeliveryInfo    `json:"delivery"`
	CreatedAt      time.Time       `json:"created_at"`
}

type AssetListQuery struct {
	ProjectID   string
	CampaignID  string
	Limit       int
	Offset      int
	Status      string
	Provider    string
	Model       string
	Source      string
	SessionID   string
	BatchID     string
	Keyword     string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

type BatchProgressQuery struct {
	ProjectID  string
	CampaignID string
	SessionID  string
	BatchID    string
	Limit      int
}

type BatchProgressTask struct {
	TaskID       string    `json:"task_id"`
	Status       string    `json:"status"`
	AssetCount   int       `json:"asset_count"`
	AttemptCount int       `json:"attempt_count"`
	Retrying     bool      `json:"retrying"`
	ErrorStage   string    `json:"error_stage,omitempty"`
	ErrorCode    *string   `json:"error_code,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type BatchProgressCounts struct {
	TaskCount      int `json:"task_count"`
	QueuedCount    int `json:"queued_count"`
	RunningCount   int `json:"running_count"`
	SucceededCount int `json:"succeeded_count"`
	PartialCount   int `json:"partial_count"`
	FailedCount    int `json:"failed_count"`
	RetryingCount  int `json:"retrying_count"`
	AssetCount     int `json:"asset_count"`
	AttemptCount   int `json:"attempt_count"`
}

type BatchProgressResponse struct {
	GeneratedAt time.Time           `json:"generated_at"`
	ProjectID   string              `json:"project_id"`
	CampaignID  string              `json:"campaign_id"`
	SessionID   string              `json:"session_id,omitempty"`
	BatchID     string              `json:"batch_id,omitempty"`
	Counts      BatchProgressCounts `json:"counts"`
	Tasks       []BatchProgressTask `json:"tasks"`
}

const (
	DefaultAssetListLimit     = 50
	MaxAssetListLimit         = 100
	DefaultBatchProgressLimit = 100
	MaxBatchProgressLimit     = 500
)

var StandardMetadataFields = []string{
	"source",
	"source_agent",
	"source_thread_id",
	"session_id",
	"run_id",
	"batch_id",
	"story_id",
	"scene_id",
	"target_path",
}

type DeliveryInfo struct {
	LocalPath    string `json:"local_path"`
	DownloadURL  string `json:"download_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	MetadataURL  string `json:"metadata_url"`
}

type StorageUsageCategoryStat struct {
	Category  string `json:"category"`
	FileCount int64  `json:"file_count"`
	Bytes     int64  `json:"bytes"`
}

type StorageUsageSnapshot struct {
	ScopeType   string                     `json:"scope_type"`
	WorkspaceID string                     `json:"workspace_id,omitempty"`
	ProjectID   string                     `json:"project_id,omitempty"`
	CampaignID  string                     `json:"campaign_id,omitempty"`
	FileCount   int64                      `json:"file_count"`
	Bytes       int64                      `json:"bytes"`
	Categories  []StorageUsageCategoryStat `json:"categories"`
}

type StorageUsageScopes struct {
	Instance  StorageUsageSnapshot `json:"instance"`
	Workspace StorageUsageSnapshot `json:"workspace"`
	Project   StorageUsageSnapshot `json:"project"`
	Campaign  StorageUsageSnapshot `json:"campaign"`
}

type StorageGovernanceCountSnapshot struct {
	TaskCount           int64 `json:"task_count"`
	FailedTaskCount     int64 `json:"failed_task_count"`
	AssetCount          int64 `json:"asset_count"`
	GeneratedAssetCount int64 `json:"generated_asset_count"`
	SelectedAssetCount  int64 `json:"selected_asset_count"`
	RejectedAssetCount  int64 `json:"rejected_asset_count"`
	PublishedAssetCount int64 `json:"published_asset_count"`
}

type StorageGovernanceCounts struct {
	Instance  StorageGovernanceCountSnapshot `json:"instance"`
	Workspace StorageGovernanceCountSnapshot `json:"workspace"`
	Project   StorageGovernanceCountSnapshot `json:"project"`
	Campaign  StorageGovernanceCountSnapshot `json:"campaign"`
}

type StorageGovernanceResponse struct {
	GeneratedAt time.Time               `json:"generated_at"`
	Scope       Scope                   `json:"scope"`
	Usage       StorageUsageScopes      `json:"usage"`
	Counts      StorageGovernanceCounts `json:"counts"`
}

type StorageIntegrityResponse struct {
	CheckedAt time.Time               `json:"checked_at"`
	Scope     Scope                   `json:"scope"`
	OK        bool                    `json:"ok"`
	Summary   StorageIntegritySummary `json:"summary"`
	Issues    []StorageIntegrityIssue `json:"issues"`
}

type StorageIntegritySummary struct {
	IssueCount int            `json:"issue_count"`
	ByKind     map[string]int `json:"by_kind"`
}

type StorageIntegrityIssue struct {
	Kind       string `json:"kind"`
	Severity   string `json:"severity"`
	TaskID     string `json:"task_id,omitempty"`
	AssetID    string `json:"asset_id,omitempty"`
	VersionID  string `json:"version_id,omitempty"`
	Status     string `json:"status,omitempty"`
	FileKind   string `json:"file_kind,omitempty"`
	Message    string `json:"message"`
	RepairHint string `json:"repair_hint,omitempty"`
}

type CleanupDryRunOptions struct {
	Scope                Scope
	IncludeRejected      bool
	IncludeGenerated     bool
	IncludeFailedTaskTmp bool
	IncludeOrphans       bool
	Limit                int
}

type CleanupDryRunReport struct {
	GeneratedAt time.Time             `json:"generated_at"`
	DryRun      bool                  `json:"dry_run"`
	DryRunToken string                `json:"dry_run_token"`
	Scope       Scope                 `json:"scope"`
	Summary     CleanupDryRunSummary  `json:"summary"`
	Candidates  []CleanupCandidate    `json:"candidates"`
	Protected   CleanupProtectedStats `json:"protected"`
}

type CleanupDryRunSummary struct {
	CandidateCount int            `json:"candidate_count"`
	FileCount      int64          `json:"file_count"`
	Bytes          int64          `json:"bytes"`
	ByReason       map[string]int `json:"by_reason"`
}

type CleanupProtectedStats struct {
	SelectedAssetCount  int64 `json:"selected_asset_count"`
	PublishedAssetCount int64 `json:"published_asset_count"`
}

type CleanupCandidate struct {
	Kind      string                 `json:"kind"`
	Reason    string                 `json:"reason"`
	AssetID   string                 `json:"asset_id,omitempty"`
	TaskID    string                 `json:"task_id,omitempty"`
	Status    string                 `json:"status,omitempty"`
	FileCount int64                  `json:"file_count"`
	Bytes     int64                  `json:"bytes"`
	Files     []CleanupCandidateFile `json:"files,omitempty"`
}

type CleanupCandidateFile struct {
	Kind       string `json:"kind"`
	StorageKey string `json:"storage_key,omitempty"`
	Bytes      int64  `json:"bytes"`
}

type CleanupExecuteOptions struct {
	Scope                Scope
	IncludeRejected      bool
	IncludeGenerated     bool
	IncludeFailedTaskTmp bool
	IncludeOrphans       bool
	Limit                int
	DryRunToken          string
	Execute              bool
	Confirm              bool
	Actor                string
}

type CleanupExecutionReport struct {
	GeneratedAt  time.Time                `json:"generated_at"`
	DryRun       bool                     `json:"dry_run"`
	Executed     bool                     `json:"executed"`
	Scope        Scope                    `json:"scope"`
	DryRunToken  string                   `json:"dry_run_token"`
	Summary      CleanupExecutionSummary  `json:"summary"`
	Results      []CleanupExecutionResult `json:"results"`
	Protected    CleanupProtectedStats    `json:"protected"`
	AuditEventID string                   `json:"audit_event_id,omitempty"`
}

type CleanupExecutionSummary struct {
	CandidateCount        int            `json:"candidate_count"`
	DeletedCandidateCount int            `json:"deleted_candidate_count"`
	SkippedCandidateCount int            `json:"skipped_candidate_count"`
	FailedCandidateCount  int            `json:"failed_candidate_count"`
	FileCount             int64          `json:"file_count"`
	DeletedFileCount      int64          `json:"deleted_file_count"`
	Bytes                 int64          `json:"bytes"`
	DeletedBytes          int64          `json:"deleted_bytes"`
	ByReason              map[string]int `json:"by_reason"`
}

type CleanupExecutionResult struct {
	Kind    string                 `json:"kind"`
	Reason  string                 `json:"reason"`
	AssetID string                 `json:"asset_id,omitempty"`
	TaskID  string                 `json:"task_id,omitempty"`
	Status  string                 `json:"status,omitempty"`
	Action  string                 `json:"action"`
	Error   string                 `json:"error,omitempty"`
	Files   []CleanupExecutionFile `json:"files,omitempty"`
}

type CleanupExecutionFile struct {
	Kind       string `json:"kind"`
	StorageKey string `json:"storage_key,omitempty"`
	Bytes      int64  `json:"bytes"`
	Action     string `json:"action"`
	Error      string `json:"error,omitempty"`
}

type ReferenceImage struct {
	ID          string  `json:"id,omitempty"`
	URL         string  `json:"url,omitempty"`
	AssetID     string  `json:"asset_id,omitempty"`
	InputFileID string  `json:"input_file_id,omitempty"`
	Role        string  `json:"role,omitempty"`
	Source      string  `json:"source,omitempty"`
	MimeType    string  `json:"mime_type,omitempty"`
	Width       int     `json:"width,omitempty"`
	Height      int     `json:"height,omitempty"`
	Weight      float64 `json:"weight,omitempty"`
}

type MaskImage struct {
	ID            string `json:"id,omitempty"`
	URL           string `json:"url,omitempty"`
	AssetID       string `json:"asset_id,omitempty"`
	InputFileID   string `json:"input_file_id,omitempty"`
	TargetImageID string `json:"target_image_id,omitempty"`
	Source        string `json:"source,omitempty"`
	MimeType      string `json:"mime_type,omitempty"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	HasMask       bool   `json:"has_mask,omitempty"`
}

type QualityProfile struct {
	PromptTemplate   string           `json:"prompt_template,omitempty"`
	NegativePrompt   string           `json:"negative_prompt,omitempty"`
	StylePreset      string           `json:"style_preset,omitempty"`
	ReferenceImages  []ReferenceImage `json:"reference_images,omitempty"`
	BestOfConfig     *BestOfConfig    `json:"best_of_config,omitempty"`
	GenerationConfig json.RawMessage  `json:"generation_config,omitempty"`
}

type BestOfConfig struct {
	Strategy              string `json:"strategy,omitempty"`
	JudgePrompt           string `json:"judge_prompt,omitempty"`
	AutoRejectNonSelected bool   `json:"auto_reject_non_selected,omitempty"`
}

type CharacterProfile struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Status            string    `json:"status"`
	UpdatedAt         time.Time `json:"updated_at"`
	Role              string    `json:"role,omitempty"`
	Appearance        string    `json:"appearance,omitempty"`
	Personality       string    `json:"personality,omitempty"`
	Forbidden         []string  `json:"forbidden,omitempty"`
	PrimaryAssetID    string    `json:"primary_asset_id,omitempty"`
	ReferenceAssetIDs []string  `json:"reference_asset_ids,omitempty"`
}

type ProjectReferenceBinding struct {
	ID          string    `json:"id"`
	AssetID     string    `json:"asset_id"`
	Purpose     string    `json:"purpose"`
	Label       string    `json:"label,omitempty"`
	Weight      float64   `json:"weight,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	CharacterID string    `json:"character_id,omitempty"`
	Status      string    `json:"status"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PromptBlock struct {
	ID   string `json:"id,omitempty"`
	Role string `json:"role,omitempty"`
	Text string `json:"text"`
}

type PromptRecipe struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	Status              string          `json:"status"`
	UpdatedAt           time.Time       `json:"updated_at"`
	PromptBlocks        []PromptBlock   `json:"prompt_blocks,omitempty"`
	NegativePrompt      string          `json:"negative_prompt,omitempty"`
	DefaultAspectRatio  string          `json:"default_aspect_ratio,omitempty"`
	DefaultOutputFormat string          `json:"default_output_format,omitempty"`
	DefaultProvider     string          `json:"default_provider,omitempty"`
	DefaultModel        string          `json:"default_model,omitempty"`
	GenerationConfig    json.RawMessage `json:"generation_config,omitempty"`
}

type ProjectVisualContext struct {
	Characters    []CharacterProfile        `json:"characters,omitempty"`
	References    []ProjectReferenceBinding `json:"references,omitempty"`
	PromptRecipes []PromptRecipe            `json:"prompt_recipes,omitempty"`
	UpdatedAt     time.Time                 `json:"updated_at,omitempty"`
}

type ProjectVisualContextResponse struct {
	WorkspaceID   string               `json:"workspace_id"`
	ProjectID     string               `json:"project_id"`
	VisualContext ProjectVisualContext `json:"visual_context"`
}

type VisualContextSnapshot struct {
	Source            string                    `json:"source"`
	CharacterIDs      []string                  `json:"character_ids,omitempty"`
	ReferenceAssetIDs []string                  `json:"reference_asset_ids,omitempty"`
	PromptRecipeID    string                    `json:"prompt_recipe_id,omitempty"`
	Characters        []CharacterProfile        `json:"characters,omitempty"`
	References        []ProjectReferenceBinding `json:"references,omitempty"`
	PromptRecipe      *PromptRecipe             `json:"prompt_recipe,omitempty"`
}

type ProjectQualityProfileResponse struct {
	WorkspaceID    string         `json:"workspace_id"`
	ProjectID      string         `json:"project_id"`
	QualityProfile QualityProfile `json:"quality_profile"`
}

type ProjectProviderProfile struct {
	Enabled                  bool            `json:"enabled"`
	Provider                 string          `json:"provider,omitempty"`
	Model                    string          `json:"model,omitempty"`
	BaseURL                  string          `json:"base_url,omitempty"`
	GenerationConfig         json.RawMessage `json:"generation_config,omitempty"`
	UseProjectQualityProfile bool            `json:"use_project_quality_profile,omitempty"`
	APIMode                  string          `json:"api_mode,omitempty"`
	Stream                   *bool           `json:"stream,omitempty"`
	PartialImages            *int            `json:"partial_images,omitempty"`
	MaxN                     int             `json:"max_n,omitempty"`
	SupportsURLResult        bool            `json:"supports_url_result,omitempty"`
	PreferredResponseFormat  string          `json:"preferred_response_format,omitempty"`
	MaxConcurrency           int             `json:"max_concurrency,omitempty"`
	TimeoutSeconds           int             `json:"timeout_seconds,omitempty"`
}

type ProjectProviderProfileResponse struct {
	WorkspaceID     string                 `json:"workspace_id"`
	ProjectID       string                 `json:"project_id"`
	ProviderProfile ProjectProviderProfile `json:"provider_profile"`
}

type WorkspaceSummary struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Archived    bool   `json:"archived"`
}

type ProjectSummary struct {
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Archived    bool   `json:"archived"`
}

type CampaignSummary struct {
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	CampaignID  string `json:"campaign_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Archived    bool   `json:"archived"`
}

type WorkspaceListResponse struct {
	Workspaces []WorkspaceSummary `json:"workspaces"`
}

type ProjectListResponse struct {
	WorkspaceID string           `json:"workspace_id"`
	Projects    []ProjectSummary `json:"projects"`
}

type CampaignListResponse struct {
	WorkspaceID string            `json:"workspace_id"`
	ProjectID   string            `json:"project_id"`
	Campaigns   []CampaignSummary `json:"campaigns"`
}

type CreateWorkspaceRequest struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
}

type CreateProjectRequest struct {
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type CreateCampaignRequest struct {
	CampaignID  string `json:"campaign_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type UpdateWorkspaceRequest struct {
	Name     *string `json:"name,omitempty"`
	Archived *bool   `json:"archived,omitempty"`
}

type UpdateProjectRequest struct {
	Name     *string `json:"name,omitempty"`
	Archived *bool   `json:"archived,omitempty"`
}

type UpdateCampaignRequest struct {
	Name     *string `json:"name,omitempty"`
	Archived *bool   `json:"archived,omitempty"`
}

type InputFileResponse struct {
	InputFileID      string `json:"input_file_id"`
	WorkspaceID      string `json:"workspace_id"`
	ProjectID        string `json:"project_id"`
	CampaignID       string `json:"campaign_id"`
	Kind             string `json:"kind"`
	OriginalFilename string `json:"original_filename"`
	MimeType         string `json:"mime_type"`
	Width            int    `json:"width,omitempty"`
	Height           int    `json:"height,omitempty"`
	SizeBytes        int64  `json:"size_bytes"`
	DownloadURL      string `json:"download_url"`
	MetadataURL      string `json:"metadata_url"`
}

type ProjectAccessConfig struct {
	APIKeyEnabled bool            `json:"api_key_enabled"`
	APIKeyName    string          `json:"api_key_name,omitempty"`
	APIKeyPreview string          `json:"api_key_preview,omitempty"`
	APIKeyHash    string          `json:"api_key_hash,omitempty"`
	APIKeys       []ProjectAPIKey `json:"api_keys,omitempty"`
}

type ProjectAPIKey struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Preview string `json:"preview,omitempty"`
	Hash    string `json:"hash,omitempty"`
	Enabled bool   `json:"enabled"`
}

type ProjectAPIKeyView struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Preview string `json:"preview,omitempty"`
	Enabled bool   `json:"enabled"`
}

type ProjectAccessConfigView struct {
	APIKeyEnabled bool                `json:"api_key_enabled"`
	APIKeyName    string              `json:"api_key_name,omitempty"`
	APIKeyPreview string              `json:"api_key_preview,omitempty"`
	APIKeys       []ProjectAPIKeyView `json:"api_keys,omitempty"`
}

type ProjectAccessConfigResponse struct {
	WorkspaceID  string                  `json:"workspace_id"`
	ProjectID    string                  `json:"project_id"`
	AccessConfig ProjectAccessConfigView `json:"access_config"`
}

type ProjectAccessConfigUpdateRequest struct {
	Action        string `json:"action,omitempty"`
	APIKeyID      string `json:"api_key_id,omitempty"`
	APIKeyEnabled *bool  `json:"api_key_enabled"`
	APIKeyName    string `json:"api_key_name,omitempty"`
	APIKey        string `json:"api_key,omitempty"`
}

func (k ProjectAPIKey) Public() ProjectAPIKeyView {
	return ProjectAPIKeyView{
		ID:      strings.TrimSpace(k.ID),
		Name:    strings.TrimSpace(k.Name),
		Preview: strings.TrimSpace(k.Preview),
		Enabled: k.Enabled && strings.TrimSpace(k.Hash) != "",
	}
}

func (c ProjectAccessConfig) Keys() []ProjectAPIKey {
	keys := make([]ProjectAPIKey, 0, len(c.APIKeys)+1)
	if len(c.APIKeys) > 0 {
		for idx, key := range c.APIKeys {
			normalized := normalizeProjectAPIKeyEntry(key, fmt.Sprintf("key_%d", idx+1))
			if normalized == nil {
				continue
			}
			keys = append(keys, *normalized)
		}
		return keys
	}
	if c.APIKeyEnabled && strings.TrimSpace(c.APIKeyHash) != "" {
		legacy := normalizeProjectAPIKeyEntry(ProjectAPIKey{
			ID:      ProjectAPIKeyDefaultID,
			Name:    c.APIKeyName,
			Preview: c.APIKeyPreview,
			Hash:    c.APIKeyHash,
			Enabled: true,
		}, ProjectAPIKeyDefaultID)
		if legacy != nil {
			keys = append(keys, *legacy)
		}
	}
	return keys
}

func (c ProjectAccessConfig) IsEnabled() bool {
	for _, key := range c.Keys() {
		if key.Enabled && strings.TrimSpace(key.Hash) != "" {
			return true
		}
	}
	return false
}

func (c ProjectAccessConfig) Normalize() ProjectAccessConfig {
	keys := c.Keys()
	if len(keys) == 0 {
		return ProjectAccessConfig{}
	}
	normalized := ProjectAccessConfig{
		APIKeys: make([]ProjectAPIKey, 0, len(keys)),
	}
	normalized.APIKeys = append(normalized.APIKeys, keys...)
	primary := keys[0]
	for _, key := range keys {
		if key.Enabled {
			primary = key
			break
		}
	}
	normalized.APIKeyEnabled = normalized.IsEnabled()
	normalized.APIKeyName = primary.Name
	normalized.APIKeyPreview = primary.Preview
	normalized.APIKeyHash = primary.Hash
	return normalized
}

func (c ProjectAccessConfig) Public() ProjectAccessConfigView {
	normalized := c.Normalize()
	keys := normalized.APIKeys
	publicKeys := make([]ProjectAPIKeyView, 0, len(keys))
	for _, key := range keys {
		publicKeys = append(publicKeys, key.Public())
	}
	return ProjectAccessConfigView{
		APIKeyEnabled: normalized.IsEnabled(),
		APIKeyName:    normalized.APIKeyName,
		APIKeyPreview: normalized.APIKeyPreview,
		APIKeys:       publicKeys,
	}
}

func NormalizeMetadataJSON(raw json.RawMessage) json.RawMessage {
	var metadata map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &metadata) != nil || metadata == nil {
		return json.RawMessage(`{}`)
	}
	for _, field := range StandardMetadataFields {
		value, exists := metadata[field]
		if !exists {
			continue
		}
		text, ok := value.(string)
		if !ok {
			delete(metadata, field)
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			delete(metadata, field)
			continue
		}
		metadata[field] = text
	}
	normalized, err := json.Marshal(metadata)
	if err != nil || !json.Valid(normalized) {
		return json.RawMessage(`{}`)
	}
	return normalized
}

func normalizeProjectAPIKeyEntry(key ProjectAPIKey, fallbackID string) *ProjectAPIKey {
	hash := strings.TrimSpace(key.Hash)
	if hash == "" {
		return nil
	}
	id := strings.TrimSpace(key.ID)
	if id == "" {
		id = strings.TrimSpace(fallbackID)
	}
	if id == "" {
		id = ProjectAPIKeyDefaultID
	}
	name := strings.TrimSpace(key.Name)
	if name == "" {
		if id == ProjectAPIKeyDefaultID {
			name = ProjectAPIKeyDefaultName
		} else {
			name = id
		}
	}
	preview := strings.TrimSpace(key.Preview)
	return &ProjectAPIKey{
		ID:      id,
		Name:    name,
		Preview: preview,
		Hash:    hash,
		Enabled: key.Enabled,
	}
}

func NewID(prefix string) string {
	var buf [10]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buf[:]))
}

func NormalizeSelectionMode(value string) (string, bool) {
	switch strings.TrimSpace(value) {
	case "":
		return SelectionManualOptional, true
	case SelectionManualOptional:
		return SelectionManualOptional, true
	case SelectionAuto:
		return SelectionAuto, true
	case SelectionBestOf:
		return SelectionBestOf, true
	default:
		return "", false
	}
}

func NormalizeInputFileKind(value string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", InputFileKindReference:
		return InputFileKindReference, true
	case InputFileKindMask:
		return InputFileKindMask, true
	default:
		return "", false
	}
}

func NormalizeBestOfStrategy(value string) (string, bool) {
	switch strings.TrimSpace(value) {
	case "", BestOfStrategyLocalMetadata:
		return BestOfStrategyLocalMetadata, true
	case BestOfStrategyHTTPJudge:
		return BestOfStrategyHTTPJudge, true
	default:
		return "", false
	}
}

func ShouldAutoSelect(selectionMode string) bool {
	mode, ok := NormalizeSelectionMode(selectionMode)
	if !ok {
		return false
	}
	return mode == SelectionAuto || mode == SelectionBestOf
}
