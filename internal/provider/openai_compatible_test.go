package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestOpenAICompatibleProviderGeneratesFromBase64Response(t *testing.T) {
	sourcePNG := testPNG(t)
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/generations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("missing Authorization header: %s", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "resp_123",
			"data": []map[string]any{
				{
					"b64_json":       base64.StdEncoding.EncodeToString(sourcePNG),
					"revised_prompt": "revised",
				},
			},
			"usage": map[string]any{"total_tokens": 1},
		})
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "image-model",
	})
	task := testTask("3:4")
	task.StructuredInputJSON = []byte(`{"generation_config":{"quality":"high","moderation":"low","output_compression":80,"stream":true,"partial_images":2}}`)
	result, err := provider.Generate(context.Background(), task)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if result.ProviderRequestID != "resp_123" || result.Status != "succeeded" {
		t.Fatalf("unexpected result metadata: %#v", result)
	}
	if captured["model"] != "image-model" || captured["prompt"] != "生成一张封面图" || captured["size"] != "1024x1536" {
		t.Fatalf("unexpected request body: %#v", captured)
	}
	if got := int(captured["n"].(float64)); got != 1 {
		t.Fatalf("unexpected n: %d", got)
	}
	if _, ok := captured["response_format"]; ok {
		t.Fatalf("response_format should be omitted by default, got request body: %#v", captured)
	}
	if captured["quality"] != "high" || captured["moderation"] != "low" || int(captured["output_compression"].(float64)) != 80 {
		t.Fatalf("generation_config passthrough missing from request body: %#v", captured)
	}
	if captured["stream"] != true || int(captured["partial_images"].(float64)) != 2 {
		t.Fatalf("streaming fields missing from request body: %#v", captured)
	}
	if len(result.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(result.Files))
	}
	file := result.Files[0]
	if file.MimeType != "image/png" || file.Width != 2 || file.Height != 2 || file.Model != "image-model" {
		t.Fatalf("unexpected generated file: %#v", file)
	}
	if _, err := png.Decode(bytes.NewReader(file.Bytes)); err != nil {
		t.Fatalf("file bytes are not PNG: %v", err)
	}
	var params map[string]any
	if err := json.Unmarshal(file.ParametersRaw, &params); err != nil {
		t.Fatalf("parameters are not JSON: %v", err)
	}
	if params["revised_prompt"] != "revised" {
		t.Fatalf("missing revised prompt in parameters: %#v", params)
	}
	if params["request_mode"] != OpenAICompatibleRequestModeImagesStream || params["response_format"] != openAICompatibleResponseFormatOmit || params["endpoint"] != openAICompatibleEndpointGenerations {
		t.Fatalf("request shape missing from parameters: %#v", params)
	}
}

func TestOpenAICompatibleProviderGeneratesFromURLResponse(t *testing.T) {
	sourcePNG := testPNG(t)
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	var captured map[string]any
	mux.HandleFunc("/images/generations", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"url": server.URL + "/image.png"}},
		})
	})
	mux.HandleFunc("/image.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(sourcePNG)
	})

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "image-model",
	})
	result, err := provider.Generate(context.Background(), testTask("1:1"))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(result.Files))
	}
	if result.Files[0].Width != 2 || result.Files[0].Height != 2 {
		t.Fatalf("unexpected dimensions: %#v", result.Files[0])
	}
	if _, ok := captured["response_format"]; ok {
		t.Fatalf("response_format should be omitted for URL-preferred request: %#v", captured)
	}
}

func TestOpenAICompatibleProviderDefaultsToSingleImageRequest(t *testing.T) {
	sourcePNG := testPNG(t)
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"b64_json": base64.StdEncoding.EncodeToString(sourcePNG)}},
		})
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "image-model",
	})
	task := testTask("1:1")
	task.RequestedCount = 10
	if _, err := provider.Generate(context.Background(), task); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if got := int(captured["n"].(float64)); got != 1 {
		t.Fatalf("default openai-compatible n = %d, want 1; body=%#v", got, captured)
	}
}

