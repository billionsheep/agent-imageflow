package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

const maxImageResponseBytes = 50 << 20

const (
	OpenAICompatibleRequestModeImagesSyncURL   = "images_sync_url"
	OpenAICompatibleRequestModeImagesSyncB64   = "images_sync_b64"
	OpenAICompatibleRequestModeImagesStream    = "images_stream"
	OpenAICompatibleRequestModeResponsesStream = "responses_stream"

	openAICompatibleAPIModeImages         = "images"
	openAICompatibleAPIModeResponses      = "responses"
	openAICompatibleEndpointGenerations   = "/images/generations"
	openAICompatibleEndpointEdits         = "/images/edits"
	openAICompatibleEndpointResponses     = "/responses"
	openAICompatibleResponseFormatB64     = "b64_json"
	openAICompatibleResponseFormatURL     = "url"
	openAICompatibleResponseFormatOmit    = "omitted"
	openAICompatibleDefaultImagesModel    = "gpt-image-2"
	openAICompatibleDefaultResponsesModel = "gpt-5.5"
	openAICompatibleDefaultMaxN           = 1
)

type OpenAICompatibleConfig struct {
	BaseURL                      string
	APIKey                       string
	Model                        string
	TimeoutSeconds               int
	ConnectTimeoutSeconds        int
	ResponseHeaderTimeoutSeconds int
	TotalTimeoutSeconds          int
	HTTPClient                   *http.Client
}

type OpenAICompatibleProvider struct {
	baseURL         string
	apiKey          string
	model           string
	modelConfigured bool
	httpClient      *http.Client
}

type openAICompatibleRequestShape struct {
	APIMode        string
	Endpoint       string
	Operation      string
	RequestMode    string
	ResponseFormat string
	Stream         bool
	PartialImages  int
	N              int
	TimeoutSeconds int
	PartialCount   int
}

func NewOpenAICompatibleProvider(cfg OpenAICompatibleConfig) OpenAICompatibleProvider {
	totalTimeout := providerTimeoutDuration(firstPositive(cfg.TotalTimeoutSeconds, cfg.TimeoutSeconds, 300))
	client := cfg.HTTPClient
	if client == nil {
		connectTimeout := providerTimeoutDuration(firstPositive(cfg.ConnectTimeoutSeconds, 30))
		headerTimeout := providerTimeoutDuration(firstPositive(cfg.ResponseHeaderTimeoutSeconds, cfg.TimeoutSeconds, 300))
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.DialContext = (&net.Dialer{Timeout: connectTimeout}).DialContext
		transport.ResponseHeaderTimeout = headerTimeout
		client = &http.Client{
			Timeout:   totalTimeout,
			Transport: transport,
		}
	}
	return OpenAICompatibleProvider{
		baseURL:         strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:          strings.TrimSpace(cfg.APIKey),
		model:           firstNonEmpty(cfg.Model, openAICompatibleDefaultImagesModel),
		modelConfigured: strings.TrimSpace(cfg.Model) != "",
		httpClient:      client,
	}
}

func (p OpenAICompatibleProvider) Configured() bool {
	return p.baseURL != "" && p.apiKey != "" && p.model != ""
}

func (p OpenAICompatibleProvider) modelForTask(task domain.Task, apiMode string) string {
	input := parseTaskStructuredProviderInput(task)
	if len(input.GenerationConfig) > 0 {
		var config map[string]any
		if json.Unmarshal(input.GenerationConfig, &config) == nil {
			if value := trimmedConfigString(config, "model"); value != "" {
				return value
			}
		}
	}
	if input.ProviderProfile.Enabled &&
		strings.TrimSpace(input.ProviderProfile.Provider) == OpenAICompatibleProviderID &&
		strings.TrimSpace(input.ProviderProfile.Model) != "" {
		return strings.TrimSpace(input.ProviderProfile.Model)
	}
	if apiMode == openAICompatibleAPIModeResponses && !p.modelConfigured {
		return openAICompatibleDefaultResponsesModel
	}
	return p.model
}

type openAICompatibleTaskOptions struct {
	APIMode                 string
	Stream                  bool
	StreamConfigured        bool
	PartialImages           int
	PartialImagesConfigured bool
	PreferredResponseFormat string
	TimeoutSeconds          int
}

func openAICompatibleRequestShapeForTask(task domain.Task, endpoint, operation string, requestN int) openAICompatibleRequestShape {
	options := openAICompatibleOptionsForTask(task)
	responseFormat := options.PreferredResponseFormat
	requestMode := OpenAICompatibleRequestModeImagesSyncURL
	recordedResponseFormat := openAICompatibleResponseFormatOmit
	if responseFormat == openAICompatibleResponseFormatB64 {
		requestMode = OpenAICompatibleRequestModeImagesSyncB64
		recordedResponseFormat = openAICompatibleResponseFormatB64
	}
	if options.Stream {
		requestMode = OpenAICompatibleRequestModeImagesStream
	}
	if options.APIMode == openAICompatibleAPIModeResponses {
		requestMode = OpenAICompatibleRequestModeResponsesStream
		endpoint = openAICompatibleEndpointResponses
	}
	return openAICompatibleRequestShape{
		APIMode:        options.APIMode,
		Endpoint:       endpoint,
		Operation:      operation,
		RequestMode:    requestMode,
		ResponseFormat: recordedResponseFormat,
		Stream:         options.Stream,
		PartialImages:  options.PartialImages,
		N:              requestN,
		TimeoutSeconds: options.TimeoutSeconds,
	}
}

