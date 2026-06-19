package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/app"
	"github.com/billionsheep/agent-imageflow/internal/config"
	"github.com/billionsheep/agent-imageflow/internal/db"
	"github.com/billionsheep/agent-imageflow/internal/domain"
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
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		path := fmt.Sprintf("/api/projects/%s/campaigns/%s/assets", *projectID, *campaignID)
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

func projectCmd(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: vag project access get|set|add-key|update-key|delete-key")
	}
	switch args[0] {
	case "access":
		return projectAccessCmd(args[1:])
	default:
		return fmt.Errorf("unknown project command %q", args[0])
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

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  vag task create --file examples/tasks/sample-image-task.json
  vag task get <task_id>
  vag asset list
  vag asset get <asset_id>
  vag asset approve <asset_id>
  vag asset reject <asset_id>
  vag audit list [--limit 50] [--project prj_xxx]
  vag storage cleanup-preview [--workspace ws_default] [--project prj_xxx] [--campaign cmp_xxx]
  vag storage cleanup-execute --execute --dry-run-token <token> [--workspace ws_default] [--project prj_xxx] [--campaign cmp_xxx]
  vag project access get
  vag project access set --enabled=true --key <api_key>
  vag project access add-key --name automation --key <api_key>
  vag project access update-key --id <api_key_id> --enabled=false
  vag project access delete-key --id <api_key_id>
  vag repair scan
  vag repair requeue <task_id>
  vag repair verify-asset <asset_id>`)
}
