# Story: 048 - Minimal Export Manifest

## Status

- State: Done
- Created: 2026-06-22
- Updated: 2026-06-22

## Product Goal

让外部 agent、CLI、REST 调用方和 Web 用户在完成一批故事图片审看后，可以一次拿到稳定 JSON manifest，包含交付 URL、metadata URL、target path、scene/story/task 追踪和 visual context 摘要，而不是逐个 asset 再自行拼装。

## Source Context

- Scenario story: `docs/project/stories/slice-040-batch-story-export-scenarios.md`
- Manifest contract: `docs/project/stories/slice-041-batch-story-summary-contract.md`
- Summary API: `docs/project/stories/slice-042-batch-story-summary-api.md`
- Production View: `docs/project/stories/slice-044-web-production-view-read-only.md`
- Scene asset actions: `docs/project/stories/slice-045-scene-asset-actions.md`
- Scene regenerate implementation: `docs/project/stories/slice-047-scene-regenerate-implementation.md`
- Project plan slice: `issues/next-phase-p1-batch-story-export-foundation.csv` / `P1-BSE-009`

## In Scope

- Add read-only `batch-manifest` REST endpoint.
- Reuse `batch-summary` grouping and filters.
- Add selected-only and all-assets manifest modes.
- Add CLI `vag batch manifest`.
- Add Web Production View JSON manifest export action.
- Include tasks, scenes and assets in a machine-readable JSON response.
- Exclude local absolute paths and any secrets.

## Out of Scope

- No ZIP export in this slice.
- No NAS/WebDAV/SMB server.
- No real provider smoke.
- No automatic publishing or content calendar.
- No visual QC / AI scoring.
- No deletion, cleanup or asset mutation.
- No credential, cookie or session token inspection.

## Acceptance Criteria

- REST returns a JSON manifest for a project/campaign plus `session_id` or `batch_id`.
- `selected_only=true` returns only selected assets.
- `selected_only=false&include_rejected=false` returns generated and selected assets, excluding rejected.
- `selected_only=false&include_rejected=true` returns generated, selected and rejected assets.
- CLI can print the same manifest JSON.
- Web can export/copy/download the manifest from Production View without closing or blanking the modal.
- Manifest includes delivery URLs, metadata URLs, `target_path`, task summaries, scene/story ids and visual context summary.
- Manifest does not include provider keys, project API keys, cookies, session tokens or local absolute paths.

## Verification Plan

- Containerized Go tests for domain/app/httpapi/cmd/vag.
- Web tests and production build if Web client/action is added.
- Browser production preview smoke for opening Production View and manifest action visibility.
- `git diff --check`.
- No real provider run.

## Implementation Log

### 2026-06-22

- Started P1-BSE-009 implementation planning.
- Added REST `GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-manifest`, derived from `batch-summary` grouping.
- Added CLI `vag batch manifest` with selected-only/all-assets/rejected controls.
- Added Web Production View JSON manifest export buttons for selected, all, and all + rejected modes.
- Verification passed: containerized Go focused tests, Web tests/build and `git diff --check`.
- No ZIP export, no real provider run, and no API key / provider key / secret / cookie / session token read or printed.
