# Checkpoints

## Product Definition

- [x] 已明确不是普通网页生图工具。
- [x] 已明确当前方向不是技术图示 DSL 优先。
- [x] 已明确能力平台定位：MCP/API/CLI + provider adapter + asset registry。
- [x] 已冻结输入入口：MCP、REST API、CLI、Web UI。
- [x] 已冻结输出：任务、资产身份、原图、缩略图、metadata、交付信息。
- [x] 已冻结业务隔离：Workspace、Project、Campaign。
- [x] 已选择核心业务流程：内容系统批量生成封面图。
- [x] 已冻结第一版架构方向：小核心、多入口、适配器和大存储。
- [x] 已合并架构评审，形成最终架构指导文件。
- [x] 已完成实施前业务流程模拟。
- [x] 已确认第一阶段 provider：mock provider first，后续云端 API provider。
- [x] 已确认第一阶段交付目标：本地 API 下载 URL、缩略图 URL 和 metadata URL。
- [x] 已确认最终自托管部署方式：Docker Compose。

## MVP Validation

- [x] 已导入可运行 Web 前端底座。
- [x] Web 支持 Base URL、API Key 和多 provider 配置。
- [x] 可创建结构化图片任务。
- [x] 可在内容账号 project 和 campaign 下创建图片任务。
- [x] 可生成或模拟生成图片。
- [x] 图片可保存到稳定路径。
- [x] 资产有 `asset_id` 和 metadata。
- [x] 资产有缩略图。
- [x] 资产可按 project / campaign 隔离查询。
- [x] 选优/状态标记可变更。
- [x] API/CLI/MCP 至少一个入口可跑通。

## Remaining Product Gaps

- [ ] MCP stdio server 尚未实现。
- [ ] 真实 provider 尚未接入服务端 Worker；真实 provider 能力目前主要保留在 `web/` 浏览器侧。
- [ ] Web 尚未进入服务端托管模式，现有主流程仍以浏览器 IndexedDB / data URL 为本地事实源。
- [ ] Workspace / Project / Campaign 目前是默认 seed + REST path + CLI flags，尚无完整 Web 管理体验。
- [ ] Worker retry、repair/reconcile、真实缩略图 resize、项目级 API key 和生产部署硬化仍待补。

## Evidence Log

- 2026-06-18: 根据用户讨论，项目从知识库图示生产系统收敛/转向更通用的生图自动化资产平台。
- 2026-06-18: 输入/输出和业务隔离 v0.1 已冻结，保留未来业务扩展点但不扩大 MVP。
- 2026-06-18: 已将架构评审合并进 `ARCHITECTURE.md`，补齐状态模型、幂等重试、一致性边界、文件访问隔离、provider 失败模型和演进触发条件。
- 2026-06-18: 已初始化 Git 仓库，绑定并推送到 `git@github.com:billionsheep/agent-imageflow.git`。
- 2026-06-18: 已新增 `IMPLEMENTATION_REVIEW_AND_FLOW_SIMULATION.md`，模拟内容账号 campaign 封面图生成、候选选择、交付和失败路径。
- 2026-06-18: 已确认进入实施准备阶段；第一阶段使用 Go、PostgreSQL、Redis、本地文件系统、Docker Compose 和 mock provider，不考虑本地 GPU。
- 2026-06-18: 已回退低保真自写实现，改为基于 `GPT Image Playground` 导入 `web/` 并二开；Web 测试和构建通过。
- 2026-06-18: 已完成服务端 mock 资产闭环：`docker compose up` 启动 API/Worker/PostgreSQL/Redis；CLI 创建任务后 Worker 生成 3 个 ready asset_version；approve 兼容命令可标记推荐资产并返回 original/thumbnail/metadata URL。
- 2026-06-18: 已明确 Web 与服务端不是长期并行的两套正式系统；最终 Web/MCP/CLI/REST 都收敛到服务端资产核心，原 Web provider 能力作为迁移来源。
- 2026-06-18: 已根据小团队/单体平台定位弱化人工审核，第一版默认采用轻量选优/状态标记，质量主要通过 prompt、style preset、参考图和后续 best-of 策略保证。
