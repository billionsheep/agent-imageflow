package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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
  vag asset reject <asset_id>`)
}
