package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/app"
	"github.com/billionsheep/agent-imageflow/internal/config"
	"github.com/billionsheep/agent-imageflow/internal/db"
	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/provider"
	"github.com/billionsheep/agent-imageflow/internal/queue"
	"github.com/billionsheep/agent-imageflow/internal/storage"
	"github.com/billionsheep/agent-imageflow/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "task":
		err = taskCmd(os.Args[2:])
	case "asset":
		err = assetCmd(os.Args[2:])
	case "audit":
		err = auditCmd(os.Args[2:])
	case "benchmark":
		err = benchmarkCmd(os.Args[2:])
	case "batch":
		err = batchCmd(os.Args[2:])
	case "project":
		err = projectCmd(os.Args[2:])
	case "repair":
		err = repairCmd(os.Args[2:])
	case "storage":
		err = storageCmd(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func batchCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: vag batch progress|manifest")
	}
	switch args[0] {
	case "progress":
		fs := flag.NewFlagSet("vag batch progress", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		campaignID := fs.String("campaign", env("DEFAULT_CAMPAIGN_ID", "cmp_7day_cover"), "campaign id")
		sessionID := fs.String("session-id", "", "metadata session_id")
		batchID := fs.String("batch-id", "", "metadata batch_id")
		limit := fs.Int("limit", 100, "maximum tasks to summarize")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		values := url.Values{}
		if trimmed := strings.TrimSpace(*sessionID); trimmed != "" {
			values.Set("session_id", trimmed)
		}
		if trimmed := strings.TrimSpace(*batchID); trimmed != "" {
			values.Set("batch_id", trimmed)
		}
		values.Set("limit", strconv.Itoa(*limit))
		path := fmt.Sprintf("/api/projects/%s/campaigns/%s/batch-progress?%s", *projectID, *campaignID, values.Encode())
		return request("GET", *apiURL, path, nil)
	case "manifest":
		fs := flag.NewFlagSet("vag batch manifest", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		campaignID := fs.String("campaign", env("DEFAULT_CAMPAIGN_ID", "cmp_7day_cover"), "campaign id")
		sessionID := fs.String("session-id", "", "metadata session_id")
		batchID := fs.String("batch-id", "", "metadata batch_id")
		storyID := fs.String("story-id", "", "metadata story_id")
		source := fs.String("source", "", "metadata source")
		status := fs.String("status", "", "task status filter")
		includeSetup := fs.Bool("include-setup", false, "include setup/reference tasks")
		limit := fs.Int("limit", 100, "maximum tasks to summarize")
		selectedOnly := fs.Bool("selected-only", true, "include only selected assets")
		includeRejected := fs.Bool("include-rejected", false, "include rejected assets when selected-only=false")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		path := buildBatchManifestPath(*projectID, *campaignID, batchManifestOptions{
			SessionID:       *sessionID,
			BatchID:         *batchID,
			StoryID:         *storyID,
			Source:          *source,
			Status:          *status,
			IncludeSetup:    *includeSetup,
			Limit:           *limit,
			SelectedOnly:    *selectedOnly,
			IncludeRejected: *includeRejected,
		})
		return request("GET", *apiURL, path, nil)
	default:
		return fmt.Errorf("unknown batch command %q", args[0])
	}
}

type batchManifestOptions struct {
	SessionID       string
	BatchID         string
	StoryID         string
	Source          string
	Status          string
	IncludeSetup    bool
	Limit           int
	SelectedOnly    bool
	IncludeRejected bool
}

func buildBatchManifestPath(projectID, campaignID string, options batchManifestOptions) string {
	values := url.Values{}
	if trimmed := strings.TrimSpace(options.SessionID); trimmed != "" {
		values.Set("session_id", trimmed)
	}
	if trimmed := strings.TrimSpace(options.BatchID); trimmed != "" {
		values.Set("batch_id", trimmed)
	}
	if trimmed := strings.TrimSpace(options.StoryID); trimmed != "" {
		values.Set("story_id", trimmed)
	}
	if trimmed := strings.TrimSpace(options.Source); trimmed != "" {
		values.Set("source", trimmed)
	}
	if trimmed := strings.TrimSpace(options.Status); trimmed != "" {
		values.Set("status", trimmed)
	}
	if options.IncludeSetup {
		values.Set("include_setup", "true")
	}
	if options.Limit > 0 {
		values.Set("limit", strconv.Itoa(options.Limit))
	}
	values.Set("selected_only", strconv.FormatBool(options.SelectedOnly))
	if options.IncludeRejected {
		values.Set("include_rejected", "true")
	}
	path := fmt.Sprintf("/api/projects/%s/campaigns/%s/batch-manifest", projectID, campaignID)
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return path
}

func taskCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: vag task create|get")
	}
	switch args[0] {
	case "create":
		fs := flag.NewFlagSet("vag task create", flag.ExitOnError)
		file := fs.String("file", "", "task JSON file")
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		workspaceID := fs.String("workspace", env("DEFAULT_WORKSPACE_ID", "ws_default"), "workspace id")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		campaignID := fs.String("campaign", env("DEFAULT_CAMPAIGN_ID", "cmp_7day_cover"), "campaign id")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *file == "" {
			return fmt.Errorf("--file is required")
		}
		body, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		path := fmt.Sprintf("/api/workspaces/%s/projects/%s/campaigns/%s/tasks", *workspaceID, *projectID, *campaignID)
		return request("POST", *apiURL, path, body)
	case "get":
		fs := flag.NewFlagSet("vag task get", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("usage: vag task get <task_id>")
		}
		return request("GET", *apiURL, "/api/tasks/"+fs.Arg(0), nil)
	case "attempts":
		fs := flag.NewFlagSet("vag task attempts", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("usage: vag task attempts <task_id>")
		}
		return request("GET", *apiURL, "/api/tasks/"+fs.Arg(0)+"/attempts", nil)
	default:
		return fmt.Errorf("unknown task command %q", args[0])
	}
}

func assetCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: vag asset approve|reject|get|list")
	}
	switch args[0] {
	case "approve", "reject":
		fs := flag.NewFlagSet("vag asset "+args[0], flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("usage: vag asset %s <asset_id>", args[0])
		}
		return request("POST", *apiURL, "/api/assets/"+fs.Arg(0)+"/"+args[0], nil)
	case "get":
		fs := flag.NewFlagSet("vag asset get", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("usage: vag asset get <asset_id>")
		}
		return request("GET", *apiURL, "/api/assets/"+fs.Arg(0), nil)
	case "list":
		fs := flag.NewFlagSet("vag asset list", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		campaignID := fs.String("campaign", env("DEFAULT_CAMPAIGN_ID", "cmp_7day_cover"), "campaign id")
		limit := fs.Int("limit", domain.DefaultAssetListLimit, "maximum assets to return")
		offset := fs.Int("offset", 0, "asset list offset")
		status := fs.String("status", "", "asset status filter")
		provider := fs.String("provider", "", "provider filter")
		model := fs.String("model", "", "model filter")
		source := fs.String("source", "", "metadata_json.source filter")
		sessionID := fs.String("session", "", "metadata_json.session_id filter")
		batchID := fs.String("batch", "", "metadata_json.batch_id filter")
		keyword := fs.String("keyword", "", "keyword filter for asset/task/prompt")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		path := fmt.Sprintf("/api/projects/%s/campaigns/%s/assets", *projectID, *campaignID)
		query := []string{}
		addQuery := func(key string, value string) {
			value = strings.TrimSpace(value)
			if value != "" {
				query = append(query, key+"="+urlQueryEscape(value))
			}
		}
		if *limit > 0 {
			addQuery("limit", strconv.Itoa(*limit))
		}
		if *offset > 0 {
			addQuery("offset", strconv.Itoa(*offset))
		}
		addQuery("status", *status)
		addQuery("provider", *provider)
		addQuery("model", *model)
		addQuery("source", *source)
		addQuery("session_id", *sessionID)
		addQuery("batch_id", *batchID)
		addQuery("keyword", *keyword)
		if len(query) > 0 {
			path += "?" + strings.Join(query, "&")
		}
		return request("GET", *apiURL, path, nil)
	default:
		return fmt.Errorf("unknown asset command %q", args[0])
	}
}

func auditCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: vag audit list")
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("vag audit list", flag.ExitOnError)
		limit := fs.Int("limit", 50, "maximum audit events to return")
		workspaceID := fs.String("workspace", "", "workspace id filter")
		projectID := fs.String("project", "", "project id filter")
		campaignID := fs.String("campaign", "", "campaign id filter")
		taskID := fs.String("task", "", "task id filter")
		assetID := fs.String("asset", "", "asset id filter")
		inputFileID := fs.String("input-file", "", "input file id filter")
		action := fs.String("action", "", "action filter")
		actor := fs.String("actor", "", "actor filter")
		statusCode := fs.Int("status", 0, "status code filter")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		cfg := config.Load()
		localStorage := storage.NewLocalStorage(cfg.StorageRoot, cfg.ThumbnailMaxWidth, cfg.ThumbnailMaxHeight)
		events, err := localStorage.ListHTTPAuditEvents(context.Background(), domain.HTTPAuditQuery{
			Limit:       *limit,
			WorkspaceID: strings.TrimSpace(*workspaceID),
			ProjectID:   strings.TrimSpace(*projectID),
			CampaignID:  strings.TrimSpace(*campaignID),
			TaskID:      strings.TrimSpace(*taskID),
			AssetID:     strings.TrimSpace(*assetID),
			InputFileID: strings.TrimSpace(*inputFileID),
			Action:      strings.TrimSpace(*action),
			Actor:       strings.TrimSpace(*actor),
			StatusCode:  *statusCode,
		})
		if err != nil {
			return err
		}
		return writeJSON(domain.HTTPAuditListResponse{Events: events})
	default:
		return fmt.Errorf("unknown audit command %q", args[0])
	}
}

type benchmarkTaskSummary struct {
	TaskID               string  `json:"task_id"`
	Status               string  `json:"status"`
	AssetCount           int     `json:"asset_count"`
	Attempts             int     `json:"attempts"`
	Retries              int     `json:"retries"`
	ProviderRequestCount int     `json:"provider_request_count,omitempty"`
	QueueWaitMs          *int64  `json:"queue_wait_ms,omitempty"`
	ProviderFirstByteMs  *int64  `json:"provider_first_byte_ms,omitempty"`
	ProviderTotalMs      *int64  `json:"provider_total_ms,omitempty"`
	ResponseDownloadMs   *int64  `json:"response_download_ms,omitempty"`
	ResponseBytes        int64   `json:"response_bytes,omitempty"`
	StoreMs              *int64  `json:"store_ms,omitempty"`
	ThumbnailMs          *int64  `json:"thumbnail_ms,omitempty"`
	ErrorStage           string  `json:"error_stage,omitempty"`
	TotalMs              int64   `json:"total_ms"`
	ErrorCode            *string `json:"error_code,omitempty"`
	ErrorMessage         *string `json:"error_message,omitempty"`
}

type benchmarkMetrics struct {
	TaskCount               int            `json:"task_count"`
	CompletedCount          int            `json:"completed_count"`
	PartiallyCompletedCount int            `json:"partially_completed_count"`
	FailedCount             int            `json:"failed_count"`
	SuccessRate             float64        `json:"success_rate"`
	TotalWallMs             int64          `json:"total_wall_ms"`
	AvgTaskMs               int64          `json:"avg_task_ms"`
	P50TaskMs               int64          `json:"p50_task_ms"`
	P95TaskMs               int64          `json:"p95_task_ms"`
	AvgQueueWaitMs          int64          `json:"avg_queue_wait_ms"`
	P95QueueWaitMs          int64          `json:"p95_queue_wait_ms"`
	AvgProviderLatencyMs    int64          `json:"avg_provider_latency_ms"`
	P50ProviderLatencyMs    int64          `json:"p50_provider_latency_ms"`
	P95ProviderLatencyMs    int64          `json:"p95_provider_latency_ms"`
	AvgProviderFirstByteMs  int64          `json:"avg_provider_first_byte_ms"`
	P95ProviderFirstByteMs  int64          `json:"p95_provider_first_byte_ms"`
	AvgResponseDownloadMs   int64          `json:"avg_response_download_ms"`
	P95ResponseDownloadMs   int64          `json:"p95_response_download_ms"`
	AvgStoreMs              int64          `json:"avg_store_ms"`
	P95StoreMs              int64          `json:"p95_store_ms"`
	AvgThumbnailMs          int64          `json:"avg_thumbnail_ms"`
	P95ThumbnailMs          int64          `json:"p95_thumbnail_ms"`
	TotalAttempts           int            `json:"total_attempts"`
	RetryCount              int            `json:"retry_count"`
	ScheduledRetryCount     int            `json:"scheduled_retry_count"`
	TimeoutCount            int            `json:"timeout_count"`
	FailureReasons          map[string]int `json:"failure_reasons"`
	ErrorStages             map[string]int `json:"error_stages"`
}

type benchmarkRequestShape struct {
	APIMode              string `json:"api_mode"`
	Endpoint             string `json:"endpoint,omitempty"`
	RequestMode          string `json:"request_mode"`
	Model                string `json:"model,omitempty"`
	Stream               bool   `json:"stream"`
	PartialImages        int    `json:"partial_images"`
	ResponseFormat       string `json:"response_format"`
	N                    int    `json:"n"`
	SplitCounts          []int  `json:"split_counts,omitempty"`
	Size                 string `json:"size,omitempty"`
	Quality              string `json:"quality,omitempty"`
	OutputFormat         string `json:"output_format,omitempty"`
	Moderation           string `json:"moderation,omitempty"`
	TimeoutSeconds       int    `json:"timeout_seconds,omitempty"`
	ProviderRequestCount int    `json:"provider_request_count"`
	ResponseBytes        int64  `json:"response_bytes"`
}

type benchmarkResult struct {
	RunID                          string                 `json:"run_id"`
	Source                         string                 `json:"source"`
	Provider                       string                 `json:"provider"`
	RequestedCount                 int                    `json:"requested_count"`
	ConcurrencyLabel               string                 `json:"concurrency_label"`
	WorkerConcurrencyEnv           string                 `json:"worker_concurrency_env,omitempty"`
	OpenAICompatibleMaxConcurrency int                    `json:"openai_compatible_max_concurrency"`
	FalMaxConcurrency              int                    `json:"fal_max_concurrency"`
	ProviderTimeoutSeconds         int                    `json:"provider_timeout_seconds"`
	OpenAICompatibleConnectTimeout int                    `json:"openai_compatible_connect_timeout_seconds"`
	OpenAICompatibleHeaderTimeout  int                    `json:"openai_compatible_response_header_timeout_seconds"`
	OpenAICompatibleTotalTimeout   int                    `json:"openai_compatible_total_timeout_seconds"`
	Recommendations                []string               `json:"recommendations,omitempty"`
	RequestShape                   benchmarkRequestShape  `json:"request_shape"`
	CreatedAt                      time.Time              `json:"created_at"`
	FinishedAt                     time.Time              `json:"finished_at"`
	Metrics                        benchmarkMetrics       `json:"metrics"`
	Tasks                          []benchmarkTaskSummary `json:"tasks"`
}

func benchmarkCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: vag benchmark image-generation")
	}
	switch args[0] {
	case "image-generation":
		return benchmarkImageGenerationCmd(args[1:])
	default:
		return fmt.Errorf("unknown benchmark command %q", args[0])
	}
}

