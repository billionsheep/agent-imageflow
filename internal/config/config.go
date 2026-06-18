package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL       string
	RedisURL          string
	StorageRoot       string
	PublicBaseURL     string
	HTTPAddr          string
	DefaultWorkspace  string
	DefaultProject    string
	DefaultCampaign   string
	DefaultProvider   string
	WorkerConcurrency int
}

func Load() Config {
	return Config{
		DatabaseURL:       env("DATABASE_URL", "postgres://agent:agent@localhost:5432/agent_imageflow?sslmode=disable"),
		RedisURL:          env("REDIS_URL", "redis://localhost:6379/0"),
		StorageRoot:       env("STORAGE_ROOT", "./storage"),
		PublicBaseURL:     strings.TrimRight(env("PUBLIC_BASE_URL", "http://localhost:8081"), "/"),
		HTTPAddr:          env("HTTP_ADDR", ":8081"),
		DefaultWorkspace:  env("DEFAULT_WORKSPACE_ID", "ws_default"),
		DefaultProject:    env("DEFAULT_PROJECT_ID", "prj_xhs_anime"),
		DefaultCampaign:   env("DEFAULT_CAMPAIGN_ID", "cmp_7day_cover"),
		DefaultProvider:   env("DEFAULT_PROVIDER", "mock"),
		WorkerConcurrency: envInt("WORKER_CONCURRENCY", 1),
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
