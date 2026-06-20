package main

import (
	"encoding/json"
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