func openAICompatibleOptionsForTask(task domain.Task) openAICompatibleTaskOptions {
	input := parseTaskStructuredProviderInput(task)
	options := openAICompatibleTaskOptions{
		APIMode:                 openAICompatibleAPIModeImages,
		PreferredResponseFormat: openAICompatibleResponseFormatURL,
	}
	if input.ProviderProfile.Enabled && strings.TrimSpace(input.ProviderProfile.Provider) == OpenAICompatibleProviderID {
		if strings.TrimSpace(input.ProviderProfile.APIMode) == openAICompatibleAPIModeResponses {
			options.APIMode = openAICompatibleAPIModeResponses
		}
		if input.ProviderProfile.Stream != nil {
			options.Stream = *input.ProviderProfile.Stream
			options.StreamConfigured = true
		}
		if input.ProviderProfile.PartialImages != nil {
			options.PartialImages = clampInt(*input.ProviderProfile.PartialImages, 0, 3)
			options.PartialImagesConfigured = true
		}
		if strings.TrimSpace(input.ProviderProfile.PreferredResponseFormat) == openAICompatibleResponseFormatB64 {
			options.PreferredResponseFormat = openAICompatibleResponseFormatB64
		}
		if input.ProviderProfile.TimeoutSeconds > 0 {
			options.TimeoutSeconds = input.ProviderProfile.TimeoutSeconds
		}
	}
	var config map[string]any
	if len(input.GenerationConfig) > 0 && json.Unmarshal(input.GenerationConfig, &config) == nil {
		if value := trimmedConfigString(config, "api_mode"); value == openAICompatibleAPIModeImages || value == openAICompatibleAPIModeResponses {
			options.APIMode = value
		}
		if value, ok := configBool(config, "stream"); ok {
			options.Stream = value
			options.StreamConfigured = true
		}
		if value, ok := configIntInRange(config, "partial_images", 0, 3); ok {
			options.PartialImages = value
			options.PartialImagesConfigured = true
		}
		if value := trimmedConfigString(config, "preferred_response_format"); value == openAICompatibleResponseFormatB64 || value == openAICompatibleResponseFormatURL {
			options.PreferredResponseFormat = value
		}
		if value, ok := configIntInRange(config, "timeout_seconds", 1, 3600); ok {
			options.TimeoutSeconds = value
		}
	}
	if options.APIMode == openAICompatibleAPIModeResponses && !options.StreamConfigured {
		options.Stream = true
	}
	if options.Stream && !options.PartialImagesConfigured {
		options.PartialImages = 1
	}
	return options
}

func (p OpenAICompatibleProvider) Generate(ctx context.Context, task domain.Task) (Result, error) {
	if !p.Configured() {
		return Result{}, fmt.Errorf("openai-compatible provider is not configured")
	}

	size := sizeForAspectRatio(task.AspectRatio)
	outputFormat := firstNonEmpty(task.OutputFormat, "png")
	input := parseTaskStructuredProviderInput(task)
	editInput, err := resolveStructuredEditInput(input)
	if err != nil {
		return Result{
			Status:       "failed",
			CostRaw:      []byte(`{"provider":"openai-compatible"}`),
			ErrorCode:    "invalid_edit_input",
			ErrorMessage: err.Error(),
		}, err
	}
	options := openAICompatibleOptionsForTask(task)
	if options.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(options.TimeoutSeconds)*time.Second)
		defer cancel()
	}
	if options.APIMode == openAICompatibleAPIModeResponses {
		return p.generateResponses(ctx, task, size, outputFormat, editInput)
	}
	if editInput != nil {
		return p.generateEdit(ctx, task, size, outputFormat, editInput)
	}
	return p.generateGeneration(ctx, task, size, outputFormat)
}

func (p OpenAICompatibleProvider) generateGeneration(ctx context.Context, task domain.Task, size, outputFormat string) (Result, error) {
	model := p.modelForTask(task, openAICompatibleAPIModeImages)
	requestN := max(1, min(task.RequestedCount, taskProviderMaxN(task, OpenAICompatibleProviderID, openAICompatibleDefaultMaxN)))
	shape := openAICompatibleRequestShapeForTask(task, openAICompatibleEndpointGenerations, "generation", requestN)
	body := map[string]any{
		"model":         model,
		"prompt":        task.Prompt,
		"n":             requestN,
		"size":          size,
		"output_format": outputFormat,
	}
	if shape.ResponseFormat == openAICompatibleResponseFormatB64 {
		body["response_format"] = openAICompatibleResponseFormatB64
	}
	if shape.Stream {
		body["stream"] = true
		body["partial_images"] = shape.PartialImages
	}
	if task.NegativePrompt != "" {
		body["negative_prompt"] = task.NegativePrompt
	}
	if task.StylePreset != "" {
		body["style_preset"] = task.StylePreset
	}
	for key, value := range openAICompatiblePassthroughParams(task) {
		body[key] = value
	}

	requestBytes, err := json.Marshal(body)
	if err != nil {
		return Result{}, err
	}
	url := p.baseURL + openAICompatibleEndpointGenerations
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBytes))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	started := time.Now()
	var firstByteAt time.Time
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			firstByteAt = time.Now()
		},
	}))
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return openAIRequestErrorResult(task, started, firstByteAt, err), err
	}
	var result Result
	var parseErr error
	if shape.Stream && isOpenAIEventStreamResponse(resp) {
		result, parseErr = p.parseImagesStreamResponse(ctx, task, size, shape, model, resp)
	} else {
		result, parseErr = p.parseImageResponse(ctx, task, size, shape, model, resp)
	}
	applyOpenAIRequestMetrics(&result, started, firstByteAt)
	return result, parseErr
}

