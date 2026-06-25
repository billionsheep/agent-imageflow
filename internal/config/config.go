package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL                    string
	RedisURL                       string
	StorageRoot                    string
	PublicBaseURL                  string
	HTTPAddr                       string
	RateLimitWindowSeconds         int
	RateLimitInstanceMaxReq        int
	RateLimitProjectMaxReq         int
	DefaultWorkspace               string
	DefaultProject                 string
	DefaultCampaign                string
	DefaultProvider                string
	BasicAuthUsername              string
	BasicAuthPassword              string
	AdminUsername                  string
	AdminPassword                  string
	AdminSessionSecret             string
	AdminSessionTTLSeconds         int
	FalBaseURL                     string
	FalRestBaseURL                 string
	FalAPIKey                      string
	FalModel                       string
	FalMaxConcurrency              int
	FalPollIntervalMs              int
	OpenAICompatibleBaseURL        string
	OpenAICompatibleAPIKey         string
	OpenAICompatibleModel          string
	OpenAICompatibleMaxConcurrency int
	OpenAICompatibleConnectTimeout int
	OpenAICompatibleHeaderTimeout  int
	OpenAICompatibleTotalTimeout   int
	BestOfHTTPScorerURL            string
	BestOfHTTPScorerAPIKey         string
	BestOfHTTPScorerTimeout        int
	ProviderTimeoutSeconds         int
	WorkerConcurrency              int
	WorkerMaxRetries               int
	WorkerRetryBaseDelaySec        int
	ThumbnailMaxWidth              int
	ThumbnailMaxHeight             int
	BuildVersion                   string
	BuildCommit                    string
	BuildTime                      string
	ImageTag                       string
}

func Load() Config {
	providerTimeout := envInt("PROVIDER_TIMEOUT_SECONDS", 300)
	return Config{
		DatabaseURL:                    env("DATABASE_URL", "postgres://agent:agent@localhost:5432/agent_imageflow?sslmode=disable"),
		RedisURL:                       env("REDIS_URL", "redis://localhost:6379/0"),
		StorageRoot:                    env("STORAGE_ROOT", "./storage"),
		PublicBaseURL:                  strings.TrimRight(env("PUBLIC_BASE_URL", "http://localhost:8081"), "/"),
		HTTPAddr:                       env("HTTP_ADDR", ":8081"),
		RateLimitWindowSeconds:         envInt("RATE_LIMIT_WINDOW_SECONDS", 60),
		RateLimitInstanceMaxReq:        envNonNegativeInt("RATE_LIMIT_INSTANCE_MAX_REQUESTS", 0),
		RateLimitProjectMaxReq:         envNonNegativeInt("RATE_LIMIT_PROJECT_MAX_REQUESTS", 0),
		DefaultWorkspace:               env("DEFAULT_WORKSPACE_ID", "ws_default"),
		DefaultProject:                 env("DEFAULT_PROJECT_ID", "prj_xhs_anime"),
		DefaultCampaign:                env("DEFAULT_CAMPAIGN_ID", "cmp_7day_cover"),
		DefaultProvider:                env("DEFAULT_PROVIDER", "mock"),
		BasicAuthUsername:              env("BASIC_AUTH_USERNAME", ""),
		BasicAuthPassword:              env("BASIC_AUTH_PASSWORD", ""),
		AdminUsername:                  env("ADMIN_USERNAME", env("BASIC_AUTH_USERNAME", "")),
		AdminPassword:                  env("ADMIN_PASSWORD", env("BASIC_AUTH_PASSWORD", "")),
		AdminSessionSecret:             env("ADMIN_SESSION_SECRET", ""),
		AdminSessionTTLSeconds:         envInt("ADMIN_SESSION_TTL_SECONDS", 12*60*60),
		FalBaseURL:                     strings.TrimRight(env("FAL_BASE_URL", "https://queue.fal.run"), "/"),
		FalRestBaseURL:                 strings.TrimRight(env("FAL_REST_BASE_URL", "https://rest.fal.ai"), "/"),
		FalAPIKey:                      env("FAL_API_KEY", ""),
		FalModel:                       env("FAL_MODEL", "openai/gpt-image-2"),
		FalMaxConcurrency:              envNonNegativeInt("FAL_MAX_CONCURRENCY", 3),
		FalPollIntervalMs:              envInt("FAL_POLL_INTERVAL_MS", 1000),
		OpenAICompatibleBaseURL:        strings.TrimRight(env("OPENAI_COMPATIBLE_BASE_URL", ""), "/"),
		OpenAICompatibleAPIKey:         env("OPENAI_COMPATIBLE_API_KEY", ""),
		OpenAICompatibleModel:          env("OPENAI_COMPATIBLE_MODEL", ""),
		OpenAICompatibleMaxConcurrency: envNonNegativeInt("OPENAI_COMPATIBLE_MAX_CONCURRENCY", 3),
		OpenAICompatibleConnectTimeout: envInt("OPENAI_COMPATIBLE_CONNECT_TIMEOUT_SECONDS", 30),
		OpenAICompatibleHeaderTimeout:  envInt("OPENAI_COMPATIBLE_RESPONSE_HEADER_TIMEOUT_SECONDS", providerTimeout),
		OpenAICompatibleTotalTimeout:   envInt("OPENAI_COMPATIBLE_TOTAL_TIMEOUT_SECONDS", providerTimeout),
		BestOfHTTPScorerURL:            env("BEST_OF_HTTP_SCORER_URL", ""),
		BestOfHTTPScorerAPIKey:         env("BEST_OF_HTTP_SCORER_API_KEY", ""),
		BestOfHTTPScorerTimeout:        envInt("BEST_OF_HTTP_SCORER_TIMEOUT_SECONDS", 30),
		ProviderTimeoutSeconds:         providerTimeout,
		WorkerConcurrency:              envInt("WORKER_CONCURRENCY", 6),
		WorkerMaxRetries:               envNonNegativeInt("WORKER_MAX_RETRIES", 3),
		WorkerRetryBaseDelaySec:        envInt("WORKER_RETRY_BASE_DELAY_SECONDS", 15),
		ThumbnailMaxWidth:              envInt("THUMBNAIL_MAX_WIDTH", 720),
		ThumbnailMaxHeight:             envInt("THUMBNAIL_MAX_HEIGHT", 720),
		BuildVersion:                   env("AGENT_IMAGEFLOW_VERSION", ""),
		BuildCommit:                    env("AGENT_IMAGEFLOW_COMMIT", ""),
		BuildTime:                      env("AGENT_IMAGEFLOW_BUILD_TIME", ""),
		ImageTag:                       env("AGENT_IMAGEFLOW_IMAGE_TAG", ""),
	}
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func envNonNegativeInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}
