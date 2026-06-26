# Story: 055 - Caption Edit Lineage MCP First

## Status

- State: Done
- Created: 2026-06-26
- Updated: 2026-06-26

## Product Goal

让外部 agent 通过 MCP 引用已有 asset 创建加字 edit 任务时，派生图能以最小正式语义追溯原图、caption 文案和来源 scene，而不是只靠临时口头约定。

## Source Context

- Product spec: Agent ImageFlow 作为图片资产事实源，负责资产生产、metadata、审核状态和交付。
- Project plan slice: `issues/next-phase-p1-caption-edit-lineage.csv`
- Tech spec: CreateTask、provider parameters、asset metadata、batch summary / manifest 继续复用现有 metadata 和 structured input 链路。
- Related decisions: Story Continuity 与 Caption/Edit Lineage 不扩成 Web 加字入口、批量 caption UI 或 renderer。

## User Flow

1. Agent 已有一张 selected 原图 asset。
2. Agent 通过 MCP `create_image_task` 传 `reference_images[].asset_id`，role 为 `edit_target`。
3. Agent 在 `metadata_json` 写入 `derived_from_asset_id`、`derivation_type=caption_edit`、`caption_text`、`caption_style`、`source_task_id` 和 `source_scene_id`。
4. Worker / provider 参数和 manifest 能看到 caption lineage 摘要，交付 JSON 不暴露本地路径。

## In Scope

- 定义最小 `CaptionLineageSummary`。
- 从 task `metadata_json` 提取 caption lineage，写入 structured input / provider parameters。
- 在 batch summary / manifest asset 上透传 caption lineage。
- 新增 MCP caption edit 示例。
- 更新 P1-CAP-002、P1-CAP-004、P1-CAP-007 和项目状态文档。

## Out of Scope

- Web 一键加字入口。
- 批量 caption UI。
- 确定性 caption renderer。
- 真实 provider canary。
- 新增 MCP 删除工具或 destructive tool。

## Acceptance Criteria

- Given MCP create task metadata 包含 caption lineage，when provider parameters 生成，then 可看到 `caption_lineage` 摘要。
- Given batch manifest 包含 caption edit 派生 asset，when 导出 JSON，then asset 可见 `caption_lineage` 且不包含 `local_path`。
- Given 新 agent 查看示例，when 按示例创建任务，then 使用 `reference_images[].asset_id` 和 `role=edit_target`，provider 默认为 `mock` 且不包含真实 key。

## Technical Approach

- 复用 `metadata_json` 作为第一版输入 contract，避免数据库迁移。
- 在 domain 层新增 `CaptionLineageSummary` 和 metadata / structured input 提取 helper。
- `normalizeTaskRequest` 将 lineage 摘要写入 `structured_input_json.caption_lineage`。
- provider 参数优先读取 structured `caption_lineage`，否则回退解析 `metadata_json`。
- batch summary 从 task structured input 派生 asset caption lineage，manifest 原样透传。

## Data / Interface Impact

- 新增公开 JSON 字段：`caption_lineage`。
- 支持字段：`derived_from_asset_id`、`derivation_type`、`caption_text`、`caption_style`、`source_task_id`、`source_scene_id`。
- 不新增数据库列，不改变现有 MCP tool schema。

## Files or Subsystems Likely to Change

- `internal/domain/types.go`
- `internal/provider/parameters.go`
- `internal/app/service.go`
- `internal/store/postgres.go`
- `examples/mcp/create-caption-edit-task.json`
- `issues/next-phase-p1-caption-edit-lineage.csv`
- `docs/project/TASKS.md`
- `docs/project/CHECKPOINTS.md`
- `docs/project/PROJECT_STATUS_MAP.md`
- `docs/project/V1_BASELINE_AND_ROADMAP.md`
- `docs/project/DECISIONS.md`

## Verification Plan

```bash
docker run --rm -v "$PWD":/src -w /src golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./internal/provider ./internal/app'
docker run --rm -v "$PWD":/src -w /src golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./internal/domain ./internal/provider ./internal/app ./internal/store ./internal/mcp'
git diff --check
```

## Assumptions and Risks

- 第一版只表达派生关系，不证明真实 provider 文字质量。
- 同一个 caption edit task 生成的多个候选图共享同一 lineage 摘要。
- `caption_text` 可能包含用户文案，manifest 会按任务 metadata 原样输出给有权限的交付调用方。

## Implementation Log

### 2026-06-26

- Changes: 新增 caption lineage domain 摘要、provider parameters 透传、manifest asset 透传、MCP 示例和项目状态更新。
- Verification: `docker run --rm -v "$PWD":/src -w /src golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./internal/provider ./internal/app'` 通过；相关包 `go test ./internal/domain ./internal/provider ./internal/app ./internal/store ./internal/mcp` 通过；`python3 -m json.tool examples/mcp/create-caption-edit-task.json >/dev/null` 通过；`git diff --check` 通过。
- Remaining gaps: Web 加字入口、批量 caption UI、renderer 和真实 provider canary 仍后置。
