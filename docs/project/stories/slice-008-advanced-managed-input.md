# Story: Slice 008 - Advanced Managed Input

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让 Web/MCP/REST 创建服务端托管图片任务时，可以把参考图、mask/edit 描述和更多生成参数带入统一 `ImageTask` 输入，先保证任务、资产 metadata 和后续 provider adapter 不丢上下文。

## Source Context

- Product spec: Agent ImageFlow 是 AI 可调用的图片资产生成与管理平台，输入需要支持 prompt、参考图、风格、生成参数和结构化业务上下文。
- Project plan slice: Best-of 后的下一片优先迁移 Web 高级输入到服务端托管任务。
- Tech spec: 多入口必须进入同一 application core，provider adapter 通过结构化任务输入读取参数。
- Related decisions: Web 与服务端不长期并行；质量优先通过 prompt/template/reference/generation config 和选优策略保证。

## User Flow

1. 用户在 Web 托管模式下输入 prompt，并可附加参考图或 mask/edit 草稿。
2. Web 创建服务端 `ImageTask`，把输入图 ID、来源、MIME、角色和 mask target 作为 descriptor 提交。
3. Worker 使用当前 provider 生成候选资产，资产 `parameters_json` 保留 reference/mask/generation config 快照。
4. 后续真实 provider edit/mask adapter 可以基于这些 descriptor 补充取图和 provider-specific 请求。

## In Scope

- `CreateTaskRequest` 增加 `mask_image` 描述符。
- `reference_images` 增加 `source`、`mime_type`、`width`、`height` 描述字段。
- 服务端将 reference/mask/generation config 写入 `structured_input_json`。
- provider 生成的 asset version `parameters_json` 保留 reference/mask/generation config 快照。
- MCP `create_image_task` 支持传递高级输入描述符。
- Web managed mode 不再拒绝输入图或 mask，而是提交 descriptor 并保留本地任务关联。

## Out of Scope

- 不在本片实现真实 provider 的 image edit/mask HTTP 请求。
- 不把 Web IndexedDB 中的原图或 mask data URL 上传到服务端。
- 不新增文件上传 API、数据库表或对象存储取回协议。
- 不做复杂 Web project/campaign 管理 UI。

## Acceptance Criteria

- Given REST/MCP 创建任务传入 `reference_images` 扩展字段和 `mask_image`，when 任务进入 application core，then 字段被保留到 `structured_input_json`。
- Given Worker 使用 mock provider 生成资产，when 查询 asset delivery info 或 metadata，then `parameters_json` 包含 reference/mask/generation config 快照。
- Given Web managed mode 有输入图或 mask，when 用户提交任务，then 不再提示“只支持纯文本 prompt”，并创建服务端 `ImageTask`。
- Given Web managed mode 创建带输入图的任务，when 本地任务记录保存，then `inputImageIds`、`maskTargetImageId` 和 `maskImageId` 不丢失。
- Given MCP `create_image_task` 传 `mask_image`，when tool call 进入 application core，then `MaskImage` 字段不会丢失。

## Technical Approach

- 在 `domain.CreateTaskRequest` 增加 `MaskImage`，并扩展 `ReferenceImage` 描述字段。
- 在 app 层归一化 reference/mask descriptor，继续使用 `structured_input_json`，不迁移数据库 schema。
- 增加 provider 参数快照 helper，从 `Task.StructuredInputJSON` 提取 reference/mask/generation config 写入 `parameters_json`。
- Web managed mode 只提交 descriptor：本地图片 ID、来源、MIME、角色和 mask target；二进制仍留在 IndexedDB。
- MCP schema 和 args 复用 domain type，确保 REST/MCP/Web 语义一致。

## Data / Interface Impact

- REST/MCP/Web task input 新增 `mask_image`。
- `reference_images` descriptor 可包含 `source`、`mime_type`、`width`、`height`。
- Asset version `parameters_json` 可能新增 `reference_images`、`mask_image`、`generation_config`。
- 不新增数据库表、列或生产环境变量。

## Files or Subsystems Likely to Change

- `internal/domain/types.go`
- `internal/app/service.go`
- `internal/app/quality.go`
- `internal/provider/*`
- `internal/mcp/server.go`
- `web/src/lib/agentImageflowApi.ts`
- `web/src/store.ts`
- `docs/project/*`

## Verification Plan

```bash
go test ./...
npm --prefix web test -- --run
npm --prefix web run build
docker compose config
docker compose build
docker compose up -d postgres redis api worker
# REST smoke: create task with reference_images + mask_image + generation_config, then assert asset parameters_json keeps descriptors.
```

## Assumptions and Risks

- 本片先解决“高级输入不再被 Web managed mode 拒绝、服务端不丢上下文”；真实 edit/mask 调用需要后续文件取回/上传边界。
- Web 本地图片 ID 只在本机 IndexedDB 有意义，服务端本片仅把它作为可追踪 descriptor 保存。
- 如果下一片要做真实 provider edit/mask，需要新增服务端可访问的输入图片存储或上传路径。

## Implementation Log

### 2026-06-18

- Changes: 新增 `mask_image` 任务输入；扩展 `reference_images` descriptor；服务端将 reference/mask/generation config 写入 `structured_input_json`；mock 和 OpenAI-compatible provider 的 asset `parameters_json` 保留高级输入快照；MCP schema 支持 `mask_image`；Web managed mode 不再拒绝输入图或 mask，并保留本地任务输入图/mask 关联。
- Verification: `go test ./...`、`npm --prefix web test -- --run`、`npm --prefix web run build`、`docker compose config`、`docker compose build` 通过；REST smoke 创建带 reference/mask/generation config 的 mock 任务后完成，asset `parameters_json` 中保留 `reference_images[0].id=web_ref_1`、`mask_image.target_image_id=web_ref_1` 和 `generation_config.quality=high`，且 auto selected 数量为 1。
- Remaining gaps: 真实 provider edit/mask 请求、服务端输入图片取回和 Web project/campaign 管理体验仍待后续 slice。
