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

如果只想确认 MCP 进程能启动，也可以单独执行：

```bash
docker compose run -T --rm api /app/mcp
```

但该方式不适合作为完整 mock 生图 smoke，因为独立容器不会自动替代已运行的 `worker`。

## 鉴权边界

先记住一句话：MCP `stdio`、HTTP Basic Auth、Project API Key、Web Admin Login 是四条不同边界。

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

### 4. Admin Login

- 只用于 Web 控制台的人类登录态。
- 不给 MCP、CLI、mock 示例使用。

### 5. Provider Key

- 只属于服务端环境变量或受控部署配置。
- 本指南默认不用真实 provider，也不读取、不打印、不写入任何真实 provider key。

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

### `get_image_task`

按 `task_id` 查询任务状态。最常看：

- `status`
- `error`
- `assets`

任务刚创建时通常先看到 `queued` 或 `running`，等 `worker` 完成后再变成 `completed`。

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

- `original_url`
- `thumbnail_url`
- `metadata_url`

如果调用方只需要可交付结果，这通常是最后一步。

## 最小 mock 生图流程

推荐使用以下顺序：

1. `tools/list`
2. `create_image_task`
3. 等几秒后 `get_image_task`
4. `list_image_assets`
5. `get_asset_delivery_info`

其中第 2 步建议直接复用 `examples/mcp/create-pet-scene.json`。

## 最小任务示例

示例文件：`examples/mcp/create-pet-scene.json`

这个示例体现了：

- 固定 scope：`workspace/project/campaign`
- 业务追踪：`source/session_id/batch_id/story_id/scene_id/target_path`
- Project Visual Context 引用：`character_ids`、`prompt_recipe_id`、`use_project_visual_context`
- mock 生成：`provider=mock`

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
- 人工 smoke 手册：`examples/mcp/smoke.md`

## 当前状态

本轮已补齐 guide 和 examples，并完成 JSON parse / 静态校验。

当前仍建议把 MCP mock smoke 作为人工步骤执行一次后，再把该切片标记为 done。
