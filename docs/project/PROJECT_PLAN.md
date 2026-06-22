# Project Plan

## Current Phase

- Phase: MVP 核心闭环、scope 管理、真实输入复用、best-of auto reject、HTTP 基础限流、本地审计日志、项目级多 key 策略、第二阶段 P0 visibility、P1 Storage Governance、P1 Asset Production Readiness、P1 Web Performance / Startup、并发性能专项、P1 Provider Throughput & Reliability、P1 Web Console Auth & Asset Visibility、P1 Project Production Context P1-PCTX-001 到 P1-PCTX-007，以及 P1 Web UX Smoothness P1-UX-001 到 P1-UX-009 均已完成；当前主入口为 Project Production Context Web 面板与萌宠故事回归。
- Goal: 服务端资产闭环、Web 托管、高级输入、真实 edit/mask、repair/reconcile、自动 retry/backoff、真实缩略图、项目级鉴权、自托管文档、独立 scope 管理、OpenAI-compatible / fal 输入复用、限流、审计、多 key 生产 hardening，以及 Web 服务端资产同步、最小资产库、Scope Dashboard、source/session metadata 标准都已可验证；当前已补 storage usage scanner、只读 storage-governance API、Web 存储占用展示、cleanup dry-run preview、受控 cleanup execute、只读 storage-integrity 治理视图、资产筛选分页、source/session/batch 查询、项目级非敏感 provider profile、Web 首屏 render budget、启动性能护栏、Codex 批量生产示例、可控 Worker/provider 并发、独立 provider cap、timeout profile、task attempts 阶段指标、diagnostic benchmark、batch progress、Web Admin Recent Assets、Web UX Smoothness 全部 P1 收口，以及 `project.metadata_json.visual_context`。当前 `issues/next-phase-p1-project-production-context.csv` 已完成 P1-PCTX-001 到 P1-PCTX-007，服务端支持 Character/Mascot Profile、Project Reference Library、Prompt Recipe / Quality Profile 2.0、Task Input Integration 和 MCP/REST/CLI examples；P1-PCTX-008 到 P1-PCTX-009 的最小 Web 面板和萌宠故事多 scene 回归后续继续。Batch/Export/NAS/WebDAV/Usage Tracking 后置。
- Status: 输入/输出 v0.1 已冻结；业务隔离模型已冻结；核心业务流程已选定为内容系统批量生成封面图；架构评审已合并；Web 底座已导入；Go + PostgreSQL + Redis + 本地文件系统 + Docker Compose 的 mock 资产闭环已跑通；MCP stdio server 已接入并通过 smoke；服务端 OpenAI-compatible provider adapter 已接入并通过本地 HTTP mock 集成 smoke；服务端 `provider=fal` 已接入 queue + rest storage adapter，并通过本地 Docker smoke 验证 `GET /remote.png`、两次 `POST /rest/storage/upload/initiate`、`POST /queue/openai/gpt-image-2/edit` 和 `task_0dbae47c6d0459cd8c2c -> asset_96d78f9da6b1fcdb0cca`；Web 已新增服务端托管模式，可创建服务端 `ImageTask`、轮询 assets、展示服务端候选图并执行 select/reject；服务端已支持项目级 quality profile 保存/读取，并可在 REST/MCP/Web 托管任务创建时复用 prompt template、style preset、reference image 参数、generation config 和 `best_of_config`；`selection_mode=auto` / `best_of` 已可在 Worker 完成候选资产登记后自动 selected 一张推荐图，当前 scorer registry 支持 `local_metadata_v1` 与 `http_judge_v1`，也支持 `best_of_config.auto_reject_non_selected=true` 将未入选候选自动标记为 rejected；本地 smoke 已验证 `task_79ee5fdfe639cd532805` 产生 1 张 selected + 2 张 rejected，并可将 auto rejected 的 `asset_5d207d1a89b3ba6d6793` 手动重新 select 为 approved/selected；Web/MCP/REST 已能提交 reference image、mask/edit descriptor 和更多 generation config，asset `parameters_json` 会保留这些高级输入快照；本地 `vag repair scan/requeue/verify-asset` 已能发现入队失败任务、重入队修复并校验资产文件；Worker 遇到 provider 瞬时失败时会写入 `task_attempt.retry_after`、进入 Redis delayed queue，并按指数退避自动重试；服务端缩略图现在基于原图统一生成 `.webp`，按配置宽高约束落盘并通过 `GET /api/assets/{id}/thumbnail` 交付；实例级 Basic Auth、项目级 API key、CLI/Web 鉴权透传、access-config 管理接口，以及 Web 设置页的 scope 同步/快速新建能力均已完成并通过 Docker smoke；当前 scope 内 `input-files` 上传/取回和 OpenAI-compatible `/images/edits` multipart 已打通，Docker smoke 已验证 `task_dd1a410a094e30f06fc5 -> asset_fb9f0bbe559c4c95aa88`；服务端创建任务现在还能解析匿名 remote URL 和当前项目 `asset_id`，Docker smoke 已验证 `GET /remote.png`、`POST /v1/images/edits image_count=2` 和 `task_91237d5d15aa7252bed4 -> asset_9ab0aeca719c6e9a2f66`；独立 Web scope 管理入口、workspace/project/campaign rename/archive/delete，以及 archived scope 过滤与 delete 清理链路均已完成并通过 smoke；README 与 Runbook 已补 quickstart、demo、自托管最小暴露面和反向代理/TLS 样例，`docker compose config`、`curl -sf http://localhost:8081/healthz` 和 `docker compose ps` 已验证通过；HTTP API 基础限流现已接入，`RATE_LIMIT_INSTANCE_MAX_REQUESTS` / `RATE_LIMIT_PROJECT_MAX_REQUESTS` 可返回 `429` 与 `Retry-After`，Docker smoke 已验证实例级与 project 级阈值都能生效；HTTP / API 第一版结构化审计日志也已接入，本地 Docker smoke 已验证 `create_task` / `get_task` 和 `404 not_found` 请求都会写入 `STORAGE_ROOT/audit/http-api/YYYY-MM-DD.jsonl`，并可通过 `vag audit list` 按 project / task / status 过滤查询；项目级多 key 策略也已接入，Docker smoke 已验证 `prj_multi_key_1781784728` 可同时接受 `default` 与 `rollout` 两把 key，旧 key disable/delete 后新 key 仍可访问同一 `task_fc9e1275b4dcb665e766`，且审计里会记录命中的 `rollout` key 名称。
- Product update: 第一版已弱化人工审核，默认采用轻量选优/状态标记。质量优先通过 prompt 模板、style preset、参考图、生成参数和后续 best-of 自动选优保证。

