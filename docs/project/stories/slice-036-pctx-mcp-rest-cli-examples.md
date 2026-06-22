# Slice 036: P1 Project Context MCP REST CLI Examples

## Status

- State: Done
- Created: 2026-06-21
- Updated: 2026-06-21

## Product Goal

让外部 agent、脚本和人工 smoke 都能直接照 examples 使用 Project Visual Context：先保存 project context，再通过 CLI、REST 或 MCP 创建带 `character_ids` / `prompt_recipe_id` / `use_project_visual_context` 的萌宠故事 scene 任务。

## Source Context

- Project plan slice: `issues/next-phase-p1-project-production-context.csv` 的 `P1-PCTX-007 MCP REST CLI contract and examples`。
- Current state: 服务端核心 P1-PCTX-001 到 P1-PCTX-006 已完成；已有基础 context JSON 和单 scene task JSON，但缺少成套 REST/MCP/多 scene 使用示例。
- Existing boundary: 不运行真实 provider，不读取、不打印 API key / provider key / secret。

## User Flow

1. 用户或 agent 用 `sample-project-visual-context.json` 保存角色卡和 prompt recipe。
2. CLI 可直接用 scene JSON 创建任务。
3. REST 调用方可复用同字段创建任务。
4. MCP 调用方可用 `tools/call create_image_task` 示例创建任务。
5. 每个 scene 的 metadata 都保留 `source/session_id/batch_id/story_id/scene_id/target_path`，方便后续资产库和 batch progress 查询。

## In Scope

- 补充多 scene 萌宠故事 task JSON 示例。
- 补充 REST create task body 示例。
- 补充 MCP `tools/call` JSON-RPC 示例。
- 补充 examples 使用说明。
- 更新 CSV 和项目管理文档 evidence。

## Out of Scope

- 不改 API、MCP schema、CLI 参数或服务端业务逻辑。
- 不运行真实 provider。
- 不做 Web context 面板；该项仍属于 P1-PCTX-008。
- 不做完整萌宠故事 mock 回归；该项仍属于 P1-PCTX-009。

## Acceptance Criteria

- Examples 明确展示 `character_ids`、`prompt_recipe_id`、`use_project_visual_context` 和 story/scene metadata。
- REST、CLI、MCP 示例字段与现有契约一致。
- JSON 示例可被解析。
- 文档说明 reference asset id 需来自同 workspace/project，且 project API key 规则不变。

## Technical Approach

- 只新增 examples 和文档，不触碰 Go/TS runtime。
- 用 JSON parse 检查所有新增示例文件。
- 用现有 tests/build 作为回归保护。

## Data / Interface Impact

- 无 API、数据库或运行时行为变更。
- 新增 examples 文件作为外部 agent / script 的契约样例。

## Files or Subsystems Likely to Change

- `examples/tasks/`
- `docs/project/RUNBOOK.md`
- `docs/project/TASKS.md`
- `docs/project/PROJECT_PLAN.md`
- `docs/project/PROJECT_STATUS_MAP.md`
- `docs/project/CHECKPOINTS.md`
- `issues/next-phase-p1-project-production-context.csv`

## Verification Plan

```bash
node -e 'for (const f of process.argv.slice(1)) JSON.parse(require("fs").readFileSync(f,"utf8"))' <json files>
npm --prefix web test -- --run
npm --prefix web run build
git diff --check
```

## Assumptions and Risks

- Examples use mock provider defaults and do not imply real provider cost.
- REST examples omit API key headers; if project API key is enabled, callers must add their existing `X-API-Key` or Bearer token without writing it into example files.

## Implementation Log

### 2026-06-21

- Started implementation for `P1-PCTX-007 MCP REST CLI contract and examples`.
- Added `project-visual-context-usage.md`, scene 002/003 CLI task examples, REST create task body example, and JSONL-compatible MCP `tools/call` example.
- JSON parse verification passed for 6 Project Visual Context example files.
- Mock smoke used existing `prj_pctx_smoke_1781964012 / cmp_pctx_smoke_1781964012`: visual context set succeeded, CLI task `task_d1bdf01c13450287d6d2`, REST task `task_bf41909b09cdb53c7387`, and MCP task `task_7efdece0e104a504f613` were created with `visual_context_snapshot`.
- Batch progress for `sample_pet_story_visual_context_batch` returned `task_count=4`, `succeeded_count=4`, `asset_count=8`.
- Default project correctly required project API key; no key was read or printed.
- Verification passed: `npm --prefix web test -- --run`, `npm --prefix web run build`, and `git diff --check`.
