package main

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/config"
	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/provider"
)

func TestBenchmarkRealProviderRequiresExplicitOptIn(t *testing.T) {
	err := benchmarkImageGenerationCmd([]string{
		"--provider", "openai-compatible",
		"--tasks", "1",
		"--requested-count", "1",
	})
	if err == nil {
		t.Fatal("expected benchmark to reject real provider without explicit opt-in")
	}
	if !strings.Contains(err.Error(), "--allow-paid-provider") {
		t.Fatalf("expected opt-in hint, got %v", err)
	}
}

func TestBatchManifestPathBuildsExpectedQuery(t *testing.T) {
	path := buildBatchManifestPath("prj_demo", "cmp_demo", batchManifestOptions{
		SessionID:       "session_1",
		BatchID:         "batch_1",
		StoryID:         "story_1",
		Source:          "codex",
		Status:          "completed",
		IncludeSetup:    true,
		Limit:           250,
		SelectedOnly:    false,
		IncludeRejected: true,
	})

	parsed, err := url.Parse(path)
	if err != nil {
		t.Fatalf("parse path: %v", err)
	}
	if parsed.Path != "/api/projects/prj_demo/campaigns/cmp_demo/batch-manifest" {
		t.Fatalf("unexpected path %q", parsed.Path)
	}
	query := parsed.Query()
	for key, want := range map[string]string{
		"session_id":       "session_1",
		"batch_id":         "batch_1",
		"story_id":         "story_1",
		"source":           "codex",
		"status":           "completed",
		"include_setup":    "true",
		"limit":            "250",
		"selected_only":    "false",
		"include_rejected": "true",
	} {
		if got := query.Get(key); got != want {
			t.Fatalf("query %s = %q, want %q in %s", key, got, want, path)
		}
	}
}

func TestBatchManifestPathBuildsFinalDeliveryViewQuery(t *testing.T) {
	path := buildBatchManifestPath("prj_demo", "cmp_demo", batchManifestOptions{
		SessionID:    "session_1",
		SelectedOnly: true,
		View:         domain.BatchManifestViewFinalDelivery,
	})

	parsed, err := url.Parse(path)
	if err != nil {
		t.Fatalf("parse path: %v", err)
	}
	if got := parsed.Query().Get("view"); got != domain.BatchManifestViewFinalDelivery {
		t.Fatalf("query view = %q, want %q", got, domain.BatchManifestViewFinalDelivery)
	}
}

func TestBenchmarkPercentileAndAverage(t *testing.T) {
	values := []int64{100, 20, 300, 40}
	if got := avgInt64(values); got != 115 {
		t.Fatalf("avgInt64 = %d, want 115", got)
	}
	if got := percentileInt64(values, 50); got != 40 {
		t.Fatalf("p50 = %d, want 40", got)
	}
	if got := percentileInt64(values, 95); got != 100 {
		t.Fatalf("p95 = %d, want 100", got)
	}
}

func TestBenchmarkTimeoutAttemptDetection(t *testing.T) {
	code := "http_timeout"
	latency := 119_500
	attempt := domain.TaskAttempt{
		Status:    domain.AttemptFailed,
		ErrorCode: &code,
		LatencyMs: &latency,
	}
	if !isTimeoutAttempt(attempt, 120) {
		t.Fatal("expected timeout attempt to be detected")
	}
}

func TestInferBenchmarkRequestShapeForOpenAICompatibleURLMode(t *testing.T) {
	structured, err := json.Marshal(map[string]any{
		"provider_profile": map[string]any{
			"enabled":                   true,
			"provider":                  provider.OpenAICompatibleProviderID,
			"max_n":                     4,
			"preferred_response_format": "url",
		},
		"generation_config": map[string]any{
			"quality":    "high",
			"moderation": "auto",
		},
	})
	if err != nil {
		t.Fatalf("marshal structured input: %v", err)
	}
	shape := inferBenchmarkRequestShape(provider.OpenAICompatibleProviderID, 10, config.Config{
		ProviderTimeoutSeconds:         300,
		OpenAICompatibleTotalTimeout:   600,
		OpenAICompatibleMaxConcurrency: 2,
	}, []domain.TaskResponse{{
		Task: domain.Task{
			AspectRatio:         "1:1",
			OutputFormat:        "png",
			StructuredInputJSON: structured,
		},
	}})

	if shape.APIMode != "images" || shape.Endpoint != "/images/generations" {
		t.Fatalf("unexpected api shape: %#v", shape)
	}
	if shape.RequestMode != provider.OpenAICompatibleRequestModeImagesSyncURL || shape.ResponseFormat != "omitted" {
		t.Fatalf("unexpected response shape: %#v", shape)
	}
	if shape.N != 4 || len(shape.SplitCounts) != 3 || shape.SplitCounts[0] != 4 || shape.SplitCounts[1] != 4 || shape.SplitCounts[2] != 2 {
		t.Fatalf("unexpected n/split counts: %#v", shape)
	}
	if shape.Quality != "high" || shape.Moderation != "auto" || shape.TimeoutSeconds != 600 {
		t.Fatalf("unexpected config fields: %#v", shape)
	}
}

func TestInferBenchmarkRequestShapeDefaultsOpenAICompatibleMaxNToOne(t *testing.T) {
	shape := inferBenchmarkRequestShape(provider.OpenAICompatibleProviderID, 3, config.Config{
		ProviderTimeoutSeconds:       300,
		OpenAICompatibleTotalTimeout: 300,
	}, []domain.TaskResponse{{
		Task: domain.Task{
			AspectRatio:  "1:1",
			OutputFormat: "png",
		},
	}})

	if shape.N != 1 || len(shape.SplitCounts) != 3 || shape.SplitCounts[0] != 1 || shape.SplitCounts[1] != 1 || shape.SplitCounts[2] != 1 {
		t.Fatalf("unexpected default split counts: %#v", shape)
	}
}

func TestInferBenchmarkRequestShapeForResponsesStream(t *testing.T) {
	structured, err := json.Marshal(map[string]any{
		"provider_profile": map[string]any{
			"enabled":  true,
			"provider": provider.OpenAICompatibleProviderID,
			"api_mode": "images",
			"model":    "gpt-image-2",
		},
		"generation_config": map[string]any{
			"api_mode":        "responses",
			"model":           "gpt-5.5",
			"stream":          true,
			"partial_images":  2,
			"timeout_seconds": 600,
		},
	})
	if err != nil {
		t.Fatalf("marshal structured input: %v", err)
	}
	shape := inferBenchmarkRequestShape(provider.OpenAICompatibleProviderID, 1, config.Config{
		ProviderTimeoutSeconds:       300,
		OpenAICompatibleTotalTimeout: 300,
	}, []domain.TaskResponse{{
		Task: domain.Task{
			AspectRatio:         "1:1",
			OutputFormat:        "png",
			StructuredInputJSON: structured,
		},
	}})

	if shape.APIMode != "responses" || shape.Endpoint != "/responses" || shape.RequestMode != provider.OpenAICompatibleRequestModeResponsesStream {
		t.Fatalf("unexpected responses request shape: %#v", shape)
	}
	if shape.Model != "gpt-5.5" {
		t.Fatalf("unexpected model override: %#v", shape)
	}
	if !shape.Stream || shape.PartialImages != 2 || shape.TimeoutSeconds != 600 {
		t.Fatalf("unexpected stream fields: %#v", shape)
	}
}