## Regenerated Phase Plan

当前进展已经到“Web/MCP/REST/CLI 多入口共用服务端资产核心”。后续按低风险顺序推进：

1. MCP entry: 已实现 MCP stdio server，让 Codex/Claude 能创建任务、查询资产、标记 selected/rejected、获取交付信息。
2. Real provider: 已迁移 OpenAI-compatible provider adapter；继续复用现有 Worker、asset processor 和 storage。
3. Web managed mode: 已新增服务端托管模式；Web 可创建服务端 `ImageTask`，展示服务端候选 `Asset`，执行 select/reject。
4. Quality foundation: 已实现项目级 quality profile 保存/读取；创建任务可显式复用项目模板/风格/参考图参数/生成配置，并把有效快照写入 `structured_input_json`。
5. Best-of selection: 已实现第一版本地启发式自动推荐；`selection_mode=auto` / `best_of` 会自动 selected 一张候选，其他候选保持 generated。
6. Advanced managed input: 已将 reference image、mask/edit descriptor 和更多 generation config 迁移到服务端托管任务，并补齐 scope 内 `input-files` 上传/取回；OpenAI-compatible 已可在存在已解析输入文件时走 `/images/edits`。
7. Hardening: Done. Repair/reconcile、retry/backoff、真实缩略图、项目级鉴权、README/demo、自托管最小暴露面说明和真实 edit/mask 边界都已完成。
8. Scope management UX: Done. Web 已补独立 scope 管理 modal，可执行 rename/archive/delete 并设为当前 scope。
9. Input reuse foundation: Done. OpenAI-compatible 已支持 scope `input-files`、匿名 remote URL 和当前项目 `asset_id` 复用。
10. Provider input reuse expansion: Done. `provider=fal` 已支持 queue 文生图，以及基于 `resolved_input_files` 的 storage upload + edit 闭环。
11. Best-of scoring upgrade: Done. `best_of_config` 已进入任务/quality profile 输入，服务端 scorer registry 当前支持 `local_metadata_v1` 与 `http_judge_v1`，外部 judge 失败时回退本地启发式。
12. Best-of auto reject: Done. `best_of_config.auto_reject_non_selected=true` 时，未入选候选会自动 rejected，且人工仍可重新 select。
13. Production hardening: Done. 基础限流、本地审计日志和项目级多 key 策略均已完成。
14. Local dev hygiene: Done. 已补本地 Web `.vite/` 生成目录的 ignore 规则，避免运行态缓存长期出现在 `git status`。
15. Scenario-driven next stage planning: P0 done, P1 Storage Governance done, P1 Asset Production Readiness done, P1 Web Performance / Startup done, Concurrency/Benchmark done, P1 Provider Throughput & Reliability done, P1 Web Console Auth & Asset Visibility done. 已根据真实试用反馈形成第二阶段需求，并完成 P0 visibility CSV、P1 storage governance CSV、合并后的 asset production readiness CSV、Web performance CSV、并发性能专项、provider reliability hardening 与 Web console visibility；下一步不再生成泛 P2 大包，而是并行执行已落盘的 `issues/next-phase-p1-project-production-context.csv` 和 `issues/next-phase-p1-web-ux-smoothness.csv`。
16. Project Production Context: Partially done. P1-PCTX-001 到 P1-PCTX-007 已完成：第一版不新增数据库表，使用 `project.metadata_json.visual_context` 保存 Character/Mascot Profile、Project Reference Library 和 Prompt Recipe，并在 CreateTask 阶段展开为稳定 `structured_input_json` / `parameters_json` 快照；REST、CLI 和 MCP create task 已接入；examples 已覆盖 CLI 多 scene、REST create task、MCP `tools/call` 和 usage 文档。Web 最小 project context 面板和萌宠故事多 scene 回归仍按 P1-PCTX-008 到 P1-PCTX-009 后续推进，批量故事视图、Export Pack、NAS/WebDAV/SMB 和 Usage Tracking 后置。
17. Web UX Smoothness: Done. P1-UX-001 到 P1-UX-009 已完成：`ServerAssetLibrary` 只订阅 Agent ImageFlow 相关 settings 字段，刷新/error/scope incomplete 不再误清空已有资产，文本筛选加 300ms debounce 并继续忽略旧请求，资产卡 `Scope` 动作一次性写入必要 scope 字段；Settings 托管 scope selector、手动兜底 ID 输入和 ScopeManager 设当前 scope 已避免把不完整 workspace/project/campaign 写进全局 settings；Scope 管理统计已从层级加载中拆出，后台延迟启动并忽略旧请求写回；Settings/Scope/Detail/Lightbox/Mask/Agent 懒加载 fallback 已有稳定 overlay/skeleton，并在常用入口 hover/focus/pointerdown 预加载 chunk；Task/Asset 卡片已补 `React.memo`、稳定 per-card handler 和收窄 store 订阅；最终 production preview / Browser 回归证据已写回 CSV、CHECKPOINTS 和 RUNBOOK。

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
- Web 已进入第一版“服务端托管模式”：可创建服务端 `ImageTask`，展示服务端候选 `Asset`，执行 select/reject。当前 `approve/reject` 作为兼容命名保留。
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
8.5. Quality foundation
8.6. Best-of auto selection
8.7. Advanced managed input
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