func (p OpenAICompatibleProvider) generateEdit(ctx context.Context, task domain.Task, size, outputFormat string, input *resolvedEditInput) (Result, error) {
	model := p.modelForTask(task, openAICompatibleAPIModeImages)
	requestN := max(1, min(task.RequestedCount, taskProviderMaxN(task, OpenAICompatibleProviderID, openAICompatibleDefaultMaxN)))
	shape := openAICompatibleRequestShapeForTask(task, openAICompatibleEndpointEdits, "edit", requestN)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", model); err != nil {
		return Result{}, err
	}
	if err := writer.WriteField("prompt", task.Prompt); err != nil {
		return Result{}, err
	}
	if err := writer.WriteField("size", size); err != nil {
		return Result{}, err
	}
	if shape.ResponseFormat == openAICompatibleResponseFormatB64 {
		if err := writer.WriteField("response_format", openAICompatibleResponseFormatB64); err != nil {
			return Result{}, err
		}
	}
	if shape.Stream {
		if err := writer.WriteField("stream", "true"); err != nil {
			return Result{}, err
		}
		if err := writer.WriteField("partial_images", fmt.Sprintf("%d", shape.PartialImages)); err != nil {
			return Result{}, err
		}
	}
	if err := writer.WriteField("output_format", outputFormat); err != nil {
		return Result{}, err
	}
	for key, value := range openAICompatiblePassthroughParams(task) {
		if err := writer.WriteField(key, fmt.Sprint(value)); err != nil {
			return Result{}, err
		}
	}
	if requestN > 1 {
		if err := writer.WriteField("n", fmt.Sprintf("%d", requestN)); err != nil {
			return Result{}, err
		}
	}
	for index, item := range input.ReferenceImages {
		fileBytes, mimeType, err := p.readEditInputFile(item.FilePath, item.MimeType, input.MaskImage != nil && index == 0)
		if err != nil {
			return Result{}, err
		}
		if err := writeMultipartImageFile(writer, "image[]", fileBytes, fmt.Sprintf("input-%d%s", index+1, fileExtensionForMultipartMime(mimeType))); err != nil {
			return Result{}, err
		}
	}
	if input.MaskImage != nil {
		maskBytes, _, err := p.readEditInputFile(input.MaskImage.FilePath, input.MaskImage.MimeType, true)
		if err != nil {
			return Result{}, err
		}
		if err := writeMultipartImageFile(writer, "mask", maskBytes, "mask.png"); err != nil {
			return Result{}, err
		}
	}
	if err := writer.Close(); err != nil {
		return Result{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+openAICompatibleEndpointEdits, &body)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	started := time.Now()
	var firstByteAt time.Time
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			firstByteAt = time.Now()
		},
	}))
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return openAIRequestErrorResult(task, started, firstByteAt, err), err
	}
	var result Result
	var parseErr error
	if shape.Stream && isOpenAIEventStreamResponse(resp) {
		result, parseErr = p.parseImagesStreamResponse(ctx, task, size, shape, model, resp)
	} else {
		result, parseErr = p.parseImageResponse(ctx, task, size, shape, model, resp)
	}
	applyOpenAIRequestMetrics(&result, started, firstByteAt)
	return result, parseErr
}

func (p OpenAICompatibleProvider) generateResponses(ctx context.Context, task domain.Task, size, outputFormat string, input *resolvedEditInput) (Result, error) {
	model := p.modelForTask(task, openAICompatibleAPIModeResponses)
	requestN := max(1, min(task.RequestedCount, taskProviderMaxN(task, OpenAICompatibleProviderID, openAICompatibleDefaultMaxN)))
	shape := openAICompatibleRequestShapeForTask(task, openAICompatibleEndpointResponses, "responses", requestN)
	shape.APIMode = openAICompatibleAPIModeResponses
	shape.Endpoint = openAICompatibleEndpointResponses
	shape.RequestMode = OpenAICompatibleRequestModeResponsesStream
	if !shape.Stream {
		shape.Stream = true
	}
	if shape.PartialImages < 0 || shape.PartialImages > 3 {
		shape.PartialImages = clampInt(shape.PartialImages, 0, 3)
	}

	body, err := p.responsesRequestBody(task, model, size, outputFormat, shape, input)
	if err != nil {
		return Result{}, err
	}
	requestBytes, err := json.Marshal(body)
	if err != nil {
		return Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+openAICompatibleEndpointResponses, bytes.NewReader(requestBytes))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	started := time.Now()
	var firstByteAt time.Time
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			firstByteAt = time.Now()
		},
	}))
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return openAIRequestErrorResult(task, started, firstByteAt, err), err
	}
	var result Result
	var parseErr error
	if shape.Stream && isOpenAIEventStreamResponse(resp) {
		result, parseErr = p.parseResponsesStreamResponse(ctx, task, size, shape, model, resp)
	} else {
		result, parseErr = p.parseResponsesResponse(ctx, task, size, shape, model, resp)
	}
	applyOpenAIRequestMetrics(&result, started, firstByteAt)
	return result, parseErr
}

