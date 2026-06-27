# V0.2 MCP Production Hardening

本文保留为 `v0.2.0` 的范围记录。它承接 2026-06-26 的真实萌宠业务试跑结论：Agent ImageFlow 的 MCP-first 图片资产生产链路已经可用，但要成为可长期复用的 agent 图片能力，还需要把上下文准备、参考图诊断、连续性语义、加字派生和 NAS 交付治理补稳。2026-06-27 已完成 `V02-MCPH-002` 到 `V02-MCPH-011`，本文件从“下一阶段计划”转为“已发布范围说明”。

## 结论

`v0.2.0` 没有把平台扩成 Web 漫画编辑器、通用 DAM 或小红书运营后台，而是收口为 **MCP Production Hardening**：

- 让新 agent 不需要 Admin Web session 或 DB 直连，也能安全准备 project/campaign/context。
- 让平台在生成前后说明 reference 是否真的有图、是否参与 provider、是否存在物种/环境漂移风险。
- 让 caption edit 不只是“加一句文字”，而能表达说话人、气泡位置和派生交付语义。
- 让连续 story 不只是“引用上一格保形”，而能表达情绪、姿态和关系推进。
- 让 NAS/服务器部署下的文件访问、备份、manifest 和 DB 事实源关系清楚。

当前最小业务判断：

- 已证明：2 格、低背景、强台词、明确情侣/萌宠关系的文案卡路线可用。
- 有条件可用：5-6 格低并发连续故事，在人工选图和单格 regenerate 条件下可试用。
- 尚未证明：复杂表演连续性稳定量产、强环境连续性、自动视觉质检、Web 主创作。

## 优先级

### P0: 先补 agent 接入主链

1. Agent-friendly project/campaign/context setup
   - 问题：真实试跑中准备新 project/campaign/context 仍依赖 Admin Web session 或 DB 操作。
   - 目标：给 agent 一条非 destructive、非 cookie、非 provider key 的准备路径。
   - 边界：不授予删除权限；MCP 是否新增工具需先审视 contract。

2. Project Visual Context reference diagnostics
   - 问题：角色参考弱时会狗变熊，平台没有提前报警。
   - 目标：区分 image-backed、text-constrained、missing environment reference、weak species lock。
   - 边界：不做 AI 自动视觉质检，只做配置和参与链路诊断。

### P1: 补生产语义和 NAS 交付

3. Caption speaker/bubble anchor semantics
   - 增加 `speaker_character_id`、`bubble_anchor`、`tail_direction`、`caption_intent` 等轻量字段。
   - 服务端负责快照、prompt 展开、metadata 和 manifest，不做 Web 排版编辑器。

4. Panel state transition semantics
   - 增加 `emotion_before/after`、`pose_change`、`relationship_shift`、`must_change`、`must_not_keep`。
   - 解决“连续性只会保形，不会推进表演”的问题。

5. Caption derivative delivery semantics
   - 明确 base asset、caption derivative、final selected delivery 的关系。
   - 支持显式 `auto_select_derivative`，但默认不覆盖人工已选 base asset。

6. NAS storage adaptation and delivery governance
   - NAS 是 P1，不是 P0：当前没有阻断生成，但会影响长期自托管、备份和人工交付。
   - 第一版靠 storage root bind mount + manifest/target_path + 只读 SMB/WebDAV/Finder 指南。
   - 不内置 WebDAV/SMB server，不把文件夹结构当业务状态。

### P2: 降低使用摩擦

7. Provider partial success semantics
   - 把 `requested_count`、`delivered_count`、`partial_success_reason` 讲清楚。

8. Single asset readable production summary
   - 给单个 metadata/asset card 一个 PM/运营能读的摘要层。

9. Local Web review packaged environment
   - 让本地审图环境一条命令可复放，不再靠临时 preview 记忆。

## 非目标

本版本不做：

- Web 漫画编辑器、画布、图层排版器。
- 小红书发布、内容日历、账号运营后台。
- 通用 DAM、模板市场、多人协作。
- SaaS 注册、多租户、复杂 RBAC、每用户 provider key。
- MCP 删除 workspace/project/campaign/asset。
- AI 自动视觉质检作为裁决。
- 大规模真实 provider benchmark。
- 内置 WebDAV/SMB server。

## 与 MCP 的关系

Agent ImageFlow 的核心仍是服务器托管的图片资产生产能力。Web 是审图、管理和诊断控制台，不是主创作入口。

建议的 agent 分工：

- Story Continuity Agent：写 Story Bible、Panel Plan、reference choices、caption/dialogue、失败重试策略。
- Agent ImageFlow MCP/API：创建任务、查询任务、列资产、select/reject、拿交付、读取/准备必要上下文。
- Admin Web/REST/CLI：删除、归档、恢复、cleanup、备份演练。

MCP 继续不开放 destructive tools。若未来需要清理能力，只考虑受限 archive 或 request-cleanup proposal，并单独确认权限、dry-run 和审计。

## 数据契约建议

第一批尽量复用现有 metadata / structured input，不急着做数据库迁移。

新增或标准化字段：

- `speaker_character_id`
- `bubble_anchor`
- `tail_direction`
- `caption_intent`
- `avoid_covering_subjects`
- `emotion_before`
- `emotion_after`
- `pose_change`
- `relationship_shift`
- `must_change`
- `must_not_keep`
- `state_transition_notes`
- `delivery_role`
- `auto_select_derivative`
- `requested_count`
- `delivered_count`
- `partial_success_reason`
- `asset_summary`

如果实现中发现 metadata 已无法满足查询或一致性要求，再单独提出 migration/backup 计划。

## 验收标准

v0.2 可认为完成时，应满足：

- 新 agent 不用 Admin cookie、DB 直连或 provider key，即可准备一个 campaign 并读取/确认 visual context。
- 任务前能看到角色/参考图/环境 reference 风险；任务后能看到 reference participation。
- caption edit 能明确说话人和气泡锚点，并在派生资产 manifest 中可追溯。
- panel 任务能表达保留项和必须变化项；metadata/manifest 可读。
- selected manifest 能清楚区分 base、caption derivative 和最终交付图。
- NAS 部署文档说明清楚 storage root、只读文件访问、备份、恢复和 DB 事实源边界。
- Web 仍只作为审图和状态控制台，不承担主创作。

## 执行入口

- CSV：`issues/next-phase-v0-2-mcp-production-hardening.csv`
- 新会话提示词：`docs/project/V0_2_MCP_PRODUCTION_HARDENING_HANDOFF_PROMPT.md`
- 背景工作流：`docs/project/PET_STORY_PRODUCTION_WORKFLOW.md`
- Agent 指南：`docs/project/STORY_CONTINUITY_AGENT_GUIDE.md`
- MCP 接入指南：`docs/project/MCP_SERVICE_GUIDE.md`
