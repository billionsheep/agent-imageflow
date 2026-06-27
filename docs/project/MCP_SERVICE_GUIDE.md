# MCP Service Guide

本文面向第一次接入 Agent ImageFlow 的 agent、脚本作者和维护者，目标是让调用方在不阅读整套项目文档的前提下，完成本地 MCP 启动、mock 生图任务创建、任务查询、资产查询和 delivery info 获取。

## 适用边界

- 适用入口：本地 `stdio` MCP。
- 默认 provider：`mock`。
- 默认目标：验证图片资产生成闭环，不验证真实 provider。
- 当前 MCP 工具固定为 6 个，不在本指南内扩展 schema 或 destructive tools。

## 启动前提

推荐先启动本地服务栈，再把 MCP client 接到运行中的 `api` 容器：

```bash
docker compose up -d postgres redis api worker
curl -sf http://localhost:8081/healthz
docker compose exec -T api /app/mcp
```

说明：

- `api` 和 `worker` 要同时存在。`create_image_task` 只负责入队，真正生成 mock 资产的是 `worker`。
- `/app/mcp` 通过 stdin/stdout 收发换行分隔 JSON-RPC。
- MCP 进程复用与 API 相同的环境变量、数据库、Redis、默认 scope 和本地存储。
- 示例文件按“一行一条 JSON-RPC”保存；不要把同一条请求格式化成多行后直接 pipe 给 `/app/mcp`。

如果只想确认 MCP 进程能启动，也可以单独执行：

```bash
docker compose run -T --rm api /app/mcp
```

但该方式不适合作为完整 mock 生图 smoke，因为独立容器不会自动替代已运行的 `worker`。

## 鉴权边界

先记住一句话：MCP `stdio`、HTTP Basic Auth、Project API Key、Agent Setup Token、Web Admin Login 是五条不同边界。

### 1. MCP `stdio`

- 本地 agent 直接启动 `/app/mcp`。
- 不走 HTTP，不需要在 tool 参数里传 Basic Auth、cookie 或 Project API Key。
- 权限边界来自本机环境、Docker、数据库、Redis 和默认 scope 配置。

### 2. HTTP Basic Auth

- 只保护 Web/API 的 HTTP 入口。
- 典型场景：浏览器打开 Web、`curl` 调 API、人工访问 delivery URL。
- 不属于 MCP tool 参数，也不应该写进示例任务 JSON。

### 3. Project API Key

- 用于 project 级 REST/CLI/部分 HTTP 调用保护。
- 典型场景：`curl /api/tasks/...`、`curl /api/projects/.../assets`。
- 本地 `stdio` MCP 本身不消费这把 key，但如果你后续要用 HTTP 查补充信息，就可能需要它。

### 4. Agent Setup Token

- 用于 agent/bootstrap 的非 destructive REST 准备路径。
- 服务端通过环境变量 `AGENT_SETUP_TOKEN` 启用；请求头名是 `X-Agent-Setup-Token`。
- 当前只允许：
  - `GET/POST /api/workspaces`
  - `GET/POST /api/workspaces/{workspace_id}/projects`
  - `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/campaigns`
  - `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/visual-context`
- 明确不允许：
  - `POST /api/tasks`
  - asset select/reject/delivery
  - archive/restore
  - cleanup
  - delete
- `vag` 如果检测到环境变量 `AGENT_IMAGEFLOW_SETUP_TOKEN`，会自动转发这个 header。
- 这条路径的目标是让新 agent 能准备 project/campaign/context，而不是替代 project API key 或 Admin session。

### 5. Admin Login

- 只用于 Web 控制台的人类登录态。
- 不给 MCP、CLI、mock 示例使用。

### 6. Provider Key

- 只属于服务端环境变量或受控部署配置。
- 本指南默认不用真实 provider，也不读取、不打印、不写入任何真实 provider key。

## Agent-Friendly Bootstrap

如果新 agent 还没有现成的 `project/campaign/visual context`，不要先去找 Admin cookie，也不要直连 DB。当前推荐路径是：

