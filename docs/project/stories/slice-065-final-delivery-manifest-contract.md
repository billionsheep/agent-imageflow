# Story: 065 - Final Delivery Manifest Contract

## Status

- State: Done
- Created: 2026-06-27
- Updated: 2026-06-27

## Product Goal

在不改 canonical storage、不中断现有工程 manifest 调用方的前提下，为人工复盘、再次寻找和 NAS 浏览补一层更直给的 final delivery 视图，让调用方可以按 `story/scene/batch` 直接读到“最终该交付哪张图”。

## Source Context

- Delivery export CSV: `issues/next-phase-p1-final-delivery-nas-readable-export.csv`
- Existing manifest slice: `docs/project/stories/slice-048-minimal-export-manifest.md`
- Caption lineage slice: `issues/next-phase-p1-caption-edit-lineage.csv`
- Story continuity guide: `docs/project/STORY_CONTINUITY_AGENT_GUIDE.md`

## In Scope

- 扩展 `GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-manifest`，新增 `view=engineering|final_delivery`。
- 扩展 CLI `vag batch manifest --view engineering|final_delivery`。
- Web Production View 新增 `Final delivery manifest` 导出动作。
- 在兼容保留旧顶层 manifest 的同时，新增 `manifest_view` 和只读 `final_delivery` block。
- `final_delivery` 按 `counts/stories/scenes/final_assets` 输出最终交付摘要。
- 将 caption 派生图关系扁平为 `derived_from_asset_id` / `derivation_type`。
- 继续屏蔽 `local_path`、绝对路径、cookie、session 和 secret-like 字段。

## Out of Scope

- 不做独立导出接口或新页面。
- 不做 story/batch export pack。
- 不做 NAS readable mirror。
- 不做 project delivery defaults。
- 不做 archive/restore/cleanup 与 export/mirror 的治理联动代码。
- 不改 canonical storage、asset/task 状态机或数据库表。
- 不运行真实 provider。

## Acceptance Criteria

- REST、CLI 和 Web 都可以请求 `view=final_delivery`。
- 默认 `engineering` 视图保持现有兼容，不破坏旧调用方。
- `final_delivery.final_assets` 统一表达 scene 最终交付资产，并区分 base selected 与 caption derivative selected。
- 人工在不翻深层 metadata 的前提下，可直接读到 `story_id`、`scene_id`、`primary_selected_asset_id`、`delivery_role`、`derived_from_asset_id`、`derivation_type`、`download_url`、`thumbnail_url`、`metadata_url`、`target_path` 和 continuity / visual context 摘要。
- 响应不包含 `local_path`、宿主机绝对路径或任何 secret-like 字段。

## Verification Plan

- Containerized Go focused tests for `internal/app`、`internal/httpapi`、`cmd/vag`。
- Web focused tests for API client 和 Production View manifest action。
- `git diff --check`。
- 不跑真实 provider。

## Implementation Log

### 2026-06-27

- 后端复用现有 `batch-manifest` builder，新增 `view` 解析、`manifest_view` 和 `final_delivery` read model。
- `final_delivery.final_assets` 固定收集 scene 内所有 `delivery_role=final_delivery` 的资产，并把 caption 派生关系扁平到顶层字段。
- CLI 新增 `--view`，Web Production View 新增 `Final delivery manifest` 按钮。
- focused 验证通过：
  - `docker run --rm -v /Users/moon/Workspace/tools/agent-imageflow:/src -w /src golang:1.25.3-alpine sh -c 'go test ./internal/app ./internal/httpapi ./cmd/vag'`
  - `npm --prefix web test -- --run src/lib/agentImageflowApi.test.ts src/components/ProductionViewModal.test.ts`
- 本轮未实现 ZIP、mirror 或 defaults 代码，未运行真实 provider，未读取或打印任何 key、cookie、session 或 secret。