当前已接入：

- Web 托管模式下生成结果进入 PostgreSQL asset registry。
- Web 详情页可展示服务端候选资产并执行 select/reject。
- Web 主界面已新增服务端资产库，可同步当前 scope 下来自 Web / MCP / REST / CLI 的服务端资产，并展示 thumbnail、prompt、provider、model、status、source、task_id、asset_id、created_at 和 delivery links；P1 起支持 status、provider、source、session_id、batch_id、keyword 筛选，首屏默认有限加载、图片 lazy loading 和加载更多。
- Web 顶栏已可打开独立 scope 管理入口，对 workspace / project / campaign 执行 rename/archive/delete。
- Scope 管理已升级为控制台视图，可显示 workspace / project / campaign 的 project/campaign/asset 数、selected/rejected 数和最近活动；遇到需要项目 API key 的 scope 会跳过对应统计并提示。
- P1 Storage Governance 已完成：服务端可只读统计 instance/workspace/project/campaign 的存储占用和 task/asset/failed counts；Web Scope 管理可展示 project/campaign storage/original/thumbnail/metadata/input-files 分类；`vag storage cleanup-preview` 可 dry-run 预览 rejected/generated/tmp/orphan 清理候选；`vag storage cleanup-execute` 需要 `--execute` 加匹配 dry-run token 或显式 `--confirm`，仅清理 draft/rejected 及 tmp/orphan 文件，selected/published/approved 默认 protected；`storage-integrity` 可只读展示缺失文件、invalid current_version 和 stale queued/running 摘要且不暴露本地路径。
- 服务端创建任务可直接解析匿名 remote URL 和当前项目 `asset_id`，由 OpenAI-compatible 继续走 `/images/edits`。
- 服务端 `provider=fal` 已支持 queue 文生图，以及基于 scope `input-files`、匿名 remote URL、当前项目 `asset_id` 的 edit 输入复用。
- 服务端 best-of 已支持 `best_of_config`、`local_metadata_v1`、`http_judge_v1` 和可选 auto reject；外部 judge 失败时自动回退到本地启发式，auto rejected 候选仍可人工重新 select。
- Asset response 已暴露任务输入中的 `metadata_json`，资产库可显示 `source/session/batch/story/scene/target_path`。
- REST/CLI/Web 资产库已支持 `limit/status/provider/model/source/session_id/batch_id/keyword/created_from/created_to` 查询参数；服务端默认 limit=50、max=100，Web 服务端资产库默认 limit=24。
- 项目级 provider profile 已接入 `project.metadata_json.provider_profile`，只保存 `enabled/provider/model/base_url/generation_config/use_project_quality_profile` 等非敏感默认值；创建任务未显式传 provider 时可复用项目默认 provider/model，真实 provider secret 仍只走环境变量或后续确认的安全策略。
- provider profile 已扩展非敏感 capability 字段：`max_n`、`supports_url_result`、`preferred_response_format`、`max_concurrency`、`timeout_seconds`；`requested_count` 超过 `max_n` 时按同 task 拆分 provider 请求，多 scene 仍由外部编排为独立 task。
- P1 Web Performance / Startup 已完成：`initStore` 幂等、thumbnail backfill 后台预算、本地 TaskGrid 渲染预算、服务端资产库节点上限、fal/custom 恢复轮询上限、Scope 控制台统计缓存/边界，以及 Agent/Settings/Scope/Detail/Lightbox/Mask/Markdown/KaTeX 的按需加载都已接入。

