# Story: 046 - Scene Regenerate Design

## Status

- State: Done
- Created: 2026-06-22
- Updated: 2026-06-22

## Product Goal

定义第一版 scene 级 regenerate 语义，让用户或外部 agent 可以针对失败、效果差或仍想探索的单个 scene 创建新的生成任务，而不是重跑整批故事图；旧 task、旧 assets、selected/rejected 状态和 batch/story/scene 归属必须保持可追溯且不被覆盖。

## Source Context

- Product spec: `docs/project/PRODUCT_SPEC.md`
- Contract: `docs/project/stories/slice-041-batch-story-summary-contract.md`
- API implementation: `docs/project/stories/slice-042-batch-story-summary-api.md`
- Web production view: `docs/project/stories/slice-044-web-production-view-read-only.md`
- Scene asset actions: `docs/project/stories/slice-045-scene-asset-actions.md`
- Project plan slice: `issues/next-phase-p1-batch-story-export-foundation.csv` / `P1-BSE-007`
- Related decisions: `docs/project/DECISIONS.md`

## User Flow

1. 外部 agent、CLI、API 调用方或 Web 用户打开一个 `session_id/batch_id/story_id/scene_id` 对应的 batch summary。
2. 某个 scene 的原 task 失败、生成效果差、已有 selected asset 但用户想继续探索，或 prompt / prompt recipe / reference / generation config 需要局部调整。
3. 调用方对单个 scene 发起 regenerate，并提供 `source_task_id`，或提供 scene identity 让服务端解析该 scene 的 latest task。
4. 服务端不 retry 原 task，也不覆盖旧 assets，而是复制原 task 的业务归属和生产上下文，合并允许的 overrides，创建一个新的 `ImageTask`。
5. 新 task 保持同一 `project_id/campaign_id/session_id/batch_id/story_id/scene_id`，并写入 `regenerated_from_task_id`、`regenerate_no` 和可读的 visual context snapshot 摘要。
6. batch summary、asset list、progress 和未来 manifest 都能把旧 task、新 task 和候选 assets 聚合到同一个 scene 下，同时保留每张 asset 原有的 selected/rejected/generated 状态。

## In Scope

- 只做设计和项目管理文档更新。
- 定义第一版语义：`create-new-task-as-regeneration`，不是 retry 原 task。
- 定义 REST/CLI/MCP/Web 后续实现入口草案。
- 定义输入输出契约草案。
- 定义 metadata-only 记录规则，不新增数据库表。
- 定义 batch summary、progress、asset list 和未来 manifest 的可读规则。
- 定义 P1-BSE-008 实现验收标准。

## Out of Scope

- 不实现业务代码。
- 不改 Go/TS 业务逻辑。
- 不新增数据库 schema migration。
- 不运行真实 provider 或真实 provider smoke。
- 不读取、打印或处理 API key、provider key、secret、cookie 或 session token。
- 不做图像相似度 AI 质检。
- 不做自动挑图或自动替换 selected asset。
- 不做多版本树 UI。
- 不覆盖旧资产或旧 task。
- 不做跨 scene 批量重生。
- 不把该能力扩成 Usage Tracking、edit lineage 或通用 DAM。

## Regenerate Semantics

第一版采用 `create-new-task-as-regeneration`：

- Regenerate 总是创建新的 `generation_task.id`。
- 原 task 状态、attempts、error、assets 和 selection events 不变。
- 原 selected/rejected/generated/published/deprecated asset 状态不自动改变。
- 新 task 初始状态走现有 create task / queue / worker 流程，生成的新 assets 初始仍为 `generated`，除非原任务输入或 overrides 显式保留 `selection_mode=auto|best_of`，让现有 best-of 逻辑在新 task 完成后对新 task 的候选生效。
- 同一 scene 可以有多个 regeneration task；batch summary 继续按 `story_id/scene_id` 聚合它们。
- `primary_selected_asset_id` 仍按 summary contract 推导：同一 scene 有多个 selected asset 时，用最新 selected asset 的 `created_at`，再用 `asset_id` 作为 tie breaker；regenerate 创建新 task 本身不改变 primary selected。

