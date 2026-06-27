package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestInitializeAndToolsList(t *testing.T) {
	server := New(&fakeService{}, Defaults{WorkspaceID: "ws_default", ProjectID: "prj_xhs_anime", CampaignID: "cmp_7day_cover"})
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		"",
	}, "\n")

	var out bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}
	responses := decodeResponses(t, out.String())
	if len(responses) != 2 {
		t.Fatalf("got %d responses, want 2: %s", len(responses), out.String())
	}
	initialize := responses[0]["result"].(map[string]any)
	if initialize["protocolVersion"] != "2025-11-25" {
		t.Fatalf("unexpected protocol version: %#v", initialize["protocolVersion"])
	}
	capabilities := initialize["capabilities"].(map[string]any)
	if _, ok := capabilities["tools"]; !ok {
		t.Fatalf("initialize result did not declare tools capability: %#v", capabilities)
	}

	list := responses[1]["result"].(map[string]any)
	tools := list["tools"].([]any)
	names := map[string]bool{}
	for _, item := range tools {
		tool := item.(map[string]any)
		names[tool["name"].(string)] = true
		if _, ok := tool["inputSchema"].(map[string]any); !ok {
			t.Fatalf("tool is missing inputSchema: %#v", tool)
		}
	}
	for _, name := range []string{"create_image_task", "get_image_task", "list_image_assets", "select_image_asset", "reject_image_asset", "get_asset_delivery_info"} {
		if !names[name] {
			t.Fatalf("tools/list missing %s in %#v", name, names)
		}
	}
	for _, item := range tools {
		tool := item.(map[string]any)
		if tool["name"] != "list_image_assets" {
			continue
		}
		schema := tool["inputSchema"].(map[string]any)
		properties := schema["properties"].(map[string]any)
		for _, property := range []string{"project_id", "campaign_id", "source", "session_id", "batch_id", "status", "keyword", "limit"} {
			if _, ok := properties[property]; !ok {
				t.Fatalf("list_image_assets schema missing %s: %#v", property, properties)
			}
		}
		return
	}
	t.Fatalf("list_image_assets schema was not found")
}

func TestInitializeUsesConfiguredServerVersion(t *testing.T) {
	server := New(&fakeService{}, Defaults{
		WorkspaceID: "ws_default",
		ProjectID:   "prj_xhs_anime",
		CampaignID:  "cmp_7day_cover",
		Version:     "0.2.0",
	})
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test"}}}`,
		"",
	}, "\n")

	var out bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}
	responses := decodeResponses(t, out.String())
	if len(responses) != 1 {
		t.Fatalf("got %d responses, want 1: %s", len(responses), out.String())
	}
	initialize := responses[0]["result"].(map[string]any)
	serverInfo := initialize["serverInfo"].(map[string]any)
	if serverInfo["version"] != "0.2.0" {
		t.Fatalf("unexpected server version: %#v", serverInfo["version"])
	}
}

func TestCreateImageTaskSchemaDocumentsCaptionLineageSemantics(t *testing.T) {
	server := New(&fakeService{}, Defaults{})
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		"",
	}, "\n")

	var out bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}
	tools := decodeResponses(t, out.String())[1]["result"].(map[string]any)["tools"].([]any)
	for _, item := range tools {
		tool := item.(map[string]any)
		if tool["name"] != "create_image_task" {
			continue
		}
		properties := tool["inputSchema"].(map[string]any)["properties"].(map[string]any)
		metadata := properties["metadata_json"].(map[string]any)
		metadataProps := metadata["properties"].(map[string]any)
		lineage := metadataProps["caption_lineage"].(map[string]any)
		lineageProps := lineage["properties"].(map[string]any)
		for _, key := range []string{"speaker_character_id", "caption_text", "bubble_anchor", "tail_direction", "caption_intent", "auto_select_derivative", "avoid_covering_subjects"} {
			if _, ok := lineageProps[key]; !ok {
				t.Fatalf("caption_lineage schema missing %s: %#v", key, lineageProps)
			}
		}
		return
	}
	t.Fatal("create_image_task schema was not found")
}

