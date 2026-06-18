# Project Plan

## Current Phase

- Phase: Server asset loop completed; MCP / Web managed mode next
- Goal: 在可运行 Web 底座和服务端 mock 资产闭环之上，继续补 MCP tool schema 与 Web 托管模式。
- Status: 输入/输出 v0.1 已冻结；业务隔离模型已冻结；核心业务流程已选定为内容系统批量生成封面图；架构评审已合并；Web 底座已导入；Go + PostgreSQL + Redis + 本地文件系统 + Docker Compose 的 mock 资产闭环已跑通。
- Product update: 第一版已弱化人工审核，默认采用轻量选优/状态标记。质量优先通过 prompt 模板、style preset、参考图、生成参数和后续 best-of 自动选优保证。

## Regenerated Phase Plan

当前进展已经到“服务端 mock 资产闭环完成”。后续按低风险顺序推进：

1. MCP entry: 先实现 MCP stdio server，让 Codex/Claude 能创建任务、查询资产、标记 selected/rejected、获取交付信息。
2. Quality foundation: 服务端保存 prompt template、style preset、reference image 参数和生成配置，为稳定质量和复用打底。
3. Real provider: 从 Web 参考项目迁移一个真实云端 provider adapter，优先 OpenAI-compatible；继续复用现有 Worker、asset processor 和 storage。
4. Web managed mode: Web 从浏览器本地事实源切到服务端 `ImageTask/Asset`，展示候选图并做轻量 select/reject。
5. Best-of selection: 在多候选结果上增加自动打分、排序、推荐和批量 reject/selected；强人工审核只作为未来项目级可选策略。
6. Hardening: 补 retry/backoff、repair/reconcile、真实缩略图 resize、项目级 API key 和自托管部署说明。

## Direction Lock

当前项目处在从“浏览器生图工作台”升级为“服务端资产平台”的过渡阶段。

最终方向不是长期维护两套生图系统，而是收敛为：

```text
Web UI / MCP / REST / CLI
        |
        v
同一个服务端 Application Core
        |
        v
ImageTask / Provider Adapter / Asset Registry / Selection / Delivery
```

因此：

- `web/` 当前保留原 `GPT Image Playground` 的成熟交互和 provider 经验。
- Go 服务端是未来正式事实源，负责任务、队列、provider 调用、文件落盘、资产登记、轻量选优和交付。
- Web 后续应进入“服务端托管模式”：创建服务端 `ImageTask`，展示服务端候选 `Asset`，执行 select/reject。当前 `approve/reject` 作为兼容命名保留。
- MCP、CLI、REST 和 Web 不应各自实现不同业务状态机。
- 原 Web 的 OpenAI-compatible、fal.ai、自定义 HTTP provider 能力应逐步迁移为服务端 `ProviderAdapter`，而不是仅停留在浏览器直连。

## Milestones

1. Product/MVP lock
2. Provider and delivery lock
3. First vertical slice plan
4. Implementation kickoff
5. Service-side mock asset loop
6. Agent-callable MCP entry
7. First real provider adapter
8. Web managed mode
9. MVP completion

## Implementation Target v0

第一阶段实施拆成两个连续步骤。

步骤 A：前端底座二开：

```text
导入 gpt_image_playground
  -> 改为 Agent ImageFlow 品牌和本地存储命名空间
  -> 保留 Base URL / API Key / provider / 画廊 / Agent / 遮罩能力
  -> 本地 Vite 验证
```

步骤 B：服务端资产闭环：

```text
Web/REST/CLI 创建 ImageTask
  -> PostgreSQL 记录任务
  -> Redis 入队
  -> Go Worker 消费任务
  -> mock provider 生成示例图片
  -> 本地文件系统保存原图、缩略图、metadata
  -> PostgreSQL 登记 asset / asset_version
  -> select asset or use generated asset directly
  -> 返回 original / thumbnail / metadata / delivery info
```

当前尚未接入：

- MCP server。
- Web 生成结果同步到 PostgreSQL asset registry。
- 真实云端 API provider。
- Docker Compose 生产部署硬化。

第一阶段仍不做：

