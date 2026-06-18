# Decisions

## 2026-06-18: 项目定位为 Agent ImageFlow

- Decision: 项目定位从 DiagramOps / 技术图示网关调整为 Agent ImageFlow / AI 图片资产生成与管理平台。
- Reason: 用户明确表示不希望第一版重点做 SVG、Mermaid 或技术图示，因为 Codex 已能完成部分图示生成；更想解决生图自动化、海报设计、小说配图和其他图片资产场景。
- Impact: 第一版优先验证生图、落盘、轻量选优、复用、MCP/API 调用闭环；暂不优先做技术图示 DSL。

## 2026-06-18: 降级人工审核为轻量选优/状态标记

- Decision: 第一版不把“每张图片必须人工审核通过”作为默认流程，改为 `generated -> selected/rejected/published` 的轻量选优和状态标记。当前代码中的 `draft/approved`、`approve/reject` 保留为兼容命名，产品语义上分别映射为 `generated/selected`、`select/reject`。
- Reason: 项目当前面向单体平台或小团队，强人工审核会增加使用成本；质量优先通过 prompt 优化、style preset、参考图、模板复用和多候选 best-of 逻辑保证。
- Impact: 后续 MCP、Web managed mode 和计划文档优先使用 `select_image_asset`、候选图选优视图和自动选优策略；强审核只作为未来项目级可选策略，不进入 MVP 默认路径。

## 2026-06-18: 当前阶段只写项目文档，不实现代码

- Decision: 初始化项目上下文和产品规格文档，暂不创建应用代码。
- Reason: 产品能力和定位还要再次规格定义，过早实现容易固化错误边界。
- Impact: 后续需要产品/MVP lock 后再进入实现。

## 2026-06-18: 能力平台而非网页生图工具

- Decision: 产品应作为 MCP/API/CLI 能力平台，而不是只面向人工点击的网页工具。
- Reason: 核心痛点是 AI 和自动化系统无法稳定拿到图片资产句柄、元数据、候选图状态和交付路径。
- Impact: 未来接口设计优先考虑结构化输入输出、任务状态和资产登记。

## 2026-06-18: 冻结输入与输出 v0.1

- Decision: 第一版输入固定为 MCP、REST API、CLI、Web UI 四类；输出固定为任务结果、资产身份、原图文件、缩略图、metadata JSON、交付信息六类。
- Reason: 产品需要先稳定契约，避免继续扩散到过多调用方式和输出形态。
- Impact: 后续接口、MCP tools、CLI 和 UI 都围绕 `ImageTask`、`Asset`、原图、缩略图、metadata 展开。

## 2026-06-18: 冻结业务隔离模型 v0.1

- Decision: 第一版采用 `Workspace -> Project -> Campaign -> ImageTask -> Asset` 的业务隔离模型。
- Reason: 小红书账号、小说配图、海报活动、技术博客等业务不能混在同一个资产池里；同时项目不应过早扩展成完整 DAM。
- Impact: 任务和资产必须带 `workspace_id`、`project_id`、`campaign_id`；未来业务能力通过 metadata、adapter 和模块扩展，而不是第一版写死所有业务字段。

## 2026-06-18: 选择内容系统批量封面图作为核心业务流程

- Decision: 第一版核心业务流程选定为“内容系统批量生成封面图”，demo 方向使用小红书/内容账号 campaign 素材生产。
- Reason: 这个流程最能验证批量任务、业务隔离、候选图、缩略图、选优、文件获取和 metadata 交付，同时比小说角色一致性和电商商品海报更轻。
- Impact: 首个 vertical slice 围绕内容账号 project、7 天封面图 campaign、批量 ImageTask、候选 Asset 和选优交付展开。

## 2026-06-18: 冻结第一版架构方向

- Decision: 第一版采用“小核心 + 多入口 + 多适配器 + 大存储”的模块化单体架构。入口层包含 MCP、REST API、CLI、Web UI；核心层统一处理 Workspace / Project / Campaign / ImageTask / Asset；Worker 异步调用 provider；资产处理层负责原图、缩略图、hash 和 metadata；存储层优先本地大磁盘，后续扩展 MinIO/S3。
- Reason: 这个架构能支撑 AI/自动化调用和未来业务扩展，同时避免过早拆微服务或做成纯网页生图工具。
- Impact: 实现阶段应优先打通 API/Worker/Postgres/Redis/File Storage 的核心管线；Provider、Storage、Delivery 都以 adapter 方式设计。

