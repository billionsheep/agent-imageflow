package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/config"
	"github.com/billionsheep/agent-imageflow/internal/provider"
)

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