func benchmarkImageGenerationCmd(args []string) error {
	fs := flag.NewFlagSet("vag benchmark image-generation", flag.ExitOnError)
	workspaceID := fs.String("workspace", env("DEFAULT_WORKSPACE_ID", "ws_default"), "workspace id")
	projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
	campaignID := fs.String("campaign", env("DEFAULT_CAMPAIGN_ID", "cmp_7day_cover"), "campaign id")
	providerID := fs.String("provider", env("DEFAULT_PROVIDER", "mock"), "provider id")
	taskCount := fs.Int("tasks", 8, "number of tasks to create")
	requestedCount := fs.Int("requested-count", 1, "requested images per task")
	concurrencyLabel := fs.String("concurrency-label", "", "human label for the worker/provider concurrency being tested")
	runID := fs.String("run-id", "", "benchmark run id")
	timeout := fs.Duration("timeout", 30*time.Minute, "maximum benchmark duration")
	pollInterval := fs.Duration("poll-interval", 2*time.Second, "task polling interval")
	prompt := fs.String("prompt", "Agent ImageFlow benchmark image", "prompt prefix")
	mockDelayMs := fs.Int("mock-delay-ms", 0, "mock provider artificial latency per task in milliseconds")
	model := fs.String("model", "", "model override for benchmark tasks")
	apiMode := fs.String("api-mode", "", "openai-compatible api mode override for benchmark tasks: images or responses")
	stream := fs.String("stream", "", "openai-compatible stream override for benchmark tasks: true or false")
	partialImages := fs.Int("partial-images", -1, "openai-compatible partial_images override, 0-3")
	preferredResponseFormat := fs.String("preferred-response-format", "", "openai-compatible response format override: url or b64_json")
	maxN := fs.Int("max-n", 0, "provider max_n override for benchmark tasks")
	timeoutSeconds := fs.Int("timeout-seconds", 0, "provider timeout_seconds override for benchmark tasks")
	allowPaidProvider := fs.Bool("allow-paid-provider", false, "allow non-mock provider benchmark that may incur cost")
	if err := fs.Parse(args); err != nil {
		return err
	}

	providerName := strings.TrimSpace(*providerID)
	if providerName == "" {
		providerName = provider.MockProviderID
	}
	if *taskCount < 1 {
		return fmt.Errorf("--tasks must be >= 1")
	}
	if providerName != provider.MockProviderID && *taskCount > 8 {
		return fmt.Errorf("real provider benchmark is capped at 8 tasks per run")
	}
	if providerName == provider.MockProviderID && *taskCount > 32 {
		return fmt.Errorf("mock benchmark is capped at 32 tasks per run")
	}
	if *requestedCount < 1 || *requestedCount > 10 {
		return fmt.Errorf("--requested-count must be between 1 and 10")
	}
	if *mockDelayMs < 0 || *mockDelayMs > 10_000 {
		return fmt.Errorf("--mock-delay-ms must be between 0 and 10000")
	}
	if *partialImages < -1 || *partialImages > 3 {
		return fmt.Errorf("--partial-images must be between 0 and 3")
	}
	if *maxN < 0 || *maxN > 10 {
		return fmt.Errorf("--max-n must be between 0 and 10")
	}
	if *timeoutSeconds < 0 {
		return fmt.Errorf("--timeout-seconds must be >= 0")
	}
	if providerName != provider.MockProviderID && !*allowPaidProvider {
		return fmt.Errorf("provider %q may incur real API cost; rerun with --allow-paid-provider after confirming the small sample size", providerName)
	}
	if *timeout <= 0 {
		return fmt.Errorf("--timeout must be positive")
	}
	if *pollInterval <= 0 {
		return fmt.Errorf("--poll-interval must be positive")
	}

	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	service, cleanup, err := newRepairService(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	id := strings.TrimSpace(*runID)
	if id == "" {
		id = domain.NewID("bench")
	}
	label := strings.TrimSpace(*concurrencyLabel)
	if label == "" {
		label = "worker-" + env("WORKER_CONCURRENCY", "unknown")
	}
	scope := domain.Scope{
		WorkspaceID: strings.TrimSpace(*workspaceID),
		ProjectID:   strings.TrimSpace(*projectID),
		CampaignID:  strings.TrimSpace(*campaignID),
	}
	generationConfigMap := map[string]any{}
	if providerName == provider.MockProviderID && *mockDelayMs > 0 {
		generationConfigMap["mock_delay_ms"] = *mockDelayMs
	}
	if trimmed := strings.TrimSpace(*model); trimmed != "" {
		generationConfigMap["model"] = trimmed
	}
	if trimmed := strings.TrimSpace(*apiMode); trimmed != "" {
		if trimmed != "images" && trimmed != "responses" {
			return fmt.Errorf("--api-mode must be images or responses")
		}
		generationConfigMap["api_mode"] = trimmed
	}
	if parsed, ok, err := parseOptionalBoolFlag(*stream, "--stream"); err != nil {
		return err
	} else if ok {
		generationConfigMap["stream"] = parsed
	}
	if *partialImages >= 0 {
		generationConfigMap["partial_images"] = *partialImages
	}
	if trimmed := strings.TrimSpace(*preferredResponseFormat); trimmed != "" {
		if trimmed != "url" && trimmed != "b64_json" {
			return fmt.Errorf("--preferred-response-format must be url or b64_json")
		}
		generationConfigMap["preferred_response_format"] = trimmed
	}
	if *maxN > 0 {
		generationConfigMap["max_n"] = *maxN
	}
	if *timeoutSeconds > 0 {
		generationConfigMap["timeout_seconds"] = *timeoutSeconds
	}
	var generationConfig json.RawMessage
	if len(generationConfigMap) > 0 {
		raw, err := json.Marshal(generationConfigMap)
		if err != nil {
			return err
		}
		generationConfig = raw
	}
	started := time.Now().UTC()
	createdTasks := make([]domain.TaskResponse, 0, *taskCount)
	for i := 0; i < *taskCount; i++ {
		metadata, err := json.Marshal(map[string]any{
			"source":                    "benchmark",
			"source_agent":              "vag",
			"session_id":                id,
			"run_id":                    id,
			"batch_id":                  id,
			"concurrency_label":         label,
			"benchmark_provider":        providerName,
			"benchmark_requested_count": *requestedCount,
			"benchmark_task_index":      i + 1,
			"mock_delay_ms":             *mockDelayMs,
		})
		if err != nil {
			return err
		}
		response, err := service.CreateTask(ctx, scope, domain.CreateTaskRequest{
			IdempotencyKey:   fmt.Sprintf("%s-%03d", id, i+1),
			Title:            fmt.Sprintf("Benchmark %s %03d", id, i+1),
			Purpose:          "benchmark",
			Prompt:           fmt.Sprintf("%s #%03d run %s", strings.TrimSpace(*prompt), i+1, id),
			AspectRatio:      "1:1",
			OutputFormat:     "png",
			RequestedCount:   *requestedCount,
			Provider:         providerName,
			SelectionMode:    domain.SelectionManualOptional,
			ReviewRequired:   false,
			GenerationConfig: generationConfig,
			MetadataJSON:     metadata,
		})
		if err != nil {
			return err
		}
		createdTasks = append(createdTasks, response)
	}

	finalTasks, err := waitForBenchmarkTasks(ctx, service, createdTasks, *pollInterval)
	if err != nil {
		return err
	}
	finished := time.Now().UTC()
	result, err := summarizeBenchmark(ctx, service, cfg, id, providerName, *requestedCount, label, started, finished, finalTasks)
	if err != nil {
		return err
	}
	return writeJSON(result)
}

func waitForBenchmarkTasks(ctx context.Context, service *app.Service, created []domain.TaskResponse, pollInterval time.Duration) ([]domain.TaskResponse, error) {
	finalByID := map[string]domain.TaskResponse{}
	for {
		for _, item := range created {
			if _, ok := finalByID[item.ID]; ok {
				continue
			}
			current, err := service.GetTask(ctx, item.ID)
			if err != nil {
				return nil, err
			}
			if isTerminalTaskStatus(current.Status) {
				finalByID[item.ID] = current
			}
		}
		if len(finalByID) == len(created) {
			break
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("benchmark timed out waiting for tasks: %w", ctx.Err())
		case <-time.After(pollInterval):
		}
	}
	finalTasks := make([]domain.TaskResponse, 0, len(created))
	for _, item := range created {
		finalTasks = append(finalTasks, finalByID[item.ID])
	}
	return finalTasks, nil
}

func summarizeBenchmark(ctx context.Context, service *app.Service, cfg config.Config, runID, providerName string, requestedCount int, concurrencyLabel string, started, finished time.Time, finalTasks []domain.TaskResponse) (benchmarkResult, error) {
	metrics := benchmarkMetrics{
		TaskCount:      len(finalTasks),
		FailureReasons: map[string]int{},
		ErrorStages:    map[string]int{},
	}
	requestShape := inferBenchmarkRequestShape(providerName, requestedCount, cfg, finalTasks)
	taskSummaries := make([]benchmarkTaskSummary, 0, len(finalTasks))
	taskDurations := []int64{}
	queueWaits := []int64{}
	providerLatencies := []int64{}
	firstByteLatencies := []int64{}
	responseDownloads := []int64{}
	storeDurations := []int64{}
	thumbnailDurations := []int64{}

	for _, task := range finalTasks {
		attemptResponse, err := service.ListTaskAttempts(ctx, task.ID)
		if err != nil {
			return benchmarkResult{}, err
		}
		attempts := attemptResponse.Attempts
		totalMs := task.UpdatedAt.Sub(task.CreatedAt).Milliseconds()
		if totalMs < 0 {
			totalMs = 0
		}
		taskDurations = append(taskDurations, totalMs)
		retries := 0
		if len(attempts) > 1 {
			retries = len(attempts) - 1
		}
		metrics.TotalAttempts += len(attempts)
		metrics.RetryCount += retries
		var queueWaitMs *int64
		var firstByteMs *int64
		var providerTotalMs *int64
		var responseDownloadMs *int64
		var storeMs *int64
		var thumbnailMs *int64
		var errorStage string
		providerRequestCount := 0
		var responseBytes int64
		if len(attempts) > 0 {
			wait := attempts[0].StartedAt.Sub(task.CreatedAt).Milliseconds()
			if attempts[0].QueueWaitMs != nil {
				wait = int64(*attempts[0].QueueWaitMs)
			}
			if wait < 0 {
				wait = 0
			}
			queueWaitMs = &wait
			queueWaits = append(queueWaits, wait)
		}
		for _, attempt := range attempts {
			requestCount := benchmarkProviderRequestCount(providerName, requestShape, attempt)
			providerRequestCount += requestCount
			responseBytes += attempt.ResponseBytes
			if attempt.ProviderTotalMs != nil {
				value := int64(*attempt.ProviderTotalMs)
				providerLatencies = append(providerLatencies, value)
				if providerTotalMs == nil || value > *providerTotalMs {
					providerTotalMs = &value
				}
			} else if attempt.LatencyMs != nil {
				value := int64(*attempt.LatencyMs)
				providerLatencies = append(providerLatencies, value)
				if providerTotalMs == nil || value > *providerTotalMs {
					providerTotalMs = &value
				}
			}
			if attempt.ProviderFirstByteMs != nil {
				value := int64(*attempt.ProviderFirstByteMs)
				firstByteLatencies = append(firstByteLatencies, value)
				if firstByteMs == nil || value > *firstByteMs {
					firstByteMs = &value
				}
			}
			if attempt.ResponseDownloadMs != nil {
				value := int64(*attempt.ResponseDownloadMs)
				responseDownloads = append(responseDownloads, value)
				if responseDownloadMs == nil || value > *responseDownloadMs {
					responseDownloadMs = &value
				}
			}
			if attempt.StoreMs != nil {
				value := int64(*attempt.StoreMs)
				storeDurations = append(storeDurations, value)
				if storeMs == nil || value > *storeMs {
					storeMs = &value
				}
			}
			if attempt.ThumbnailMs != nil {
				value := int64(*attempt.ThumbnailMs)
				thumbnailDurations = append(thumbnailDurations, value)
				if thumbnailMs == nil || value > *thumbnailMs {
					thumbnailMs = &value
				}
			}
			if strings.TrimSpace(attempt.ErrorStage) != "" {
				errorStage = strings.TrimSpace(attempt.ErrorStage)
				metrics.ErrorStages[errorStage]++
			}
			if attempt.RetryAfter != nil {
				metrics.ScheduledRetryCount++
			}
			if isTimeoutAttempt(attempt, cfg.ProviderTimeoutSeconds) {
				metrics.TimeoutCount++
			}
		}
		requestShape.ProviderRequestCount += providerRequestCount
		requestShape.ResponseBytes += responseBytes
		switch task.Status {
		case domain.TaskCompleted:
			metrics.CompletedCount++
		case domain.TaskPartiallyCompleted:
			metrics.PartiallyCompletedCount++
		case domain.TaskFailed, domain.TaskEnqueueFailed:
			metrics.FailedCount++
		}
		if task.ErrorCode != nil && strings.TrimSpace(*task.ErrorCode) != "" {
			metrics.FailureReasons[strings.TrimSpace(*task.ErrorCode)]++
		} else if task.Status == domain.TaskFailed || task.Status == domain.TaskEnqueueFailed {
			metrics.FailureReasons[task.Status]++
		}
		taskSummaries = append(taskSummaries, benchmarkTaskSummary{
			TaskID:               task.ID,
			Status:               task.Status,
			AssetCount:           len(task.Assets),
			Attempts:             len(attempts),
			Retries:              retries,
			ProviderRequestCount: providerRequestCount,
			QueueWaitMs:          queueWaitMs,
			ProviderFirstByteMs:  firstByteMs,
			ProviderTotalMs:      providerTotalMs,
			ResponseDownloadMs:   responseDownloadMs,
			ResponseBytes:        responseBytes,
			StoreMs:              storeMs,
			ThumbnailMs:          thumbnailMs,
			ErrorStage:           errorStage,
			TotalMs:              totalMs,
			ErrorCode:            task.ErrorCode,
			ErrorMessage:         task.ErrorMessage,
		})
	}

	if metrics.TaskCount > 0 {
		metrics.SuccessRate = float64(metrics.CompletedCount+metrics.PartiallyCompletedCount) / float64(metrics.TaskCount)
	}
	metrics.TotalWallMs = finished.Sub(started).Milliseconds()
	metrics.AvgTaskMs = avgInt64(taskDurations)
	metrics.P50TaskMs = percentileInt64(taskDurations, 50)
	metrics.P95TaskMs = percentileInt64(taskDurations, 95)
	metrics.AvgQueueWaitMs = avgInt64(queueWaits)
	metrics.P95QueueWaitMs = percentileInt64(queueWaits, 95)
	metrics.AvgProviderLatencyMs = avgInt64(providerLatencies)
	metrics.P50ProviderLatencyMs = percentileInt64(providerLatencies, 50)
	metrics.P95ProviderLatencyMs = percentileInt64(providerLatencies, 95)
	metrics.AvgProviderFirstByteMs = avgInt64(firstByteLatencies)
	metrics.P95ProviderFirstByteMs = percentileInt64(firstByteLatencies, 95)
	metrics.AvgResponseDownloadMs = avgInt64(responseDownloads)
	metrics.P95ResponseDownloadMs = percentileInt64(responseDownloads, 95)
	metrics.AvgStoreMs = avgInt64(storeDurations)
	metrics.P95StoreMs = percentileInt64(storeDurations, 95)
	metrics.AvgThumbnailMs = avgInt64(thumbnailDurations)
	metrics.P95ThumbnailMs = percentileInt64(thumbnailDurations, 95)

	return benchmarkResult{
		RunID:                          runID,
		Source:                         "benchmark",
		Provider:                       providerName,
		RequestedCount:                 requestedCount,
		ConcurrencyLabel:               concurrencyLabel,
		WorkerConcurrencyEnv:           strings.TrimSpace(os.Getenv("WORKER_CONCURRENCY")),
		OpenAICompatibleMaxConcurrency: cfg.OpenAICompatibleMaxConcurrency,
		FalMaxConcurrency:              cfg.FalMaxConcurrency,
		ProviderTimeoutSeconds:         cfg.ProviderTimeoutSeconds,
		OpenAICompatibleConnectTimeout: cfg.OpenAICompatibleConnectTimeout,
		OpenAICompatibleHeaderTimeout:  cfg.OpenAICompatibleHeaderTimeout,
		OpenAICompatibleTotalTimeout:   cfg.OpenAICompatibleTotalTimeout,
		Recommendations:                benchmarkRecommendations(providerName, cfg, metrics),
		RequestShape:                   requestShape,
		CreatedAt:                      started,
		FinishedAt:                     finished,
		Metrics:                        metrics,
		Tasks:                          taskSummaries,
	}, nil
}

func inferBenchmarkRequestShape(providerName string, requestedCount int, cfg config.Config, finalTasks []domain.TaskResponse) benchmarkRequestShape {
	shape := benchmarkRequestShape{
		APIMode:        providerName,
		RequestMode:    providerName,
		ResponseFormat: "provider_default",
		N:              requestedCount,
		TimeoutSeconds: cfg.ProviderTimeoutSeconds,
	}
	if len(finalTasks) == 0 {
		return shape
	}
	task := finalTasks[0].Task
	shape.Size = benchmarkSizeForAspectRatio(task.AspectRatio)
	shape.OutputFormat = firstNonEmptyString(task.OutputFormat, "png")
	shape.Quality, shape.Moderation = benchmarkQualityAndModeration(task.StructuredInputJSON)
	if providerName != provider.OpenAICompatibleProviderID {
		return shape
	}

	maxN := benchmarkProviderMaxN(task, providerName)
	apiMode := benchmarkAPIMode(task, providerName)
	shape.APIMode = apiMode
	shape.Endpoint = "/images/generations"
	if apiMode == "responses" {
		shape.Endpoint = "/responses"
	}
	shape.Model = benchmarkModel(task, providerName, apiMode, cfg)
	shape.Stream = benchmarkStream(task, providerName, apiMode)
	shape.PartialImages = benchmarkPartialImages(task, providerName, shape.Stream)
	shape.SplitCounts = benchmarkSplitCounts(requestedCount, maxN)
	if len(shape.SplitCounts) > 0 {
		shape.N = shape.SplitCounts[0]
	}
	shape.TimeoutSeconds = firstPositiveInt(cfg.OpenAICompatibleTotalTimeout, cfg.ProviderTimeoutSeconds)
	if configuredTimeout := benchmarkTimeoutSeconds(task, providerName); configuredTimeout > 0 {
		shape.TimeoutSeconds = configuredTimeout
	}
	if apiMode == "responses" {
		shape.RequestMode = provider.OpenAICompatibleRequestModeResponsesStream
		shape.ResponseFormat = "omitted"
	} else if shape.Stream {
		shape.RequestMode = provider.OpenAICompatibleRequestModeImagesStream
		shape.ResponseFormat = "omitted"
		if benchmarkPreferredResponseFormat(task, providerName) == "b64_json" {
			shape.ResponseFormat = "b64_json"
		}
	} else if benchmarkPreferredResponseFormat(task, providerName) == "b64_json" {
		shape.RequestMode = provider.OpenAICompatibleRequestModeImagesSyncB64
		shape.ResponseFormat = "b64_json"
	} else {
		shape.RequestMode = provider.OpenAICompatibleRequestModeImagesSyncURL
		shape.ResponseFormat = "omitted"
	}
	return shape
}

func benchmarkProviderMaxN(task domain.Task, providerName string) int {
	maxN := 4
	if providerName == provider.OpenAICompatibleProviderID {
		maxN = 1
	}
	var input struct {
		GenerationConfig json.RawMessage               `json:"generation_config"`
		ProviderProfile  domain.ProjectProviderProfile `json:"provider_profile"`
	}
	if len(task.StructuredInputJSON) > 0 && json.Unmarshal(task.StructuredInputJSON, &input) == nil {
		if input.ProviderProfile.Enabled &&
			strings.TrimSpace(input.ProviderProfile.Provider) == providerName &&
			input.ProviderProfile.MaxN > 0 {
			maxN = input.ProviderProfile.MaxN
		}
		if value, ok := benchmarkGenerationConfigInt(input.GenerationConfig, "max_n", 1, 10); ok {
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

func benchmarkPreferredResponseFormat(task domain.Task, providerName string) string {
	var input struct {
		GenerationConfig json.RawMessage               `json:"generation_config"`
		ProviderProfile  domain.ProjectProviderProfile `json:"provider_profile"`
	}
	if len(task.StructuredInputJSON) > 0 && json.Unmarshal(task.StructuredInputJSON, &input) == nil {
		if input.ProviderProfile.Enabled &&
			strings.TrimSpace(input.ProviderProfile.Provider) == providerName &&
			strings.TrimSpace(input.ProviderProfile.PreferredResponseFormat) == "b64_json" {
			return "b64_json"
		}
		if value := benchmarkGenerationConfigString(input.GenerationConfig, "preferred_response_format"); value == "b64_json" || value == "url" {
			return value
		}
	}
	return "url"
}

func benchmarkAPIMode(task domain.Task, providerName string) string {
	var input struct {
		GenerationConfig json.RawMessage               `json:"generation_config"`
		ProviderProfile  domain.ProjectProviderProfile `json:"provider_profile"`
	}
	if len(task.StructuredInputJSON) > 0 && json.Unmarshal(task.StructuredInputJSON, &input) == nil {
		mode := "images"
		if input.ProviderProfile.Enabled &&
			strings.TrimSpace(input.ProviderProfile.Provider) == providerName &&
			strings.TrimSpace(input.ProviderProfile.APIMode) == "responses" {
			mode = "responses"
		}
		if value := benchmarkGenerationConfigString(input.GenerationConfig, "api_mode"); value == "images" || value == "responses" {
			mode = value
		}
		return mode
	}
	return "images"
}

func benchmarkModel(task domain.Task, providerName string, apiMode string, cfg config.Config) string {
	model := ""
	var input struct {
		GenerationConfig json.RawMessage               `json:"generation_config"`
		ProviderProfile  domain.ProjectProviderProfile `json:"provider_profile"`
	}
	if len(task.StructuredInputJSON) > 0 && json.Unmarshal(task.StructuredInputJSON, &input) == nil {
		if input.ProviderProfile.Enabled &&
			strings.TrimSpace(input.ProviderProfile.Provider) == providerName {
			model = strings.TrimSpace(input.ProviderProfile.Model)
		}
		if value := benchmarkGenerationConfigString(input.GenerationConfig, "model"); value != "" {
			model = value
		}
	}
	if model != "" {
		return model
	}
	if providerName == provider.OpenAICompatibleProviderID {
		if configured := strings.TrimSpace(cfg.OpenAICompatibleModel); configured != "" {
			return configured
		}
		if apiMode == "responses" {
			return "gpt-5.5"
		}
		return "gpt-image-2"
	}
	return ""
}

func benchmarkStream(task domain.Task, providerName string, apiMode string) bool {
	var input struct {
		GenerationConfig json.RawMessage               `json:"generation_config"`
		ProviderProfile  domain.ProjectProviderProfile `json:"provider_profile"`
	}
	stream := apiMode == "responses"
	if len(task.StructuredInputJSON) > 0 && json.Unmarshal(task.StructuredInputJSON, &input) == nil {
		if input.ProviderProfile.Enabled &&
			strings.TrimSpace(input.ProviderProfile.Provider) == providerName &&
			input.ProviderProfile.Stream != nil {
			stream = *input.ProviderProfile.Stream
		}
		if value, ok := benchmarkGenerationConfigBool(input.GenerationConfig, "stream"); ok {
			stream = value
		}
	}
	return stream
}

func benchmarkPartialImages(task domain.Task, providerName string, stream bool) int {
	if !stream {
		return 0
	}
	partialImages := 1
	var input struct {
		GenerationConfig json.RawMessage               `json:"generation_config"`
		ProviderProfile  domain.ProjectProviderProfile `json:"provider_profile"`
	}
	if len(task.StructuredInputJSON) > 0 && json.Unmarshal(task.StructuredInputJSON, &input) == nil {
		if input.ProviderProfile.Enabled &&
			strings.TrimSpace(input.ProviderProfile.Provider) == providerName &&
			input.ProviderProfile.PartialImages != nil {
			partialImages = *input.ProviderProfile.PartialImages
		}
		if value, ok := benchmarkGenerationConfigInt(input.GenerationConfig, "partial_images", 0, 3); ok {
			partialImages = value
		}
	}
	if partialImages < 0 {
		return 0
	}
	if partialImages > 3 {
		return 3
	}
	return partialImages
}

func benchmarkTimeoutSeconds(task domain.Task, providerName string) int {
	var input struct {
		GenerationConfig json.RawMessage               `json:"generation_config"`
		ProviderProfile  domain.ProjectProviderProfile `json:"provider_profile"`
	}
	if len(task.StructuredInputJSON) > 0 && json.Unmarshal(task.StructuredInputJSON, &input) == nil {
		timeout := 0
		if input.ProviderProfile.Enabled &&
			strings.TrimSpace(input.ProviderProfile.Provider) == providerName &&
			input.ProviderProfile.TimeoutSeconds > 0 {
			timeout = input.ProviderProfile.TimeoutSeconds
		}
		if value, ok := benchmarkGenerationConfigInt(input.GenerationConfig, "timeout_seconds", 1, 3600); ok {
			timeout = value
		}
		return timeout
	}
	return 0
}

func benchmarkQualityAndModeration(raw json.RawMessage) (string, string) {
	var input struct {
		GenerationConfig map[string]any `json:"generation_config"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &input) != nil {
		return "", ""
	}
	return benchmarkConfigString(input.GenerationConfig, "quality"), benchmarkConfigString(input.GenerationConfig, "moderation")
}

func benchmarkConfigString(config map[string]any, key string) string {
	if len(config) == 0 {
		return ""
	}
	value, ok := config[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func benchmarkGenerationConfigString(raw json.RawMessage, key string) string {
	if len(raw) == 0 {
		return ""
	}
	var config map[string]any
	if json.Unmarshal(raw, &config) != nil {
		return ""
	}
	return benchmarkConfigString(config, key)
}

func benchmarkGenerationConfigBool(raw json.RawMessage, key string) (bool, bool) {
	if len(raw) == 0 {
		return false, false
	}
	var config map[string]any
	if json.Unmarshal(raw, &config) != nil {
		return false, false
	}
	value, ok := config[key]
	if !ok || value == nil {
		return false, false
	}
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		parsed, ok, err := parseOptionalBoolFlag(typed, key)
		if err != nil {
			return false, false
		}
		return parsed, ok
	default:
		return false, false
	}
}

func benchmarkGenerationConfigInt(raw json.RawMessage, key string, minValue, maxValue int) (int, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var config map[string]any
	if json.Unmarshal(raw, &config) != nil {
		return 0, false
	}
	value, ok := config[key]
	if !ok || value == nil {
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

func benchmarkSplitCounts(requestedCount, maxPerRequest int) []int {
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

func providerRequestCountFromID(providerRequestID string) int {
	trimmed := strings.TrimSpace(providerRequestID)
	if trimmed == "" {
		return 0
	}
	count := 0
	for _, part := range strings.Split(trimmed, ",") {
		if strings.TrimSpace(part) != "" {
			count++
		}
	}
	return count
}

func benchmarkProviderRequestCount(providerName string, shape benchmarkRequestShape, attempt domain.TaskAttempt) int {
	count := providerRequestCountFromID(attempt.ProviderRequestID)
	if providerName == provider.OpenAICompatibleProviderID && attempt.ProviderTotalMs != nil && len(shape.SplitCounts) > count {
		count = len(shape.SplitCounts)
	}
	if count == 0 && attempt.ProviderTotalMs != nil {
		count = 1
	}
	return count
}

func benchmarkSizeForAspectRatio(aspectRatio string) string {
	switch aspectRatio {
	case "3:4", "9:16":
		return "1024x1536"
	case "4:3", "16:9":
		return "1536x1024"
	default:
		return "1024x1024"
	}
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstNonEmptyString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func parseOptionalBoolFlag(value, name string) (bool, bool, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return false, false, nil
	}
	switch trimmed {
	case "true", "1", "yes", "y":
		return true, true, nil
	case "false", "0", "no", "n":
		return false, true, nil
	default:
		return false, false, fmt.Errorf("%s must be true or false", name)
	}
}

func isTerminalTaskStatus(status string) bool {
	switch status {
	case domain.TaskCompleted, domain.TaskPartiallyCompleted, domain.TaskFailed, domain.TaskEnqueueFailed:
		return true
	default:
		return false
	}
}

func isTimeoutAttempt(attempt domain.TaskAttempt, timeoutSeconds int) bool {
	code := ""
	if attempt.ErrorCode != nil {
		code = strings.ToLower(*attempt.ErrorCode)
	}
	message := ""
	if attempt.ErrorMessage != nil {
		message = strings.ToLower(*attempt.ErrorMessage)
	}
	if strings.Contains(code, "timeout") || strings.Contains(message, "timeout") || strings.Contains(message, "deadline") {
		return true
	}
	if timeoutSeconds > 0 && attempt.Status == domain.AttemptFailed && attempt.LatencyMs != nil {
		return *attempt.LatencyMs >= (timeoutSeconds*1000 - 1000)
	}
	return false
}

func benchmarkRecommendations(providerName string, cfg config.Config, metrics benchmarkMetrics) []string {
	recommendations := []string{}
	if providerName != provider.MockProviderID {
		recommendations = append(recommendations, "真实 provider 建议按 provider cap=2 -> 3 -> 4 的顺序小样本复测，不要直接把 worker 并发等同于 provider 并发。")
	}
	if metrics.TimeoutCount > 0 {
		recommendations = append(recommendations, "本轮出现 timeout；优先查看 error_stage、provider_first_byte_ms 和 provider_total_ms，再决定调低 provider cap 或调高 timeout。")
	}
	if cfg.OpenAICompatibleMaxConcurrency > 4 {
		recommendations = append(recommendations, "openai-compatible cap 当前高于 4；真实 provider 生产环境建议先回到 2/3/4 找稳定档。")
	}
	if metrics.P95QueueWaitMs > metrics.P95ProviderLatencyMs && metrics.P95QueueWaitMs > 0 {
		recommendations = append(recommendations, "P95 queue wait 高于 provider latency；瓶颈可能在 worker 并发或队列消费。")
	}
	if metrics.P95ProviderFirstByteMs > 0 && metrics.P95ProviderFirstByteMs >= metrics.P95ProviderLatencyMs*8/10 {
		recommendations = append(recommendations, "大部分 provider 耗时发生在首字节前；优先排查 provider 排队、网关 header timeout 或并发 cap。")
	}
	return recommendations
}

func avgInt64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	var total int64
	for _, value := range values {
		total += value
	}
	return total / int64(len(values))
}

func percentileInt64(values []int64, percentile int) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	if percentile <= 0 {
		return sorted[0]
	}
	if percentile >= 100 {
		return sorted[len(sorted)-1]
	}
	index := (len(sorted) - 1) * percentile / 100
	return sorted[index]
}

func projectCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: vag project access|provider|context")
	}
	switch args[0] {
	case "access":
		return projectAccessCmd(args[1:])
	case "provider":
		return projectProviderCmd(args[1:])
	case "context":
		return projectContextCmd(args[1:])
	default:
		return fmt.Errorf("unknown project command %q", args[0])
	}
}

func projectContextCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: vag project context get|set")
	}
	switch args[0] {
	case "get":
		fs := flag.NewFlagSet("vag project context get", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		workspaceID := fs.String("workspace", env("DEFAULT_WORKSPACE_ID", "ws_default"), "workspace id")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		path := fmt.Sprintf("/api/workspaces/%s/projects/%s/visual-context", *workspaceID, *projectID)
		return request("GET", *apiURL, path, nil)
	case "set":
		fs := flag.NewFlagSet("vag project context set", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		workspaceID := fs.String("workspace", env("DEFAULT_WORKSPACE_ID", "ws_default"), "workspace id")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		file := fs.String("file", "", "project visual context JSON file")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if strings.TrimSpace(*file) == "" {
			return fmt.Errorf("--file is required")
		}
		body, err := os.ReadFile(*file)
		if err != nil {
			return err
		}
		wrapped, err := wrapProjectVisualContextPayload(body)
		if err != nil {
			return err
		}
		path := fmt.Sprintf("/api/workspaces/%s/projects/%s/visual-context", *workspaceID, *projectID)
		return request("POST", *apiURL, path, wrapped)
	default:
		return fmt.Errorf("unknown project context command %q", args[0])
	}
}

func wrapProjectVisualContextPayload(raw []byte) ([]byte, error) {
	if !json.Valid(raw) {
		return nil, fmt.Errorf("project visual context file must be valid JSON")
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("project visual context file must be a JSON object: %w", err)
	}
	if _, ok := payload["visual_context"]; ok {
		return raw, nil
	}
	wrapped, err := json.Marshal(map[string]json.RawMessage{
		"visual_context": json.RawMessage(raw),
	})
	if err != nil {
		return nil, err
	}
	return wrapped, nil
}

func projectProviderCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: vag project provider get|set")
	}
	switch args[0] {
	case "get":
		fs := flag.NewFlagSet("vag project provider get", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		workspaceID := fs.String("workspace", env("DEFAULT_WORKSPACE_ID", "ws_default"), "workspace id")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		path := fmt.Sprintf("/api/workspaces/%s/projects/%s/provider-profile", *workspaceID, *projectID)
		return request("GET", *apiURL, path, nil)
	case "set":
		fs := flag.NewFlagSet("vag project provider set", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		workspaceID := fs.String("workspace", env("DEFAULT_WORKSPACE_ID", "ws_default"), "workspace id")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		enabled := fs.Bool("enabled", true, "enable project provider profile")
		provider := fs.String("provider", "", "default provider id")
		model := fs.String("model", "", "default model or endpoint id")
		baseURL := fs.String("base-url", "", "non-sensitive provider base URL reference")
		generationConfig := fs.String("generation-config", "", "generation config JSON object")
		useQualityProfile := fs.Bool("use-quality-profile", false, "default to project quality profile when creating tasks")
		apiMode := fs.String("api-mode", "", "provider API mode metadata: images or responses")
		stream := fs.String("stream", "", "streaming preference metadata: true or false")
		partialImages := fs.Int("partial-images", -1, "partial image event count metadata, 0-3")
		maxN := fs.Int("max-n", 0, "maximum images per provider request")
		supportsURLResult := fs.Bool("supports-url-result", false, "record whether this provider profile supports URL result payloads")
		preferredResponseFormat := fs.String("preferred-response-format", "", "preferred response format metadata: b64_json or url")
		maxConcurrency := fs.Int("max-concurrency", 0, "recommended max provider concurrency metadata")
		timeoutSeconds := fs.Int("timeout-seconds", 0, "recommended provider timeout metadata")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		payload := domain.ProjectProviderProfile{
			Enabled:                  *enabled,
			Provider:                 strings.TrimSpace(*provider),
			Model:                    strings.TrimSpace(*model),
			BaseURL:                  strings.TrimSpace(*baseURL),
			UseProjectQualityProfile: *useQualityProfile,
			APIMode:                  strings.TrimSpace(*apiMode),
			MaxN:                     *maxN,
			SupportsURLResult:        *supportsURLResult,
			PreferredResponseFormat:  strings.TrimSpace(*preferredResponseFormat),
			MaxConcurrency:           *maxConcurrency,
			TimeoutSeconds:           *timeoutSeconds,
		}
		if parsed, ok, err := parseOptionalBoolFlag(*stream, "--stream"); err != nil {
			return err
		} else if ok {
			payload.Stream = &parsed
		}
		if *partialImages >= 0 {
			value := *partialImages
			payload.PartialImages = &value
		}
		if trimmed := strings.TrimSpace(*generationConfig); trimmed != "" {
			if !json.Valid([]byte(trimmed)) {
				return fmt.Errorf("--generation-config must be a JSON object")
			}
			payload.GenerationConfig = []byte(trimmed)
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		path := fmt.Sprintf("/api/workspaces/%s/projects/%s/provider-profile", *workspaceID, *projectID)
		return request("POST", *apiURL, path, body)
	default:
		return fmt.Errorf("unknown project provider command %q", args[0])
	}
}

func projectAccessCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: vag project access get|set|add-key|update-key|delete-key")
	}
	switch args[0] {
	case "get":
		fs := flag.NewFlagSet("vag project access get", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		workspaceID := fs.String("workspace", env("DEFAULT_WORKSPACE_ID", "ws_default"), "workspace id")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		path := fmt.Sprintf("/api/workspaces/%s/projects/%s/access-config", *workspaceID, *projectID)
		return request("GET", *apiURL, path, nil)
	case "set":
		fs := flag.NewFlagSet("vag project access set", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		workspaceID := fs.String("workspace", env("DEFAULT_WORKSPACE_ID", "ws_default"), "workspace id")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		enabled := fs.Bool("enabled", true, "enable project api key")
		name := fs.String("name", "", "project api key display name")
		key := fs.String("key", "", "new project api key; omit to keep the existing key when already enabled")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		payload := map[string]any{
			"api_key_enabled": *enabled,
		}
		if trimmed := strings.TrimSpace(*name); trimmed != "" {
			payload["api_key_name"] = trimmed
		}
		if trimmed := strings.TrimSpace(*key); trimmed != "" {
			payload["api_key"] = trimmed
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		path := fmt.Sprintf("/api/workspaces/%s/projects/%s/access-config", *workspaceID, *projectID)
		return request("POST", *apiURL, path, body)
	case "add-key":
		fs := flag.NewFlagSet("vag project access add-key", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		workspaceID := fs.String("workspace", env("DEFAULT_WORKSPACE_ID", "ws_default"), "workspace id")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		name := fs.String("name", "", "project api key display name")
		key := fs.String("key", "", "new project api key")
		enabled := fs.Bool("enabled", true, "enable the new project api key immediately")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		payload := map[string]any{
			"action":          domain.ProjectAccessActionAddKey,
			"api_key_name":    strings.TrimSpace(*name),
			"api_key":         strings.TrimSpace(*key),
			"api_key_enabled": *enabled,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		path := fmt.Sprintf("/api/workspaces/%s/projects/%s/access-config", *workspaceID, *projectID)
		return request("POST", *apiURL, path, body)
	case "update-key":
		fs := flag.NewFlagSet("vag project access update-key", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		workspaceID := fs.String("workspace", env("DEFAULT_WORKSPACE_ID", "ws_default"), "workspace id")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		keyID := fs.String("id", "", "project api key id")
		name := fs.String("name", "", "updated project api key display name")
		key := fs.String("key", "", "rotated project api key secret")
		enabled := fs.String("enabled", "", "set enabled to true or false; omit to keep unchanged")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		payload := map[string]any{
			"action":     domain.ProjectAccessActionUpdateKey,
			"api_key_id": strings.TrimSpace(*keyID),
		}
		if trimmed := strings.TrimSpace(*name); trimmed != "" {
			payload["api_key_name"] = trimmed
		}
		if trimmed := strings.TrimSpace(*key); trimmed != "" {
			payload["api_key"] = trimmed
		}
		if trimmed := strings.TrimSpace(*enabled); trimmed != "" {
			parsed, err := strconv.ParseBool(trimmed)
			if err != nil {
				return fmt.Errorf("invalid --enabled value %q: %w", trimmed, err)
			}
			payload["api_key_enabled"] = parsed
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		path := fmt.Sprintf("/api/workspaces/%s/projects/%s/access-config", *workspaceID, *projectID)
		return request("POST", *apiURL, path, body)
	case "delete-key":
		fs := flag.NewFlagSet("vag project access delete-key", flag.ExitOnError)
		apiURL := fs.String("api-url", defaultAPIURL(), "API base URL")
		workspaceID := fs.String("workspace", env("DEFAULT_WORKSPACE_ID", "ws_default"), "workspace id")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		keyID := fs.String("id", "", "project api key id")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		payload := map[string]any{
			"action":     domain.ProjectAccessActionDeleteKey,
			"api_key_id": strings.TrimSpace(*keyID),
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		path := fmt.Sprintf("/api/workspaces/%s/projects/%s/access-config", *workspaceID, *projectID)
		return request("POST", *apiURL, path, body)
	default:
		return fmt.Errorf("unknown project access command %q", args[0])
	}
}

func repairCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: vag repair scan|requeue|verify-asset")
	}
	switch args[0] {
	case "scan":
		fs := flag.NewFlagSet("vag repair scan", flag.ExitOnError)
		limit := fs.Int("limit", 100, "maximum tasks/assets/files to scan per category")
		staleAfter := fs.Duration("stale-after", 10*time.Minute, "age after which queued/running tasks are reported")
		includeOrphans := fs.Bool("orphans", true, "scan final storage directories for files not referenced by asset_version")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		ctx := context.Background()
		service, cleanup, err := newRepairService(ctx)
		if err != nil {
			return err
		}
		defer cleanup()
		report, err := service.RepairScan(ctx, app.RepairScanOptions{
			Limit:          *limit,
			StaleAfter:     *staleAfter,
			IncludeOrphans: *includeOrphans,
		})
		if err != nil {
			return err
		}
		return writeJSON(report)
	case "requeue":
		fs := flag.NewFlagSet("vag repair requeue", flag.ExitOnError)
		dryRun := fs.Bool("dry-run", false, "show whether the task can be requeued without changing state")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("usage: vag repair requeue <task_id>")
		}
		ctx := context.Background()
		service, cleanup, err := newRepairService(ctx)
		if err != nil {
			return err
		}
		defer cleanup()
		result, err := service.RepairRequeueTask(ctx, fs.Arg(0), *dryRun)
		if err != nil {
			_ = writeJSON(result)
			return err
		}
		return writeJSON(result)
	case "verify-asset":
		fs := flag.NewFlagSet("vag repair verify-asset", flag.ExitOnError)
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("usage: vag repair verify-asset <asset_id>")
		}
		ctx := context.Background()
		service, cleanup, err := newRepairService(ctx)
		if err != nil {
			return err
		}
		defer cleanup()
		result, err := service.RepairVerifyAsset(ctx, fs.Arg(0))
		if err != nil {
			return err
		}
		return writeJSON(result)
	default:
		return fmt.Errorf("unknown repair command %q", args[0])
	}
}

func storageCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: vag storage cleanup-preview|cleanup-execute")
	}
	switch args[0] {
	case "cleanup-preview":
		fs := flag.NewFlagSet("vag storage cleanup-preview", flag.ExitOnError)
		workspaceID := fs.String("workspace", env("DEFAULT_WORKSPACE_ID", "ws_default"), "workspace id")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		campaignID := fs.String("campaign", env("DEFAULT_CAMPAIGN_ID", "cmp_7day_cover"), "campaign id")
		limit := fs.Int("limit", 100, "maximum candidates per category")
		includeRejected := fs.Bool("rejected", true, "include rejected assets")
		includeGenerated := fs.Bool("generated", true, "include generated but unselected assets")
		includeTmp := fs.Bool("tmp", true, "include temporary files")
		includeOrphans := fs.Bool("orphans", true, "include orphan final files")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		ctx := context.Background()
		service, cleanup, err := newRepairService(ctx)
		if err != nil {
			return err
		}
		defer cleanup()
		report, err := service.CleanupDryRun(ctx, domain.CleanupDryRunOptions{
			Scope: domain.Scope{
				WorkspaceID: strings.TrimSpace(*workspaceID),
				ProjectID:   strings.TrimSpace(*projectID),
				CampaignID:  strings.TrimSpace(*campaignID),
			},
			IncludeRejected:      *includeRejected,
			IncludeGenerated:     *includeGenerated,
			IncludeFailedTaskTmp: *includeTmp,
			IncludeOrphans:       *includeOrphans,
			Limit:                *limit,
		})
		if err != nil {
			return err
		}
		return writeJSON(report)
	case "cleanup-execute":
		fs := flag.NewFlagSet("vag storage cleanup-execute", flag.ExitOnError)
		workspaceID := fs.String("workspace", env("DEFAULT_WORKSPACE_ID", "ws_default"), "workspace id")
		projectID := fs.String("project", env("DEFAULT_PROJECT_ID", "prj_xhs_anime"), "project id")
		campaignID := fs.String("campaign", env("DEFAULT_CAMPAIGN_ID", "cmp_7day_cover"), "campaign id")
		limit := fs.Int("limit", 100, "maximum candidates per category")
		includeRejected := fs.Bool("rejected", true, "include rejected assets")
		includeGenerated := fs.Bool("generated", true, "include generated but unselected assets")
		includeTmp := fs.Bool("tmp", true, "include temporary files")
		includeOrphans := fs.Bool("orphans", true, "include orphan final files")
		dryRunToken := fs.String("dry-run-token", "", "matching dry-run token from cleanup-preview")
		execute := fs.Bool("execute", false, "allow cleanup execution")
		confirm := fs.Bool("confirm", false, "explicitly confirm execution when no dry-run token is supplied")
		actor := fs.String("actor", "vag", "audit actor")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		ctx := context.Background()
		service, cleanup, err := newRepairService(ctx)
		if err != nil {
			return err
		}
		defer cleanup()
		report, err := service.CleanupExecute(ctx, domain.CleanupExecuteOptions{
			Scope: domain.Scope{
				WorkspaceID: strings.TrimSpace(*workspaceID),
				ProjectID:   strings.TrimSpace(*projectID),
				CampaignID:  strings.TrimSpace(*campaignID),
			},
			IncludeRejected:      *includeRejected,
			IncludeGenerated:     *includeGenerated,
			IncludeFailedTaskTmp: *includeTmp,
			IncludeOrphans:       *includeOrphans,
			Limit:                *limit,
			DryRunToken:          strings.TrimSpace(*dryRunToken),
			Execute:              *execute,
			Confirm:              *confirm,
			Actor:                strings.TrimSpace(*actor),
		})
		if err != nil {
			_ = writeJSON(report)
			return err
		}
		return writeJSON(report)
	default:
		return fmt.Errorf("unknown storage command %q", args[0])
	}
}

func newRepairService(ctx context.Context) (*app.Service, func(), error) {
	cfg := config.Load()
	conn, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		_ = conn.Close()
	}
	if err := db.Migrate(ctx, conn); err != nil {
		cleanup()
		return nil, nil, err
	}
	q, err := queue.NewRedisQueue(cfg.RedisURL)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	cleanup = func() {
		_ = q.Close()
		_ = conn.Close()
	}
	return app.NewService(cfg, store.NewPostgresStore(conn), q, storage.NewLocalStorage(cfg.StorageRoot, cfg.ThumbnailMaxWidth, cfg.ThumbnailMaxHeight)), cleanup, nil
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func request(method, apiURL, path string, body []byte) error {
	apiURL = strings.TrimRight(apiURL, "/")
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, apiURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if apiKey := strings.TrimSpace(os.Getenv("AGENT_IMAGEFLOW_API_KEY")); apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	basicUser := strings.TrimSpace(os.Getenv("AGENT_IMAGEFLOW_BASIC_USER"))
	basicPass := strings.TrimSpace(os.Getenv("AGENT_IMAGEFLOW_BASIC_PASS"))
	if basicUser != "" || basicPass != "" {
		req.SetBasicAuth(basicUser, basicPass)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, respBody, "", "  ") == nil {
		respBody = pretty.Bytes()
	}
	fmt.Println(string(respBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed: HTTP %d", resp.StatusCode)
	}
	return nil
}

func defaultAPIURL() string {
	return env("AGENT_IMAGEFLOW_API_URL", "http://localhost:8081")
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func urlQueryEscape(value string) string {
	return url.QueryEscape(value)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  vag task create --file examples/tasks/sample-image-task.json
  vag task get <task_id>
  vag task attempts <task_id>
  vag asset list
  vag asset get <asset_id>
  vag asset approve <asset_id>
  vag asset reject <asset_id>
  vag audit list [--limit 50] [--project prj_xxx]
  vag benchmark image-generation --provider mock --tasks 32 --requested-count 1 --concurrency-label worker-4
  vag batch progress --session-id <session_id> --batch-id <batch_id>
  vag batch manifest --session-id <session_id> --batch-id <batch_id> [--selected-only=false --include-rejected]
  vag storage cleanup-preview [--workspace ws_default] [--project prj_xxx] [--campaign cmp_xxx]
  vag storage cleanup-execute --execute --dry-run-token <token> [--workspace ws_default] [--project prj_xxx] [--campaign cmp_xxx]
  vag project access get
  vag project access set --enabled=true --key <api_key>
  vag project access add-key --name automation --key <api_key>
  vag project access update-key --id <api_key_id> --enabled=false
  vag project access delete-key --id <api_key_id>
  vag project provider get
  vag project provider set --enabled=true --provider mock --model mock-image
  vag project context get
  vag project context set --file examples/tasks/sample-project-visual-context.json
  vag repair scan
  vag repair requeue <task_id>
  vag repair verify-asset <asset_id>`)
}
