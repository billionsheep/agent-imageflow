# Story: 045 - Scene Asset Actions

## Status

- State: Done
- Created: 2026-06-22
- Updated: 2026-06-22

## Product Goal

Allow users to select or reject candidate assets directly inside the Web Production View scene cards, so a batch/story can be reviewed without jumping back to the general asset library.

## Source Context

- Contract: `docs/project/stories/slice-041-batch-story-summary-contract.md`
- Previous Web slice: `docs/project/stories/slice-044-web-production-view-read-only.md`
- Project plan slice: `issues/next-phase-p1-batch-story-export-foundation.csv` / `P1-BSE-006`

## In Scope

- Reuse existing server-side asset review endpoints from Web:
  - Select uses `POST /api/assets/{asset_id}/approve` through the existing compatible client helper.
  - Reject uses `POST /api/assets/{asset_id}/reject`.
- Show whether each scene already has a selected asset.
- Add per-asset Select / Reject buttons in each scene candidate asset card.
- Keep the modal open and update only the current summary state after a successful action.
- Show per-asset pending and error states.
- Show `unauthorized / login required` for 401/403 action failures.

## Out of Scope

- No scene regenerate.
- No manifest or ZIP export.
- No multi-user review workflow.
- No backend, MCP or provider changes.
- No real provider calls.
- No API key, provider key, secret, cookie or session inspection.

## Implementation Notes

- `agentImageflowApi` now exposes `buildAgentImageflowAssetReviewUrl`, and the existing `selectAgentImageflowAsset` / `rejectAgentImageflowAsset` helpers reuse it.
- `ProductionViewModal` keeps action state per `asset_id`, so one candidate can show `Saving` or an inline error without blanking the modal.
- On success, the modal updates the matching summary asset status locally and recomputes scene selected/rejected counts, `primary_selected_asset_id`, story selected scene counts, and top-level selected/generated/rejected coverage counts.
- The scene header now includes a stable `selected asset` / `missing selected` pill, and action buttons use fixed height/min-width to avoid layout shifts.

## Verification

- `npm --prefix web test -- --run`
  - Result: 17 files / 226 tests passed.
- `npm --prefix web run build`
  - Result: passed; Vite still reports the existing large chunk warning family.

## Remaining Risks

- Browser visual smoke was not run in this turn; layout is covered by tests/build and conservative fixed-size button styling, but not by screenshot inspection.
- The general asset library is not force-refreshed from this modal. Server state is updated through the shared review API, and the asset library will reflect it on its next refresh/query.
