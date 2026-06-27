# V1 Baseline and Roadmap

本文档记录 Agent ImageFlow 当前第一版基线、可验收范围、剩余任务和未来方向。它用于后续部署、试用和新一轮 CSV 拆分，不替代 `TASKS.md` 或 `PROJECT_STATUS_MAP.md`。

`v0.1.0` 仍保留为首个 V1 baseline tag；当前代码与文档已经收口到 `v0.2.1`，作为最新 MCP-first production hardening 发布版本。后续工作按版本化维护推进：`v0.2.x` 之后优先补服务器部署演练、final delivery / NAS 可读交付层和真实业务生产试用，再决定 `v0.3.x` 是否继续强化 IP 工作流或 Settings 信息架构。

2026-06-25 后新增的产品判断：连续漫画不是简单多次单图生成。下一阶段应把“固定角色 + 固定场景 + 连续故事 + 加字派生”拆成 Story Bible、Panel Plan、reference roles、Story Continuity Agent 和 Caption/Edit Lineage；平台不承担创作脑，不扩成漫画编辑器或运营后台。外部评审后已完成 Story Continuity MVC 平台侧收敛；当前下一步不再把每个 smoke 当产品需求，而是按 `docs/project/PET_STORY_PRODUCTION_WORKFLOW.md` 执行 MCP-first 真实萌宠故事生产试用。

2026-06-26 后的 v0.2 默认方向是 MCP Production Hardening：`docs/project/V0_2_MCP_PRODUCTION_HARDENING.md` 与 `issues/next-phase-v0-2-mcp-production-hardening.csv` 已作为下一版本执行入口。核心不是扩 Web 创作界面，而是把 agent 接入、上下文准备、reference 诊断、caption/panel 结构语义、caption 派生交付和 NAS 治理补成可维护的生产能力。

2026-06-27 新增的产品判断是：人工复盘、再次寻找和 NAS 浏览比 agent 取图更痛。当前已完成 `issues/next-phase-p1-final-delivery-nas-readable-export.csv` 第一轮 `P1-DLV-001/002/003/005/008`：继续复用现有 `batch-manifest`，补出 `view=final_delivery`、`manifest_view` 和 `final_delivery` block，让人工可按 `story/scene/batch` 直接看最终交付图；同时落地 batch-first NAS readable mirror，让运维可把 `manifest.final.json`、final originals 和 thumbnails materialize 到 `STORAGE_ROOT/final-delivery-mirror/workspaces/<workspace>/projects/<project>/campaigns/<campaign>/[sessions/<session>/]batches/<batch>`。后续 `P1-DLV-004/006/007` 仍只保留为 story/batch export pack、project delivery defaults 和治理联动方案，canonical storage 继续保持不变。

## V1 Baseline

V1 的产品定义：

Agent ImageFlow 是 AI 可调用的图片资产生成与管理能力平台。它让 Codex、Claude、Cursor、自动化脚本、内容系统或业务后台通过 MCP / REST API / CLI / Web 创建图片资产，并拿到可追踪、可审核、可复用、可交付的正式结果。

V1 已具备：

- 多入口统一事实源：Web / MCP / REST / CLI 共用 Go application core。
- 服务端闭环：create task -> worker -> provider -> 文件落盘 -> asset/version/metadata 登记 -> thumbnail -> delivery URL。
- Scope 隔离：workspace / project / campaign / task / asset 层级已可用。
- Provider：mock、openai-compatible、fal 基础适配和输入复用已接入。
- 资产治理：select/reject、best-of、auto reject、thumbnail、metadata、repair/reconcile、storage governance、integrity view。
- 可见性：Web Recent Assets、服务端资产库、筛选、分页、lazy loading、Admin session、跨 scope 最近资产。
- Project Visual Context：project 级角色卡、reference binding、prompt recipe、quality defaults 和 task/asset 快照。
- Batch / Story / Scene：batch summary、Production View、scene asset actions、scene regenerate、engineering/final-delivery 双视图 JSON manifest。
- Web 审图体验：长 ID 和工程字段默认折叠，卡片优先显示图片、剧情/画面摘要、story/scene/source/created/target 和状态。
- Web 控制台入口：未登录时只显示 Admin 登录页；登录后进入完整服务器托管控制台，使用服务器配置的 provider 能力，资产库不再二次登录。
- 部署发布：GitHub Actions 构建 GHCR 私有 API/Web 镜像，服务器用 `docker-compose.prod.yml` 拉取镜像运行。
- 部署可用性：Web 生产镜像可同源代理 `/api/*` 与 `/healthz`，Admin session 可读取图片 delivery，控制台提供安全 runtime status 和 Current Scope 导航。