1. 用 `X-Agent-Setup-Token` 走非 destructive REST bootstrap。
2. 或者在 CLI 环境里设置 `AGENT_IMAGEFLOW_SETUP_TOKEN`，让 `vag` 自动转发 header。
3. bootstrap 完成后，再用现有 MCP 6 工具做任务创建、查任务、列资产、select/reject 和 delivery。

示例文件：

- `examples/tasks/agent-setup-bootstrap-usage.md`

这个边界很重要：setup token 只负责“准备上下文”，不负责“生产任务”或“清理数据”。

## Project Visual Context Diagnostics

Project Visual Context 现在除了 `visual_context` 本体，还会在读取时返回只读 `reference_diagnostics` 摘要，用来提前判断：

- `image_backed`
- `text_constrained`
- `missing_environment_reference`
- `weak_species_lock`

同时会附带角色有图数量、缺图角色、environment reference 数、species drift negative prompt 覆盖、identity signal 和 provider reference participation risk。

当任务使用 project visual context 时：

- task metadata 会保留 `project_visual_context_diagnostics`
- `structured_input_json` 会保留同名摘要
- batch summary / manifest 会在 `visual_context.reference_diagnostics` 中继续透传

因此新 agent 不需要靠真实 provider 先试一次，才能知道当前 project context 是否偏弱。

## Caption Lineage Speaker / Bubble Semantics

Caption edit 现在推荐把派生语义统一写进 `metadata_json.caption_lineage`，而不是散落在 metadata 根节点。第一版支持：

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
- `auto_select_derivative`
- `avoid_covering_subjects`

服务端会做三件事：

- 归一化 `metadata_json.caption_lineage`，兼容旧的根字段输入
- 把归一化结果写入 `structured_input_json.caption_lineage`
- 把气泡/说话人语义追加成 provider 可见的 prompt 约束，并在 provider parameters、batch summary / manifest asset 上继续透传 `caption_lineage`

建议：

- `speaker_character_id` 复用 project visual context 的角色 id
- delivery 变体尽量保留原 `scene_id`，不要为了 caption edit 新开一个 scene
- `bubble_anchor`、`tail_direction` 先使用近似方向语义，例如 `top_right`、`above_speaker`、`toward_left`
- 如果希望 caption 派生图在完成后直接成为最终交付图，可把 `auto_select_derivative=true` 放进 `metadata_json.caption_lineage`；第一版会在 `manual_optional` caption derivative task 中自动选中第一张派生图，但不会改写 base asset 的事实状态

caption derivative delivery 第一版的只读交付语义如下：

- `delivery_role=base_original`：原图事实仍保留，但当前不是最终交付图
- `delivery_role=caption_derivative`：这是 caption 派生候选，但还没有成为最终交付图
- `delivery_role=final_delivery`：这是 `selected_only` manifest 应优先交付的结果，可能是 base selected，也可能是被选中的 caption derivative

因此：

- `selected_only` manifest 面向交付，会优先保留 scene 内的 `final_delivery`
- `all-assets` manifest 面向复盘，会继续同时保留 base + derivative 事实

## Story Panel State Transition Semantics

连续故事任务现在推荐在 `metadata_json.story_context_v1.panel_plan[]` 中补充状态推进字段：

- `emotion_before`
- `emotion_after`
- `pose_change`
- `relationship_shift`
- `must_change`
- `must_not_keep`
- `state_transition_notes`

服务端会做三件事：

- 把这些字段回写到 task metadata，方便 summary / manifest / Web 审图直接读取
- 在创建任务时追加成 provider 可见的 `State transition requirements` prompt 约束
- 让 batch summary / manifest continuity 能同时表达“保留项”和“必须推进项”

这层语义的目标是帮助 agent 少走“上一格形保住了，但表演没推进”的弯路。平台不会替调用方生成剧情，只负责保存、透传和约束提示。

## 当前 6 个工具

### `create_image_task`

创建图片任务并返回 `task_id`。常用字段：

- `workspace_id`、`project_id`、`campaign_id`
- `title`、`purpose`、`prompt`
- `provider`，本指南固定 `mock`
- `requested_count`
- `selection_mode`
- `character_ids`
- `reference_asset_ids`
- `prompt_recipe_id`
- `use_project_visual_context`
- `metadata_json.source`
- `metadata_json.session_id`
- `metadata_json.batch_id`
- `metadata_json.story_id`
- `metadata_json.scene_id`
- `metadata_json.target_path`
- `metadata_json.caption_lineage`

