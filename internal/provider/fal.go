package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

const (
	defaultFalQueueBaseURL = "https://queue.fal.run"
	defaultFalRestBaseURL  = "https://rest.fal.ai"
	maxFalInputImages      = 4
)

type FalConfig struct {
	BaseURL             string
	RestBaseURL         string
	APIKey              string
	Model               string
	TimeoutSeconds      int
	PollIntervalMs      int
	StartTimeoutSeconds int
	HTTPClient          *http.Client
}

type FalProvider struct {
	baseURL             string
	restBaseURL         string
	apiKey              string
	model               string
	httpClient          *http.Client
	pollInterval        time.Duration
	startTimeoutSeconds int
}

func NewFalProvider(cfg FalConfig) FalProvider {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	pollInterval := time.Duration(cfg.PollIntervalMs) * time.Millisecond
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	return FalProvider{
		baseURL:             strings.TrimRight(firstNonEmpty(cfg.BaseURL, defaultFalQueueBaseURL), "/"),
		restBaseURL:         strings.TrimRight(firstNonEmpty(cfg.RestBaseURL, defaultFalRestBaseURL), "/"),
		apiKey:              strings.TrimSpace(cfg.APIKey),
		model:               firstNonEmpty(cfg.Model, "openai/gpt-image-2"),
		httpClient:          client,
		pollInterval:        pollInterval,
		startTimeoutSeconds: cfg.StartTimeoutSeconds,
	}
}

func (p FalProvider) Configured() bool {
	return p.baseURL != "" && p.apiKey != "" && p.model != ""
}

func (p FalProvider) Generate(ctx context.Context, task domain.Task) (Result, error) {
	if !p.Configured() {
		return Result{}, fmt.Errorf("fal provider is not configured")
	}

	input := parseTaskStructuredProviderInput(task)
	editInput, err := resolveStructuredEditInput(input)
	if err != nil {
		return Result{
			Status:       "failed",
			CostRaw:      []byte(`{"provider":"fal"}`),
			ErrorCode:    "invalid_edit_input",
			ErrorMessage: err.Error(),
		}, err
	}

	model := taskProviderModel(task, FalProviderID, p.model)
	endpointID := falEndpointID(model, editInput != nil)
	requestMode := "generation"
	if editInput != nil {
		requestMode = "edit"
	}
	requestInput, err := p.createRequestInput(ctx, task, editInput)
	if err != nil {
		return Result{
			Status:       "failed",
			CostRaw:      []byte(`{"provider":"fal"}`),
			ErrorCode:    "input_upload_failed",
			ErrorMessage: err.Error(),
		}, err
	}

	submitStatus, submitRaw, err := p.submit(ctx, endpointID, requestInput)
	if err != nil {
		return Result{
			Status:       "failed",
			RawResponse:  submitRaw,
			CostRaw:      []byte(`{"provider":"fal"}`),
			ErrorCode:    "submit_failed",
			ErrorMessage: err.Error(),
		}, err
	}

	finalStatus, statusRaw, err := p.waitForCompletion(ctx, endpointID, submitStatus.RequestID)
	if err != nil {
		return Result{
			ProviderRequestID: submitStatus.RequestID,
			Status:            "failed",
			RawResponse:       rawProviderEnvelope(submitRaw, statusRaw, nil),
			CostRaw:           falCostRaw(submitStatus.RequestID, nil),
			ErrorCode:         "queue_status_failed",
			ErrorMessage:      err.Error(),
		}, err
	}

	payload, resultRaw, err := p.result(ctx, endpointID, submitStatus.RequestID)
	if err != nil {
		return Result{
			ProviderRequestID: submitStatus.RequestID,
			Status:            "failed",
			RawResponse:       rawProviderEnvelope(submitRaw, statusRaw, resultRaw),
			CostRaw:           falCostRaw(submitStatus.RequestID, finalStatus.Metrics),
			ErrorCode:         "result_failed",
			ErrorMessage:      err.Error(),
		}, err
	}

	files, err := p.generatedFilesFromPayload(ctx, task, requestMode, model, endpointID, requestInput["image_size"], submitStatus.RequestID, finalStatus.Metrics, payload)
	if err != nil {
		return Result{
			ProviderRequestID: submitStatus.RequestID,
			Status:            "failed",
			RawResponse:       rawProviderEnvelope(submitRaw, statusRaw, resultRaw),
			CostRaw:           falCostRaw(submitStatus.RequestID, finalStatus.Metrics),
			ErrorCode:         "invalid_result",
			ErrorMessage:      err.Error(),
		}, err
	}

	return Result{
		ProviderRequestID: submitStatus.RequestID,
		Status:            "succeeded",
		Files:             files,
		RawResponse:       rawProviderEnvelope(submitRaw, statusRaw, resultRaw),
		CostRaw:           falCostRaw(submitStatus.RequestID, finalStatus.Metrics),
	}, nil
}

