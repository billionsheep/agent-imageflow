# Tasks

## Todo

- [ ] 实现 MCP stdio server，复用现有服务端 application core。
- [ ] 细化并实现 `create_image_task` / `get_image_task` / `list_image_assets` / `select_image_asset` / `reject_image_asset` / `get_asset_delivery_info` MCP tool schema。
- [ ] 迁移第一个真实 provider 到服务端 Worker，优先 OpenAI-compatible。
- [ ] 将 Web 现有生成交互接入服务端 `ImageTask/Asset` 托管模式。
- [ ] 设计 Web 候选图选优视图的 project / campaign 选择和 select/reject 交互。
- [ ] 增加 prompt 模板、style preset 和参考图参数的服务端保存/复用策略。
- [ ] 设计自动选优/多候选 best-of 的后续 story，不把每张图人工审核作为默认闸门。
- [ ] 增强缩略图处理为真实图片 resize / webp 输出。
- [ ] 增加 repair/reconcile smoke 命令。

## Doing

- [ ] 确认 MCP stdio server 最小 story 范围。

## Done

- [x] 初始化项目目录。
- [x] 创建首版产品规格书。
- [x] 创建项目计划、技术规格、决策、检查点和运行说明。
- [x] 冻结输入输出 v0.1。
- [x] 冻结业务隔离模型 v0.1。
- [x] 选择核心业务流程：内容系统批量生成封面图。
- [x] 落盘第一版架构文档。
- [x] 合并架构评审，形成最终架构指导文件。
- [x] 初始化 Git 仓库并绑定 GitHub remote。
- [x] 完成实施前审视与业务流程模拟。
- [x] 锁定第一阶段实施目标：Go + PostgreSQL + Redis + 本地文件系统 + Docker Compose + mock provider。
- [x] 回退低保真自写 Web/API/Worker 实现。
- [x] 导入 `gpt_image_playground` 作为 `web/` 前端底座。
- [x] 将 Web 品牌、PWA 信息、本地存储命名空间调整为 Agent ImageFlow。
- [x] 创建 Go + Docker Compose 实施骨架。
- [x] 设计 Web 生成结果接入 Agent ImageFlow `ImageTask/Asset` 服务端模型的最小 API client 边界。
- [x] 设计 workspace / project / campaign 的最小创建和选择流程：默认 seed + REST path + CLI flags。
- [x] 用 mock provider 实现“内容账号 campaign 封面图生成闭环”。
- [x] 实现 REST/CLI 创建任务、查询任务、approve/reject 兼容状态标记和 asset delivery info。
- [x] 将产品计划从强人工审核调整为轻量选优/状态标记。

## Acceptance Criteria For Next Step

- 可以通过 `docker compose up` 启动 API、Worker、PostgreSQL 和 Redis。
- 可以通过 `vag task create --file /app/examples/tasks/sample-image-task.json` 创建 mock 图片任务。
- 任务完成后可以得到 `task_id`、`asset_id`、原图、缩略图、metadata 和 delivery URL。
- 下一步优先实现 MCP stdio server，让 Codex/Claude 能通过结构化 tool 调用当前服务端资产闭环，并以 `select_image_asset` 表达推荐候选图。
