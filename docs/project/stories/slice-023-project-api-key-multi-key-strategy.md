# Story: Slice 023 - Project API Key Multi-Key Strategy

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让 Agent ImageFlow 的项目级 access config 从“每个 project 只有一把活动 key”升级为“每个 project 可维护多把命名 key”，从而支持低风险轮换、灰度切换和按系统分发凭据；同时保持当前 Basic Auth、Web managed mode、CLI、MCP、限流和本地审计链路不回退。

## Source Context

- Project plan slice: 启动本片时，`Production hardening` 的下一条 pending slice 是项目级多 key 策略。
- Tasks: 启动本片时，Todo / Doing 第一条都是 `扩展项目级更多 key 策略`。
- Decisions: `第一版项目鉴权采用实例级 Basic Auth + 项目级单 key` 已明确这是后续 hardening 方向。
- Tech spec: 当前 access config 仍只描述单把 key，需要在不做数据库迁移的前提下升级。

## User Flow

1. 管理员读取某个 project 的当前 access config。
2. 管理员在不影响现有调用方的前提下，为该 project 新增第二把命名 key。
3. 新调用方开始使用新 key；旧调用方仍可继续使用旧 key。
4. 管理员确认切换完成后，可单独 disable 或 delete 旧 key。
5. HTTP API 对启用 project key 的资源，接受任意一把已启用 key；审计记录应尽量反映命中的 key 名称。

## In Scope

- 将 `project.metadata_json.access_config` 升级为支持多把 key 的结构化存储。
- 保持现有 `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/access-config` 路由不变，并在返回结构中加入 key 列表。
- 在 `POST access-config` 中增加最小多 key 管理动作：新增、更新/轮换、禁用、删除。
- 项目级鉴权接受任意一把启用 key，并把命中的 key 名称用于审计 actor。
- CLI 增加最小多 key 管理命令。
- 补 focused tests、build 和本地 smoke。

## Out of Scope

- 不新增数据库表或迁移。
- 不做 Web access-config 管理页面。
- 不做 key 过期时间、usage 计数、last used 时间、IP 白名单或 RBAC。
- 不做 per-key rate limit 或计费。
- 不改 MCP 协议层；MCP 继续通过既有 HTTP/API key 链路受益。

## Acceptance Criteria

- Given 某个 project 已启用一把 key，when 管理员新增第二把 key，then 两把 key 都能访问该 project 级资源。
- Given 某把旧 key 已完成迁移，when 管理员单独 disable 或 delete 该 key，then 其他启用 key 仍可继续访问。
- Given 客户端继续使用旧的单 key `GET/POST access-config` 方式，when 不关心多 key 时，then 旧行为仍兼容可用。
- Given HTTP API 命中了某把已启用 key，when 审计日志记录该请求，then 审计 actor 会优先记录匹配到的 key 名称。
- Given Web managed mode、CLI、MCP、基础限流和现有 provider/input reuse 闭环，when 本片完成后运行现有关键验证，then 不发生回退。

## Technical Approach

- 在 `internal/domain` 为 project access config 增加 `api_keys` 列表结构，同时保留顶层单 key 字段作为兼容视图。
- 在 `internal/app/access.go` 里把 access config 规范化为 canonical multi-key 结构，并支持 `add_key`、`update_key`、`delete_key` 等最小动作。
- 保持 `GET/POST access-config` 路由不变；`POST` 根据请求中的 `action` 和 `api_key_id` 处理多 key 变更。
- 继续使用标准库 `sha256` + 常量时间比较，不引入新的加密依赖。
- 复用现有 metadata_json 持久化路径，不做 schema migration。
- 项目级鉴权在比较 key 时返回匹配到的 key 信息，用于审计 actor 命名。

## Data / Interface Impact