func (p FalProvider) createRequestInput(ctx context.Context, task domain.Task, editInput *resolvedEditInput) (map[string]any, error) {
	requestInput := map[string]any{
		"prompt":        task.Prompt,
		"image_size":    falImageSize(task.AspectRatio),
		"quality":       falQuality(task),
		"num_images":    max(1, min(task.RequestedCount, taskProviderMaxN(task, FalProviderID, maxFalInputImages))),
		"output_format": falOutputFormat(task.OutputFormat),
	}
	if task.NegativePrompt != "" {
		requestInput["negative_prompt"] = task.NegativePrompt
	}
	if editInput == nil {
		return requestInput, nil
	}

	imageURLs := make([]string, 0, min(len(editInput.ReferenceImages), maxFalInputImages))
	for index, item := range editInput.ReferenceImages {
		if index >= maxFalInputImages {
			break
		}
		uploadedURL, err := p.uploadInputFile(ctx, item, false)
		if err != nil {
			return nil, err
		}
		imageURLs = append(imageURLs, uploadedURL)
	}
	if len(imageURLs) == 0 {
		return nil, fmt.Errorf("fal edit requires at least one uploaded input image")
	}
	requestInput["image_urls"] = imageURLs
	if editInput.MaskImage != nil {
		maskURL, err := p.uploadInputFile(ctx, *editInput.MaskImage, true)
		if err != nil {
			return nil, err
		}
		requestInput["mask_url"] = maskURL
	}
	return requestInput, nil
}

func (p FalProvider) uploadInputFile(ctx context.Context, item resolvedTaskInputFile, forcePNG bool) (string, error) {
	fileBytes, mimeType, err := readTaskInputFile(item.FilePath, item.MimeType, forcePNG)
	if err != nil {
		return "", err
	}

	initiateURL := p.restBaseURL + "/storage/upload/initiate?storage_type=fal-cdn-v3"
	filename := falUploadFilename(item, mimeType)
	requestBody := map[string]any{
		"content_type": mimeType,
		"file_name":    filename,
	}
	raw, err := p.requestJSON(ctx, http.MethodPost, initiateURL, requestBody, falAuthHeaders(p.apiKey))
	if err != nil {
		return "", err
	}

	var upload falUploadInitiateResponse
	if err := json.Unmarshal(raw, &upload); err != nil {
		return "", fmt.Errorf("parse fal upload initiate response: %w", err)
	}
	if upload.UploadURL == "" || upload.FileURL == "" {
		return "", fmt.Errorf("fal upload initiate returned incomplete URLs")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, upload.UploadURL, bytes.NewReader(fileBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mimeType)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxImageResponseBytes))
		return "", fmt.Errorf("fal upload failed: %s", falErrorMessage(respBytes, resp.Status))
	}

	return upload.FileURL, nil
}

func (p FalProvider) submit(ctx context.Context, endpointID string, input map[string]any) (falQueueStatus, []byte, error) {
	headers := falAuthHeaders(p.apiKey)
	if p.startTimeoutSeconds > 1 {
		headers["X-Fal-Request-Timeout"] = fmt.Sprintf("%d", p.startTimeoutSeconds)
	}
	raw, err := p.requestJSON(ctx, http.MethodPost, p.queueURL(endpointID), input, headers)
	if err != nil {
		return falQueueStatus{}, raw, err
	}
	var status falQueueStatus
	if err := json.Unmarshal(raw, &status); err != nil {
		return falQueueStatus{}, raw, fmt.Errorf("parse fal submit response: %w", err)
	}
	if status.RequestID == "" {
		return falQueueStatus{}, raw, fmt.Errorf("fal submit response missing request_id")
	}
	return status, raw, nil
}

