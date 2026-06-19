package provider

import (
	"context"
	"testing"

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