V1 已验证的关键真实场景：

- 萌宠账号图片资产生产最小闭环：项目视觉上下文、prompt recipe、真实 provider、落盘、缩略图、metadata、Recent Assets 和 Production View 已打通。
- MCP 工具入口真实 provider 1 图 canary：MCP `create_image_task` 已证明不只停留在 mock 路径。
- 服务器部署路径：`main` 分支 push 后 Docker Publish workflow 已构建并发布 API/Web 镜像。

文档入口已收敛：

- 日常入口：`docs/project/README.md`。
- CSV 索引：`issues/README.md`。
- 历史 slice 索引：`docs/project/stories/README.md`。
- 已完成的 P0/P1/P2 CSV 和 slice 默认不复开，后续新增需求新建独立 CSV 或 story。

## V1 Scope Boundary

V1 做：

- 图片任务、生图 provider 适配、资产落盘、缩略图、metadata、审核/选中状态。
- MCP / REST / CLI / Web 共用同一事实源。
- project 作为长期视觉生产上下文。
- batch/story/scene 的生产查看、单 scene regenerate 和 JSON manifest。
- 小团队自托管部署、Admin session、project API key、Basic Auth、限流和审计。

V1 不做：

- 小红书发布、内容日历、账号运营后台。
- 泛设计平台、白板、模板市场、通用 DAM。
- SaaS 注册登录、多租户、复杂 RBAC、计费。
- 每用户 provider key 管理。
- 内置 WebDAV/SMB server。
- 大规模真实 provider benchmark。
- 视觉质检 AI 自动裁决。
- 数据库 migration 框架。

## Immediate Remaining Tasks

这些是 V1 之后最应该先做的运维/验收任务。

0. V0.2 MCP Production Hardening（已完成）
   - 当前入口为 `docs/project/V0_2_MCP_PRODUCTION_HARDENING.md` 和 `issues/next-phase-v0-2-mcp-production-hardening.csv`。
   - `V02-MCPH-002 Agent-friendly project/campaign/context setup API contract` 已完成：服务端通过 `AGENT_SETUP_TOKEN` / `X-Agent-Setup-Token` 放开非 destructive bootstrap 路由，`vag` 也可通过 `AGENT_IMAGEFLOW_SETUP_TOKEN` 自动转发。
   - `V02-MCPH-003 Project Visual Context reference diagnostics` 已完成：GET project visual context 会返回 `reference_diagnostics`，task metadata / structured input 会保留 `project_visual_context_diagnostics`，batch summary / manifest 的 `visual_context.reference_diagnostics` 可追溯，Web `ProjectContextModal` 也已新增只读诊断卡。
   - `V02-MCPH-004 Caption speaker / bubble anchor semantics` 已完成：`metadata_json.caption_lineage` 现可显式表达 `speaker_character_id`、`bubble_anchor`、`tail_direction`、`caption_intent` 和 `avoid_covering_subjects`；服务端会归一化到 `structured_input_json.caption_lineage`，并把这些语义追加为 provider 可见的 caption prompt 约束。
   - `V02-MCPH-005 Panel state transition and performance progression semantics` 已完成：`story_context_v1.panel_plan` 现在支持 `emotion_before`、`emotion_after`、`pose_change`、`relationship_shift`、`must_change`、`must_not_keep` 和 `state_transition_notes`；服务端会把这些状态推进语义回写到 metadata，并追加成 provider 可见的 `State transition requirements` prompt 约束，summary/manifest continuity 与 Web Production View 也能直接读到。
   - `V02-MCPH-006 Caption derivative delivery semantics` 已完成：`caption_lineage` 现支持 `auto_select_derivative`，服务端会在 `manual_optional` caption derivative task 中按该标志自动选中第一张派生图；`batch summary` / `batch manifest` 现已输出只读 `delivery_role`，并在 `selected_only` manifest 中把 `base_original`、`caption_derivative` 和 `final_delivery` 区分清楚。
   - `V02-MCPH-007 Provider partial success product semantics` 已完成：task、batch summary、manifest 与 Web 技术详情现在统一输出 `requested_count`、`delivered_count`、`partial_success_reason` 和 `provider_error_summary`，调用方不再把 provider 少回候选误判为平台全失败。
   - `V02-MCPH-009 Single asset readable production summary` 已完成：单资产 `asset` / `metadata` 响应新增只读 `asset_summary`，Web 审图标题、概览和技术详情已直接消费该摘要。
   - `V02-MCPH-008 NAS storage adaptation and delivery governance` 与 `V02-MCPH-010 Local Web review packaged environment` 已完成第一版文档/运维收口：自托管部署固定以 storage root bind mount + Postgres/storage 一致备份为边界；本地 packaged review 固定走 `docker compose` + `npm --prefix web run preview` 标准路径。
   - 2026-06-27 已完成 `V02-MCPH-011 Controlled pet business regression trial`：在本地低并发真实 provider 下完成 2 格情侣文案卡、3 格低背景连续故事和少量 caption derivative 试跑，reference diagnostics、speaker/bubble anchor、panel state transition、caption derivative final delivery、final-delivery manifest 和 delivery URL 均通过；`v0.2.0` 功能范围已闭环。
   - P1：caption speaker/bubble anchor，panel state transition，caption derivative delivery，NAS storage adaptation and delivery governance。
   - P2：provider partial success semantics，single asset readable summary，local Web review packaged environment。
   - 默认不运行真实 provider；业务回归试用需用户确认费用和图量。

