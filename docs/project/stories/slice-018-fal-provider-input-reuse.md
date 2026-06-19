# Story: Slice 018 - fal.ai Provider Input Reuse

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让服务端在 `provider=fal` 时也能复用现有托管输入链路。调用方无需为 `fal.ai` 再发明第二套协议，仍然只要提交统一的 `ImageTask`，就可以通过当前 scope 的 `input-files`、匿名 remote URL 和当前项目 `asset_id` 做 edit/mask，并继续拿到标准 `Asset` 交付结果。

## Source Context

- Product spec: Agent ImageFlow 是能力平台，provider adapter 应接 `fal.ai`、Replicate、OpenAI Images 或其他后端，而不是让不同入口各自实现不同任务流。
- Project plan slice: 当前第一条 pending slice 是 `Phase 6.2: Input reuse expansion`，要求把复用输入能力扩到 OpenAI-compatible 之外的至少一条真实 provider 路径。
- Tech spec: 当前服务端已经统一产出 `resolved_input_files`，OpenAI-compatible 已可消费 `input-files`、remote URL 和 `asset_id`；下一顺位 provider 是 `fal.ai`。
- Related decisions: Web 与服务端最终收敛到同一资产核心；remote URL 物化和 asset reuse 已锁定到统一 `resolved_input_files` 链路；新增 provider 不应再分叉任务状态机或输入协议。

## User Flow

1. 调用方创建 `provider=fal` 的服务端任务，可带或不带 reference image / mask。
2. 如果带输入图，服务端先按既有逻辑把 `input-files`、remote URL、`asset_id` 统一解析到 `resolved_input_files`。
3. Worker 执行 `fal` provider adapter：
   - 无输入图时走 fal queue 文生图 endpoint。
   - 有输入图时先把本地 resolved input 上传到 fal storage，再走 fal queue edit endpoint。
4. Worker 轮询 fal queue 直到完成，下载生成图并继续走现有 asset processor / storage / registry。
5. 调用方仍通过统一的 REST / MCP / CLI / Web 托管接口查询任务、候选资产和交付链接。

## In Scope

- 新增服务端 `provider=fal` adapter。
- 支持 fal 文生图和 edit/mask 两条路径。
- fal edit/mask 复用当前 `resolved_input_files`，覆盖 `input-files`、remote URL、当前项目 `asset_id` 三类来源。
- 补本地 mock 测试和 Docker smoke，证明 fal queue + storage + edit 输入复用都可闭环。

## Out of Scope

- 不实现 fal 多模型路由、智能 endpoint 选择或成本控制。
- 不补 fal Web 浏览器直连恢复逻辑到服务端。
- 不做 Replicate、自定义 HTTP provider 或更多 provider 的并行接入。
- 不扩展 `ImageTask` 输入字段，也不新增第二套 provider 专用任务状态机。

## Acceptance Criteria

- Given 调用方创建 `provider=fal` 的普通文生图任务，when Worker 执行，then 服务端能通过 fal queue 完成生成并登记 asset。
- Given 调用方创建 `provider=fal` 的 edit/mask 任务，且输入来源可能是 `input_file_id`、remote URL 或当前项目 `asset_id`，when Worker 执行，then provider 会消费统一的 `resolved_input_files`，而不是要求额外协议。
- Given fal edit/mask 任务包含已解析输入图，when provider 发请求，then 会先把本地 resolved input 上传到 fal storage，再调用 fal edit endpoint。
- Given fal queue 返回结果 URL，when Worker 完成处理，then 新资产仍进入现有 original / thumbnail / metadata 交付链路。
- Given fal provider 未配置或 fal queue/storage 返回明确错误，when 创建或执行任务，then 服务端会返回可诊断的错误，不静默退回其他 provider。

## Technical Approach

- 在 `internal/provider` 新增 `fal.go`，实现：
  - queue submit / status polling / result 获取
  - resolved input 文件上传到 fal storage
  - generation 与 edit endpoint 映射
  - 结果图片解析与 PNG 规范化
