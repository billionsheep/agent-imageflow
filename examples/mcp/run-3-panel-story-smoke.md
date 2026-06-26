# Three Panel Story Continuity Smoke

本 smoke 用于验证 Story Continuity Agent 的最小工作方式：先有 Story Bible 和 Panel Plan，再通过 MCP 顺序创建 3 个 mock scene task。默认不运行真实 provider，只证明 metadata、preflight、select/reject、summary 和 manifest 数据链路，不证明视觉连续性。

## 前提

```bash
docker compose up -d postgres redis api worker
curl -sf http://localhost:8081/healthz
```

## 参考输入

- Story Bible: `examples/mcp/create-story-bible.json`
- Panel Plan: `examples/mcp/create-panel-plan.json`

当前 MCP 没有专门的 `create_story_bible` 工具；第一版做法是把统一的 `story_context_v1` 放进 `create_image_task.arguments.metadata_json`，由 task/asset metadata、summary 和 manifest 保留追踪。

## 1. tools/list

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":"list","method":"tools/list","params":{}}' \
  | docker compose exec -T api /app/mcp
```

期望仍然只有安全工具：create/get/list/select/reject/delivery，不出现删除类工具。

## 2. 准备 panel 1 的 `story_context_v1`

推荐直接使用 `examples/mcp/create-story-context-v1.json` 作为模板，并把 `panel_plan[0]` 对应 scene 作为当前任务：

- `scene_id=scene_001`
- `panel_index=1`
- `reference_bindings.previous_panel_reference=[]`
- `continuity_policy.mode=sequential_previous_panel`
- `continuity_policy.require_previous_selected_asset=true`
- `continuity_policy.max_candidates_per_panel=2`

## 3. create scene 001 mock task

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":"story-scene-001","method":"tools/call","params":{"name":"create_image_task","arguments":{"workspace_id":"ws_default","project_id":"prj_xhs_anime","campaign_id":"cmp_7day_cover","title":"Story continuity scene 001","purpose":"story_continuity_panel","prompt":"小白坐在同一个暖粉色客厅的粉色沙发左侧，抱着书假装专心，眼睛看向门口。保持固定道具：左侧小圆桌、粉色马克杯、黄色抱枕、月亮挂灯、浅粉毯子堡垒。无字图。","provider":"mock","requested_count":2,"selection_mode":"manual_optional","review_required":true,"use_project_visual_context":false,"metadata_json":{"source":"story-continuity-agent","workflow_step":"story_panel_generation","session_id":"story_continuity_smoke_session","batch_id":"story_continuity_smoke_batch","story_id":"pet_blanket_fort_story","scene_id":"scene_001","story_context_v1":{"schema_version":"1.0","story_id":"pet_blanket_fort_story","story_revision":"rev_001","story_plan_hash":"sha256:example_replace_with_real_hash","generation_mode":"sequential_previous_panel","story_bible":{"title":"小白和鸡毛的毯子堡垒","premise":"小白假装没有等鸡毛，鸡毛端着牛奶出现，两只小狗慢慢靠近。","fixed_environment":"同一个暖粉色客厅，同一张粉色双人沙发，左侧小圆桌，桌上粉色马克杯，右侧黄色抱枕，背景有月亮挂灯。","continuity_props":["粉色双人沙发","左侧小圆桌","粉色马克杯","黄色抱枕","月亮挂灯","浅粉毯子堡垒"],"style_rules":["可爱手绘感","暖色柔光","弱背景","角色和核心道具优先"],"negative_continuity":["不要改变狗的颜色和体型","不要切换到户外或其他房间","不要新增无关角色"]},"panel_plan":[{"panel_index":1,"scene_id":"scene_001","narrative_role":"setup","previous_state":"小白独自坐在粉色沙发左侧。","trigger_event":"鸡毛还没有出现，小白想隐藏自己正在等待的心情。","visible_action":"小白抱着书假装专心，但眼睛看向门口。","resulting_state":"观众知道小白其实在等鸡毛。","dialogue":"才没有等你","dialogue_intent":"嘴硬地否认等待，为第二格鸡毛出现做铺垫。","must_keep_props":["粉色沙发","粉色马克杯","浅粉毯子堡垒"],"allowed_changes":["小白表情","书本角度"],"target_path":"stories/pet_blanket_fort_story/scene_001.png"},{"panel_index":2,"scene_id":"scene_002","narrative_role":"arrival","previous_state":"小白坐在沙发左侧，门口方向留出空间。","trigger_event":"鸡毛回应小白的等待，带着牛奶出现。","visible_action":"鸡毛从右侧出现，端着一杯热牛奶，小白偷偷抬头。","resulting_state":"两只小狗进入同一空间，牛奶成为共享道具。","dialogue":"牛奶来啦","dialogue_intent":"温柔回应小白，推进两只角色靠近。","must_keep_props":["粉色沙发","粉色马克杯","浅粉毯子堡垒"],"allowed_changes":["鸡毛进入画面","新增一杯热牛奶"],"target_path":"stories/pet_blanket_fort_story/scene_002.png"},{"panel_index":3,"scene_id":"scene_003","narrative_role":"closeness","previous_state":"鸡毛已经坐到沙发右侧，牛奶在小圆桌上。","trigger_event":"鸡毛坐下后，小白终于愿意靠近。","visible_action":"小白往鸡毛身边挪一点，鸡毛用尾巴轻轻圈住小白。","resulting_state":"两只小狗靠在一起，关系从等待转为贴近。","dialogue":"靠近一点点","dialogue_intent":"小白表达想靠近但仍然害羞。","must_keep_props":["粉色沙发","粉色马克杯","浅粉毯子堡垒","热牛奶"],"allowed_changes":["两只小狗靠近","尾巴动作"],"target_path":"stories/pet_blanket_fort_story/scene_003.png"}],"reference_bindings":{"character_reference":["dog_xiaobai","dog_jimao"],"environment_reference":["warm_pink_living_room"],"previous_panel_reference":[]},"resolved_reference_assets":[],"continuity_policy":{"mode":"sequential_previous_panel","require_previous_selected_asset":true,"max_panels":3,"max_candidates_per_panel":2,"provider_concurrency":1,"max_total_images_including_regenerate":8,"caption_enabled":false}}}}}}' \
  | docker compose exec -T api /app/mcp
```