func (p OpenAICompatibleProvider) responsesRequestBody(task domain.Task, model, size, outputFormat string, shape openAICompatibleRequestShape, input *resolvedEditInput) (map[string]any, error) {
	tool := map[string]any{
		"type":          "image_generation",
		"action":        "generate",
		"size":          size,
		"output_format": outputFormat,
	}
	for key, value := range openAICompatiblePassthroughParams(task) {
		if key == "quality" || key == "moderation" || key == "output_compression" {
			tool[key] = value
		}
	}
	if shape.PartialImages > 0 {
		tool["partial_images"] = shape.PartialImages
	}
	inputPayload := any("Use the following text as the complete prompt. Do not rewrite it:\n" + task.Prompt)
	if input != nil && len(input.ReferenceImages) > 0 {
		tool["action"] = "auto"
		content := []map[string]any{{"type": "input_text", "text": inputPayload}}
		for index, item := range input.ReferenceImages {
			fileBytes, mimeType, err := p.readEditInputFile(item.FilePath, item.MimeType, input.MaskImage != nil && index == 0)
			if err != nil {
				return nil, err
			}
			content = append(content, map[string]any{
				"type":      "input_image",
				"image_url": dataURLForImageBytes(fileBytes, mimeType),
			})
		}
		if input.MaskImage != nil {
			maskBytes, mimeType, err := p.readEditInputFile(input.MaskImage.FilePath, input.MaskImage.MimeType, true)
			if err != nil {
				return nil, err
			}
			tool["input_image_mask"] = map[string]any{
				"image_url": dataURLForImageBytes(maskBytes, mimeType),
			}
		}
		inputPayload = []map[string]any{{
			"role":    "user",
			"content": content,
		}}
	}
	body := map[string]any{
		"model":       model,
		"input":       inputPayload,
		"tools":       []map[string]any{tool},
		"tool_choice": "required",
	}
	if shape.Stream {
		body["stream"] = true
	}
	return body, nil
}

func (p OpenAICompatibleProvider) parseImageResponse(ctx context.Context, task domain.Task, size string, shape openAICompatibleRequestShape, model string, resp *http.Response) (Result, error) {
	defer resp.Body.Close()
	downloadStarted := time.Now()
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxImageResponseBytes))
	if err != nil {
		return Result{
			Status:       "failed",
			CostRaw:      []byte(`{"provider":"openai-compatible"}`),
			ErrorCode:    "response_download_failed",
			ErrorMessage: err.Error(),
			Metrics: domain.AttemptMetrics{
				ResponseDownloadMs: time.Since(downloadStarted).Milliseconds(),
				ErrorStage:         "response_download",
			},
		}, err
	}
	result := Result{
		ProviderRequestID: resp.Header.Get("X-Request-Id"),
		Status:            "received",
		RawResponse:       respBytes,
		CostRaw:           []byte(`{"provider":"openai-compatible"}`),
		Metrics: domain.AttemptMetrics{
			ResponseDownloadMs: time.Since(downloadStarted).Milliseconds(),
			ResponseBytes:      int64(len(respBytes)),
		},
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		code := fmt.Sprintf("http_%d", resp.StatusCode)
		message := responseErrorMessage(respBytes, resp.Status)
		result.ErrorCode = code
		result.ErrorMessage = message
		result.Metrics.ErrorStage = "provider_response"
		return result, fmt.Errorf("openai-compatible provider failed: %s", message)
	}

	var payload openAICompatibleResponse
	if err := json.Unmarshal(respBytes, &payload); err != nil {
		result.ErrorCode = "invalid_response"
		result.ErrorMessage = err.Error()
		result.Metrics.ErrorStage = "response_parse"
		return result, fmt.Errorf("parse openai-compatible response: %w", err)
	}
	return p.parseImagePayload(ctx, task, size, shape, model, payload, result)
}

func (p OpenAICompatibleProvider) parseImagePayload(ctx context.Context, task domain.Task, size string, shape openAICompatibleRequestShape, model string, payload openAICompatibleResponse, result Result) (Result, error) {
	if payload.Error != nil && payload.Error.Message != "" {
		result.ErrorCode = firstNonEmpty(payload.Error.Code, "provider_error")
		result.ErrorMessage = payload.Error.Message
		result.Metrics.ErrorStage = "provider_response"
		return result, fmt.Errorf("openai-compatible provider error: %s", payload.Error.Message)
	}
	if payload.ID != "" {
		result.ProviderRequestID = payload.ID
	}
	if result.ProviderRequestID == "" {
		result.ProviderRequestID = "openai_compatible_" + task.ID
	}
	if payload.Usage != nil {
		if costBytes, err := json.Marshal(map[string]any{
			"provider": "openai-compatible",
			"usage":    payload.Usage,
		}); err == nil {
			result.CostRaw = costBytes
		}
	}

	files := make([]GeneratedFile, 0, len(payload.Data))
	for i, item := range payload.Data {
		imageBytes, resultKind, err := p.imageBytes(ctx, item)
		if err != nil {
			result.ErrorCode = "image_decode_failed"
			result.ErrorMessage = err.Error()
			result.Metrics.ErrorStage = "response_download"
			return result, err
		}
		pngBytes, width, height, err := normalizePNG(imageBytes)
		if err != nil {
			result.ErrorCode = "unsupported_image"
			result.ErrorMessage = err.Error()
			result.Metrics.ErrorStage = "response_parse"
			return result, err
		}
		parameterBytes := taskProviderParameters(task, map[string]any{
			"provider":        "openai-compatible",
			"model":           model,
			"api_mode":        shape.APIMode,
			"endpoint":        shape.Endpoint,
			"request_mode":    shape.RequestMode,
			"operation":       shape.Operation,
			"stream":          shape.Stream,
			"partial_images":  shape.PartialImages,
			"partial_count":   shape.PartialCount,
			"response_format": shape.ResponseFormat,
			"result_kind":     resultKind,
			"n":               shape.N,
			"slot":            i,
			"size":            size,
			"aspect_ratio":    task.AspectRatio,
			"output_format":   "png",
			"revised_prompt":  item.RevisedPrompt,
		})
		files = append(files, GeneratedFile{
			Slot:          i,
			Bytes:         pngBytes,
			Thumbnail:     pngBytes,
			MimeType:      "image/png",
			Width:         width,
			Height:        height,
			ThumbnailW:    width,
			ThumbnailH:    height,
			Model:         model,
			ParametersRaw: parameterBytes,
			CostRaw:       result.CostRaw,
		})
	}
	if len(files) == 0 {
		result.ErrorCode = "empty_response"
		result.ErrorMessage = "openai-compatible provider returned no image data"
		result.Metrics.ErrorStage = "response_parse"
		return result, errors.New(result.ErrorMessage)
	}
	result.Status = "succeeded"
	result.Files = files
	return result, nil
}