### `get_image_task`

按 `task_id` 查询任务状态。最常看：

- `status`
- `requested_count`
- `delivered_count`
- `partial_success_reason`
- `provider_error_summary`
- `error`
- `assets`

任务刚创建时通常先看到 `queued` 或 `running`，等 `worker` 完成后再变成 `completed`。

如果真实 provider 少回候选，第一版产品语义是：

- `completed`：`delivered_count == requested_count`
- `partially_completed`：`0 < delivered_count < requested_count`
- `failed`：`delivered_count == 0`

`partially_completed` 不会阻断已经生成的 asset 交付；它的作用是提醒调用方“provider 本次少回了一部分候选”，不要误判成平台完全失败。

### `list_image_assets`

按 project/campaign 列资产。当前已支持的常用过滤：

- `project_id`
- `campaign_id`
- `source`
- `session_id`
- `batch_id`
- `status`
- `keyword`
- `limit`

适合按一次 session/batch 回收这批图，不需要自己从整 campaign 里手工筛。

### `select_image_asset`

把候选图标记为产品语义 `selected`。底层兼容状态仍可能是 `approved`，但 MCP 输出统一映射为 `selected`。

### `reject_image_asset`

把候选图标记为 `rejected`。

### `get_asset_delivery_info`

按 `asset_id` 返回交付信息。第一版最重要的是：

- `download_url`：原图交付链接，也就是业务语义里的 original image URL
- `thumbnail_url`
- `metadata_url`

如果调用方只需要可交付结果，这通常是最后一步。

## 单资产可读摘要

REST / Web 的单资产详情现在额外提供只读 `asset_summary`，收敛：

- `story_id` / `scene_id` / `panel_index`
- `dialogue` / `caption_text`
- `derived_from_asset_id` / `derivation_type`
- `previous_panel_asset_id`
- `provider_reference_participation`
- `provider` / `model`
- `asset_status`
- `delivery_role`

建议把它当成 PM / 运营 / 调用方复盘时的第一阅读层，而不是自己去翻深层 `metadata_json`。

边界：

- `asset_summary` 和 `provider_error_summary` 都不应包含 `local_path`、cookie、token、provider key、project key 或任何 secret-like 字段。
- `delivery_role` 是只读交付语义，不是新的 asset status。
- 需要完整 scene 复盘时，用 `batch-summary` / `batch-manifest`；需要看单图是什么、从哪来、是不是最终交付时，看 `asset_summary` 即可。
- 需要做本地 Web 审图 replay 或 NAS/self-host 交付治理时，不要在 MCP 示例里继续猜路径；直接看 `RUNBOOK.md` 的 packaged review / NAS 指南和 `SERVER_DEPLOYMENT_GUIDE.md` 的部署步骤。

## 最小 mock 生图流程

推荐使用以下顺序：

1. `tools/list`
2. `create_image_task`
3. 等几秒后 `get_image_task`
4. `list_image_assets`
5. `get_asset_delivery_info`

其中第 2 步建议直接复用 `examples/mcp/create-pet-scene.json`。该示例默认使用本地 seed scope `ws_default / prj_xhs_anime / cmp_7day_cover`，因此新 agent 不需要先创建 workspace/project/campaign。

## 最小任务示例

示例文件：`examples/mcp/create-pet-scene.json`

这个示例体现了：

- 固定 scope：默认 seed 的 `workspace/project/campaign`
- 业务追踪：`source/session_id/batch_id/story_id/scene_id/target_path`
- mock 生成：`provider=mock`

为保证新 agent 在干净默认环境中可以直接跑通，示例默认不启用 Project Visual Context。业务 project 配好角色卡、参考图和 prompt recipe 后，再在任务参数中追加 `character_ids`、`reference_asset_ids`、`prompt_recipe_id` 和 `use_project_visual_context=true`。

示例不包含：

- 真实 API key
- 真实 provider key
- cookie
- session token

## 如何拿 delivery info

最短路径：

