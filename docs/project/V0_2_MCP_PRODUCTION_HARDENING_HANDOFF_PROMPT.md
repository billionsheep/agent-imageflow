# V0.2 MCP Production Hardening Handoff Prompt

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
1. 先实现 P0：agent-friendly project/campaign/context setup contract，以及 Project Visual Context reference diagnostics。
2. 再实现 P1：caption speaker/bubble anchor semantics、panel state transition semantics、caption derivative delivery semantics、NAS storage adaptation and delivery governance。
3. P2 只在 P0/P1 完成后继续：partial success 语义、single asset readable summary、本地 Web review packaged environment。

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

第一步请先审视 CSV 中 P0/P1 的依赖关系，给出实现顺序，然后开始执行第一个可独立闭环的切片。
```