func TestToolCallUsesDefaultsAndStructuredContent(t *testing.T) {
	service := &fakeService{}
	server := New(service, Defaults{WorkspaceID: "ws_default", ProjectID: "prj_xhs_anime", CampaignID: "cmp_7day_cover"})
	input := `{"jsonrpc":"2.0","id":"create","method":"tools/call","params":{"name":"create_image_task","arguments":{"prompt":"生成一张封面图","prompt_template":"{{prompt}}，清爽留白","template_variables":{"channel":"xiaohongshu"},"reference_images":[{"id":"ref_local","url":"https://example.com/ref.png","role":"style","source":"web-indexeddb","mime_type":"image/png"}],"mask_image":{"target_image_id":"ref_local","source":"web-mask-draft","mime_type":"image/png","has_mask":true},"generation_config":{"quality":"high"},"use_project_quality_profile":true,"selection_mode":"best_of","requested_count":2}}}` + "\n"

	var out bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}
	if service.createScope.WorkspaceID != "ws_default" || service.createScope.ProjectID != "prj_xhs_anime" || service.createScope.CampaignID != "cmp_7day_cover" {
		t.Fatalf("defaults were not applied to create scope: %#v", service.createScope)
	}
	if service.createRequest.Prompt != "生成一张封面图" || service.createRequest.RequestedCount != 2 {
		t.Fatalf("create request was not passed through: %#v", service.createRequest)
	}
	if service.createRequest.PromptTemplate != "{{prompt}}，清爽留白" || !service.createRequest.UseProjectQualityProfile {
		t.Fatalf("quality fields were not passed through: %#v", service.createRequest)
	}
	if service.createRequest.SelectionMode != "best_of" {
		t.Fatalf("selection_mode was not passed through: %#v", service.createRequest)
	}
	if service.createRequest.TemplateVariables["channel"] != "xiaohongshu" || len(service.createRequest.ReferenceImages) != 1 {
		t.Fatalf("quality variable/reference fields were not passed through: %#v", service.createRequest)
	}
	if service.createRequest.ReferenceImages[0].Source != "web-indexeddb" || service.createRequest.ReferenceImages[0].MimeType != "image/png" {
		t.Fatalf("reference descriptor fields were not passed through: %#v", service.createRequest.ReferenceImages[0])
	}
	if service.createRequest.MaskImage == nil || service.createRequest.MaskImage.TargetImageID != "ref_local" || !service.createRequest.MaskImage.HasMask {
		t.Fatalf("mask descriptor was not passed through: %#v", service.createRequest.MaskImage)
	}
	if string(service.createRequest.GenerationConfig) != `{"quality":"high"}` {
		t.Fatalf("generation_config was not passed through: %s", service.createRequest.GenerationConfig)
	}

	response := decodeResponses(t, out.String())[0]
	result := response["result"].(map[string]any)
	if result["isError"] != false {
		t.Fatalf("tool result was error: %#v", result)
	}
	structured := result["structuredContent"].(map[string]any)
	if structured["task_id"] != "task_fake" {
		t.Fatalf("unexpected structured content: %#v", structured)
	}
	content := result["content"].([]any)[0].(map[string]any)
	if !strings.Contains(content["text"].(string), `"task_id": "task_fake"`) {
		t.Fatalf("text content did not include serialized JSON: %#v", content)
	}
}

func TestToolCallMapsCompatibilityStatuses(t *testing.T) {
	server := New(&fakeService{}, Defaults{})
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_asset_delivery_info","arguments":{"asset_id":"asset_fake"}}}` + "\n"

	var out bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}
	response := decodeResponses(t, out.String())[0]
	result := response["result"].(map[string]any)
	structured := result["structuredContent"].(map[string]any)
	if structured["status"] != "selected" {
		t.Fatalf("approved status was not mapped to selected: %#v", structured["status"])
	}
	content := result["content"].([]any)[0].(map[string]any)
	text := content["text"].(string)
	if !strings.Contains(text, `"status": "selected"`) || strings.Contains(text, `"status": "approved"`) {
		t.Fatalf("text content did not use semantic status: %s", text)
	}
	if strings.Contains(text, "local_path") {
		t.Fatalf("text content exposed local path: %s", text)
	}
	delivery := structured["delivery"].(map[string]any)
	if _, ok := delivery["local_path"]; ok {
		t.Fatalf("structured content exposed local path: %#v", delivery)
	}
}

func TestListImageAssetsPassesFiltersAndDefaults(t *testing.T) {
	service := &fakeService{}
	server := New(service, Defaults{WorkspaceID: "ws_default", ProjectID: "prj_default", CampaignID: "cmp_default"})
	input := `{"jsonrpc":"2.0","id":"list","method":"tools/call","params":{"name":"list_image_assets","arguments":{"source":"mcp","session_id":"session_1","batch_id":"batch_1","status":"selected","keyword":"hero","limit":24}}}` + "\n"

	var out bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}
	query := service.listQuery
	if query.ProjectID != "prj_default" || query.CampaignID != "cmp_default" {
		t.Fatalf("defaults were not applied to list scope: %#v", query)
	}
	if query.Source != "mcp" || query.SessionID != "session_1" || query.BatchID != "batch_1" {
		t.Fatalf("metadata filters were not passed through: %#v", query)
	}
	if query.Status != "selected" || query.Keyword != "hero" || query.Limit != 24 {
		t.Fatalf("list filters were not passed through: %#v", query)
	}

	response := decodeResponses(t, out.String())[0]
	result := response["result"].(map[string]any)
	if result["isError"] != false {
		t.Fatalf("tool result was error: %#v", result)
	}
	text := result["content"].([]any)[0].(map[string]any)["text"].(string)
	if strings.Contains(text, "local_path") {
		t.Fatalf("list response exposed local path: %s", text)
	}
	assets := result["structuredContent"].([]any)
	delivery := assets[0].(map[string]any)["delivery"].(map[string]any)
	if _, ok := delivery["local_path"]; ok {
		t.Fatalf("list structured content exposed local path: %#v", delivery)
	}
}