1. 服务器/NAS 部署复验
   - 使用 `docs/project/SERVER_DEPLOYMENT_GUIDE.md`。
   - 当前已新增 `issues/next-phase-p1-server-deployment-rehearsal.csv` 和 `docs/project/stories/slice-053-server-deployment-rehearsal.md` 作为演练入口。
   - Volcengine 旧服务已完成 GHCR `main` 更新、升级前备份、临时 HTTP health/Web、MCP `tools/list` 和 mock benchmark smoke。
   - 准备服务器 `.env.prod`，不要提交或打印。
   - 推荐 `PUBLIC_BASE_URL` 指向 Web/HTTPS origin；Web 镜像可代理 `/api/*` 与 `/healthz` 到内部 API。
   - 继续补 HTTPS/Caddy 正式同源入口、浏览器 Admin、Recent Assets 缩略图、original/metadata delivery smoke。
   - 演练一次 restore 和 `IMAGE_TAG` 回滚。

2. 真实业务生产试用
   - 当前入口为 `docs/project/PET_STORY_PRODUCTION_WORKFLOW.md` 和 `issues/next-phase-p1-pet-account-real-workflow-trial.csv`。
   - 准备真实 project/campaign、角色参考、环境或风格参考、prompt recipe 和非敏感 provider/model 摘要。
   - 由外部 Story Continuity Agent 产出 3-6 格 Story Bible、Panel Plan 和 `story_context_v1`。
   - Agent 通过 MCP 顺序创建 panel task、查询状态、select/reject、拿 delivery info；Web 只作为审图/管理/manifest 控制台。
   - 观察平台资产模型、agent 编排、provider 参考参与、Web 审图和 manifest/NAS 交付摩擦。
   - mock 3 格数据链路、1 图真实 provider canary、delivery spot check 和 Web 截图只作为可选证据；不把它们自动升级为产品需求。

3. Story Continuity / Comic Workflow
   - `issues/next-phase-p1-story-continuity-mvc.csv` 仍记录平台侧 MVC 状态：`story_context_v1`、panel causality、reference_bindings / resolved_reference_assets 分离、sequential preflight、Production View 最小连续性展示和 manifest 摘要已完成。
   - 后续 Story Continuity 能力优先服务真实萌宠生产试用；mock smoke 只证明数据链路，不证明视觉连续性。
   - `issues/next-phase-p1-story-continuity-comic-workflow.csv` 作为上位路线保留，待真实试用总结后再决定是否拆 Story Review 等扩展。

4. Caption / Edit Lineage
   - MCP-first 最小切片已完成：基于固定 asset 加字 edit 的结果可通过 `caption_lineage` 表达派生资产摘要。
   - 记录 `derived_from_asset_id`、`derivation_type=caption_edit`、`caption_text`、`caption_style`、`source_task_id` 和 `source_scene_id`。
   - 当前入口为 `issues/next-phase-p1-caption-edit-lineage.csv`；Web 加字入口、批量 caption 和 renderer 预留仍后置到 Story Continuity MVC 之后。