- 在 `internal/config`、`docker-compose.yml` 和 `app.NewService(...)` 中增加 fal 配置注入。
- 继续复用当前 `parseTaskStructuredProviderInput(...)` 和 `resolved_input_files` 结构，不改 `CreateTask` 协议。

## Data / Interface Impact

- 外部 `ImageTask` 输入结构不变，只新增 `provider=fal` 可选值。
- `parameters_json` 会新增 fal 的 provider 基础参数快照，例如 endpoint、request_mode、image_size。
- `asset.provider`、`asset_version.provider` 将出现 `fal`。

## Files or Subsystems Likely to Change

- `internal/provider/fal.go`
- `internal/provider/fal_test.go`
- `internal/provider/provider.go`
- `internal/config/config.go`
- `internal/app/service.go`
- `docker-compose.yml`
- `docs/project/*`

## Verification Plan

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./internal/provider ./internal/app'
docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'
docker compose build api worker

# manual smoke:
# 1. 启动本地 fal queue/storage mock
# 2. 用 Docker Compose 注入 FAL_* 环境变量启动 api/worker
# 3. 先生成一个 seed asset
# 4. 创建 provider=fal 且带 remote URL + asset_id 的 edit 任务
# 5. 轮询任务完成，并确认 mock 收到了 storage upload、queue submit、status、result 请求
```

## Assumptions and Risks

- 第一版 fal adapter 采用标准 queue + rest storage HTTP 协议，不引入 Go SDK 新依赖。
- 输入文件上传先只覆盖当前 MVP 常见图片大小，不实现 multipart upload 优化。
- fal 不同模型的输入 schema 可能不同；第一版先按当前 Web 已使用的 `openai/gpt-image-2` / `/edit` 路径实现保守兼容。

## Implementation Log

### 2026-06-18

- Changes:
  - 新增 `internal/provider/fal.go`，以 queue + rest storage HTTP 协议实现 `provider=fal`，支持文生图与 edit 两条路径。
  - 在 `internal/provider/openai_compatible.go` 抽出通用的 `resolveStructuredEditInput(...)` / `readTaskInputFile(...)`，让 OpenAI-compatible 与 fal 共享同一套 `resolved_input_files` 解析与本地文件读取逻辑。
  - 在 `internal/provider/provider.go`、`internal/config/config.go`、`internal/app/service.go`、`docker-compose.yml` 中接入 fal provider、配置项和默认环境变量。
  - 新增 `internal/provider/fal_test.go`，覆盖 queue result、edit 上传链路和 submit 错误路径。
  - 新增 `examples/tasks/sample-fal-task.json` 作为最小 fal smoke 输入。
- Verification:
  - `docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./internal/provider ./internal/app'`
  - `docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'`
  - `docker compose config`
  - `docker compose build api worker`
  - 本地 Docker smoke：先生成 seed mock 资产 `task_7ebf5ca4611196b806dc -> asset_29e343a269640402ef8d`，再创建 fal edit 任务 `task_0dbae47c6d0459cd8c2c -> asset_96d78f9da6b1fcdb0cca`；mock 日志确认命中 `GET /remote.png`、两次 `POST /rest/storage/upload/initiate`、两次 `PUT /uploads/...`、`POST /queue/openai/gpt-image-2/edit`、`GET /queue/openai/gpt-image-2/edit/requests/fal_req_001/status` 和 `GET /results/fal_req_001.png`。
- Remaining gaps:
  - best-of 仍是 `local_metadata_v1` 本地启发式，尚未升级为可插拔视觉/LLM 打分策略。
  - 自定义 HTTP provider、Replicate 等更多 provider 仍未接入，但本片要求的第二条真实输入复用路径已经完成。
  - 本地 Web 开发会生成未忽略的 `.vite/` 运行态缓存；这不影响本片产品验收，但后续应补 ignore/清理规则。
