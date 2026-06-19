# Story: 004 - OpenAI-compatible provider adapter

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让服务端 Worker 不再只能使用 mock provider，而是可以在配置真实云端 API 后，通过 OpenAI-compatible `images/generations` 形态生成图片，并继续走现有落盘、资产登记、选优和交付闭环。

## Source Context

- Product spec: `docs/project/PRODUCT_SPEC.md`
- Project plan slice: `docs/project/PROJECT_PLAN.md` 的 “Phase 4: First real provider adapter”
- Tech spec: `docs/project/TECH_SPEC.md`
- Architecture: `docs/project/ARCHITECTURE.md`
- Related decisions: `docs/project/DECISIONS.md`

## User Flow

1. 开发者配置 `OPENAI_COMPATIBLE_BASE_URL`、`OPENAI_COMPATIBLE_API_KEY` 和 `OPENAI_COMPATIBLE_MODEL`。
2. 调用方通过 REST、CLI 或 MCP 创建 `provider=openai-compatible` 的图片任务。
3. API 校验 provider 已配置并将任务入队。
4. Worker 调用 OpenAI-compatible provider，解析 base64 或 URL 图片结果。
5. 系统把图片规范化为 PNG，保存原图、缩略图和 metadata，登记 asset/version。
6. 调用方继续用现有查询、select/reject 和 delivery 接口取得结果。

## In Scope

- 抽象 provider adapter 接口，保留 mock provider。
- 新增 OpenAI-compatible provider adapter。
- 支持 `data[].b64_json` 和 `data[].url` 响应。
- 支持下载 URL 图片并规范化为 PNG。
- 将 provider raw response、model、参数和基础 cost/usage 信息写入现有结果结构。
- 增加本地 HTTP mock 单元测试，不调用外部真实 API。
- 更新配置、示例任务和运行文档。

## Out of Scope

- 不做真实外部 API smoke，避免密钥和成本风险。
- 不做 fal.ai、Replicate 或自定义 HTTP provider。
- 不做 reference image、mask/edit、多轮异步 polling。
- 不做 provider routing、成本预算或多 provider 调度。
- 不做 Web managed mode。

## Acceptance Criteria

- Given 未配置 OpenAI-compatible provider，when 创建 `provider=openai-compatible` 任务，then 返回明确的 provider 未启用错误。
- Given 配置了 OpenAI-compatible provider，when Worker 处理任务，then 向 `{base_url}/images/generations` 发送带 Bearer token 的 JSON 请求。
- Given provider 返回 `data[].b64_json`，when Worker 处理完成，then 生成 ready asset_version，并保留 provider/model/parameters/cost/raw response。
- Given provider 返回 `data[].url`，when Worker 处理完成，then 下载图片并完成同样资产登记。
- Given provider 返回 HTTP 错误或不可识别响应，when Worker 处理任务，then 任务进入 failed 或 partially_completed，并记录结构化错误信息。
- Given 默认配置未设置真实 provider，when 运行现有 mock smoke，then 行为不变。

## Technical Approach

- 在 `internal/provider` 中定义 `Adapter` 接口。
- `app.Service` 持有 provider registry，根据 `task.Provider` 选择 adapter。
- `config.Load()` 增加 OpenAI-compatible 配置项和 provider timeout。
- OpenAI-compatible adapter 使用标准库 HTTP client；请求体包含 model、prompt、n、size、response_format、output_format 和兼容 metadata。
- 图片响应统一解码并重新编码为 PNG，避免本地 storage 当前固定 `.png` 路径与 MIME 不一致。

## Data / Interface Impact

- 新增 provider id：`openai-compatible`。
- 新增环境变量：`OPENAI_COMPATIBLE_BASE_URL`、`OPENAI_COMPATIBLE_API_KEY`、`OPENAI_COMPATIBLE_MODEL`、`PROVIDER_TIMEOUT_SECONDS`。
- 新增示例任务：`examples/tasks/sample-openai-compatible-task.json`。
- 不修改数据库 schema。

## Files or Subsystems Likely to Change

- `internal/provider/`
- `internal/app/service.go`
- `internal/config/config.go`
- `examples/tasks/`
- `docs/project/`

## Verification Plan

```bash
docker run --rm -v "$PWD":/src -w /src golang:1.25.3-alpine sh -lc '/usr/local/go/bin/gofmt -w cmd internal && /usr/local/go/bin/go test ./...'
docker compose config
docker compose build
docker compose up -d postgres redis api worker
docker compose exec -T api /app/vag task create --file /app/examples/tasks/sample-image-task.json
```

## Assumptions and Risks

- 第一版 OpenAI-compatible adapter 只覆盖同步 `images/generations`，异步 task polling 留给后续 provider profile。
- 输出统一 PNG，避免当前 storage 扩展名和 MIME 不一致。
- 真实 provider smoke 需要用户自行配置 API key；本 story 只用本地 HTTP mock 自动验证。

## Implementation Log

### 2026-06-18

- Changes: 新增 provider `Adapter` 接口和 OpenAI-compatible provider adapter；服务端 `app.Service` 改为 provider registry；新增真实 provider 配置项、Docker Compose 环境传递和 `sample-openai-compatible-task.json`；adapter 支持 `data[].b64_json`、`data[].url`、URL 下载、PNG 规范化、raw response / cost / parameters 记录；更新 README、Runbook、项目计划、任务、检查点和决策记录。
- Verification: `go test ./...` 通过；`docker compose config` 通过；`docker compose build` 通过；未配置真实 provider 时 `sample-openai-compatible-task.json` 返回明确 HTTP 400；默认 mock provider 回归 smoke 通过；本地 HTTP mock OpenAI-compatible 集成 smoke 通过，Worker 生成 `provider=openai-compatible` ready asset。
- Remaining gaps: 未做真实外部 provider smoke；未做 fal.ai、异步 polling、reference image、edit/mask、多 provider routing；Web 尚未进入服务端托管模式。