- `project.metadata_json.access_config` 从单 key 扩展为支持 `api_keys` 列表。
- `GET /access-config` 返回中新增 `api_keys` 列表，但保留原有顶层 `api_key_enabled` / `api_key_name` / `api_key_preview` 兼容字段。
- `POST /access-config` 新增 action 型更新能力。
- CLI 新增 `vag project access add-key|update-key|delete-key`。

## Files or Subsystems Likely to Change

- `internal/domain/*`
- `internal/app/access.go`
- `internal/app/access_test.go`
- `internal/store/postgres.go`
- `internal/httpapi/server.go`
- `cmd/vag/main.go`
- `docs/project/*`

## Verification Plan

```bash
docker build --target build -t agent-imageflow-build .
docker run --rm -v /Users/moon/Workspace/tools/agent-imageflow:/src -w /src agent-imageflow-build sh -lc '/usr/local/go/bin/go test ./internal/app ./internal/httpapi ./cmd/vag ./...'
npm --prefix web run build
docker compose config
docker compose up -d --build --force-recreate api worker
# manual smoke:
# 1. 读取 project access config
# 2. 新增第二把 key，并分别用两把 key 访问同一 project 资源
# 3. disable/delete 旧 key，确认新 key 仍可用
# 4. 用 vag audit list 确认审计里出现命中的 key 名称
```

## Assumptions and Risks

- 第一版多 key 仍只面向小团队/单体平台，不追求完整 secret manager 能力。
- 为了兼容现有调用方，顶层单 key 字段会继续保留为兼容视图，这意味着响应里会同时存在“兼容字段 + key 列表”两层表达。
- 本片不追踪 key usage / last_used，因此轮换后的清理仍依赖人工确认。

## Implementation Log

### 2026-06-18

- Changes:
  - 更新 `internal/domain/types.go`，为 `access_config` 增加 `api_keys` 列表结构、兼容视图和多 key action 常量。
  - 更新 `internal/app/access.go`，把 access config 规范化为 canonical multi-key 结构，支持 legacy single-key、`add_key`、`update_key`、`delete_key`，并在鉴权时返回命中的 key 信息。
  - 更新 `internal/store/postgres.go`，读取 `project.metadata_json.access_config` 时自动规范化旧数据与新数据结构。
  - 更新 `internal/httpapi/server.go`，项目级鉴权接受任意一把启用 key，并把命中的 key 名称写入审计 actor。
  - 更新 `cmd/vag/main.go`，新增 `project access add-key|update-key|delete-key` 命令，同时保留 `get|set` 兼容入口。
  - 更新 `internal/app/access_test.go`，补多 key add/update/delete、重复 key 名称和命中启用 key 的 focused tests。
  - 同步更新 README、Project Plan、Tasks、Checkpoints、Decisions 和 Runbook。
- Verification:
  - `docker run --rm -v /Users/moon/Workspace/tools/agent-imageflow:/src -w /src golang:1.25 sh -lc '/usr/local/go/bin/go test ./internal/app ./internal/httpapi ./cmd/vag'`
  - `docker run --rm -v /Users/moon/Workspace/tools/agent-imageflow:/src -w /src golang:1.25 sh -lc '/usr/local/go/bin/go test ./...'`
  - `npm --prefix web run build`
  - `docker compose config`
  - `docker compose up -d --build --force-recreate postgres redis api worker`
  - Docker smoke 已验证 `prj_multi_key_1781784728` 下的 `default` 与 `rollout` 两把 key 都能访问同一 project 资源；disable/delete `default` 后，`rollout` 仍可读取 `task_fc9e1275b4dcb665e766`，且审计记录中出现 `actor=rollout`、`project_api_key_name=rollout`。
- Remaining gaps:
  - 当前只实现多把命名 key 的最小管理动作；尚未提供 key usage、last_used、过期时间、Web 管理 UI 或更细权限模型。
  - MVP 产品缺口已清零，下一条 follow-up 转向本地 Web `.vite/` 运行态目录的 ignore/清理规则。
