# Story: 002 - Server Asset Loop

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

在当前 Web 底座之外，打通 Agent ImageFlow 第一条服务端资产闭环：调用方可以创建结构化图片任务，服务端用 mock provider 生成候选图，保存原图、缩略图和 metadata，登记资产，并通过兼容状态接口标记推荐/拒绝后返回稳定交付信息。

Note: 该 story 已完成时接口命名为 `draft/approved` 和 `approve/reject`。当前产品计划已将强人工审核降级为轻量选优/状态标记，后续新 story 应优先使用 `generated/selected` 和 `select/reject` 语义。

## Source Context

- Product spec: `docs/project/PRODUCT_SPEC.md`
- Project plan slice: `docs/project/PROJECT_PLAN.md` 的 “内容账号 campaign 封面图生成闭环”
- Tech spec: `docs/project/TECH_SPEC.md`
- Architecture: `docs/project/ARCHITECTURE.md`
- Flow simulation: `docs/project/IMPLEMENTATION_REVIEW_AND_FLOW_SIMULATION.md`
- Related decisions: `docs/project/DECISIONS.md`

## User Flow

1. 开发者通过 `docker compose up` 启动 API、Worker、PostgreSQL 和 Redis。
2. 调用方用 REST 或 CLI 在默认 workspace / project / campaign 下创建一条 mock 图片任务。
3. Worker 从 Redis 消费任务，mock provider 生成候选图片。
4. 系统按 workspace / project / campaign 保存 original、thumbnail 和 metadata，并写入 `asset` / `asset_version`。
5. 调用方查询任务得到 `task_id`、`asset_id` 和候选资产状态。
6. 调用方通过兼容 `approve/reject` 命令标记候选资产。
7. 被标记为推荐的资产返回 original URL、thumbnail URL、metadata URL 和 local path。

## In Scope

- Go API、Worker、CLI 的最小骨架。
- PostgreSQL schema 和启动时迁移。
- Redis 队列与 Worker 消费。
- mock provider 生成本地示例 PNG。
- 本地文件系统保存 original、thumbnail 和 metadata。
- REST 创建任务、查询任务、查询资产、approve/reject、获取原图和缩略图。
- CLI smoke 命令：创建任务、查询任务、approve/reject 资产。
- `docker compose up` 默认启动 API、Worker、PostgreSQL、Redis。
- 为当前 Web 底座保留最小接入边界文档和运行配置，不深改 Web 交互。

## Out of Scope

- 真实云端 provider 凭据和调用。
- ComfyUI、本地 GPU、MinIO/S3、webhook、权限计费。
- MCP server。
- 完整 Web 候选图选优视图。
- 复杂内容日历、自动发布、小红书平台集成。

## Acceptance Criteria

- Given 本地执行 `docker compose up`，when 服务启动完成，then API、Worker、PostgreSQL、Redis 均可用。
- Given 一个结构化图片任务，when 调用 REST 或 CLI 创建任务，then 返回 `task_id` 和 `queued` 状态。
- Given Worker 消费任务完成，when 查询任务，then 返回 `completed` 状态和至少一个 `asset_id`。
- Given 任务完成，when 检查存储目录，then original、thumbnail 和 metadata 按 workspace / project / campaign 隔离落盘。
- Given draft/generated 资产，when 调用 approve/select 或 reject，then 资产状态变更并记录状态事件。
- Given approved/selected 资产，when 查询资产，then 返回 original URL、thumbnail URL、metadata URL 和 local path。
- Given 当前 Web 底座，when 运行现有测试和构建，then 测试与构建仍通过。

## Technical Approach

- 使用 Go 模块化单体，API、Worker、CLI 共享 `internal/` domain、store、queue、storage 和 mock provider。
- 使用 PostgreSQL 保存 workspace、project、campaign、generation_task、task_attempt、asset、asset_version、review_event。当前 `review_event` 是兼容表名，后续语义等价于 `selection_event`。
- 使用 Redis list 作为第一版队列，任务创建后入队，Worker 通过阻塞消费处理。
- API 启动时执行幂等迁移并 seed 默认 workspace/project/campaign。
- mock provider 生成确定性的 PNG；asset processor 写临时文件、生成缩略图、计算 SHA-256、写 metadata、原子移动到正式目录。
- REST 文件获取只接受 `asset_id`，通过数据库查路径后返回文件。
- Web 接入边界先固定为后续调用 REST 的服务端托管模式，不把浏览器 IndexedDB 作为正式事实源。

## Data / Interface Impact

- 新增 `go.mod`、`cmd/api`、`cmd/worker`、`cmd/vag`、`internal/`、`examples/`、`docker-compose.yml`、服务端 `Dockerfile`。
- 新增 REST API：
  - `GET /healthz`
  - `POST /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/tasks`
  - `GET /api/tasks/{task_id}`
  - `GET /api/projects/{project_id}/campaigns/{campaign_id}/assets`
  - `GET /api/assets/{asset_id}`
  - `POST /api/assets/{asset_id}/approve`
  - `POST /api/assets/{asset_id}/reject`
  - `GET /api/assets/{asset_id}/original`
  - `GET /api/assets/{asset_id}/thumbnail`

## Files or Subsystems Likely to Change

- `cmd/`
- `internal/`
- `examples/`
- `docker-compose.yml`
- `Dockerfile`
- `README.md`
- `docs/project/TASKS.md`
- `docs/project/PROJECT_PLAN.md`
- `docs/project/CHECKPOINTS.md`
- `docs/project/RUNBOOK.md`
- `docs/project/DECISIONS.md`

## Verification Plan

```bash
docker compose build
docker compose up
docker compose exec api /app/vag task create --file /app/examples/tasks/sample-image-task.json
docker compose exec api /app/vag task get <task_id>
docker compose exec api /app/vag asset approve <asset_id>
curl http://localhost:8081/api/assets/<asset_id>
npm --prefix web test -- --run
npm --prefix web run build
```

## Assumptions and Risks

- 本机未安装 Go，Go 验证通过 Docker 镜像执行。
- 第一版 mock provider 不依赖外部服务，不产生真实 provider 成本。
- 第一版缩略图可以复用 mock PNG 文件，先验证稳定路径和元数据闭环；后续再增强真实缩略图尺寸处理。
- Docker 首次拉取镜像和 Go modules 需要网络。

## Implementation Log

### 2026-06-18

- Changes: 新增 Go API、Worker、CLI、PostgreSQL schema、Redis queue、本地文件存储、mock provider、Docker Compose、示例任务和 Web 侧服务端 API client 边界。
- Verification: `go test ./...` 通过；`docker compose build` 通过；干净 volume 下 `docker compose up -d` 成功；CLI 创建任务得到 `task_7b61c7463cb993dd78a8`，Worker 完成 3 个 ready `asset_version`；approve `asset_70948a43b0b74e34c81e` 后 asset metadata 返回 original/thumbnail/metadata URL；reject `asset_c652f332b2059c8e2358` 后状态变为 `rejected`；original 和 thumbnail HTTP HEAD 均返回 `200 OK` / `image/png`；`npm --prefix web test -- --run` 17 files / 219 tests passed；`npm --prefix web run build` 通过。
- Remaining gaps: MCP server 已在 slice 003 补齐；OpenAI-compatible provider 已在 slice 004 补齐；Web 还未深度接入服务端托管任务流；mock 缩略图先生成独立 PNG，后续可增强为真实 resize / webp。
