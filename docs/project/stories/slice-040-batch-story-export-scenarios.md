# Slice 040: Batch Story Export Foundation Scenarios

## Conclusion

P1 Project Production Context has closed the visual-consistency foundation. The next useful product slice is not another project-context field, but a small production layer around existing `source/session_id/batch_id/story_id/scene_id/target_path` metadata.

Recommended next CSV:

```text
issues/next-phase-p1-batch-story-export-foundation.csv
```

This slice should first make one pet-story batch easy to monitor, review and hand off. It should not become a story-writing system, Xiaohongshu publishing system, content calendar, general DAM, template marketplace, WebDAV server, multi-user workflow or AI visual-quality judge.

## Product Problem

The current platform can already generate images and preserve project visual context. A second agent can create 3 to 8 scene tasks with stable characters, references and prompt recipe. The remaining user pain is operational:

- The user has to know task IDs or asset filters to understand one story batch.
- Success, failure, retry, selected/rejected and per-scene coverage are scattered across assets, tasks and batch progress.
- A failed or weak scene needs scene-level retry/regenerate without rebuilding the whole batch manually.
- After selecting images, the user needs a simple delivery bundle: original URLs, thumbnails, metadata URLs, target paths and a machine-readable manifest.
- On NAS/self-hosted deployment, direct filesystem access is often enough for image files, while Agent ImageFlow should remain the metadata and audit source of truth.

## Existing Foundation

Already available:

- `metadata_json` standard fields: `source`, `source_agent`, `source_thread_id`, `session_id`, `run_id`, `batch_id`, `story_id`, `scene_id`, `target_path`.
- REST and CLI asset list filters for `source/session_id/batch_id/status/provider/model/keyword/date`.
- Admin Recent Assets filters for cross-scope recent assets after Admin login.
- `vag batch progress` and REST `batch-progress` for task count, success/failure, asset count and attempt count.
- Asset delivery URLs for original, thumbnail and metadata.
- Project Visual Context snapshots on task and asset records.
- Web Server Asset Library with filters, pagination, lazy loading, metadata/parameters summary and Project Context panel.

Known gap:

- MCP `list_image_assets` can list campaign assets, but session/batch filtering needs parity with REST/CLI/Admin before it can be clean evidence for batch-scoped agent workflows.

## Workflow Scenarios

### Scenario 1: Agent Creates A Story Batch

1. Story-writing agent outputs 3 to 8 scene specs outside Agent ImageFlow.
2. Image-production agent calls MCP/REST/CLI `create_image_task` once per scene.
3. Each task uses the same `workspace/project/campaign/session_id/batch_id/story_id`.
4. Each task sets its own `scene_id` and `target_path`.
5. Each task references project visual context with `character_ids`, `reference_asset_ids`, `prompt_recipe_id` and `use_project_visual_context=true`.

Expected platform role:

- Accept and persist structured metadata.
- Keep project visual context snapshots.
- Let the user query one batch without knowing every task ID.

### Scenario 2: User Monitors Batch Progress

The user opens a batch/story production view and sees:

- Total scene tasks.
- Completed, failed, running, queued and retrying counts.
- Asset count and attempt count.
- Per-scene task status.
- Per-scene selected/rejected/generated coverage.
- Slow or failed scenes with task attempt timing.

Expected first version:

- Read-only grouping is enough.
- Reuse existing task/asset/batch-progress data.
- No new story table unless a later requirement proves metadata-only grouping is not enough.

### Scenario 3: User Reviews One Scene

The user clicks `scene_002` and sees:

- Prompt/purpose summary.
- Character/reference/recipe snapshot.
- Generated candidate assets.
- Selected/rejected/generated status.
- Thumbnail/original/metadata links.
- Retry/regenerate action for this scene.

Expected first version:

- Retry/regenerate may create a new task with copied metadata and a new idempotency key.
- It must preserve `session_id/batch_id/story_id/scene_id` and should record a `regenerated_from_task_id` metadata field.
- It should not silently overwrite old assets.

### Scenario 4: User Exports A Batch Manifest

After selecting one or more assets, the user exports a batch manifest:

```json
{
  "workspace_id": "ws_default",
  "project_id": "prj_pet_account",
  "campaign_id": "cmp_story_001",
  "session_id": "pet_story_session_001",
  "batch_id": "pet_story_batch_001",
  "story_id": "rainy_window_cat",
  "assets": [
    {
      "scene_id": "scene_001",
      "asset_id": "asset_xxx",
      "status": "selected",
      "download_url": "/api/assets/asset_xxx/original",
      "thumbnail_url": "/api/assets/asset_xxx/thumbnail",
      "metadata_url": "/api/assets/asset_xxx/metadata",
      "target_path": "stories/rainy-window-cat/scene-001.png"
    }
  ]
}
```

Expected first version:

- JSON manifest first.
- ZIP can follow after manifest is stable.
- Include selected-only and all-assets modes.
- Do not push to Xiaohongshu or external CMS.

### Scenario 5: NAS / Docker / WebDAV Or SMB Deployment

For small-team self-hosting, the simplest model is:

- Docker writes image files under a configured storage root.
- NAS exposes that storage root through Finder/SMB/WebDAV at the infrastructure layer.
- Agent ImageFlow DB remains the source of truth for asset ID, status, prompt, visual context, scene, batch, delivery URL and metadata.
- Export manifest bridges DB metadata to filesystem access.

Recommended boundary:

- Do not implement a WebDAV/SMB server inside Agent ImageFlow in this phase.
- Do document stable directory layout, read-only sharing recommendations and backup expectations.
- Do not let filesystem folder names become the primary database.

## Minimal Closed Loop

The smallest useful loop for the pet account:

1. Use an existing project with three characters, one style reference and one `pet_story_cover` recipe.
2. Create a clean campaign/session/batch/story with 3 scene tasks via MCP/REST/CLI using mock provider.
3. Open Web and find one grouped batch/story view.
4. See 3 scenes, 3 completed tasks, 6 assets and selected/rejected/generated status.
5. Select one asset per scene from the grouped view.
6. Retry/regenerate one scene and confirm the new task keeps the same story/scene/batch metadata.
7. Export a JSON manifest for selected assets.
8. Confirm original/thumbnail/metadata URLs and target paths are valid and do not require provider secrets.

## Functional Design Direction

- Continue using `workspace -> project -> campaign -> task -> asset`.
- Continue using metadata grouping for `session_id/batch_id/story_id/scene_id` before adding new tables.
- Add API/CLI/Web only where they reduce task-ID hunting.
- Keep grouping read-only first; selection/rejection can reuse existing asset actions.
- Add regenerate as a task-copy action, not as in-place mutation.
- Add manifest before ZIP.
- Treat NAS/WebDAV/SMB as deployment/file-access guidance first, not a platform subsystem.

## Must Not Do In This Slice

- Xiaohongshu publishing, scheduling, captions, analytics or account ops.
- Story writing, script parsing or content calendar.
- General DAM tags, folder taxonomy, template marketplace or multi-person review.
- WebDAV/SMB server implementation inside the app.
- Provider key management in Web.
- Real provider benchmark or paid generation without explicit confirmation.
- AI visual-quality scoring or character-consistency judging.
- New database tables unless metadata-only grouping fails in implementation review.

## CSV Task Order

1. Baseline and scope guard.
2. Story scene summary contract.
3. Batch story summary API.
4. MCP list filters alignment.
5. Web production view read-only.
6. Scene asset actions.
7. Scene regenerate design.
8. Scene regenerate implementation.
9. Minimal export manifest.
10. Export pack ZIP boundary.
11. NAS Docker access guide and regression docs.
