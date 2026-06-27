# Slice 056 - Agent-Friendly Setup Contract

## 背景

`v0.2 MCP Production Hardening` 的第一个 P0 问题不是生图本身，而是新 agent 在真实业务试跑前，仍然需要依赖 Admin Web session、手工点控制台，甚至本地 DB 直连，才能准备 `workspace/project/campaign/visual-context`。这和 Agent ImageFlow 作为 MCP/API-first 图片资产平台的定位不一致。

## 决策

- 不新增 destructive MCP tool。
- 不让新 agent 依赖 Admin cookie/session。
- 在现有 REST bootstrap 路由上增加受控 setup auth：服务端环境变量 `AGENT_SETUP_TOKEN`，请求头 `X-Agent-Setup-Token`。
- 该 token 只允许访问非 destructive 准备路径：
  - `GET/POST /api/workspaces`
  - `GET/POST /api/workspaces/{workspace_id}/projects`
  - `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/campaigns`
  - `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/visual-context`
- 明确不放开：
  - `POST /api/tasks`
  - asset select/reject/delivery
  - archive/restore
  - cleanup
  - 任何 delete 路由

## 实现

### 服务端配置

- `internal/config.Config` 新增 `AgentSetupToken`。
- `cmd/api` 把 `AGENT_SETUP_TOKEN` 传入 `httpapi.Options`。
- `.env.example.prod` 新增注释和占位项，明确这是可选、非 destructive 的 bootstrap token。

### HTTP 鉴权

- `internal/httpapi.Server` 新增 `authorizeAgentSetupToken`。
- `internal/httpapi.Server` 新增 `routeAllowsAgentSetupToken`，只允许上述 bootstrap 路由。
- `authorizeRequest` 在 Admin session 校验之后、常规 project/basic auth 之前，判断 setup token 是否可用于当前路由。
- 授权成功时，actor 记录为 `agent_setup_token`，便于后续审计和问题归因。
- CORS 允许 `X-Agent-Setup-Token`。

### CLI 配合

- `cmd/vag` 在检测到环境变量 `AGENT_IMAGEFLOW_SETUP_TOKEN` 时，会自动附带 `X-Agent-Setup-Token`，让 agent 或脚本可直接通过 CLI 完成 bootstrap。

## 测试

新增/调整测试：

- `internal/config/config_test.go`
  - 验证 `AGENT_SETUP_TOKEN` 能被正确加载。
- `internal/httpapi/server_test.go`
  - 验证 setup token 仅允许非 destructive bootstrap 路由。
  - 验证可读取 project visual context。
  - 验证不能借 setup token 创建 image task。
  - 验证 CORS 暴露 `X-Agent-Setup-Token`。

容器化验证：

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine go test ./internal/config ./internal/httpapi ./cmd/api ./cmd/vag
```

## 结果

- 新 agent 现在可以在不使用 Admin cookie/session、DB 直连或 provider key 的前提下，准备 project/campaign/context。
- MCP 当前 6 个安全工具保持不变，没有被误扩成 setup/delete 混合入口。
- setup token 仍然不是“万能 agent token”，它不会越权到任务创建、资产生命周期或清理链路。

## 后续

下一个独立闭环切片是 `V02-MCPH-003 Project Visual Context reference diagnostics`。setup contract 已经把“能不能安全准备上下文”的问题拆清，接下来需要解决“上下文是否足够强、是否存在狗变熊风险”的诊断问题。
