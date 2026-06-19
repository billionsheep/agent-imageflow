# Story: Slice 019 - Best-of Pluggable Scoring

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让 Agent ImageFlow 的自动选优从“只能使用写死的本地元数据启发式”升级到“可配置、可替换的评分策略”。调用方或项目级质量配置可以显式指定 best-of scorer；服务端继续保留 `local_metadata_v1` 作为默认，但也能接一条外部视觉/LLM judge 评分路径，而不改变当前 `generated -> selected/rejected` 轻量状态模型。

## Source Context

- Product spec: 第一版质量优先通过 prompt 模板、style preset、参考图、生成参数和自动选优策略提升，不强制逐张人工审核。
- Project plan slice: `Best-of scoring upgrade` 是当前第一条 pending slice。
- Tech spec: 当前 `selection_mode=auto` / `best_of` 已进入统一任务流，但评分仍是 `local_metadata_v1`；下一步要把它升级为可插拔策略。
- Related decisions: `Best-of 第一版采用本地启发式` 明确指出后续应替换或扩展 scoring strategy，而不是改变资产状态模型。

## User Flow

1. 调用方通过 REST / MCP / CLI / Web 托管任务创建多候选任务，并设置 `selection_mode=auto` 或 `best_of`。
2. 调用方可在任务请求或项目级 quality profile 中提供 `best_of_config`，显式指定评分策略，例如默认本地启发式，或外部 `http_judge_v1`。
3. Worker 生成并登记多个候选资产后，服务端按 `best_of_config` 选择 scorer：
   - 默认使用 `local_metadata_v1`
   - 若指定 `http_judge_v1`，则将候选缩略图/元数据交给外部视觉/LLM judge
4. 服务端将最高分候选标记为 selected；若外部 judge 暂时失败，则回退到 `local_metadata_v1`，避免自动选优链路中断。
5. 调用方查询任务或资产时，仍拿到统一的 selected/generated 状态，不需要理解内部 scorer 细节。

## In Scope

- 在任务输入和项目级 quality profile 中新增 `best_of_config`。
- 把 best-of 评分抽成可注册的 scorer 接口，不再写死在单个函数里。
- 保留 `local_metadata_v1` 作为默认 scorer。
- 新增一条可 mock 的外部评分策略 `http_judge_v1`，用于接视觉/LLM judge。
- 在自动选优 note 中记录 requested/applied strategy 和 fallback 信息。
- 补 focused tests，验证：
  - `best_of_config` 可被 quality profile 复用/覆盖
  - `http_judge_v1` 可选中指定候选
  - 外部 scorer 失败时会回退到 `local_metadata_v1`

## Out of Scope

- 不实现真实 OpenAI / fal 视觉审美 API 直连。
- 不自动 reject 未选中的其他候选。
- 不新增复杂的 Web 打分解释 UI。
- 不改变 `selection_mode`、asset 状态模型或手动 select/reject 接口。
- 不做数据库迁移。

## Acceptance Criteria

- Given 调用方创建 `selection_mode=auto` 或 `best_of` 的任务，when 未提供 `best_of_config`，then 服务端继续使用 `local_metadata_v1` 自动 selected 一张候选。
- Given 调用方在任务请求或项目级 quality profile 中指定 `best_of_config.strategy=http_judge_v1`，when Worker 完成候选登记，then 服务端会调用外部 judge scorer，而不是固定只走本地启发式。
- Given `http_judge_v1` 返回结构化分数或选中的 `asset_id`，when 自动选优发生，then 服务端会把对应候选标记为 selected，并在 note 中记录 applied strategy。
- Given `http_judge_v1` 临时失败，when 自动选优发生，then 服务端会回退到 `local_metadata_v1`，而不是让任务停留在“多候选但无推荐图”的状态。
- Given `best_of_config` 通过 quality profile 保存，when REST/MCP/Web 托管任务复用项目级质量配置，then `best_of_config` 不会丢失。

## Technical Approach

- 在 `internal/domain` 中新增 `BestOfConfig`，挂到 `CreateTaskRequest` 与 `QualityProfile`。
- 在 `internal/app/quality.go` 中补 `best_of_config` 的归一化、quality profile 复用和 snapshot。
- 在 `internal/app/bestof.go` 中引入 scorer registry：
  - `local_metadata_v1`
  - `http_judge_v1`
