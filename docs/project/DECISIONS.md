# Decisions

## 2026-06-18: 项目定位为 Agent ImageFlow

- Decision: 项目定位从 DiagramOps / 技术图示网关调整为 Agent ImageFlow / AI 图片资产生成与管理平台。
- Reason: 用户明确表示不希望第一版重点做 SVG、Mermaid 或技术图示，因为 Codex 已能完成部分图示生成；更想解决生图自动化、海报设计、小说配图和其他图片资产场景。
- Impact: 第一版优先验证生图、落盘、审核、复用、MCP/API 调用闭环；暂不优先做技术图示 DSL。

## 2026-06-18: 当前阶段只写项目文档，不实现代码

- Decision: 初始化项目上下文和产品规格文档，暂不创建应用代码。
- Reason: 产品能力和定位还要再次规格定义，过早实现容易固化错误边界。
- Impact: 后续需要产品/MVP lock 后再进入实现。

## 2026-06-18: 能力平台而非网页生图工具

- Decision: 产品应作为 MCP/API/CLI 能力平台，而不是只面向人工点击的网页工具。
- Reason: 核心痛点是 AI 和自动化系统无法稳定拿到图片资产句柄、元数据、审核状态和交付路径。
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
- Reason: 这个流程最能验证批量任务、业务隔离、候选图、缩略图、审核、文件获取和 metadata 交付，同时比小说角色一致性和电商商品海报更轻。
- Impact: 首个 vertical slice 围绕内容账号 project、7 天封面图 campaign、批量 ImageTask、候选 Asset 和审核交付展开。

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
