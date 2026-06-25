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

推荐顺序：

1. 读取 project 的角色卡、参考图、prompt recipe 和已有 selected assets。
2. 创建 `story_bible`：固定场景、固定道具、角色关系、画风和禁止变化项。
3. 创建 `panel_plan`：每格的上一格状态、动作、对白、镜头、必须保留物和允许变化物。
4. 对每格调用 MCP `create_image_task`，把 `story_bible` 和当前 panel 写入 `metadata_json` 或结构化输入快照。
5. 等待 `get_image_task` completed，再用 `list_image_assets` 或 Web Production View 审图。
6. 用 `select_image_asset` 标记每格最终图；错图用 `reject_image_asset`，不要删除。
7. 如需加字，基于 selected asset 再创建 caption edit task，记录 `derived_from_asset_id` 和 `caption_text`。
8. 用 `get_asset_delivery_info` 或 batch manifest 交付最终资产。

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

## Panel Plan 最小字段

示例见 `examples/mcp/create-panel-plan.json`。第一版建议每格字段：

- `scene_id`
- `previous_state`
- `action`
- `dialogue`
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

如果当前 MCP schema 还没有独立 role 字段，可以先把 role 写入 `metadata_json.reference_roles`，并在 prompt 中明确说明。

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

- 3 格 mock story 能完成。
- 每格都有 `story_id`、`scene_id`、panel plan 和 reference role 摘要。
- Web 能按 story/scene 审图并 select/reject。
- manifest 或 delivery info 能拿到最终图。

真实 provider 只做人工确认后的低频 canary，不做 benchmark。