func TestOpenAICompatibleProviderSendsBase64ResponseFormatWhenExplicitlyConfigured(t *testing.T) {
	sourcePNG := testPNG(t)
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"b64_json": base64.StdEncoding.EncodeToString(sourcePNG)}},
		})
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "image-model",
	})
	task := testTask("1:1")
	task.StructuredInputJSON = []byte(`{"provider_profile":{"enabled":true,"provider":"openai-compatible","preferred_response_format":"b64_json"}}`)
	result, err := provider.Generate(context.Background(), task)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if captured["response_format"] != openAICompatibleResponseFormatB64 {
		t.Fatalf("response_format = %#v, want b64_json; body=%#v", captured["response_format"], captured)
	}
	var params map[string]any
	if err := json.Unmarshal(result.Files[0].ParametersRaw, &params); err != nil {
		t.Fatalf("parameters are not JSON: %v", err)
	}
	if params["request_mode"] != OpenAICompatibleRequestModeImagesSyncB64 || params["response_format"] != openAICompatibleResponseFormatB64 {
		t.Fatalf("expected b64 request shape in parameters: %#v", params)
	}
}

func TestOpenAICompatibleProviderParsesImagesStreamingSSE(t *testing.T) {
	sourcePNG := testPNG(t)
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/generations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		partial := base64.StdEncoding.EncodeToString(sourcePNG)
		final := partial
		_, _ = fmt.Fprintf(w, "data: {\"type\":\"image_generation.partial_image\",\"partial_image_index\":0,\"b64_json\":%q}\n\n", partial)
		_, _ = fmt.Fprintf(w, "data: {\"type\":\"image_generation.completed\",\"b64_json\":%q,\"revised_prompt\":\"stream revised\"}\n\n", final)
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "image-model",
	})
	task := testTask("1:1")
	task.StructuredInputJSON = []byte(`{"generation_config":{"stream":true,"partial_images":1}}`)
	result, err := provider.Generate(context.Background(), task)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if captured["stream"] != true || int(captured["partial_images"].(float64)) != 1 {
		t.Fatalf("streaming fields missing from request body: %#v", captured)
	}
	if result.Metrics.ErrorStage != "" {
		t.Fatalf("unexpected error_stage: %#v", result.Metrics)
	}
	if len(result.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(result.Files))
	}
	var params map[string]any
	if err := json.Unmarshal(result.Files[0].ParametersRaw, &params); err != nil {
		t.Fatalf("parameters are not JSON: %v", err)
	}
	if params["request_mode"] != OpenAICompatibleRequestModeImagesStream || int(params["partial_count"].(float64)) != 1 {
		t.Fatalf("expected stream request shape with partial count: %#v", params)
	}
}

func TestOpenAICompatibleProviderParsesResponsesStreamingSSE(t *testing.T) {
	sourcePNG := testPNG(t)
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		b64 := base64.StdEncoding.EncodeToString(sourcePNG)
		_, _ = fmt.Fprintf(w, "data: {\"type\":\"response.image_generation_call.partial_image\",\"partial_image_index\":0,\"partial_image_b64\":%q}\n\n", b64)
		_, _ = fmt.Fprintf(w, "data: {\"type\":\"response.output_item.done\",\"item\":{\"id\":\"img_1\",\"type\":\"image_generation_call\",\"result\":{\"b64_json\":%q},\"revised_prompt\":\"responses revised\"}}\n\n", b64)
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
	})
	task := testTask("1:1")
	task.StructuredInputJSON = []byte(`{"generation_config":{"api_mode":"responses","stream":true,"partial_images":1}}`)
	result, err := provider.Generate(context.Background(), task)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if captured["model"] != openAICompatibleDefaultResponsesModel || captured["stream"] != true {
		t.Fatalf("unexpected responses request body: %#v", captured)
	}
	tools := captured["tools"].([]any)
	tool := tools[0].(map[string]any)
	if tool["type"] != "image_generation" || int(tool["partial_images"].(float64)) != 1 {
		t.Fatalf("unexpected image_generation tool: %#v", tool)
	}
	if len(result.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(result.Files))
	}
	var params map[string]any
	if err := json.Unmarshal(result.Files[0].ParametersRaw, &params); err != nil {
		t.Fatalf("parameters are not JSON: %v", err)
	}
	if params["api_mode"] != openAICompatibleAPIModeResponses || params["request_mode"] != OpenAICompatibleRequestModeResponsesStream || int(params["partial_count"].(float64)) != 1 {
		t.Fatalf("expected responses stream request shape: %#v", params)
	}
}

