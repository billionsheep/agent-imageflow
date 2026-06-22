# Slice 030: P1 Project Production Context Core

## Status

- State: Done
- Created: 2026-06-20
- Updated: 2026-06-20

## Product Goal

让一个 project 可以承载长期视觉生产上下文：角色/主形象、项目级参考图和 prompt recipe 能被 agent、REST、CLI、MCP 和后续 Web 任务稳定引用，并在创建图片任务时展开成可追踪快照。

## Source Context

- Product spec: Project / Campaign 隔离、metadata_json 扩展、轻量选优和本地优先交付。
- Project plan slice: `issues/next-phase-p1-project-production-context.csv` 的 P1-PCTX-001 到 P1-PCTX-006。
- Tech spec: 复用 Go application core、PostgreSQL `project.metadata_json`、现有 `structured_input_json` / `parameters_json` 快照链路。
- Related decisions: Quality profile 先复用 project metadata；第一版服务端输入文件落 scope 本地存储；不扩展为 DAM、运营系统、多租户、RBAC 或模板市场。

## User Flow

1. 用户或 agent 在 project 下保存角色卡，例如两只狗和一只橘猫的外观、性格、禁止项和主参考资产。
2. 用户把已有 selected/generated asset 标记为项目参考图，说明用途是 character、style、scene 或 prop。
3. 用户保存一个 prompt recipe，包含角色块、风格块、镜头块、渠道要求、negative prompt 和默认输出参数。
4. agent 创建任务时只传 `character_ids`、`reference_asset_ids`、`prompt_recipe_id` 和 `use_project_visual_context`。
5. 服务端在创建任务阶段展开上下文，写入 `structured_input_json`，并让生成资产的 `parameters_json` 保留关键快照。

## In Scope

- 只用 `project.metadata_json.visual_context` 保存第一版数据契约。
- Character/Mascot Profile: `id`、`name`、`status`、`updated_at`、`role`、`appearance`、`personality`、`forbidden`、`primary_asset_id`、`reference_asset_ids`。
- Project Reference Library: 复用已有 asset，保存 `id`、`asset_id`、`purpose`、`label`、`weight`、`notes`、`character_id`、`status`。
- Prompt Recipe / Quality Profile 2.0: 多 recipe，保存 `id`、`name`、`status`、`prompt_blocks`、`negative_prompt`、`default_aspect_ratio`、`default_output_format`、`default_provider`、`default_model`、`generation_config`。
- CreateTask 输入扩展：`character_ids`、`reference_asset_ids`、`prompt_recipe_id`、`use_project_visual_context`。
- 跨 project asset 引用拒绝；归档数据不参与默认展开。
- REST/CLI/MCP 进入同一服务端逻辑。

## Out of Scope

- 不新增数据库表或迁移。
- 不做小红书发布、内容日历、账号运营系统、Usage Tracking、Export Pack。
- 不做通用 DAM、复杂标签体系、模板市场、多人协作、多租户、RBAC 或账号系统。
- 不运行真实 provider，不读取或打印 provider key / API key / secret。
- Web 仅在后续 P1-PCTX-008 做最小面板；本 slice 先完成服务端核心和自动化入口。

## Acceptance Criteria

- Given 一个 project，when 保存角色卡、参考绑定和 prompt recipe，then 它们进入 `project.metadata_json.visual_context`，未知字段不会破坏读取。
- Given 一个角色或 reference binding 引用 asset，when asset 不属于同 workspace/project，then 服务端拒绝保存。
- Given agent 创建任务并引用 visual context，when 任务入队，then `structured_input_json` 包含 `visual_context_snapshot`、展开后的 references、recipe 和角色信息。
- Given 任务显式传入 prompt、negative_prompt、provider、aspect_ratio、output_format 或 generation_config，when recipe/project 默认也存在，then 显式任务字段优先。
- Given mock provider 完成任务，when 查询 asset，then asset `parameters_json` 保留 reference / generation / visual context 关键快照，不暴露本地绝对路径或 provider secret。

## Technical Approach

- 在 domain 增加 project visual context 数据结构和 create task 引用字段。
- 在 store 复用 `jsonb_set(metadata_json, '{visual_context}', ...)`，不做 schema migration。
- 在 service 增加 normalize / validate / update visual context 逻辑，并在 `normalizeTaskRequest` 中先展开 visual context，再应用已有 quality profile 和 input reuse。
- 在 REST 增加 `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/visual-context`。
- 在 CLI 增加 `vag project context get/set`，输入输出均为 JSON。
- 在 MCP `create_image_task` schema 中增加 visual context 引用字段。

## Data / Interface Impact

- 新增 project metadata key: `visual_context`。
- 新增 CreateTask JSON fields: `character_ids`、`reference_asset_ids`、`prompt_recipe_id`、`use_project_visual_context`。
- 任务快照新增 `visual_context_snapshot`，用于后续追溯角色、reference、recipe 和展开来源。
- 不改变现有 `quality_profile`、`provider_profile`、`access_config` 和 asset 状态语义。

## Files or Subsystems Likely to Change

- `internal/domain/types.go`
- `internal/store/postgres.go`
- `internal/app/service.go`
- `internal/app/quality.go`
- `internal/provider/parameters.go`
- `internal/httpapi/server.go`
- `internal/mcp/server.go`
- `cmd/vag/main.go`
- `examples/tasks/`
- `docs/project/`
- `issues/next-phase-p1-project-production-context.csv`

## Verification Plan

```bash
go test ./internal/app ./internal/store ./internal/provider ./internal/mcp ./internal/httpapi ./cmd/vag
go test ./...
docker compose config
# mock-only smoke; no real provider and no secret output
docker compose exec api /app/vag project context set --file /app/examples/tasks/sample-project-visual-context.json
docker compose exec api /app/vag task create --file /app/examples/tasks/sample-pet-story-visual-context-task.json
```

## Assumptions and Risks

- `project.metadata_json` 足够承载第一版数据量；如果后续需要全文检索、跨项目共享或引用治理，再单独确认表设计。
- 当前服务端 asset 读取接口仍按 `asset_id` 返回交付信息；本 slice 只新增 reference binding，不复制文件。
- Web 面板在 P1-PCTX-008 后续实现，本轮服务端核心先保证自动化入口可用。

## Implementation Log

### 2026-06-20

- Changes: 完成 P1-PCTX-001 到 P1-PCTX-006。新增 `project.metadata_json.visual_context` 契约、角色卡、reference binding、prompt recipe、`CreateTask` visual context 展开、REST `GET/POST /visual-context`、CLI `vag project context get/set`、MCP create task 引用字段、provider parameters 快照和示例 JSON。
- Verification: `go test ./...` passed；`npm --prefix web test -- --run` passed；`npm --prefix web run build` passed with existing chunk-size warning；`docker compose config --quiet` passed；Docker mock smoke 创建 `prj_pctx_smoke_1781964012 / cmp_pctx_smoke_1781964012`，CLI/REST task `task_099adb62c6c7d7cb25cb -> asset_84172190f72d61f701c2` completed/approved，MCP task `task_582c7440fd46425545cc -> asset_50d7842428660394d469` completed/approved。
- Remaining gaps at the time: P1-PCTX-007 examples 后续已完成；P1-PCTX-008 到 P1-PCTX-009 当时排在后续实施计划中。Current status: P1-PCTX-008 已在 `slice-038` 完成，P1-PCTX-009 已在 `slice-039` 完成。