5. MCP Service Pack smoke
   - 接入 guide、MCP config 示例、萌宠 scene 示例和 smoke 说明已落地。
   - 下一步只需回填 `tools/list`、mock create task、get task 和 delivery info 的实际 evidence。
   - 继续明确 Project API Key、Basic Auth、Admin Login 和 provider key 的边界。

6. Character Reference Intake and Consistency
   - 后端已支持把 campaign input-files 提升为正式 project reference asset。
   - 已支持角色主图/参考图字段、任务自动带参考资产、provider 参考参与 metadata 和参考图失败诊断。
   - Web 角色卡已显示主图/参考图缩略图和缺图警告。
   - 从资产卡打开 Project Context 时，已可把当前 asset 设为角色主图、加入角色参考图，或保存为项目参考图。
   - 下一步补完整 mock pet consistency smoke、browser smoke 和人工确认后的 1 图真实参考 canary。

7. Web Review Feedback and Stability
   - Web 前端和 tests 范围已完成：Select / Reject 后卡片、scene header、coverage count 和按钮状态会局部变化。
   - 下拉和筛选刷新保留旧内容，审图请求已有去重、旧响应丢弃和 429 友好提示。
   - 下一步只建议补一次 browser smoke evidence。

8. Safe Delete and Trial Reset
   - CLI + Admin-only REST foundation 已完成：cleanup preview/execute 可按 asset/task/session/batch/story/campaign 定位候选。
   - Web Scope 管理已提供当前 campaign 的数据清理入口，可 preview/execute，并要求确认短语。
   - 继续要求 dry-run token 或显式确认，保护 selected/published/approved 并写 audit。
   - 单 asset archive/restore 已补到 Admin REST/Web/CLI；archived 默认不进 cleanup。
   - 下一步补 task/input-file reset、完整 browser smoke 和生产备份演练；MCP 第一轮不删除 workspace/project/campaign/asset。

9. 发布版本策略
   - `v0.1.0` 保留为首个 V1 baseline tag。
- 当前发版目标为 `v0.2.1`：Git tag 为 `v0.2.1`，GHCR 镜像发布 `v0.2.1` 与 `sha-<commit>`。
   - 服务器升级、HTTPS 同源 smoke 和 IMAGE_TAG 回滚仍独立于本地代码发版执行。

10. 备份与恢复演练
   - Postgres dump。
   - storage root / NAS snapshot。
   - `.env.prod` 安全备份。
   - 验证恢复后的 asset original/thumbnail/metadata URL。

## Recommended Next CSV Slices

### P1 Ops: Server Deployment Rehearsal

建议文件名：

```text
issues/next-phase-p1-server-deployment-rehearsal.csv
```

当前状态：

- CSV 与 `docs/project/stories/slice-053-server-deployment-rehearsal.md` 已新增。
- `docs/project/SERVER_DEPLOYMENT_GUIDE.md` 已补部署演练证据模板。
- 本地发布材料仍通过静态检查；Volcengine 旧服务已完成 `main` 更新、升级前备份、临时 HTTP health/Web、MCP `tools/list` 和 mock benchmark smoke；HTTPS/Caddy 正式入口、浏览器 Admin delivery smoke、restore 和回滚仍待执行。

目标：

- 把 GHCR 镜像部署到真实服务器/NAS。
- 完成 HTTPS 反代、Admin 登录、mock 生成、MCP tools/list、备份/回滚 smoke。
- 不运行真实 provider，除非单独确认 1 图 canary。

### P0/P1 Closed: Deployment Auth, Scope and Project Console

记录文件：

```text
issues/next-phase-p0-p1-deployment-auth-scope-project-console.csv
```

结果：

- 已修复生产 Web/API 不同 origin 导致的缩略图鉴权问题。
- 已明确 Basic Auth、Admin Login、Project API Key、provider key 四种凭据语义。
- 已增加 Current Scope 导航和 Project Context 工作区摘要，降低手填长 ID 的操作负担。

### P1 Trial: Pet Account Real Workflow Trial

建议文件名：

```text
issues/next-phase-p1-pet-account-real-workflow-trial.csv
```

目标：