func (p FalProvider) waitForCompletion(ctx context.Context, endpointID, requestID string) (falQueueStatus, []byte, error) {
	statusURL := p.queueURL(endpointID) + "/requests/" + url.PathEscape(requestID) + "/status?logs=1"
	for {
		raw, err := p.requestJSON(ctx, http.MethodGet, statusURL, nil, falAuthHeaders(p.apiKey))
		if err != nil {
			return falQueueStatus{}, raw, err
		}
		var status falQueueStatus
		if err := json.Unmarshal(raw, &status); err != nil {
			return falQueueStatus{}, raw, fmt.Errorf("parse fal queue status response: %w", err)
		}
		if strings.EqualFold(status.Status, "COMPLETED") {
			return status, raw, nil
		}
		select {
		case <-ctx.Done():
			return falQueueStatus{}, raw, ctx.Err()
		case <-time.After(p.pollInterval):
		}
	}
}

func (p FalProvider) result(ctx context.Context, endpointID, requestID string) (map[string]any, []byte, error) {
	raw, err := p.requestJSON(ctx, http.MethodGet, p.queueURL(endpointID)+"/requests/"+url.PathEscape(requestID), nil, falAuthHeaders(p.apiKey))
	if err != nil {
		return nil, raw, err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, raw, fmt.Errorf("parse fal result response: %w", err)
	}
	return payload, raw, nil
}

func (p FalProvider) generatedFilesFromPayload(
	ctx context.Context,
	task domain.Task,
	requestMode string,
	model string,
	endpointID string,
	imageSize any,
	requestID string,
	metrics any,
	payload map[string]any,
) ([]GeneratedFile, error) {
	candidates := falImageCandidates(payload)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("fal provider returned no image data")
	}

	costRaw := falCostRaw(requestID, metrics)
	files := make([]GeneratedFile, 0, len(candidates))
	for index, candidate := range candidates {
		imageBytes, err := p.generatedImageBytes(ctx, candidate)
		if err != nil {
			return nil, err
		}
		pngBytes, width, height, err := normalizePNG(imageBytes)
		if err != nil {
			return nil, err
		}
		parameters := taskProviderParameters(task, map[string]any{
			"provider":      FalProviderID,
			"model":         model,
			"endpoint_id":   endpointID,
			"request_mode":  requestMode,
			"slot":          index,
			"image_size":    imageSize,
			"aspect_ratio":  task.AspectRatio,
			"output_format": "png",
		})
		files = append(files, GeneratedFile{
			Slot:          index,
			Bytes:         pngBytes,
			Thumbnail:     pngBytes,
			MimeType:      "image/png",
			Width:         width,
			Height:        height,
			ThumbnailW:    width,
			ThumbnailH:    height,
			Model:         model,
			ParametersRaw: parameters,
			CostRaw:       costRaw,
		})
	}
	return files, nil
}

func (p FalProvider) generatedImageBytes(ctx context.Context, candidate any) ([]byte, error) {
	switch value := candidate.(type) {
	case string:
		return p.imageBytesFromString(ctx, value)
	case map[string]any:
		for _, key := range []string{"url", "b64_json", "base64", "data"} {
			if rawValue, ok := value[key].(string); ok && strings.TrimSpace(rawValue) != "" {
				return p.imageBytesFromString(ctx, rawValue)
			}
		}
	}
	return nil, fmt.Errorf("fal result item has no supported image value")
}

func (p FalProvider) imageBytesFromString(ctx context.Context, value string) ([]byte, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("fal result image value is empty")
	}
	if isHTTPURL(trimmed) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimmed, nil)
		if err != nil {
			return nil, err
		}
		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("download fal result image failed: %s", resp.Status)
		}
		return io.ReadAll(io.LimitReader(resp.Body, maxImageResponseBytes))
	}
	return base64.StdEncoding.DecodeString(stripDataURLPrefix(trimmed))
}

