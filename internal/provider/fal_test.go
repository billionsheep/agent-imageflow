package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestFalProviderGeneratesFromQueueResult(t *testing.T) {
	sourcePNG := testPNG(t)
	var captured map[string]any
	var statusCalls int

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/queue/openai/gpt-image-2":
			if r.Header.Get("Authorization") != "Key test-key" {
				t.Fatalf("missing fal auth header: %s", r.Header.Get("Authorization"))
			}
			if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
				t.Fatalf("decode submit body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":     "IN_QUEUE",
				"request_id": "fal_req_123",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/queue/openai/gpt-image-2/requests/fal_req_123/status":
			statusCalls++
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":     "COMPLETED",
				"request_id": "fal_req_123",
				"metrics":    map[string]any{"inference_time": 1.23},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/queue/openai/gpt-image-2/requests/fal_req_123":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"images": []map[string]any{
					{"url": server.URL + "/result.png"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/result.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(sourcePNG)
		default:
			t.Fatalf("unexpected fal test request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	provider := NewFalProvider(FalConfig{
		BaseURL:        server.URL + "/queue",
		RestBaseURL:    server.URL + "/rest",
		APIKey:         "test-key",
		Model:          "openai/gpt-image-2",
		PollIntervalMs: 1,
		HTTPClient:     server.Client(),
	})
	task := testTask("1:1")
	task.Provider = FalProviderID
	task.RequestedCount = 2

	result, err := provider.Generate(context.Background(), task)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if result.ProviderRequestID != "fal_req_123" || result.Status != "succeeded" {
		t.Fatalf("unexpected fal result metadata: %#v", result)
	}
	if statusCalls == 0 {
		t.Fatal("expected fal status polling to occur")
	}
	if captured["prompt"] != "生成一张封面图" || captured["image_size"] != "square_hd" {
		t.Fatalf("unexpected fal submit body: %#v", captured)
	}
	if got := int(captured["num_images"].(float64)); got != 2 {
		t.Fatalf("unexpected num_images: %d", got)
	}
	if len(result.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(result.Files))
	}
	var params map[string]any
	if err := json.Unmarshal(result.Files[0].ParametersRaw, &params); err != nil {
		t.Fatalf("parameters are not JSON: %v", err)
	}
	if params["request_mode"] != "generation" || params["endpoint_id"] != "openai/gpt-image-2" {
		t.Fatalf("unexpected fal parameters: %#v", params)
	}
}

func TestFalProviderUsesEditEndpointAndUploadsResolvedInputs(t *testing.T) {
	sourcePNG := testPNG(t)
	tempDir := t.TempDir()
	refOnePath := tempDir + "/reference-one.png"
	refTwoPath := tempDir + "/reference-two.png"
	maskPath := tempDir + "/mask.png"
	for _, path := range []string{refOnePath, refTwoPath, maskPath} {
		if err := os.WriteFile(path, sourcePNG, 0o644); err != nil {
			t.Fatalf("write input file %s: %v", path, err)
		}
	}

	var uploadInitiateCount int
	var uploadPutCount int
	var submitPath string
	var submitBody map[string]any
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/rest/storage/upload/initiate":
			uploadInitiateCount++
			if r.Header.Get("Authorization") != "Key test-key" {
				t.Fatalf("missing fal auth header on upload initiate: %s", r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"upload_url": server.URL + "/upload/" + strings.TrimPrefix(r.URL.RawQuery, ""),
				"file_url":   server.URL + "/files/file-" + strings.TrimPrefix(string(rune('0'+uploadInitiateCount)), ""),
			})
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/upload/"):
			uploadPutCount++
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/queue/openai/gpt-image-2/edit":
			submitPath = r.URL.Path
			if err := json.NewDecoder(r.Body).Decode(&submitBody); err != nil {
				t.Fatalf("decode edit submit body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":     "IN_QUEUE",
				"request_id": "fal_req_edit_123",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/queue/openai/gpt-image-2/edit/requests/fal_req_edit_123/status":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":     "COMPLETED",
				"request_id": "fal_req_edit_123",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/queue/openai/gpt-image-2/edit/requests/fal_req_edit_123":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"images": []map[string]any{
					{"url": server.URL + "/result.png"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/result.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(sourcePNG)
		default:
			t.Fatalf("unexpected fal edit test request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	provider := NewFalProvider(FalConfig{
		BaseURL:        server.URL + "/queue",
		RestBaseURL:    server.URL + "/rest",
		APIKey:         "test-key",
		Model:          "openai/gpt-image-2",
		PollIntervalMs: 1,
		HTTPClient:     server.Client(),
	})
	task := testTask("4:3")
	task.Provider = FalProviderID
	task.StructuredInputJSON = []byte(`{
		"reference_images":[
			{"id":"ref_one","input_file_id":"inp_ref_one","role":"style"},
			{"id":"ref_two","input_file_id":"inp_ref_two","role":"edit_target"}
		],
		"mask_image":{"input_file_id":"inp_mask","target_image_id":"ref_two","has_mask":true},
		"resolved_input_files":{
			"reference_images":[
				{"input_file_id":"inp_ref_one","kind":"reference","file_path":"` + refOnePath + `","mime_type":"image/png","role":"style"},
				{"input_file_id":"inp_ref_two","kind":"reference","file_path":"` + refTwoPath + `","mime_type":"image/png","role":"edit_target"}
			],
			"mask_image":{"input_file_id":"inp_mask","kind":"mask","file_path":"` + maskPath + `","mime_type":"image/png","target_image_id":"ref_two"}
		}
	}`)

	result, err := provider.Generate(context.Background(), task)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if submitPath != "/queue/openai/gpt-image-2/edit" {
		t.Fatalf("unexpected fal edit submit path: %s", submitPath)
	}
	if uploadInitiateCount != 3 || uploadPutCount != 3 {
		t.Fatalf("expected 3 upload initiates and 3 upload PUTs, got initiates=%d puts=%d", uploadInitiateCount, uploadPutCount)
	}
	imageURLs, ok := submitBody["image_urls"].([]any)
	if !ok || len(imageURLs) != 2 {
		t.Fatalf("unexpected image_urls payload: %#v", submitBody["image_urls"])
	}
	if _, ok := submitBody["mask_url"].(string); !ok {
		t.Fatalf("missing mask_url in fal edit payload: %#v", submitBody)
	}
	if len(result.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(result.Files))
	}
	var params map[string]any
	if err := json.Unmarshal(result.Files[0].ParametersRaw, &params); err != nil {
		t.Fatalf("parameters are not JSON: %v", err)
	}
	if params["request_mode"] != "edit" || params["endpoint_id"] != "openai/gpt-image-2/edit" {
		t.Fatalf("unexpected fal edit parameters: %#v", params)
	}
}

func TestFalProviderReturnsStructuredSubmitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"detail": "bad prompt",
		})
	}))
	defer server.Close()

	provider := NewFalProvider(FalConfig{
		BaseURL:        server.URL + "/queue",
		RestBaseURL:    server.URL + "/rest",
		APIKey:         "test-key",
		Model:          "openai/gpt-image-2",
		PollIntervalMs: 1,
		HTTPClient:     server.Client(),
	})
	task := testTask("1:1")
	task.Provider = FalProviderID

	result, err := provider.Generate(context.Background(), task)
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if result.ErrorCode != "submit_failed" || !strings.Contains(result.ErrorMessage, "bad prompt") {
		t.Fatalf("unexpected fal error result: %#v", result)
	}
}
