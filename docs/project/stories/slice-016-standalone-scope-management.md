# Story: Slice 016 - Standalone Scope Management

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

把当前只存在于设置页中的 scope 同步/新建能力，推进为一个独立的 Web scope 管理入口。用户可以在不展开设置页内联表单的情况下，查看 workspace / project / campaign 树，执行基础 rename、archive/unarchive、delete，并把某个 campaign 设为当前托管 scope。

## Source Context

- Tasks: 本 slice 对应当时的第一条 Todo：“补独立 Web scope 管理页与基础 rename/delete/归档体验”。
- Project plan: 本 slice 对应当时的 Phase 6.1 pending 项，目标是从设置页内联操作升级到独立管理入口。
- Tech spec: 当时 Web scope selector / quick create 已完成；本 slice 补独立管理页与 rename/delete/archive。
- Decisions: scope 管理仍属于实例级管理能力；若启用 Basic Auth，则这些管理接口只要求 Basic Auth。

## User Flow

1. 用户在 Web 顶栏或设置入口打开独立 scope 管理弹窗。
2. 弹窗从服务端加载 workspace、project、campaign，并展示 active / archived 状态。
3. 用户选择某个 workspace，再查看其 project；选择 project 后查看其 campaign。
4. 用户可对选中的 scope 执行 rename、archive/unarchive、delete。
5. 用户可把某个 active campaign 设为当前托管 scope，回到托管模式继续创建任务。

## In Scope

- Go 服务端新增 scope 的 update/delete 能力：
  - workspace rename + archive/unarchive + delete
  - project rename + archive/unarchive + delete
  - campaign rename + archive/unarchive + delete
- 独立 Web scope 管理入口：
  - 不再只依赖设置页内联 section
  - 展示 workspace / project / campaign 层级
  - 显示 archived 状态
  - 支持 rename、archive/unarchive、delete、设为当前 scope
- 设置页 scope selector 保持可用，并对 archived scope 做兼容处理，避免继续默认选中归档项。

## Out of Scope

- 不做完整后台页面、路由系统或新的持久化表。
- 不做 scope 权限模型细化；仍按当前实例级管理能力处理。
- 不补远程 URL 抓取、asset reuse、更多 provider edit/mask。
- 不补 best-of 视觉/LLM 打分。
- 不补 project/campaign description 编辑。

## Acceptance Criteria

- Given 用户在 Web 中需要管理业务空间，when 打开独立 scope 管理入口，then 可以看到 workspace / project / campaign 列表，而不是只在设置页里操作。
- Given 用户重命名 workspace/project/campaign，when 操作成功，then 列表立即显示新名称。
- Given 用户归档或恢复某个 scope，when 操作成功，then UI 能显示 archived 状态变化，并且设置页不再默认把 archived 项作为当前可选项。
- Given 用户尝试删除可安全删除的 scope，when 操作成功，then 该项从列表消失；when scope 仍非空，then 返回明确错误而不是静默失败。
- Given 用户在管理页中选择某个 active campaign，when 点击设为当前 scope，then 托管模式后续会使用该 workspace/project/campaign。

## Technical Approach

- 服务端在现有 scope list/create 接口旁，补充 `PATCH` / `DELETE`：
  - `PATCH /api/workspaces/{workspace_id}`
  - `DELETE /api/workspaces/{workspace_id}`
  - `PATCH /api/workspaces/{workspace_id}/projects/{project_id}`
  - `DELETE /api/workspaces/{workspace_id}/projects/{project_id}`
  - `PATCH /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}`
  - `DELETE /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}`
- `archived` 状态先复用现有 `metadata_json`，不引入数据库迁移。
- Web 新增独立 modal，复用现有 managed mode 的 Base URL / Basic Auth / API key 设置。
- 设置页继续保留 quick create，但 selector 默认过滤 archived 项。

## Data / Interface Impact

- REST scope summary 返回增加 `archived` 字段。
- REST 新增 scope update/delete 接口。
- 不修改任务、资产、provider 或 MCP 协议。

## Files or Subsystems Likely to Change

- `internal/domain/types.go`
- `internal/app/scope.go`
- `internal/httpapi/server.go`
- `internal/store/postgres.go`
- `internal/storage/*`（如删除 campaign 时需要清理输入文件）
- `web/src/lib/agentImageflowApi.ts`
- `web/src/lib/agentImageflowApi.test.ts`
- `web/src/store.ts`
- `web/src/components/Header.tsx`
- `web/src/components/SettingsModal.tsx`
- `web/src/components/icons.tsx`
- `web/src/components/ScopeManagerModal.tsx`

## Verification Plan

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'
npm --prefix web test -- --run
npm --prefix web run build
docker compose build api worker
docker compose up -d --force-recreate api worker
```

手工 / smoke：

- `curl -u admin:secret http://localhost:8081/api/workspaces`
- `curl -u admin:secret -X PATCH ...` 验证 rename/archive
- `curl -u admin:secret -X DELETE ...` 验证空 scope 删除与非空 scope 报错

## Assumptions and Risks

- 本片继续把 scope 管理视为实例级管理能力，不新增 project API key 约束。
- 归档先作为组织管理状态，不额外阻断已有任务/资产读取。
- 由于 `input-files` 不入数据库，campaign 删除时若存在输入文件目录，需要一起清理或阻止删除，避免留下明显孤儿数据。

## Implementation Log

### 2026-06-18

- Changes:
  - 服务端新增 workspace / project / campaign 的 `PATCH` / `DELETE` 接口，支持 rename、archive/unarchive 和空 scope 删除。
  - scope summary 返回新增 `archived` 字段；archive 状态先复用 `metadata_json.archived_at`，不引入新的数据库迁移。
  - campaign 删除会同时清理当前 scope 下的本地 `input-files` 目录；project/workspace 删除则要求下级 scope 为空，避免留下明显孤儿数据。
  - Web 顶栏新增独立 `Scope 管理` 入口，打开独立 modal 展示 workspace / project / campaign 层级，可 rename/archive/delete，并把 active campaign 设为当前托管 scope。
  - 设置页保留 quick create，但 scope selector 默认过滤 archived 项；若当前保存的 scope 已归档或不存在，会自动回退到首个 active 项或清空选择。
- Verification:
  - `docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'`
  - `npm --prefix web test -- --run`
  - `npm --prefix web run build`
  - `docker compose build api worker`
  - `BASIC_AUTH_USERNAME=admin BASIC_AUTH_PASSWORD=secret docker compose up -d --force-recreate api worker`
  - Docker smoke 已验证：workspace/project/campaign rename、archive/unarchive、非空 project 删除报错、campaign 删除后 scope 输入文件目录被清理、project/workspace 最终可完整删除。
- Remaining gaps:
  - 远程 URL 抓取、asset reuse 与更多 provider 的 edit/mask 输入复用仍未接入。
  - best-of 仍是本地元数据启发式，尚未升级为可插拔视觉/LLM 打分。
