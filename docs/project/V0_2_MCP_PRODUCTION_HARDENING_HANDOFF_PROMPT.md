# V0.2 MCP Production Hardening Handoff Prompt

注：`v0.2.0` 已在本地完成并进入正式发版收口。本文件保留为历史 handoff 材料；新会话默认不再从这里继续实现 `V02-MCPH-*`，而应优先查看 `docs/project/PROJECT_STATUS_MAP.md`、`issues/README.md`、`issues/next-phase-p1-server-deployment-rehearsal.csv` 和 `issues/next-phase-p1-final-delivery-nas-readable-export.csv`。

下面提示词可直接复制到新会话，用来推进下一阶段实现。默认先做 P0/P1，不运行真实 provider，除非用户明确确认费用和图量。

```text
你现在在 /Users/moon/Workspace/tools/agent-imageflow 项目中工作。

请作为 Agent ImageFlow 的 PM + senior engineer，推进 v0.2 MCP Production Hardening。目标不是扩 Web 创作工具，而是把平台作为 MCP-first 图片资产生产能力继续收稳：agent 可接入、上下文可准备、reference 可诊断、caption/panel 语义更清晰、NAS 交付可治理。

请先阅读并遵守：
- AGENTS.md
- docs/project/README.md
- docs/project/V0_2_MCP_PRODUCTION_HARDENING.md
- issues/next-phase-v0-2-mcp-production-hardening.csv
- docs/project/PET_STORY_PRODUCTION_WORKFLOW.md
- docs/project/STORY_CONTINUITY_AGENT_GUIDE.md
- docs/project/MCP_SERVICE_GUIDE.md
- docs/project/TASKS.md
- docs/project/PROJECT_STATUS_MAP.md
- docs/project/V1_BASELINE_AND_ROADMAP.md
- docs/project/DECISIONS.md
- docs/project/CHECKPOINTS.md

当前产品定义：
Agent ImageFlow 是服务器托管的图片资产生成与管理能力平台。外部 agent / 自动化 / CLI / REST / Web 可以创建图片任务，服务端负责 provider 调用、资产落盘、缩略图、metadata、select/reject、manifest 和 delivery。Web 是审图、管理、诊断控制台，不是主创作工具。

本轮优先级：
1. 先实现 P0：`V02-MCPH-002 agent-friendly project/campaign/context setup contract` 与 `V02-MCPH-003 Project Visual Context reference diagnostics` 已完成；`V02-MCPH-004/005/006` 也已完成。
2. `V02-MCPH-007/008/009/010` 已完成：partial-success runtime semantics、single-asset readable summary、NAS 治理口径和本地 packaged review 路径都已落地。
3. `V02-MCPH-011 Controlled pet business regression trial` 已完成；`V02-MCPH-002` 到 `V02-MCPH-011` 均已闭环，本文件仅保留实现边界与证据追溯。

当前状态：
- `V02-MCPH-002` 已完成：服务端新增 `AGENT_SETUP_TOKEN` / `X-Agent-Setup-Token` 非 destructive bootstrap 路径，可读取/创建 workspace、project、campaign，并读取/更新 project visual context。
- `V02-MCPH-003` 已完成：GET project visual context 响应顶层返回 `reference_diagnostics`；task metadata / structured input 会保留 `project_visual_context_diagnostics`；batch summary / manifest 的 `visual_context.reference_diagnostics` 可追溯；Web `ProjectContextModal` 有只读诊断卡。
- `V02-MCPH-004` 已完成：`metadata_json.caption_lineage` 现支持 `speaker_character_id`、`bubble_anchor`、`tail_direction`、`caption_intent` 和 `avoid_covering_subjects`；服务端会归一化到 `structured_input_json.caption_lineage`，并把这些语义追加为 provider 可见的 prompt 约束。
- `V02-MCPH-005` 已完成：`story_context_v1.panel_plan` 现支持 `emotion_before`、`emotion_after`、`pose_change`、`relationship_shift`、`must_change`、`must_not_keep` 和 `state_transition_notes`；服务端会把这些状态推进字段回写到 task metadata，透传到 summary / manifest continuity，并追加成 provider 可见的 `State transition requirements` prompt 约束。
- `V02-MCPH-006` 已完成：`metadata_json.caption_lineage` 新增 `auto_select_derivative`；`manual_optional` caption derivative task 在该标志为 true 时会自动选中第一张派生图；`batch summary` / `batch manifest` 已新增只读 `delivery_role`，并在 `selected_only` manifest 中优先保留 `final_delivery`。
- `V02-MCPH-007` 已完成：task、batch summary、manifest 和 Web 技术详情现在统一输出 `requested_count`、`delivered_count`、`partial_success_reason` 与 `provider_error_summary`。
- `V02-MCPH-009` 已完成：单资产 `asset` / `metadata` 响应新增只读 `asset_summary`，Web 标题、概览和技术详情优先消费该摘要。
- `V02-MCPH-008` 已完成：`RUNBOOK.md` 与 `SERVER_DEPLOYMENT_GUIDE.md` 已固定 NAS/self-host 边界，明确 DB/metadata/manifest 是事实源，storage root bind mount + Postgres/storage 一致备份是第一版治理方式。
- `V02-MCPH-010` 已完成：`RUNBOOK.md` 已提供标准本地 packaged review 命令链和按 `session_id` / `batch_id` / `story_id` replay 的步骤。
- MCP 工具仍只有 6 个安全工具，没有新增 setup/delete 工具。
- 不要重复重开 `V02-MCPH-002` 到 `V02-MCPH-011`；`v0.2.0` 之后的新工作应转到服务器部署演练、真实业务试用、final delivery / NAS readable export 或 Settings IA。

重要边界：
- 不做 Web 漫画编辑器、通用 DAM、小红书发布、内容日历、账号运营后台。
- 不给 MCP 增加 destructive tools，不允许 MCP 删除 workspace/project/campaign/asset。
- 不读取、不打印、不处理任何 provider key、API key、Basic/Auth 密码、Admin cookie/session 或 secret。
- 默认不运行真实 provider；真实 canary 需要用户明确确认费用、图量和停止条件。
- 尽量复用 metadata / structured input；如果发现必须数据库迁移，先暂停并给 migration/backup 计划。
- 每完成一个切片，必须同步更新 CSV、TASKS、PROJECT_STATUS_MAP、V1_BASELINE_AND_ROADMAP、DECISIONS、CHECKPOINTS 和相关 guide/examples。

建议执行方式：
- 开始前检查 git status，保护已有未提交业务试跑证据。
- 对每个切片先写/调整测试或 fixture，再实现最小代码。
- 优先用 mock/provider fake 验证；Web 只补审图/诊断展示，不做复杂创作入口。
- 完成后运行相关 Go/Web tests、CSV parse、git diff --check，并汇报未跑真实 provider的原因。

第一步请先确认当前会话是否真的需要追溯 `v0.2.0` 的实现细节；如果不是版本回顾或 bug 定位，不要再从这个 handoff 开始，而是直接转到当前活跃 CSV。
```
