# Story: Slice 024 - Storage Governance Usage and Dry Run

## Status

- State: Done
- Created: 2026-06-19
- Updated: 2026-06-19

## Product Goal

让 Agent ImageFlow 从“能生成和展示资产”继续升级到“能看见存储占用、识别可治理候选、并在执行删除前有安全 dry-run”的资产治理基础能力。

## Source Context

- CSV: `issues/next-phase-p1-storage-governance.csv`
- Completed in this pass: `P1-STOR-001` to `P1-STOR-008`

## User Flow

1. 用户在 Web Scope 管理中查看 workspace / project / campaign 的资产、任务和存储占用。
2. 管理员通过 REST 读取当前 scope 与实例级 storage-governance 统计。
3. 管理员通过本地 CLI 执行 cleanup dry-run，预览 rejected / generated / tmp / orphan 候选。
4. 系统默认保护 selected / approved / published，不把它们放入默认清理候选。
5. 管理员使用 dry-run token 或显式确认参数执行本地 CLI 清理。
6. 管理员通过 REST/Web 查看脱敏 storage-integrity 摘要。

## In Scope

- 只读 storage usage scanner。
- 只读 REST storage-governance API。
- Web Scope 管理存储占用展示。
- 本地 CLI cleanup dry-run preview。
- 本地 CLI cleanup execute 受控执行。
- 只读 REST/Web storage-integrity 摘要。
- selected / approved / published 默认保护规则。

## Out of Scope

- 不暴露匿名远程清理入口。
- 不做完整 retention policy UI。
- 不做通用 DAM、模板市场或账号运营系统。

## Acceptance Criteria

- `ScanUsage` 可统计 `original`、`thumbnail`、`metadata`、`input_files`、`audit`、`tmp`、`orphan` 的文件数与字节数。
- 缺失 storage root 返回 0，不报错。
- REST 返回当前 scope 与 instance 统计，并复用 project API key 鉴权。
- Web Scope 管理显示 storage 总占用与分类占用。
- CLI dry-run 只返回候选，不修改文件或数据库。
- selected / published 不进入清理候选。
- CLI cleanup execute 必须带 `--execute`，并需要匹配 dry-run token 或显式 `--confirm`。
- cleanup execute 只清理 draft/rejected asset candidates、tmp 和 orphan files。
- cleanup execute 写入 `source=cli` audit log。
- REST/Web storage-integrity 不暴露本地绝对路径。

## Technical Approach

- 在 `internal/storage` 中新增只读 scanner，按 storage root 相对路径分类。
- 在 `internal/store` 中新增 task/asset/count 聚合与 cleanup asset 候选读取。
- 在 `internal/app` 中组合 storage usage、DB counts 和 cleanup dry-run。
- 在 `internal/httpapi` 新增 `GET /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/storage-governance`。
- 在 Web API client 与 Scope 管理控制台中消费该接口。
- 在 `cmd/vag` 增加 `storage cleanup-preview` 本地 dry-run 命令。
- 在 `cmd/vag` 增加 `storage cleanup-execute` 本地执行命令。
- 在 `internal/app` / `internal/store` 中实现只允许 draft/rejected 的事务式 asset 候选删除。
- 在 `internal/storage` 中实现 storage-root 相对 key 删除保护。
- 在 `internal/httpapi` 和 Web Scope 管理中接入只读 `storage-integrity` 摘要。

## Data / Interface Impact

- 新增只读 REST endpoint：`GET /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/storage-governance`
- 新增 CLI：`vag storage cleanup-preview`
- 新增只读 REST endpoint：`GET /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/storage-integrity`
- 新增 CLI：`vag storage cleanup-execute`
- 不新增数据库表。
- 不改变现有 asset/task 状态机。

## Verification

- `npm --prefix web test -- --run`
- `npm --prefix web run build`
- `docker run --rm -v /Users/moon/Workspace/tools/agent-imageflow:/src -w /src golang:1.25.3-alpine go test ./...`
- `curl -sf http://localhost:8081/healthz`
- `curl /storage-governance` 未带 project key 返回 `401`
- `curl /storage-governance` 带 smoke project key 返回 `200`
- `docker compose exec -T api /app/vag storage cleanup-preview --workspace ws_default --project prj_xhs_anime --campaign cmp_7day_cover --limit 5`
- `docker compose exec -T api /app/vag storage cleanup-execute --workspace ws_p1stor_smoke_1781852988 --project prj_p1stor_smoke_1781852988 --campaign cmp_p1stor_smoke_1781852988 --execute --dry-run-token cleanup_1d34d5d167d4261d2115d0ac30a96371`
- `GET /storage-integrity` for `ws_p1stor_smoke_1781852988` returned `ok=true` and `issue_count=0`.
- Browser smoke: Scope 管理显示 `storage/original/thumbnail/metadata` 统计。

## Implementation Log

### 2026-06-19

- Added storage usage scanner and tests for missing root, empty root, normal fixtures, orphan detection and large-file stat.
- Added storage-governance app/store/http API path.
- Added Web storage governance client and Scope Manager storage display.
- Added cleanup dry-run service and `vag storage cleanup-preview`.
- Added protection tests for selected/published/deprecated assets.
- Added controlled cleanup execution with dry-run token gate, `--execute`, optional `--confirm`, storage key safety, DB candidate deletion and CLI audit log.
- Added scoped `storage-integrity` read endpoint and Web Scope Manager summary counts.
- Smoke fixture `ws_p1stor_smoke_1781852988` verified 1 approved asset protected, 1 rejected + 1 draft deleted, second dry-run empty and integrity ok.
- Updated `TASKS.md`, `CHECKPOINTS.md`, `PROJECT_PLAN.md`, `RUNBOOK.md` and CSV evidence.

## Remaining Gaps

- 第一版清理执行只提供本地 CLI 入口，不提供远程 REST 执行或 Web 批量删除按钮。
- 过期策略、配额、retention policy UI、批量清理审批流属于后续 P1/P2。
