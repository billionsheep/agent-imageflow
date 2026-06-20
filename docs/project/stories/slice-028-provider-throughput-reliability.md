# Slice 028: P1 Provider Throughput & Reliability

## Status

Done on 2026-06-19.

## Goal

治理真实 provider 并发不稳、timeout 粒度不足、阶段耗时不可观测、benchmark 诊断不足和 batch progress 不清晰的问题。

本 slice 不做 streaming、partial images、adaptive backpressure、复杂 provider probe，也不扩展为自媒体运营系统、内容日历、自动发布或通用 DAM。

## Scope

- 保留 `WORKER_CONCURRENCY=6`，将真实 provider 默认入口 cap 收敛为 `OPENAI_COMPATIBLE_MAX_CONCURRENCY=3` 和 `FAL_MAX_CONCURRENCY=3`。
- 将默认 provider timeout 提升为 `300s`，并给 openai-compatible 增加 connect/header/total timeout profile。
- `task_attempt` 增加 `queue_wait_ms`、`provider_first_byte_ms`、`provider_total_ms`、`response_download_ms`、`store_ms`、`thumbnail_ms`、`retry_count`、`error_stage`、`response_bytes`。
- `provider_profile` 增加非敏感 capability 字段：`max_n`、`supports_url_result`、`preferred_response_format`、`max_concurrency`、`timeout_seconds`。
- 同 prompt 多图按 `provider_profile.max_n` 合并或拆分 provider 请求；多 scene 仍由外部编排为独立 task 并保留 metadata。
- `vag benchmark image-generation` 输出 provider cap、timeout profile、queue/provider/download/store/thumbnail、retry/timeout/error_stage 和调参建议。
- 增加最小 batch progress：`GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-progress` 与 `vag batch progress`。

## Verification

- Docker `gofmt -w cmd internal && go test ./...` passed.
- `npm --prefix web test -- --run` passed: 17 files / 224 tests.
- `npm --prefix web run build` passed with existing chunk-size warning.
- `docker compose config` passed.
- Mock benchmark `bench_p1_provider_rel_batch` passed: 3 tasks, 6 assets, 0 retry, 0 timeout.
- Batch progress for `bench_p1_provider_rel_batch` returned `task_count=3`, `succeeded_count=3`, `asset_count=6`, `attempt_count=3`.
- Task attempts for `task_6330ef96181ccb074ca5` showed `queue_wait_ms`, `provider_total_ms`, `store_ms`, `thumbnail_ms`, and `retry_count`.

## Notes

- Real provider benchmark was not run in this slice because it has `needs_confirmation=yes` and may incur cost.
- Project API key auth remained in force for batch progress routes.
- No provider secret was added to project profile, responses, docs, CSV evidence, or story docs.
