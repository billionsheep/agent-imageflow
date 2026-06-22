package app

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/config"
	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/provider"
)

type recordingProviderAdapter struct {
	mu        sync.Mutex
	counts    []int
	active    int
	maxActive int
	delay     time.Duration
}

func (a *recordingProviderAdapter) Generate(ctx context.Context, task domain.Task) (provider.Result, error) {
	a.mu.Lock()
	a.counts = append(a.counts, task.RequestedCount)
	a.active++
	if a.active > a.maxActive {
		a.maxActive = a.active
	}
	a.mu.Unlock()
	if a.delay > 0 {
		select {
		case <-time.After(a.delay):
		case <-ctx.Done():
			a.mu.Lock()
			a.active--
			a.mu.Unlock()
			return provider.Result{}, ctx.Err()
		}
	}
	a.mu.Lock()
	a.active--
	a.mu.Unlock()

	files := make([]provider.GeneratedFile, 0, task.RequestedCount)
	for i := 0; i < task.RequestedCount; i++ {
		files = append(files, provider.GeneratedFile{
			Slot:          i,
			Bytes:         []byte("png"),
			Thumbnail:     []byte("thumb"),
			MimeType:      "image/png",
			Model:         "fake",
			ParametersRaw: []byte(`{}`),
		})
	}
	return provider.Result{
		ProviderRequestID: "fake",
		Status:            "succeeded",
		Files:             files,
		Metrics: domain.AttemptMetrics{
			ProviderTotalMs: int64(task.RequestedCount),
		},
	}, nil
}

func TestNewProviderLimitersUsesIndependentProviderCaps(t *testing.T) {
	limiters := newProviderLimiters(config.Config{
		OpenAICompatibleMaxConcurrency: 2,
		FalMaxConcurrency:              1,
	})
	openAILimiter := limiters[provider.OpenAICompatibleProviderID]
	if openAILimiter == nil {
		t.Fatal("expected openai-compatible limiter")
	}
	if cap(openAILimiter) != 2 {
		t.Fatalf("openai-compatible limiter cap = %d, want 2", cap(openAILimiter))
	}
	falLimiter := limiters[provider.FalProviderID]
	if falLimiter == nil {
		t.Fatal("expected fal limiter")
	}
	if cap(falLimiter) != 1 {
		t.Fatalf("fal limiter cap = %d, want 1", cap(falLimiter))
	}
}

func TestNewProviderLimitersAllowsDisablingProviderCaps(t *testing.T) {
	limiters := newProviderLimiters(config.Config{
		OpenAICompatibleMaxConcurrency: 0,
		FalMaxConcurrency:              0,
	})
	if limiters[provider.OpenAICompatibleProviderID] != nil {
		t.Fatal("expected no openai-compatible limiter when cap is 0")
	}
	if limiters[provider.FalProviderID] != nil {
		t.Fatal("expected no fal limiter when cap is 0")
	}
}

func TestGenerateWithProviderLimitSplitsRequestedCountByProviderMaxN(t *testing.T) {
	adapter := &recordingProviderAdapter{delay: 25 * time.Millisecond}
	service := &Service{
		providerLimiters: map[string]chan struct{}{
			provider.MockProviderID: make(chan struct{}, 2),
		},
	}
	task := domain.Task{
		ID:             "task_split",
		Provider:       provider.MockProviderID,
		RequestedCount: 10,
		StructuredInputJSON: []byte(`{
			"provider_profile": {
				"enabled": true,
				"provider": "mock",
				"max_n": 4
			}
		}`),
	}

	result, err := service.generateWithProviderLimit(context.Background(), task, adapter)
	if err != nil {
		t.Fatalf("generateWithProviderLimit returned error: %v", err)
	}
	counts := append([]int(nil), adapter.counts...)
	sort.Ints(counts)
	if len(counts) != 3 || counts[0] != 2 || counts[1] != 4 || counts[2] != 4 {
		t.Fatalf("provider requested counts = %#v, want [2 4 4]", counts)
	}
	if adapter.maxActive != 2 {
		t.Fatalf("max concurrent split requests = %d, want 2", adapter.maxActive)
	}
	if len(result.Files) != 10 {
		t.Fatalf("generated files = %d, want 10", len(result.Files))
	}
	for index, file := range result.Files {
		if file.Slot != index {
			t.Fatalf("file %d slot = %d, want %d", index, file.Slot, index)
		}
	}
}