当前尚未接入：

- 过期策略配置、批量清理 UI、更细 retention 策略和存储用量配额仍属于后续 P1/P2；第一版真实清理执行仅提供本地 CLI 受控入口，不提供远程匿名清理。
- 当前模型没有显式 `session/run/source_thread` 表；P0 先通过 `metadata_json` 留存，后续再决定是否升为正式模型。
- 参考图库、账号主形象、prompt recipe、prompt/edit lineage 仍是后续能力。
- 云端对外开放还缺注册/开通、配额、provider key 策略和生产部署 override。

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

Status: done. 当前 Docker image 内包含 `/app/mcp`，并已用真实 PostgreSQL/Redis/Worker 跑通 MCP stdio smoke。

### Phase 4: First real provider adapter

- 从原 Web 的 OpenAI-compatible / fal.ai provider 经验中迁移一个服务端 provider adapter。
- 第一优先级建议 OpenAI-compatible，因为 Base URL / API Key / model 形态最通用。
- Worker 负责调用真实 provider、下载 URL 或解析 base64，然后继续走现有 asset processor。
- 密钥只走环境变量或本地配置，不进入仓库。

Status: done. 当前已支持 `provider=openai-compatible`，通过 `OPENAI_COMPATIBLE_BASE_URL`、`OPENAI_COMPATIBLE_API_KEY`、`OPENAI_COMPATIBLE_MODEL` 配置；自动验证使用本地 HTTP mock，未调用真实外部 API。

