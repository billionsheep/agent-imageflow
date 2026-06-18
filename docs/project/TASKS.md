# Tasks

## Todo

- [ ] 创建 Go + Docker Compose 实施骨架。
- [ ] 细化 `create_image_task` 的 MCP tool schema。
- [ ] 设计 workspace / project / campaign 的最小创建和选择流程。
- [ ] 用 mock provider 实现“内容账号 campaign 封面图生成闭环”。

## Doing

- [ ] 第一条 vertical slice 实施准备。

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

## Acceptance Criteria For Next Step

- 可以通过 `docker compose up` 启动 PostgreSQL、Redis、API 和 Worker。
- 可以通过 CLI 或 REST 创建一条 mock 图片任务。
- 任务完成后能得到 `task_id`、`asset_id`、原图、缩略图和 metadata。
