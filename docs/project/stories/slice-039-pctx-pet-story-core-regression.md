# Slice 039: P1-PCTX-009 Pet Story Core Regression

## Status

- State: Done
- Created: 2026-06-22
- Updated: 2026-06-22

## Product Goal

验证 Project Visual Context 不是孤立配置，而是真的能支撑“萌宠账号图片资产生产”工作流：外部 agent 拆好的故事 scene 可以引用 project 角色卡、同 project 参考图和 prompt recipe，经 mock provider 生成后，任务、资产、metadata、batch progress 和 Web/Admin Recent Assets 可追溯不断链。

## In Scope

- 只用本地 mock provider。
- 使用 clean project/campaign/session/batch，避免旧 smoke 数据影响验收。
- 在 project 下准备两只狗和一只橘猫角色卡、一个同 project style reference asset、一个 `pet_story_cover` prompt recipe。
- 创建 3 个 scene task，每个 task 传 `character_ids`、`reference_asset_ids`、`prompt_recipe_id`、`use_project_visual_context=true`。
- 验证 task / asset snapshot、batch progress、asset list、Admin Recent Assets 和基础 Web preview。
- 写回 CSV、TASKS、PROJECT_PLAN、PROJECT_STATUS_MAP、CHECKPOINTS 和 RUNBOOK evidence。

## Out Of Scope

- 不实现新功能或修改 Web 业务逻辑。
- 不运行真实 provider。
- 不读取、打印或处理 API key / provider key / secret。
- 不做小红书发布、内容日历、账号运营后台、通用 DAM、Batch / Story / Scene 新 UI、Export Pack、NAS/WebDAV/SMB 或 Usage Tracking。

## Smoke Context

- Workspace: `ws_default`
- Project: `prj_pet_story_pctx009_1782094416`
- Campaign: `cmp_pet_story_pctx009_1782094416`
- Style reference setup session/batch: `pet_story_pctx009_session_1782094416` / `pet_story_pctx009_batch_1782094416`
- Final scene-only session/batch/story: `pet_story_pctx009_scene_session_1782094416` / `pet_story_pctx009_scene_batch_1782094416` / `pet_story_pctx009_scene_story_1782094416`
- Style reference task / asset: `task_4dfbbd870dbb99f2e9fc` / `asset_b8d5272e4afa0e249e5f`

## Scene Tasks

| Scene | Task | Assets |
| --- | --- | --- |
| `scene_001` | `task_6e6cf178fcf656386d62` | `asset_0aaa4b95e0914ba51c6d`, `asset_1735fa3123d84caea912` |
| `scene_002` | `task_2fca7b06875125b64d01` | `asset_743f12cb989a42fe002a`, `asset_4b174cc3330569378cc0` |
| `scene_003` | `task_108de9bcdaafc384302f` | `asset_b4232047c2b6df882314`, `asset_c486950784aef846a424` |

## Acceptance Evidence

- Visual context readback returned 3 characters: `dog_mochi`, `dog_biscuit`, `cat_orange`.
- Visual context readback returned 1 active style reference binding to `asset_b8d5272e4afa0e249e5f`.
- Visual context readback returned active recipe `pet_story_cover`.
- The final scene-only batch returned `task_count=3`, `succeeded_count=3`, `failed_count=0`, `asset_count=6`, `attempt_count=3`.
- Asset list filtered by `source=codex`, `session_id=pet_story_pctx009_scene_session_1782094416`, `batch_id=pet_story_pctx009_scene_batch_1782094416` returned 6 assets covering `scene_001`, `scene_002`, `scene_003`.
- Admin Recent Assets with the same filters returned the same 6 assets and scope `ws_default/prj_pet_story_pctx009_1782094416/cmp_pet_story_pctx009_1782094416`.
- Each scene task kept `structured_input_json.character_ids`, `reference_asset_ids`, `prompt_recipe_id`, `metadata_json.story_id`, `scene_id`, `batch_id`, and `visual_context_snapshot`.
- Each scene asset kept thumbnail and metadata URLs, plus `parameters_json.visual_context_snapshot` / `metadata_json.visual_context_snapshot` containing character ids, reference asset ids and recipe id.
- Production preview root at `http://127.0.0.1:4173/` returned HTTP 200.

## Verification

- `docker compose ps`: API, worker, postgres and redis were running.
- `curl -sf http://localhost:8081/healthz`: returned `{"status":"ok"}`.
- `docker compose config --quiet`: passed.
- `npm --prefix web test -- --run`: passed, 17 files / 226 tests.
- `npm --prefix web run build`: passed with the existing Vite chunk size warning.
- Host `go` was unavailable; RUNBOOK containerized command was used.
- `docker run --rm -v "$PWD":/src -w /src golang:1.25.3-alpine sh -lc '/usr/local/go/bin/go test ./...'`: passed.
- MCP read-only `list_image_assets` against the campaign returned assets, but did not apply the supplied `session_id/batch_id` filters; REST / CLI / Admin Recent Assets are the evidence source for clean scene-only filtering in this slice.

## Implementation Log

- Created a clean smoke project and campaign through the local Admin session without printing cookie/session token.
- Generated one mock style reference asset inside the same project and bound it as a project reference.
- Wrote project visual context with three character profiles, one style reference and one prompt recipe.
- Created one initial setup batch and then a final scene-only batch; final evidence uses the scene-only batch so style reference setup does not pollute `task_count` / `asset_count`.
- Created and completed three mock scene tasks with two generated assets each.
- Verified task snapshots, asset snapshots, batch progress, filtered asset list and Admin Recent Assets.
- No real provider was run, and no API key / provider key / secret was read or printed.

## Remaining Risks

- This is a mock-provider regression, so it proves platform data flow and traceability, not real-model visual consistency quality.
- Browser UI visibility was validated via production preview HTTP 200 and Admin Recent Assets API; no additional browser click-through was needed for this regression slice.
- MCP `list_image_assets` batch filtering may need a later contract check if MCP clients require source/session/batch filtering directly; no code change was made in P1-PCTX-009.