### Phase 5: Web managed mode

- 给 Web 增加服务端托管模式。
- Web 创建服务端 `ImageTask`，轮询任务状态，展示服务端 thumbnail/original。
- Web 侧 select/reject 调用服务端 API，`selected` 作为推荐候选而非强制人工审核闸门。
- 原浏览器直连 provider 能力可保留为 legacy / playground mode，但正式资产流默认走服务端。

Status: done. 当前 Web 设置页可开启服务端托管模式，配置 API URL、workspace/project/campaign 和 provider；提交 prompt 后创建服务端 `ImageTask`，轮询任务状态，展示服务端候选图，并在详情页执行 select/reject 和打开 original / metadata URL。

Remaining gaps:

- 托管模式现在会先上传 reference/mask 到当前 scope 的 `input-files`；OpenAI-compatible 与 fal 在存在已解析输入文件时都可消费统一的 edit/mask 输入复用链路。
- 托管模式已能选择复用服务端项目级 quality profile，但 Web 暂未提供完整 profile 编辑界面。

### Phase 5.5: Quality foundation

- 保存 prompt template、style preset 和 reference image 参数。
- Web 托管模式创建任务时可以复用服务端模板/风格配置。
- 为多候选 best-of 自动选优提供稳定输入。

Status: done. 当前通过 `project.metadata_json.quality_profile` 保存项目级配置，REST 可读取/更新，REST/MCP/Web 创建任务可传 `use_project_quality_profile` 复用配置；有效 prompt/template/style/reference/config 快照写入任务 `structured_input_json`。

### Phase 5.6: Best-of auto selection

- 在多候选任务上增加第一版服务端自动选优。
- `selection_mode=auto` / `best_of` 时，Worker 生成资产后自动 selected 一张推荐候选。
- 当前默认使用 `local_metadata_v1`，也可通过 `best_of_config.strategy=http_judge_v1` 接入外部 judge；若显式设置 `best_of_config.auto_reject_non_selected=true`，则未入选候选会自动 rejected。

Status: done. 当前 `selection_mode` 已进入 REST/MCP/Web 托管输入、任务 `structured_input_json` 和 `GET /api/tasks/{id}` 响应；`best_of_config` 已进入任务输入、quality profile 和 quality snapshot；Web managed mode 默认传 `selection_mode=auto`；REST smoke 已验证 auto 恰好 1 张 selected、manual_optional 0 张 selected；`http_judge_v1` 本地 smoke 已验证 `task_9f3b4f5551fbdf5b8e06 -> asset_1837f5fd3e8e6977dcb3`；`auto_reject_non_selected` 本地 smoke 已验证 `task_79ee5fdfe639cd532805` 产生 1 张 selected + 2 张 rejected，且 auto rejected 候选仍可人工重新 select。

Remaining gaps:

- 当前 MVP 范围内的生产 hardening 已完成；更进一步的 key usage 计数、到期时间或 RBAC 不在 MVP 内。

### Phase 5.7: Advanced managed input

- 将 Web/MCP/REST 的 reference image、mask/edit descriptor 和更多 generation config 带入服务端托管任务。
- 让 asset `parameters_json` 保留高级输入快照，方便后续 provider adapter 读取。
- Web managed mode 不再因为输入图或 mask 阻止提交，但本片不上传原图或执行真实 edit/mask provider 请求。

Status: done. 当前 `CreateTaskRequest` 支持 `mask_image`，`reference_images` 支持 `source` / `mime_type` / `width` / `height`；mock REST smoke 已验证 asset `parameters_json` 包含 reference/mask/generation config 快照。