- 本地 GPU 或 ComfyUI。
- MinIO/S3。
- webhook。
- 用户权限和计费。

## Vertical Slices

### Slice 1: 内容账号 campaign 封面图生成闭环

- Goal: 在一个内容账号 project 的 campaign 下，从结构化任务生成封面图候选资产，并完成落盘、缩略图、登记、轻量选优和结果返回。
- User flow:
  1. 创建或使用默认 workspace。
  2. 创建内容账号 project。
  3. 创建“7 天封面图计划” campaign。
  4. 在 Web、REST 或 CLI 下提交一个或多个封面图任务。
  5. 系统入队并调用 provider。
  6. 系统保存原图、生成缩略图，并登记 asset metadata。
  7. 用户、调用方或自动策略标记推荐图，也可以直接使用 generated 结果。
  8. 调用方取得原图、缩略图、路径/URL/metadata。
- Acceptance criteria:
  - Web 工作台保留参考项目成熟交互，并支持 Base URL / API Key 配置。
  - 能返回 `task_id`、`asset_id`、状态、文件路径或 URL。
  - 能按 `asset_id` 获取原图和缩略图。
  - 生成执行状态与候选图选优/使用状态分离。
  - 任务和资产按 project / campaign 隔离。
  - 失败任务可看到错误原因。
- Verification:
  - 本地运行一条 demo 任务。
  - 检查数据库记录与文件存储结果一致。
  - 选优/状态标记后可再次查询。

## Roadmap

### Phase 0: Product and architecture lock

- 写清产品规格、MVP 范围和核心业务流程。
- 选择第一版 provider，并确认本地存储根目录、URL 映射和第一版交付目标。
- 设计 API / MCP tool schema 草案。
- 用伪 provider 或本地示例文件模拟完整任务流。
- 使用 `INPUT_OUTPUT_SPEC.md` 校验输入、输出、隔离边界。
- 使用 `BUSINESS_SCENARIOS.md` 校验业务流程是否仍然聚焦。

Status: done.

### Phase 1: Web foundation

- 基于参考项目完成 Web 工作台底座二开。

Status: done. 当前 Web 可运行，保留原项目成熟交互和 provider 配置。

### Phase 2: Service-side mock asset loop

- 已实现任务创建、状态查询、资产登记、兼容状态标记。
- 已使用 mock provider 跑通完整闭环。
- 已支持本地文件存储。
- 已提供 CLI / REST API smoke test。

Status: done. 当前服务端能通过 Docker Compose 启动 API、Worker、PostgreSQL、Redis，并跑通 mock 任务。

### Phase 3: Agent-callable MCP

- 实现 MCP stdio server。
- 暴露 `create_image_task`、`get_image_task`、`list_image_assets`、`select_image_asset`、`reject_image_asset`、`get_asset_delivery_info`。
- MCP tools 复用现有 application core / service，不绕过服务端状态机。
- 用本地 Codex/Claude 类调用方式验证结构化 JSON 输出。

Recommended next slice.

### Phase 4: First real provider adapter

- 从原 Web 的 OpenAI-compatible / fal.ai provider 经验中迁移一个服务端 provider adapter。
- 第一优先级建议 OpenAI-compatible，因为 Base URL / API Key / model 形态最通用。
- Worker 负责调用真实 provider、下载 URL 或解析 base64，然后继续走现有 asset processor。
- 密钥只走环境变量或本地配置，不进入仓库。

### Phase 5: Web managed mode

- 给 Web 增加服务端托管模式。
- Web 创建服务端 `ImageTask`，轮询任务状态，展示服务端 thumbnail/original。
- Web 侧 select/reject 调用服务端 API，`selected` 作为推荐候选而非强制人工审核闸门。
- 原浏览器直连 provider 能力可保留为 legacy / playground mode，但正式资产流默认走服务端。

### Phase 6: MVP hardening

- 补 repair/reconcile 命令。
- 增强 worker retry / backoff / duplicate handling。
- 缩略图改为真实 resize / webp。
- 增加项目级 API key、基本鉴权和配置样例。
- 完善 README、demo 流程和自托管运行说明。

### 30-day portfolio version

- React 控制台。
- 缩略图预览和候选图选优页面。
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
