# Architecture

本文档是 Agent ImageFlow 第一版最终架构指导文件，由原 `ARCHITECTURE.md` v0.1 与 `ARCHITECTURE_REVIEW.md` 合并而来。`ARCHITECTURE_REVIEW.md` 保留为评审输入，本文件作为后续实现、拆分任务和验收的主准绳。

## Architecture Decision

第一版采用本地优先的模块化单体架构：

```text
多入口 -> Application Core -> Queue / Worker -> Provider Adapter
      -> Asset Processing -> Storage / Delivery
```

核心判断：

- 产品不是网页生图工具，而是图片资产生成、登记、轻量选优、复用和交付能力平台。
- MCP、REST API、CLI、Web UI 都必须进入同一套 application core，不能各自实现业务规则。
- 生图必须异步执行，API 不同步等待 provider 完整生成。
- 生成任务状态、资产选优/使用状态、文件版本状态必须拆开建模。
- PostgreSQL 记录事实、索引、状态和审计；图片文件进入本地文件系统，后续扩展 MinIO/S3。
- 第一版保持模块化单体和少量进程，不拆微服务。
- 第一条 vertical slice 优先打通内容账号 campaign 封面图生成闭环。

## Product Boundary

第一版只服务这条闭环：

```text
结构化 ImageTask
  -> 入队
  -> Worker 调用 provider
  -> 保存原图
  -> 生成缩略图
  -> 写入 asset / asset_version / metadata
  -> 轻量选优或状态标记
  -> 按 asset_id 获取原图、缩略图、metadata 和交付信息
```

做：

- 结构化图片任务。
- 生图 provider 适配。
- 文件落盘和 metadata 登记。
- 任务状态、资产状态、选优/状态事件。
- MCP / REST / CLI / Web UI 多入口。
- 本地文件获取和 JSON metadata 交付。

暂不做：

- 泛用设计编辑器。
- 白板或复杂拖拽排版。
- 模板市场。
- 完整 DAM。
- 自动发布到小红书、CMS 或其他平台。
- 企业级多租户、SSO、计费和复杂权限。
- 多 provider 智能调度。

## High-level Architecture

```text
Codex / Claude / Cursor     CMS / n8n / SaaS      CLI / CI       Web Console
          |                       |                  |                |
          v                       v                  v                v
       MCP Server             REST API          CLI Adapter       Web API
          \                       |                  |                /
           \                      |                  |               /
            v                     v                  v              v
                         Application Core
          Workspace / Project / Campaign / ImageTask / Asset
                                   |
                +------------------+------------------+
                |                                     |
                v                                     v
          PostgreSQL                           Redis Queue
  Task / Asset / Version / Selection               |
                ^                                  v
                |                                Worker
                |                                  |
                |                                  v
                |                           Provider Adapter
                |                        Mock / Cloud API
                |                                  |
                |                                  v
                +------------------------ Asset Processor
                                      save / thumbnail / hash / metadata
                                                  |
                                                  v
                                           File Service
                              local storage first, MinIO/S3 later
```

## Architecture Principles

1. 多入口，一套核心逻辑。
2. 生成、选优、交付三件事分层处理。
3. `ImageTask.status` 只描述生成执行；`Asset.status` 只描述候选图选择、推荐和使用；`AssetVersion.status` 只描述文件是否可用。
4. 图片文件不进数据库，数据库只保存事实记录、索引、状态和路径。
5. Provider、Storage、Delivery 都通过 adapter 扩展，业务核心不依赖具体实现。
6. `workspace_id`、`project_id`、`campaign_id` 从第一天贯穿任务、资产、文件路径、查询和交付。
7. 文件读取只能通过 `asset_id` 或服务端生成的 URL，不能让调用方传任意本地路径。
8. Worker 必须考虑重复消费、重试、超时、部分成功和中途崩溃。
9. 第一版用 mock provider 优先打通闭环，再接一个云端 API provider。
10. 架构升级必须由明确触发信号驱动，不凭感觉提前扩张。

## Runtime Topology

第一版仍是模块化单体，但建议拆成少量进程：

