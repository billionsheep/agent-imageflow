# AGENTS.md

本项目是 `Agent ImageFlow` 的产品规格与后续实现工作区。

## 项目定位

这是一个 AI 可调用的图片资产生成与管理能力平台，目标是让 Codex、Claude、Cursor、自动化脚本、内容系统或业务后台通过 MCP/API/CLI 创建图片资产，并拿到可追踪、可审核、可复用、可交付的正式结果。

## 工作规则

- 默认使用简体中文沟通和编写项目文档。
- 项目上下文统一维护在 `docs/project/`。
- 当前已进入实施准备阶段；实现前必须遵守 `docs/project/ARCHITECTURE.md` 和 `docs/project/IMPLEMENTATION_REVIEW_AND_FLOW_SIMULATION.md`。
- 不把项目扩展成泛设计平台、白板工具、模板市场或通用 DAM，除非用户重新确认方向。
- 第一版优先验证图片资产生成闭环，不优先做 Mermaid/D2/SVG 技术图示。
- 修改产品方向、MVP 范围、技术栈或关键接口时，同步更新 `docs/project/DECISIONS.md`。
- 每次推进任务后，按需更新 `docs/project/TASKS.md`、`PROJECT_PLAN.md` 和 `CHECKPOINTS.md`。

## 必读文档

开始工作前先读：

1. `docs/project/PRODUCT_SPEC.md`
2. `docs/project/PROJECT_PLAN.md`
3. `docs/project/TECH_SPEC.md`
4. `docs/project/INPUT_OUTPUT_SPEC.md`
5. `docs/project/BUSINESS_SCENARIOS.md`
6. `docs/project/ARCHITECTURE.md`
7. `docs/project/IMPLEMENTATION_REVIEW_AND_FLOW_SIMULATION.md`
8. `docs/project/TASKS.md`
9. `docs/project/DECISIONS.md`
10. `docs/project/CHECKPOINTS.md`
11. `docs/project/RUNBOOK.md`

## 当前默认边界

- 做：结构化图片任务、生图 provider 适配、资产落盘、元数据、审核状态、MCP/API/CLI 调用、复用交付。
- 暂不做：正文事实编写、知识库自动改写、设计画布、多人协作编辑、模板商城、企业级权限计费。