func TestOpenAICompatibleProviderGenerationConfigOverridesProviderProfileShape(t *testing.T) {
	sourcePNG := testPNG(t)
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/generations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"b64_json": base64.StdEncoding.EncodeToString(sourcePNG)}},
		})
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "image-model",
	})
	task := testTask("1:1")
	task.StructuredInputJSON = []byte(`{
		"provider_profile":{"enabled":true,"provider":"openai-compatible","api_mode":"responses","stream":true,"partial_images":2,"preferred_response_format":"b64_json"},
		"generation_config":{"api_mode":"images","stream":false,"preferred_response_format":"url","max_n":10}
	}`)
	task.RequestedCount = 10
	result, err := provider.Generate(context.Background(), task)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if captured["stream"] != nil || captured["response_format"] != nil {
		t.Fatalf("task generation_config should disable stream and b64: %#v", captured)
	}
	if int(captured["n"].(float64)) != 10 {
		t.Fatalf("generation_config max_n should allow n=10, got body: %#v", captured)
	}
	var params map[string]any
	if err := json.Unmarshal(result.Files[0].ParametersRaw, &params); err != nil {
		t.Fatalf("parameters are not JSON: %v", err)
	}
	if params["api_mode"] != openAICompatibleAPIModeImages || params["request_mode"] != OpenAICompatibleRequestModeImagesSyncURL {
		t.Fatalf("generation_config did not override provider profile: %#v", params)
	}
}

func TestOpenAICompatibleProviderReturnsStructuredHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "bad prompt",
				"code":    "invalid_request",
			},
		})
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "image-model",
	})
	result, err := provider.Generate(context.Background(), testTask("1:1"))
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if result.ErrorCode != "http_400" || result.ErrorMessage != "bad prompt" {
		t.Fatalf("unexpected error result: %#v", result)
	}
}

func TestOpenAICompatibleProviderMarksSlowHeaderTimeoutStage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL:                      server.URL,
		APIKey:                       "test-key",
		Model:                        "image-model",
		ResponseHeaderTimeoutSeconds: 1,
		TotalTimeoutSeconds:          5,
	})
	result, err := provider.Generate(context.Background(), testTask("1:1"))
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if result.Metrics.ErrorStage != "provider_first_byte" {
		t.Fatalf("error_stage = %q, want provider_first_byte; result=%#v", result.Metrics.ErrorStage, result)
	}
	if result.Metrics.ProviderTotalMs <= 0 {
		t.Fatalf("provider_total_ms should be recorded, got %#v", result.Metrics)
	}
}

func TestOpenAICompatibleProviderMarksSlowBodyTimeoutStage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		time.Sleep(1100 * time.Millisecond)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL:             server.URL,
		APIKey:              "test-key",
		Model:               "image-model",
		TotalTimeoutSeconds: 1,
	})
	result, err := provider.Generate(context.Background(), testTask("1:1"))
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if result.Metrics.ErrorStage != "response_download" {
		t.Fatalf("error_stage = %q, want response_download; result=%#v", result.Metrics.ErrorStage, result)
	}
	if result.Metrics.ResponseDownloadMs <= 0 {
		t.Fatalf("response_download_ms should be recorded, got %#v", result.Metrics)
	}
}

func TestClassifyOpenAIRequestErrorStageConnect(t *testing.T) {
	if got := classifyOpenAIRequestErrorStage(errors.New("dial tcp: connect: connection refused")); got != "connect" {
		t.Fatalf("stage = %q, want connect", got)
	}
}