```text
api
  REST API
  Web UI API
  MCP bridge or MCP stdio entry

worker
  queue consumer
  provider call
  file download
  asset processing

cli
  local smoke test
  batch command
  CI helper

postgres
  task / asset / version / selection / delivery facts

redis
  queue
  short locks
  retry delay

storage
  local large disk mounted at /data/agent-imageflow
```

这种形态保证代码仍是一套，但 API 和 Worker 可以独立重启、观察和扩容。

## Layer Responsibilities

### Input Layer

包含：

- MCP Server
- REST API
- CLI Adapter
- Web API

职责：

- 接收输入并归一为 `ImageTask`。
- 从路径、请求体或默认配置补齐 `workspace_id`、`project_id`、`campaign_id`。
- 处理认证、鉴权和参数校验的边界。
- 调用 application service。
- 返回结构化 JSON。

禁止：

- 在入口层直接写 provider 调用逻辑。
- 在入口层直接拼接文件路径读取文件。
- MCP、CLI、Web UI 各自实现不同业务状态机。

统一调用的 application service：

```text
createImageTask()
getImageTask()
getProjectQualityProfile()
updateProjectQualityProfile()
listImageAssets()
selectAsset()
rejectAsset()
getAssetDeliveryInfo()
getAssetFile()
getAssetMetadata()
```

### Application Core

核心对象：

```text
Workspace
Project
Campaign
ImageTask
TaskAttempt
Asset
AssetVersion
SelectionEvent
DeliveryEvent
DeliveryInfo
```

职责：

- 校验业务隔离上下文。
- 创建和查询任务。
- 维护任务状态机。
- 维护资产状态机。
- 写选优/状态事件和交付事件。
- 生成交付信息。
- 屏蔽 provider、queue、storage 的实现细节。

Application core 不应该知道 fal.ai、Replicate、OpenAI-compatible、OpenAI Images、MinIO 或 S3 的具体协议细节。

### Queue / Worker Layer

职责：

- 消费任务。
- 标记任务执行 attempt。
- 调用 provider adapter。
- 处理 provider 超时、限流、部分成功和失败。
- 下载或接收生成文件。
- 调用 asset processor。
- 写入 asset / asset_version / metadata。
- 更新 task 终态。

第一版推荐 Redis queue。不要同步阻塞 API 请求等待图片生成完成。

Worker 必须支持：

- 并发数配置。
- 每个 provider 的并发上限。
- 任务超时。
- 最大重试次数。
- 指数退避。
- 任务锁，避免同一任务被多个 Worker 同时处理。
- `requested_count` 上限，避免批量任务压垮 provider 或磁盘。

### Provider Adapter Layer

统一接口：

```text
ProviderAdapter.generate(task, attempt) -> ProviderResult
```

`ProviderResult` 建议包含：

```text
provider_request_id
status
files[]
error_code
error_message
raw_response_json
cost_json
```

Provider adapter 只负责生图和返回结果，不负责：

- 选优状态。
- 业务隔离。
- 文件最终落盘路径。
- 资产登记。
- 交付信息。

第一版 provider 策略：

- 先用 `mock provider` 跑通全链路。
- 真实 provider 从云端 API provider 中选择一个，例如 fal.ai、Replicate、OpenAI-compatible 或 OpenAI Images。
- MVP 不考虑本地 GPU 或 ComfyUI，ComfyUI 只作为后续可选 provider。
- 真实 provider 接入前，接口和状态机必须已经能处理成功、失败、超时和部分成功。

### Asset Processing Layer

这是 Agent ImageFlow 区别于普通 provider API wrapper 的关键层。

每个生成结果必须完成以下步骤后才可登记为可追踪资产：

1. 写入临时目录。
2. 校验文件存在、类型、大小和基础图片信息。
3. 生成缩略图。
4. 计算 hash。
5. 写 metadata JSON。
6. 原子移动到正式路径。
7. 写入 `asset` 和 `asset_version`。
8. 将资产初始状态设置为 `generated`。

如果任一步失败：

- 不应把半成品标记为 ready。
- 应记录 `error_code` 和 `error_message`。
- 已写入的临时文件进入可清理区域。
- 任务可进入 failed 或 partially_completed。

### Storage / Delivery Layer

第一版使用本地文件系统，路径建议：

