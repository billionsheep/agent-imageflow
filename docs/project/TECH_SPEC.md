# Tech Spec

## Tech Lock

- App type: 本地优先的 Web 控制台 + API + Worker + CLI/MCP 能力平台。
- Runtime/platform: Docker Compose 本地运行和自托管部署优先。
- Frontend: 基于 `/Users/moon/Workspace/tools/gpt_image_playground` 二开，使用 Vite + React + TypeScript + Tailwind + Zustand。
- Backend: Go。API、Worker、CLI、MCP 共享 domain code，优先支持单二进制或少量进程交付。
- Persistence: PostgreSQL。
- Queue/cache: Redis。
- Storage: 本地文件系统优先，后续扩展 MinIO/S3。
- External APIs: 第一版先用 mock provider 打通闭环，再接一个云端生图 API provider；MVP 不考虑本地 GPU 或 ComfyUI。
- Deployment: MVP 和最终自托管版本均以 Docker Compose 为默认部署方式；公开 SaaS 版再考虑 Kubernetes 或托管云部署。

## Implementation Lock

第一阶段实施目标已经锁定：

- 不接本地 GPU，不依赖 ComfyUI。
- 生图能力通过外部 API provider 提供。
- 当前第一条实现 slice 改为导入并二开 `gpt_image_playground` 前端底座，避免从低保真 Web 重新造轮子。
- 前端先保留参考项目已有的 Base URL、API Key、多 provider、参考图、遮罩和 Agent 模式能力。
- 服务端资产登记、轻量选优/状态标记和交付模型已通过 mock provider 跑通；MCP stdio、OpenAI-compatible provider adapter、Web 托管模式和项目级 quality profile 复用已接入。
- 多候选 best-of 第三版已接入：`selection_mode=auto` / `best_of` 时，Worker 会自动 selected 一张候选；任务和项目级 quality profile 可通过 `best_of_config` 指定 scorer，并可通过 `auto_reject_non_selected` 自动 rejected 未入选候选；当前支持 `local_metadata_v1` 与 `http_judge_v1`，外部 judge 失败时回退到本地启发式。
- Web/MCP/REST 高级输入第一版已接入：reference image、mask/edit descriptor 和 generation config 会进入服务端任务与资产参数快照；当前 scope 下的 `input-files` 上传/取回、远程 URL 物化、同项目 `asset_id` 复用与 OpenAI-compatible `/images/edits`、fal storage upload + queue `/edit` 已打通。
- 本地 repair/reconcile smoke 已接入：`vag repair scan/requeue/verify-asset` 可扫描可恢复任务、重入队 `enqueue_failed` 任务并校验资产文件一致性。
- 项目级 API key / Basic Auth / 多 key 策略已接入：REST 支持 access-config 管理、实例级 Basic Auth 和项目级 `X-API-Key` / Bearer；Web managed mode 与 CLI 均已支持透传鉴权。
- HTTP API 基础限流已接入：复用 Redis 做固定窗口计数，支持实例级 / project 级阈值、`429` + `Retry-After`，并在限流后端异常时 fail-open。
- HTTP / API 第一版结构化审计日志已接入：`api` 进程会把 `/api/*` 请求写入 `STORAGE_ROOT/audit/http-api/YYYY-MM-DD.jsonl`，并通过 `vag audit list` 提供本地查询。
- Web scope selector / quick create 第一版已接入：REST 可列出/创建 workspace、project、campaign；设置页可同步和快速新建 scope。
- 独立 Web scope 管理第二版已接入：REST 可 rename/archive/delete workspace、project、campaign；Web 顶栏和设置页可进入独立管理 modal，archived scope 会在 selector 中被过滤。
- 开发环境存储根目录默认使用项目内 `./storage`。
- 部署环境将 `./storage` 映射到容器内 `/data/agent-imageflow`。
- 第一入口先提供 Web 生图工作台；REST API、CLI、MCP 作为服务端能力逐步接入。
- 当前 REST API、CLI、MCP stdio、OpenAI-compatible / fal provider 本地 mock smoke、Web managed mode 和 quality profile 本地 smoke 已接入。
- 当前 README / Docker Compose / Runbook 已补 quickstart、demo、自托管最小暴露面和反向代理/TLS 说明；基础 HTTP 限流、本地审计日志和项目级多 key 策略也已接入，MVP 产品缺口已清零。
- 最终默认部署方式是 Docker Compose。

## Architecture

