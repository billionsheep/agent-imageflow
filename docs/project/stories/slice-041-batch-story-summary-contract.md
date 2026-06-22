# Story: 041 - Batch Story Summary Contract

## Status

- State: Done
- Created: 2026-06-22
- Updated: 2026-06-22

## Product Goal

Define the shared contract for viewing one generated story batch as scenes, tasks and assets before implementing API or Web UI. This prevents the backend, MCP, CLI, Web production view and manifest export from inventing incompatible grouping rules.

## Source Context

- Product spec: `docs/project/PRODUCT_SPEC.md`
- Project plan slice: `issues/next-phase-p1-batch-story-export-foundation.csv` / `P1-BSE-002`
- Scenario design: `docs/project/stories/slice-040-batch-story-export-scenarios.md`
- Related decisions: `docs/project/DECISIONS.md`

## User Flow

1. An external agent creates multiple scene tasks under the same `project_id`, `campaign_id`, `session_id`, `batch_id` and `story_id`.
2. The user opens a production view or calls an API with `session_id` and/or `batch_id`.
3. Agent ImageFlow groups matching scene tasks by `story_id` and `scene_id`.
4. The user sees scene status, selected coverage, candidate assets, delivery URLs and retry/regenerate trace fields.
5. Later export manifest uses the same grouping contract.

## In Scope

- Define summary route, query parameters and response shape.
- Define scene grouping and ordering.
- Define setup/reference task exclusion.
- Define scene-level regenerate metadata.
- Define manifest relationship to summary.
- Keep this as a contract/design slice; do not implement handlers or Web UI here.

## Out of Scope

- No new database tables or migrations.
- No Web production view implementation.
- No scene regenerate implementation.
- No JSON manifest endpoint implementation.
- No ZIP implementation.
- No real provider calls.
- No key/secret/cookie/session reading or printing.

## Contract

### REST Summary Route

First implementation should add a read-only route:

```text
GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-summary
```

Route decision: use `batch-summary` rather than `story-summary` because the existing platform already has `batch-progress`, and `story_id` remains an optional filter inside the batch.

Query parameters:

- `session_id`: optional but at least one of `session_id` or `batch_id` is required.
- `batch_id`: optional but at least one of `session_id` or `batch_id` is required.
- `story_id`: optional exact filter.
- `source`: optional exact filter using `metadata_json.source`.
- `status`: optional task status filter for troubleshooting views.
- `limit`: optional task limit, same default/cap family as `batch-progress`.
- `include_setup`: optional boolean, default `false`.

Default behavior:

- Match tasks by `generation_task.project_id` and `campaign_id`.
- Match `session_id`, `batch_id`, `story_id`, and `source` through `generation_task.structured_input_json.metadata_json`.
- By default include only scene production tasks with non-empty `metadata_json.scene_id`.
- Exclude setup/reference tasks without `scene_id`; report the count in `counts.excluded_setup_task_count`.
- Exclude tasks when `metadata_json.task_role` is `setup`, `reference_setup`, `visual_context_setup` or `calibration`.
- Exclude tasks when `metadata_json.exclude_from_story_summary=true`.
- Never return provider keys, API keys, cookies, local absolute file paths or raw secret-bearing configuration.

### Summary Response Shape

Recommended JSON shape:

