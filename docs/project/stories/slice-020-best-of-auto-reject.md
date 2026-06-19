# Story: Slice 020 - Best-of Auto Reject

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让 Agent ImageFlow 在自动选优已经选出推荐图后，可以按调用方或项目级质量配置的显式开关，把未入选候选自动标记为 rejected，从而减少小团队手工清理候选状态的成本；同时保留人工后续改选的能力，不把 auto reject 变成不可逆闸门。

## Source Context

- Product spec: 第一版采用 `generated -> selected/rejected` 轻量状态模型，`rejected` 用于排除候选，但不应把交付流重新做回强审核闸门。
- Project plan slice: `Best-of auto reject` 是当前第一条 pending slice。
- Tech spec: 多入口共用同一 application core；`best_of_config` 已进入任务输入和项目级 quality profile。
- Related decisions: `降级人工审核为轻量选优/状态标记`、`Best-of 第二版采用可插拔 scorer + HTTP judge adapter`、`Best-of 第一版采用本地启发式`。

## User Flow

1. 调用方通过 REST / MCP / CLI / Web 托管模式创建 `selection_mode=auto` 或 `best_of` 的多候选任务。
2. 调用方可以在任务输入或项目级 quality profile 中设置 `best_of_config.auto_reject_non_selected=true`。
3. Worker 生成并登记多个候选资产，按当前 scorer 选出一张 selected。
4. 如果 `auto_reject_non_selected=true`，服务端会把其他未入选候选自动标记为 rejected；如果未开启，则它们继续保持 generated。
5. 若用户后续认为某张 auto rejected 候选更好，仍可通过现有 select/reject 入口手动改选。

## In Scope

- 在 `best_of_config` 中新增 `auto_reject_non_selected` 开关。
- 让任务输入和项目级 quality profile 都能保存/复用这个开关。
- 自动选优后，按开关决定是否将未入选候选自动标记为 rejected。
- 保留 `selected/rejected` 现有输出语义，不新增数据库字段。
- 允许人工把已 rejected 的候选重新 select，避免 auto reject 变成不可逆状态。
- 补 focused tests 和本地 smoke。

## Out of Scope

- 不新增复杂的“二次审核”“待复核”状态。
- 不重写 Web UI 候选图排序或批量操作面板。
- 不做数据库迁移。
- 不改动 `local_metadata_v1` / `http_judge_v1` 的评分逻辑本身。
- 不进入限流、审计、多 key 等生产 hardening。

## Acceptance Criteria

- Given `selection_mode=auto` 或 `best_of` 的任务未设置 `best_of_config.auto_reject_non_selected=true`，when 自动选优完成，then 推荐图为 selected，其他候选继续保持 generated。
- Given 任务输入或项目级 quality profile 设置了 `best_of_config.auto_reject_non_selected=true`，when 自动选优完成，then 恰好一张候选为 selected，其余同批候选为 rejected。
- Given `best_of_config` 通过项目级 quality profile 保存了 `auto_reject_non_selected=true`，when REST/MCP/Web 托管任务复用质量配置，then 该开关不会丢失。
- Given 某张候选因 auto reject 被标记为 rejected，when 用户后续调用现有 select 入口，then 该候选仍可重新变为 selected。
- Given 任务使用 `http_judge_v1` 或 fallback 到 `local_metadata_v1`，when auto reject 开启，then 自动 reject 仍沿用最终 applied strategy 的选优结果，不改变当前 scorer/fallback 行为。

## Technical Approach

- 在 `domain.BestOfConfig` 中新增 `AutoRejectNonSelected` 字段，并同步到 REST/MCP/Web types。
- 复用现有 `normalizeBestOfConfig`、quality profile 继承和 quality snapshot 流程，无需新增 schema。
- 在 best-of 自动选优里基于 `best_of_config` 生成一份“selected + rejected siblings”决策。
- 为避免 auto selection 和 auto reject 产生部分成功状态，服务端存储层增加一条批量应用 best-of 决策的事务方法。
- 现有手动 `approve/select` 状态迁移补充 `rejected -> approved` 兼容路径，保证人工 override 不被 auto reject 封死。

## Data / Interface Impact

- `CreateTaskRequest.best_of_config` 新增 `auto_reject_non_selected`。
- `QualityProfile.best_of_config` 新增 `auto_reject_non_selected`。
- `structured_input_json` 与 quality snapshot 会保留 `auto_reject_non_selected`。
- 不新增数据库表和字段。

## Files or Subsystems Likely to Change

- `internal/domain/types.go`
- `internal/app/quality.go`
- `internal/app/quality_test.go`
- `internal/app/bestof.go`
- `internal/app/bestof_test.go`
- `internal/app/service.go`
- `internal/store/postgres.go`
- `internal/mcp/server.go`
- `web/src/lib/agentImageflowApi.ts`
- `examples/tasks/`
- `docs/project/*`

## Verification Plan

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./internal/app ./internal/mcp'
docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'
docker compose config
docker compose up -d postgres redis api worker
# manual smoke:
# 1. 创建 selection_mode=auto + best_of_config.auto_reject_non_selected=true 的 mock 任务
# 2. 轮询任务完成，检查 1 张 selected + 其余 rejected
# 3. 对一张 auto rejected 候选调用 approve/select，确认可重新 selected
```

## Assumptions and Risks

- `auto_reject_non_selected` 默认关闭，避免无意改变当前自动选优行为。
- auto reject 应是可逆的业务标记，而不是永久封禁某张候选；因此人工重新 select 必须保留。
- 若当前 store 层不做事务式批量更新，可能出现“selected 已写入、部分 reject 未写入”的中间态；本片优先避免这种不一致。

## Implementation Log

### 2026-06-18

- Changes:
  - 在 `internal/domain/types.go` 中为 `BestOfConfig` 新增 `auto_reject_non_selected`，并同步到 REST/MCP/Web 类型。
  - 在 `internal/app/quality.go` 中补齐 `auto_reject_non_selected` 的归一化、quality profile 复用与 snapshot 写入；允许仅设置 auto reject 而不显式指定 strategy，此时沿用默认 `local_metadata_v1`。
  - 在 `internal/app/bestof.go` 中让自动选优决策同时携带 auto reject 开关；开启时改走批量事务式 `selected + rejected siblings` 写入，而不是只 selected 一张推荐图。
  - 在 `internal/store/postgres.go` 中新增 `ApplyBestOfSelection(...)`，并把 review 状态迁移抽到事务 helper；手动 review 现在允许 `rejected -> approved`，保留人工改选路径。
  - 在 `examples/tasks/` 中新增 `sample-best-of-auto-reject-task.json`，用于本地 smoke。
- Verification:
  - `docker build --target build -t agent-imageflow-build .`
  - `docker run --rm agent-imageflow-build sh -lc 'cd /src && /usr/local/go/bin/go test ./internal/app ./internal/mcp ./internal/store'`
  - `docker run --rm agent-imageflow-build sh -lc 'cd /src && /usr/local/go/bin/go test ./...'`
  - `npm --prefix web run build`
  - `docker compose config`
  - `docker compose build api worker`
  - `docker compose up -d --force-recreate api worker`
  - `curl -sf http://localhost:8081/healthz`
  - 本地 smoke：创建 `task_79ee5fdfe639cd532805`，验证自动选优后 1 张 selected + 2 张 rejected；随后对 auto rejected 的 `asset_5d207d1a89b3ba6d6793` 调用 `/approve`，确认可重新 selected。
- Remaining gaps:
  - 当前只完成了 auto reject 这条选优状态流；更强的生产 hardening（限流、审计、多 key 策略）仍待后续 slice。
