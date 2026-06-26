# Pet Story Production Workflow

本文是“真实萌宠账号业务推进”的执行入口。目标不是再补一组验收 smoke，而是把 Agent ImageFlow 用作 MCP-first 的图片资产生产平台：外部 Story Continuity Agent 负责连续故事创作与编排，平台负责图片生成、资产追踪、审图状态和交付事实源。

## 结论

下一步真实业务主线是：

```text
Project / Campaign 准备
  -> Story Continuity Agent 生成 Story Bible + Panel Plan
  -> Agent 通过 MCP 顺序创建 3-6 个 panel task
  -> 平台生成候选图并保存 task / asset / metadata
  -> Web Production View 人工 select / reject
  -> 导出 selected manifest 或 delivery info
  -> 按问题类型回填后续 backlog
```

Web 是审图、管理和交付控制台，不是主创作入口。Story Bible、Panel Plan、重试说明和每格因果关系应由外部 Story Continuity Agent 产出，并通过 MCP 写入 `create_image_task.arguments.metadata_json.story_context_v1`。

## 角色分工

### Story Continuity Agent

- 读取 project 的角色卡、参考图、prompt recipe、quality/profile 和已有 selected assets。
- 生成 Story Bible：固定角色、固定场景、固定道具、风格规则和禁止变化项。
- 生成 3-6 格 Panel Plan：每格的前置状态、触发事件、可见动作、结果状态、对白意图、必须保留物和允许变化物。
- 选择 `generation_mode`：
  - `sequential_previous_panel`：强连续故事；第 2 格以后必须引用上一格 selected asset。
  - `parallel_minimal_props`：轻连续快速产出；少角色、少道具、弱背景、问答递进，不证明上一格参考参与。
- 通过 MCP 调用 `create_image_task`、`get_image_task`、`list_image_assets`、`select_image_asset`、`reject_image_asset` 和 `get_asset_delivery_info`。
- 发现漂移时，只对单格生成重试说明，不覆盖旧 task、旧 asset 或其他格 selected 状态。

Story Continuity Agent 不持有 provider key、Admin cookie/session、服务器环境变量读取权限或删除权限。

### Agent ImageFlow

- 保存 `workspace -> project -> campaign -> task -> asset` 的事实源。
- 调 provider 生成图片，保存 original、thumbnail 和 metadata。
- 持久化 `story_context_v1`、`panel_index`、`previous_panel_asset_id`、`resolved_reference_assets`、`provider_reference_participation` 和 continuity warnings。
- 通过 Web Recent Assets / Production View 支持人工审图、select/reject、查看连续性摘要和导出 manifest。
- 通过 delivery URL、metadata URL、manifest 和 NAS/文件系统只读访问完成交付。

平台不负责写小红书正文、自动发布、内容日历、账号运营、漫画编辑器、图层编辑或通用 DAM。

## 对象如何协作

| 对象 | 责任 | 第一版约定 |
| --- | --- | --- |
| `project` | 代表真实萌宠账号或 IP 空间 | 保存角色卡、参考图绑定、prompt recipe、quality/profile 和 provider/model 非敏感默认值。 |
| `campaign` | 代表一次可复盘生产批次 | 例如 `2026_07_pet_story_trial`，用于隔离本轮 story tasks、assets 和 manifest。 |
| `story_context_v1` | 每个任务携带的故事快照 | 放在 `metadata_json.story_context_v1`；包含 Story Bible、完整 Panel Plan、reference_bindings、resolved_reference_assets 和 continuity_policy。 |
| `panel` | 业务上的一格故事 | 使用 `panel_index` 排序，`scene_id` 作为业务标识；每格创建独立 task。 |
| `asset` | 平台生成或复用的图片资产 | 候选图保留 `asset_id`、status、provider/model、metadata、thumbnail/original/delivery；selected 表示本格推荐图。 |
| `manifest` | 交付和复盘清单 | 输出 selected panels 的 story/scene/panel/asset/status/delivery/metadata/target_path 和 continuity 摘要；不输出本地绝对路径或 secret。 |

## 必须生产步骤

### 1. 准备 Project / Campaign

执行前确认：

- 一个真实 project，例如萌宠账号或具体 IP。
- 一个 campaign，用于本次 3-6 格故事试用。
- 至少 2 个角色的主图或参考图 asset，或明确标记为文字约束风险。
- 至少 1 个固定环境、风格或核心道具参考。
- 一个 prompt recipe 和 quality/profile 摘要。
- provider/model 和低并发策略已确认；真实 provider 执行前必须确认费用。