记录返回的 `task_id`。mock provider 下该任务会落 1-2 张候选图，但仍需要人工 `select_image_asset` 选中 1 张，后续 panel 2 才能通过 preflight。

## 4. select panel 1

先通过 `list_image_assets` 按 `session_id/batch_id/story_id/scene_id` 找到 `scene_001` 的 asset，再调用 `select_image_asset` 选中其中一张。记下它的 `asset_id`，下文用 `<panel1_selected_asset_id>` 表示。

## 5. create scene 002 / 003

按 `examples/mcp/create-panel-plan.json` 的 `scene_002` 和 `scene_003` 复制上一条命令，并同时更新 `story_context_v1`：

- `id`
- `title`
- `prompt`
- `scene_id`
- `panel_plan[*]` 保持完整 3 格
- 当前格对应的 `panel_index`
- `reference_bindings.previous_panel_reference`

scene 002 必须把 panel 1 的 selected asset 写入 `story_context_v1.reference_bindings.previous_panel_reference`：

```json
["<panel1_selected_asset_id>"]
```

scene 003 同理，必须先选中 panel 2，再把它的 selected asset 写入 `previous_panel_reference`。如果没有上一格 selected asset，当前后端 preflight 会直接拒绝创建强连续任务。

## 6. 查询和交付

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
- 每张 asset metadata 都有 `story_id`、`scene_id`、`panel_index`、`narrative_role`、`dialogue`、`dialogue_intent`、`provider_reference_participation` 和 `story_context_v1`。
- scene 002 metadata 有 `previous_panel_asset_id=<panel1_selected_asset_id>`；scene 003 同理指向 panel 2 selected asset。
- summary / Production View / manifest 能看到 `panel_index`、`previous_panel_asset_id`、resolved reference 摘要和 continuity warnings。
- mock 只证明数据链路，不证明视觉连续性；不运行真实 provider，不打印任何 key、cookie、session 或 provider secret。