该语义和 worker retry/backoff 的区别：

- retry/backoff 是同一个 task 的执行恢复，适用于 provider 瞬时失败。
- scene regenerate 是用户或 agent 主动创建新 task，适用于失败后重生、效果差重生、继续探索或更换 prompt/recipe/config 后重生。

## User Scenarios

- 失败重生：`scene_002` 的 latest task 为 `failed` 或 `partially_completed`，用户点击 regenerate，服务端复制 scene 归属和上下文创建新 task；旧 failed task 仍保留在 scene 的 task history 中。
- 效果差重生：scene 已有 `generated` 或 `selected` assets，但用户认为画面不满意，发起 regenerate；旧 assets 继续可见，不自动 rejected。
- 已有 selected asset 后继续探索：scene 已有首选图，用户想再试一个风格版本；新 task 成功后新 assets 只是候选，是否改选由用户或后续显式 select 决定。
- 更换 prompt / recipe 后重生：调用方传 overrides，例如 `prompt`、`negative_prompt`、`prompt_recipe_id`、`character_ids`、`reference_asset_ids` 或允许覆盖的 `generation_config`；服务端记录哪些字段被覆盖。
- CLI/API/MCP agent 发起：请求必须带 `source` / `source_agent` / `source_thread_id` 或继承原 task metadata；新 task metadata 记录 `regenerated_by`、`regenerate_reason` 和 `regenerate_request_source`，让只看 asset list / manifest 的调用方也能追踪来源。

## Input Contract Draft

推荐第一版 REST action：

```text
POST /api/projects/{project_id}/campaigns/{campaign_id}/scene-regenerations
```

输入必须包含 `source_task_id` 或 scene identity 二选一：

```json
{
  "project_id": "prj_pet_account",
  "campaign_id": "cmp_story_001",
  "source_task_id": "task_old",
  "scene_identity": {
    "session_id": "pet_story_session_001",
    "batch_id": "pet_story_batch_001",
    "story_id": "rainy_window_cat",
    "scene_id": "scene_002",
    "source": "codex",
    "task_selector": "latest"
  },
  "regenerate_reason": "scene failed provider timeout",
  "created_by": "codex",
  "overrides": {
    "prompt": "updated scene prompt",
    "negative_prompt": "low quality, blurry",
    "prompt_recipe_id": "pet_story_cover_v2",
    "character_ids": ["dog_mochi", "cat_orange"],
    "reference_asset_ids": ["asset_style_ref"],
    "quality_profile_id": "project_default",
    "generation_config": {
      "quality": "high",
      "seed": 12345
    },
    "requested_count": 2,
    "selection_mode": "manual_optional"
  }
}
```

字段规则：

- `project_id`、`campaign_id` 来自路径或请求体，但必须与 source task 匹配。
- `source_task_id` 优先；若未提供，则必须提供 `scene_identity.session_id`、`scene_identity.batch_id`、`scene_identity.story_id`、`scene_identity.scene_id`，并用 `task_selector=latest` 解析该 scene 的 latest task。
- `scene_identity.source` 可选，用于避免同一 session/batch 下不同入口混淆。
- `overrides` 可选；未传时完整复用 source task 的 prompt、negative prompt、provider、model、requested_count、selection_mode、quality snapshot、visual context ids 和 generation config。
- 第一版允许覆盖的字段限定为：`prompt`、`negative_prompt`、`prompt_recipe_id`、`character_ids`、`reference_asset_ids`、`reference_images`、`quality_profile_id`、`generation_config` 中现有 create task 已支持且非敏感的字段、`requested_count`、`selection_mode`、`aspect_ratio`、`output_format`、`provider`、`model`。
- 第一版不允许通过 regenerate 覆盖 `workspace_id`、`project_id`、`campaign_id`、`session_id`、`batch_id`、`story_id`、`scene_id`、`target_path`，除非未来另开“move/copy scene”设计。
- 第一版不接受 provider secret、API key、cookie、session token 或任意本地文件路径。

## Output Contract Draft

推荐响应：

