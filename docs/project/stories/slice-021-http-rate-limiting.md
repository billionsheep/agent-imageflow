# Story: Slice 021 - HTTP Rate Limiting

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让 Agent ImageFlow 的 HTTP API 在自托管或对公网暴露时，至少具备一层基础的实例级和 project 级限流保护，避免单个入口或单个项目在短时间内无限放大请求量；同时不破坏现有 Basic Auth、项目级 API key、Web 托管模式、CLI 和正常的服务端任务流。

## Source Context

- Product spec: 第一版已确认是能力平台，默认提供 REST API 给自动化系统、Web 和 CLI 使用。
- Project plan slice: `Production hardening` 当前第一条待做工作被细化为“第一版实例级 / project 级限流能力”。
- Tech spec: 当前 API、Worker、CLI、MCP 共用同一套 application core，Redis 已作为现有运行时依赖。
- Related decisions: `第一版项目鉴权采用实例级 Basic Auth + 项目级单 key`、`生产入口默认交给反向代理，保持 compose 为开发友好`。

## User Flow

1. 运维或开发者通过环境变量为 API 配置基础限流阈值。
2. Web managed mode、CLI 或外部系统通过 HTTP API 正常访问 Agent ImageFlow。
3. 当整体请求量超过实例级阈值时，服务端返回明确的 `429` 限流错误。
4. 当某个 project 的请求量超过 project 级阈值时，服务端只对该 project 返回明确的 `429` 限流错误。
5. MCP stdio、本地 Worker 和已有资产处理闭环不受这一层 HTTP 限流直接影响。

## In Scope

- 增加 API 级基础限流配置，默认关闭。
- 增加实例级限流。
- 增加 project 级限流。
- 限流命中时返回 `429`、结构化错误 JSON 和 `Retry-After`。
- 复用现有 Redis 运行时，不新增第三方依赖。
- 补 focused tests、build 和本地 smoke。

## Out of Scope

- 不实现复杂的用户级、IP 级或 endpoint 级限流策略。
- 不实现审计日志、配额面板或限流可视化页面。
- 不给 MCP stdio 增加独立限流。
- 不做多节点全局一致性优化之外的复杂治理。
- 不进入多 key 策略或完整 API usage 计费。

## Acceptance Criteria

- Given 未配置限流阈值，when HTTP API 正常接收请求，then 当前行为不变，不会平白返回 `429`。
- Given 配置了实例级限流阈值，when 同一窗口内总请求数超过阈值，then API 返回 `429`、结构化 `error_code/error_message` 和 `Retry-After`。
- Given 配置了 project 级限流阈值，when 同一窗口内某个 project 的请求数超过阈值，then 只有该 project 的请求被限流，其他 project 或无 project 作用域的请求不受该 project 计数影响。
- Given 限流命中，when Web managed mode 或 CLI 继续通过 HTTP API 调用，then 它们会收到明确错误，而不是静默失败或服务端崩溃。
- Given Redis 限流后端发生瞬时错误，when API 处理请求，then 服务端记录日志并 fail-open，不把限流组件故障放大成整体 API 不可用。

## Technical Approach

- 在 `internal/config` 中新增 rate limit 配置项，使用环境变量控制，默认关闭。
- 在 `internal/httpapi` 中新增基于 Redis 的简单固定窗口限流器。
- 限流只挂在 HTTP API 入口；`/healthz`、CORS preflight 和 MCP stdio 不进入这一层。
- 先完成 Basic Auth / project scope 解析，再按实例级和 project 级顺序执行限流。
- Redis 计数使用原子 `INCR + PEXPIRE` Lua/Eval 模式，避免窗口初始化竞争。
- 限流后端出错时记录日志并放行；只有真正超限才返回 `429`。

## Data / Interface Impact

- 新增环境变量，例如实例级阈值、project 级阈值和限流窗口秒数。
- HTTP API 新增 `429 Too Many Requests` 返回语义。
- 不新增数据库字段和表。

## Files or Subsystems Likely to Change

- `internal/config/config.go`
- `internal/httpapi/server.go`
- `internal/httpapi/*`
- `cmd/api/main.go`
- `docker-compose.yml`
- `docs/project/*`

## Verification Plan

```bash
docker build --target build -t agent-imageflow-build .
docker run --rm agent-imageflow-build sh -lc 'cd /src && /usr/local/go/bin/go test ./...'
npm --prefix web run build
docker compose config
RATE_LIMIT_WINDOW_SECONDS=60 RATE_LIMIT_INSTANCE_MAX_REQUESTS=2 docker compose up -d --force-recreate api
# manual smoke:
# 1. 连续请求 3 次 GET /api/workspaces，确认第 3 次返回 429
# 2. 配 project 级阈值后，对同一 project 连续请求 task/create 或 asset/get，确认命中 429
```

## Assumptions and Risks

- 第一版限流默认关闭，避免影响当前本地开发和既有 smoke。
- 先做固定窗口限流足够满足 MVP hardening 的第一步，不追求复杂令牌桶或精细配额。
- Redis 已是当前运行时依赖，复用它比引入本地内存限流更贴近自托管实际部署。
- Redis 限流组件故障时选择 fail-open，是为了优先保证 API 可用性；后续如果进入更严格公网场景，再评估 fail-closed 或更细告警策略。

## Implementation Log

### 2026-06-18

- Changes:
  - `internal/config/config.go` 增加 `RATE_LIMIT_WINDOW_SECONDS`、`RATE_LIMIT_INSTANCE_MAX_REQUESTS`、`RATE_LIMIT_PROJECT_MAX_REQUESTS` 配置读取。
  - `internal/httpapi/ratelimit.go` 新增 Redis 固定窗口限流器；`internal/httpapi/server.go` 在认证成功后增加实例级 / project 级限流，并返回 `429`、结构化错误 JSON、`Retry-After`。
  - `cmd/api/main.go` 在 `api` 进程内按需创建 Redis limiter；`docker-compose.yml` 补对应环境变量透传。
  - `internal/httpapi/server_test.go` 增加 focused tests，覆盖实例级限流、project 级限流、无 project scope 时跳过 project 计数，以及 Redis 后端 fail-open。
- Verification:
  - `docker build --target build -t agent-imageflow-build .`
  - `docker run --rm agent-imageflow-build sh -lc 'cd /src && /usr/local/go/bin/go test ./...'`
  - `npm --prefix web run build`
  - `docker compose config`
  - Docker smoke:
    - 实例级: `RATE_LIMIT_INSTANCE_MAX_REQUESTS=2` 下，`GET /api/workspaces` 在同一窗口内返回 `429` 与 `Retry-After`；本地因并行 API 流量，实际观测序列为 `200 -> 429 -> 429`。
    - project 级: 独立 `prj_rate_limit_smoke/cmp_rate_limit_smoke` 下，`POST /tasks` 观测序列为 `201 -> 429`。
    - 回归: `docker compose up -d --force-recreate api` 恢复默认无阈值配置后，`curl -sf http://localhost:8081/healthz` 正常。
- Remaining gaps:
  - 当前基础限流只覆盖 HTTP API，不含 MCP stdio 独立限流、审计日志、多 key 配额、IP 级 / endpoint 级策略和可视化面板。
