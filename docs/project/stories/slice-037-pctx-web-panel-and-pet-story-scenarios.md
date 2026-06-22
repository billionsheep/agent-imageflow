# Slice 037: P1 Project Context Web Panel And Pet Story Scenarios

## Status

- State: Done for scenario design; P1-PCTX-008 implemented in `slice-038`; P1-PCTX-009 regression completed in `slice-039`
- Created: 2026-06-22
- Updated: 2026-06-22

## Product Goal

把 P1-PCTX-008 和 P1-PCTX-009 从“补一个 Web 面板、跑一次回归”细化为可执行场景：先让用户能在 Web 上看见和维护当前 project 的角色卡、参考图和 prompt recipe，再用 mock provider 验证一个萌宠账号故事批量生图流程能从外部 agent 到 Web 资产查看完整闭环。

本 slice 只补场景与功能设计，不实现代码，不运行真实 provider，不读取或打印任何 API key / provider key / secret。后续 P1-PCTX-008 的 Web 实现证据已记录在 `docs/project/stories/slice-038-pctx-web-project-context-panel.md`；P1-PCTX-009 萌宠故事回归证据已记录在 `docs/project/stories/slice-039-pctx-pet-story-core-regression.md`。

## Product Principle

- 先打通，不做大系统：第一版 Web 只服务当前 project 的视觉上下文维护和托管任务创建，不做通用 DAM、白板、模板市场或账号运营后台。
- Project 是长期视觉生产上下文：角色、参考图和 recipe 都属于 project；campaign 只承载一次故事、一期封面或一组批次。
- Web 是管理和校验入口：agent / CLI / REST / MCP 仍然是批量生产主入口，Web 负责补齐人工维护、查看、标记和最终验收。
- 所有入口共用服务端事实源：Web 不另建一套角色或 recipe 状态，继续读写 `project.metadata_json.visual_context`。

## Actors

- Admin user: 小团队自用管理员，登录 Web 后维护 project context、查看资产和手动标记参考图。
- Story agent: 在平台外写故事脚本、拆分 scene，不要求 Agent ImageFlow 理解小红书运营或自动发布。
- Image agent: 调 MCP / REST / CLI 创建图片任务，传 `character_ids`、`reference_asset_ids`、`prompt_recipe_id`、`story_id`、`scene_id`、`batch_id`。
- Reviewer: 在 Web Recent Assets 或当前 scope 资产库里查看 scene 结果、selected/rejected 状态和 metadata。

## P1-PCTX-008 Scenario Matrix

### 1. 发现和进入 Project Context

- Given Admin 已登录且 Web 当前是 managed mode
- When 用户在顶部或 Scope 区域点击 Project Context / Visual Context
- Then 打开稳定 modal 或 panel，显示当前 workspace / project / campaign 标识、context 更新时间、characters / references / prompt recipes 数量
- Edge: 未登录时显示明确 unauthorized/login required；scope 缺 project 时提示先选择 project；不得表现成“空数据”

### 2. 查看已有角色卡

- Given project 已有 `dog_mochi`、`dog_biscuit`、`cat_orange`
- When 打开 Characters 区域
- Then 能看到 name、role/species、status、主参考 asset、reference 数量、外观摘要和禁止项摘要
- Edge: 长外观描述和 negative prompt 要折叠或换行；窄屏不能横向溢出；archived 角色默认弱化显示或可筛选

### 3. 新增 / 编辑 / 归档角色卡

- Given 用户要新增固定角色
- When 填写 id、name、role/species、appearance、personality、forbidden、primary_asset_id、reference_asset_ids
- Then 保存后写入同一个 `visual_context.characters`，再次打开仍可看到
- Edge: id 为空、重复 id、引用跨 project asset、引用不存在 asset 要显示错误；归档角色不删除历史任务快照

### 4. 把资产标记为项目参考图

- Given 资产卡显示一张当前 project 下的 generated/selected asset
- When 用户点击 Mark as reference
- Then 可选择 purpose=`character|style|scene|prop`，可选 linked character、label、weight、notes，保存后新增 reference binding
- Edge: 只创建/更新 reference binding，不复制文件、不改变 asset 状态、不删除原 asset；跨 project asset 必须被服务端拒绝；同一 asset 多用途绑定要可理解

### 5. 查看和维护 Prompt Recipe

- Given project 有 `pet_story_cover` recipe
- When 打开 Prompt Recipes 区域
- Then 能看到 recipe name/status、角色块、风格块、镜头块、渠道要求、negative prompt、默认画幅/provider/model/generation config 摘要
- Edge: 第一版可以用结构化表单 + JSON preview/edit 兜底；不做模板市场、版本比较或多人协作

### 6. Web 托管任务选择 Project Context

- Given Web 当前处于服务端托管模式
- When 用户创建图片任务
- Then 可以选择 prompt recipe、多个 characters、多个 reference assets，并可开启 `use_project_visual_context`
- Then 创建任务请求携带 `character_ids`、`reference_asset_ids`、`prompt_recipe_id`、`use_project_visual_context`
- Edge: browser direct/legacy provider 模式不需要接入；选择项缺失或 archived 时应提示；显式 prompt、provider、aspect、generation_config 仍优先于 recipe 默认值

### 7. 失败和恢复

- Unauthorized: 面板和保存动作都显示登录态问题，不误报为空 context。
- Network error: 保留当前可见 context，显示保存失败，避免刷新闪空。
- Stale data: 保存前以最新服务端 context 为准；第一版可采用 reload-before-save 或保存后重新拉取，不做复杂冲突合并。
- Empty state: 新 project 下显示“暂无角色/参考/recipe”的创建入口，而不是空白屏。
- UX smoothness: 打开/保存/切换 tab 不触发整页闪烁；常见入口可复用 P1-UX 的 stable lazy modal fallback。