```json
{
  "task_id": "task_new",
  "status": "queued",
  "regenerated_from_task_id": "task_old",
  "regenerate_no": 2,
  "project_id": "prj_pet_account",
  "campaign_id": "cmp_story_001",
  "session_id": "pet_story_session_001",
  "batch_id": "pet_story_batch_001",
  "story_id": "rainy_window_cat",
  "scene_id": "scene_002",
  "copied_visual_context_snapshot": {
    "character_ids": ["dog_mochi", "cat_orange"],
    "reference_asset_ids": ["asset_style_ref"],
    "prompt_recipe_id": "pet_story_cover_v2",
    "character_count": 2,
    "reference_count": 1,
    "has_prompt_recipe": true
  },
  "warnings": [
    {
      "code": "selected_asset_preserved",
      "message": "Existing selected assets were not changed."
    }
  ]
}
```

输出规则：

- `task_id` 是新 task。
- `status` 通常为 `queued`，若入队失败则按现有 create task 语义返回 `enqueue_failed` 或错误。
- `regenerated_from_task_id` 必须等于 source task。
- `regenerate_no` 是同一 scene 下 regeneration task 的顺序号；第一版可通过查询同 scene 内已有 `metadata_json.regenerated_from_task_id` 任务数量推导，或用 source task lineage chain 推导，但响应和 metadata 必须稳定。
- `copied_visual_context_snapshot` 只返回摘要，不返回可能很大的完整 prompt/reference 原始 payload。
- `warnings` 用于提示 selected 不自动改变、overrides 被忽略、source task 不是 latest、source task visual context snapshot 缺失等非致命情况。

## Metadata Rules

第一版继续使用 metadata-only 方案，不新增表。

### `generation_task.structured_input_json`

新 task 的 `structured_input_json` 应包含：

- 复制后的统一 task 输入快照：`prompt`、`negative_prompt`、`style_preset`、`aspect_ratio`、`output_format`、`provider`、`model`、`requested_count`、`selection_mode`、`best_of_config`、`generation_config`、`reference_images`、`mask_image`、`character_ids`、`reference_asset_ids`、`prompt_recipe_id`、`use_project_visual_context`。
- `metadata_json` 内保留原 scene 归属：`source`、`source_agent`、`source_thread_id`、`session_id`、`run_id`、`batch_id`、`story_id`、`scene_id`、`scene_order`、`target_path`。
- `metadata_json.regenerated_from_task_id`。
- `metadata_json.regenerate_no`。
- `metadata_json.regenerate_reason`。
- `metadata_json.regenerate_request_source`，例如 `web|cli|rest|mcp|automation`。
- `metadata_json.regenerated_by`，例如 `admin`、`codex`、`cli` 或调用方传入的非敏感 actor。
- `metadata_json.regenerated_at`。
- `metadata_json.regeneration_overrides`，只记录覆盖字段名和非敏感值；不得记录 secret 或认证材料。
- `metadata_json.source_scene_identity`，记录解析 source task 时使用的 `session_id/batch_id/story_id/scene_id/source/task_selector`。
- `metadata_json.regeneration_root_task_id` 可选；若 source task 已经是 regeneration，则继承最早 root，便于未来展示链路。

### `asset_version.parameters_json`

新 assets 的 `parameters_json` 应继续由新 task 快照写入，并额外保留：

- `regenerated_from_task_id`
- `regenerate_no`
- `regenerate_reason`
- `regeneration_root_task_id` 可选
- `visual_context_snapshot` 摘要或完整现有快照，遵循当前 asset parameters 写法
- `generation_config` 的实际生效非敏感字段
- `reference_images` / `mask_image` 的 descriptor，不包含本地绝对路径或 secret

### Asset metadata URL / exported metadata

`GET /api/assets/{asset_id}/metadata` 和未来 manifest 应能读到：

- `task_id`
- `regenerated_from_task_id`
- `regenerate_no`
- `session_id`
- `batch_id`
- `story_id`
- `scene_id`
- `target_path`
- `prompt_recipe_id`
- `character_ids`
- `reference_asset_ids`
- `visual_context_snapshot` 摘要

不得返回：

- provider key
- project API key
- Basic Auth password
- cookie / session token
- 本地绝对路径
- raw provider request body 中可能包含的认证材料

