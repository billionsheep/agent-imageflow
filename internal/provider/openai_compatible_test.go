package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
	result, err := provider.Generate(context.Background(), testTask("3:4"))
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
}

func TestOpenAICompatibleProviderGeneratesFromURLResponse(t *testing.T) {
	sourcePNG := testPNG(t)
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/images/generations", func(w http.ResponseWriter, r *http.Request) {
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
	var imageCount int
	var maskCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		capturedPrompt = r.FormValue("prompt")
		capturedModel = r.FormValue("model")
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
	if params["request_mode"] != "edit" {
		t.Fatalf("expected edit request_mode, got %#v", params["request_mode"])
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