- 用真实萌宠账号工作流跑 1 个 project、1 个 campaign、3-6 个 story panels。
- 验证 Story Continuity Agent 写 Story Bible / Panel Plan / `story_context_v1` -> MCP 顺序生图 -> Web Production View select/reject -> JSON manifest/NAS 交付。
- 只做低频真实 provider canary，不做 benchmark；mock 和 canary 只作为证据，不作为产品需求本身。

当前状态：

- CSV 已从验收清单收敛为必须生产步骤 + 可选证据记录。
- 新增 `docs/project/PET_STORY_PRODUCTION_WORKFLOW.md` 和 `examples/mcp/pet-story-production-plan.json`。
- 本任务用于真实业务生产试用，不直接实现新功能；观察结果应归类到 Story Continuity、Caption Lineage、Settings IA、Safe Delete、provider follow-up 或部署/NAS 运维。

### P1 Delivery: Final Delivery / NAS Readable Export

执行入口：

```text
issues/next-phase-p1-final-delivery-nas-readable-export.csv
```

当前状态：

- 第一轮 `P1-DLV-001/002/003/008` 已完成：`GET /batch-manifest`、`vag batch manifest` 和 Web Production View 已支持 `view=engineering|final_delivery`。
- `view=final_delivery` 会在兼容保留旧顶层 `counts/tasks/assets/scenes/stories` 的同时，追加 `final_delivery.counts/stories/scenes/final_assets`，方便人工按 story/scene/batch 查看最终交付图。
- `final_delivery.final_assets` 统一以 scene 内 `delivery_role=final_delivery` 的资产为准，并把 caption 派生关系扁平为 `derived_from_asset_id` / `derivation_type`；响应继续不暴露 `local_path`、cookie、session 或 secret-like 字段。

剩余待做：

- `P1-DLV-004` story/batch export pack。
- `P1-DLV-005` NAS readable mirror。
- `P1-DLV-006` project delivery defaults。
- `P1-DLV-007` export/mirror 与 archive/restore/cleanup/restore drill 的治理联动。

### P1 Story: Story Continuity / Comic Workflow

第一执行入口：

```text
issues/next-phase-p1-story-continuity-mvc.csv
```

目标：

- 验证“3 格、无字、顺序生成、人工选图、真实参考图参与”的连续故事最小闭环。
- 统一 `story_context_v1`，区分 `reference_bindings` 与 `resolved_reference_assets`。
- 增加 preflight，避免上一格 selected asset 缺失时静默退化为纯文生图。
- 第一轮复用 Production View / Technical details 和 manifest，不新建完整 Story Review 页面。
- 真实 provider canary 限定 cap=1、每格最多 2 候选、总图量最多 8 张。

当前实现状态：

- 后端 `CreateTask` 已能解析 `metadata_json.story_context_v1`，展开 reference bindings，并在 task `structured_input_json`、provider parameters、batch summary 和 manifest 中输出 continuity 摘要。
- Sequential Previous Panel Mode 已强制 `selection_mode=manual_optional`，且 panel 2/3 必须引用上一格 selected asset。
- Web Production View 已显示 `panel_index`、`narrative_role`、`dialogue`、`previous_panel_asset_id`、resolved reference count、provider reference participation 和 continuity warnings。
- MCP/mock 3 格数据链路已完成，可证明 metadata/status/select/manifest 链路；真实 provider canary 仍需费用确认后执行，且不应把 mock 结果误判为视觉连续性已验收。

上位路线文件：

建议文件名：

```text
issues/next-phase-p1-story-continuity-comic-workflow.csv
```

目标：

- 解决“多张图只是同风格散图，不是连续故事”的问题。
- 定义 Story Bible、Panel Plan、Reference Roles 和 Continuity Metadata。
- Web 后续可按 story/scene 顺序展示候选图、对白、动作、参考图、已选状态和 regenerate 入口。
- 第一版优先复用 metadata，不做复杂数据库迁移，不做漫画编辑器或 AI 自动视觉质检。

### P1 Agent: Story Continuity Agent

建议文档：

```text
docs/project/STORY_CONTINUITY_AGENT_GUIDE.md
examples/mcp/story-continuity-agent.local.json
examples/mcp/create-story-bible.json
examples/mcp/create-panel-plan.json
examples/mcp/run-3-panel-story-smoke.md
```

目标：

