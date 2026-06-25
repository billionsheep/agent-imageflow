package config

import "testing"

func TestLoadOpenAICompatibleMaxConcurrency(t *testing.T) {
	t.Setenv("OPENAI_COMPATIBLE_MAX_CONCURRENCY", "")
	if got := Load().OpenAICompatibleMaxConcurrency; got != 3 {
		t.Fatalf("default OPENAI_COMPATIBLE_MAX_CONCURRENCY = %d, want 3", got)
	}

	t.Setenv("OPENAI_COMPATIBLE_MAX_CONCURRENCY", "4")
	if got := Load().OpenAICompatibleMaxConcurrency; got != 4 {
		t.Fatalf("override OPENAI_COMPATIBLE_MAX_CONCURRENCY = %d, want 4", got)
	}

	t.Setenv("OPENAI_COMPATIBLE_MAX_CONCURRENCY", "-1")
	if got := Load().OpenAICompatibleMaxConcurrency; got != 3 {
		t.Fatalf("invalid OPENAI_COMPATIBLE_MAX_CONCURRENCY = %d, want fallback 3", got)
	}
}

func TestLoadFalMaxConcurrency(t *testing.T) {
	t.Setenv("FAL_MAX_CONCURRENCY", "")
	if got := Load().FalMaxConcurrency; got != 3 {
		t.Fatalf("default FAL_MAX_CONCURRENCY = %d, want 3", got)
	}

	t.Setenv("FAL_MAX_CONCURRENCY", "2")
	if got := Load().FalMaxConcurrency; got != 2 {
		t.Fatalf("override FAL_MAX_CONCURRENCY = %d, want 2", got)
	}

	t.Setenv("FAL_MAX_CONCURRENCY", "-1")
	if got := Load().FalMaxConcurrency; got != 3 {
		t.Fatalf("invalid FAL_MAX_CONCURRENCY = %d, want fallback 3", got)
	}
}

func TestLoadProviderTimeoutProfile(t *testing.T) {
	t.Setenv("PROVIDER_TIMEOUT_SECONDS", "")
	t.Setenv("OPENAI_COMPATIBLE_CONNECT_TIMEOUT_SECONDS", "")
	t.Setenv("OPENAI_COMPATIBLE_RESPONSE_HEADER_TIMEOUT_SECONDS", "")
	t.Setenv("OPENAI_COMPATIBLE_TOTAL_TIMEOUT_SECONDS", "")
	cfg := Load()
	if cfg.ProviderTimeoutSeconds != 300 {
		t.Fatalf("default PROVIDER_TIMEOUT_SECONDS = %d, want 300", cfg.ProviderTimeoutSeconds)
	}
	if cfg.OpenAICompatibleConnectTimeout != 30 {
		t.Fatalf("default connect timeout = %d, want 30", cfg.OpenAICompatibleConnectTimeout)
	}
	if cfg.OpenAICompatibleHeaderTimeout != 300 || cfg.OpenAICompatibleTotalTimeout != 300 {
		t.Fatalf("default openai timeout profile = header %d total %d, want 300/300", cfg.OpenAICompatibleHeaderTimeout, cfg.OpenAICompatibleTotalTimeout)
	}

	t.Setenv("PROVIDER_TIMEOUT_SECONDS", "240")
	cfg = Load()
	if cfg.ProviderTimeoutSeconds != 240 || cfg.OpenAICompatibleHeaderTimeout != 240 || cfg.OpenAICompatibleTotalTimeout != 240 {
		t.Fatalf("legacy provider timeout should feed header/total fallback, got provider=%d header=%d total=%d", cfg.ProviderTimeoutSeconds, cfg.OpenAICompatibleHeaderTimeout, cfg.OpenAICompatibleTotalTimeout)
	}

	t.Setenv("OPENAI_COMPATIBLE_CONNECT_TIMEOUT_SECONDS", "5")
	t.Setenv("OPENAI_COMPATIBLE_RESPONSE_HEADER_TIMEOUT_SECONDS", "60")
	t.Setenv("OPENAI_COMPATIBLE_TOTAL_TIMEOUT_SECONDS", "180")
	cfg = Load()
	if cfg.OpenAICompatibleConnectTimeout != 5 || cfg.OpenAICompatibleHeaderTimeout != 60 || cfg.OpenAICompatibleTotalTimeout != 180 {
		t.Fatalf("explicit openai timeout profile was not applied: %#v", cfg)
	}
}

func TestLoadWorkerConcurrency(t *testing.T) {
	t.Setenv("WORKER_CONCURRENCY", "")
	if got := Load().WorkerConcurrency; got != 6 {
		t.Fatalf("default WORKER_CONCURRENCY = %d, want 6", got)
	}

	t.Setenv("WORKER_CONCURRENCY", "4")
	if got := Load().WorkerConcurrency; got != 4 {
		t.Fatalf("override WORKER_CONCURRENCY = %d, want 4", got)
	}

	t.Setenv("WORKER_CONCURRENCY", "0")
	if got := Load().WorkerConcurrency; got != 6 {
		t.Fatalf("invalid WORKER_CONCURRENCY = %d, want fallback 6", got)
	}
}

func TestLoadAdminCredentialsFallbackToBasicAuth(t *testing.T) {
	t.Setenv("BASIC_AUTH_USERNAME", "basic-admin")
	t.Setenv("BASIC_AUTH_PASSWORD", "basic-secret")
	t.Setenv("ADMIN_USERNAME", "")
	t.Setenv("ADMIN_PASSWORD", "")
	t.Setenv("ADMIN_SESSION_SECRET", "")
	t.Setenv("ADMIN_SESSION_TTL_SECONDS", "")

	cfg := Load()
	if cfg.AdminUsername != "basic-admin" || cfg.AdminPassword != "basic-secret" {
		t.Fatalf("admin credentials should fall back to basic auth, got username=%q password=%q", cfg.AdminUsername, cfg.AdminPassword)
	}
	if cfg.AdminSessionTTLSeconds != 12*60*60 {
		t.Fatalf("default admin session ttl = %d, want 43200", cfg.AdminSessionTTLSeconds)
	}

	t.Setenv("ADMIN_USERNAME", "console-admin")
	t.Setenv("ADMIN_PASSWORD", "console-secret")
	t.Setenv("ADMIN_SESSION_SECRET", "session-secret")
	t.Setenv("ADMIN_SESSION_TTL_SECONDS", "60")
	cfg = Load()
	if cfg.AdminUsername != "console-admin" || cfg.AdminPassword != "console-secret" || cfg.AdminSessionSecret != "session-secret" {
		t.Fatalf("explicit admin config was not applied: %#v", cfg)
	}
	if cfg.AdminSessionTTLSeconds != 60 {
		t.Fatalf("admin session ttl = %d, want 60", cfg.AdminSessionTTLSeconds)
	}
}

func TestLoadRuntimeBuildMetadata(t *testing.T) {
	t.Setenv("AGENT_IMAGEFLOW_VERSION", "0.1.0")
	t.Setenv("AGENT_IMAGEFLOW_COMMIT", "abc1234")
	t.Setenv("AGENT_IMAGEFLOW_BUILD_TIME", "2026-06-25T03:00:00Z")
	t.Setenv("AGENT_IMAGEFLOW_IMAGE_TAG", "sha-abc1234")

	cfg := Load()
	if cfg.BuildVersion != "0.1.0" || cfg.BuildCommit != "abc1234" || cfg.BuildTime != "2026-06-25T03:00:00Z" || cfg.ImageTag != "sha-abc1234" {
		t.Fatalf("runtime build metadata was not loaded: %#v", cfg)
	}
}
