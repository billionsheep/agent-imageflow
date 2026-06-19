# Story: Slice 010 - Worker Retry Backoff

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让 Worker 在遇到 provider 限流、超时或短暂网络错误时，能够自动按指数退避重试，而不是立即把任务打成 failed，减少小团队自托管场景下对 `repair requeue` 的依赖。

## Source Context

- Product spec: Agent ImageFlow 需要提供稳定任务、资产、交付和自动化闭环，而不是一次性 provider wrapper。
- Architecture: Worker retry 应支持瞬时 provider 失败、`task_attempt` 记录、`retry_after`、指数退避和重复消费保护。
- Project plan slice: Phase 6 MVP hardening 还缺 Worker retry / backoff。
- Related decisions: PostgreSQL 是事实源，Redis 是队列；repair/reconcile 已补齐，本片继续把常见临时失败从手工补救前移到自动恢复。

## User Flow

1. 调用方创建图片任务。
2. Worker 调用 provider 时遇到限流、超时或短暂错误。
3. 服务端将本次 `task_attempt` 标记失败并写入 `retry_after`。
4. 任务重新进入延迟队列，稍后自动重试。
5. 若后续尝试成功，任务正常进入 completed / partially_completed；若超过最大重试次数，任务进入 failed。

## In Scope

- 针对 provider 瞬时失败增加自动 retry/backoff。
- 使用 Redis 延迟重试队列，不阻塞 worker goroutine 睡眠等待。
- 将 `retry_after` 写入 `task_attempt`。
- 给 mock provider 增加可控的 `transient_once` 失败模式，供测试与 smoke 使用。
- 在最大重试次数内，任务状态从失败回退到 `queued` 并自动重试。

## Out of Scope

- 不对资产处理失败做自动重试。
- 不做完整后台 repair job。
- 不引入按 provider 单独配置的复杂策略。
- 不做跨进程精确一次的延迟队列保证。
- 不做真实 provider edit/mask 自动重试策略。

## Acceptance Criteria

- Given provider 返回瞬时失败，when 当前 attempt 未超过最大重试次数，then `task_attempt.retry_after` 被写入，任务状态回到 `queued`，并在延迟后自动再次执行。
- Given 同一任务后续尝试成功，when 查询任务，then 任务最终进入 `completed` 或 `partially_completed`，而不是停留在 failed。
- Given 瞬时失败持续超过最大重试次数，when 最后一轮尝试仍失败，then 任务进入 `failed`。
- Given mock task 配置 `mock_failure_mode=transient_once`，when Worker 处理任务，then 第一轮失败后自动重试，第二轮成功。

## Technical Approach

- 在 `queue.RedisQueue` 增加 scheduled zset，用于 `EnqueueAfter` 与 `PromoteScheduled`。
- 在 `config.Config` 增加 Worker 重试上限和基础退避配置。
- 在 `app.Service.ProcessTask` 中保留 attemptNo，识别 retryable provider error，并调度延迟重试。
- 为 `task_attempt` 的 `retry_after` 增加 store 写入支持。
- mock provider 从 `generation_config` 读取测试用 failure mode，并在 `transient_once` 时按 task id 仅失败一次。

## Data / Interface Impact

- 不新增 REST/MCP 接口。
- 不新增数据库表；复用已有 `task_attempt.retry_after` 字段。
- 新增环境变量：`WORKER_MAX_RETRIES`、`WORKER_RETRY_BASE_DELAY_SECONDS`。

## Files or Subsystems Likely to Change

- `internal/queue/redis.go`
- `internal/config/config.go`
- `internal/app/service.go`
- `internal/app/retry.go`
- `internal/store/postgres.go`
- `internal/provider/mock.go`
- `cmd/worker/main.go`
- `docs/project/*`

## Verification Plan

```bash
go test ./...
docker compose config
docker compose build
docker compose up -d postgres redis api worker
# smoke: create mock task with generation_config.mock_failure_mode=transient_once, poll task until completed, verify at least 2 attempts and first attempt.retry_after is set.
```

## Assumptions and Risks

- 第一版延迟重试队列允许极小概率的重复 promote/requeue；现有 task lock 和终态 no-op 会吸收这类重复消费。
- 最大重试次数和基础退避先使用全局默认值，不做 provider 级差异化。
- `transient_once` 是测试和 smoke 用能力，不是产品暴露的正式业务参数。

## Implementation Log

### 2026-06-18

- Changes:
  - 在 `queue.RedisQueue` 增加 Redis delayed queue：`EnqueueAfter` 与 `PromoteScheduled`。
  - 在 `Service.ProcessTask` 增加 provider 瞬时失败的 retry/backoff 调度逻辑。
  - `task_attempt.retry_after` 已由 store 写入。
  - mock provider 新增 `generation_config.mock_failure_mode=transient_once`，用于 smoke 和测试。
  - Worker 启动循环增加 scheduled queue promote。
- Verification:
  - `docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine /usr/local/go/bin/go test ./...`
  - `docker compose config`
  - Docker smoke：`mock_failure_mode=transient_once` 任务第 1 次 attempt `temporary_unavailable`，第 2 次 attempt 自动成功；`task_attempt.retry_after` 已写入。
- Remaining gaps: 真实缩略图 resize、项目级 API key、Web project/campaign 管理体验和真实 provider edit/mask 边界仍待后续 slice。