- 让额外 agent 负责故事连续性、分镜、reference choices 和重试策略。
- Agent 只调用 MCP 安全工具，不持有 provider key、Admin cookie 或删除权限。
- 新 agent 先跑 3 格 mock story，再在人工确认费用后做真实 provider canary。

### P1 Caption: Caption Edit Lineage

建议文件名：

```text
issues/next-phase-p1-caption-edit-lineage.csv
```

目标：

- 把“基于固定 asset 加可爱文字”的 edit 结果纳入正式派生资产语义。
- 记录 `derived_from_asset_id`、`derivation_type`、`caption_text`、`caption_style` 和 source task。
- Web 后续提供“基于此图加字”入口；Story 工作流后续支持 selected panels 批量加字。
- 第一版保留 future caption renderer slot，区分风格化 AI edit 和稳定确定性贴字。

当前状态：

- MCP-first 最小 contract 已完成：`metadata_json` 中的 caption lineage 会写入 `structured_input_json.caption_lineage`，provider parameters 输出 `caption_lineage`，batch manifest asset 透传 `caption_lineage`。
- 已新增 `examples/mcp/create-caption-edit-task.json`，示例使用原 `asset_id`、`role=edit_target` 和 `provider=mock`，不包含真实 key。
- 未实现 Web 一键加字、批量 caption UI、renderer 或真实 provider canary。

### P1 Agent Onboarding: MCP Service Pack

建议文件名：

```text
issues/next-phase-p1-mcp-service-pack.csv
```

当前状态：

- 已新增 `docs/project/MCP_SERVICE_GUIDE.md`、`examples/mcp/agent-imageflow.local.json`、`examples/mcp/create-pet-scene.json` 和 `examples/mcp/smoke.md`。
- 已明确 Project API Key、Basic Auth、Admin Login 和 provider key 的边界；MCP 配置示例不写真实 secret，也不需要 Admin cookie 或 provider key。
- 已补删除边界：MCP 继续负责 create/get/list/select/reject/delivery，删除和试用重置走 Admin Web/REST/CLI。

剩余目标：

- 跑 `tools/list`、mock create task、get task、list assets 和 delivery info smoke。
- 把 smoke evidence 回填到 CSV / CHECKPOINTS。
- 暂不做远程 HTTP MCP、新账号系统、多用户权限、provider key 下发、新工具 schema 大改或真实 provider 默认 smoke。

### P1 Visual Consistency: Character Reference Intake

建议文件名：

```text
issues/next-phase-p1-character-reference-intake-consistency.csv
```

当前状态：

- 后端已支持 input-file promote 为正式 reference asset，并写入 asset/version/thumbnail/metadata。
- Character Profile 已支持 `primary_asset_id`、`reference_asset_ids`、`reference_policy` 和 `appearance_lock_notes`。
- CreateTask 使用 `character_ids` 与 `use_project_visual_context=true` 时可自动展开角色参考资产，并在 task/asset metadata 中记录参考参与诊断。
- Web Project Context 角色卡已展示主图/参考图缩略图和缺图警告；从资产卡带 `asset_id` 打开时，已提供“设为主图 / 加入参考图 / 保存为项目参考图”的快捷绑定区。

剩余目标：

- 跑完整 mock pet character consistency smoke。
- 在人工确认费用后跑 1 图真实参考 canary；失败时明确告诉用户参考图没有参与生成。
- 补完整 browser smoke，确认角色缩略图、快捷绑定、参考参与 metadata 和 Web 状态展示不断链。
- 验收分三层：平台链路成功、参考图参与成功、角色一致性人工判断通过。
- 暂不做通用 DAM、模板市场、AI 自动视觉质检、provider key 下发或大规模真实 provider benchmark。

### P1 Web UX Follow-up: Review Feedback and Stability

建议文件名：

```text
issues/next-phase-p1-web-review-feedback-stability.csv
```

当前状态：

- Web 前端与 tests 范围已完成：`ServerAssetLibrary` 与 `ProductionViewModal` 已有 per-asset pending、optimistic status/style、失败回滚、局部 counts 更新、旧内容保留、请求去重/旧响应丢弃和 429 友好提示。

剩余目标：

- 增加 browser smoke，把点击选择/拒绝、切换下拉和 Production View 状态变化纳入回归证据。
- 这是 P1 Web UX Smoothness 之后的真实试用 follow-up，不复开旧 CSV，不大改 Settings 信息架构。