func TestOpenAICompatibleProviderUsesEditsEndpointForResolvedInputs(t *testing.T) {
	sourcePNG := testPNG(t)
	tempDir := t.TempDir()
	refPath := tempDir + "/reference.png"
	maskPath := tempDir + "/mask.png"
	if err := os.WriteFile(refPath, sourcePNG, 0o644); err != nil {
		t.Fatalf("write reference input: %v", err)
	}
	if err := os.WriteFile(maskPath, sourcePNG, 0o644); err != nil {
		t.Fatalf("write mask input: %v", err)
	}

	var capturedPath string
	var capturedPrompt string
	var capturedModel string
	var capturedQuality string
	var capturedModeration string
	var capturedOutputCompression string
	var capturedResponseFormat string
	var capturedStream string
	var capturedPartialImages string
	var imageCount int
	var maskCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		capturedPrompt = r.FormValue("prompt")
		capturedModel = r.FormValue("model")
		capturedQuality = r.FormValue("quality")
		capturedModeration = r.FormValue("moderation")
		capturedOutputCompression = r.FormValue("output_compression")
		capturedResponseFormat = r.FormValue("response_format")
		capturedStream = r.FormValue("stream")
		capturedPartialImages = r.FormValue("partial_images")
		imageCount = len(r.MultipartForm.File["image[]"])
		maskCount = len(r.MultipartForm.File["mask"])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "resp_edit_123",
			"data": []map[string]any{
				{
					"b64_json": base64.StdEncoding.EncodeToString(sourcePNG),
				},
			},
		})
	}))
	defer server.Close()

	provider := NewOpenAICompatibleProvider(OpenAICompatibleConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "image-model",
	})
	task := testTask("1:1")
	task.StructuredInputJSON = []byte(`{
		"generation_config":{"quality":"medium","moderation":"auto","output_compression":70,"stream":true,"partial_images":2},
		"reference_images":[{"id":"web_ref_1","input_file_id":"inp_ref_1","role":"edit_target"}],
		"mask_image":{"input_file_id":"inp_mask_1","target_image_id":"web_ref_1","has_mask":true},
		"resolved_input_files":{
			"reference_images":[{"input_file_id":"inp_ref_1","kind":"reference","file_path":"` + refPath + `","mime_type":"image/png","role":"edit_target"}],
			"mask_image":{"input_file_id":"inp_mask_1","kind":"mask","file_path":"` + maskPath + `","mime_type":"image/png","target_image_id":"web_ref_1"}
		}
	}`)

	result, err := provider.Generate(context.Background(), task)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if capturedPath != "/images/edits" {
		t.Fatalf("unexpected request path: %s", capturedPath)
	}
	if capturedModel != "image-model" || capturedPrompt != "生成一张封面图" {
		t.Fatalf("unexpected multipart fields: model=%q prompt=%q", capturedModel, capturedPrompt)
	}
	if capturedQuality != "medium" || capturedModeration != "auto" || capturedOutputCompression != "70" {
		t.Fatalf("generation_config passthrough missing from multipart fields: quality=%q moderation=%q compression=%q", capturedQuality, capturedModeration, capturedOutputCompression)
	}
	if capturedResponseFormat != "" {
		t.Fatalf("response_format should be omitted by default for edits, got %q", capturedResponseFormat)
	}
	if capturedStream != "true" || capturedPartialImages != "2" {
		t.Fatalf("streaming fields missing from multipart request: stream=%q partial=%q", capturedStream, capturedPartialImages)
	}
	if imageCount != 1 || maskCount != 1 {
		t.Fatalf("unexpected multipart file counts: images=%d mask=%d", imageCount, maskCount)
	}
	if len(result.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(result.Files))
	}
	var params map[string]any
	if err := json.Unmarshal(result.Files[0].ParametersRaw, &params); err != nil {
		t.Fatalf("parameters are not JSON: %v", err)
	}
	if params["request_mode"] != OpenAICompatibleRequestModeImagesStream || params["operation"] != "edit" || params["endpoint"] != openAICompatibleEndpointEdits {
		t.Fatalf("expected edit URL request shape, got %#v", params)
	}
}

func TestReferenceParticipationErrorIncludesUserReadableSourceAndMIME(t *testing.T) {
	err := referenceParticipationError(resolvedTaskInputFile{
		InputFileID: "inp_missing",
		Kind:        "reference",
		FilePath:    "/tmp/does-not-exist.png",
		MimeType:    "image/webp",
		Role:        "character_primary",
	}, errors.New("open /tmp/does-not-exist.png: no such file or directory"))
	if err == nil {
		t.Fatal("expected error")
	}
	message := err.Error()
	for _, want := range []string{"参考图未参与生成", "input_file_id=inp_missing", "mime_type=image/webp", "source=input_file"} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected error to contain %q, got %q", want, message)
		}
	}
}

func testTask(aspectRatio string) domain.Task {
	return domain.Task{
		ID:             "task_test",
		Prompt:         "生成一张封面图",
		NegativePrompt: "low quality",
		StylePreset:    "anime-cover",
		AspectRatio:    aspectRatio,
		OutputFormat:   "png",
		RequestedCount: 1,
		Provider:       OpenAICompatibleProviderID,
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 0, color.RGBA{G: 255, A: 255})
	img.Set(0, 1, color.RGBA{B: 255, A: 255})
	img.Set(1, 1, color.RGBA{R: 255, G: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test PNG: %v", err)
	}
	return buf.Bytes()
}