```json
{
  "generated_at": "2026-06-22T00:00:00Z",
  "project_id": "prj_pet_account",
  "campaign_id": "cmp_story_001",
  "session_id": "pet_story_session_001",
  "batch_id": "pet_story_batch_001",
  "source": "codex",
  "story_id": "rainy_window_cat",
  "counts": {
    "story_count": 1,
    "scene_count": 3,
    "scene_with_selected_count": 3,
    "scene_missing_selected_count": 0,
    "task_count": 3,
    "queued_count": 0,
    "running_count": 0,
    "succeeded_count": 3,
    "partial_count": 0,
    "failed_count": 0,
    "retrying_count": 0,
    "asset_count": 6,
    "generated_asset_count": 3,
    "selected_asset_count": 3,
    "rejected_asset_count": 0,
    "attempt_count": 3,
    "excluded_setup_task_count": 1
  },
  "stories": [
    {
      "story_id": "rainy_window_cat",
      "scene_count": 3,
      "selected_scene_count": 3,
      "scenes": ["scene_001", "scene_002", "scene_003"]
    }
  ],
  "scenes": [
    {
      "story_id": "rainy_window_cat",
      "scene_id": "scene_001",
      "scene_order": 1,
      "target_path": "stories/rainy-window-cat/scene-001.png",
      "status": "completed",
      "latest_task_id": "task_xxx",
      "primary_selected_asset_id": "asset_xxx",
      "regenerated_from_task_id": "",
      "regeneration_count": 0,
      "counts": {
        "task_count": 1,
        "succeeded_count": 1,
        "failed_count": 0,
        "asset_count": 2,
        "selected_asset_count": 1,
        "rejected_asset_count": 0,
        "attempt_count": 1
      },
      "visual_context": {
        "character_ids": ["dog_mochi", "dog_biscuit"],
        "reference_asset_ids": ["asset_style_ref"],
        "prompt_recipe_id": "pet_story_cover"
      },
      "tasks": [
        {
          "task_id": "task_xxx",
          "status": "completed",
          "asset_count": 2,
          "attempt_count": 1,
          "retrying": false,
          "error_stage": "",
          "error_code": "",
          "error_message": "",
          "created_at": "2026-06-22T00:00:00Z",
          "updated_at": "2026-06-22T00:01:00Z"
        }
      ],
      "assets": [
        {
          "asset_id": "asset_xxx",
          "task_id": "task_xxx",
          "status": "selected",
          "provider": "mock",
          "model": "mock-image",
          "prompt": "scene prompt",
          "download_url": "/api/assets/asset_xxx/original",
          "thumbnail_url": "/api/assets/asset_xxx/thumbnail",
          "metadata_url": "/api/assets/asset_xxx/metadata",
          "target_path": "stories/rainy-window-cat/scene-001.png",
          "created_at": "2026-06-22T00:01:00Z"
        }
      ]
    }
  ]
}
```

Implementation can reuse existing `BatchProgressCounts` fields, but should expose extra scene/asset coverage counts because Web needs them without client-side reconstruction.

### Scene Status Derivation

Scene status is derived from grouped task statuses:

- `completed`: all scene tasks completed and at least one asset exists.
- `partial`: any task is partially completed or mixed completed/failed state exists.
- `failed`: all scene tasks are failed or enqueue_failed.
- `running`: any task is running.
- `queued`: any task is queued and no running task exists.
- `retrying`: any task has an active retry_after.
- `empty`: scene row exists but no task or asset can be resolved. This should be rare and mostly for future saved scene plans.

If multiple rules match, priority is `retrying`, `running`, `queued`, `partial`, `failed`, `completed`, `empty`.

Asset status in summary should use public semantic status:

- `draft` maps to `generated`.
- `approved` maps to `selected`.
- `rejected`, `published` and `deprecated` keep their current meaning.

### Scene Ordering

Use the first available rule:

1. Numeric `metadata_json.scene_order` if present.
2. Trailing number parsed from `scene_id`, such as `scene_001 -> 1`.
3. Lexical `scene_id`.
4. Earliest task `created_at` and then task id as tie breaker.

### Setup Task Exclusion

Default `include_setup=false` excludes tasks that match the batch filters but have empty `metadata_json.scene_id`.

Callers may explicitly tag setup tasks with:

```json
{
  "task_role": "setup"
}
```

Accepted setup role values are `setup`, `reference_setup`, `visual_context_setup` and `calibration`. A task with `exclude_from_story_summary=true` is also excluded. Excluded tasks are counted; they do not enter scene coverage, selected coverage or manifest exports by default.

### Scene Regenerate Metadata

Scene regenerate should create a new task, not mutate or overwrite the old one.

The new task must preserve:

