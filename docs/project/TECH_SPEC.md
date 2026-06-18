# Tech Spec

## Tech Lock

- App type: 本地优先的 Web 控制台 + API + Worker + CLI/MCP 能力平台。
- Runtime/platform: Docker Compose 本地运行和自托管部署优先。
- Frontend: React + TypeScript，进入实现阶段后再选择具体构建工具。
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
- 第一条 vertical slice 先使用 mock provider，不消耗额度。
- 开发环境存储根目录默认使用项目内 `./storage`。
- 部署环境将 `./storage` 映射到容器内 `/data/agent-imageflow`。
- 第一入口优先实现 REST API + CLI smoke test。
- MCP server 和 Web UI 在核心闭环通过后接入。
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
- API 负责创建任务、查询状态、审核资产、返回 metadata，并保持业务隔离上下文。
- Worker 负责消费任务、调用 provider、下载/保存图片、写入 asset_version。
- MCP server 复用 API/domain 能力，对 Codex/Claude 暴露结构化 tools。
- CLI 用于本地 smoke test、批处理和 CI。
- File service 负责原图、缩略图、版本文件和 metadata 的指定获取。
- 第一版保持模块化单体和少量进程，不做微服务。

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

review_event:
- id
- asset_id
- version_id
- action
- reviewer
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
- `POST /api/assets/{id}/approve`
  - Output: updated asset status
- `POST /api/assets/{id}/reject`
  - Output: updated asset status
- `GET /api/assets/{id}`
  - Output: asset metadata and versions
- `GET /api/assets/{id}/original`
  - Output: original image file
- `GET /api/assets/{id}/thumbnail`
  - Output: thumbnail image file

### MCP tool draft

- `create_image_task`
- `get_image_task`
- `list_image_assets`
- `approve_image_asset`
- `reject_image_asset`
- `get_asset_delivery_info`

### CLI draft

- `vag task create --file task.json`
- `vag task get <task_id>`
- `vag asset approve <asset_id>`
- `vag asset list`
- `vag asset file <asset_id> --kind original`
- `vag asset file <asset_id> --kind thumbnail`

## File Structure

```text
agent-imageflow/
  AGENTS.md
  README.md
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

实现阶段再加入 `apps/`、`cmd/`、`internal/` 或其他代码目录。

## Test / Verification Strategy

- 产品规格阶段：检查文档是否覆盖目标用户、核心场景、非目标和验收标准。
- MVP 实现阶段：
  - 单元测试覆盖状态机和 provider adapter。
  - API smoke test 覆盖创建任务、查询、审核。
  - 本地文件检查确认图片与 metadata 可追踪。
  - MCP tool 用一个模拟调用验证结构化输出。

## Risks

- 如果只做 provider API wrapper，产品价值会很薄。
- 如果过早做泛平台，会被设计工具、DAM、模型平台夹击。
- 如果第一版 provider 依赖付费 API，验证成本可能偏高。
- 如果没有审核与资产登记，无法区别于普通网页生图工具。
- 如果不先选定一个明确场景，任务 schema 会变得空泛。