### Phase 5.8: Web scope selector and quick create

- 给 Web 设置页补服务端 scope 同步体验。
- 可以从服务端读取已有 workspace / project / campaign，并用下拉切换当前 scope。
- 可以在设置页直接快速新建 workspace / project / campaign，并在创建后自动切换到新 scope。
- 保留手填 ID 作为兼容兜底。

Status: done. 当前 REST 已支持列出/创建 workspace、project、campaign；Web 设置页可同步 scope、选择当前业务空间，并快速新建后直接用于托管任务创建。

### Phase 5.9: Managed input upload and real edit/mask boundary

- 给当前 scope 新增 `input-files` 上传、元数据和内容读取接口。
- Web 托管模式在存在 reference image / mask 时，先上传输入文件，再创建带 `input_file_id` 的服务端任务。
- `CreateTask` 会把公开 `input_file_id` 解析成内部 `resolved_input_files`。
- `openai-compatible` provider 在存在已解析输入文件时走 `/images/edits` multipart。

Status: done. 当前 Docker smoke 已验证上传 reference/mask 后，Worker 对本地 HTTP mock 发出 `/images/edits` multipart 请求（`image_count=1`、`mask_count=1`），并成功完成服务端 asset 闭环。

### Phase 6: MVP hardening

- 已补 repair/reconcile 命令：`vag repair scan`、`vag repair requeue <task_id>`、`vag repair verify-asset <asset_id>`。
- 已补 worker retry / backoff / delayed queue handling：provider 瞬时失败会写 `task_attempt.retry_after`，重新排入 Redis delayed queue，并在后续 attempt 自动恢复。
- 已补服务端真实缩略图处理：基于原图生成按比例缩放的 `.webp` 缩略图，并统一 `thumbnail_path` / `Content-Type`。
- 增加项目级 API key、基本鉴权和配置样例。
- 完善 README、demo 流程和自托管运行说明。

Status: done. Repair/reconcile smoke、自动 retry/backoff、真实缩略图 resize/webp、项目级 API key、Basic Auth、配置样例、README/demo、自托管最小暴露面和真实 edit/mask 边界均已完成。

### Phase 6.1: Standalone scope management

- 给 Web 增加独立的 scope 管理入口，而不是只在设置页内联处理。
- 支持 workspace / project / campaign 的基础 rename、archive 和 delete。
- 保持现有设置页的同步、新建和托管任务流程继续可用。

Status: done. 当前 Web 顶栏和设置页都可进入独立 scope 管理 modal；REST 已支持 workspace/project/campaign 的 `PATCH` / `DELETE`，列表返回 `archived` 状态；Docker smoke 已验证 rename、archive/unarchive、非空 scope 删除报错、删除 campaign 时的输入文件目录清理，以及当前托管 scope 的切换不回退。

### Phase 6.2: Input reuse expansion

- 让服务端托管任务支持远程 URL 抓取和已有 `asset_id` 复用，而不是只依赖 `input-files` 上传。
- 扩大 provider adapter 对复用输入的消费能力，至少覆盖当前 OpenAI-compatible 之外的一条真实 edit/mask 路径。
- 保持 `structured_input_json` / `parameters_json` 的可追踪快照一致性。

Status: done. 当前服务端已支持 remote URL 抓取、当前项目 `asset_id` 复用，以及 OpenAI-compatible / fal 两条真实 provider 输入复用路径；本地 Docker smoke 已验证 fal queue/storage edit 流程命中 `GET /remote.png`、两次 `POST /rest/storage/upload/initiate`、`POST /queue/openai/gpt-image-2/edit`，并成功完成 `task_0dbae47c6d0459cd8c2c -> asset_96d78f9da6b1fcdb0cca`。

### Phase 6.3: Project access multi-key strategy

- 在不改数据库 schema 的前提下，把 `project.metadata_json.access_config` 从单 key 升级为兼容视图 + `api_keys` 列表。
- 保持 `GET/POST /access-config` 路由不变，并支持新增、更新/轮换、禁用、删除单把 key。
- 项目级鉴权接受任意一把启用 key，并把命中的 key 名称带入审计 actor。

