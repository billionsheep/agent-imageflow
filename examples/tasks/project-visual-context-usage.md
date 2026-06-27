# Project Visual Context Examples

这些示例面向外部 agent、自动化脚本和本地 smoke。它们只使用 `mock` provider 字段，不会要求真实 provider 费用，也不应包含 API key、provider key 或其他 secret。

## 1. 保存 Project Visual Context

```bash
docker compose exec -T api /app/vag project context set \
  --workspace ws_default \
  --project prj_xhs_anime \
  --file /app/examples/tasks/sample-project-visual-context.json
```

`sample-project-visual-context.json` 包含三张角色卡：

- `dog_mochi`
- `dog_biscuit`
- `cat_orange`

以及一个 prompt recipe：

- `pet_story_cover`

保存后可以顺手读取一次 project visual context，确认 `reference_diagnostics`：

```bash
curl http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/visual-context
```

现在响应除了 `visual_context` 本体，还会返回只读 `reference_diagnostics`，帮助你提前识别：

- `image_backed`
- `text_constrained`
- `missing_environment_reference`
- `weak_species_lock`

如果这里已经显示 `text_constrained` 或 `missing_environment_reference`，不要把后续真实 provider 漂移误判成“纯 provider 随机问题”。

## 2. CLI 创建多 scene 任务

```bash
docker compose exec -T api /app/vag task create \
  --workspace ws_default \
  --project prj_xhs_anime \
  --campaign cmp_7day_cover \
  --file /app/examples/tasks/sample-pet-story-visual-context-task.json

docker compose exec -T api /app/vag task create \
  --workspace ws_default \
  --project prj_xhs_anime \
  --campaign cmp_7day_cover \
  --file /app/examples/tasks/sample-pet-story-visual-context-scene-002.json

docker compose exec -T api /app/vag task create \
  --workspace ws_default \
  --project prj_xhs_anime \
  --campaign cmp_7day_cover \
  --file /app/examples/tasks/sample-pet-story-visual-context-scene-003.json
```

每个 scene 都会传：

```json
{
  "character_ids": ["dog_mochi", "dog_biscuit", "cat_orange"],
  "prompt_recipe_id": "pet_story_cover",
  "use_project_visual_context": true
}
```

服务端会把角色卡、recipe 和项目默认值展开进 `structured_input_json.visual_context_snapshot`，生成资产时也会把关键快照写入 `parameters_json.visual_context_snapshot`。

## 3. REST 创建任务

```bash
curl -X POST \
  http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  --data @examples/tasks/sample-pet-story-visual-context-rest-create-task.json
```

如果当前 project 已启用 project API key，请调用方自行加现有 `X-API-Key` 或 Bearer header。不要把 key 写入示例文件、日志或项目文档。

## 4. MCP 创建任务

`sample-pet-story-visual-context-mcp-call.json` 是 JSONL 兼容的单行 `tools/call` 示例，核心 tool arguments 与 REST/CLI 使用同一组字段：

```bash
docker compose exec -T api /app/mcp < examples/tasks/sample-pet-story-visual-context-mcp-call.json
```

实际 MCP 客户端通常会先发送 `initialize`，再调用 `tools/list` 和 `tools/call`；本文件用于展示 `create_image_task` 的 arguments 契约。

## 5. 可选 reference asset

当已经有同 workspace/project 下的 selected/generated asset 可作为参考图时，任务可以额外传：

```json
{
  "reference_asset_ids": ["asset_same_project_reference"]
}
```

`reference_asset_ids` 必须属于同一个 workspace/project。服务端会拒绝跨 project asset，且不会复制或删除原 asset；它只把该 asset 作为本次任务的 visual context reference 展开。

## 6. 查询批次进度

```bash
docker compose exec -T api /app/vag batch progress \
  --project prj_xhs_anime \
  --campaign cmp_7day_cover \
  --session-id sample_pet_story_visual_context \
  --batch-id sample_pet_story_visual_context_batch
```

Web Recent Assets 或服务端资产库也可以按 `source/session_id/batch_id/story_id/scene_id` 追踪这些资产。