func (p OpenAICompatibleProvider) parseImagesStreamResponse(ctx context.Context, task domain.Task, size string, shape openAICompatibleRequestShape, model string, resp *http.Response) (Result, error) {
	defer resp.Body.Close()
	started := time.Now()
	result := Result{
		ProviderRequestID: resp.Header.Get("X-Request-Id"),
		Status:            "received",
		CostRaw:           []byte(`{"provider":"openai-compatible"}`),
		Metrics:           domain.AttemptMetrics{},
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxImageResponseBytes))
		result.RawResponse = respBytes
		result.Metrics.ResponseDownloadMs = time.Since(started).Milliseconds()
		result.Metrics.ResponseBytes = int64(len(respBytes))
		result.ErrorCode = fmt.Sprintf("http_%d", resp.StatusCode)
		result.ErrorMessage = responseErrorMessage(respBytes, resp.Status)
		result.Metrics.ErrorStage = "provider_response"
		return result, fmt.Errorf("openai-compatible provider failed: %s", result.ErrorMessage)
	}

	completedItems := []openAICompatibleDataItem{}
	var resultPayload *openAICompatibleResponse
	summary := openAIStreamSummary{APIMode: shape.APIMode, RequestMode: shape.RequestMode, Endpoint: shape.Endpoint}
	err := readOpenAIJSONServerSentEvents(resp.Body, func(raw []byte, event map[string]any) error {
		summary.EventCount++
		summary.ResponseBytes += int64(len(raw))
		eventType := stringValue(event, "type")
		if eventType != "" {
			summary.EventTypes = appendUniqueString(summary.EventTypes, eventType)
		}
		if message := streamEventErrorMessage(event); message != "" {
			result.ErrorCode = firstNonEmpty(stringValue(event, "code"), "provider_stream_error")
			result.ErrorMessage = message
			return fmt.Errorf("openai-compatible stream failed: %s", message)
		}
		if eventType == "image_generation.partial_image" || eventType == "image_edit.partial_image" {
			if stringValue(event, "b64_json") != "" {
				summary.PartialImageCount++
			}
			return nil
		}
		if object := stringValue(event, "object"); object == "image.generation.result" || object == "image.edit.result" {
			var payload openAICompatibleResponse
			if err := json.Unmarshal(raw, &payload); err != nil {
				return err
			}
			resultPayload = &payload
			return nil
		}
		if eventType == "image_generation.completed" || eventType == "image_edit.completed" {
			item := openAICompatibleDataItem{
				B64JSON:       stringValue(event, "b64_json"),
				URL:           stringValue(event, "url"),
				RevisedPrompt: stringValue(event, "revised_prompt"),
			}
			if item.B64JSON != "" || item.URL != "" {
				completedItems = append(completedItems, item)
			}
			return nil
		}
		return nil
	})
	result.Metrics.ResponseDownloadMs = time.Since(started).Milliseconds()
	result.Metrics.ResponseBytes = summary.ResponseBytes
	if raw, marshalErr := json.Marshal(summary); marshalErr == nil {
		result.RawResponse = raw
	}
	if err != nil {
		if result.ErrorCode == "" {
			result.ErrorCode = "provider_stream_failed"
		}
		if result.ErrorMessage == "" {
			result.ErrorMessage = err.Error()
		}
		result.Metrics.ErrorStage = "provider_stream"
		return result, err
	}

	payload := openAICompatibleResponse{Data: completedItems}
	if resultPayload != nil {
		payload = *resultPayload
	}
	if payload.ID != "" {
		result.ProviderRequestID = payload.ID
	}
	if result.ProviderRequestID == "" {
		result.ProviderRequestID = "openai_compatible_" + task.ID
	}
	shape.PartialCount = summary.PartialImageCount
	parsed, parseErr := p.parseImagePayload(ctx, task, size, shape, model, payload, result)
	if parseErr != nil && parsed.Metrics.ErrorStage == "" {
		parsed.Metrics.ErrorStage = "provider_stream"
	}
	return parsed, parseErr
}

func (p OpenAICompatibleProvider) parseResponsesResponse(ctx context.Context, task domain.Task, size string, shape openAICompatibleRequestShape, model string, resp *http.Response) (Result, error) {
	defer resp.Body.Close()
	downloadStarted := time.Now()
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxImageResponseBytes))
	if err != nil {
		return Result{
			Status:       "failed",
			CostRaw:      []byte(`{"provider":"openai-compatible"}`),
			ErrorCode:    "response_download_failed",
			ErrorMessage: err.Error(),
			Metrics: domain.AttemptMetrics{
				ResponseDownloadMs: time.Since(downloadStarted).Milliseconds(),
				ErrorStage:         "response_download",
			},
		}, err
	}
	result := Result{
		ProviderRequestID: resp.Header.Get("X-Request-Id"),
		Status:            "received",
		RawResponse:       respBytes,
		CostRaw:           []byte(`{"provider":"openai-compatible"}`),
		Metrics: domain.AttemptMetrics{
			ResponseDownloadMs: time.Since(downloadStarted).Milliseconds(),
			ResponseBytes:      int64(len(respBytes)),
		},
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.ErrorCode = fmt.Sprintf("http_%d", resp.StatusCode)
		result.ErrorMessage = responseErrorMessage(respBytes, resp.Status)
		result.Metrics.ErrorStage = "provider_response"
		return result, fmt.Errorf("openai-compatible provider failed: %s", result.ErrorMessage)
	}
	var payload openAIResponsesResponse
	if err := json.Unmarshal(respBytes, &payload); err != nil {
		result.ErrorCode = "invalid_response"
		result.ErrorMessage = err.Error()
		result.Metrics.ErrorStage = "response_parse"
		return result, fmt.Errorf("parse openai-compatible responses payload: %w", err)
	}
	return p.parseResponsesPayload(ctx, task, size, shape, model, payload, result)
}

