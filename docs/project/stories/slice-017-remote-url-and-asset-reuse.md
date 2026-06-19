# Story: Slice 017 - Remote URL and Asset Reuse

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让服务端托管任务在 edit/mask 场景下不再只依赖先上传本地 `input-files`。调用方可以直接传远程图片 URL，或复用当前项目下已有的 `asset_id` 作为 reference image / mask 输入，并继续走现有 Worker + provider + asset delivery 闭环。

## Source Context

- Product spec: Agent ImageFlow 的核心价值之一是图片资产可追踪、可复用，而不是只返回一次性的浏览器结果。
- Project plan slice: 当前 pending slice 是 `Phase 6.2: Input reuse expansion`，目标是补远程 URL 抓取和已有 `asset_id` 复用。
- Tech spec: 当前 `reference_images` / `mask_image` 已包含 `url`、`asset_id`、`input_file_id`；`openai-compatible` 已能消费 `resolved_input_files`，但目前主要来源仍是 scope 内上传文件。
- Related decisions: slice-014 已确定第一版输入文件索引不入数据库，而是落到 scope 内本地存储；本片应在这个前提下扩展输入来源，而不是新增表或重写 provider 状态机。

## User Flow

1. 调用方创建服务端任务时，为 `reference_images` 或 `mask_image` 传入远程 `url`，或当前项目下已存在的 `asset_id`。
2. 服务端在创建任务时解析这些输入来源：
   - 远程 URL 会由服务端抓取并落到当前 scope 的 `input-files`。
   - `asset_id` 会解析为当前项目内已有资产的原图文件，并转换为 provider 可消费的内部输入引用。
3. 任务入队后，Worker 继续走现有 `openai-compatible` provider 路径。
4. 如果任务包含已解析 reference / mask，provider 走 `/images/edits` multipart；否则继续走普通 `/images/generations`。
5. 任务完成后，生成资产仍进入现有 `Asset / AssetVersion / Delivery` 闭环，任务和资产参数快照里保留输入来源信息。

## In Scope

- 服务端支持 `reference_images[].url` / `mask_image.url` 的远程抓取。
- 服务端支持 `reference_images[].asset_id` / `mask_image.asset_id` 复用当前项目已有资产。
- `CreateTask` 统一把上述来源解析为内部 `resolved_input_files`，供 provider 读取。
- 继续复用现有 `openai-compatible` edit/mask 路径，不新增第二套状态机。
- 补测试和 Docker smoke，证明 remote URL / asset reuse 均能形成可执行 edit 输入。

## Out of Scope

- 不新增数据库表、对象存储索引或远程 URL 缓存表。
- 不补 fal.ai、自定义 HTTP provider 或更多 provider 的输入复用。
- 不做 Web UI 新交互来选择已有 asset；本片先保证 REST/MCP/统一输入结构已可用。
- 不做跨 project 的 asset 复用；第一版只允许复用当前 workspace/project 下的已有资产。
- 不做远程 URL 的鉴权、签名下载、重试队列或大规模抓取治理。

## Acceptance Criteria

- Given 调用方在 `reference_images` 里传远程 `url`，when 创建 `provider=openai-compatible` 的任务，then 服务端会先抓取该图片并把它解析为可供 Worker 使用的输入文件。
- Given 调用方在 `reference_images` 或 `mask_image` 里传当前项目下已有的 `asset_id`，when 创建任务，then 服务端会复用该资产原图作为输入，而不是要求再次上传文件。
- Given 任务已经解析出 reference image 和可选 mask，when Worker 执行 `openai-compatible`，then provider 请求继续走 `/images/edits` multipart，并完成现有 asset 闭环。
- Given 传入的 `asset_id` 不属于当前 workspace/project，when 创建任务，then API 会在入队前返回明确错误，不允许跨项目复用。
- Given 远程 URL 无法下载、超出限制或不是图片，when 创建任务，then API 会在入队前报错，不生成脏任务。

## Technical Approach

