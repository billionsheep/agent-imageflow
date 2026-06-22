package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

type fakeRateLimiter struct {
	decisions map[string]RateLimitDecision
	errs      map[string]error
	calls     []rateLimitCall
}

func TestAdminLoginMeLogoutSessionFlow(t *testing.T) {
	server := &Server{
		options: Options{
			AdminUsername:   "admin",
			AdminPassword:   "secret",
			AdminSessionTTL: time.Hour,
		},
	}

	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewBufferString(`{"username":"admin","password":"secret"}`))
	server.ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", loginRecorder.Code, loginRecorder.Body.String())
	}
	setCookie := loginRecorder.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, adminSessionCookieName+"=") || !strings.Contains(setCookie, "HttpOnly") {
		t.Fatalf("expected admin session HttpOnly cookie, got %q", setCookie)
	}

	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected login cookie")
	}
	meRecorder := httptest.NewRecorder()
	meRequest := httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
	meRequest.AddCookie(cookies[0])
	server.ServeHTTP(meRecorder, meRequest)
	if meRecorder.Code != http.StatusOK {
		t.Fatalf("me status = %d body=%s", meRecorder.Code, meRecorder.Body.String())
	}
	var me adminSessionResponse
	if err := json.Unmarshal(meRecorder.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if !me.Authenticated || me.Username != "admin" || !me.Configured {
		t.Fatalf("unexpected me response: %#v", me)
	}

	logoutRecorder := httptest.NewRecorder()
	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/admin/logout", nil)
	server.ServeHTTP(logoutRecorder, logoutRequest)
	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf("logout status = %d body=%s", logoutRecorder.Code, logoutRecorder.Body.String())
	}
	if !strings.Contains(logoutRecorder.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("expected logout to clear cookie, got %q", logoutRecorder.Header().Get("Set-Cookie"))
	}
}

func TestAdminLoginDisabledWithoutPassword(t *testing.T) {
	server := &Server{options: Options{AdminUsername: "admin"}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewBufferString(`{"username":"admin","password":"secret"}`))

	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected admin_not_configured status 503, got %d", recorder.Code)
	}
}