func (p FalProvider) requestJSON(ctx context.Context, method, targetURL string, body any, headers map[string]string) ([]byte, error) {
	var requestBody io.Reader
	if body != nil {
		requestBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		requestBody = strings.NewReader(string(requestBytes))
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, requestBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxImageResponseBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return raw, fmt.Errorf("fal provider failed: %s", falErrorMessage(raw, resp.Status))
	}
	return raw, nil
}

func falEndpointID(model string, isEdit bool) string {
	model = strings.TrimSpace(model)
	if !isEdit || strings.HasSuffix(model, "/edit") {
		return model
	}
	return model + "/edit"
}

func (p FalProvider) queueURL(endpointID string) string {
	return p.baseURL + "/" + strings.TrimLeft(endpointID, "/")
}

func falUploadFilename(item resolvedTaskInputFile, mimeType string) string {
	filename := filepath.Base(strings.TrimSpace(item.FilePath))
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		filename = item.InputFileID + fileExtensionForMultipartMime(mimeType)
	}
	if filepath.Ext(filename) == "" {
		filename += fileExtensionForMultipartMime(mimeType)
	}
	return filename
}

func falAuthHeaders(apiKey string) map[string]string {
	return map[string]string{
		"Authorization": "Key " + strings.TrimSpace(apiKey),
	}
}

func falImageSize(aspectRatio string) string {
	switch aspectRatio {
	case "3:4":
		return "portrait_4_3"
	case "4:3":
		return "landscape_4_3"
	case "9:16":
		return "portrait_16_9"
	case "16:9":
		return "landscape_16_9"
	default:
		return "square_hd"
	}
}

func falOutputFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "jpg", "jpeg":
		return "jpeg"
	case "webp":
		return "webp"
	default:
		return "png"
	}
}

func falQuality(task domain.Task) string {
	var input struct {
		GenerationConfig json.RawMessage `json:"generation_config"`
	}
	if len(task.StructuredInputJSON) == 0 || json.Unmarshal(task.StructuredInputJSON, &input) != nil || len(input.GenerationConfig) == 0 {
		return "high"
	}
	var config struct {
		Quality string `json:"quality"`
	}
	if json.Unmarshal(input.GenerationConfig, &config) != nil {
		return "high"
	}
	switch strings.ToLower(strings.TrimSpace(config.Quality)) {
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high":
		return "high"
	default:
		return "high"
	}
}

func falImageCandidates(payload map[string]any) []any {
	candidates := make([]any, 0, 4)
	if images, ok := payload["images"].([]any); ok {
		candidates = append(candidates, images...)
	}
	if image, ok := payload["image"]; ok {
		candidates = append(candidates, image)
	}
	if imageURL, ok := payload["url"]; ok {
		candidates = append(candidates, imageURL)
	}
	return candidates
}

func falErrorMessage(raw []byte, fallback string) string {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err == nil {
		if detail, ok := payload["detail"].(string); ok && strings.TrimSpace(detail) != "" {
			return detail
		}
		if message, ok := payload["message"].(string); ok && strings.TrimSpace(message) != "" {
			return message
		}
		if errValue, ok := payload["error"].(string); ok && strings.TrimSpace(errValue) != "" {
			return errValue
		}
	}
	text := strings.TrimSpace(string(raw))
	if text != "" && len(text) <= 500 {
		return text
	}
	return fallback
}

func falCostRaw(requestID string, metrics any) []byte {
	payload := map[string]any{
		"provider":   FalProviderID,
		"request_id": requestID,
	}
	if metrics != nil {
		payload["metrics"] = metrics
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return []byte(`{"provider":"fal"}`)
	}
	return raw
}

func rawProviderEnvelope(submitRaw, statusRaw, resultRaw []byte) []byte {
	payload := map[string]json.RawMessage{}
	if len(submitRaw) > 0 {
		payload["submit"] = submitRaw
	}
	if len(statusRaw) > 0 {
		payload["status"] = statusRaw
	}
	if len(resultRaw) > 0 {
		payload["result"] = resultRaw
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return []byte(`{}`)
	}
	return raw
}

func isHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

type falUploadInitiateResponse struct {
	UploadURL string `json:"upload_url"`
	FileURL   string `json:"file_url"`
}

type falQueueStatus struct {
	Status    string `json:"status"`
	RequestID string `json:"request_id"`
	Metrics   any    `json:"metrics"`
}
