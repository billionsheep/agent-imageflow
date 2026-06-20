# Slice 027: Concurrency And Benchmark

## Product Goal

把 Agent ImageFlow 的服务端生图链路从实际串行推进为可控并发，并提供可重复 benchmark 数据，定位平台自身吞吐、provider timeout/retry 和真实 API 并发边界。

## User Flow

1. 运维者通过环境变量设置 `WORKER_CONCURRENCY` 和 `OPENAI_COMPATIBLE_MAX_CONCURRENCY`。
2. Worker 并发消费 Redis 队列，但 openai-compatible provider 调用受 provider cap 保护。
3. 用户或排障者通过 API/CLI/Web 查看任务 attempts、provider latency、retry_after 和失败原因。
4. 运维者运行 mock benchmark 验证平台自身吞吐，不产生费用。
5. 运维者在明确费用风险后运行小样本真实 provider benchmark，比较并发 1/2/4 的成功率、P50/P95、timeout 和 retry。

## In Scope

- `OPENAI_COMPATIBLE_MAX_CONCURRENCY` provider 级 backpressure。
- `GET /api/tasks/{task_id}/attempts` 和 `vag task attempts <task_id>`。
- Web 托管任务详情展示 attempts / retry / timeout 摘要。
- openai-compatible `/images/generations` 与 `/images/edits` 透传 `quality`、`moderation`、`output_compression`。
- `vag benchmark image-generation` 小样本 benchmark 命令，真实 provider 默认费用保护。
- 文档记录优化前后对比方法、推荐并发判定和 Responses/streaming 设计结论。

## Out Of Scope

- 不实现 Responses API image_generation adapter。
- 不接 streaming partial images 状态机。
- 不做大规模真实 provider 压测。
- 不读取、打印、提交任何 API key 或 secret。
- 不推进 Reference Library、角色库或运营系统能力。

## Acceptance Criteria

- `WORKER_CONCURRENCY` 可通过 Compose 环境变量覆盖，worker 启动日志能反映实际并发。
- `OPENAI_COMPATIBLE_MAX_CONCURRENCY` 默认 2，允许设为 0 禁用 cap。
- attempt API/CLI 只返回脱敏诊断字段，不返回 `raw_response_json`、Authorization 或 API key。
- Web 托管任务详情能看到最近 attempts、latency、retry_after、error_code 和 error_message。
- Benchmark 输出包含成功率、平均耗时、P50/P95、timeout、retry、queue wait 和每个任务摘要。
- 真实 provider benchmark 未显式 `--allow-paid-provider` 时拒绝执行。

## Technical Approach

- 复用现有 `task_attempt` 表，不新增迁移。
- 在 application service 的 provider 调用外层加按 provider 的 channel semaphore。
- Benchmark 直接复用 application core 创建任务和轮询任务，避免绕过服务端资产闭环。
- Web 轮询 task 时并行读取 attempts；attempt 读取失败不影响主任务状态。

## Data / Interface Impact

- 新增只读 REST：`GET /api/tasks/{task_id}/attempts`。
- 新增 CLI：`vag task attempts <task_id>`、`vag benchmark image-generation ...`。
- 新增配置：`OPENAI_COMPATIBLE_MAX_CONCURRENCY`。
- 前端 `TaskRecord` 新增可选 `imageflowAttempts` 字段。

## Verification

- `go test ./...`
- `npm --prefix web test -- --run`
- `npm --prefix web run build`
- `docker compose config`
- mock benchmark：并发 1/2/4 分档执行，比较总 wall-clock 和 P95。
- real provider benchmark：仅用户确认费用后执行小样本。

## Assumptions And Risks

- openai-compatible provider 仍走同步 Images API 最终图接口，因此单个 attempt latency 主要由 provider 出图耗时决定。
- provider cap 过高可能触发 429 或 timeout；默认从 2 起步。
- 当前本轮先解决平台吞吐，Responses API / streaming partial images 只作为后续设计评估。

## Implementation Log

- Added `OPENAI_COMPATIBLE_MAX_CONCURRENCY` with default `6`; `0` disables the cap.
- Set default `WORKER_CONCURRENCY` to `6` in config and Compose.
- Added provider-level semaphore around `adapter.Generate(...)` for `openai-compatible`.
- Added `GET /api/tasks/{task_id}/attempts` and `vag task attempts <task_id>`; response excludes raw provider response and secrets.
- Added Web managed task detail attempts summary.
- Passed `quality`、`moderation`、`output_compression` from `generation_config` to openai-compatible generation/edit requests.
- Added `vag benchmark image-generation` with mock and real-provider safeguards; real provider requires `--allow-paid-provider` and is capped at 8 tasks per run.
- Added `mock_delay_ms` for no-cost provider latency simulation.
- Verification:
  - `go test ./...` passed in `golang:1.25.3-alpine`.
  - `npm --prefix web test -- --run` passed: 17 files / 224 tests.
  - `npm --prefix web run build` passed with existing chunk-size warning.
  - `docker compose config` passed and showed `WORKER_CONCURRENCY` override plus `OPENAI_COMPATIBLE_MAX_CONCURRENCY`.
  - Attempt API/CLI smoke passed on task `task_c3c9a0ff7cbae96a427d`.
  - Mock benchmark baseline: worker=1, 32 tasks, `mock_delay_ms=250`, wall-clock `12.427s`, P95 queue wait `10.987s`.
  - Mock benchmark optimized: worker=4, 32 tasks, `mock_delay_ms=250`, wall-clock `2.979s`, P95 queue wait `2.464s`.
  - Mock `requested_count=4`: worker=4, 8 tasks, `mock_delay_ms=250`, wall-clock `1.318s`, 8/8 completed.
  - Mock worker=6 max sample: 32 tasks, `requested_count=4`, `mock_delay_ms=2000`, wall-clock `14.239s`, 32/32 completed, 128 assets produced.
  - Real provider benchmark after explicit confirmation: worker=6/provider cap=6, 6 tasks, `requested_count=1`, wall-clock `120.628s`, 4/6 completed, 2/6 timeout, worker memory peak about `50.60MiB`, API memory peak about `26.48MiB`.
  - Real provider c6 does not meet the recommendation threshold; next paid sample should test c2/c4 before using c6 for production.
