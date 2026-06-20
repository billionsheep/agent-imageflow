package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

type fakeRateLimiter struct {
	decisions map[string]RateLimitDecision
	errs      map[string]error
	calls     []rateLimitCall
}

type rateLimitCall struct {
	key    string
	max    int
	window time.Duration
}

func (f *fakeRateLimiter) Allow(_ context.Context, key string, maxRequests int, window time.Duration) (RateLimitDecision, error) {
	f.calls = append(f.calls, rateLimitCall{
		key:    key,
		max:    maxRequests,
		window: window,
	})
	if err, ok := f.errs[key]; ok {
		return RateLimitDecision{}, err
	}
	if decision, ok := f.decisions[key]; ok {
		return decision, nil
	}
	return RateLimitDecision{Allowed: true, RetryAfter: time.Second}, nil
}

func (f *fakeRateLimiter) Close() error {
	return nil
}

func TestEnforceRateLimitsInstanceLimitExceeded(t *testing.T) {
	limiter := &fakeRateLimiter{
		decisions: map[string]RateLimitDecision{
			rateLimitInstanceKey(): {
				Allowed:    false,
				Count:      3,
				RetryAfter: 4 * time.Second,
			},
		},
	}
	server := &Server{
		options: Options{
			RateLimiter:                  limiter,
			RateLimitWindow:              time.Minute,
			RateLimitInstanceMaxRequests: 2,
		},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/workspaces", nil)

	allowed := server.enforceRateLimits(recorder, request, requestAuthScope{})
	if allowed {
		t.Fatalf("expected request to be rate limited")
	}
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Retry-After"); got != "4" {
		t.Fatalf("expected Retry-After=4, got %q", got)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if got := body["error_code"]; got != "rate_limited" {
		t.Fatalf("expected error_code=rate_limited, got %v", got)
	}
	if got := body["rate_limit_scope"]; got != "instance" {
		t.Fatalf("expected rate_limit_scope=instance, got %v", got)
	}
	if len(limiter.calls) != 1 {
		t.Fatalf("expected 1 limiter call, got %d", len(limiter.calls))
	}
	if limiter.calls[0].key != rateLimitInstanceKey() {
		t.Fatalf("expected instance limiter key %q, got %q", rateLimitInstanceKey(), limiter.calls[0].key)
	}
}

func TestEnforceRateLimitsProjectLimitExceeded(t *testing.T) {
	scope := requestAuthScope{
		WorkspaceID: "ws_default",
		ProjectID:   "prj_demo",
	}
	projectKey := rateLimitProjectKey(scope)
	limiter := &fakeRateLimiter{
		decisions: map[string]RateLimitDecision{
			rateLimitInstanceKey(): {
				Allowed:    true,
				Count:      1,
				RetryAfter: 2 * time.Second,
			},
			projectKey: {
				Allowed:    false,
				Count:      2,
				RetryAfter: 7 * time.Second,
			},
		},
	}
	server := &Server{
		options: Options{
			RateLimiter:                  limiter,
			RateLimitWindow:              time.Minute,
			RateLimitInstanceMaxRequests: 10,
			RateLimitProjectMaxRequests:  1,
		},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/tasks/task_123", nil)

	allowed := server.enforceRateLimits(recorder, request, scope)
	if allowed {
		t.Fatalf("expected request to be rate limited")
	}
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Retry-After"); got != "7" {
		t.Fatalf("expected Retry-After=7, got %q", got)
	}

	if len(limiter.calls) != 2 {
		t.Fatalf("expected 2 limiter calls, got %d", len(limiter.calls))
	}
	if limiter.calls[0].key != rateLimitInstanceKey() {
		t.Fatalf("expected first limiter key %q, got %q", rateLimitInstanceKey(), limiter.calls[0].key)
	}
	if limiter.calls[1].key != projectKey {
		t.Fatalf("expected second limiter key %q, got %q", projectKey, limiter.calls[1].key)
	}
}

func TestEnforceRateLimitsSkipsProjectLimitWithoutProjectScope(t *testing.T) {
	limiter := &fakeRateLimiter{
		decisions: map[string]RateLimitDecision{
			rateLimitInstanceKey(): {
				Allowed:    true,
				Count:      1,
				RetryAfter: 2 * time.Second,
			},
		},
	}
	server := &Server{
		options: Options{
			RateLimiter:                  limiter,
			RateLimitWindow:              time.Minute,
			RateLimitInstanceMaxRequests: 10,
			RateLimitProjectMaxRequests:  1,
		},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/workspaces", nil)

	allowed := server.enforceRateLimits(recorder, request, requestAuthScope{})
	if !allowed {
		t.Fatalf("expected request to pass when only instance limit is in scope")
	}
	if len(limiter.calls) != 1 {
		t.Fatalf("expected only 1 limiter call, got %d", len(limiter.calls))
	}
	if limiter.calls[0].key != rateLimitInstanceKey() {
		t.Fatalf("expected instance limiter key %q, got %q", rateLimitInstanceKey(), limiter.calls[0].key)
	}
}

func TestEnforceRateLimitsBackendErrorFailsOpen(t *testing.T) {
	limiter := &fakeRateLimiter{
		errs: map[string]error{
			rateLimitInstanceKey(): errors.New("redis unavailable"),
		},
	}
	server := &Server{
		options: Options{
			RateLimiter:                  limiter,
			RateLimitWindow:              time.Minute,
			RateLimitInstanceMaxRequests: 1,
		},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/workspaces", nil)

	allowed := server.enforceRateLimits(recorder, request, requestAuthScope{})
	if !allowed {
		t.Fatalf("expected request to pass when limiter backend errors")
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty response body on fail-open, got %q", recorder.Body.String())
	}
	if recorder.Header().Get("Retry-After") != "" {
		t.Fatalf("expected no Retry-After header on fail-open")
	}
}

func TestEnrichHTTPAuditEventFromResponse(t *testing.T) {
	event := domain.HTTPAuditEvent{
		Action: "create_task",
	}
	parts := []string{"api", "workspaces", "ws_default", "projects", "prj_demo", "campaigns", "cmp_demo", "tasks"}
	body := []byte(`{"task_id":"task_123","workspace_id":"ws_default","project_id":"prj_demo","campaign_id":"cmp_demo","error_code":"rate_limited","error_message":"retry later"}`)

	enrichHTTPAuditEvent(&event, parts, body)

	if event.TaskID != "task_123" {
		t.Fatalf("expected task_id from response body, got %q", event.TaskID)
	}
	if event.WorkspaceID != "ws_default" || event.ProjectID != "prj_demo" || event.CampaignID != "cmp_demo" {
		t.Fatalf("expected scope ids from response body, got %#v", event)
	}
	if event.ErrorCode != "rate_limited" || event.ErrorMessage != "retry later" {
		t.Fatalf("expected error payload to be extracted, got %#v", event)
	}
}

func TestInferAuditRoute(t *testing.T) {
	route, action := inferAuditRoute([]string{"api", "assets", "asset_1", "thumbnail"}, http.MethodGet)
	if route != "/api/assets/{asset_id}/thumbnail" {
		t.Fatalf("unexpected route %q", route)
	}
	if action != "get_asset_thumbnail" {
		t.Fatalf("unexpected action %q", action)
	}

	route, action = inferAuditRoute([]string{"api", "tasks", "task_1", "attempts"}, http.MethodGet)
	if route != "/api/tasks/{task_id}/attempts" {
		t.Fatalf("unexpected attempts route %q", route)
	}
	if action != "list_task_attempts" {
		t.Fatalf("unexpected attempts action %q", action)
	}

	route, action = inferAuditRoute([]string{"api", "projects", "prj_demo", "campaigns", "cmp_demo", "batch-progress"}, http.MethodGet)
	if route != "/api/projects/{project_id}/campaigns/{campaign_id}/batch-progress" {
		t.Fatalf("unexpected batch-progress route %q", route)
	}
	if action != "get_batch_progress" {
		t.Fatalf("unexpected batch-progress action %q", action)
	}
}

func TestStorageGovernanceRouteUsesProjectScopeAuth(t *testing.T) {
	server := &Server{}
	tests := []struct {
		name       string
		leaf       string
		wantAction string
	}{
		{name: "governance", leaf: "storage-governance", wantAction: "get_storage_governance"},
		{name: "integrity", leaf: "storage-integrity", wantAction: "get_storage_integrity"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parts := []string{"api", "workspaces", "ws_default", "projects", "prj_demo", "campaigns", "cmp_demo", tc.leaf}
			scope, ok, err := server.resolveRequestAuthScope(
				httptest.NewRequest(http.MethodGet, "/api/workspaces/ws_default/projects/prj_demo/campaigns/cmp_demo/"+tc.leaf, nil),
				parts,
			)
			if err != nil {
				t.Fatalf("resolveRequestAuthScope returned error: %v", err)
			}
			if !ok {
				t.Fatalf("expected %s route to resolve auth scope", tc.leaf)
			}
			if scope.WorkspaceID != "ws_default" || scope.ProjectID != "prj_demo" || scope.CampaignID != "cmp_demo" {
				t.Fatalf("unexpected scope: %#v", scope)
			}
			if scope.AllowBasicOnly {
				t.Fatalf("%s should use project API key rules when configured", tc.leaf)
			}

			route, action := inferAuditRoute(parts, http.MethodGet)
			wantRoute := "/api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/" + tc.leaf
			if route != wantRoute {
				t.Fatalf("unexpected route %q", route)
			}
			if action != tc.wantAction {
				t.Fatalf("unexpected action %q", action)
			}
		})
	}
}

func TestProviderProfileRouteUsesProjectScopeAuth(t *testing.T) {
	server := &Server{}
	parts := []string{"api", "workspaces", "ws_default", "projects", "prj_demo", "provider-profile"}
	scope, ok, err := server.resolveRequestAuthScope(
		httptest.NewRequest(http.MethodGet, "/api/workspaces/ws_default/projects/prj_demo/provider-profile", nil),
		parts,
	)
	if err != nil {
		t.Fatalf("resolveRequestAuthScope returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected provider-profile route to resolve auth scope")
	}
	if scope.WorkspaceID != "ws_default" || scope.ProjectID != "prj_demo" {
		t.Fatalf("unexpected scope: %#v", scope)
	}
	if scope.AllowBasicOnly {
		t.Fatal("provider-profile should use project API key rules")
	}

	route, action := inferAuditRoute(parts, http.MethodGet)
	if route != "/api/workspaces/{workspace_id}/projects/{project_id}/provider-profile" {
		t.Fatalf("unexpected route %q", route)
	}
	if action != "get_provider_profile" {
		t.Fatalf("unexpected action %q", action)
	}
}

func TestParseAssetListQuery(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/projects/prj_demo/campaigns/cmp_demo/assets?limit=500&offset=7&status=selected&provider=mock&model=mock-image&source=mcp&session_id=s1&batch_id=b1&keyword=hero&created_from=2026-06-19T01:02:03Z&created_to=2026-06-19T02:03:04Z", nil)

	query, err := parseAssetListQuery(request, "prj_demo", "cmp_demo")
	if err != nil {
		t.Fatalf("parseAssetListQuery returned error: %v", err)
	}
	if query.ProjectID != "prj_demo" || query.CampaignID != "cmp_demo" {
		t.Fatalf("unexpected scope: %#v", query)
	}
	if query.Limit != 500 || query.Offset != 7 || query.Status != "selected" || query.Provider != "mock" || query.Model != "mock-image" {
		t.Fatalf("basic filters were not parsed: %#v", query)
	}
	if query.Source != "mcp" || query.SessionID != "s1" || query.BatchID != "b1" || query.Keyword != "hero" {
		t.Fatalf("metadata filters were not parsed: %#v", query)
	}
	if query.CreatedFrom == nil || query.CreatedTo == nil {
		t.Fatalf("date filters were not parsed: %#v", query)
	}
}

func TestParseBatchProgressQuery(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/projects/prj_demo/campaigns/cmp_demo/batch-progress?session_id=s1&batch_id=b1&limit=500", nil)

	query, err := parseBatchProgressQuery(request, "prj_demo", "cmp_demo")
	if err != nil {
		t.Fatalf("parseBatchProgressQuery returned error: %v", err)
	}
	if query.ProjectID != "prj_demo" || query.CampaignID != "cmp_demo" || query.SessionID != "s1" || query.BatchID != "b1" {
		t.Fatalf("unexpected query: %#v", query)
	}
	if query.Limit != domain.MaxBatchProgressLimit {
		t.Fatalf("limit = %d, want cap %d", query.Limit, domain.MaxBatchProgressLimit)
	}
}