详细架构见 `docs/project/ARCHITECTURE.md`。技术规格只保留摘要。

```text
MCP / REST API / CLI / Web UI
        |
        v
Application Core
Workspace / Project / Campaign / ImageTask / Asset
        |
        v
Queue + Worker
        |
        v
Provider Adapter
Mock / fal.ai / Replicate / OpenAI-compatible / OpenAI Images
        |
        v
Asset Processing
保存原图 / 生成缩略图 / 计算 hash / 写 metadata
        |
        v
Storage + Delivery
本地大磁盘 / MinIO-S3 / 原图获取 / 缩略图获取 / JSON metadata
```

核心思路：

- 多入口共用一套 application core。
- API 负责创建任务、查询状态、标记候选资产、返回 metadata，并保持业务隔离上下文。
- Worker 负责消费任务、调用 provider、下载/保存图片、写入 asset_version。
- MCP server 复用 API/domain 能力，对 Codex/Claude 暴露结构化 tools。
- CLI 用于本地 smoke test、批处理和 CI。
- File service 负责原图、缩略图、版本文件和 metadata 的指定获取。
- 第一版保持模块化单体和少量进程，不做微服务。

## Web / Server Convergence

当前 `web/` 和 Go 服务端不是最终并行的两套产品，而是过渡状态：

- `web/` 来自 `GPT Image Playground`，已具备成熟浏览器生图交互和真实 provider 调用经验。
- Go 服务端是 Agent ImageFlow 的正式事实源，负责 `ImageTask`、队列、资产登记、轻量选优、版本、文件和交付。
- MCP、REST、CLI 和 Web 托管模式都进入同一个服务端 application core。
- 原 Web 的 provider 直连能力可以短期保留为 playground / legacy mode，但正式资产生产应逐步改为服务端 Worker 调 provider。
- Web 当前已可通过服务端 API 创建任务、轮询状态、展示候选资产、select/reject；浏览器 IndexedDB / data URL 路径保留为 legacy playground mode，不作为正式资产流的长期事实源。

Web managed mode 当前边界：

- 设置页配置 `imageflowManagedMode`、API URL、project API key、Basic 用户名/密码、workspace、project、campaign 和 provider。
- 设置页会尝试从服务端同步已有 scope，并通过下拉选择当前 workspace / project / campaign；仍保留手填 ID 作为兜底。
- 设置页可直接快速新建 workspace / project / campaign，创建成功后自动切换。
- Web 顶栏新增独立 scope 管理入口；可以查看 workspace / project / campaign 层级，执行 rename/archive/delete，并把 active campaign 设为当前托管 scope。
- 托管模式提交 prompt 后创建服务端 `ImageTask`，任务卡和详情页展示服务端 `Asset` thumbnail/original。
- 详情页可 select/reject 当前候选资产，并打开 original / metadata URL。
- Web 托管任务默认可通过 `use_project_quality_profile` 复用服务端项目级 prompt template / style preset / reference 参数 / generation config / `best_of_config`。
- Web 托管任务默认传 `selection_mode=auto`，多候选任务完成后会自动推荐一张 selected 资产。
- Web 托管任务在存在 reference image / mask 时，会先把输入文件上传到当前 scope 的 `input-files`，再提交带 `input_file_id` 的服务端任务。
- 服务端会在 `structured_input_json` 中保留公开 descriptor，以及仅供 Worker/provider 使用的 `resolved_input_files`。
- archived workspace / project / campaign 当前只作为组织管理状态，不阻断历史任务/资产读取；设置页 selector 默认不再优先选择 archived 项。
- 当前 `openai-compatible` 和 `fal` provider 在存在已解析输入文件时都会消费统一的 `resolved_input_files`；前者走 `/images/edits` multipart，后者走 fal storage upload + queue `/edit`。这些已解析输入现已支持 scope `input-files`、匿名 `http/https` 远程 URL 物化以及同 workspace/project 下 `asset_id` 复用。
- 若 `selection_mode=auto` / `best_of`，服务端会从任务输入或项目级 quality profile 中解析 `best_of_config`；`http_judge_v1` 使用服务端缩略图生成 `data:` URL 发送给外部 judge，失败时回退 `local_metadata_v1`；当 `auto_reject_non_selected=true` 时，服务端会在自动 selected 后把其他候选标记为 rejected，但仍允许后续人工重新 select。

Provider 迁移顺序建议：