func TestAuthorizeRequestAllowsAdminSessionForRecentAssets(t *testing.T) {
	server := &Server{
		options: Options{
			AdminUsername:   "admin",
			AdminPassword:   "secret",
			AdminSessionTTL: time.Hour,
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/admin/assets/recent?limit=24", nil)
	request.AddCookie(server.newAdminSessionCookie("admin", time.Now().Add(time.Hour)))
	recorder := httptest.NewRecorder()

	authorized, scope, actor, err := server.authorizeRequest(recorder, request, []string{"api", "admin", "assets", "recent"})
	if err != nil {
		t.Fatalf("authorizeRequest returned error: %v", err)
	}
	if !authorized {
		t.Fatalf("expected admin session to authorize recent assets, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !scope.RequireAdmin || !scope.AllowAdmin {
		t.Fatalf("expected recent assets scope to require admin: %#v", scope)
	}
	if actor.AuthMode != "admin_session" || actor.Actor != "admin" {
		t.Fatalf("unexpected audit actor: %#v", actor)
	}
}

func TestPromoteInputFileAssetRouteAllowsAdminAndAuditsAction(t *testing.T) {
	parts := []string{"api", "workspaces", "ws_demo", "projects", "prj_demo", "campaigns", "cmp_demo", "input-files", "inp_demo", "promote-asset"}
	if !routeAllowsAdminSession(parts, http.MethodPost) {
		t.Fatal("expected promote input-file asset route to allow admin session")
	}
	route, action := inferAuditRoute(parts, http.MethodPost)
	if route != "/api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/input-files/{input_file_id}/promote-asset" ||
		action != "promote_input_file_asset" {
		t.Fatalf("unexpected audit route/action: %s %s", route, action)
	}
	server := &Server{}
	request := httptest.NewRequest(http.MethodPost, "/api/workspaces/ws_demo/projects/prj_demo/campaigns/cmp_demo/input-files/inp_demo/promote-asset", nil)
	scope, ok, err := server.resolveRequestAuthScope(request, parts)
	if err != nil {
		t.Fatalf("resolveRequestAuthScope returned error: %v", err)
	}
	if !ok || scope.WorkspaceID != "ws_demo" || scope.ProjectID != "prj_demo" || scope.CampaignID != "cmp_demo" || scope.InputFileID != "inp_demo" {
		t.Fatalf("unexpected auth scope: ok=%v scope=%#v", ok, scope)
	}
}

func TestAuthorizeRequestRejectsAnonymousRecentAssets(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/admin/assets/recent?limit=24", nil)

	authorized, _, _, err := server.authorizeRequest(recorder, request, []string{"api", "admin", "assets", "recent"})
	if err != nil {
		t.Fatalf("authorizeRequest returned error: %v", err)
	}
	if authorized {
		t.Fatal("expected anonymous recent assets request to be rejected")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAssetDeliveryUnauthorizedDoesNotSendBasicChallenge(t *testing.T) {
	server := &Server{
		options: Options{
			BasicAuthUsername: "basic",
			BasicAuthPassword: "secret",
		},
	}
	for _, parts := range [][]string{
		{"api", "assets", "asset_1", "thumbnail"},
		{"api", "assets", "asset_1", "original"},
		{"api", "assets", "asset_1", "metadata"},
	} {
		if !routeAllowsAdminSession(parts, http.MethodGet) {
			t.Fatalf("expected delivery route %v to allow admin session", parts)
		}
		if shouldSendBasicChallenge(parts, http.MethodGet) {
			t.Fatalf("delivery route %v should suppress browser Basic challenge", parts)
		}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/"+strings.Join(parts, "/"), nil)

		authorized, _, _, err := server.authorizeRequest(recorder, request, parts)
		if err != nil {
			t.Fatalf("authorizeRequest returned error for %v: %v", parts, err)
		}
		if authorized {
			t.Fatalf("expected anonymous delivery request to be rejected for %v", parts)
		}
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for %v, got %d body=%s", parts, recorder.Code, recorder.Body.String())
		}
		if got := recorder.Header().Get("WWW-Authenticate"); got != "" {
			t.Fatalf("delivery route %v should not send browser Basic challenge, got %q", parts, got)
		}
	}
}

func TestRuntimeStatusRequiresAdminAndRedactsSecrets(t *testing.T) {
	server := &Server{
		options: Options{
			BasicAuthUsername: "basic",
			BasicAuthPassword: "basic-secret",
			AdminUsername:     "admin",
			AdminPassword:     "admin-secret",
			AdminSessionTTL:   time.Hour,
			Runtime: RuntimeStatusOptions{
				PublicBaseURL:                  "https://imageflow.example.com",
				DefaultProvider:                "openai-compatible",
				OpenAICompatibleModel:          "gpt-image-2",
				OpenAICompatibleConfigured:     true,
				OpenAICompatibleMaxConcurrency: 2,
				FalModel:                       "fal-model",
				FalConfigured:                  false,
				FalMaxConcurrency:              1,
				ProviderTimeoutSeconds:         300,
				WorkerConcurrency:              1,
				RateLimitWindowSeconds:         60,
				RateLimitInstanceMaxRequests:   120,
				RateLimitProjectMaxRequests:    60,
			},
		},
	}

	unauthorizedRecorder := httptest.NewRecorder()
	server.ServeHTTP(unauthorizedRecorder, httptest.NewRequest(http.MethodGet, "/api/admin/runtime-status", nil))
	if unauthorizedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized runtime status without admin session, got %d", unauthorizedRecorder.Code)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/admin/runtime-status", nil)
	request.AddCookie(server.newAdminSessionCookie("admin", time.Now().Add(time.Hour)))
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("runtime status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, forbidden := range []string{"basic-secret", "admin-secret", "api_key", "password", "secret", "cookie", "token"} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(forbidden)) {
			t.Fatalf("runtime status leaked forbidden text %q in body %s", forbidden, body)
		}
	}
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime status: %v", err)
	}
	if payload["authenticated"] != true || payload["username"] != "admin" {
		t.Fatalf("unexpected runtime auth payload: %#v", payload)
	}
	if payload["default_provider"] != "openai-compatible" {
		t.Fatalf("unexpected default provider: %#v", payload)
	}
	providers, ok := payload["providers"].(map[string]any)
	if !ok {
		t.Fatalf("expected providers map, got %#v", payload["providers"])
	}
	openai, ok := providers["openai_compatible"].(map[string]any)
	if !ok || openai["configured"] != true || openai["model"] != "gpt-image-2" {
		t.Fatalf("unexpected openai-compatible status: %#v", providers["openai_compatible"])
	}
}

func TestSetCORSEchoesOriginWhenCredentialsAreAllowed(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/admin/me", nil)
	request.Header.Set("Origin", "http://localhost:8080")

	server.setCORS(recorder, request)
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:8080" {
		t.Fatalf("unexpected CORS origin %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials CORS header, got %q", got)
	}
	if methods := recorder.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(methods, "PATCH") || !strings.Contains(methods, "DELETE") {
		t.Fatalf("expected PATCH/DELETE in CORS methods, got %q", methods)
	}
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

	route, action = inferAuditRoute([]string{"api", "assets", "asset_1", "metadata"}, http.MethodGet)
	if route != "/api/assets/{asset_id}/metadata" {
		t.Fatalf("unexpected metadata route %q", route)
	}
	if action != "get_asset_metadata" {
		t.Fatalf("unexpected metadata action %q", action)
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

	route, action = inferAuditRoute([]string{"api", "projects", "prj_demo", "campaigns", "cmp_demo", "batch-summary"}, http.MethodGet)
	if route != "/api/projects/{project_id}/campaigns/{campaign_id}/batch-summary" {
		t.Fatalf("unexpected batch-summary route %q", route)
	}
	if action != "get_batch_summary" {
		t.Fatalf("unexpected batch-summary action %q", action)
	}

	route, action = inferAuditRoute([]string{"api", "projects", "prj_demo", "campaigns", "cmp_demo", "batch-manifest"}, http.MethodGet)
	if route != "/api/projects/{project_id}/campaigns/{campaign_id}/batch-manifest" {
		t.Fatalf("unexpected batch-manifest route %q", route)
	}
	if action != "get_batch_manifest" {
		t.Fatalf("unexpected batch-manifest action %q", action)
	}

	route, action = inferAuditRoute([]string{"api", "admin", "assets", "recent"}, http.MethodGet)
	if route != "/api/admin/assets/recent" {
		t.Fatalf("unexpected recent assets route %q", route)
	}
	if action != "list_recent_assets" {
		t.Fatalf("unexpected recent assets action %q", action)
	}
}

func TestBatchSummaryRouteUsesProjectScopeAuth(t *testing.T) {
	server := &Server{}
	parts := []string{"api", "projects", "prj_demo", "campaigns", "cmp_demo", "batch-summary"}
	scope, ok, err := server.resolveRequestAuthScope(
		httptest.NewRequest(http.MethodGet, "/api/projects/prj_demo/campaigns/cmp_demo/batch-summary", nil),
		parts,
	)
	if err != nil {
		t.Fatalf("resolveRequestAuthScope returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected batch-summary route to resolve auth scope")
	}
	if scope.ProjectID != "prj_demo" || scope.CampaignID != "cmp_demo" || !scope.AllowAdmin {
		t.Fatalf("unexpected scope: %#v", scope)
	}
	if !routeAllowsAdminSession(parts, http.MethodGet) {
		t.Fatal("batch-summary should allow admin session reads")
	}
}

func TestBatchManifestRouteUsesProjectScopeAuth(t *testing.T) {
	server := &Server{}
	parts := []string{"api", "projects", "prj_demo", "campaigns", "cmp_demo", "batch-manifest"}
	scope, ok, err := server.resolveRequestAuthScope(
		httptest.NewRequest(http.MethodGet, "/api/projects/prj_demo/campaigns/cmp_demo/batch-manifest", nil),
		parts,
	)
	if err != nil {
		t.Fatalf("resolveRequestAuthScope returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected batch-manifest route to resolve auth scope")
	}
	if scope.ProjectID != "prj_demo" || scope.CampaignID != "cmp_demo" || !scope.AllowAdmin {
		t.Fatalf("unexpected scope: %#v", scope)
	}
	if !routeAllowsAdminSession(parts, http.MethodGet) {
		t.Fatal("batch-manifest should allow admin session reads")
	}
}

func TestSceneRegenerationRouteUsesProjectScopeAuth(t *testing.T) {
	server := &Server{}
	parts := []string{"api", "projects", "prj_demo", "campaigns", "cmp_demo", "scene-regenerations"}
	scope, ok, err := server.resolveRequestAuthScope(
		httptest.NewRequest(http.MethodPost, "/api/projects/prj_demo/campaigns/cmp_demo/scene-regenerations", nil),
		parts,
	)
	if err != nil {
		t.Fatalf("resolveRequestAuthScope returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected scene-regenerations route to resolve auth scope")
	}
	if scope.ProjectID != "prj_demo" || scope.CampaignID != "cmp_demo" || !scope.AllowAdmin {
		t.Fatalf("unexpected scope: %#v", scope)
	}
	if !routeAllowsAdminSession(parts, http.MethodPost) {
		t.Fatal("scene-regenerations should allow admin session writes")
	}

	route, action := inferAuditRoute(parts, http.MethodPost)
	if route != "/api/projects/{project_id}/campaigns/{campaign_id}/scene-regenerations" {
		t.Fatalf("unexpected route %q", route)
	}
	if action != "regenerate_scene" {
		t.Fatalf("unexpected action %q", action)
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

func TestParseBatchStorySummaryQuery(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/projects/prj_demo/campaigns/cmp_demo/batch-summary?session_id=s1&batch_id=b1&story_id=story_1&source=codex&status=completed&include_setup=true&limit=999", nil)

	query, err := parseBatchStorySummaryQuery(request, "prj_demo", "cmp_demo")
	if err != nil {
		t.Fatalf("parseBatchStorySummaryQuery returned error: %v", err)
	}
	if query.ProjectID != "prj_demo" || query.CampaignID != "cmp_demo" || query.SessionID != "s1" || query.BatchID != "b1" {
		t.Fatalf("unexpected scope/query ids: %#v", query)
	}
	if query.StoryID != "story_1" || query.Source != "codex" || query.Status != "completed" || !query.IncludeSetup {
		t.Fatalf("optional filters were not parsed: %#v", query)
	}
	if query.Limit != domain.MaxBatchProgressLimit {
		t.Fatalf("limit = %d, want cap %d", query.Limit, domain.MaxBatchProgressLimit)
	}
}

func TestParseBatchManifestQuery(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/projects/prj_demo/campaigns/cmp_demo/batch-manifest?session_id=s1&batch_id=b1&story_id=story_1&source=codex&status=completed&include_setup=true&limit=999&selected_only=false&include_rejected=true", nil)

	query, err := parseBatchManifestQuery(request, "prj_demo", "cmp_demo")
	if err != nil {
		t.Fatalf("parseBatchManifestQuery returned error: %v", err)
	}
	if query.ProjectID != "prj_demo" || query.CampaignID != "cmp_demo" || query.SessionID != "s1" || query.BatchID != "b1" {
		t.Fatalf("unexpected scope/query ids: %#v", query)
	}
	if query.StoryID != "story_1" || query.Source != "codex" || query.Status != "completed" || !query.IncludeSetup {
		t.Fatalf("optional filters were not parsed: %#v", query)
	}
	if query.SelectedOnly || !query.IncludeRejected {
		t.Fatalf("manifest options were not parsed: %#v", query)
	}
	if query.Limit != domain.MaxBatchProgressLimit {
		t.Fatalf("limit = %d, want cap %d", query.Limit, domain.MaxBatchProgressLimit)
	}
}

func TestParseBatchManifestQueryDefaultsSelectedOnly(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/projects/prj_demo/campaigns/cmp_demo/batch-manifest?session_id=s1", nil)

	query, err := parseBatchManifestQuery(request, "prj_demo", "cmp_demo")
	if err != nil {
		t.Fatalf("parseBatchManifestQuery returned error: %v", err)
	}
	if !query.SelectedOnly || query.IncludeRejected {
		t.Fatalf("unexpected manifest defaults: %#v", query)
	}
}

func TestParseBatchManifestQueryRequiresSessionOrBatch(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/projects/prj_demo/campaigns/cmp_demo/batch-manifest", nil)

	if _, err := parseBatchManifestQuery(request, "prj_demo", "cmp_demo"); err == nil {
		t.Fatal("expected missing session_id and batch_id to fail")
	}
}