func (p OpenAICompatibleProvider) parseResponsesStreamResponse(ctx context.Context, task domain.Task, size string, shape openAICompatibleRequestShape, model string, resp *http.Response) (Result, error) {
	defer resp.Body.Close()
	started := time.Now()
	result := Result{
		ProviderRequestID: resp.Header.Get("X-Request-Id"),
		Status:            "received",
		CostRaw:           []byte(`{"provider":"openai-compatible"}`),
		Metrics:           domain.AttemptMetrics{},
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxImageResponseBytes))
		result.RawResponse = respBytes
		result.Metrics.ResponseDownloadMs = time.Since(started).Milliseconds()
		result.Metrics.ResponseBytes = int64(len(respBytes))
		result.ErrorCode = fmt.Sprintf("http_%d", resp.StatusCode)
		result.ErrorMessage = responseErrorMessage(respBytes, resp.Status)
		result.Metrics.ErrorStage = "provider_response"
		return result, fmt.Errorf("openai-compatible provider failed: %s", result.ErrorMessage)
	}

	outputItems := []openAIResponsesOutputItem{}
	var completedPayload *openAIResponsesResponse
	summary := openAIStreamSummary{APIMode: shape.APIMode, RequestMode: shape.RequestMode, Endpoint: shape.Endpoint}
	err := readOpenAIJSONServerSentEvents(resp.Body, func(raw []byte, event map[string]any) error {
		summary.EventCount++
		summary.ResponseBytes += int64(len(raw))
		eventType := stringValue(event, "type")
		if eventType != "" {
			summary.EventTypes = appendUniqueString(summary.EventTypes, eventType)
		}
		if message := streamEventErrorMessage(event); message != "" {
			result.ErrorCode = firstNonEmpty(stringValue(event, "code"), "provider_stream_error")
			result.ErrorMessage = message
			return fmt.Errorf("openai-compatible stream failed: %s", message)
		}
		if eventType == "response.image_generation_call.partial_image" {
			if stringValue(event, "partial_image_b64") != "" {
				summary.PartialImageCount++
			}
			return nil
		}
		payload, err := responsesPayloadFromStreamEvent(raw, event)
		if err != nil || payload == nil {
			return err
		}
		if eventType == "response.output_item.done" && len(payload.Output) > 0 {
			outputItems = append(outputItems, payload.Output...)
			return nil
		}
		if eventType == "response.completed" || event["response"] != nil {
			completedPayload = payload
		}
		return nil
	})
	result.Metrics.ResponseDownloadMs = time.Since(started).Milliseconds()
	result.Metrics.ResponseBytes = summary.ResponseBytes
	if raw, marshalErr := json.Marshal(summary); marshalErr == nil {
		result.RawResponse = raw
	}
	if err != nil {
		if result.ErrorCode == "" {
			result.ErrorCode = "provider_stream_failed"
		}
		if result.ErrorMessage == "" {
			result.ErrorMessage = err.Error()
		}
		result.Metrics.ErrorStage = "provider_stream"
		return result, err
	}
	payload := openAIResponsesResponse{Output: outputItems}
	if completedPayload != nil {
		payload = *completedPayload
		if len(payload.Output) == 0 && len(outputItems) > 0 {
			payload.Output = outputItems
		}
	}
	shape.PartialCount = summary.PartialImageCount
	parsed, parseErr := p.parseResponsesPayload(ctx, task, size, shape, model, payload, result)
	if parseErr != nil && parsed.Metrics.ErrorStage == "" {
		parsed.Metrics.ErrorStage = "provider_stream"
	}
	return parsed, parseErr
}

func (p OpenAICompatibleProvider) parseResponsesPayload(ctx context.Context, task domain.Task, size string, shape openAICompatibleRequestShape, model string, payload openAIResponsesResponse, result Result) (Result, error) {
	if payload.ID != "" {
		result.ProviderRequestID = payload.ID
	}
	if result.ProviderRequestID == "" {
		result.ProviderRequestID = "openai_compatible_" + task.ID
	}
	if payload.Usage != nil {
		if costBytes, err := json.Marshal(map[string]any{
			"provider": "openai-compatible",
			"usage":    payload.Usage,
		}); err == nil {
			result.CostRaw = costBytes
		}
	}
	imagePayload := openAICompatibleResponse{ID: payload.ID, Data: responsesOutputToImageData(payload.Output)}
	return p.parseImagePayload(ctx, task, size, shape, model, imagePayload, result)
}

func responsesPayloadFromStreamEvent(raw []byte, event map[string]any) (*openAIResponsesResponse, error) {
	if response, ok := event["response"].(map[string]any); ok {
		rawResponse, err := json.Marshal(response)
		if err != nil {
			return nil, err
		}
		var payload openAIResponsesResponse
		if err := json.Unmarshal(rawResponse, &payload); err != nil {
			return nil, err
		}
		return &payload, nil
	}
	if item, ok := event["item"].(map[string]any); ok {
		rawItem, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		var outputItem openAIResponsesOutputItem
		if err := json.Unmarshal(rawItem, &outputItem); err != nil {
			return nil, err
		}
		return &openAIResponsesResponse{Output: []openAIResponsesOutputItem{outputItem}}, nil
	}
	var payload openAIResponsesResponse
	if err := json.Unmarshal(raw, &payload); err == nil && len(payload.Output) > 0 {
		return &payload, nil
	}
	return nil, nil
}