- `http_judge_v1` 通过配置的 HTTP endpoint 发送候选缩略图 data URL、任务信息和 `judge_prompt`，使用结构化 JSON 响应返回分数。
- `Service.autoSelectBestAsset(...)` 记录 requested/applied strategy；外部 scorer 失败时回退到 `local_metadata_v1`。
- `best_of_config` 写入 `structured_input_json`，MCP schema 与 Web/REST types 同步透传。

## Data / Interface Impact

- `CreateTaskRequest` 新增可选 `best_of_config`。
- `QualityProfile` 新增可选 `best_of_config`，支持项目级默认 scorer。
- `structured_input_json` 与 quality snapshot 会保留 `best_of_config`。
- 不新增数据库字段。

## Files or Subsystems Likely to Change

- `internal/domain/types.go`
- `internal/config/config.go`
- `internal/app/service.go`
- `internal/app/quality.go`
- `internal/app/quality_test.go`
- `internal/app/bestof.go`
- `internal/app/bestof_test.go`
- `internal/mcp/server.go`
- `web/src/lib/agentImageflowApi.ts`
- `docker-compose.yml`
- `docs/project/*`

## Verification Plan

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./internal/app ./internal/mcp'
docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'
docker compose config

# manual smoke:
# 1. 启动本地 http_judge_v1 mock
# 2. 注入 BEST_OF_HTTP_SCORER_URL 后重启 api/worker
# 3. 创建 selection_mode=auto + best_of_config.strategy=http_judge_v1 的 mock 任务
# 4. 轮询任务完成，并检查 mock 收到 scorer 请求、任务恰好 1 张 selected
```

## Assumptions and Risks

- 第一版外部 scorer 使用通用 HTTP judge 适配器，而不是直接绑定某一家视觉模型 API；真实视觉/LLM 能力可放在适配器后面演进。
- 为控制 payload，大图默认使用服务端已生成的缩略图 data URL 参与外部 judge，而不是再次上传原图。
- 外部 judge 是增强项，不应让自动选优链路成为单点故障；因此本片允许回退到 `local_metadata_v1`。

## Implementation Log

### 2026-06-18

- Changes:
  - 在 `internal/domain/types.go` 中新增 `BestOfConfig`、`best_of_config` 归一化和 `local_metadata_v1` / `http_judge_v1` 策略常量，并把它接入 `CreateTaskRequest` 与 `QualityProfile`。
  - 在 `internal/app/quality.go` 中补齐 `best_of_config` 的 request/profile 归一化、项目级 quality profile 复用与 quality snapshot 写入；`structured_input_json` 现在会保留有效 `best_of_config`。
  - 在 `internal/app/bestof.go` / `internal/app/bestof_http.go` 中把自动选优抽成 scorer registry，保留 `local_metadata_v1`，新增 `http_judge_v1` HTTP judge adapter；外部 scorer 失败时自动回退到本地启发式，并在 `review_event.note` 中记录 requested/applied/fallback 信息。
  - 在 `internal/app/service.go` 中按配置注册可用 scorer，并在 `selection_mode=auto` / `best_of` 时校验所选策略是否可用。
  - 在 `internal/mcp/server.go`、`web/src/lib/agentImageflowApi.ts`、`docker-compose.yml` 和 `examples/tasks/sample-best-of-http-judge-task.json` 中同步暴露 `best_of_config` 与 HTTP judge smoke 配置。
- Verification:
  - `docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./internal/app ./internal/mcp'`
  - `docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'`
  - `docker compose config`
  - 本地 HTTP judge smoke：以 `BEST_OF_HTTP_SCORER_URL=http://host.docker.internal:8789/score` 重启 `api/worker`，创建 `sample-best-of-http-judge-task.json` 对应任务；成功验证 `task_9f3b4f5551fbdf5b8e06 -> asset_1837f5fd3e8e6977dcb3`，mock judge 收到候选缩略图 `data:` URL，请求与 `review_event.note` 中的 `requested_strategy=http_judge_v1`、`applied_strategy=http_judge_v1` 一致。
- Remaining gaps:
  - 当前仍未自动 reject 未入选候选；下一片转向可选 auto reject 逻辑。
  - 更强的 scorer 生产硬化能力（限流、审计、多 key 策略）仍待后续 hardening。
