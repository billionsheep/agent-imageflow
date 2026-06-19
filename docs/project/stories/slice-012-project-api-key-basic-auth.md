# Story: Slice 012 - Project API Key and Basic Auth

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让自托管的 Agent ImageFlow 在不引入复杂用户权限系统的前提下，具备最小可用的生产鉴权能力：实例级 Basic Auth 负责挡住公开暴露的 HTTP 入口，项目级 API key 负责给外部系统、Web 托管模式和脚本提供按 project 隔离的调用凭据。

## Source Context

- Input/output spec: 第一版 REST API 不要求复杂鉴权，但接口设计必须预留项目级 API key。
- Project plan: Phase 6 MVP hardening 还缺项目级 API key、基本鉴权和自托管生产配置样例。
- Decisions: repair/reconcile 目前只保留在本地 CLI，因为还没有完成项目级 API key 和权限模型。
- Current code: HTTP API 目前完全公开；Web managed mode 只有 API URL / workspace / project / campaign / provider 配置，没有鉴权字段。

## User Flow

1. 自托管管理员为实例配置可选 Basic Auth。
2. 管理员为某个 project 配置或轮换项目级 API key。
3. 外部系统、CLI 或 Web managed mode 携带 Basic Auth 和 `X-API-Key` 调用项目下的 REST API。
4. 若 project 未启用 API key，则 API 保持当前无感兼容；若已启用，则相关 REST 请求必须带正确的项目 key。
5. 管理员可以查询 project 当前 access config，但不会拿回明文 key。

## In Scope

- 为 API server 增加可选实例级 Basic Auth。
- 为 project 增加 API key access config 的保存、读取和轮换能力。
- 对 REST 路由增加项目级 API key 校验。
- 新增最小 REST/CLI 管理入口，用于读取和更新 project access config。
- 给 Web managed mode 增加最小鉴权配置项，使其在启用鉴权后仍可调用服务端。
- 更新 Docker Compose / Runbook，给出自托管生产配置样例。

## Out of Scope

- 不做完整用户系统、角色权限、SSO 或企业级 RBAC。
- 不给 MCP stdio 增加远程鉴权。
- 不给 repair/reconcile 开放远程管理接口。
- 不做限流、审计日志、配额或 key 使用统计。
- 不做多 key 列表管理；第一版每个 project 只维护一个有效 API key。

## Acceptance Criteria

- Given API server 配置了 `BASIC_AUTH_USERNAME` / `BASIC_AUTH_PASSWORD`，when 未提供正确 Basic Auth 调用 REST API，then 返回 401。
- Given 某个 project 已启用 API key，when 调用该 project 相关 REST 路由未携带正确 `X-API-Key` 或 Bearer token，then 返回 401。
- Given project API key 配置成功，when 查询 access config，then 能看到 enabled/name/preview，但不会返回明文 key。
- Given Web managed mode 或 CLI 配置了 Basic Auth 和项目 API key，when 创建任务、查询任务或读取 asset，then 仍能完成现有闭环。
- Given 未启用 project API key 的 project，when 现有 mock smoke 运行，then 默认行为保持兼容，不会平白被拦截。

## Technical Approach

- 在 `project.metadata_json` 中新增 `access_config` 子对象，保存 `api_key_enabled`、`api_key_name`、`api_key_preview` 和 `api_key_hash`。
- 使用标准库 `sha256` + 常量时间比较校验 API key，不保存明文。
- API server 在 HTTP 层先校验可选 Basic Auth，再按路由解析 project scope 并校验项目级 API key。
- 通过新增 `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/access-config` 管理项目 access config。
- CLI `vag` 增加 project access 子命令，并支持从环境变量注入 Basic Auth / API key。
- Web managed mode 设置页新增 API key、Basic user、Basic password 字段；服务端调用时自动带 `X-API-Key` 与 Basic Auth header。

## Data / Interface Impact

- 不新增数据库表；继续复用 `project.metadata_json`。
- 新增 REST 接口：
  - `GET /api/workspaces/{workspace_id}/projects/{project_id}/access-config`
  - `POST /api/workspaces/{workspace_id}/projects/{project_id}/access-config`
- 新增环境变量：
  - `BASIC_AUTH_USERNAME`
  - `BASIC_AUTH_PASSWORD`
- CLI 新增 `vag project access get|set`。

## Files or Subsystems Likely to Change

- `internal/domain/types.go`
- `internal/app/*`
- `internal/store/postgres.go`
- `internal/httpapi/server.go`
- `internal/config/config.go`
- `cmd/api/main.go`
- `cmd/vag/main.go`
- `web/src/lib/agentImageflowApi.ts`
- `web/src/lib/apiProfiles.ts`
- `web/src/components/SettingsModal.tsx`
- `web/src/store.ts`
- `docker-compose.yml`
- `docs/project/*`

## Verification Plan

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine /usr/local/go/bin/go test ./...
npm --prefix web test -- --run
npm --prefix web run build
docker compose build api worker
BASIC_AUTH_USERNAME=admin BASIC_AUTH_PASSWORD=secret docker compose up -d --force-recreate api worker
# smoke:
# 1) use Basic Auth to configure project access-config
# 2) verify unauthenticated request gets 401
# 3) verify authenticated + X-API-Key request can create/get task and asset
```

## Assumptions and Risks

- 第一版 project API key 只保留一把活动 key，避免扩张到完整凭据管理系统。
- 由于不新增加密依赖，API key hash 先用标准库 `sha256`；这适合当前单体/小团队自托管场景，但不是高强度密钥管理方案。
- Web managed mode 的鉴权配置先保存在本地设置中，与当前 Web 保存 API key 的模式保持一致。

## Implementation Log

### 2026-06-18

- Changes:
  - 新增 `project.metadata_json.access_config` 读写、项目级 API key hash/name/preview 管理和 `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/access-config`。
  - API server 新增可选实例级 Basic Auth、项目级 `X-API-Key` / Bearer 校验，以及对 task/asset 路由的 project scope 解析。
  - `vag` 新增 `project access get|set`，并支持 `AGENT_IMAGEFLOW_BASIC_USER`、`AGENT_IMAGEFLOW_BASIC_PASS`、`AGENT_IMAGEFLOW_API_KEY` 透传鉴权。
  - Web managed mode 设置页新增 Project API Key / Basic 用户名 / Basic 密码，并在托管请求里自动附带鉴权 header。
  - Docker Compose、Runbook、计划/检查点/决策文档已同步。
- Verification:
  - `docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/gofmt -w internal/domain/types.go internal/app/access.go internal/app/access_test.go internal/store/postgres.go internal/httpapi/server.go internal/config/config.go cmd/api/main.go cmd/vag/main.go && /usr/local/go/bin/go test ./...'`
  - `npm --prefix web test -- --run`
  - `npm --prefix web run build`
  - `docker compose build api worker`
  - Docker smoke：启用 `BASIC_AUTH_USERNAME=admin BASIC_AUTH_PASSWORD=secret` 后，未认证请求返回 `401`；仅 Basic 访问启用 project key 的 quality-profile 返回 `401`；Basic + `X-API-Key` 可创建任务并读取 asset；CLI `task get` / `project access get` 在透传鉴权后通过。
- Remaining gaps: Web project/campaign 管理体验、真实 edit/mask 输入链路和更强 best-of 评分仍待后续 slice。
