# Story: Slice 007 - Best-of Auto Selection

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让多候选图片任务在生成完成后可以自动选出一张推荐图，减少小团队/单体平台逐张人工审核成本，并为后续更强的视觉/LLM 选优策略打基础。

## Source Context

- Product spec: 第一版不强制人工审核，质量通过 prompt、style preset、参考图、生成参数和自动选优策略提升。
- Project plan slice: `Best-of selection` 是 Quality foundation 后的推荐下一片。
- Tech spec: 多入口必须进入同一 application core；Worker 负责生成和资产登记；资产状态用于表达 generated/selected/rejected。
- Related decisions: `降级人工审核为轻量选优/状态标记`、`Quality profile 先复用 project metadata`。

## User Flow

1. 用户、Web、MCP 或 REST 创建一个 `requested_count > 1` 的图片任务，并设置 `selection_mode` 为 `auto` 或 `best_of`。
2. Worker 生成并登记多个候选资产。
3. 服务端根据可解释的本地规则给候选资产打分，自动将最高分候选标记为 selected。
4. 调用方查询任务或资产列表时，可以直接看到推荐资产状态为 selected，其他候选保持 generated。

## In Scope

- 在统一 `CreateTaskRequest` 中正式接入 `selection_mode`。
- 将 `selection_mode` 写入 `structured_input_json` 并在查询任务时返回。
- Worker 生成资产后执行第一版本地 best-of 策略。
- 自动选择使用现有 `review_event` 和兼容 `approve` 状态迁移，不新增表结构。
- MCP 和 Web 托管创建任务能传递 `selection_mode`。

## Out of Scope

- 不接外部视觉模型或 LLM 打分。
- 不自动 reject 未选中候选。
- 不做 Web 复杂候选排序/打分 UI。
- 不做数据库 schema 迁移。
- 不改变手动 select/reject 的兼容接口。

## Acceptance Criteria

- Given REST/MCP/Web 创建任务时传 `selection_mode=auto` 或 `best_of`，when Worker 完成多个候选资产登记，then 服务端自动把一张候选标记为 selected。
- Given 创建任务未传 `selection_mode` 或传 `manual_optional`，when Worker 完成多个候选资产登记，then 候选资产保持 generated，仍可人工 select/reject。
- Given 自动选优发生，when 查询任务，then 返回的 asset 列表中恰好有一张 selected 候选。
- Given MCP `create_image_task` 传 `selection_mode`，when tool call 进入 application core，then 字段不会丢失。
- Given Web 托管模式提交多候选任务，when 服务端任务完成，then Web 轮询得到的候选状态能显示 selected。

## Technical Approach

- 增加 `domain.SelectionManualOptional`、`domain.SelectionAuto`、`domain.SelectionBestOf` 常量。
- `CreateTaskRequest` 和 `Task` 增加 `SelectionMode` 字段；数据不新开列，存入 `structured_input_json`。
- `store.scanTaskWithHash` 从 `structured_input_json` 反填 `Task.SelectionMode`，兼容旧任务默认 `manual_optional`。
- `app.ProcessTask` 收集本次登记成功的候选资产，在任务进入 completed/partially_completed 前后触发 `autoSelectBestAsset`。
- 第一版打分只使用本地、可解释字段：版本 ready、图片面积、目标宽高比例接近度、hash 稳定排序。打分详情写入 `review_event.note`。
- Web managed mode 第一版默认传 `selection_mode=auto`，让多候选任务完成后能自动推荐一张。

## Data / Interface Impact

- REST/MCP/Web task input 新增或正式启用 `selection_mode`。
- `GET /api/tasks/{id}` 返回 `selection_mode`。
- 不新增数据库表和字段。
- 自动选优事件复用 `review_event`，`reviewer=auto-best-of`，`action=approve`。

## Files or Subsystems Likely to Change

- `internal/domain/types.go`
- `internal/app/service.go`
- `internal/app/bestof.go`
- `internal/store/postgres.go`
- `internal/mcp/server.go`
- `web/src/lib/agentImageflowApi.ts`
- `web/src/store.ts`
- `docs/project/*`

## Verification Plan

```bash
docker run --rm -v "$PWD":/src -w /src golang:1.25.3-alpine sh -lc '/usr/local/go/bin/gofmt -w cmd internal && /usr/local/go/bin/go test ./...'
npm --prefix web test -- --run
npm --prefix web run build
docker compose config
docker compose build
docker compose up -d postgres redis api worker
# REST smoke: create requested_count=3 selection_mode=auto, poll task, assert exactly one selected.
# Regression smoke: create requested_count=2 selection_mode=manual_optional, assert no selected.
```

## Assumptions and Risks

- 第一版 best-of 是确定性的本地启发式，不代表真实审美质量；它的价值是先打通自动推荐状态流。
- 自动 selected 不阻断其他 generated 资产交付，也不自动 reject 其他候选。
- 如果后续引入视觉模型打分，应替换或扩展当前 scoring strategy，而不是改变资产状态模型。

## Implementation Log

### 2026-06-18

- Changes: 新增 `selection_mode` 统一输入字段；Worker 在 `auto` / `best_of` 模式下使用 `local_metadata_v1` 本地启发式策略自动 selected 一张候选；Web managed mode 默认传 `selection_mode=auto`；MCP 字段转发修正；自动 selected 资产允许用户 reject 覆盖。
- Verification: `go test ./...`、`npm --prefix web test -- --run`、`npm --prefix web run build`、`docker compose config`、`docker compose build` 通过；REST smoke 验证 `selection_mode=auto` 生成 3 张候选后恰好 1 张 selected，`manual_optional` 生成 2 张候选后 0 张 selected；自动 selected 资产可被 reject。
- Remaining gaps: 当前 best-of 是本地元数据启发式，不包含视觉/LLM 审美打分；未自动 reject 非推荐候选。
