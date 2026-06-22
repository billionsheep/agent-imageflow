package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

const protocolVersion = "2025-11-25"

type Service interface {
	CreateTask(context.Context, domain.Scope, domain.CreateTaskRequest) (domain.TaskResponse, error)
	GetTask(context.Context, string) (domain.TaskResponse, error)
	ListAssets(context.Context, domain.AssetListQuery) ([]domain.AssetResponse, error)
	GetAsset(context.Context, string) (domain.AssetResponse, error)
	ReviewAsset(context.Context, string, string) (domain.AssetResponse, error)
}

type Defaults struct {
	WorkspaceID string
	ProjectID   string
	CampaignID  string
}

type Server struct {
	service  Service
	defaults Defaults
}

func New(service Service, defaults Defaults) *Server {
	return &Server{service: service, defaults: defaults}
}

func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	encoder := json.NewEncoder(out)

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			if writeErr := encoder.Encode(rpcResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   &rpcError{Code: -32700, Message: "Parse error", Data: err.Error()},
			}); writeErr != nil {
				return writeErr
			}
			continue
		}
		if len(req.ID) == 0 {
			s.handleNotification(req)
			continue
		}

		result, rpcErr := s.handleRequest(ctx, req)
		response := rpcResponse{
			JSONRPC: "2.0",
			ID:      rawID(req.ID),
			Result:  result,
			Error:   rpcErr,
		}
		if err := encoder.Encode(response); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) handleNotification(req rpcRequest) {
	switch req.Method {
	case "notifications/initialized", "notifications/cancelled":
	default:
		// Unknown notifications are ignored per JSON-RPC's no-response behavior.
	}
}

func (s *Server) handleRequest(ctx context.Context, req rpcRequest) (any, *rpcError) {
	if req.JSONRPC != "2.0" {
		return nil, &rpcError{Code: -32600, Message: "Invalid Request", Data: "jsonrpc must be 2.0"}
	}
	switch req.Method {
	case "initialize":
		return s.initializeResult(req.Params), nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": s.tools()}, nil
	case "tools/call":
		result, err := s.callTool(ctx, req.Params)
		if err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, &rpcError{Code: -32601, Message: "Method not found", Data: req.Method}
	}
}

func (s *Server) initializeResult(params json.RawMessage) map[string]any {
	version := protocolVersion
	var input struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if len(params) > 0 {
		_ = json.Unmarshal(params, &input)
	}
	if input.ProtocolVersion == "2025-06-18" {
		version = input.ProtocolVersion
	}
	return map[string]any{
		"protocolVersion": version,
		"capabilities": map[string]any{
			"tools": map[string]any{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]any{
			"name":        "agent-imageflow",
			"title":       "Agent ImageFlow",
			"version":     "0.1.0",
			"description": "MCP tools for creating and delivering traceable image assets.",
		},
		"instructions": "Create structured image tasks, poll task status, select or reject generated assets, and fetch delivery information.",
	}
}

func (s *Server) callTool(ctx context.Context, params json.RawMessage) (any, *rpcError) {
	var call toolCallParams
	if err := decodeParams(params, &call); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid params", Data: err.Error()}
	}
	if strings.TrimSpace(call.Name) == "" {
		return nil, &rpcError{Code: -32602, Message: "Invalid params", Data: "tool name is required"}
	}

	value, err := s.executeTool(ctx, call)
	if err != nil {
		return toolErrorResult(err), nil
	}
	return toolSuccessResult(value), nil
}

