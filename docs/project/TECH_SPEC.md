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
- 服务端资产登记、轻量选优/状态标记和交付模型已通过 mock provider 跑通；MCP 和 Web 托管模式后续接入。
- 开发环境存储根目录默认使用项目内 `./storage`。
- 部署环境将 `./storage` 映射到容器内 `/data/agent-imageflow`。
- 第一入口先提供 Web 生图工作台；REST API、CLI、MCP 作为服务端能力逐步接入。
- 当前 REST API 和 CLI smoke 已接入；MCP 仍未接入。
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
- MCP、REST、CLI 和未来 Web 托管模式都必须进入同一个服务端 application core。
- 原 Web 的 provider 直连能力可以短期保留为 playground / legacy mode，但正式资产生产应逐步改为服务端 Worker 调 provider。
- Web 后续应通过服务端 API 创建任务、轮询状态、展示候选资产、select/reject，而不是把 IndexedDB / data URL 作为正式资产来源。

Provider 迁移顺序建议：

1. 服务端 OpenAI-compatible provider adapter。
2. 服务端 fal.ai provider adapter。
3. 自定义 HTTP provider profile。

迁移真实 provider 前必须确认密钥来源、日志脱敏、失败结构化和最小 smoke test。

## Data Model

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
- created_at
- updated_at
- error_message

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

- `POST /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/tasks`
  - Input: structured image task
  - Output: `task_id`
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
- `vag asset select <asset_id>`
- `vag asset list`
- `vag asset file <asset_id> --kind original`
- `vag asset file <asset_id> --kind thumbnail`

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
    worker/
    vag/
  internal/
    app/
    config/
    db/
    domain/
    httpapi/
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

服务端当前已经加入 `cmd/`、`internal/`、`Dockerfile` 和 `docker-compose.yml`；后续再补 MCP server、真实 provider adapter 和 Web 托管模式 UI。

## Test / Verification Strategy

- 产品规格阶段：检查文档是否覆盖目标用户、核心场景、非目标和验收标准。
- MVP 实现阶段：
  - 单元测试覆盖状态机和 provider adapter。
  - API smoke test 覆盖创建任务、查询、选优/状态标记。
  - 本地文件检查确认图片与 metadata 可追踪。
  - MCP tool 用一个模拟调用验证结构化输出。

## Risks

- 如果只做 provider API wrapper，产品价值会很薄。
- 如果过早做泛平台，会被设计工具、DAM、模型平台夹击。
- 如果第一版 provider 依赖付费 API，验证成本可能偏高。
- 如果没有资产登记、稳定交付和候选图选优状态，无法区别于普通网页生图工具。
- 如果不先选定一个明确场景，任务 schema 会变得空泛。
