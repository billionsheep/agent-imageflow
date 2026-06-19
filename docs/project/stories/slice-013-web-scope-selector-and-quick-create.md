# Story: Slice 013 - Web Scope Selector and Quick Create

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让 Web 托管模式不再主要依赖用户手填 `workspace / project / campaign` seed scope，而是能从服务端读取已有业务空间，并在设置页里直接创建新的 workspace、project、campaign。

## Source Context

- Tasks: 当前第一优先待办是“完善 Web project / campaign 管理体验，不再只依赖设置页输入 seed scope”。
- Project plan: Web 托管模式已可创建任务和选优候选图，但完整 workspace / project / campaign 管理体验仍是主要缺口。
- Tech spec: Web 托管模式已具备 API URL、鉴权和 scope 设置字段；Go 服务端已是正式事实源。
- Decisions: Web 与服务端最终应收敛到同一资产核心，不应长期依赖浏览器本地状态或手写路径。

## User Flow

1. 用户打开设置页并开启服务端托管模式。
2. Web 从服务端加载已有 workspace 列表，并根据当前选择继续加载 project、campaign。
3. 用户直接从下拉中选择已有 scope，而不是手填 ID。
4. 若还没有合适的 project 或 campaign，用户可以在设置页直接创建。
5. 创建成功后，Web 自动切换到新 scope，后续提交任务直接落在该业务空间下。

## In Scope

- 新增服务端 scope 管理 REST：列出 / 创建 workspace、project、campaign。
- Web 设置页新增 scope 刷新、选择和最小新建体验。
- 继续保留手填字段作为兜底，但默认体验转向“查 + 选 + 新建”。
- 若启用了实例级 Basic Auth，scope 管理接口复用 Basic Auth；本片不要求 project API key 管理 scope。

## Out of Scope

- 不做 workspace / project / campaign 的删除、重命名、归档。
- 不做完整后台管理页或独立 scope 列表页面。
- 不做任务迁移、资产迁移或跨 project/campaign 批量移动。
- 不做完整 quality profile Web 编辑器。
- 不做更细的权限模型、RBAC 或审计。

## Acceptance Criteria

- Given API 与数据库中已有 workspace / project / campaign，when 用户打开 Web 设置页并启用托管模式，then 可以从服务端读取并选择已有 scope。
- Given 当前 workspace 下没有目标 project，when 用户在设置页创建 project，then 服务端保存新 project，Web 自动选中新 project。
- Given 当前 project 下没有目标 campaign，when 用户在设置页创建 campaign，then 服务端保存新 campaign，Web 自动选中新 campaign。
- Given 服务端启用了 Basic Auth，when Web 加载或创建 scope，then 仍能透传 Basic Auth 成功完成操作。
- Given 用户不想用下拉，when 手动输入 scope ID，then 现有行为保持兼容，不会被新 UI 阻断。

## Technical Approach

- 在 Go 单体内补一组轻量 scope service/store/http handler：
  - `GET /api/workspaces`
  - `POST /api/workspaces`
  - `GET /api/workspaces/{workspace_id}/projects`
  - `POST /api/workspaces/{workspace_id}/projects`
  - `GET /api/workspaces/{workspace_id}/projects/{project_id}/campaigns`
  - `POST /api/workspaces/{workspace_id}/projects/{project_id}/campaigns`
- Web 通过 `agentImageflowApi` 新增 scope client，并在设置页托管模式区域接入：
  - refresh/sync
  - workspace / project / campaign 下拉
  - inline quick create
- scope 管理接口视为实例级管理操作：当前若配置了 Basic Auth，则只要求 Basic Auth；未配置 Basic Auth 时保持当前小团队自托管兼容。

## Data / Interface Impact

- 不新增数据库表，继续复用现有 `workspace`、`project`、`campaign`。
- 新增 REST scope 接口，但不修改现有任务 / 资产接口。
- Web 设置页新增局部 UI 状态，不改持久化设置结构的核心语义。

## Files or Subsystems Likely to Change

- `internal/domain/types.go`
- `internal/app/*`
- `internal/store/postgres.go`
- `internal/httpapi/server.go`
- `web/src/lib/agentImageflowApi.ts`
- `web/src/lib/agentImageflowApi.test.ts`
- `web/src/components/SettingsModal.tsx`
- `docs/project/*`

## Verification Plan

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'
npm --prefix web test -- --run
npm --prefix web run build
docker compose build api worker
docker compose up -d api worker postgres redis

# scope REST smoke
curl http://localhost:8081/api/workspaces
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"prj_scope_smoke","name":"Scope Smoke Project"}'
curl http://localhost:8081/api/workspaces/ws_default/projects/prj_scope_smoke/campaigns
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_scope_smoke/campaigns \
  -H 'Content-Type: application/json' \
  -d '{"campaign_id":"cmp_scope_smoke","name":"Scope Smoke Campaign"}'
```

## Assumptions and Risks

- 第一版 scope 管理体验先放在设置页，而不是新开独立后台页面，目的是尽快消除手填 seed scope 的主要摩擦。
- 当前 scope 管理接口如果没有启用 Basic Auth，会保持开放兼容；更细权限收口留给后续 hardening。
- 项目内已有较多未提交改动，本片只在相关模块内局部推进，不整理其他历史 slice 产物。

## Implementation Log

### 2026-06-18

- Changes:
  - REST 新增列出/创建 workspace、project、campaign 的轻量管理接口。
  - Web 设置页托管模式区域新增服务端 scope 同步、下拉选择和快速新建。
  - 保留手填 `workspace/project/campaign` ID 作为兼容兜底。
  - scope 管理接口在启用实例级 Basic Auth 时可直接复用 Basic Auth。
- Verification:
  - `docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'`
  - `npm --prefix web test -- --run`
  - `npm --prefix web run build`
  - `docker compose build api worker`
  - Docker smoke：使用 Basic Auth 创建 `ws_scope_smoke -> prj_scope_smoke -> cmp_scope_smoke`，随后在新 campaign 下成功创建任务并产出 asset。
- Environment note:
  - 本次 Go 格式化与测试使用 Docker Go 镜像执行，因为当前宿主环境未直接提供 `go` / `gofmt` 命令；正常本地开发环境可直接运行 `go test ./...`。
- Remaining gaps: 真实 edit/mask 输入取回、best-of 更强评分、自托管生产硬化仍待后续 slice。
