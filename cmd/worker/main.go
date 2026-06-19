package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/billionsheep/agent-imageflow/internal/app"
	"github.com/billionsheep/agent-imageflow/internal/config"
	"github.com/billionsheep/agent-imageflow/internal/db"
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

	service := app.NewService(cfg, store.NewPostgresStore(conn), q, storage.NewLocalStorage(cfg.StorageRoot, cfg.ThumbnailMaxWidth, cfg.ThumbnailMaxHeight))
	log.Printf("Agent ImageFlow worker started with concurrency=%d", cfg.WorkerConcurrency)

	var wg sync.WaitGroup
	for i := 0; i < cfg.WorkerConcurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, workerID, service)
		}(i + 1)
	}
	wg.Wait()
}

func runWorker(ctx context.Context, workerID int, service *app.Service) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		promoted, err := service.Queue().PromoteScheduled(ctx, 32)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("worker %d promote scheduled error: %v", workerID, err)
			time.Sleep(time.Second)
			continue
		}
		if promoted > 0 {
			log.Printf("worker %d promoted %d scheduled task(s)", workerID, promoted)
		}

		taskID, err := service.Queue().Dequeue(ctx, time.Second)
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("worker %d dequeue error: %v", workerID, err)
			time.Sleep(time.Second)
			continue
		}
		if taskID == "" {
			continue
		}
		locked, err := service.Queue().LockTask(ctx, taskID, 10*time.Minute)
		if err != nil {
			log.Printf("worker %d lock task %s failed: %v", workerID, taskID, err)
			continue
		}
		if !locked {
			log.Printf("worker %d skipped locked task %s", workerID, taskID)
			continue
		}
		log.Printf("worker %d processing task %s", workerID, taskID)
		if err := service.ProcessTask(ctx, taskID); err != nil {
			log.Printf("worker %d task %s failed: %v", workerID, taskID, err)
		}
		service.Queue().UnlockTask(context.Background(), taskID)
	}
}
