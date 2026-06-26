# Slice 054: Story Continuity MVC

## Context

V1 baseline 已完成 Project Visual Context、Batch / Story / Scene summary、Web Production View、scene select/reject/regenerate 和 JSON manifest。当前缺口不是“能不能批量生成图”，而是“连续故事能不能形成最小闭环”。

外部评审已经把上位 Story / Caption 计划收敛为 `issues/next-phase-p1-story-continuity-mvc.csv`：第一轮只做 3 格、无字、顺序生成、人工选图、真实参考图参与。平台继续作为图片资产事实源，不扩成漫画编辑器，也不新增复杂 Story Review 页面。

## Product Goal

让 Agent ImageFlow 在现有 MCP / REST / Web 基座上，支持 3 格连续故事的最小闭环：

1. 用统一 `story_context_v1` 保存故事上下文和 panel 因果信息。
2. 顺序约束 panel 2 必须依赖 panel 1 selected asset，panel 3 必须依赖 panel 2 selected asset。
3. 区分“计划引用的参考”和“实际解析并参与 provider 的参考”。
4. 复用现有 Production View / manifest 展示连续性，而不是新建 Story Review 系统。

## User Flow

1. 外部 agent 先准备 `story_context_v1`，包含 story bible、panel plan、reference bindings 和 continuity policy。
2. Agent 创建 panel 1 任务，最多 2 个候选，不自动选中。
3. 人工在 Web Recent Assets / Production View 中选择 panel 1 的 selected asset。
4. Agent 创建 panel 2 任务，必须把 panel 1 selected asset 作为 `previous_panel_reference`。
5. 人工选择 panel 2 的 selected asset。
6. Agent 创建 panel 3 任务，必须把 panel 2 selected asset 作为 `previous_panel_reference`。
7. Web 用 Production View 显示 panel 顺序、连续性摘要和参考参与结果。
8. 交付侧通过 manifest 输出最终 3 格故事和 `story_context_v1` 摘要。

## Scope

In scope:

- `story_context_v1` contract 和 fixture。
- panel causality 字段：`panel_index`、`narrative_role`、`previous_state`、`trigger_event`、`visible_action`、`resulting_state`、`dialogue`、`dialogue_intent`、`must_keep_props`、`allowed_changes`、`target_path`。
- `reference_bindings` 与 `resolved_reference_assets` 分离。
- Sequential Previous Panel Mode preflight。
- panel 1 -> panel 2 -> panel 3 的 selected gating。
- Production View / Technical details 最小连续性展示。
- manifest 输出 `story_context_v1` 摘要。
- mock 3 格 smoke，只验证数据链路。

Out of scope:

- 完整漫画编辑器。
- 新建复杂 Story Review 页面。
- Web 一键加字、批量 caption、caption renderer。
- ZIP 导出。
- MCP 删除、清库或其他 destructive tools。
- 默认真实 provider canary。
- 批量并发连续故事 benchmark。

## Acceptance Criteria

- `CreateTask` 可接收 `metadata_json.story_context_v1`，并把同一 story revision/hash 写入 task structured input 和 asset parameters snapshot。
- 每格都能从 `story_context_v1` 中解析出当前 panel 的 `panel_index` 和因果字段，summary / manifest 按 `panel_index` 排序。
- `reference_bindings` 只表示计划引用；`resolved_reference_assets` 只表示同 project、已解析、可参与 provider 的真实参考。
- Sequential 模式下，panel 2 没有 panel 1 selected asset 时不能创建强连续任务；panel 3 同理。
- 顺序模式默认不允许自动最终选图；每格最多 2 候选。
- Production View 能显示 `panel_index`、`narrative_role`、`dialogue`、`previous_panel_asset_id`、resolved reference count、provider reference participation 和 continuity warnings。
- manifest 能按 3 格输出 selected story，并带 `story_revision`、`story_plan_hash`、`panel_index`、`previous_panel_asset_id`、resolved references 和 continuity warnings 摘要。
- mock smoke 明确写出“只验证数据链路，不证明视觉连续性”。

## Technical Approach

- 保持 metadata-only 路径，不新增数据库表或 migration。
- 在 task `structured_input_json` 中增加 `story_context_v1` 顶层快照，`metadata_json` 继续保留对外业务字段。
- 复用 Project Visual Context，把 `story_context_v1.reference_bindings` 中的角色/参考声明映射到现有 `character_ids`、`reference_asset_ids` 和 direct `reference_images`。
- 在 task normalize 阶段执行 story continuity preflight，并把结果写入 `story_context_v1.resolved_reference_assets` 与 continuity warnings。
- 在 batch summary / manifest 抽取连续性摘要，避免前端自己解析深层 metadata。
- Web 只扩展现有 Production View scene card，不新增新页面。

## Data / Interface Impact

- Go domain：新增 story continuity contract / summary types。
- `CreateTask`：增强 `metadata_json.story_context_v1` 解析、引用展开和 sequential preflight。
- batch summary / manifest：新增 continuity summary 字段。
- Web `ProductionViewModal` / `agentImageflowApi`：新增连续性展示字段和测试。

## Files Likely To Change

- `internal/domain/types.go`
- `internal/app/service.go`
- `internal/app/visual_context.go`
- `internal/app/batch_manifest_test.go`
- `internal/store/postgres.go`
- `internal/provider/parameters.go`
- `internal/provider/parameters_test.go`
- `web/src/lib/agentImageflowApi.ts`
- `web/src/components/ProductionViewModal.tsx`
- `examples/mcp/create-story-context-v1.json`
- `examples/mcp/run-3-panel-story-smoke.md`
- `issues/next-phase-p1-story-continuity-mvc.csv`
- `docs/project/TASKS.md`
- `docs/project/PROJECT_STATUS_MAP.md`
- `docs/project/V1_BASELINE_AND_ROADMAP.md`
- `docs/project/CHECKPOINTS.md`
- `docs/project/DECISIONS.md`

## Verification

- Focused Go tests for story continuity contract, preflight and manifest.
- Focused Go tests for provider parameter snapshot propagation.
- Web tests for Production View minimal continuity display.
- Mock 3-panel smoke example update.
- `git diff --check`

## Assumptions and Risks

- 第一轮连续性依赖已有 Project Visual Context；如果项目里没有可用角色/场景参考，preflight 应明确失败，而不是静默退化。
- 当前 summary / manifest 仍基于 metadata 聚合；如果后续需求扩展到大规模 story review，再评估是否升表。
- 真实 provider 连续性质量不在本轮默认验证范围内；mock 只能证明数据链路。
