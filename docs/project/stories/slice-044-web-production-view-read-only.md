# Story: 044 - Web Production View Read-only

## Status

- State: Done
- Created: 2026-06-22
- Updated: 2026-06-22

## Product Goal

Give Web users a minimal read-only production view for one batch/story/scene set, backed by the shared `batch-summary` API from `slice-041` and `slice-042`.

## Source Context

- Contract: `docs/project/stories/slice-041-batch-story-summary-contract.md`
- API implementation: `docs/project/stories/slice-042-batch-story-summary-api.md`
- Project plan slice: `issues/next-phase-p1-batch-story-export-foundation.csv` / `P1-BSE-005`

## In Scope

- Add a lazy-loaded Web entry for Batch / Story / Scene production view.
- Query `GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-summary`.
- Support `session_id`, `batch_id`, optional `story_id`, `source`, `status`, `include_setup` and `limit`.
- Show counts, stories, scenes, scene status, task and asset counts, selected coverage, thumbnails, delivery/metadata links, latest task id and error summary.
- Preserve the surrounding app while loading or refreshing; use local modal loading/error/unauthorized/empty states.

## Out of Scope

- No select/reject actions in the production view.
- No scene regenerate.
- No manifest or ZIP export.
- No backend, MCP or provider changes.
- No real provider calls.
- No API key, provider key, secret, cookie or session inspection.

## Implementation Notes

- Web API client now includes `AgentImageflowBatchStorySummary*` types, `buildAgentImageflowBatchStorySummaryUrl` and `getAgentImageflowBatchStorySummary`.
- `ProductionViewModal` is lazy-loaded from the top header and reuses existing Web settings for API base URL, Basic fallback and project API key headers; Admin session continues to work through the existing `credentials: include` request helper.
- The view keeps previous summary results visible during refresh and shows a local refreshing banner instead of blanking the full app.
- 401/403 are shown as `unauthorized / login required`, not as an empty scene list.
- Asset statuses are normalized defensively through the existing `draft -> generated` and `approved -> selected` mapping.

## Verification

- `npm --prefix web test -- --run`
  - Result: 17 files / 226 tests passed.
- `npm --prefix web run build`
  - Result: passed; Vite still reports the existing large chunk warning family. `ProductionViewModal` is emitted as a separate lazy chunk.

## Remaining Risks

- This turn did not run a browser preview smoke, so final visual spacing is covered by TypeScript/build but not by screenshot inspection.
- The production view depends on an existing `batch-summary` backend being available for the configured Web API host.
