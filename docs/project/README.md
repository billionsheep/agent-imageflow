# Agent ImageFlow Project Docs

本文是 `docs/project/` 的默认入口。V1 之后，新会话、新 agent 或维护者优先从这里进入，不需要默认通读全部历史文档。

## 当前定位

Agent ImageFlow 是服务器托管的图片资产生产平台。外部 agent、脚本、Web、CLI 或 REST 可以创建图片任务；服务端负责 provider 调用、资产落盘、缩略图、metadata、审核状态、交付 URL 和基础治理。

当前版本 `v0.1.0` 已作为 V1 baseline 推送。后续工作进入版本化维护：优先服务器部署演练、MCP-first 真实业务生产试用、Web/Settings 体验收敛和运维安全增强。

下一阶段新增的产品方向是 Story Continuity：连续叙事、分镜和重试策略由额外 agent 承担，Agent ImageFlow 继续负责图片资产生产、参考图、派生关系、审图和交付事实源。真实萌宠账号生产试用入口见 `PET_STORY_PRODUCTION_WORKFLOW.md`，新 agent 接入见 `STORY_CONTINUITY_AGENT_GUIDE.md`。

## 日常入口

日常 PM、开发、验收和新 agent 接入优先阅读：

- `PROJECT_STATUS_MAP.md`：当前完成度、下一步主线和明确不做范围。
- `V1_BASELINE_AND_ROADMAP.md`：V1 能力边界、剩余任务和后续路线。
- `TASKS.md`：当前待办、doing 和已完成事项。
- `RUNBOOK.md`：本地、部署、MCP、provider、清理和验收命令。
- `DECISIONS.md`：关键产品和技术决策。
- `CHECKPOINTS.md`：验收证据流水。
- `PET_STORY_PRODUCTION_WORKFLOW.md`：真实萌宠账号 3-6 格故事资产的 MCP-first 生产流程。

## 其他文档分层

- 产品与技术基线：`PRODUCT_SPEC.md`、`TECH_SPEC.md`、`ARCHITECTURE.md`、`INPUT_OUTPUT_SPEC.md`、`BUSINESS_SCENARIOS.md`。
- 历史设计依据：`ARCHITECTURE_REVIEW.md`、`IMPLEMENTATION_REVIEW_AND_FLOW_SIMULATION.md`。
- 阶段性需求池：`FUTURE_REQUIREMENTS_AND_SCENARIOS.md`、`NEXT_PHASE_REQUIREMENTS.md`。当前执行入口以 `V1_BASELINE_AND_ROADMAP.md` 和 `TASKS.md` 为准。
- 部署交接：`SERVER_DEPLOYMENT_GUIDE.md`。
- Agent 接入：`MCP_SERVICE_GUIDE.md`、`STORY_CONTINUITY_AGENT_GUIDE.md`、`PET_STORY_PRODUCTION_WORKFLOW.md`。
- 实现记录：`stories/` 下的 slice 文档，见 `stories/README.md`。

## 后续维护原则

- 已完成的 P0/P1/P2 CSV 不复开；新需求新建独立 CSV 或 story。
- 不把项目扩成小红书运营后台、通用 DAM、模板市场、SaaS 注册计费或每用户 provider key 系统。
- provider key、project key、Basic/Auth 密码、Admin cookie 和 session 不写入文档证据。
- 若未来要移动或归档历史文档，先做引用检查和单独 repo cleanup 计划。