## 2026-06-18: 合并架构评审为最终架构指导

- Decision: 将 `ARCHITECTURE_REVIEW.md` 中的状态模型、幂等重试、一致性边界、文件访问隔离、provider 失败模型、可观测性和演进触发条件合并进 `ARCHITECTURE.md`，并把 `ARCHITECTURE.md` 作为后续实现主准绳。
- Reason: 原架构方向正确，但如果缺少任务/资产/版本状态拆分、重复消费处理、文件与数据库一致性和 provider 失败结构化，第一版很容易退化成不可靠的生图 API wrapper。
- Impact: 实现阶段必须优先验证 mock provider 全链路，并在首个 vertical slice 中纳入 `idempotency_key`、`task_attempt`、`AssetVersion.status`、归属校验和结构化错误；`ARCHITECTURE_REVIEW.md` 保留为评审输入，不再作为并列实现规范。

## 2026-06-18: 锁定第一阶段实施栈和部署方式

- Decision: 第一阶段采用 Go + PostgreSQL + Redis + 本地文件系统 + Docker Compose；先实现 mock provider 闭环，后续再接一个云端 API provider；MVP 不考虑本地 GPU 或 ComfyUI。
- Reason: 用户明确希望只用 API、不考虑 GPU；Go 适合 API、Worker、CLI、MCP 共享 domain code；Docker Compose 能覆盖本地开发和第一版自托管部署。
- Impact: 下一步直接进入实施骨架：Docker Compose、Go API、Go Worker、CLI smoke test、PostgreSQL schema、Redis queue、local storage 和 mock provider。

## 2026-06-18: 回退低保真 Web，改为基于 GPT Image Playground 二开

- Decision: 撤回自写低保真 Web/API/Worker 实现，当前前端改为直接导入 `/Users/moon/Workspace/tools/gpt_image_playground` 到 `web/` 并二开。
- Reason: 用户明确反馈自写 Web 质量不足，且参考项目已经具备成熟的生图工作台、设置页、Base URL/API Key、多 provider、画廊、参考图、遮罩和 Agent 模式。
- Impact: 第一实现步骤改为先稳定 Web 底座，再把 Agent ImageFlow 的服务端资产登记、轻量选优、MCP 和交付模型接入进去；原架构方向保留，但实施顺序调整为 Web-first。

## 2026-06-18: 完成第一条服务端 mock 资产闭环

- Decision: 在当前 Web 底座之外新增 Go API、Worker、CLI、PostgreSQL、Redis、本地文件系统和 Docker Compose 骨架，并用 mock provider 跑通 `ImageTask -> Asset -> AssetVersion -> 状态事件 -> DeliveryInfo`。
- Reason: 产品核心价值必须来自稳定 `task_id`、`asset_id`、落盘文件、metadata、候选图状态和交付 URL，而不是浏览器端临时图片结果。
- Impact: 第一版已有 REST/CLI smoke 能力；Web 后续应通过新增的服务端 API client 进入托管模式。MCP、真实云端 provider、MinIO/S3、权限计费仍保持 out of scope。
- Implementation note: Go 依赖 `pgx` 和 `go-redis` 作为 PostgreSQL/Redis 驱动；API 默认监听 `http://localhost:8081`；Docker volume 挂载到 `/data/agent-imageflow`。

## 2026-06-18: Web 和服务端最终收敛到同一资产核心

- Decision: `web/`、MCP、REST API 和 CLI 最终都应作为入口调用同一个服务端 application core；不长期维护“浏览器直连 provider”和“服务端 Worker provider”两套正式生图系统。
- Reason: Agent ImageFlow 的产品定义是可追踪、可选优、可交付的图片资产平台。浏览器直连 provider 可以提供成熟交互和迁移经验，但不能作为 MCP/自动化系统的正式事实源。
- Impact: 后续优先补 MCP stdio server 和服务端真实 provider adapter；Web 再进入服务端托管模式。原 Web 的 OpenAI-compatible、fal.ai、自定义 HTTP provider 逻辑作为服务端 provider adapter 的参考来源。
