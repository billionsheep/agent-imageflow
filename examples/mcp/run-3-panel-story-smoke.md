# Three Panel Story Continuity Smoke

本 smoke 用于验证 Story Continuity Agent 的最小工作方式：先有 Story Bible 和 Panel Plan，再通过 MCP 创建 3 个 mock scene task。默认不运行真实 provider。

## 前提

```bash
docker compose up -d postgres redis api worker
curl -sf http://localhost:8081/healthz
```

## 参考输入

- Story Bible: `examples/mcp/create-story-bible.json`
- Panel Plan: `examples/mcp/create-panel-plan.json`

当前 MCP 没有专门的 `create_story_bible` 工具；第一版做法是把 story bible 和当前 panel 摘要写入 `create_image_task.arguments.metadata_json`，由 task/asset metadata 和 manifest 保留追踪。

## 1. tools/list

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":"list","method":"tools/list","params":{}}' \
  | docker compose exec -T api /app/mcp
```

期望仍然只有安全工具：create/get/list/select/reject/delivery，不出现删除类工具。

## 2. create scene 001 mock task

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":"story-scene-001","method":"tools/call","params":{"name":"create_image_task","arguments":{"workspace_id":"ws_default","project_id":"prj_xhs_anime","campaign_id":"cmp_7day_cover","title":"Story continuity scene 001","purpose":"story_continuity_panel","prompt":"小白坐在同一个暖粉色客厅的粉色沙发左侧，抱着书假装专心，眼睛看向门口。保持固定道具：左侧小圆桌、粉色马克杯、黄色抱枕、月亮挂灯、浅粉毯子堡垒。无字图。","provider":"mock","requested_count":1,"selection_mode":"auto","review_required":false,"use_project_visual_context":false,"metadata_json":{"source":"story-continuity-agent","workflow_step":"story_panel_generation","story_id":"pet_blanket_fort_story","scene_id":"scene_001","session_id":"story_continuity_smoke_session","batch_id":"story_continuity_smoke_batch","target_path":"stories/pet_blanket_fort_story/scene_001.png","story_bible_id":"pet_blanket_fort_story","panel_dialogue":"才没有等你","panel_action":"小白抱着书假装专心，但眼睛看向门口。","continuity_props":["粉色沙发","左侧小圆桌","粉色马克杯","黄色抱枕","月亮挂灯","浅粉毯子堡垒"],"reference_roles":{"character_reference":["dog_xiaobai"],"environment_reference":["warm_pink_living_room"]}}}}}' \
  | docker compose exec -T api /app/mcp
```

记录返回的 `task_id`。

## 3. create scene 002 / 003

按 `examples/mcp/create-panel-plan.json` 的 `scene_002` 和 `scene_003` 复制上一条命令，改：

- `id`
- `title`
- `prompt`
- `scene_id`
- `target_path`
- `panel_dialogue`
- `panel_action`
- `reference_roles`

如果已有上一格 selected asset，可在 metadata 中追加：

```json
{
  "previous_panel_asset_id": "asset_replace_with_selected_scene_001",
  "reference_roles": {
    "previous_panel_reference": [
      "asset_replace_with_selected_scene_001"
    ]
  }
}
```

## 4. 查询和交付

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":"list-story-assets","method":"tools/call","params":{"name":"list_image_assets","arguments":{"project_id":"prj_xhs_anime","campaign_id":"cmp_7day_cover","source":"story-continuity-agent","session_id":"story_continuity_smoke_session","batch_id":"story_continuity_smoke_batch","limit":10}}}' \
  | docker compose exec -T api /app/mcp
```

对目标 `asset_id`：

```bash
printf '%s\n' "{\"jsonrpc\":\"2.0\",\"id\":\"delivery\",\"method\":\"tools/call\",\"params\":{\"name\":\"get_asset_delivery_info\",\"arguments\":{\"asset_id\":\"<asset_id>\"}}}" \
  | docker compose exec -T api /app/mcp
```

## 验收

- 3 个 scene task 都进入 completed。
- 每张 asset metadata 都有 `story_id`、`scene_id`、`panel_dialogue`、`panel_action` 和 `continuity_props`。
- Web 可按 session/batch/story/scene 找到这组图。
- 不运行真实 provider，不打印任何 key、cookie、session 或 provider secret。
