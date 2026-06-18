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

- [ ] 可创建结构化图片任务。
- [ ] 可在内容账号 project 和 campaign 下创建图片任务。
- [ ] 可生成或模拟生成图片。
- [ ] 图片可保存到稳定路径。
- [ ] 资产有 `asset_id` 和 metadata。
- [ ] 资产有缩略图。
- [ ] 资产可按 project / campaign 隔离查询。
- [ ] 审核状态可变更。
- [ ] API/CLI/MCP 至少一个入口可跑通。

## Evidence Log

- 2026-06-18: 根据用户讨论，项目从知识库图示生产系统收敛/转向更通用的生图自动化资产平台。
- 2026-06-18: 输入/输出和业务隔离 v0.1 已冻结，保留未来业务扩展点但不扩大 MVP。
- 2026-06-18: 已将架构评审合并进 `ARCHITECTURE.md`，补齐状态模型、幂等重试、一致性边界、文件访问隔离、provider 失败模型和演进触发条件。
- 2026-06-18: 已初始化 Git 仓库，绑定并推送到 `git@github.com:billionsheep/agent-imageflow.git`。
- 2026-06-18: 已新增 `IMPLEMENTATION_REVIEW_AND_FLOW_SIMULATION.md`，模拟内容账号 campaign 封面图生成、审核、交付和失败路径。
- 2026-06-18: 已确认进入实施准备阶段；第一阶段使用 Go、PostgreSQL、Redis、本地文件系统、Docker Compose 和 mock provider，不考虑本地 GPU。
