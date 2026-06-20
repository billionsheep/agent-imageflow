# Slice 025: Asset Production Readiness

## Product Goal

让 Agent ImageFlow 从“能生成并显示资产”推进到“能作为日常图片资产生产能力使用”：资产可查可筛、批次可追踪、项目默认 provider 可复用，Web 资产库在资产增多时不会首屏全量渲染。

## User Flow

1. Codex / MCP / REST / CLI / Web 在同一个 project/campaign 下创建图片任务。
2. 调用方在 `metadata_json` 中传入 `source/session_id/batch_id/story_id/scene_id/target_path`。
3. 用户在 Web 服务端资产库按状态、来源、会话、批次或关键词筛选资产。
4. Web 首屏只加载有限资产，图片 lazy loading，用户按需加载更多。
5. 项目可以保存非敏感 provider 默认值，后续任务未显式 provider 时复用项目默认配置。

## In Scope

- REST/CLI asset list query: `limit`、`offset`、`status`、`provider`、`model`、`source`、`session_id`、`batch_id`、`keyword`、`created_from`、`created_to`。
- Web 服务端资产库筛选、加载更多、图片 lazy loading 和 metadata/parameters 摘要。
- `metadata_json` 标准字段归一化，未知字段保留。
- 项目级非敏感 `provider_profile`。
- Codex 批量图片资产生产示例。

## Out Of Scope

- 不保存真实 provider key。
- 不实现公网注册、配额、计费或 SaaS 控制台。
- 不做批量 delete。
- 不做小红书发布、内容日历、脚本读取。
- 不做 Reference Library、Mascot Profile、Prompt Recipe 或 edit lineage。
- 不把项目扩成通用 DAM 或设计平台。

## Acceptance Criteria

- `GET /api/projects/{project_id}/campaigns/{campaign_id}/assets` 保持 array 响应，并支持筛选和 limit 上限。
- Store 查询可按 asset、asset_version 和 `structured_input_json.metadata_json` 过滤。
- Web 资产库筛选不改变当前 scope，刷新和加载更多不会重复显示资产。
- 标准 metadata 字段空值不污染响应，未知字段保留，非法 JSON 兜底为空对象。
- Provider profile 只保存非敏感字段，不保存或回显 secret。
- 示例任务可由 `vag task create --file` 使用。

## Technical Approach

- 继续复用 `project.metadata_json`，新增 `provider_profile` 子对象，不做 schema migration。
- 继续使用 `generation_task.structured_input_json.metadata_json` 承载 source/session/batch，不新增业务表。
- Web list assets 仍消费数组响应，通过 `limit=24` 与 `offset` 做加载更多。
- Provider adapter 只读取 profile 中的 `model` 覆盖值；`base_url` 第一版仅保存为非敏感配置，不改变真实 endpoint/key 策略。

## Data / Interface Impact

- 新增 `domain.AssetListQuery`、`domain.ProjectProviderProfile` 和 `NormalizeMetadataJSON`。
- 新增 `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/provider-profile`。
- `vag asset list` 增加筛选参数。
- `vag project provider get/set` 增加非敏感 provider profile 管理。

## Verification

- `npm --prefix web test -- --run`
- `npm --prefix web run build`
- Docker Go 聚焦测试：`go test ./cmd/vag ./internal/domain ./internal/app ./internal/store ./internal/httpapi ./internal/mcp ./internal/provider`
- Final regression: Docker `go test ./...`、REST smoke、Web 手测。

## Assumptions And Risks

- `offset` 分页在同一筛选结果集内稳定；若未来需要高并发持续写入下的严格无重复翻页，再增加 cursor。
- Provider profile 不解决真实 key 托管；真实 secret、用户自带 key、云端开放策略仍需单独确认。
- Web 大 chunk warning 仍是既有构建提示，不属于本 slice。

## Implementation Log

- Added asset list query parsing, store SQL filters and CLI query flags.
- Added metadata standard normalization for source/session/run/batch/story/scene/target_path.
- Added non-sensitive project provider profile API, CLI and provider model override.
- Added Web asset library filters, render budget, load more, lazy images and safe details display.
- Added Codex batch examples for pet story images and embedded article illustrations.
