package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/billionsheep/agent-imageflow/internal/app"
	"github.com/billionsheep/agent-imageflow/internal/config"
	"github.com/billionsheep/agent-imageflow/internal/db"
	"github.com/billionsheep/agent-imageflow/internal/mcp"
	"github.com/billionsheep/agent-imageflow/internal/queue"
	"github.com/billionsheep/agent-imageflow/internal/storage"
	"github.com/billionsheep/agent-imageflow/internal/store"
)

func main() {
	log.SetOutput(os.Stderr)
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

	service := app.NewService(cfg, store.NewPostgresStore(conn), q, storage.NewLocalStorage(cfg.StorageRoot, cfg.ThumbnailMaxWidth, cfg.ThumbnailMaxHeight))
	server := mcp.New(service, mcp.Defaults{
		WorkspaceID: cfg.DefaultWorkspace,
		ProjectID:   cfg.DefaultProject,
		CampaignID:  cfg.DefaultCampaign,
		Version:     cfg.BuildVersion,
	})
	if err := server.Serve(ctx, os.Stdin, os.Stdout); err != nil && ctx.Err() == nil {
		log.Fatalf("serve mcp: %v", err)
	}
}