func responsesOutputToImageData(output []openAIResponsesOutputItem) []openAICompatibleDataItem {
	items := []openAICompatibleDataItem{}
	for _, item := range output {
		if item.Type != "image_generation_call" {
			continue
		}
		b64 := responsesResultBase64(item.Result)
		if b64 == "" {
			continue
		}
		items = append(items, openAICompatibleDataItem{
			B64JSON:       b64,
			RevisedPrompt: item.RevisedPrompt,
		})
	}
	return items
}

func responsesResultBase64(result json.RawMessage) string {
	if len(result) == 0 {
		return ""
	}
	var asString string
	if json.Unmarshal(result, &asString) == nil {
		return strings.TrimSpace(asString)
	}
	var asObject map[string]any
	if json.Unmarshal(result, &asObject) != nil {
		return ""
	}
	for _, key := range []string{"b64_json", "base64", "image", "data"} {
		if value, ok := asObject[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type openAIStreamSummary struct {
	APIMode           string   `json:"api_mode,omitempty"`
	RequestMode       string   `json:"request_mode,omitempty"`
	Endpoint          string   `json:"endpoint,omitempty"`
	EventCount        int      `json:"event_count"`
	EventTypes        []string `json:"event_types,omitempty"`
	PartialImageCount int      `json:"partial_image_count,omitempty"`
	ResponseBytes     int64    `json:"response_bytes"`
}

func readOpenAIJSONServerSentEvents(r io.Reader, onEvent func(raw []byte, event map[string]any) error) error {
	reader := bufio.NewReader(r)
	var block bytes.Buffer
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			trimmed := strings.TrimRight(string(line), "\r\n")
			if trimmed == "" {
				if err := processOpenAIEventBlock(block.Bytes(), onEvent); err != nil {
					return err
				}
				block.Reset()
			} else {
				block.Write(line)
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				if block.Len() > 0 {
					return processOpenAIEventBlock(block.Bytes(), onEvent)
				}
				return nil
			}
			return err
		}
	}
}

func processOpenAIEventBlock(block []byte, onEvent func(raw []byte, event map[string]any) error) error {
	dataLines := []string{}
	for _, line := range strings.Split(string(block), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" || strings.HasPrefix(line, ":") || !strings.HasPrefix(line, "data:") {
			continue
		}
		dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
	}
	data := strings.TrimSpace(strings.Join(dataLines, "\n"))
	if data == "" || data == "[DONE]" {
		return nil
	}
	raw := []byte(data)
	var event map[string]any
	if err := json.Unmarshal(raw, &event); err != nil {
		return fmt.Errorf("parse openai-compatible stream event: %w", err)
	}
	return onEvent(raw, event)
}

func (p OpenAICompatibleProvider) readEditInputFile(path, mimeType string, forcePNG bool) ([]byte, string, error) {
	return readTaskInputFile(path, mimeType, forcePNG)
}

func (p OpenAICompatibleProvider) imageBytes(ctx context.Context, item openAICompatibleDataItem) ([]byte, string, error) {
	if item.B64JSON != "" {
		raw, err := base64.StdEncoding.DecodeString(stripDataURLPrefix(item.B64JSON))
		return raw, openAICompatibleResponseFormatB64, err
	}
	if item.URL == "" {
		return nil, "", fmt.Errorf("response item has neither b64_json nor url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return nil, openAICompatibleResponseFormatURL, err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, openAICompatibleResponseFormatURL, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, openAICompatibleResponseFormatURL, fmt.Errorf("download image failed: %s", resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxImageResponseBytes))
	return raw, openAICompatibleResponseFormatURL, err
}

type resolvedEditInput struct {
	ReferenceImages []resolvedTaskInputFile
	MaskImage       *resolvedTaskInputFile
}

func resolveStructuredEditInput(input taskStructuredProviderInput) (*resolvedEditInput, error) {
	if input.ResolvedInputFiles == nil {
		return nil, nil
	}
	if len(input.ResolvedInputFiles.ReferenceImages) == 0 {
		if input.ResolvedInputFiles.MaskImage != nil {
			return nil, fmt.Errorf("mask image requires at least one resolved input image")
		}
		return nil, nil
	}
	return &resolvedEditInput{
		ReferenceImages: input.ResolvedInputFiles.ReferenceImages,
		MaskImage:       input.ResolvedInputFiles.MaskImage,
	}, nil
}

func readTaskInputFile(path, mimeType string, forcePNG bool) ([]byte, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	if !forcePNG && strings.EqualFold(strings.TrimSpace(mimeType), "image/png") {
		return raw, "image/png", nil
	}
	if !forcePNG && strings.HasPrefix(strings.ToLower(strings.TrimSpace(mimeType)), "image/") {
		return raw, mimeType, nil
	}
	pngBytes, _, _, err := normalizePNG(raw)
	if err != nil {
		return nil, "", err
	}
	return pngBytes, "image/png", nil
}

func normalizePNG(raw []byte) ([]byte, int, int, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, 0, 0, err
	}
	bounds := img.Bounds()
	return buf.Bytes(), bounds.Dx(), bounds.Dy(), nil
}

func writeMultipartImageFile(writer *multipart.Writer, field string, raw []byte, filename string) error {
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		return err
	}
	_, err = part.Write(raw)
	return err
}

