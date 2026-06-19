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
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

const maxImageResponseBytes = 50 << 20

type OpenAICompatibleConfig struct {
	BaseURL        string
	APIKey         string
	Model          string
	TimeoutSeconds int
	HTTPClient     *http.Client
}

type OpenAICompatibleProvider struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewOpenAICompatibleProvider(cfg OpenAICompatibleConfig) OpenAICompatibleProvider {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
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
	body := map[string]any{
		"model":           p.model,
		"prompt":          task.Prompt,
		"n":               max(1, min(task.RequestedCount, 4)),
		"size":            size,
		"response_format": "b64_json",
		"output_format":   outputFormat,
	}
	if task.NegativePrompt != "" {
		body["negative_prompt"] = task.NegativePrompt
	}
	if task.StylePreset != "" {
		body["style_preset"] = task.StylePreset
	}

	requestBytes, err := json.Marshal(body)
	if err != nil {
		return Result{}, err
	}
	url := p.baseURL + "/images/generations"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBytes))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	return p.parseImageResponse(ctx, task, size, "generation", resp)
}

func (p OpenAICompatibleProvider) generateEdit(ctx context.Context, task domain.Task, size, outputFormat string, input *resolvedEditInput) (Result, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", p.model); err != nil {
		return Result{}, err
	}
	if err := writer.WriteField("prompt", task.Prompt); err != nil {
		return Result{}, err
	}
	if err := writer.WriteField("size", size); err != nil {
		return Result{}, err
	}
	if err := writer.WriteField("response_format", "b64_json"); err != nil {
		return Result{}, err
	}
	if err := writer.WriteField("output_format", outputFormat); err != nil {
		return Result{}, err
	}
	if count := max(1, min(task.RequestedCount, 4)); count > 1 {
		if err := writer.WriteField("n", fmt.Sprintf("%d", count)); err != nil {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/images/edits", &body)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	return p.parseImageResponse(ctx, task, size, "edit", resp)
}

func (p OpenAICompatibleProvider) parseImageResponse(ctx context.Context, task domain.Task, size, requestMode string, resp *http.Response) (Result, error) {
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxImageResponseBytes))
	if err != nil {
		return Result{}, err
	}
	result := Result{
		ProviderRequestID: resp.Header.Get("X-Request-Id"),
		Status:            "received",
		RawResponse:       respBytes,
		CostRaw:           []byte(`{"provider":"openai-compatible"}`),
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		code := fmt.Sprintf("http_%d", resp.StatusCode)
		message := responseErrorMessage(respBytes, resp.Status)
		result.ErrorCode = code
		result.ErrorMessage = message
		return result, fmt.Errorf("openai-compatible provider failed: %s", message)
	}

	var payload openAICompatibleResponse
	if err := json.Unmarshal(respBytes, &payload); err != nil {
		result.ErrorCode = "invalid_response"
		result.ErrorMessage = err.Error()
		return result, fmt.Errorf("parse openai-compatible response: %w", err)
	}
	if payload.Error != nil && payload.Error.Message != "" {
		result.ErrorCode = firstNonEmpty(payload.Error.Code, "provider_error")
		result.ErrorMessage = payload.Error.Message
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
		imageBytes, err := p.imageBytes(ctx, item)
		if err != nil {
			result.ErrorCode = "image_decode_failed"
			result.ErrorMessage = err.Error()
			return result, err
		}
		pngBytes, width, height, err := normalizePNG(imageBytes)
		if err != nil {
			result.ErrorCode = "unsupported_image"
			result.ErrorMessage = err.Error()
			return result, err
		}
		parameterBytes := taskProviderParameters(task, map[string]any{
			"provider":       "openai-compatible",
			"model":          p.model,
			"request_mode":   requestMode,
			"slot":           i,
			"size":           size,
			"aspect_ratio":   task.AspectRatio,
			"output_format":  "png",
			"revised_prompt": item.RevisedPrompt,
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
			Model:         p.model,
			ParametersRaw: parameterBytes,
			CostRaw:       result.CostRaw,
		})
	}
	if len(files) == 0 {
		result.ErrorCode = "empty_response"
		result.ErrorMessage = "openai-compatible provider returned no image data"
		return result, errors.New(result.ErrorMessage)
	}
	result.Status = "succeeded"
	result.Files = files
	return result, nil
}

func (p OpenAICompatibleProvider) readEditInputFile(path, mimeType string, forcePNG bool) ([]byte, string, error) {
	return readTaskInputFile(path, mimeType, forcePNG)
}

func (p OpenAICompatibleProvider) imageBytes(ctx context.Context, item openAICompatibleDataItem) ([]byte, error) {
	if item.B64JSON != "" {
		return base64.StdEncoding.DecodeString(stripDataURLPrefix(item.B64JSON))
	}
	if item.URL == "" {
		return nil, fmt.Errorf("response item has neither b64_json nor url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download image failed: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxImageResponseBytes))
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