### P1 Data Lifecycle: Safe Delete and Trial Reset

建议文件名：

```text
issues/next-phase-p1-safe-delete-and-trial-reset.csv
```

当前状态：

- CLI `vag storage cleanup-preview/execute` 已支持 `--asset-id`、`--task-id`、`--session-id`、`--batch-id`、`--story-id` 和 `--deprecated`。
- Admin-only REST `storage-cleanup-preview/execute` 已接入，必须使用 Admin session；Project API Key 和 Basic Auth 不授予清理权限。
- Web `ScopeManagerModal` 已提供当前 campaign 的数据清理面板，支持 cleanup preview/execute，展示候选统计、保护计数和脱敏 dry-run token，execute 前必须输入确认短语。
- 清理继续复用 dry-run token / confirm、selected/published/approved 保护和 audit；MCP 不开放 destructive tools。

剩余目标：

- 单 asset archive/restore 已完成；补 task/input-file 级 reset、完整 browser smoke 和生产备份演练。
- 补完整 browser smoke 和 RUNBOOK 运维命令整理，确认 Web 清理入口、CLI/REST foundation、审计和备份策略一致。
- 不做通用 DAM、回收站复杂权限、多用户审批流、MCP 大范围 destructive tool 或绕过备份的生产清库。

### P1 Product: Settings Information Architecture

建议文件名：

```text
issues/next-phase-p1-settings-information-architecture.csv
```

目标：

- 重新设计设置页分组：控制台状态、业务空间、服务端能力、高级/旧模式、数据管理、关于。
- 明确 Admin Login、Basic Auth、Project API Key、provider key/base URL 的边界。
- 先做结构和文案方案，再进入实现，避免继续在旧 playground 设置页上补丁式堆功能。

### P2 Product: Usage Tracking and Edit Lineage

建议文件名：

```text
issues/next-phase-p2-usage-lineage.csv
```

目标：

- 记录资产被用于哪个内容、文章、笔记、封面或导出包。
- 记录 edit/regenerate lineage，明确一张图从哪个 prompt/reference/source task 发展而来。
- 不做小红书发布，不做内容日历。

### P2 Delivery: Export Pack ZIP and Multi Select

建议文件名：

```text
issues/next-phase-p2-export-pack-zip.csv
```

目标：

- 在 JSON manifest 稳定后，支持小批量 selected assets ZIP。
- 限制数量、总大小和路径。
- ZIP 内包含 original、thumbnail、metadata、manifest。
- 不扩展成通用 DAM 下载中心。

### P2 Safety: Deployment and Secret Hardening

建议文件名：

```text
issues/next-phase-p2-deployment-secret-hardening.csv
```

目标：

- Web 显示当前登录态、API host、server-first provider 状态和 safe config 摘要。
- 登录失败限流、Secure cookie / HTTPS 反代检查指南。
- non-exposure regression 覆盖 provider key、project key、cookie/session、local path。
- 不实现每用户 provider key。

### P3 Quality: Visual Consistency Review Aid

建议文件名：

```text
issues/next-phase-p3-visual-consistency-review-aid.csv
```

目标：

- 先做人工审图辅助，不做自动裁决。
- 对角色、style、scene 的 prompt/reference 快照做对比展示。
- 未来再评估 AI 视觉质检。

## Future Direction

产品方向继续保持：

```text
workspace = 个人/团队/客户/业务大边界
project   = 长期经营对象，例如萌宠账号、品牌 IP、产品线、技术图账号
campaign  = 一次具体生产批次，例如一期故事、一组四格漫画、一周封面图
asset     = 生成或上传后可追踪、可审核、可复用、可交付的图片资产
```

下一阶段不要扩成账号运营系统，而是沿着三条线推进：

- 部署线：服务器/NAS 上线、备份、回滚、HTTPS、GHCR 权限。
- 试用线：用真实小批量业务流发现 Web 审图和 agent 调用中的摩擦。
- 产品线：usage tracking、edit lineage、导出包、deployment secret hardening。

继续后置：

- SaaS 化、多租户、复杂 RBAC。
- 小红书自动发布、内容日历、账号运营后台。
- 通用 DAM、模板市场、多人协作审核流。
- 内置 WebDAV/SMB server。
- 大规模 provider benchmark。
- 过早视觉质检 AI。