## Batch Summary / Progress / Asset List / Manifest Readability

- `batch-summary` 不需要为 P1-BSE-008 改变 grouping contract；新 task 因为保留同一 `session_id/batch_id/story_id/scene_id`，自然进入同一 scene row。
- scene row 应继续暴露 `latest_task_id`，实现时建议 latest task 按 `created_at` 再 `task_id` 推导；如 latest task 是 regeneration，scene row 的 `regenerated_from_task_id` 和 `regeneration_count` 应可读。
- task rows 应显示每个 task 的 `regenerated_from_task_id`、`regenerate_no` 和 `created_at`，方便 Web 展开 history。
- `batch-progress` 仍按 session/batch 统计任务数量；regenerate 新 task 会增加 `task_count`，这符合“同一批生产内新增一次 scene 尝试”的语义。
- REST/CLI/MCP asset list 可通过现有 `source/session_id/batch_id/status/keyword` 查回新 assets；后续可增加 `regenerated_from_task_id` filter，但 P1-BSE-008 不强制。
- 未来 manifest 默认按 selected-only 输出时，不因新 task 自动替换旧 selected；只有用户显式 select 新 asset 后，selected-only manifest 才指向新 selected。

## Web / CLI / API / MCP Implementation Impact

### REST/API

- 新增 `POST /api/projects/{project_id}/campaigns/{campaign_id}/scene-regenerations` 或等价 action route。
- 复用 application core 的 create task 路径，避免入口层直接写 provider 调用。
- 校验 source task 属于同一 `project_id/campaign_id`，并继承 `workspace_id`。
- 校验 scene identity 解析出的 latest task 唯一且属于同一 scope。
- 审计日志 action 建议为 `regenerate_scene`，记录 `project_id/campaign_id/source_task_id/task_id/story_id/scene_id`，不记录 secret。

### CLI

- 新增命令草案：

```bash
vag scene regenerate \
  --project <project_id> \
  --campaign <campaign_id> \
  --source-task <task_id> \
  --reason "scene failed" \
  --prompt-file scene-002-prompt.txt
```

- 也可支持 scene identity：

```bash
vag scene regenerate \
  --project <project_id> \
  --campaign <campaign_id> \
  --session <session_id> \
  --batch <batch_id> \
  --story <story_id> \
  --scene <scene_id>
```

### MCP

- 新增 tool 草案：`regenerate_scene_task`。
- 输入字段与 REST contract 对齐，优先支持 `source_task_id`，再支持 scene identity。
- MCP 输出必须是结构化 JSON，包含 `task_id/status/regenerated_from_task_id/regenerate_no/warnings`。
- MCP 不新增真实 provider smoke；实现验收使用 mock provider。

### Web

- 在 Production View 的 scene header 或 task row 加 `Regenerate` action。
- 第一版按钮可打开最小 modal：显示 source task、scene identity、可选 reason、prompt / negative prompt / recipe / generation config overrides。
- 若 scene 已有 selected asset，按钮不改变 selected；UI 应明确显示“existing selected preserved”。
- 成功创建新 task 后保持 modal 打开或返回 scene row，并局部插入新 task pending/queued 状态或刷新 batch summary；不得清空整个 Production View。

### Later, Not P1-BSE-008

- 不做多版本树 UI。
- 不做跨 scene 批量 regenerate。
- 不做 AI similarity / visual QC。
- 不做自动挑图或自动替换 selected。
- 不做复杂 lineage graph。

## Acceptance Criteria For P1-BSE-008