## P1-PCTX-008 Minimal Feature Design

- Entry: 顶部或 Scope 区域增加轻量 Project Context 入口，优先 modal/panel，不新增完整后台页面。
- Data access: 复用 `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/visual-context`，第一版保存完整 `visual_context` 文档；不新增表或迁移。
- Sections: Characters、References、Prompt Recipes 三个 tab 或分区；Quality defaults 只做 recipe 默认参数摘要，不另起复杂 profile 编辑。
- Character form: 支持最小字段 `id/name/status/role/appearance/personality/forbidden/primary_asset_id/reference_asset_ids`。
- Reference binding: 支持从 Project Context 面板新增，也支持从 asset card 标记；purpose 只允许 `character/style/scene/prop`。
- Recipe form: 支持 `id/name/status/prompt_blocks/negative_prompt/default_aspect_ratio/default_output_format/default_provider/default_model/generation_config`。
- Task integration: Web managed create task 增加 context selector；只负责把 ids 传给服务端，不在前端拼最终 prompt。
- Guardrails: 不展示 provider key；不要求 project API key 才能使用 Admin Console；不把 reference library 扩成通用 DAM。

## P1-PCTX-009 Scenario Matrix

### 1. 准备长期 project context

- Given 一个萌宠账号 project，例如 `prj_pet_story_regression`
- When 设置 visual context
- Then project 下有两只狗和一只橘猫角色卡、一个 style reference、一个 `pet_story_cover` prompt recipe
- Evidence: `GET /visual-context` 能返回 characters、references、prompt_recipes，且没有 provider secret

### 2. 外部 agent 拆故事并创建 campaign

- Given Story agent 在平台外写好三幕故事
- When Image agent 创建或使用 story campaign，例如 `cmp_pet_story_regression`
- Then 每个 scene task 都带 `source/session_id/batch_id/story_id/scene_id/target_path`
- Boundary: 平台不读取原始小红书脚本、不做发布计划、不决定发文时间

### 3. 三个 scene 任务批量生成

- Given scene 001/002/003 分别引用不同 character 组合和同一个 recipe
- When 通过 CLI、REST 或 MCP 创建任务
- Then mock provider 生成每个 scene 的候选 asset，任务进入 completed，asset 有 original / thumbnail / metadata URL
- Edge: 如果一个 scene 失败，batch progress 应显示 failed/retry，而不是让整个 project context 不可用

### 4. 资产可追溯和可查看

- Given scene assets 已生成
- When 查询 task、asset list、metadata、Admin Recent Assets 或 Web 当前 scope
- Then 每张 asset 都能追溯 `character_ids`、`reference_asset_ids`、`prompt_recipe_id`、`visual_context_snapshot`、`story_id`、`scene_id`、`batch_id`
- Then Web 能看到缩略图、selected/rejected 状态、metadata/parameters 摘要和 scope

### 5. 最小验收不追求完美运营闭环

- Does: 验证 project context 维护、任务创建、mock 生成、资产落盘、metadata、batch progress、Web 查看。
- Does not: 不做小红书发布、不做内容日历、不做最终排版、不做视觉质检 AI、不做复杂 story board。

## P1-PCTX-009 Minimal Regression Design

- Fixture: 使用独立 project/campaign/session/batch，避免旧 smoke 数据污染。
- Context: 三个角色、一个 style reference、一个 pet story recipe。
- Tasks: 2-3 个 scene task，每个 task 请求 1-2 张 mock asset。
- Checks: visual context readback、task completion、asset count、thumbnail/metadata URLs、batch progress、asset list filters、Admin Recent Assets/Web visibility。
- Documentation: 把 task ids、asset ids、batch progress 和 Web 观察写回 CHECKPOINTS / RUNBOOK / CSV evidence。
- Safety: mock provider only；不读取、打印或处理任何 key；如果 project 已启用 key，只验证错误状态或使用用户显式提供的非输出方式。

## Implementation Order For Next Threads

1. Implement P1-PCTX-008 Web read-only shell: Project Context entry, load state, empty/unauthorized/error states.
2. Add Characters and Recipes editing: minimal forms, save full context, reload after save.
3. Add References UX: Project Context reference list and asset card Mark as reference.
4. Add Web managed task context selector: pass recipe/character/reference ids to existing create task API.
5. Run P1-PCTX-008 verification: frontend tests, production build, browser mock smoke, no real provider.
6. Execute P1-PCTX-009 regression: clean pet-story project/campaign, mock scene tasks, REST/CLI/MCP or chosen minimal paths, Web visibility. Status: done in `slice-039`.
7. Update TASKS / PROJECT_PLAN / PROJECT_STATUS_MAP / CHECKPOINTS / RUNBOOK / CSV evidence. Status: done in `slice-039`.

## Acceptance Criteria Summary

- Admin can manage current project visual context in Web without touching API keys.
- Asset cards can mark current-project assets as project references without changing asset file/status.
- Web managed task creation can pass context ids to service core.
- Mock pet story batch proves characters, reference, recipe and scene metadata survive into task and asset snapshots.
- Web remains smooth enough for the core workflow: no whole-screen blanking, no false empty state on unauthorized, no narrow-screen overflow.

## Open Risks

- Full-document save can overwrite concurrent edits; acceptable for single-admin first pass, but implementation should reload after save and keep errors visible.
- Long prompt blocks can make forms unwieldy; first pass should favor collapsible textareas and summaries over complex editors.
- Reference binding can drift if source asset is later cleaned up; existing storage governance protects selected/published, and P1-PCTX-009 should use protected or current mock assets for references.
- Batch / Story / Scene View is still a later product area; P1-PCTX-009 uses batch progress and asset filters instead of building a new storyboard UI.