```text
/data/agent-imageflow/
  workspaces/{workspace_id}/
    projects/{project_id}/
      campaigns/{campaign_id}/
        originals/{asset_id}/{version}.{ext}
        thumbnails/{asset_id}/{version}.webp
        metadata/{asset_id}/{version}.json
        tmp/
```

数据库记录：

- `file_path`：本地部署内部路径。
- `object_key`：未来对象存储 key。
- `public_url`：未来公开或 CDN URL。
- `hash`：文件内容 hash。
- `mime_type`、`width`、`height`：文件基础信息。

交付接口第一版至少支持：

```text
GET /api/assets/{asset_id}
GET /api/assets/{asset_id}/original
GET /api/assets/{asset_id}/thumbnail
```

后续可扩展：

```text
GET /api/assets/{asset_id}/versions/{version}/original
GET /api/files/{file_id}
signed_url
public_url
cdn_url
export_pack
markdown_snippet
html_snippet
webhook_payload
```

## State Model

### ImageTask.status

`ImageTask.status` 只回答“生成任务执行到哪里了”。

```text
queued
running
completed
partially_completed
failed
canceled
enqueue_failed
```

推荐流转：

```text
queued -> running -> completed
queued -> running -> partially_completed
queued -> running -> failed
queued -> canceled
running -> canceled
queued -> enqueue_failed
enqueue_failed -> queued
```

说明：

- `completed`：请求数量全部生成并登记资产。
- `partially_completed`：部分生成结果成功登记，部分失败。
- `failed`：没有可用资产或不可恢复错误。
- `enqueue_failed`：数据库任务已创建，但入队失败，等待补偿或重入队。
- `selection_pending` 不作为任务状态，它是 UI 可以根据 generated assets 推导出的视图状态。

### Asset.status

`Asset.status` 只回答“这张图片资产在候选、推荐、弃用和交付中的位置”。

```text
generated
selected
rejected
published
deprecated
```

推荐流转：

```text
generated -> selected
generated -> rejected
selected -> published
generated -> deprecated
selected -> deprecated
published -> deprecated
rejected -> deprecated
```

说明：

- Worker 新生成资产默认是 `generated`。
- `selected` 代表人工、调用方或自动策略认为它是推荐候选，不是强制交付闸门。
- 小团队/单体平台第一版不要求每张图都经过人工审核；`generated` 资产可以用于预览和内部交付，调用方可按需要标记 `selected` 或 `rejected`。
- `published` 代表已经被某个外部目标使用或发布。
- `deprecated` 代表不再推荐使用，但记录保留。
- regenerate 应创建新任务、新资产或新版本，不应把旧 rejected 资产直接改写成新图片。
- 当前服务端代码里的 `draft` / `approved` 是第一条 vertical slice 的兼容命名，产品语义上分别映射为 `generated` / `selected`；后续 MCP、Web managed mode 和数据库迁移应优先采用新语义。

### AssetVersion.status

`AssetVersion.status` 只回答“这个版本的文件是否完整可读”。

```text
processing
ready
failed
deleted
```

推荐规则：

- 只有原图、缩略图、metadata 都完成后才进入 `ready`。
- `asset.current_version_id` 只能指向 `ready` 版本。
- 文件缺失或校验失败时，版本不能进入 `ready`。

## Data Model

第一版推荐表：

```text
workspace:
- id
- name
- metadata_json
- created_at
- updated_at

project:
- id
- workspace_id
- name
- description
- style_preset
- metadata_json
- created_at
- updated_at

campaign:
- id
- workspace_id
- project_id
- name
- description
- metadata_json
- created_at
- updated_at

generation_task:
- id
- workspace_id
- project_id
- campaign_id
- idempotency_key
- title
- purpose
- prompt
- negative_prompt
- style_preset
- aspect_ratio
- output_format
- structured_input_json
- provider
- status
- requested_count
- created_by
- trace_id
- created_at
- updated_at
- error_code
- error_message

task_attempt:
- id
- task_id
- attempt_no
- status
- provider
- provider_request_id
- started_at
- finished_at
- latency_ms
- retry_after
- error_code
- error_message
- raw_response_json
- cost_json

asset:
- id
- workspace_id
- project_id
- campaign_id
- task_id
- name
- type
- current_version_id
- status
- created_at
- updated_at

asset_version:
- id
- asset_id
- version
- status
- file_path
- thumbnail_path
- metadata_path
- object_key
- public_url
- mime_type
- width
- height
- hash
- provider
- model
- prompt
- parameters_json
- cost_json
- created_at

selection_event:
- id
- asset_id
- version_id
- action
- actor
- note
- created_at

delivery_event:
- id
- asset_id
- version_id
- target_type
- target_ref
- snippet
- created_at
```

