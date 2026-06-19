# Story: Slice 014 - Managed Input Upload and Real Edit/Mask Boundary

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让服务端托管模式下的 reference image 和 mask 不再只是 descriptor，而是能先上传到当前 workspace / project / campaign 下，再由真实 provider 在 Worker 里实际读取这些输入文件，完成第一版 edit/mask 闭环。

## Source Context

- Tasks: 当前第一优先待办是“接入真实 provider edit/mask 调用：补服务端可访问的输入图片/遮罩上传或取回路径”。
- Project plan: Web 托管模式和高级输入 descriptor 已完成，但真实 provider edit/mask 和服务端输入图片取回路径仍是主要缺口。
- Tech spec: 当前不希望引入数据库迁移和新的第三方依赖；Go 服务端是正式事实源，Web managed mode 应逐步从浏览器本地图片转向服务端可访问输入。
- Decisions: Web 与服务端应收敛到同一资产核心，不能长期依赖浏览器本地 data URL 作为正式事实源。

## User Flow

1. 用户在 Web 托管模式下带 reference image 或 mask 提交任务。
2. Web 先把输入图片和遮罩上传到当前 workspace / project / campaign 对应的服务端输入文件路径。
3. Web 创建服务端 `ImageTask` 时，不再只传浏览器本地 descriptor，而是传上传后的 `input_file_id`。
4. 服务端在创建任务时解析这些 `input_file_id`，把它们映射为当前 scope 内可访问的输入文件。
5. Worker 调用 `openai-compatible` provider 时，如果任务包含已解析的输入文件，就走 `/images/edits` multipart，并附带图片与 mask。
6. 任务完成后，资产仍进入现有 `Asset / AssetVersion / Delivery` 闭环。

## In Scope

- 新增当前 scope 下的输入文件上传与读取 REST 能力。
- Web managed mode 在提交带 reference/mask 的任务前，先上传输入文件，再把 `input_file_id` 写入任务请求。
- `CreateTask` 在服务端解析上传后的输入文件引用，并把内部解析结果写入任务 `structured_input_json`。
- `openai-compatible` provider 在存在已解析输入文件时走 `/images/edits`，支持带 mask 的 multipart 请求。
- 补最小测试与 smoke，证明真实 edit/mask 边界已经成立。

## Out of Scope

- 不做数据库新表或输入文件持久化索引迁移。
- 不做 MCP/CLI 的文件上传交互界面。
- 不做远程 URL 抓取、外部对象存储拉取或 asset-to-input 自动复用。
- 不做 fal.ai / 自定义 HTTP provider 的 edit/mask 迁移。
- 不做输入文件删除、过期回收和配额管理。

## Acceptance Criteria

- Given 当前 scope 下上传了一张 reference image，when Web 托管模式创建 `provider=openai-compatible` 的任务，then 服务端任务输入中能引用该上传文件，而不是只保留浏览器本地 descriptor。
- Given 当前 scope 下上传了 reference image 和 mask，when Worker 执行 `openai-compatible` 任务，then provider 请求会走 `/images/edits` multipart，并附带输入图片和遮罩。
- Given 上传了输入文件，when 调用 scope 下的输入文件读取接口，then 可以返回元数据并读取文件内容。
- Given 没有输入文件，when 创建普通 `openai-compatible` 任务，then 仍保持现有 `/images/generations` 路径不回归。
- Given 上传的 `input_file_id` 不存在或不属于当前 scope，when 创建任务，then API 会在入队前返回错误，不生成脏任务。

## Technical Approach

- 在 `internal/storage` 中新增 scope 内输入文件落盘与元数据读取能力，继续使用本地文件系统，不新增数据库表。
- 在 `internal/httpapi` 增加 scope 下的 `input-files` 上传、查询和内容读取接口。
- 在 `internal/app.Service` 创建任务前解析 `reference_images[].input_file_id` 和 `mask_image.input_file_id`，把内部可用的本地文件信息写入 `structured_input_json` 的专用字段。
- 在 `internal/provider/openai_compatible.go` 中根据已解析输入文件决定走 `images/generations` 还是 `images/edits`，并复用当前 PNG 规范化和结果登记流程。
- 在 Web managed mode 中复用现有 data URL -> Blob 能力，先上传文件，再创建任务。

## Data / Interface Impact

- `reference_images` 与 `mask_image` 新增可选 `input_file_id`。
- 新增 scope 下的 `input-files` REST 接口。
- `structured_input_json` 新增服务端内部解析后的输入文件字段，用于 Worker/provider 读取；不改变现有 `Asset` 输出结构。

## Files or Subsystems Likely to Change

- `internal/domain/types.go`
- `internal/storage/*`
- `internal/app/*`
- `internal/httpapi/server.go`
- `internal/provider/openai_compatible.go`
- `internal/provider/*test.go`
- `web/src/lib/agentImageflowApi.ts`
- `web/src/lib/agentImageflowApi.test.ts`
- `web/src/store.ts`
- `docs/project/*`

## Verification Plan

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'
npm --prefix web test -- --run
npm --prefix web run build
docker compose build api worker

# manual smoke:
# 1. 启动本地 openai-compatible edit mock server
# 2. 上传 reference / mask 到当前 scope 的 input-files
# 3. 创建 provider=openai-compatible 任务并轮询完成
# 4. 检查 mock server 日志确认走 /images/edits 且带 multipart 文件
```

## Assumptions and Risks

- 第一版输入文件索引不入数据库，改为 scope 内文件系统元数据；这是为了避免当前 slice 引入数据库迁移。
- `openai-compatible` edit/mask 只保证服务端上传文件闭环，不在本片补远程 URL 抓取或更多 provider。
- Web managed mode 会新增一次上传步骤，失败时任务应直接报错而不是创建半成品任务。

## Implementation Log

### 2026-06-18

- Changes:
  - REST 新增当前 scope 下的 `input-files` 上传、metadata 查询和内容读取接口。
  - 服务端 `CreateTask` 会解析 `reference_images[].input_file_id` 与 `mask_image.input_file_id`，并把内部 `resolved_input_files` 写入任务 `structured_input_json`。
  - `openai-compatible` provider 在存在已解析输入文件时会走 `/images/edits` multipart；没有输入文件时继续走 `/images/generations`。
  - Web managed mode 现在会先上传 reference image / mask，再创建带 `input_file_id` 的服务端任务。
- Verification:
  - `docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'`
  - `npm --prefix web test -- --run`
  - `npm --prefix web run build`
  - `docker compose build api worker`
  - Docker smoke：启用 Basic Auth 和 project API key 后，上传 `input-files`、创建 `provider=openai-compatible` 任务；本地 HTTP mock 日志确认 Worker 真实调用 `/images/edits` multipart（`image_count=1`、`mask_count=1`），并成功完成 `task_dd1a410a094e30f06fc5 -> asset_fb9f0bbe559c4c95aa88`。
- Environment note:
  - 本次 Go 格式化与测试继续使用 Docker Go 镜像执行，因为当前宿主环境未直接提供 `go` / `gofmt` 命令。
  - 本地运行 Vite dev server 期间可能出现未跟踪 `.vite/` 目录，这是运行时产物，不属于源码。
- Remaining gaps:
  - 远程 URL 抓取、asset reuse、fal.ai / 自定义 HTTP provider 的 edit/mask，以及输入文件治理仍待后续 slice。
