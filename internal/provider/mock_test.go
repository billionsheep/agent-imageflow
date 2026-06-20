package provider

import (
	"context"
	"testing"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestMockProviderTransientOnce(t *testing.T) {
	p := MockProvider{}
	task := domain.Task{
		ID:             "task_transient_once_test",
		Prompt:         "生成一张图",
		AspectRatio:    "1:1",
		OutputFormat:   "png",
		RequestedCount: 1,
		Provider:       MockProviderID,
		StructuredInputJSON: []byte(`{
			"generation_config": {
				"mock_failure_mode": "transient_once"
			}
		}`),
	}

	firstResult, firstErr := p.Generate(context.Background(), task)
	if firstErr == nil {
		t.Fatal("first Generate should fail")
	}
	if firstResult.ErrorCode != "temporary_unavailable" {
		t.Fatalf("unexpected first error code: %#v", firstResult)
	}

	secondResult, secondErr := p.Generate(context.Background(), task)
	if secondErr != nil {
		t.Fatalf("second Generate should succeed: %v", secondErr)
	}
	if len(secondResult.Files) != 1 {
		t.Fatalf("unexpected second result: %#v", secondResult)
	}
}

func TestMockProviderDelay(t *testing.T) {
	p := MockProvider{}
	task := domain.Task{
		ID:             "task_delay_test",
		Prompt:         "生成一张图",
		AspectRatio:    "1:1",
		OutputFormat:   "png",
		RequestedCount: 1,
		Provider:       MockProviderID,
		StructuredInputJSON: []byte(`{
			"generation_config": {
				"mock_delay_ms": 20
			}
		}`),
	}

	started := time.Now()
	result, err := p.Generate(context.Background(), task)
	if err != nil {
		t.Fatalf("Generate should succeed: %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if elapsed := time.Since(started); elapsed < 15*time.Millisecond {
		t.Fatalf("mock delay was not applied, elapsed=%s", elapsed)
	}
}