func (s *Server) executeTool(ctx context.Context, call toolCallParams) (any, error) {
	switch call.Name {
	case "create_image_task":
		var args createImageTaskArgs
		if err := decodeArgs(call.Arguments, &args); err != nil {
			return nil, err
		}
		return s.service.CreateTask(ctx, s.scope(args.WorkspaceID, args.ProjectID, args.CampaignID), args.request())
	case "get_image_task":
		var args taskIDArgs
		if err := decodeArgs(call.Arguments, &args); err != nil {
			return nil, err
		}
		if strings.TrimSpace(args.TaskID) == "" {
			return nil, errors.New("task_id is required")
		}
		return s.service.GetTask(ctx, args.TaskID)
	case "list_image_assets":
		var args listAssetsArgs
		if err := decodeArgs(call.Arguments, &args); err != nil {
			return nil, err
		}
		projectID := firstNonEmpty(args.ProjectID, s.defaults.ProjectID)
		campaignID := firstNonEmpty(args.CampaignID, s.defaults.CampaignID)
		return s.service.ListAssets(ctx, domain.AssetListQuery{
			ProjectID:  projectID,
			CampaignID: campaignID,
		})
	case "select_image_asset":
		var args assetIDArgs
		if err := decodeArgs(call.Arguments, &args); err != nil {
			return nil, err
		}
		if strings.TrimSpace(args.AssetID) == "" {
			return nil, errors.New("asset_id is required")
		}
		return s.service.ReviewAsset(ctx, args.AssetID, "approve")
	case "reject_image_asset":
		var args assetIDArgs
		if err := decodeArgs(call.Arguments, &args); err != nil {
			return nil, err
		}
		if strings.TrimSpace(args.AssetID) == "" {
			return nil, errors.New("asset_id is required")
		}
		return s.service.ReviewAsset(ctx, args.AssetID, "reject")
	case "get_asset_delivery_info":
		var args assetIDArgs
		if err := decodeArgs(call.Arguments, &args); err != nil {
			return nil, err
		}
		if strings.TrimSpace(args.AssetID) == "" {
			return nil, errors.New("asset_id is required")
		}
		return s.service.GetAsset(ctx, args.AssetID)
	default:
		return nil, fmt.Errorf("unknown tool: %s", call.Name)
	}
}

func (s *Server) scope(workspaceID, projectID, campaignID string) domain.Scope {
	return domain.Scope{
		WorkspaceID: firstNonEmpty(workspaceID, s.defaults.WorkspaceID),
		ProjectID:   firstNonEmpty(projectID, s.defaults.ProjectID),
		CampaignID:  firstNonEmpty(campaignID, s.defaults.CampaignID),
	}
}