func dataURLForImageBytes(raw []byte, mimeType string) string {
	mimeType = strings.TrimSpace(mimeType)
	if mimeType == "" {
		mimeType = "image/png"
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(raw)
}

func fileExtensionForMultipartMime(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}

func isOpenAIEventStreamResponse(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	return strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream")
}

func openAICompatiblePassthroughParams(task domain.Task) map[string]any {
	input := parseTaskStructuredProviderInput(task)
	if len(input.GenerationConfig) == 0 {
		return nil
	}
	var config map[string]any
	if err := json.Unmarshal(input.GenerationConfig, &config); err != nil {
		return nil
	}
	params := map[string]any{}
	if value := trimmedConfigString(config, "quality"); value != "" {
		params["quality"] = value
	}
	if value := trimmedConfigString(config, "moderation"); value != "" {
		params["moderation"] = value
	}
	if value, ok := configIntInRange(config, "output_compression", 0, 100); ok {
		params["output_compression"] = value
	}
	if len(params) == 0 {
		return nil
	}
	return params
}

func trimmedConfigString(config map[string]any, key string) string {
	value, ok := config[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func configIntInRange(config map[string]any, key string, minValue, maxValue int) (int, bool) {
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

func configBool(config map[string]any, key string) (bool, bool) {
	value, ok := config[key]
	if !ok || value == nil {
		return false, false
	}
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true":
			return true, true
		case "false":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func stringValue(source map[string]any, key string) string {
	value, ok := source[key]
	if !ok || value == nil {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func streamEventErrorMessage(event map[string]any) string {
	if raw, ok := event["error"]; ok {
		if message, ok := raw.(string); ok && strings.TrimSpace(message) != "" {
			return strings.TrimSpace(message)
		}
		if nested, ok := raw.(map[string]any); ok {
			if message := stringValue(nested, "message"); message != "" {
				return message
			}
		}
	}
	eventType := stringValue(event, "type")
	if strings.HasSuffix(eventType, ".failed") {
		if message := stringValue(event, "message"); message != "" {
			return message
		}
		return "stream event failed"
	}
	return ""
}

func appendUniqueString(values []string, next string) []string {
	if next == "" {
		return values
	}
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}

func sizeForAspectRatio(aspectRatio string) string {
	switch aspectRatio {
	case "3:4", "9:16":
		return "1024x1536"
	case "4:3", "16:9":
		return "1536x1024"
	default:
		return "1024x1024"
	}
}

func responseErrorMessage(raw []byte, fallback string) string {
	var payload openAICompatibleResponse
	if err := json.Unmarshal(raw, &payload); err == nil && payload.Error != nil && payload.Error.Message != "" {
		return payload.Error.Message
	}
	text := strings.TrimSpace(string(raw))
	if text != "" && len(text) <= 500 {
		return text
	}
	return fallback
}

func stripDataURLPrefix(value string) string {
	if comma := strings.Index(value, ","); strings.HasPrefix(value, "data:") && comma >= 0 {
		return value[comma+1:]
	}
	return value
}

func openAIRequestErrorResult(task domain.Task, started time.Time, firstByteAt time.Time, err error) Result {
	metrics := domain.AttemptMetrics{
		ProviderTotalMs: time.Since(started).Milliseconds(),
		ErrorStage:      classifyOpenAIRequestErrorStage(err),
	}
	if !firstByteAt.IsZero() {
		metrics.ProviderFirstByteMs = firstByteAt.Sub(started).Milliseconds()
	}
	return Result{
		ProviderRequestID: "openai_compatible_" + task.ID,
		Status:            "failed",
		CostRaw:           []byte(`{"provider":"openai-compatible"}`),
		ErrorCode:         "provider_request_failed",
		ErrorMessage:      err.Error(),
		Metrics:           metrics,
	}
}

func applyOpenAIRequestMetrics(result *Result, started time.Time, firstByteAt time.Time) {
	if result == nil {
		return
	}
	if !firstByteAt.IsZero() {
		result.Metrics.ProviderFirstByteMs = firstByteAt.Sub(started).Milliseconds()
	} else if result.Metrics.ProviderFirstByteMs == 0 {
		result.Metrics.ProviderFirstByteMs = time.Since(started).Milliseconds()
	}
	result.Metrics.ProviderTotalMs = time.Since(started).Milliseconds()
}

func classifyOpenAIRequestErrorStage(err error) string {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "awaiting headers") ||
		strings.Contains(message, "response header") ||
		strings.Contains(message, "first byte") {
		return "provider_first_byte"
	}
	if strings.Contains(message, "dial") ||
		strings.Contains(message, "connect") ||
		strings.Contains(message, "no such host") ||
		strings.Contains(message, "connection refused") {
		return "connect"
	}
	if strings.Contains(message, "context deadline") ||
		strings.Contains(message, "client.timeout") ||
		strings.Contains(message, "timeout") {
		return "provider_total"
	}
	return "provider_request"
}

func providerTimeoutDuration(seconds int) time.Duration {
	if seconds < 1 {
		seconds = 1
	}
	return time.Duration(seconds) * time.Second
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 1
}

func firstNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

type openAICompatibleResponse struct {
	ID      string                     `json:"id"`
	Created int64                      `json:"created"`
	Data    []openAICompatibleDataItem `json:"data"`
	Usage   any                        `json:"usage"`
	Error   *openAICompatibleError     `json:"error"`
}

type openAIResponsesResponse struct {
	ID     string                      `json:"id"`
	Output []openAIResponsesOutputItem `json:"output"`
	Usage  any                         `json:"usage"`
}

type openAIResponsesOutputItem struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Result        json.RawMessage `json:"result"`
	RevisedPrompt string          `json:"revised_prompt"`
}

type openAICompatibleDataItem struct {
	B64JSON       string `json:"b64_json"`
	URL           string `json:"url"`
	RevisedPrompt string `json:"revised_prompt"`
}

type openAICompatibleError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}
