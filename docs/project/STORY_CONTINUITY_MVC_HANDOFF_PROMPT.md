# Story Continuity MVC Handoff Prompt

把下面提示词复制到新会话，用于推进第一实施切片。

```text
你现在在 /Users/moon/Workspace/tools/agent-imageflow 项目中工作。

请先阅读并遵守：
- AGENTS.md
- docs/project/README.md
- docs/project/V1_BASELINE_AND_ROADMAP.md
- docs/project/PROJECT_STATUS_MAP.md
- docs/project/TASKS.md
- docs/project/DECISIONS.md
- docs/project/CHECKPOINTS.md
- docs/project/STORY_CONTINUITY_AGENT_GUIDE.md
- issues/README.md
- issues/next-phase-p1-story-continuity-mvc.csv
- examples/mcp/create-story-context-v1.json
- examples/mcp/run-3-panel-story-smoke.md

任务目标：
推进 P1 Story Continuity MVC：3 格、无字、顺序生成、人工选图、真实参考图参与的连续故事最小闭环。

请严格遵守边界：
- 不要做完整漫画编辑器。
- 不要新建复杂 Story Review 页面，第一轮只在现有 Production View / manifest / metadata 上做最小增强。
- 不要做 Web 一键加字、批量 caption、caption renderer 或 ZIP。
- 不要新增 MCP 删除、清库、workspace/project/campaign/asset destructive tools。
- 不要读取、打印、提交任何 provider key、project key、Basic/Auth 密码、Admin cookie、session 或 .env secret。
- 默认先用 mock 验数据链路；真实 provider canary 只有在用户明确确认费用后执行，cap=1，每格最多 2 候选，总图量最多 8 张。

请先只读审视当前代码和文档，然后给出实现计划。若开始实现，优先顺序是：
1. story_context_v1 metadata contract 和 fixture。
2. Panel Plan 因果字段增强：panel_index、narrative_role、trigger_event、visible_action、resulting_state、dialogue_intent。
3. 区分 reference_bindings 和 resolved_reference_assets。
4. sequential preflight：panel_index > 1 必须存在上一格 selected asset。
5. Sequential Previous Panel Mode：第 1 格生成并人工 select 后，才能创建第 2 格；第 2 格 select 后才能创建第 3 格。
6. Production View / Technical details 最小展示连续性信息。
7. manifest 输出 story_context_v1 摘要。
8. mock 3 格 smoke，只证明数据链路，不声明视觉连续性成功。
9. 可选真实 provider 3 格 canary，人工评分角色、场景、道具、因果和参考参与。

验收标准：
- 3 格都有数值 panel_index。
- task/asset 保存同一 story_revision 和 story_plan_hash。
- metadata 能区分 reference_bindings 与 resolved_reference_assets。
- 第二格实际引用第一格 selected asset。
- 第三格实际引用第二格 selected asset。
- preflight 失败不静默退化为纯文生图。
- regenerate 不覆盖旧 task/asset 或其他格 selected 状态。
- manifest 不暴露 local_path、key、cookie、session 或 secret。

每完成一个子任务后，必须同步更新：
- issues/next-phase-p1-story-continuity-mvc.csv
- docs/project/TASKS.md
- docs/project/PROJECT_STATUS_MAP.md
- docs/project/V1_BASELINE_AND_ROADMAP.md
- docs/project/CHECKPOINTS.md
- docs/project/DECISIONS.md（如有新增产品/技术决策）

请不要从原来的 next-phase-p1-story-continuity-comic-workflow.csv 全量开工；它现在只是上位路线，第一执行入口是 next-phase-p1-story-continuity-mvc.csv。
```
