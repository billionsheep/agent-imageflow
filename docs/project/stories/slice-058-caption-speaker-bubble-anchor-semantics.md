# Story: 058 - Caption Speaker / Bubble Anchor Semantics

## Status

- State: Done
- Created: 2026-06-26
- Updated: 2026-06-26

## Background

真实萌宠情侣文案卡试跑已经证明 caption edit 可以走通最小 lineage，但也暴露了更细的生产问题：调用方虽然能传 `caption_text`，平台却缺少“这句话是谁说的、气泡大概应该放哪、尾巴朝向谁、是否要避开主角脸和关键道具”的结构语义。

如果这些信息只留在自然语言 prompt 里，调用方写法会分散，manifest 也无法稳定追溯。

## Goal

在不新增数据库表、不新增 MCP 工具、不扩 Web 排版器的前提下，让 MCP / REST 调用方可以把 caption 说话人和气泡锚点语义结构化地传给平台，并让这些语义进入：

- `metadata_json.caption_lineage`
- `structured_input_json.caption_lineage`
- provider 可见的 prompt 约束
- provider parameters
- batch summary / manifest asset

## Scope

### In Scope

- 扩展 `CaptionLineageSummary`
- 兼容旧根字段输入，并统一归一化到 `metadata_json.caption_lineage`
- 在创建任务时把 caption 语义追加成 provider 可见的 prompt 约束
- 更新 MCP `create_image_task` schema 对 `metadata_json.caption_lineage` 的说明
- 更新 caption edit 示例和状态文档

### Out of Scope

- Web 一键加字入口
- 确定性 caption renderer
- 批量 caption UI
- 真实 provider canary
- 新增 MCP destructive tools

## Contract

第一版 `caption_lineage` 支持：

- `derived_from_asset_id`
- `derivation_type`
- `caption_text`
- `caption_style`
- `source_task_id`
- `source_scene_id`
- `speaker_character_id`
- `bubble_anchor`
- `tail_direction`
- `caption_intent`
- `avoid_covering_subjects`

推荐调用方式：

- 外部输入优先写到 `metadata_json.caption_lineage`
- delivery 变体尽量保留原 `scene_id`
- `speaker_character_id` 复用 project visual context 的角色 id

## Implementation

- `internal/domain/types.go`
  - 扩展 `CaptionLineageSummary`
  - 扩展 metadata / structured input 提取与 merge helper
- `internal/app/service.go`
  - 新增 caption lineage metadata 归一化 helper
  - 把 caption speaker / bubble 语义追加到 provider 可见 prompt
- `internal/mcp/server.go`
  - 为 `metadata_json.caption_lineage` 增加显式 schema 提示
- `examples/mcp/create-caption-edit-task.json`
  - 示例改为使用嵌套 `caption_lineage`
  - delivery variant 保留原 `scene_id`

## Verification

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25.3 go test ./internal/domain ./internal/app ./internal/provider ./internal/mcp
npm --prefix web test -- src/lib/agentImageflowApi.test.ts
python3 -m json.tool examples/mcp/create-caption-edit-task.json >/dev/null
git diff --check
```

## Result

`V02-MCPH-004` 已完成。当前平台已经能让调用方用结构字段表达 caption 说话人和气泡锚点语义，并把这层语义稳定保留到 structured input、provider parameters、summary / manifest 和交付示例中。

这一轮仍然不承诺 deterministic 气泡排版；真实视觉效果稳定性要到后续受控 canary 再验证。