func TestGenerateWithProviderLimitDefaultsOpenAICompatibleToSingleImageRequests(t *testing.T) {
	adapter := &recordingProviderAdapter{delay: 10 * time.Millisecond}
	service := &Service{
		providerLimiters: map[string]chan struct{}{
			provider.OpenAICompatibleProviderID: make(chan struct{}, 2),
		},
	}
	task := domain.Task{
		ID:             "task_openai_split_default",
		Provider:       provider.OpenAICompatibleProviderID,
		RequestedCount: 3,
	}

	result, err := service.generateWithProviderLimit(context.Background(), task, adapter)
	if err != nil {
		t.Fatalf("generateWithProviderLimit returned error: %v", err)
	}
	counts := append([]int(nil), adapter.counts...)
	sort.Ints(counts)
	if len(counts) != 3 || counts[0] != 1 || counts[1] != 1 || counts[2] != 1 {
		t.Fatalf("provider requested counts = %#v, want [1 1 1]", counts)
	}
	if adapter.maxActive != 2 {
		t.Fatalf("max concurrent split requests = %d, want 2", adapter.maxActive)
	}
	if len(result.Files) != 3 {
		t.Fatalf("generated files = %d, want 3", len(result.Files))
	}
}

func TestRetryAfterForRetryableHTTPError(t *testing.T) {
	service := &Service{cfg: config.Config{
		WorkerMaxRetries:        3,
		WorkerRetryBaseDelaySec: 2,
	}}

	before := time.Now().UTC()
	retryAfter, ok := service.retryAfter(2, provider.Result{ErrorCode: "http_429"}, context.DeadlineExceeded)
	if !ok {
		t.Fatal("retryAfter should be enabled for retryable error")
	}
	delay := retryAfter.Sub(before)
	if delay < 4*time.Second || delay > 5*time.Second {
		t.Fatalf("unexpected retry delay: %s", delay)
	}
}

func TestRetryAfterStopsAfterMaxRetries(t *testing.T) {
	service := &Service{cfg: config.Config{
		WorkerMaxRetries:        1,
		WorkerRetryBaseDelaySec: 2,
	}}

	if _, ok := service.retryAfter(2, provider.Result{ErrorCode: "http_500"}, context.DeadlineExceeded); ok {
		t.Fatal("retryAfter should stop after max retries")
	}
}

func TestIsRetryableProviderError(t *testing.T) {
	tests := []struct {
		name   string
		result provider.Result
		err    error
		want   bool
	}{
		{
			name:   "http_500",
			result: provider.Result{ErrorCode: "http_500"},
			err:    context.DeadlineExceeded,
			want:   true,
		},
		{
			name:   "temporary_unavailable",
			result: provider.Result{ErrorCode: "temporary_unavailable"},
			err:    errors.New("temporary upstream failure"),
			want:   true,
		},
		{
			name:   "invalid_response",
			result: provider.Result{ErrorCode: "invalid_response"},
			err:    errors.New("provider returned malformed payload"),
			want:   false,
		},
		{
			name:   "context_canceled",
			result: provider.Result{ErrorCode: "http_429"},
			err:    context.Canceled,
			want:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isRetryableProviderError(tc.result, tc.err)
			if got != tc.want {
				t.Fatalf("isRetryableProviderError() = %v, want %v", got, tc.want)
			}
		})
	}
}