1. 先 `get_image_task`，拿到任务关联的 `asset_id`
2. 对目标 `asset_id` 调 `get_asset_delivery_info`

如果一次任务生成多张候选图：

- 手动选择推荐图时，先调用 `select_image_asset`
- 再对选中的 `asset_id` 调 `get_asset_delivery_info`

如果你按批次工作，也可以先 `list_image_assets`：

- 用 `source + session_id + batch_id` 缩小范围
- 找到目标 `asset_id`
- 再调 `get_asset_delivery_info`

## 常见错误

### `queued` 一直不变

通常表示 `worker` 没启动，或没有成功消费 Redis 队列。先检查：

```bash
docker compose ps
```

### `tools/list` 正常，但创建任务后没有资产

常见原因：

- 只启动了 `api`，没启动 `worker`
- `provider` 写成了未配置的真实 provider
- 当前任务还没从 `queued/running` 进入 `completed`

### delivery URL 能看到，但 Web 里没自动显示

这不是 MCP 错误。MCP、REST、CLI、Web 共享同一服务端资产事实源，但 Web 是否可见还取决于：

- 当前 Web scope
- Admin 登录态
- 是否在对应 project/campaign 下查看

### 想在 MCP 里直接传 Basic Auth / Project API Key

不需要。对本地 `stdio` MCP 来说，这两类凭据不是 tool 参数。若你后续人工 `curl` 某个 HTTP API，再按 HTTP 入口要求补。

### 想把 provider key 写进示例文件

不要这么做。provider key 只能留在本地环境变量、`.env` 或部署系统里。

### 想删除错图、批次或重新开始试用

MCP 第一轮不提供删除 workspace / project / campaign / asset 的工具，也不提供 archive/restore。新 agent 只能通过 `reject_image_asset` 标记错图、通过 `select_image_asset` 确认好图、通过查询和 delivery 工具拿交付结果。

单资产归档/恢复走 Admin Web / REST / CLI 的 `archive/restore` 路径；真正的数据清理或试用重置走 Admin Web / REST / CLI 的受控 cleanup 流程：先 dry-run 预览候选，再用 dry-run token 或明确确认执行。不要把 Admin cookie、cleanup token、provider key 或真实 project key 写进 MCP 配置示例。

若人类在 Web 中删除整个 workspace / project / campaign，平台会按 scope 生命周期做级联删除，包括子级、任务、资产、缩略图、metadata、原图和 selected / approved / published 结果；这仍然是 Admin 受控操作，不是 MCP 能力。

## 配置示例

复制本地配置文件：

- `examples/mcp/agent-imageflow.local.json`

说明：

- 该文件给出一个可复制的 `mcpServers` 结构。
- 其中的 Project API Key、Basic Auth 只是占位符，方便你在需要手动 HTTP 跟查时统一放在本地配置里。
- `/app/mcp` 本身不会读取这些占位符字段作为 tool 参数。

## 建议接入姿势

- 把一次内容生产视为一个 `session_id`
- 把一批 scene 视为一个 `batch_id`
- 每张图都带 `story_id`、`scene_id`
- `metadata_json.source` 固定写调用方名字，例如 `codex`、`claude`、`cursor`、`automation`
- 萌宠故事、封面图、海报等都优先按同一 contract 填 metadata，而不是临时自造字段

## 配套文件

- 配置模板：`examples/mcp/agent-imageflow.local.json`
- 最小 `tools/call` 示例：`examples/mcp/create-pet-scene.json`
- agent bootstrap 示例：`examples/tasks/agent-setup-bootstrap-usage.md`
- 人工 smoke 手册：`examples/mcp/smoke.md`

## 当前状态

本轮已补齐 guide、examples 和 agent-friendly bootstrap contract：服务端新增 `AGENT_SETUP_TOKEN` / `X-Agent-Setup-Token` 非 destructive 准备路径，`vag` 支持 `AGENT_IMAGEFLOW_SETUP_TOKEN` 自动转发，MCP 工具仍固定为 6 个安全工具。

当前仍建议把 MCP mock smoke 作为人工步骤执行一次；但 `V02-MCPH-002` 自身的 Go contract tests 已完成，不需要依赖真实 provider。
