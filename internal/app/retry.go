package app

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/provider"
)

func (s *Service) scheduleRetry(ctx context.Context, taskID string, attemptID string, attemptNo int, result provider.Result, started time.Time, metrics domain.AttemptMetrics, err error) (bool, error) {
	retryAfter, ok := s.retryAfter(attemptNo, result, err)
	if !ok {
		return false, nil
	}

	code := firstNonEmpty(result.ErrorCode, "provider_retry_scheduled")
	message := firstNonEmpty(result.ErrorMessage, err.Error())
	if finishErr := s.store.FinishAttempt(ctx, attemptID, domain.AttemptFailed, result, started, metrics, &code, &message, &retryAfter); finishErr != nil {
		return false, finishErr
	}
	if updateErr := s.store.UpdateTaskStatus(ctx, taskID, domain.TaskQueued, &code, &message); updateErr != nil {
		return false, updateErr
	}

	delay := time.Until(retryAfter)
	if delay < time.Second {
		delay = time.Second
	}
	if enqueueErr := s.queue.EnqueueAfter(ctx, taskID, delay); enqueueErr != nil {
		_ = s.store.MarkTaskEnqueueFailed(ctx, taskID, enqueueErr)
		return false, enqueueErr
	}
	return true, nil
}

func (s *Service) retryAfter(attemptNo int, result provider.Result, err error) (time.Time, bool) {
	if !isRetryableProviderError(result, err) {
		return time.Time{}, false
	}
	if attemptNo > s.cfg.WorkerMaxRetries {
		return time.Time{}, false
	}
	baseDelay := time.Duration(s.cfg.WorkerRetryBaseDelaySec) * time.Second
	if baseDelay <= 0 {
		baseDelay = 15 * time.Second
	}
	delay := baseDelay
	for i := 1; i < attemptNo; i++ {
		delay *= 2
	}
	return time.Now().UTC().Add(delay), true
}

func isRetryableProviderError(result provider.Result, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}

	code := strings.TrimSpace(strings.ToLower(result.ErrorCode))
	switch {
	case code == "":
		return false
	case strings.HasPrefix(code, "http_5"):
		return true
	case code == "http_408", code == "http_409", code == "http_425", code == "http_429":
		return true
	case strings.Contains(code, "rate_limit"):
		return true
	case strings.Contains(code, "timeout"):
		return true
	case strings.Contains(code, "temporary"):
		return true
	case strings.Contains(code, "unavailable"):
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
