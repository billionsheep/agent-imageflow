# Story: Slice 009 - Repair Reconcile Smoke

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让本地部署者在 Redis 入队失败、Worker 中断或资产文件与数据库不一致时，可以用一个最小 CLI 发现问题，并对可恢复任务执行重入队修复，降低 MVP 自托管运行时的手动查库成本。

## Source Context

- Product spec: Agent ImageFlow 必须提供稳定任务、资产、文件和交付信息，而不是一次性 provider wrapper。
- Architecture: 第一版应预留 `vag repair scan`、`vag repair requeue <task_id>`、`vag repair verify-asset <asset_id>`。
- Project plan slice: Phase 6 MVP hardening 推荐补 repair/reconcile、worker retry/backoff、真实缩略图和部署硬化。
- Related decisions: PostgreSQL 是事实源，Redis 是队列；数据库写入成功但入队失败时任务进入 `enqueue_failed`，后续通过 repair/requeue 补偿。

## User Flow

1. 本地部署者怀疑有任务卡住或资产不可读。
2. 运行 `vag repair scan`，看到 `enqueue_failed`、长时间 `queued/running`、资产文件缺失或孤儿文件报告。
3. 对可恢复任务运行 `vag repair requeue <task_id>`。
4. Worker 消费重入队任务，任务重新进入正常生成闭环。
5. 对单个资产运行 `vag repair verify-asset <asset_id>`，确认 original/thumbnail/metadata 三类文件是否存在。

## In Scope

- 新增本地 CLI：`vag repair scan`、`vag repair requeue <task_id>`、`vag repair verify-asset <asset_id>`。
- `scan` 输出结构化 JSON，默认只读。
- `requeue` 支持 `enqueue_failed`、长时间 `queued`、长时间 `running` 任务重新标记为 `queued` 并写入 Redis queue。
- `verify-asset` 检查当前 ready 版本的 original、thumbnail、metadata 文件是否存在且非空。
- `scan` 报告资产当前版本异常、ready 文件缺失和正式存储目录下没有数据库记录的孤儿文件。

## Out of Scope

- 不新增数据库表或 outbox。
- 不实现自动后台 repair job。
- 不实现完整 worker retry/backoff 策略。
- 不自动删除孤儿文件。
- 不自动修复 asset/version 数据库记录。

## Acceptance Criteria

- Given 数据库中存在 `enqueue_failed` 任务，when 运行 `vag repair scan`，then 输出 JSON issue 包含该 `task_id` 和 `repair_hint=requeue_task`。
- Given 该任务可恢复，when 运行 `vag repair requeue <task_id>`，then 任务状态变为 `queued` 并写入 Redis queue。
- Given 任务被 requeue 且 Worker 运行，when 查询任务，then 任务最终能进入 `completed` 或明确失败状态。
- Given asset 当前版本文件都存在，when 运行 `vag repair verify-asset <asset_id>`，then 输出 `ok=true`。
- Given asset 当前版本缺文件，when 运行 `vag repair scan` 或 `verify-asset`，then 输出对应 `missing_file` issue。

## Technical Approach

- 在 `app.Service` 增加 repair/reconcile 方法，复用现有 store、queue 和 storage。
- 在 `store.PostgresStore` 增加只读查询：可修复任务、当前资产版本、异常 current_version 和已知文件路径。
- `vag repair` 作为本地维护命令直接读取 `DATABASE_URL`、`REDIS_URL`、`STORAGE_ROOT`，不经 HTTP API 暴露管理能力。
- 文件检查使用本地 `os.Stat`；孤儿文件扫描只扫描 `originals`、`thumbnails`、`metadata` 目录，跳过 `tmp`。

## Data / Interface Impact

- 新增 CLI 命令，不新增 REST/MCP 接口。
- 不新增数据库表、字段或环境变量。
- `generation_task` 可由 repair 命令从 `enqueue_failed/running/queued` 重置为 `queued` 并清空错误字段。

## Files or Subsystems Likely to Change

- `cmd/vag/main.go`
- `internal/app/repair.go`
- `internal/store/postgres.go`
- `docs/project/*`

## Verification Plan

```bash
go test ./...
docker compose config
docker compose build
docker compose up -d postgres redis api
# smoke: stop worker, create task, simulate enqueue_failed + queue loss, repair scan, repair requeue, start worker, poll completed.
```

## Assumptions and Risks

- 第一版 repair 是人工显式操作，不做自动周期任务。
- `requeue` 可能造成 Redis 中重复 task id；Worker 现有 task lock 和终态 no-op 会避免重复处理成重复资产。
- `running` 任务是否真的卡住只能通过 `updated_at` 年龄判断，默认阈值应保守。

## Implementation Log

### 2026-06-18

- Changes: 新增本地 `vag repair scan`、`vag repair requeue <task_id>`、`vag repair verify-asset <asset_id>`；app 层新增 repair/reconcile service；store 层新增可恢复任务、当前资产版本、异常 current_version 和已知文件路径查询；scan 可报告 `enqueue_failed`、stale queued/running、invalid current version、missing/empty file 和 orphan file。
- Verification: `go test ./...`、`docker compose config`、`docker compose build` 通过；Docker smoke 停止 worker 后模拟 `enqueue_failed` 和 Redis queue 丢失，`repair scan` 报告 `repair_hint=requeue_task`，`repair requeue` 将任务重入队，启动 worker 后任务完成；`repair verify-asset` 对正常资产返回 `ok=true`，临时移走 original 文件后返回 `missing_file` issue，验证后已恢复文件。
- Remaining gaps: Worker 自动 retry/backoff、真实缩略图 resize、项目级 API key 和生产部署硬化仍待后续 slice。