- Given `source_task_id` 属于目标 project/campaign 且有 `session_id/batch_id/story_id/scene_id`，when 调用 scene regenerate，then 系统创建一个新 task，并返回 `task_id/status/regenerated_from_task_id/regenerate_no`。
- Given 原 scene 有 selected asset，when regenerate 新 task，then 原 selected asset 仍保持 selected，新 task 生成的 assets 不自动成为 selected，除非后续显式 select 或新 task 自身 `selection_mode=auto|best_of` 只在新候选内生效。
- Given 原 task 失败，when regenerate，then 原 task 仍保持 failed，新 task 进入 queued/running/completed 等独立状态。
- Given 调用方传 prompt 或 recipe overrides，when 新 task 创建完成，then `structured_input_json.metadata_json.regeneration_overrides` 记录覆盖字段，新 task 的 prompt/recipe/config 使用覆盖后的有效值。
- Given 调用方只传 scene identity，when scene 下存在多个 task，then `task_selector=latest` 选择最新 task；响应 warnings 应提示 source task 是否不是 latest 或是由 selector 解析得到。
- Given 查询 `batch-summary`，when 新 task 已创建，then 同一 `story_id/scene_id` 的 scene row 包含旧 task 和新 task，`regeneration_count` 增加，旧 assets 仍可见。
- Given 查询 asset metadata 或未来 manifest，when asset 来自 regenerated task，then 可读到 `regenerated_from_task_id` 和 `regenerate_no`。
- Given 使用 CLI 或 MCP 发起，when 请求成功，then 输出结构化 JSON，不读取、打印或处理任何 key/secret/cookie/session token。
- Given 运行实现验证，then 使用 Go 单测、mock provider smoke、Web/CLI 最小验证和 `git diff --check`；不运行真实 provider smoke。

## Technical Approach

- 在 domain 层新增 scene regenerate request/response DTO，保持和 create task DTO 接近。
- 在 app service 中解析 source task 或 scene identity，复制 source task 的非敏感输入快照，合并 overrides，再调用现有 create task 流程。
- Store 层只需要查询 source task、同 scene regeneration count 和 latest scene task；第一版不新增表。
- HTTP/MCP/CLI 入口都走同一个 service 方法。
- Web 只发 action request，不在前端自行复制完整 task JSON 或拼 final prompt。

## Data / Interface Impact

- No database migration.
- New REST action planned: `POST /api/projects/{project_id}/campaigns/{campaign_id}/scene-regenerations`.
- New CLI command planned: `vag scene regenerate`.
- New MCP tool planned: `regenerate_scene_task`.
- Web Production View adds a scene-level action after backend exists.
- Existing `batch-summary`, `batch-progress`, asset list and metadata endpoints remain compatible.

## Files or Subsystems Likely to Change In P1-BSE-008

- `internal/domain/types.go`
- `internal/app/service.go`
- `internal/store/postgres.go`
- `internal/store/postgres_test.go`
- `internal/httpapi/server.go`
- `internal/httpapi/server_test.go`
- `internal/httpapi/audit.go`
- `cmd/vag/main.go`
- `internal/mcp/server.go`
- `internal/mcp/server_test.go`
- `web/src/lib/agentImageflowApi.ts`
- `web/src/components/ProductionViewModal.tsx`
- `web/src/types.ts`

## Verification Plan

```bash
git diff --check
python3 - <<'PY'
import csv
from pathlib import Path
with Path('issues/next-phase-p1-batch-story-export-foundation.csv').open(newline='') as f:
    rows = list(csv.DictReader(f))
print([(row['id'], row['status'], row['evidence']) for row in rows if row['id'] == 'P1-BSE-007'])
PY
```

P1-BSE-008 implementation should additionally run focused Go tests, MCP tests if tool is added, Web tests/build if Web button is added, and a mock provider smoke. It must not run a real provider smoke.

## Assumptions and Risks

- Metadata-only lineage is sufficient for the first implementation.
- `source_task_id` is the safest default because it avoids ambiguous scene identity resolution.
- `task_selector=latest` is useful for CLI/MCP automation, but implementation must warn when resolving automatically.
- If future users need persistent scene plans before any task exists, saved scene rows may require a new table and product confirmation.
- If future users need a visual version tree, that should be a separate UX story, not part of first regenerate.

## Implementation Log

### 2026-06-22

- Changes: Defined scene regenerate semantics, input/output contracts, metadata rules, Web/CLI/API/MCP impacts, and P1-BSE-008 acceptance criteria.
- Verification: Documentation and CSV checks only; no business code changed and no real provider run.
- Remaining gaps: P1-BSE-008 must implement the backend/action surface and verify with mock provider.
