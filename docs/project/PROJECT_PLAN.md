# Project Plan

## Current Phase

- Phase: Implementation preparation
- Goal: 基于已冻结的输入/输出、业务隔离、核心业务流程和最终架构指导，进入第一条 vertical slice 的工程实施。
- Status: 输入/输出 v0.1 已冻结；业务隔离模型已冻结；核心业务流程已选定为内容系统批量生成封面图；架构评审已合并；实施前业务流程模拟已完成；第一阶段采用 Go + PostgreSQL + Redis + 本地文件系统 + Docker Compose；先用 mock provider，后接云端 API provider；不考虑本地 GPU。

## Milestones

1. Product/MVP lock
2. Provider and delivery lock
3. First vertical slice plan
4. Implementation kickoff
5. MVP completion

## Implementation Target v0

第一阶段实施只验证一条闭环：

```text
REST/CLI 创建 ImageTask
  -> PostgreSQL 记录任务
  -> Redis 入队
  -> Go Worker 消费任务
  -> mock provider 生成示例图片
  -> 本地文件系统保存原图、缩略图、metadata
  -> PostgreSQL 登记 asset / asset_version
  -> approve asset
  -> 返回 original / thumbnail / metadata / delivery info
```

第一阶段不做：

- 真实 provider 凭据。
- 本地 GPU 或 ComfyUI。
- Web UI。
- MCP server。
- MinIO/S3。
- webhook。
- 用户权限和计费。

## Vertical Slices

### Slice 1: 内容账号 campaign 封面图生成闭环

- Goal: 在一个内容账号 project 的 campaign 下，从结构化任务生成封面图候选资产，并完成落盘、缩略图、登记、审核和结果返回。
- User flow:
  1. 创建或使用默认 workspace。
  2. 创建内容账号 project。
  3. 创建“7 天封面图计划” campaign。
  4. 在 campaign 下提交一个或多个封面图任务。
  5. 系统入队并调用 provider。
  6. 系统保存原图、生成缩略图，并登记 asset metadata。
  7. 用户审核通过。
  8. 调用方取得原图、缩略图、路径/URL/metadata。
- Acceptance criteria:
  - 能返回 `task_id`、`asset_id`、状态、文件路径或 URL。
  - 能按 `asset_id` 获取原图和缩略图。
  - 生成与审核状态分离。
  - 任务和资产按 project / campaign 隔离。
  - 失败任务可看到错误原因。
- Verification:
  - 本地运行一条 demo 任务。
  - 检查数据库记录与文件存储结果一致。
  - 审核状态变更后可再次查询。

## Roadmap

### 3-day validation

- 写清产品规格、MVP 范围和核心业务流程。
- 选择第一版 provider，并确认本地存储根目录、URL 映射和第一版交付目标。
- 设计 API / MCP tool schema 草案。
- 用伪 provider 或本地示例文件模拟完整任务流。
- 使用 `INPUT_OUTPUT_SPEC.md` 校验输入、输出、隔离边界。
- 使用 `BUSINESS_SCENARIOS.md` 校验业务流程是否仍然聚焦。

### 7-day MVP

- 实现任务创建、状态查询、资产登记、审核状态。
- 使用 mock provider 跑通完整闭环。
- 支持本地文件存储。
- 提供 CLI 或 REST API smoke test。
- 在闭环稳定后接入一个云端 API provider。

### 30-day portfolio version

- React 控制台。
- 缩略图预览和审核页面。
- MCP stdio server。
- Provider adapter 抽象。
- Docker Compose。
- README、Demo GIF、示例自动化流程。

### Later

- 多 provider 策略和成本控制。
- MinIO/S3 存储。
- 本地 ComfyUI / GPU provider。
- webhook。
- 公开 API key。
- Notion / GitHub / CMS 交付适配。
- Streamable HTTP MCP。