func (s *Server) tools() []toolDefinition {
	return []toolDefinition{
		{
			Name:        "create_image_task",
			Title:       "Create Image Task",
			Description: "Create a structured image generation task in Agent ImageFlow and enqueue it for the worker.",
			InputSchema: objectSchema(map[string]any{
				"workspace_id":       stringProp("Workspace id. Defaults to DEFAULT_WORKSPACE_ID."),
				"project_id":         stringProp("Project id. Defaults to DEFAULT_PROJECT_ID."),
				"campaign_id":        stringProp("Campaign id. Defaults to DEFAULT_CAMPAIGN_ID."),
				"idempotency_key":    stringProp("Optional idempotency key scoped to workspace/project."),
				"title":              stringProp("Human-readable task title."),
				"purpose":            stringProp("Business purpose for the image asset."),
				"prompt":             stringProp("Image generation prompt."),
				"negative_prompt":    stringProp("Negative prompt."),
				"style_preset":       stringProp("Style preset name."),
				"prompt_template":    stringProp("Reusable prompt template. Use {{prompt}}, {{title}}, or metadata/template variables."),
				"template_variables": map[string]any{"type": "object", "description": "Variables for prompt_template rendering."},
				"reference_images": map[string]any{
					"type":        "array",
					"description": "Reference image descriptors for service-managed input reuse. openai-compatible can consume url, asset_id, or input_file_id after server-side resolution.",
					"items": objectSchema(map[string]any{
						"id":            stringProp("Optional client-side reference id."),
						"url":           stringProp("Reference image URL."),
						"asset_id":      stringProp("Existing Agent ImageFlow asset id."),
						"input_file_id": stringProp("Existing scope input file id."),
						"role":          stringProp("Reference role such as style, character, product, or composition."),
						"source":        stringProp("Descriptor source such as web-indexeddb or external-url."),
						"mime_type":     stringProp("Reference image MIME type when known."),
						"width":         map[string]any{"type": "integer"},
						"height":        map[string]any{"type": "integer"},
						"weight":        map[string]any{"type": "number"},
					}, nil),
				},
				"character_ids":              map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Project visual context character profile ids to expand into this task."},
				"reference_asset_ids":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Existing project asset ids to use as visual context references."},
				"prompt_recipe_id":           stringProp("Project visual context prompt recipe id to expand."),
				"use_project_visual_context": map[string]any{"type": "boolean", "description": "Expand project visual context before enqueueing."},
				"best_of_config": map[string]any{
					"type":        "object",
					"description": "Optional best-of scoring configuration. strategy may be local_metadata_v1 or http_judge_v1; auto_reject_non_selected rejects losing candidates after auto selection.",
					"properties": map[string]any{
						"strategy":                 stringProp("Best-of scoring strategy."),
						"judge_prompt":             stringProp("Optional instruction for external visual/LLM judge strategies."),
						"auto_reject_non_selected": map[string]any{"type": "boolean", "description": "When true, non-selected candidates are automatically rejected after best-of completes."},
					},
				},
				"mask_image": map[string]any{
					"type":        "object",
					"description": "Optional mask/edit descriptor. openai-compatible can consume url, asset_id, or input_file_id after server-side resolution.",
					"properties": map[string]any{
						"id":              stringProp("Optional client-side mask image id."),
						"url":             stringProp("Optional mask image URL."),
						"asset_id":        stringProp("Existing Agent ImageFlow asset id for the mask."),
						"input_file_id":   stringProp("Existing scope input file id for the mask."),
						"target_image_id": stringProp("Reference image id that the mask edits."),
						"source":          stringProp("Descriptor source such as web-mask-draft."),
						"mime_type":       stringProp("Mask image MIME type when known."),
						"width":           map[string]any{"type": "integer"},
						"height":          map[string]any{"type": "integer"},
						"has_mask":        map[string]any{"type": "boolean"},
					},
				},
				"generation_config":           map[string]any{"type": "object", "description": "Provider-facing generation parameters to store with the task."},
				"use_project_quality_profile": map[string]any{"type": "boolean", "description": "Apply the project-level quality profile before enqueueing."},
				"aspect_ratio":                stringProp("Aspect ratio such as 1:1, 3:4, or 16:9."),
				"output_format":               stringProp("Output format. Defaults to png."),
				"requested_count":             map[string]any{"type": "integer", "minimum": 1, "maximum": 10, "description": "Number of candidate images to generate."},
				"provider":                    stringProp("Provider id. Use mock or a configured provider such as openai-compatible."),
				"selection_mode":              stringProp("Optional product-level selection mode such as manual_optional or auto."),
				"review_required":             map[string]any{"type": "boolean", "description": "Compatibility flag; first MCP slice normally leaves this false."},
				"metadata_json":               map[string]any{"type": "object", "description": "Arbitrary structured metadata for downstream workflows."},
			}, []string{"prompt"}),
		},
		{
			Name:        "get_image_task",
			Title:       "Get Image Task",
			Description: "Get task status and generated asset ids.",
			InputSchema: objectSchema(map[string]any{"task_id": stringProp("Task id.")}, []string{"task_id"}),
		},
		{
			Name:        "list_image_assets",
			Title:       "List Image Assets",
			Description: "List image assets for a project/campaign. Defaults to the configured demo project and campaign.",
			InputSchema: objectSchema(map[string]any{
				"project_id":  stringProp("Project id. Defaults to DEFAULT_PROJECT_ID."),
				"campaign_id": stringProp("Campaign id. Defaults to DEFAULT_CAMPAIGN_ID."),
			}, nil),
		},
		{
			Name:        "select_image_asset",
			Title:       "Select Image Asset",
			Description: "Mark a generated candidate as selected. Uses the existing approve transition internally.",
			InputSchema: objectSchema(map[string]any{"asset_id": stringProp("Asset id.")}, []string{"asset_id"}),
		},
		{
			Name:        "reject_image_asset",
			Title:       "Reject Image Asset",
			Description: "Mark a generated candidate as rejected.",
			InputSchema: objectSchema(map[string]any{"asset_id": stringProp("Asset id.")}, []string{"asset_id"}),
		},
		{
			Name:        "get_asset_delivery_info",
			Title:       "Get Asset Delivery Info",
			Description: "Get asset metadata and stable original/thumbnail/metadata delivery URLs.",
			InputSchema: objectSchema(map[string]any{"asset_id": stringProp("Asset id.")}, []string{"asset_id"}),
		},
	}
}

func toolSuccessResult(value any) map[string]any {
	semantic := semanticValue(value)
	return map[string]any{
		"content": []map[string]string{
			{"type": "text", "text": prettyJSON(semantic)},
		},
		"structuredContent": semantic,
		"isError":           false,
	}
}

func toolErrorResult(err error) map[string]any {
	return map[string]any{
		"content": []map[string]string{
			{"type": "text", "text": err.Error()},
		},
		"isError": true,
	}
}

func semanticValue(value any) any {
	data, err := json.Marshal(value)
	if err != nil {
		return map[string]any{"value": fmt.Sprint(value)}
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return map[string]any{"value": string(data)}
	}
	return mapStatuses(decoded)
}