Status: done. 当前 Docker smoke 已验证 `prj_multi_key_1781784728` 下的 `default` 与 `rollout` 两把 key 都能访问同一 project 资源；disabled/deleted `default` 后，`rollout` 仍可继续读取 `task_fc9e1275b4dcb665e766`，并可通过 `vag audit list --project prj_multi_key_1781784728` 查到命中的 `rollout` actor。

### Phase 6.4: Concurrency and benchmark hardening

- 让 `WORKER_CONCURRENCY` 通过环境变量覆盖，避免平台默认只能实际串行处理任务。
- 增加 `OPENAI_COMPATIBLE_MAX_CONCURRENCY` provider 级 backpressure，避免 worker 并发升高后直接打爆真实 provider。
- 暴露 task attempts 的只读 API/CLI/Web 摘要，用于定位 provider latency、timeout、retry_after 和失败原因。
- 增加 `vag benchmark image-generation`，支持 mock 无费用压测和真实 provider 小样本费用保护压测。

Status: done. 本地 Docker 已验证 worker=1 与 worker=4 的 mock 延迟压测：32 个任务、`mock_delay_ms=250` 下 wall-clock 从 12.427s 降到 2.979s，提升约 4.17x；真实 provider benchmark 未自动执行，需用户确认费用后运行。

### Phase 6.5: Provider throughput and reliability hardening

- 将真实 provider 默认入口 cap 收敛为 `OPENAI_COMPATIBLE_MAX_CONCURRENCY=3`、`FAL_MAX_CONCURRENCY=3`，保留 `WORKER_CONCURRENCY=6`。
- 将默认 provider timeout 提升为 `300s`，并为 openai-compatible 增加 connect/header/total timeout profile。
- `task_attempt` 新增 queue/provider/download/store/thumbnail/retry/error_stage/response_bytes 阶段指标。
- `provider_profile` 增加非敏感 capability 字段；`requested_count` 可在同 prompt 场景下按 `max_n` 拆分 provider 请求。
- `vag benchmark image-generation` 输出诊断指标和调参建议，并用 `session_id/batch_id=run_id` 支持 `vag batch progress`。

Status: done. Docker `go test ./...`、`npm --prefix web test -- --run`、`npm --prefix web run build`、`docker compose config` 均通过；mock benchmark `bench_p1_provider_rel_batch` 3 任务 / 6 资产全部完成，batch progress 返回 `task_count=3`、`succeeded_count=3`、`asset_count=6`、`attempt_count=3`；真实 provider benchmark 未运行，仍需用户确认费用。

### 30-day portfolio version

- React 控制台。
- 缩略图预览和候选图选优页面。
- MCP stdio server。
- Provider adapter 抽象。
- Docker Compose。
- README、Demo GIF、示例自动化流程。

### Later

- Web server asset sync：让 MCP / REST / CLI 生成的资产在 Web 管理视图中可见。
- Asset Library & Storage Governance：资产库筛选、存储占用、过期策略、批量清理、异常治理。
- Platform Console / Scope Dashboard：展示所有 workspace / project / campaign、资产数量、任务数量、最近活动和当前 scope。
- Session / run / source tracking：按 Codex thread、自动化运行、内容账号会话追踪资产。
- Reference Library / Prompt Recipe / Mascot Profile：保留账号主形象、参考图、prompt 和 edit lineage。
- Cloud deployment hardening：生产 compose override、关闭 DB/Redis 公网端口、反向代理/TLS/鉴权模板。
- External onboarding：注册或管理员开通、project API key 发放、配额和 provider key 策略。
- Diagram source track：若要支撑嵌入式架构图的可编辑源，补 Mermaid/D2/SVG/source retention 与渲染链路。
- 多 provider 策略和成本控制。
- MinIO/S3 存储。
- 本地 ComfyUI / GPU provider。
- webhook。
- 公开 API key。
- Notion / GitHub / CMS 交付适配。
- Streamable HTTP MCP。
