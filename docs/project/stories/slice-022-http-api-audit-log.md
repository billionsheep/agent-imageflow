# Story: Slice 022 - HTTP / API Audit Log

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

给 Agent ImageFlow 的 HTTP API 增加第一版结构化审计日志，至少能回答“谁在什么时候，对哪个 workspace / project / campaign / task / asset 做了什么，请求结果如何”，并保持当前 Web managed mode、CLI、Basic Auth、project API key 和 MCP/Worker 主流程不被打断。

## Source Context

- Project plan slice: 启动本片时，`Production hardening` 的下一条 pending slice 是审计日志。
- Tasks: 启动本片时，Todo 第一条是 `补第一版 HTTP / API 审计日志`。
- Architecture: 架构文档已明确 PostgreSQL 负责事实和审计，但当前项目规则不希望这一片直接引入新的数据库迁移时，应优先选择更小的 boring default。
- Related decisions: 基础限流刚完成，剩余 hardening 缺口已收敛到审计和多 key 策略。

## User Flow

1. 运维或开发者通过 Web、CLI 或外部系统调用 Agent ImageFlow HTTP API。
2. API 在处理请求后，写入一条结构化审计事件。
3. 审计事件至少记录时间、action、route、状态码、actor、鉴权方式和关键业务 scope / resource id。
4. 本地维护者可通过 CLI 查看最近的审计事件，并按 project / task / asset 等条件过滤。
5. 当请求失败、被拒绝或被限流时，审计事件仍会保留结果信息，便于排查。

## In Scope

- 给 HTTP API 增加结构化审计事件写入。
- 记录关键字段：时间、method、route、action、status、duration、auth mode、actor、workspace/project/campaign/task/asset/input_file id、error code。
- 审计落本地文件系统，复用当前 `STORAGE_ROOT`。
- 增加本地 `vag audit list` 查询入口。
- 补 focused tests、build 和本地 smoke。

## Out of Scope

- 不新增数据库表或迁移。
- 不做 Web 审计页面或 REST 审计查询接口。
- 不做日志聚合、告警、retention 治理或对象存储归档。
- 不做多 key usage 统计或计费。
- 不做 MCP stdio 独立审计协议扩展。

## Acceptance Criteria

- Given HTTP API 正常处理关键入口请求，when 请求完成，then 本地会新增一条结构化审计事件。
- Given 请求命中 Basic Auth、project API key、401、429 或普通 2xx，when 查看审计事件，then 能看到对应 `auth_mode`、actor、action、status code 和 error code。
- Given 创建 task、读取 asset、上传 input-file 等常见请求，when 查看审计事件，then 能看到对应的 scope / resource id。
- Given 运维执行 `vag audit list`，when 指定 `--project`、`--task` 或 `--asset` 等过滤条件，then 能得到过滤后的结构化 JSON 输出。
- Given 审计写入失败，when API 继续处理请求，then 业务请求本身不因为审计失败而返回 5xx。

## Technical Approach

- 在 `internal/domain` 定义 HTTP audit event/query/list response。
- 在 `internal/storage` 增加基于本地 JSONL 的 audit sink 和读取过滤逻辑，目录放在 `storage/audit/http-api/`。
- 在 `internal/httpapi` 增加 response capture + request audit metadata，针对 `/api/*` 请求在返回后写审计事件。
- 复用现有 route/scope 解析；新建资源场景通过响应 JSON 反填 `task_id` / `asset_id` / `workspace_id` 等字段。
- 通过 `cmd/api` 把 `LocalStorage` 作为 audit sink 注入。
- 通过 `cmd/vag audit list` 提供本地查询能力。

## Data / Interface Impact

- 新增本地文件：`storage/audit/http-api/YYYY-MM-DD.jsonl`
- 新增 CLI：`vag audit list`
- 不新增数据库 schema，不新增外部依赖。

## Files or Subsystems Likely to Change

- `internal/domain/*`
- `internal/storage/*`
- `internal/httpapi/*`
- `internal/app/access.go`
- `cmd/api/main.go`
- `cmd/vag/main.go`
- `docs/project/*`

## Verification Plan

```bash
docker build --target build -t agent-imageflow-build .
docker run --rm agent-imageflow-build sh -lc 'cd /src && /usr/local/go/bin/go test ./internal/httpapi ./internal/storage ./cmd/vag ./...'
npm --prefix web run build
docker compose config
docker compose up -d --force-recreate api worker
# manual smoke:
# 1. 调一次 GET /api/workspaces 和一次 POST /tasks
# 2. docker compose exec api /app/vag audit list --limit 5
# 3. 确认输出里有 action/status/project_id/task_id 等字段
```

## Assumptions and Risks

- 第一版审计先以本地 JSONL 作为 boring default，优先满足“有记录、能查、不中断业务”。
- 当前 slice 不直接写 PostgreSQL 审计表，是为了避免在未确认前引入数据库迁移。
- 审计事件默认落在 `STORAGE_ROOT` 下，会增加少量本地 I/O；若未来需要 retention/归档，再补后续 slice。
- 当前 actor 先以 Basic 用户名和 project API key 名称为主，不进入更细粒度的用户体系。

## Implementation Log

### 2026-06-18

- Changes:
  - 新增 `internal/domain/audit.go`，定义 `HTTPAuditEvent`、`HTTPAuditQuery` 和 `HTTPAuditListResponse`。
  - 新增 `internal/storage/audit.go` 与 `internal/storage/audit_test.go`，把 `/api/*` 审计事件落到 `STORAGE_ROOT/audit/http-api/YYYY-MM-DD.jsonl`，并支持按 project / task / asset / actor / status 过滤查询。
  - 新增 `internal/httpapi/audit.go`，实现 request/response capture、actor/auth metadata、route/action 推断，以及从响应 JSON 反填 `task_id` / `asset_id` / `error_code`。
  - 更新 `internal/httpapi/server.go`，在 HTTP API 请求返回后写入审计事件；审计写入异常时 fail-open。
  - 更新 `internal/app/access.go`，补充 project access config 查询辅助方法，用于把 project API key 名称写入审计 actor。
  - 更新 `cmd/api/main.go`，注入 `LocalStorage` 作为 audit sink。
  - 更新 `cmd/vag/main.go`，新增 `vag audit list` 本地查询命令。
  - 同步更新 README、Runbook、Project Plan、Tasks、Checkpoints 和 Decisions。
- Verification:
  - `docker build --target build -t agent-imageflow-build .`
  - `docker run --rm -v /Users/moon/Workspace/tools/agent-imageflow:/src -w /src agent-imageflow-build sh -lc '/usr/local/go/bin/go test ./internal/httpapi ./internal/storage ./cmd/vag'`
  - `docker run --rm agent-imageflow-build sh -lc 'cd /src && /usr/local/go/bin/go test ./...'`
  - `npm --prefix web run build`
  - `docker compose config`
  - `docker compose up -d --build --force-recreate api worker`
  - Docker smoke 已验证 `create_task` / `get_task` 和 `GET /api/tasks/task_missing -> 404 not_found` 都会写入审计记录，并可通过 `docker compose exec api /app/vag audit list --project prj_audit_smoke --task task_071eb31a8b161a4de0b5` 查询。
- Remaining gaps:
  - 当前审计只提供本地 JSONL + CLI 查询；尚未提供 REST 查询接口、Web 审计页、retention 治理或外部日志归档。
  - 当前只覆盖 HTTP API，不扩展到 MCP stdio 单独审计协议。
  - 下一条生产 hardening slice 仍是项目级多 key 策略。