func mapStatuses(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == "status" {
				if status, ok := child.(string); ok {
					typed[key] = semanticStatus(status)
					continue
				}
			}
			typed[key] = mapStatuses(child)
		}
		return typed
	case []any:
		for i, child := range typed {
			typed[i] = mapStatuses(child)
		}
		return typed
	default:
		return value
	}
}

func semanticStatus(status string) string {
	switch status {
	case domain.AssetDraft:
		return "generated"
	case domain.AssetApproved:
		return "selected"
	default:
		return status
	}
}

func prettyJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(data)
}

func decodeParams(raw json.RawMessage, dest any) error {
	if len(raw) == 0 {
		raw = []byte(`{}`)
	}
	return json.Unmarshal(raw, dest)
}

func decodeArgs(raw json.RawMessage, dest any) error {
	if len(raw) == 0 {
		raw = []byte(`{}`)
	}
	return json.Unmarshal(raw, dest)
}

func rawID(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var id any
	if err := json.Unmarshal(raw, &id); err != nil {
		return nil
	}
	return id
}

func firstNonEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringProp(description string) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": description,
	}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type toolDefinition struct {
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type createImageTaskArgs struct {
	WorkspaceID              string                  `json:"workspace_id"`
	ProjectID                string                  `json:"project_id"`
	CampaignID               string                  `json:"campaign_id"`
	IdempotencyKey           string                  `json:"idempotency_key"`
	Title                    string                  `json:"title"`
	Purpose                  string                  `json:"purpose"`
	Prompt                   string                  `json:"prompt"`
	NegativePrompt           string                  `json:"negative_prompt"`
	StylePreset              string                  `json:"style_preset"`
	PromptTemplate           string                  `json:"prompt_template"`
	TemplateVariables        map[string]any          `json:"template_variables"`
	ReferenceImages          []domain.ReferenceImage `json:"reference_images"`
	CharacterIDs             []string                `json:"character_ids"`
	ReferenceAssetIDs        []string                `json:"reference_asset_ids"`
	PromptRecipeID           string                  `json:"prompt_recipe_id"`
	UseProjectVisualContext  bool                    `json:"use_project_visual_context"`
	BestOfConfig             *domain.BestOfConfig    `json:"best_of_config"`
	MaskImage                *domain.MaskImage       `json:"mask_image"`
	GenerationConfig         json.RawMessage         `json:"generation_config"`
	UseProjectQualityProfile bool                    `json:"use_project_quality_profile"`
	AspectRatio              string                  `json:"aspect_ratio"`
	OutputFormat             string                  `json:"output_format"`
	RequestedCount           int                     `json:"requested_count"`
	Provider                 string                  `json:"provider"`
	SelectionMode            string                  `json:"selection_mode"`
	ReviewRequired           bool                    `json:"review_required"`
	MetadataJSON             json.RawMessage         `json:"metadata_json"`
}

func (a createImageTaskArgs) request() domain.CreateTaskRequest {
	return domain.CreateTaskRequest{
		IdempotencyKey:           a.IdempotencyKey,
		Title:                    a.Title,
		Purpose:                  a.Purpose,
		Prompt:                   a.Prompt,
		NegativePrompt:           a.NegativePrompt,
		StylePreset:              a.StylePreset,
		PromptTemplate:           a.PromptTemplate,
		TemplateVariables:        a.TemplateVariables,
		ReferenceImages:          a.ReferenceImages,
		CharacterIDs:             a.CharacterIDs,
		ReferenceAssetIDs:        a.ReferenceAssetIDs,
		PromptRecipeID:           a.PromptRecipeID,
		UseProjectVisualContext:  a.UseProjectVisualContext,
		BestOfConfig:             a.BestOfConfig,
		MaskImage:                a.MaskImage,
		GenerationConfig:         a.GenerationConfig,
		UseProjectQualityProfile: a.UseProjectQualityProfile,
		AspectRatio:              a.AspectRatio,
		OutputFormat:             a.OutputFormat,
		RequestedCount:           a.RequestedCount,
		Provider:                 a.Provider,
		SelectionMode:            a.SelectionMode,
		ReviewRequired:           a.ReviewRequired,
		MetadataJSON:             a.MetadataJSON,
	}
}

type taskIDArgs struct {
	TaskID string `json:"task_id"`
}

type listAssetsArgs struct {
	ProjectID  string `json:"project_id"`
	CampaignID string `json:"campaign_id"`
}

type assetIDArgs struct {
	AssetID string `json:"asset_id"`
}
