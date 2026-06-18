package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

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

	service := app.NewService(cfg, store.NewPostgresStore(conn), q, storage.NewLocalStorage(cfg.StorageRoot))
	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: httpapi.New(service),
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
