# Agent ImageFlow

AI 可调用的图片资产生成与管理能力平台。

本项目用于探索一个面向 Codex、Claude、Cursor、自动化脚本、内容系统和业务后台的图片生成流程与资产交付平台。它不定位为普通网页生图工具，也不优先做技术图示 DSL，而是聚焦一条生产链路：

```text
结构化图片任务 -> 生图后端 -> 本地/对象存储落盘 -> 资产登记 -> 审核 -> 复用/交付
```

## 当前阶段

项目已完成产品规格、架构评审和业务流程模拟，进入第一条 vertical slice 的实施准备阶段，暂不包含应用代码。

当前已锁定的方向：

- 第一版聚焦“AI 可调用的图片资产生成与管理平台”。
- 优先验证生图、落盘、元数据、审核、API/MCP 调用闭环。
- 第一阶段使用 Go + PostgreSQL + Redis + 本地文件系统 + Docker Compose。
- 第一阶段先用 mock provider 跑通闭环，后续接云端 API provider。
- MVP 不考虑本地 GPU 或 ComfyUI。
- 暂不做 Mermaid/D2/SVG 技术图示系统。
- 暂不做白板、设计编辑器、模板市场或泛用 DAM。

## 文档入口

- `docs/project/PRODUCT_SPEC.md`：产品规格书草案。
- `docs/project/BUSINESS_SCENARIOS.md`：业务场景排序和核心业务流程。
- `docs/project/INPUT_OUTPUT_SPEC.md`：输入、输出和业务隔离 v0.1 冻结规格。
- `docs/project/ARCHITECTURE.md`：第一版架构方向、分层、运行拓扑和扩展点。
- `docs/project/IMPLEMENTATION_REVIEW_AND_FLOW_SIMULATION.md`：实施前审视与业务流程模拟。
- `docs/project/PROJECT_PLAN.md`：阶段计划与首个 vertical slice。
- `docs/project/TECH_SPEC.md`：推荐技术假设与架构草案。
- `docs/project/TASKS.md`：待办与验收。
- `docs/project/DECISIONS.md`：已记录的产品/技术决策。
- `docs/project/CHECKPOINTS.md`：验证检查点。
- `docs/project/RUNBOOK.md`：未来运行、测试、调试入口。
