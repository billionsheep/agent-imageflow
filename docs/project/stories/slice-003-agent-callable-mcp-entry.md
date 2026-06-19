# Story: 003 - Agent-callable MCP entry

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让 Codex、Claude、Cursor 等 MCP client 可以通过 stdio tool 调用现有服务端资产闭环：创建图片任务、查询任务、列出候选资产、标记推荐/拒绝，并获取交付信息。

## Source Context

- Product spec: `docs/project/PRODUCT_SPEC.md`
- Project plan slice: `docs/project/PROJECT_PLAN.md` 的 “Phase 3: Agent-callable MCP”
- Tech spec: `docs/project/TECH_SPEC.md`
- Related decisions: `docs/project/DECISIONS.md`

## User Flow

1. 开发者启动 PostgreSQL、Redis、API/Worker 或直接运行 MCP command。
2. MCP client 通过 stdio 初始化 Agent ImageFlow MCP server。
3. MCP client 发现 `create_image_task` 等 tools。
4. MCP client 创建 mock 图片任务并得到 `task_id`。
5. Worker 处理任务后，MCP client 查询任务、列出资产、标记 selected/rejected，并取得 delivery JSON。

## In Scope

- 新增 stdio MCP server command。
- 暴露 `create_image_task`、`get_image_task`、`list_image_assets`、`select_image_asset`、`reject_image_asset`、`get_asset_delivery_info`。
- MCP tool 复用现有 `app.Service`，不绕过 PostgreSQL、Redis、Worker 和资产状态机。
- MCP 层使用 `generated/selected` 产品语义；底层 `draft/approved` 保持兼容。
- 增加最小协议测试和运行说明。

## Out of Scope

- 不接真实 provider。
- 不做 Streamable HTTP MCP。
- 不做 Web managed mode。
- 不迁移数据库状态字段或表名。
- 不做 best-of 自动选优。

## Acceptance Criteria

- Given MCP client 发送 `initialize`，when server 响应，then 返回 server info 和 tools capability。
- Given MCP client 调用 `tools/list`，when 请求成功，then 返回六个 Agent ImageFlow tools 和 JSON Schema 输入定义。
- Given MCP client 调用 `create_image_task`，when 输入有效，then 复用现有服务创建 queued task 并入队。
- Given 已生成资产，when 调用 `select_image_asset` 或 `reject_image_asset`，then 返回 selected/rejected 语义状态。
- Given 任意 tool 执行成功，when 返回 MCP result，then 同时包含 `structuredContent` 和 JSON 文本 content。
- Given tool 执行失败，when 返回 MCP result，then `isError=true` 且错误文本可读。

## Technical Approach

- 新增 `internal/mcp` 包，实现换行分隔 JSON-RPC stdio handler。
- 使用官方 MCP 核心方法：`initialize`、`notifications/initialized`、`tools/list`、`tools/call` 和 `ping`。
- `cmd/mcp` 与 API/Worker 一样加载配置、迁移数据库、seed 默认 workspace/project/campaign、连接 Redis，并构造同一个 `app.Service`。
- 不新增第三方依赖，使用 Go 标准库实现协议薄封装。

## Data / Interface Impact

- 新增 MCP command binary：`mcp`。
- Docker image 内新增 `/app/mcp`。
- MCP tool 输入默认使用 `DEFAULT_WORKSPACE_ID`、`DEFAULT_PROJECT_ID`、`DEFAULT_CAMPAIGN_ID`，调用方可显式覆盖。
- MCP 输出把底层 `draft` 映射为 `generated`，`approved` 映射为 `selected`。

## Files or Subsystems Likely to Change

- `cmd/mcp/`
- `internal/mcp/`
- `Dockerfile`
- `docs/project/`
- `README.md`

## Verification Plan

```bash
docker run --rm -v "$PWD":/src -w /src golang:1.25.3-alpine sh -lc '/usr/local/go/bin/gofmt -w cmd internal && /usr/local/go/bin/go test ./...'
docker compose config
docker compose build
docker compose up -d postgres redis api worker
# MCP stdio smoke: initialize -> tools/list -> create task -> poll get task -> select/reject -> get delivery info
```

## Assumptions and Risks

- 本机 shell 当前没有 `go` 命令，Go 验证通过 Docker 镜像执行。
- MCP server 进程需要能访问 PostgreSQL 和 Redis；第一版不提供离线 MCP mock mode。
- 现有底层状态仍是 `draft/approved`，MCP 层负责展示新产品语义。

## Implementation Log

### 2026-06-18

- Changes: 新增 `internal/mcp` stdio JSON-RPC server、`cmd/mcp` command、Docker image `/app/mcp` binary；暴露六个 MCP tools；MCP 层将 `draft/approved` 兼容状态映射为 `generated/selected` 产品语义；更新 README、Runbook、项目计划、任务、检查点和决策记录。
- Verification: `docker run --rm -v "$PWD":/src -w /src golang:1.25.3-alpine sh -lc '/usr/local/go/bin/gofmt -w cmd internal && /usr/local/go/bin/go test ./...'` 通过；`docker compose config` 通过；`docker compose build` 通过；真实 MCP stdio smoke 通过，跑通 initialize -> tools/list -> create task -> get task -> select asset -> get delivery info；旧 CLI `vag asset approve` 回归通过。
- Remaining gaps: 真实 provider adapter 未接入；Web 尚未进入服务端托管模式；MCP 当前只支持 stdio，不支持 Streamable HTTP。