只记录 asset_id、role、用途和非敏感配置摘要。不读取、打印或写入 `.env`、provider key、project key、Basic/Auth 密码、Admin cookie/session。

### 2. 产出 Story Bible 和 Panel Plan

Story Continuity Agent 先产出 3-6 格计划，不调用 provider。

每格最少包含：

- `panel_index`
- `scene_id`
- `narrative_role`
- `previous_state`
- `trigger_event`
- `visible_action`
- `resulting_state`
- `dialogue`
- `dialogue_intent`
- `must_keep_props`
- `allowed_changes`
- `reference_bindings`
- `target_path`

示例见 `examples/mcp/pet-story-production-plan.json`。

### 3. 通过 MCP 顺序生成

强连续默认使用 `sequential_previous_panel`：

1. 创建 panel 1 task，`requested_count` 建议不超过 2，`selection_mode=manual_optional`，`review_required=true`。
2. 等待任务完成后，人工或 agent 调 `select_image_asset` 选择 panel 1。
3. 创建 panel 2 task，把 panel 1 selected asset 写入 `reference_bindings.previous_panel_reference`。
4. 重复到 panel 3-6。
5. 任一 panel 漂移时，只对该 panel 创建 regenerate task，并记录 `regenerated_from_task_id`、`regenerate_no` 和修正说明。

真实 provider 试用建议：

- `provider_concurrency=1`。
- 每格最多 2 个候选。
- 3-6 格总图量上限 12，包含受控 regenerate。
- 不做 benchmark。
- 不默认加字；caption/edit lineage 后置。

### 4. Web 审图和交付

人工在 Web Recent Assets / Production View 里做：

- 按 story/scene/panel 查看候选图。
- 查看 panel_index、dialogue、previous_panel_asset_id、resolved reference count、provider_reference_participation 和 warnings。
- 对每格执行 select/reject。
- 导出 selected manifest，或对单个 asset 获取 delivery info。

Web 不写 Story Bible，不改 Panel Plan，不承担主创作。Web 里发现的长 ID、字段噪音、闪烁、按钮反馈弱或下拉摩擦，应记录为 Web 审图体验问题。

### 5. 交付 Manifest / NAS

selected manifest 至少应能复盘：

- `project_id`、`campaign_id`
- `story_id`、`story_revision`、`story_plan_hash`
- `panel_index`、`scene_id`
- `asset_id`、`status`
- `download_url`、`thumbnail_url`、`metadata_url`
- `target_path`
- `previous_panel_asset_id`
- `resolved_reference_assets` 摘要
- `provider_reference_participation`
- regenerate lineage 摘要

NAS / WebDAV / SMB / Finder 只作为只读浏览、复制和备份路径。不要手动移动、重命名或删除平台文件；DB、metadata 和 manifest 继续作为状态与追踪事实源。

## 可选验收记录

以下证据有帮助，但不应被写成产品需求：

- MCP `tools/list` 安全工具列表。
- mock 3 格数据链路 smoke。
- 1 图真实 provider canary。
- delivery URL HEAD/curl spot check。
- Web 截图或操作备注。

mock 只能证明 metadata、状态、manifest 和 UI 链路，不能证明视觉连续性。真实 provider canary 必须记录费用确认、provider/model、图量、停止条件和非敏感 task/asset id。

## 失败分类

完成一次试用后，把问题分成：

- 平台资产模型：task、asset、metadata、manifest、delivery、regenerate lineage 断裂。
- Story Continuity Agent：故事因果弱、reference choices 错误、重试说明不可执行。
- Provider 能力：参考图不参与、文字错误、角色漂移、限流、timeout、provider_error。
- Web 审图体验：找不到 story、长 ID 噪音、select/reject 反馈弱、闪烁、字段过载。
- 部署/NAS 运维：HTTPS 同源、delivery 权限、备份恢复、只读访问和路径可用性。

进入后续 backlog 时，分别归入 Story Continuity、Caption Lineage、Settings IA、Safe Delete、provider follow-up 或部署演练；不要复开已完成 CSV。

## 安全边界

- 不运行 provider，除非用户确认费用和目标环境。
- 不连接服务器，除非任务明确要求并确认。
- 不读取、打印或复制 `.env`、key、cookie、session、secret。
- 不新增 Web UI。
- 不新增 MCP destructive tools。
- 不把 mock smoke、provider benchmark 或浏览器截图自动升级成产品需求。