func TestListImageAssetsAllowsExplicitScopeAndLimitPassthrough(t *testing.T) {
	service := &fakeService{}
	server := New(service, Defaults{ProjectID: "prj_default", CampaignID: "cmp_default"})
	input := `{"jsonrpc":"2.0","id":"list","method":"tools/call","params":{"name":"list_image_assets","arguments":{"project_id":"prj_explicit","campaign_id":"cmp_explicit","limit":100}}}` + "\n"

	var out bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}
	query := service.listQuery
	if query.ProjectID != "prj_explicit" || query.CampaignID != "cmp_explicit" {
		t.Fatalf("explicit scope was not passed through: %#v", query)
	}
	if query.Limit != 100 {
		t.Fatalf("limit was not passed through: %#v", query)
	}
}

func decodeResponses(t *testing.T, output string) []map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	responses := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var response map[string]any
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("invalid JSON response %q: %v", line, err)
		}
		responses = append(responses, response)
	}
	return responses
}

type fakeService struct {
	createScope   domain.Scope
	createRequest domain.CreateTaskRequest
	listQuery     domain.AssetListQuery
}

func (f *fakeService) CreateTask(ctx context.Context, scope domain.Scope, req domain.CreateTaskRequest) (domain.TaskResponse, error) {
	f.createScope = scope
	f.createRequest = req
	return domain.TaskResponse{
		Task: domain.Task{
			ID:             "task_fake",
			WorkspaceID:    scope.WorkspaceID,
			ProjectID:      scope.ProjectID,
			CampaignID:     scope.CampaignID,
			Title:          "Fake task",
			Prompt:         req.Prompt,
			Provider:       "mock",
			Status:         domain.TaskQueued,
			RequestedCount: req.RequestedCount,
			CreatedAt:      time.Unix(1, 0).UTC(),
			UpdatedAt:      time.Unix(1, 0).UTC(),
		},
		AssetIDs: []string{},
		Assets:   []domain.AssetListEntry{},
	}, nil
}

func (f *fakeService) GetTask(ctx context.Context, taskID string) (domain.TaskResponse, error) {
	return domain.TaskResponse{
		Task: domain.Task{
			ID:        taskID,
			Status:    domain.TaskCompleted,
			CreatedAt: time.Unix(1, 0).UTC(),
			UpdatedAt: time.Unix(2, 0).UTC(),
		},
		AssetIDs: []string{"asset_fake"},
		Assets: []domain.AssetListEntry{
			{AssetID: "asset_fake", Status: domain.AssetDraft},
		},
	}, nil
}

func (f *fakeService) ListAssets(ctx context.Context, query domain.AssetListQuery) ([]domain.AssetResponse, error) {
	f.listQuery = query
	return []domain.AssetResponse{fakeAsset(domain.AssetDraft)}, nil
}

func (f *fakeService) GetAsset(ctx context.Context, assetID string) (domain.AssetResponse, error) {
	return fakeAsset(domain.AssetApproved), nil
}

func (f *fakeService) ReviewAsset(ctx context.Context, assetID, action string) (domain.AssetResponse, error) {
	if action == "reject" {
		return fakeAsset(domain.AssetRejected), nil
	}
	return fakeAsset(domain.AssetApproved), nil
}

func fakeAsset(status string) domain.AssetResponse {
	return domain.AssetResponse{
		AssetID:        "asset_fake",
		WorkspaceID:    "ws_default",
		ProjectID:      "prj_xhs_anime",
		CampaignID:     "cmp_7day_cover",
		TaskID:         "task_fake",
		CurrentVersion: 1,
		Status:         status,
		Hash:           "sha256:fake",
		Provider:       "mock",
		Model:          "mock-image-v1",
		Prompt:         "生成一张封面图",
		Delivery: domain.DeliveryInfo{
			LocalPath:    "/data/agent-imageflow/original.png",
			DownloadURL:  "http://localhost:8081/api/assets/asset_fake/original",
			ThumbnailURL: "http://localhost:8081/api/assets/asset_fake/thumbnail",
			MetadataURL:  "http://localhost:8081/api/assets/asset_fake",
		},
		CreatedAt: time.Unix(1, 0).UTC(),
	}
}
