package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/app"
	"github.com/billionsheep/agent-imageflow/internal/config"
	"github.com/billionsheep/agent-imageflow/internal/db"
	"github.com/billionsheep/agent-imageflow/internal/httpapi"
	"github.com/billionsheep/agent-imageflow/internal/queue"
	"github.com/billionsheep/agent-imageflow/internal/storage"
	"github.com/billionsheep/agent-imageflow/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	conn, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer conn.Close()
	if err := db.Migrate(ctx, conn); err != nil {
		log.Fatalf("migrate database: %v", err)
	}
	if err := db.SeedDefaults(ctx, conn, cfg.DefaultWorkspace, cfg.DefaultProject, cfg.DefaultCampaign); err != nil {
		log.Fatalf("seed defaults: %v", err)
	}

	q, err := queue.NewRedisQueue(cfg.RedisURL)
	if err != nil {
		log.Fatalf("create redis queue: %v", err)
	}
	defer q.Close()
	if err := q.Ping(ctx); err != nil {
		log.Fatalf("ping redis: %v", err)
	}

	var rateLimiter httpapi.RateLimiter
	if cfg.RateLimitWindowSeconds > 0 && (cfg.RateLimitInstanceMaxReq > 0 || cfg.RateLimitProjectMaxReq > 0) {
		rateLimiter, err = httpapi.NewRedisRateLimiter(cfg.RedisURL)
		if err != nil {
			log.Fatalf("create redis rate limiter: %v", err)
		}
		defer rateLimiter.Close()
	}

	localStorage := storage.NewLocalStorage(cfg.StorageRoot, cfg.ThumbnailMaxWidth, cfg.ThumbnailMaxHeight)
	service := app.NewService(cfg, store.NewPostgresStore(conn), q, localStorage)
	server := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: httpapi.New(service, httpapi.Options{
			AgentSetupToken:    cfg.AgentSetupToken,
			BasicAuthUsername:  cfg.BasicAuthUsername,
			BasicAuthPassword:  cfg.BasicAuthPassword,
			AdminUsername:      cfg.AdminUsername,
			AdminPassword:      cfg.AdminPassword,
			AdminSessionSecret: cfg.AdminSessionSecret,
			AdminSessionTTL:    time.Duration(cfg.AdminSessionTTLSeconds) * time.Second,
			Runtime: httpapi.RuntimeStatusOptions{
				PublicBaseURL:                  cfg.PublicBaseURL,
				DefaultProvider:                cfg.DefaultProvider,
				BuildVersion:                   cfg.BuildVersion,
				BuildCommit:                    cfg.BuildCommit,
				BuildTime:                      cfg.BuildTime,
				ImageTag:                       cfg.ImageTag,
				OpenAICompatibleModel:          cfg.OpenAICompatibleModel,
				OpenAICompatibleConfigured:     cfg.OpenAICompatibleAPIKey != "",
				OpenAICompatibleMaxConcurrency: cfg.OpenAICompatibleMaxConcurrency,
				FalModel:                       cfg.FalModel,
				FalConfigured:                  cfg.FalAPIKey != "",
				FalMaxConcurrency:              cfg.FalMaxConcurrency,
				ProviderTimeoutSeconds:         cfg.ProviderTimeoutSeconds,
				WorkerConcurrency:              cfg.WorkerConcurrency,
				RateLimitWindowSeconds:         cfg.RateLimitWindowSeconds,
				RateLimitInstanceMaxRequests:   cfg.RateLimitInstanceMaxReq,
				RateLimitProjectMaxRequests:    cfg.RateLimitProjectMaxReq,
			},
			AuditSink:                    localStorage,
			RateLimiter:                  rateLimiter,
			RateLimitWindow:              time.Duration(cfg.RateLimitWindowSeconds) * time.Second,
			RateLimitInstanceMaxRequests: cfg.RateLimitInstanceMaxReq,
			RateLimitProjectMaxRequests:  cfg.RateLimitProjectMaxReq,
		}),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5_000_000_000)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("Agent ImageFlow API listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}
