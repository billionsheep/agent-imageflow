# Story Continuity Agent Guide

本文面向要接入 Agent ImageFlow 的“故事连续性 agent”。它的目标不是替代平台，而是在调用 MCP 生图前，先把连续故事、固定场景、道具和对白组织成结构化计划。

## 产品边界

Agent ImageFlow 负责：

- 保存 project / campaign / task / asset 的事实源。
- 调 provider 生成或编辑图片。
- 保存 original、thumbnail、metadata 和 delivery URL。
- 记录 Project Visual Context、reference participation、select/reject、manifest 和 audit。

Story Continuity Agent 负责：

- 把用户一句需求扩写成连续故事。
- 生成 Story Bible 和 Panel Plan。
- 为每格选择角色、环境、上一格和 edit target 参考资产。
- 调 MCP 创建任务、查询状态、选择/拒绝资产、拿交付链接。
- 发现角色跑偏、场景漂移、文字不清时，生成重试说明。

Story Continuity Agent 不应该拥有：

- provider key。
- Admin cookie 或 session。
- 服务器环境变量读取权限。
- workspace / project / campaign / asset 删除权限。

## 工作流

第一轮推荐顺序采用 **Sequential Previous Panel Mode**，不要并发提交全部分镜：

1. 读取 project 的角色卡、参考图、prompt recipe 和已有 selected assets。
2. 创建 `story_bible`：固定场景、固定道具、角色关系、画风和禁止变化项。
3. 创建 `panel_plan`：每格的上一格状态、动作、对白、镜头、必须保留物和允许变化物。
4. 创建第一格无字任务，最多 2 个候选，等待人工 `select_image_asset`。
5. 创建第二格任务时，必须把第一格 selected asset 作为 `previous_panel_reference`。
6. 创建第三格任务时，必须把第二格 selected asset 作为 `previous_panel_reference`。
7. 如需重生图，只重生单格，不覆盖旧 task、旧 asset 或其他格 selected 状态。
8. 第一轮不加字；加字作为 Caption/Edit Lineage 后续切片。
9. 用 `get_asset_delivery_info` 或 batch manifest 交付最终资产。

轻连续试用可以使用 **Parallel Minimal Props Mode**：两个角色、少量道具、弱背景、问答递进，并发生成。它适合快速产出，但不能证明上一格参考参与，也不能替代强连续验收。

## Story Bible 最小字段

示例见 `examples/mcp/create-story-bible.json`。第一版建议字段：

- `story_id`
- `title`
- `premise`
- `fixed_environment`
- `characters`
- `continuity_props`
- `style_rules`
- `negative_continuity`
- `delivery_notes`

原则：Story Bible 描述“整组图都必须一致”的东西，不写每格具体动作。

## Story Context V1

第一轮建议把故事上下文统一放进 `story_context_v1`，不要散落在 metadata 根节点：

- `schema_version`
- `story_id`
- `story_revision`
- `story_plan_hash`
- `generation_mode`: `sequential_previous_panel` 或 `parallel_minimal_props`
- `story_bible`
- `panel_plan`
- `reference_bindings`
- `resolved_reference_assets`
- `continuity_policy`

`reference_bindings` 是计划想使用的参考；`resolved_reference_assets` 是平台实际解析到、属于当前 project 且可送入 provider 的资产。只有后者存在，才能把任务称为 reference-assisted continuity。

当前平台第一轮实现约定：

- 从 `create_image_task.arguments.metadata_json.story_context_v1` 读取该 contract。
- Sequential Previous Panel Mode 会强制 `selection_mode=manual_optional`。
- 创建任务后，平台会把当前格的 `panel_index`、因果字段、`previous_panel_asset_id` 和 `provider_reference_participation` 回写到 metadata 根节点，供 summary / manifest / Production View 直接读取。

## Panel Plan 最小字段

示例见 `examples/mcp/create-panel-plan.json`。第一版建议每格字段：

- `panel_index`
- `scene_id`
- `narrative_role`
- `previous_state`
- `trigger_event`
- `visible_action`
- `resulting_state`
- `dialogue`
- `dialogue_intent`
- `camera`
- `must_keep_props`
- `allowed_changes`
- `reference_roles`
- `target_path`

原则：Panel Plan 描述“这一格为什么和上一格连起来”，不要只写一句情绪文案。

## Reference Roles

创建任务时，参考图应该带用途语义。第一版约定：

- `character_reference`：角色主图或角色参考图。
- `environment_reference`：固定场景、房间、沙发、桌面等。
- `previous_panel_reference`：上一格 selected asset，用于延续姿态、物品和构图。
- `style_reference`：画风参考。
- `edit_target`：要被编辑的原图，例如加字时的输入 asset。

如果当前 MCP schema 还没有独立 role 字段，可以先把 role 写入 `story_context_v1.reference_bindings`，并在 prompt 中明确说明。但验收时必须检查 `story_context_v1.resolved_reference_assets`，避免把文字说明误判成真实参考图参与。

## Preflight

创建每格任务前，agent 或平台应确认：

- 角色有真实 reference asset。
- 环境参考存在，或明确标记为文字约束环境。
- 所有引用 asset 属于当前 project。
- provider 支持所需参考图数量。
- `panel_index > 1` 时，上一格必须已经有 selected asset。
- 参考图解析失败时，不应静默退化为纯文生图；必须阻止任务或写入降级 warning。

第一轮实现里，panel 2 / 3 若缺少上一格 selected asset，会直接在 `CreateTask` preflight 失败，而不是继续创建一个看似成功的 mock/文生图任务。

## 生成策略

无字连续图优先：

- 先生成无字的 3-5 格故事图。
- Web 人工选中每格最佳图。
- 再对 selected assets 批量加字。

加字策略：

- 风格化加字：通过 provider edit 生成气泡、星星、手写感，可能轻微重绘画面。
- 稳定贴字：未来 caption renderer 负责不改原图、文字准确、批量秒级产出。

## 失败分类

Story Continuity Agent 应把失败分清楚：

- 平台链路失败：task 没完成、asset 没落盘、delivery 不可用。
- 参考参与失败：reference image 没进入 provider 或 MIME/content type 错误。
- 角色一致性失败：角色颜色、体型、关键特征跑偏。
- 场景连续性失败：背景、道具、镜头关系漂移。
- 文案失败：对白缺因果、文字错误、气泡遮挡。
- Provider 失败：timeout、429、503/504/524、provider_error。

不同失败走不同重试：平台失败查日志和 task attempts；参考失败查 input_file/asset；一致性失败补 reference；文案失败改 Panel Plan；provider 失败降并发或稍后重试。

## 安全边界

- MCP 只用 create/get/list/select/reject/delivery 六类安全工具。
- 不通过 MCP 删除 workspace/project/campaign/asset。
- 不把 provider key、project key、Basic Auth、Admin cookie、session 或 `.env` 内容写进 prompt、metadata、manifest 或 evidence。
- 删除、清理、备份、恢复走 Admin Web / Admin REST / CLI 和 runbook。

## 验收方式

第一轮只要求：

- 3 格无字 story 能顺序完成。
- 每格都有 `story_context_v1`、`panel_index`、panel plan 和 resolved reference 摘要。
- 第二格实际引用第一格 selected asset；第三格实际引用第二格 selected asset。
- Web 能按 story/scene 审图并 select/reject。
- manifest 或 delivery info 能拿到最终图。

Mock 只验证数据链路，不证明视觉连续性。真实 provider 只做人工确认后的低频 canary：provider cap=1，每格最多 2 个候选，总量上限 8 张，不做 benchmark。