1. 服务端 OpenAI-compatible provider adapter。Status: done.
2. 服务端 fal.ai provider adapter。Status: done.
3. 自定义 HTTP provider profile。Status: pending.

迁移真实 provider 前必须确认密钥来源、日志脱敏、失败结构化和最小 smoke test。OpenAI-compatible adapter 的自动验证使用本地 HTTP mock；真实 smoke 需要用户自行配置 API key。

## Data Model

```text
workspace:
- id
- name
- metadata_json (`archived_at` 用于 workspace 归档状态)
- created_at
- updated_at

project:
- id
- workspace_id
- name
- description
- style_preset
- metadata_json (`quality_profile` 保存项目级质量复用配置，`access_config` 保存项目级 API key 兼容视图与 `api_keys` 列表，`archived_at` 用于 project 归档状态)
- created_at
- updated_at

campaign:
- id
- workspace_id
- project_id
- name
- description
- metadata_json (`archived_at` 用于 campaign 归档状态)
- created_at
- updated_at

generation_task:
- id
- workspace_id
- project_id
- campaign_id
- title
- purpose
- prompt
- negative_prompt
- style_preset
- aspect_ratio
- output_format
- structured_input_json
- provider
- selection_mode
- status
- requested_count
- created_by
- created_at
- updated_at
- error_message

`selection_mode` 当前保存在 `structured_input_json` 中，查询时由服务端反填；默认 `manual_optional`，传 `auto` 或 `best_of` 时 Worker 生成后会自动 selected 一张候选。

`structured_input_json` 当前会保存有效质量配置快照和高级输入 descriptor，包括 `prompt_template`、`template_variables`、`reference_images`、`mask_image`、`generation_config`、`best_of_config`、`use_project_quality_profile`、`selection_mode`、`metadata_json.quality_profile_snapshot`，以及仅供 Worker/provider 使用的 `resolved_input_files`。当输入来自远程 URL 时，快照会同时保留原始 `url` 与服务端生成的 `input_file_id`；当输入来自资产复用时，快照会保留原始 `asset_id`。quality snapshot 中也会保留项目级 `best_of_config`，保证 REST/MCP/Web 托管任务复用时不丢 scorer 配置；其中 `best_of_config.auto_reject_non_selected` 可显式开启自动 rejected 未入选候选。

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
- file_path
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

`parameters_json` 当前会保留 provider 基础参数，并从 `structured_input_json` 快照写入 `reference_images`、`mask_image` 和 `generation_config`，便于后续 provider adapter 或交付系统追踪高级输入。远程 URL 和资产复用来源也会以 descriptor 形式保留在参数快照中。

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

## Interfaces

### REST draft

- Auth: 可选实例级 Basic Auth；project 级资源在启用后要求 `X-API-Key` 或 Bearer。
- Rate limit: 当配置 `RATE_LIMIT_*` 阈值后，HTTP API 可能返回 `429 Too Many Requests`，并带 `Retry-After` 与结构化错误 JSON。
- `GET /api/workspaces`
  - Output: workspace list
- `POST /api/workspaces`
  - Input: `workspace_id` + `name`
  - Output: created workspace
- `PATCH /api/workspaces/{workspace_id}`
  - Input: `name` and/or `archived`
  - Output: updated workspace
- `DELETE /api/workspaces/{workspace_id}`
  - Output: `204 No Content` when workspace is empty
- `GET /api/workspaces/{workspace_id}/projects`
  - Output: project list under workspace
- `POST /api/workspaces/{workspace_id}/projects`
  - Input: `project_id` + `name` + optional `description`
  - Output: created project
- `PATCH /api/workspaces/{workspace_id}/projects/{project_id}`
  - Input: `name` and/or `archived`
  - Output: updated project
- `DELETE /api/workspaces/{workspace_id}/projects/{project_id}`
  - Output: `204 No Content` when project is empty
- `GET /api/workspaces/{workspace_id}/projects/{project_id}/campaigns`
  - Output: campaign list under project
- `POST /api/workspaces/{workspace_id}/projects/{project_id}/campaigns`
  - Input: `campaign_id` + `name` + optional `description`
  - Output: created campaign
- `PATCH /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}`
  - Input: `name` and/or `archived`
  - Output: updated campaign
- `DELETE /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}`
  - Output: `204 No Content` when campaign has no task / asset and the local scope input-files directory can be safely cleaned up
- `POST /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/tasks`
  - Input: structured image task
  - Output: `task_id`
- `POST /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/input-files`
  - Input: multipart `file` + optional `kind=reference|mask`
  - Output: uploaded input file metadata with `input_file_id`
- `GET /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/input-files/{input_file_id}`
  - Output: input file metadata
- `GET /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/input-files/{input_file_id}/content`
  - Output: uploaded input image bytes
- `GET /api/workspaces/{workspace_id}/projects/{project_id}/quality-profile`
  - Output: project-level prompt template / style preset / reference image parameters / generation config
- `POST /api/workspaces/{workspace_id}/projects/{project_id}/quality-profile`
  - Input: project-level quality profile
  - Output: saved quality profile
- `GET /api/workspaces/{workspace_id}/projects/{project_id}/access-config`
  - Output: project-level API key enabled/name/preview compatibility view plus `api_keys` list
- `POST /api/workspaces/{workspace_id}/projects/{project_id}/access-config`
  - Input: legacy single-key `api_key_enabled` / `api_key_name` / `api_key` or multi-key action payload (`action`, `api_key_id`, `api_key_enabled`, `api_key_name`, `api_key`)
  - Output: saved project access config without plaintext key
- `GET /api/tasks/{id}`
  - Output: task status and generated assets
- `POST /api/assets/{id}/select`
  - Output: updated asset status
- `POST /api/assets/{id}/reject`
  - Output: updated asset status
- `GET /api/assets/{id}`
  - Output: asset metadata and versions
- `GET /api/assets/{id}/original`
  - Output: original image file
- `GET /api/assets/{id}/thumbnail`
  - Output: thumbnail image file

当前实现中的 `/approve` 与 `vag asset approve` 保留为兼容入口，产品语义上等价于 `select`；后续新增 MCP 和 Web managed mode 时优先暴露 `select` 命名。

### MCP tool draft

- `create_image_task`
- `get_image_task`
- `list_image_assets`
- `select_image_asset`
- `reject_image_asset`
- `get_asset_delivery_info`

### CLI draft

- `vag task create --file task.json`
- `vag task get <task_id>`
- `vag project access get`
- `vag project access set --enabled=true --key <api_key>`
- `vag project access add-key --name rollout --key <api_key>`
- `vag project access update-key --id <api_key_id> --enabled=false`
- `vag project access delete-key --id <api_key_id>`
- `vag asset select <asset_id>`
- `vag asset list`
- `vag asset file <asset_id> --kind original`
- `vag asset file <asset_id> --kind thumbnail`
- `vag repair scan`
- `vag repair requeue <task_id>`
- `vag repair verify-asset <asset_id>`

## File Structure

```text
agent-imageflow/
  AGENTS.md
  README.md
  Dockerfile
  docker-compose.yml
  go.mod
  cmd/
    api/
    mcp/
    worker/
    vag/
  internal/
    app/
    config/
    db/
    domain/
    httpapi/
    mcp/
    provider/
    queue/
    storage/
    store/
  examples/
    tasks/
  web/
    src/
    package.json
  docs/
    project/
      PRODUCT_SPEC.md
      INPUT_OUTPUT_SPEC.md
      PROJECT_PLAN.md
      TECH_SPEC.md
      TASKS.md
      DECISIONS.md
      CHECKPOINTS.md
      RUNBOOK.md
      stories/
```

服务端当前已经加入 `cmd/`、`internal/`、`Dockerfile`、`docker-compose.yml`、MCP stdio 入口、Web 托管模式支撑、OpenAI-compatible / fal provider adapter、基础限流、本地审计日志和项目级多 key 策略；当前 MVP 产品能力已闭环，剩余 follow-up 主要转向本地开发环境清理。

## Test / Verification Strategy

- 产品规格阶段：检查文档是否覆盖目标用户、核心场景、非目标和验收标准。
- MVP 实现阶段：
  - 单元测试覆盖状态机和 provider adapter。
  - API smoke test 覆盖创建任务、查询、选优/状态标记。
  - 本地文件检查确认图片与 metadata 可追踪。
  - MCP tool 用单元测试和真实 stdio smoke 验证结构化输出。

## Risks

- 如果只做 provider API wrapper，产品价值会很薄。
- 如果过早做泛平台，会被设计工具、DAM、模型平台夹击。
- 如果第一版 provider 依赖付费 API，验证成本可能偏高。
- 如果没有资产登记、稳定交付和候选图选优状态，无法区别于普通网页生图工具。
- 如果不先选定一个明确场景，任务 schema 会变得空泛。