- `source`
- `session_id`
- `batch_id`
- `story_id`
- `scene_id`
- `target_path`
- `character_ids`
- `reference_asset_ids`
- `prompt_recipe_id`
- `use_project_visual_context`

The new task should add:

```json
{
  "regenerated_from_task_id": "task_old",
  "regenerate_no": 1,
  "regenerate_reason": "optional user supplied reason"
}
```

Selection rules:

- Old assets remain available.
- Existing selected/rejected states are not automatically changed.
- If multiple assets are selected within one scene, `primary_selected_asset_id` is the latest selected asset by `created_at`, then `asset_id`.

### Manifest Relationship

Batch manifest must use the same filters and scene grouping as summary.

Recommended future route:

```text
GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-manifest
```

Recommended query parameters:

- Same as `batch-summary`.
- `selected_only=true|false`, default should be `true` for Web download and explicit in CLI.
- `include_rejected=true|false`, default `false`.

Manifest must include delivery URLs and target paths, but must not include local absolute paths or secrets.

ZIP is explicitly a later boundary decision. If implemented, ZIP should be built from manifest-safe paths and must enforce count/size limits.

## Acceptance Criteria

- Given the next worker reads this story, when implementing `P1-BSE-003`, then it has a concrete route, query and response contract.
- Given a batch contains setup/reference tasks without `scene_id`, when summary is requested with default options, then those tasks are excluded and counted.
- Given a scene is regenerated later, when summary groups tasks by `story_id/scene_id`, then old and new tasks stay in the same scene without overwriting old assets.
- Given a manifest is implemented later, when it filters selected-only assets, then it uses the same `session_id/batch_id/story_id/scene_id` grouping as summary.

## Technical Approach

- Add new domain types in `internal/domain/types.go` near `BatchProgress*`.
- Add a store method that joins `generation_task`, `asset`, `asset_version` and latest attempt data, or reuses existing query patterns with a focused aggregate query.
- Add HTTP parser mirroring `parseBatchProgressQuery`, extended with `story_id`, `source`, `status` and `include_setup`.
- Add service method as a thin pass-through plus normalization, following `GetBatchProgress`.
- Add Web client types only after the backend route is implemented.

## Data / Interface Impact

- No database migration for the contract.
- New REST surface planned: `GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-summary`.
- New future REST surface planned: `GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-manifest`.
- MCP `list_image_assets` should gain filters in P1-BSE-004; a new `get_batch_summary` MCP tool can be considered after REST/CLI summary proves useful.

## Files or Subsystems Likely to Change

- `internal/domain/types.go`
- `internal/store/postgres.go`
- `internal/store/postgres_test.go`
- `internal/app/service.go`
- `internal/httpapi/server.go`
- `internal/httpapi/server_test.go`
- `internal/httpapi/audit.go`
- `cmd/vag/main.go`
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- `web/src/lib/agentImageflowApi.ts`
- `web/src/components/*`

## Verification Plan

```bash
git diff --check
python3 - <<'PY'
import csv
from pathlib import Path
with Path('issues/next-phase-p1-batch-story-export-foundation.csv').open(newline='') as f:
    rows = list(csv.DictReader(f))
print([(row['id'], row['status']) for row in rows if row['id'].startswith('P1-BSE-')])
PY
```

Implementation slices after this contract should run focused Go tests, Web tests and mock smoke checks.

## Assumptions and Risks

- Metadata-only grouping is sufficient for the first production view.
- `scene_id` is required for scene production rows; setup tasks without `scene_id` are excluded by default.
- Summary may duplicate some fields from asset list responses to keep Web simple and avoid flicker-prone client-side joins.
- If future saved scene plans are needed before tasks exist, that will require a separate story and possibly a new table.

## Implementation Log

### 2026-06-22

- Changes: Defined `batch-summary`, scene grouping, setup exclusion, regenerate metadata and manifest relationship.
- Verification: Contract-only slice; no business code changed and no real provider run.
- Remaining gaps: P1-BSE-003 must implement the summary API; P1-BSE-004 must align MCP list filters.