约束建议：

- `generation_task.idempotency_key` 在同一 workspace 或 project 范围内唯一，允许为空。
- `asset_version` 对同一 `asset_id` 的 `version` 唯一。
- 同一 `task_id` 下相同 `hash` 的 ready 文件不重复登记为多个资产，除非业务明确需要保留重复候选。
- 所有任务和资产查询都必须带业务隔离校验。

## Core Flows

### Create Task

```text
Caller -> API/MCP/CLI/Web
  -> validate input
  -> normalize ImageTask
  -> check workspace/project/campaign
  -> apply idempotency_key
  -> insert generation_task(status=queued)
  -> enqueue task_id
  -> return task_id
```

如果数据库写入成功但入队失败：

- 将任务标记为 `enqueue_failed`。
- 返回可解释错误或保留任务等待补偿。
- 后续通过 repair/requeue 命令重新入队。

### Worker Generate

```text
Worker -> consume task_id
  -> acquire task lock
  -> create task_attempt
  -> mark task running
  -> call ProviderAdapter.generate()
  -> process each returned file
  -> write temp original/thumbnail/metadata
  -> atomic move to final path
  -> insert asset + asset_version(status=ready)
  -> mark task completed / partially_completed / failed
```

### Select / Reject Asset

```text
Caller -> select/reject asset_id
  -> load asset and current ready version
  -> check workspace/project/campaign
  -> apply state transition
  -> insert selection_event
  -> return updated asset
```

选择/拒绝操作必须幂等：

- 已 `selected` 再 select 返回当前状态。
- 已 `rejected` 再 reject 返回当前状态。
- 非法状态转换返回明确错误，不静默吞掉。

### Get Delivery Info

```text
Caller -> get_asset_delivery_info(asset_id)
  -> load asset + current_version
  -> check isolation and status
  -> build download_url / thumbnail_url / metadata_url / local_path
  -> optionally insert delivery_event
  -> return structured JSON
```

第一版默认允许 `generated`、`selected`、`published` 资产返回交付信息；`selected` 只是推荐优先级，不是唯一可交付条件。后续如果出现强合规或多人审核场景，再通过项目级策略开启强审核闸门。

## Idempotency and Retry

### API Idempotency

外部调用方可以传 `idempotency_key`。语义：

- 同一业务隔离范围内，相同 `idempotency_key` 和等价输入返回同一个 `task_id`。
- 相同 `idempotency_key` 但输入冲突，应返回冲突错误。
- 未提供 `idempotency_key` 时，系统正常创建新任务。

### Worker Retry

Worker 重试规则：

- Provider 超时、限流、短暂网络错误可以重试。
- 输入校验失败、图片格式不支持、业务隔离缺失不应重试。
- 每次执行写 `task_attempt`。
- 超过最大重试次数后任务进入 `failed` 或 `partially_completed`。
- 重试必须能识别已完成的 ready asset，避免重复登记。

### Duplicate Consumption

Redis queue 和 Worker 重启可能导致重复消费。第一版必须具备：

- `task:{id}:lock` 或数据库行级锁。
- `task_attempt` 记录。
- `asset_version.hash` 去重策略。
- 文件写入临时目录后原子移动。
- select / reject 幂等。当前 `approve / reject` 接口保留为兼容别名。

## Consistency Boundaries

系统同时依赖 PostgreSQL、Redis 和文件系统，因此必须明确一致性边界。

### Database then Queue

创建任务时：

1. 先写 PostgreSQL。
2. 再入 Redis queue。
3. 入队失败则标记 `enqueue_failed` 或记录待补偿。

不建议先入队再写数据库。

### Temp then Final

处理文件时：