- 在 `internal/app` 的输入解析阶段扩展 `resolveTaskInputFiles`：
  - `input_file_id` 保持原逻辑。
  - `url` 通过服务端 HTTP client 抓取，再复用现有 `storage.StoreTaskInputFile(...)` 落到当前 scope。
  - `asset_id` 通过现有 asset 查询拿到原图路径，并校验其归属必须与当前 workspace/project 一致。
- 远程 URL 解析成功后，把生成的 `input_file_id` 写回请求 descriptor，并继续生成 `resolved_input_files`。
- `asset_id` 解析成功后，在任务 snapshot 中保留原始 `asset_id`，同时把内部文件路径写入 `resolved_input_files`。
- `openai-compatible` provider 继续只读取 `resolved_input_files`，不直接处理 remote URL 或 asset 查询。

## Data / Interface Impact

- 不新增新的外部 REST/MCP 字段；复用当前已存在的 `reference_images[].url`、`reference_images[].asset_id`、`mask_image.url`、`mask_image.asset_id`。
- `structured_input_json` 中的 `resolved_input_files` 将覆盖三类来源：上传文件、远程 URL、资产复用。
- 对于远程 URL，任务 snapshot 会同时保留原始 `url` 和服务端生成的 `input_file_id`；对于资产复用，任务 snapshot 会保留 `asset_id`。

## Files or Subsystems Likely to Change

- `internal/app/input_files.go`
- `internal/app/service.go`
- `internal/domain/types.go`（如需补辅助字段或约束说明）
- `internal/provider/openai_compatible_test.go`
- `internal/httpapi/server.go`（如需调整错误映射或说明）
- `docs/project/*`

## Verification Plan

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'
npm --prefix web test -- --run
npm --prefix web run build
docker compose build api worker

# manual smoke:
# 1. 启动 api/worker 和本地 openai-compatible mock server
# 2. 先创建一个基础 mock/openai-compatible 资产，拿到 asset_id
# 3. 用 remote URL + asset_id 创建新的 openai-compatible edit 任务
# 4. 轮询任务完成，并确认 provider 请求走 /images/edits
```

## Assumptions and Risks

- 远程 URL 第一版只支持匿名 `http/https` 图片下载，不处理签名 URL 刷新、Cookie、鉴权头透传。
- `asset_id` 复用限定在当前 workspace/project，避免绕过项目级隔离。
- 远程 URL 解析发生在创建任务阶段，因此失败会直接体现在 API 请求上，而不是异步延后到 Worker。

## Implementation Log

### 2026-06-18

- Changes:
  - 服务端在 `CreateTask` 的输入解析阶段新增对 `reference_images` / `mask_image` 的 `url`、`asset_id` 和 `input_file_id` 统一处理，保持原有 `resolved_input_files` 消费链路不变。
  - 匿名远程 `http/https` URL 会在创建任务时抓取并物化到当前 scope 的 `input-files`，任务快照同时保留原始 `url` 与生成的 `input_file_id`。
  - 当前 workspace/project 下已有 `asset_id` 可直接复用原图作为 edit 输入，并在创建阶段校验跨项目访问。
  - `openai-compatible` 继续只消费 `resolved_input_files`，因此无需新增第二套 provider 状态机。
  - MCP `create_image_task` schema 已补 `reference_images[].input_file_id` / `mask_image.input_file_id`，并同步到当前服务端输入语义。
- Verification:
  - `docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'`
  - `npm --prefix web test -- --run`
  - `npm --prefix web run build`
  - `docker compose build api worker`
  - Docker smoke: `task_91237d5d15aa7252bed4 -> asset_9ab0aeca719c6e9a2f66`，本地 mock 日志确认 provider 走 `/v1/images/edits`，`image_count=2`。
- Remaining gaps:
  - 更多 provider 还未复用这条 remote URL / asset reuse 输入解析链路。
  - best-of 仍是本地启发式，还不是视觉/LLM 评分。
