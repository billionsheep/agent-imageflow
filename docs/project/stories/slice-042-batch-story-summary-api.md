# Story: 042 - Batch Story Summary API

## Status

- State: Done
- Created: 2026-06-22
- Updated: 2026-06-22

## Product Goal

Implement the read-only batch/story/scene summary endpoint defined in `slice-041`, so Web production view, CLI/MCP follow-ups and future manifest export can share one backend grouping source.

## Source Context

- Contract: `docs/project/stories/slice-041-batch-story-summary-contract.md`
- Project plan slice: `issues/next-phase-p1-batch-story-export-foundation.csv` / `P1-BSE-003`
- Architecture: `docs/project/ARCHITECTURE.md`

## In Scope

- Add `GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-summary`.
- Require at least one of `session_id` or `batch_id`.
- Support optional `story_id`, `source`, `status`, `include_setup` and `limit`.
- Group scene production tasks by `story_id` and `scene_id`.
- Return story, scene, task and asset summaries with selected coverage counts.
- Map public asset status `draft -> generated` and `approved -> selected`.
- Exclude local absolute paths and secret-bearing configuration from the summary response.

## Out of Scope

- No new database tables or migrations.
- No Web production view implementation.
- No MCP filter parity changes.
- No manifest or ZIP export implementation.
- No real provider calls.
- No API key, provider key, secret, cookie or session inspection.

## Implementation Notes

- Domain DTOs were added for `BatchStorySummaryQuery`, response counts, story rows, scene rows, task rows and asset rows.
- Store implementation reuses `generation_task.structured_input_json.metadata_json` for `source/session_id/batch_id/story_id/scene_id/target_path/scene_order`.
- Store query first limits matching tasks, then joins assets and attempts, so multi-candidate assets do not consume the task limit.
- Default filtering excludes empty `scene_id`, setup roles `setup/reference_setup/visual_context_setup/calibration`, and `exclude_from_story_summary=true`.
- `excluded_setup_task_count` is counted against the same project/campaign/session/batch/story/source/status filter set.
- Summary asset URLs use `/api/assets/{asset_id}/original`, `/thumbnail` and `/metadata`; the new metadata route returns asset metadata without `local_path`.
- HTTP auth, Admin session allowance and audit route inference now include `batch-summary`.

## Verification

- `docker run --rm -v /Users/moon/Workspace/tools/agent-imageflow:/src -w /src golang:1.25.3-alpine sh -c 'gofmt -w internal/domain/types.go internal/store/postgres.go internal/store/postgres_test.go internal/app/service.go internal/httpapi/server.go internal/httpapi/server_test.go internal/httpapi/audit.go internal/httpapi/admin_session.go && go test ./internal/domain ./internal/store ./internal/app ./internal/httpapi'`
- Result: `internal/domain`, `internal/store`, `internal/app` and `internal/httpapi` passed.

## Remaining Risks

- This slice validates query construction and HTTP parsing with unit tests; it does not run a live Postgres smoke batch in this turn.
- `include_setup=true` allows setup-role tasks with a `scene_id` to enter grouping, but empty `scene_id` rows still cannot produce scene rows.
- Future manifest export must reuse this grouping logic rather than rebuilding a separate contract.