1. 写入 `tmp/`。
2. 校验原图、缩略图、metadata。
3. 原子移动到正式目录。
4. 再写入或更新 `asset_version(status=ready)`。

如果数据库写入失败，不应留下不可追踪的正式文件；至少应记录 repair log，后续 reconcile。

### Reconcile

第一版已接入本地 repair/reconcile smoke 命令：

```text
vag repair scan
vag repair requeue <task_id>
vag repair verify-asset <asset_id>
```

检查内容：

- 数据库有 asset_version 但文件缺失。
- 文件存在但没有数据库记录。
- `current_version_id` 指向非 ready 版本。
- `enqueue_failed` 任务是否需要重新入队。

## File Access and Isolation

文件访问规则：

- 外部接口只接受 `asset_id`，不接受任意本地路径。
- 服务端通过数据库查 `workspace_id/project_id/campaign_id/file_path`。
- 读取文件前校验调用方是否有权访问该 workspace/project/campaign。
- 返回文件时使用服务端控制的下载 URL。
- `local_path` 只作为本地部署和 CLI 交付信息，不作为公网 API 的安全边界。
- 后续进入对象存储后，用 `object_key` 和 signed URL 替代本地 path 直出。

文件状态规则：

- 只有 `AssetVersion.status=ready` 的文件可被正式读取。
- 正式交付默认允许 `Asset.status in (generated, selected, published)`。
- 选优视图预览 generated 资产时，应走内部预览权限和相同归属校验。

## Provider Failure Model

Provider adapter 必须把失败结构化，而不是只返回自然语言错误。

常见失败：

- timeout
- rate_limited
- quota_exceeded
- invalid_request
- provider_error
- partial_success
- download_failed
- unsupported_format
- unsafe_content
- unknown

处理规则：

- provider 原始响应进入 `raw_response_json`，注意不要记录密钥。
- provider request id 进入 `provider_request_id`。
- 费用、模型和参数进入 `cost_json` / `parameters_json`。
- 部分成功时，成功文件照常登记资产，失败部分记录到 task attempt。
- provider 生成成功但下载失败，应区分于 provider 生成失败。

## Observability

第一版即使不做完整监控系统，也要在数据结构和日志里保留追踪字段。

关键字段：

- `trace_id`
- `task_id`
- `attempt_id`
- `asset_id`
- `asset_version_id`
- `provider`
- `provider_request_id`
- `status`
- `error_code`
- `error_message`
- `latency_ms`
- `retry_count`
- `requested_count`
- `generated_count`
- `cost_json`

建议日志点：

- task created
- task enqueued
- worker attempt started
- provider request started
- provider result received
- file processing started
- asset version ready
- task completed / partially_completed / failed
- selection event created
- delivery info requested

避免：

- 记录 API key、provider token、用户密钥。
- 高频循环中重复写冗余日志。
- 静默吞掉 provider 或文件系统异常。

## Interface Draft

### REST

```text
POST /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/tasks
GET  /api/tasks/{task_id}
GET  /api/projects/{project_id}/campaigns/{campaign_id}/assets
GET  /api/assets/{asset_id}
POST /api/assets/{asset_id}/select
POST /api/assets/{asset_id}/reject
GET  /api/assets/{asset_id}/original
GET  /api/assets/{asset_id}/thumbnail
```

当前实现中的 `POST /api/assets/{asset_id}/approve` 保留为兼容别名，语义等价于 `select`。

### MCP Tools

```text
create_image_task
get_image_task
list_image_assets
select_image_asset
reject_image_asset
get_asset_delivery_info
```

### CLI

```text
vag task create --file task.json
vag task get <task_id>
vag asset list
vag asset select <asset_id>
vag asset reject <asset_id>
vag asset file <asset_id> --kind original
vag asset file <asset_id> --kind thumbnail
vag repair scan
```

CLI 在第一版可以作为 smoke test 工具，不要求完整交互体验。当前 `vag asset approve` 是兼容命令，后续可补 `select` 别名。

## Configuration

第一版需要的关键配置：

