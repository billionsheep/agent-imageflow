package provider

import (
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

	openAICompatibleAPIModeImages       = "images"
	openAICompatibleEndpointGenerations = "/images/generations"
	openAICompatibleEndpointEdits       = "/images/edits"
	openAICompatibleResponseFormatB64   = "b64_json"
	openAICompatibleResponseFormatURL   = "url"
	openAICompatibleResponseFormatOmit  = "omitted"
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
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
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
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     strings.TrimSpace(cfg.APIKey),
		model:      firstNonEmpty(cfg.Model, "gpt-image-2"),
		httpClient: client,
	}
}

func (p OpenAICompatibleProvider) Configured() bool {
	return p.baseURL != "" && p.apiKey != "" && p.model != ""
}

func openAICompatibleRequestShapeForTask(task domain.Task, endpoint, operation string, requestN int) openAICompatibleRequestShape {
	responseFormat := openAICompatiblePreferredResponseFormat(task)
	requestMode := OpenAICompatibleRequestModeImagesSyncURL
	recordedResponseFormat := openAICompatibleResponseFormatOmit
	if responseFormat == openAICompatibleResponseFormatB64 {
		requestMode = OpenAICompatibleRequestModeImagesSyncB64
		recordedResponseFormat = openAICompatibleResponseFormatB64
	}
	return openAICompatibleRequestShape{
		APIMode:        openAICompatibleAPIModeImages,
		Endpoint:       endpoint,
		Operation:      operation,
		RequestMode:    requestMode,
		ResponseFormat: recordedResponseFormat,
		Stream:         false,
		PartialImages:  0,
		N:              requestN,
	}
}

func openAICompatiblePreferredResponseFormat(task domain.Task) string {
	input := parseTaskStructuredProviderInput(task)
	if input.ProviderProfile.Enabled &&
		strings.TrimSpace(input.ProviderProfile.Provider) == OpenAICompatibleProviderID &&
		strings.TrimSpace(input.ProviderProfile.PreferredResponseFormat) == openAICompatibleResponseFormatB64 {
		return openAICompatibleResponseFormatB64
	}
	return openAICompatibleResponseFormatURL
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
	if editInput != nil {
		return p.generateEdit(ctx, task, size, outputFormat, editInput)
	}
	return p.generateGeneration(ctx, task, size, outputFormat)
}

func (p OpenAICompatibleProvider) generateGeneration(ctx context.Context, task domain.Task, size, outputFormat string) (Result, error) {
	model := taskProviderModel(task, OpenAICompatibleProviderID, p.model)
	requestN := max(1, min(task.RequestedCount, taskProviderMaxN(task, OpenAICompatibleProviderID, 4)))
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
	result, parseErr := p.parseImageResponse(ctx, task, size, shape, model, resp)
	applyOpenAIRequestMetrics(&result, started, firstByteAt)
	return result, parseErr
}

func (p OpenAICompatibleProvider) generateEdit(ctx context.Context, task domain.Task, size, outputFormat string, input *resolvedEditInput) (Result, error) {
	model := taskProviderModel(task, OpenAICompatibleProviderID, p.model)
	requestN := max(1, min(task.RequestedCount, taskProviderMaxN(task, OpenAICompatibleProviderID, 4)))
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
	result, parseErr := p.parseImageResponse(ctx, task, size, shape, model, resp)
	applyOpenAIRequestMetrics(&result, started, firstByteAt)
	return result, parseErr
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
