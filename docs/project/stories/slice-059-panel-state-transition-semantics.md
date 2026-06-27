# Story: 059 - Panel State Transition Semantics

## Status

- State: Done
- Created: 2026-06-26
- Updated: 2026-06-26

## Background

真实连续故事试跑已经证明 `sequential_previous_panel` 能保住“上一格引用”的基础形态，但也暴露出明显短板：如果平台只知道 `previous_state`、`visible_action` 和 `must_keep_props`，它更容易把上一格“冻住”，不容易把情绪、姿态和关系真正推进到下一格。

这会让业务结果变成“连续但不生动”。

## Goal

在不新增数据库表、不把服务端变成创作脑的前提下，让 `story_context_v1.panel_plan` 能显式表达：

- 上一格和这一格的情绪变化
- 姿态变化
- 关系推进
- 这一格必须变化什么
- 这一格不能继续保留什么
- 对 provider 的补充状态转场说明

并把这些语义贯通到 metadata、summary / manifest、Web 审图和 provider prompt 约束。

## Scope

### In Scope

- 扩展 `StoryPanelPlanEntry`
- 扩展 `BatchStoryContinuitySummary`
- 在创建任务时把状态转场字段回写到 metadata
- 追加 provider 可见的 `State transition requirements` prompt 约束
- 在 summary / manifest continuity 与 Web Production View 中透传展示
- 更新 story context 示例和状态文档

### Out of Scope

- 平台替调用方生成剧情
- 自动推断角色表演
- AI 自动判定“表演是否自然”
- 真实 provider canary
- 新增数据库表或 destructive tools

## Contract

`story_context_v1.panel_plan[]` 第一版新增：

- `emotion_before`
- `emotion_after`
- `pose_change`
- `relationship_shift`
- `must_change`
- `must_not_keep`
- `state_transition_notes`

建议语义：

- `must_keep_props` / `allowed_changes` 表达“保留哪些事实”
- `must_change` / `must_not_keep` 表达“这一格必须推进什么、不能继续冻结什么”
- `emotion_before/after`、`pose_change` 和 `relationship_shift` 尽量写成观众能看懂的具体变化

## Implementation

- `internal/domain/types.go`
  - 扩展 `StoryPanelPlanEntry`
  - 扩展 `BatchStoryContinuitySummary`
- `internal/app/story_continuity.go`
  - metadata 回写新增状态转场字段
  - 新增 provider prompt 约束 helper
- `internal/store/postgres.go`
  - 提取 continuity summary 时透传新字段
- `web/src/components/ProductionViewModal.tsx`
  - 审图态新增前后情绪、姿态变化、关系推进、必须变化/去掉和状态说明
- `examples/mcp/create-story-context-v1.json`
  - 同步为新字段示例

## Verification

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25.3 go test ./internal/app ./internal/store ./internal/provider ./internal/mcp ./internal/httpapi ./cmd/api ./cmd/vag
npm --prefix web test -- src/lib/agentImageflowApi.test.ts
npm --prefix web run build
python3 -m json.tool examples/mcp/create-story-context-v1.json >/dev/null
git diff --check
```

## Result

`V02-MCPH-005` 已完成。现在平台已经能把“上一格要保住什么、这一格必须推进什么”明确写进 contract，并稳定透传到 metadata、summary / manifest、Web 审图和 provider prompt 约束。

这一轮仍然不承诺“复杂长链一定自然”，但至少把失败归因从“平台吞语义”收敛成“调用方脚本质量 / provider 能力 / reference 条件”的更清晰边界。