```text
DATABASE_URL
REDIS_URL
STORAGE_ROOT=/data/agent-imageflow
PUBLIC_BASE_URL
DEFAULT_WORKSPACE_ID
DEFAULT_PROVIDER=mock
WORKER_CONCURRENCY=1
PROVIDER_CONCURRENCY=1
TASK_TIMEOUT_SECONDS
MAX_TASK_RETRIES
THUMBNAIL_MAX_WIDTH
THUMBNAIL_MAX_HEIGHT
OPENAI_COMPATIBLE_BASE_URL
OPENAI_COMPATIBLE_API_KEY
OPENAI_COMPATIBLE_MODEL
PROVIDER_TIMEOUT_SECONDS
```

OpenAI-compatible provider 已作为第一版真实 provider adapter 接入。不得提交真实密钥；真实 smoke 需要通过本地环境变量注入。

## First Vertical Slice

第一条 vertical slice：

```text
内容账号 project
  -> 7 天封面图 campaign
  -> 创建 ImageTask
  -> mock provider 生成候选图
  -> 保存 original
  -> 生成 thumbnail
  -> 写 asset metadata
  -> select asset or use generated asset directly
  -> GET original / thumbnail / metadata / delivery info
```

实现顺序：

1. 用 mock provider 跑通 create task -> queue -> worker -> local files -> asset metadata -> select/reject -> delivery。
2. 加入 `ImageTask.status`、`Asset.status`、`AssetVersion.status` 和 `selection_event`。当前实现可继续用 `review_event` 表名作为兼容承载。
3. 加入缩略图、hash、metadata JSON。
4. 加入 `idempotency_key`、task attempt 和 Worker 重试。
5. 增加 MCP stdio server。
6. 接入第一个云端 API provider。
7. 增加 Web 候选图选优视图。
8. 再考虑 MinIO/S3、webhook、多 provider 和成本统计增强。

第一版不要求：

- 多 provider。
- 云部署。
- 复杂权限。
- 自动发布小红书。
- 完整内容日历。
- 设计编辑器。

## Evolution Triggers

架构升级应由明确信号触发：

| 触发信号 | 建议动作 |
|---|---|
| 本地磁盘容量、备份或多机访问成为风险 | 引入 MinIO / S3 |
| 单 provider 失败影响所有任务 | 引入第二 provider 和 provider routing |
| Worker 队列长期积压 | 独立扩容 Worker，增加 provider 并发控制 |
| 文件与数据库不一致频繁出现 | 引入 outbox / repair job / 更严格状态机 |
| 候选图数量或选优量上升 | 增加候选筛选、批量选优、状态视图和自动选优策略 |
| 多业务方接入 | 增加项目级 API key、限流、审计日志 |
| provider 成本不可控 | 增加成本预算、模型路由、任务配额 |
| 任务链路排障困难 | 接入集中日志、metrics 和 tracing |

## Implementation Guardrails

后续实现时必须遵守：

- 先实现 mock provider，优先验证闭环。
- 入口层不能绕过 application service。
- Provider adapter 不能直接写最终业务文件路径。
- Storage adapter 不能跳过 asset ownership 校验。
- API 不能接受任意本地文件路径作为读取输入。
- `asset.current_version_id` 不能指向非 ready 版本。
- Worker 失败必须记录错误，不得静默吞异常。
- 新增真实云端 provider 前必须说明密钥管理、成本风险和最小 smoke test。
- 不因为未来可能性提前创建小说、电商、海报、小红书等大量固定业务表。

## Minimum Acceptance Criteria

架构进入实现前，第一版应满足这些验收标准：

- 给定结构化 `ImageTask`，系统返回 `task_id`。
- Worker 能生成或模拟生成图片。
- 原图、缩略图、metadata 能按 workspace / project / campaign 隔离落盘。
- 数据库中能查到 `asset_id`、`asset_version`、hash、provider、prompt 和状态。
- `generated` 资产可返回稳定交付信息；标记为 `selected` 后可作为推荐交付结果优先返回。
- 重复提交同一 `idempotency_key` 不会创建不可控重复任务。
- Worker 重复消费同一任务不会产生不可控重复资产。
- provider 失败时任务有明确 `error_code` 和 `error_message`。
- 文件获取必须通过 `asset_id` 并校验归属。
- 第一条内容账号 campaign 封面图闭环可以用 mock provider 本地跑通。
