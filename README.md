# Agent ImageFlow

AI 可调用的图片资产生成与管理能力平台。

本项目用于探索一个面向 Codex、Claude、Cursor、自动化脚本、内容系统和业务后台的图片生成流程与资产交付平台。它不定位为普通网页生图工具，也不优先做技术图示 DSL，而是聚焦一条生产链路：

```text
结构化图片任务 -> 生图后端 -> 本地/对象存储落盘 -> 资产登记 -> 选优/状态标记 -> 复用/交付
```

## 当前阶段

项目已完成产品规格、架构评审和业务流程模拟，并已进入可运行 MVP 骨架阶段：当前已有 Web 前端底座和第一条服务端 mock 资产闭环。

当前已锁定的方向：

- 第一版聚焦“AI 可调用的图片资产生成与管理平台”。
- 优先验证生图、落盘、元数据、轻量选优、API/MCP 调用闭环。
- 当前 `web/` 已基于 `GPT Image Playground` 导入并二开，保留其画廊、设置页、Base URL/API Key、多 provider、参考图、遮罩和 Agent 模式能力。
- 第一阶段使用 Go + PostgreSQL + Redis + 本地文件系统 + Docker Compose。
- 当前服务端已支持 REST/CLI 创建 mock 图片任务、Worker 异步生成、文件落盘、资产登记、approve/reject 兼容状态接口和交付 URL。
- 当前默认不把每张图片人工审核作为交付闸门，质量优先通过 prompt、style preset、参考图、模板复用和后续 best-of 选优策略保证。
- 后续方向是让 Web、MCP、REST API 和 CLI 都收敛到同一个服务端资产核心；Web 现有 provider 能力作为服务端 provider adapter 的迁移来源。
- 下一阶段优先补 MCP stdio server，然后接服务端真实 provider，再做 Web 服务端托管模式。
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

## Web 本地运行

```bash
npm --prefix web install
npm --prefix web run dev -- --host 0.0.0.0 --port 8080
```

打开：

```text
http://localhost:8080
```

## 服务端本地运行

```bash
docker compose up
```

API 默认地址：

```text
http://localhost:8081
```

最小 smoke：

```bash
docker compose exec api /app/vag task create --file /app/examples/tasks/sample-image-task.json
docker compose exec api /app/vag task get <task_id>
docker compose exec api /app/vag asset approve <asset_id> # 兼容命令，语义等价于 select
```
