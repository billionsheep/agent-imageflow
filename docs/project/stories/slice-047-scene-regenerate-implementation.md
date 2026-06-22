# Story: 047 - Scene Regenerate Implementation

## Status

- State: Done
- Created: 2026-06-22
- Updated: 2026-06-22

## Product Goal

让用户在 Batch / Story / Scene Production View 中对单个失败、效果差或想继续探索的 scene 创建新的生成任务，而不是重跑整批故事图；旧 task、旧 assets、selected/rejected 状态保持不变，新 task 通过 metadata lineage 可追溯。

## Source Context

- Design story: `docs/project/stories/slice-046-scene-regenerate-design.md`
- Summary contract: `docs/project/stories/slice-041-batch-story-summary-contract.md`
- Summary API: `docs/project/stories/slice-042-batch-story-summary-api.md`
- Web production view: `docs/project/stories/slice-044-web-production-view-read-only.md`
- Scene asset actions: `docs/project/stories/slice-045-scene-asset-actions.md`
- Project plan slice: `issues/next-phase-p1-batch-story-export-foundation.csv` / `P1-BSE-008`

## In Scope

- Add REST action `POST /api/projects/{project_id}/campaigns/{campaign_id}/scene-regenerations`.
- Support regenerate from `source_task_id`, with backend support for scene identity latest-task resolution.
- Copy source task prompt, visual context ids, reference descriptors, quality/profile flags, generation config and scene metadata into a new task request.
- Merge safe overrides in the service layer.
- Write metadata lineage: `regenerated_from_task_id`, `regenerate_no`, reason/source/actor/time, overrides and source scene identity.
- Reuse existing `CreateTask` queue/worker path.
- Add minimal Web Production View scene-level Regenerate action using `latest_task_id`.
- Keep existing selected/rejected assets unchanged.

## Out of Scope

- No real provider smoke.
- No CLI/MCP regenerate command in this slice.
- No prompt/recipe/quality override UI in Web.
- No visual version tree.
- No automatic selected replacement.
- No cross-scene batch regenerate.
- No schema migration.
- No key, provider secret, cookie or session token inspection.

## Implementation Notes

- `internal/domain/types.go` now defines scene regeneration request/response DTOs, scene identity, safe overrides and warnings.
- `internal/store/postgres.go` can read the source task, resolve latest task by scene identity, and count existing regeneration tasks for the same scene.
- `internal/app/service.go` adds `RegenerateSceneTask`, validates project/campaign ownership, derives scene metadata, builds a new `CreateTaskRequest`, applies safe overrides, writes metadata lineage and calls `CreateTask`.
- `internal/httpapi/server.go` exposes `POST /api/projects/{project_id}/campaigns/{campaign_id}/scene-regenerations`; admin session allowance and audit route/action were added.
- `web/src/lib/agentImageflowApi.ts` adds scene regeneration URL builder, request/response types and client helper.
- `web/src/components/ProductionViewModal.tsx` adds per-scene reason input, Regenerate button, pending/error/success state, selected-preserved hint and post-success summary refresh without closing or clearing the modal.

## Acceptance

- A scene can create a new task from `latest_task_id` through Web Production View.
- The new task keeps the same project/campaign/session/batch/story/scene lineage.
- The source task and old assets are not mutated.
- Existing selected/rejected assets are not automatically changed.
- The new task metadata records `regenerated_from_task_id` and `regenerate_no`.
- Batch summary can include the new task naturally through existing scene metadata grouping.
- The modal keeps old summary visible while refreshing after success.

## Verification

- `docker run --rm -v "$PWD":/src -w /src -e GOCACHE=/tmp/gocache -e GOMODCACHE=/tmp/gomodcache golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./internal/domain ./internal/store ./internal/app ./internal/httpapi ./internal/mcp'`
  - Result: passed.
- `npm --prefix web test -- --run`
  - Result: 17 files / 227 tests passed.
- `git diff --check`
  - Result: passed.

## Remaining Risks

- This slice did not run a real provider smoke and did not create a live DB task through authenticated Web, by design.
- CLI and MCP regenerate commands remain future follow-ups; external agents can use the REST action first.
- Web override UI is intentionally deferred until the basic regenerate path is exercised.

## Implementation Log

### 2026-06-22

- Backend REST/core and Web Production View regenerate action implemented through supervised subagents.
- PM integration verified Go, MCP and Web tests.
- No real provider run and no API key / provider key / secret / cookie / session token read or printed.
