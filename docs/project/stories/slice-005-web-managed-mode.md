# Story: 005 - Web managed mode

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让 Web 工作台可以把图片生成提交到 Agent ImageFlow 服务端托管任务流，并在页面中查看服务端候选资产、执行 selected/rejected 状态标记和打开交付文件。

## Source Context

- Product spec: `docs/project/PRODUCT_SPEC.md`
- Project plan slice: `docs/project/PROJECT_PLAN.md` 的 “Phase 5: Web managed mode”
- Tech spec: `docs/project/TECH_SPEC.md`
- Architecture: `docs/project/ARCHITECTURE.md`
- Related decisions: `docs/project/DECISIONS.md`

## User Flow

1. 用户在 Web 设置中开启服务端托管模式，并确认 API URL、workspace、project、campaign 和 provider。
2. 用户在现有输入框提交 prompt。
3. Web 调用服务端 REST API 创建 `ImageTask`，页面显示本地任务卡并轮询服务端状态。
4. Worker 完成生成后，任务卡和详情页展示服务端 thumbnail/original 候选图。
5. 用户在详情页对单张候选资产执行 select 或 reject，并可以打开 original / thumbnail / metadata 交付 URL。

## In Scope

- 新增 Web 设置项：是否启用服务端托管模式、服务端 API URL、workspace/project/campaign、provider。
- 托管模式下提交任务调用现有 REST API，生成结果仍由服务端 Worker 处理。
- 轮询 `GET /api/tasks/{task_id}`，把服务端 assets 同步到本地任务记录。
- 在任务卡和详情页展示服务端候选图。
- 在详情页支持 select/reject 当前候选图；产品语义使用 selected/rejected，底层 REST 仍兼容调用 `/approve`。
- 保留现有浏览器直连 provider / IndexedDB playground mode。

## Out of Scope

- 不做完整 Workspace / Project / Campaign 管理界面。
- 不做真实 provider 配置写入服务端。
- 不做 reference image、mask/edit 参数迁移到服务端。
- 不做自动 best-of 选优。
- 不做数据库状态命名迁移或后端 `/select` REST 别名。
- 不做生产鉴权、项目级 API key 或部署硬化。

## Acceptance Criteria

- Given 用户未开启服务端托管模式，when 提交任务，then 现有浏览器直连生成路径保持不变。
- Given 用户开启服务端托管模式，when 提交 prompt，then Web 创建服务端 `ImageTask` 并展示 running 任务卡。
- Given 服务端任务完成，when Web 轮询到 assets，then 任务卡显示服务端 thumbnail，详情页可查看多张候选图。
- Given 当前候选图处于 generated，when 用户点击 select，then Web 调用服务端兼容状态接口并把候选图显示为 selected。
- Given 当前候选图处于 generated 或 selected，when 用户点击 reject，then Web 调用服务端状态接口并把候选图显示为 rejected。
- Given 服务端任务失败，when Web 轮询到 failed，then 本地任务进入 error 并展示服务端错误信息。

## Technical Approach

- 扩展 `agentImageflowApi.ts`，补齐 `selectAgentImageflowAsset`、`getAgentImageflowAsset` 和状态语义映射。
- 扩展 `TaskRecord`，保存服务端 task id、scope、status、assets 和 delivery URL。
- `submitTask` 在设置启用托管模式时切换到 `submitManagedImageflowTask`，不走浏览器 provider。
- 轮询逻辑只更新当前任务；完成、部分完成、失败和超时都写入本地任务状态。
- 任务卡优先使用服务端资产 thumbnail；详情页优先使用服务端 original/thumbnail，并提供 select/reject 和交付打开按钮。

## Data / Interface Impact

- 新增前端本地设置字段，不修改后端 schema。
- 新增前端本地 `TaskRecord` 字段，不影响服务端 API。
- REST 仍使用现有接口：
  - `POST /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/tasks`
  - `GET /api/tasks/{task_id}`
  - `GET /api/assets/{asset_id}`
  - `POST /api/assets/{asset_id}/approve`
  - `POST /api/assets/{asset_id}/reject`

## Files or Subsystems Likely to Change

- `web/src/types.ts`
- `web/src/lib/agentImageflowApi.ts`
- `web/src/lib/agentImageflowApi.test.ts`
- `web/src/lib/apiProfiles.ts`
- `web/src/store.ts`
- `web/src/components/SettingsModal.tsx`
- `web/src/components/TaskCard.tsx`
- `web/src/components/DetailModal.tsx`
- `docs/project/`

## Verification Plan

```bash
npm --prefix web test -- --run
npm --prefix web run build
docker compose config
docker compose build
docker compose up -d postgres redis api worker
# Manual smoke: enable Web managed mode with mock provider, submit prompt, poll task, select/reject an asset.
```

## Assumptions and Risks

- 第一版 Web managed mode 使用默认 seed scope：`ws_default`、`prj_xhs_anime`、`cmp_7day_cover`。
- 前端无法直接知道服务端 provider 是否启用；创建任务失败时展示服务端错误。
- 第一版不把服务端资产导入 IndexedDB，因此旧的编辑输出和 ZIP 下载仍主要服务 legacy playground 图片。
- 如果 `PUBLIC_BASE_URL` 与浏览器可访问地址不一致，缩略图 URL 可能需要用户在服务端配置中调整。

## Implementation Log

### 2026-06-18

- Changes: 新增 Web 服务端托管设置；扩展 `agentImageflowApi` 的 task/asset 查询、select/reject 和状态语义映射；`submitTask` 在托管模式下创建服务端 `ImageTask` 并轮询状态；本地 `TaskRecord` 增加服务端 task/scope/assets 元数据；任务卡和详情页可展示服务端 thumbnail/original，详情页支持 select/reject 和打开 original / metadata URL；保留 legacy playground mode。
- Verification: `npm --prefix web test -- --run` 通过；`npm --prefix web run build` 通过；`docker compose config` 通过；`docker compose build` 通过；`docker compose up -d postgres redis api worker` 启动成功；REST smoke 跑通 create task -> poll completed -> approve first asset -> reject second asset -> get delivery URL；Web dev server 已启动并通过 `curl -I http://localhost:8082/` 返回 200。
- Remaining gaps: Web 尚无完整 workspace/project/campaign 管理体验；本 slice 完成时托管模式仍偏纯文本 prompt，后续 slice 已补 reference/mask descriptor、quality profile 和 best-of；真实 provider edit/mask 调用仍待后续 slice。
