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
- [x] API/CLI/MCP 入口可跑通。
- [x] MCP stdio server 可列出 tools 并通过 tool call 创建任务、查询任务、标记 selected 和获取 delivery。
- [x] 服务端 Worker 已支持第一个真实 provider adapter：OpenAI-compatible。
- [x] Web 已支持服务端托管模式：创建服务端 ImageTask、轮询 assets、展示服务端候选图并执行 select/reject。
- [x] Prompt template、style preset、reference image 参数和 generation config 已形成服务端项目级复用策略。
- [x] 多候选 best-of 自动选优已形成第一版服务端策略：`selection_mode=auto` / `best_of` 时自动 selected 一张候选。
- [x] best-of 已支持可插拔 scorer：任务/quality profile 可指定 `best_of_config`，当前已接入 `local_metadata_v1` 与 `http_judge_v1`，外部 judge 失败时回退本地启发式。
- [x] best-of 已支持可选 auto reject：`best_of_config.auto_reject_non_selected=true` 时，未入选候选会自动 rejected，且后续仍可人工重新 select。
- [x] Web/MCP/REST 已能把 reference image、mask/edit descriptor 和更多 generation config 带入服务端任务，并写入 asset `parameters_json`。
- [x] Web/REST 已支持 scope 内 `input-files` 上传；OpenAI-compatible 任务在存在已解析输入文件时会走 `/images/edits` multipart，并完成服务端 asset 闭环。
- [x] REST/MCP 已支持 remote URL 抓取和当前项目 `asset_id` 复用；OpenAI-compatible 可把这些来源解析到 `resolved_input_files` 并继续完成 `/images/edits` 闭环。
- [x] 服务端 `provider=fal` 已支持 queue 文生图，以及基于 `resolved_input_files` 的 storage upload + edit 闭环，可复用 `input-files`、匿名 remote URL 和当前项目 `asset_id`。
- [x] 本地 repair/reconcile smoke 已接入：`vag repair scan`、`vag repair requeue`、`vag repair verify-asset` 可发现并修复入队失败任务、校验资产文件。
- [x] HTTP API 已支持基础实例级 / project 级限流：配置阈值后返回 `429`、结构化错误 JSON 和 `Retry-After`，MCP stdio / Worker 不受直接影响。
- [x] HTTP / API 已支持第一版结构化审计日志：`/api/*` 请求会写入本地 JSONL 审计事件，并可通过 `vag audit list` 按 project / task / status 过滤查询。
- [x] 项目级 access config 已支持多把命名 key：可 add/update/disable/delete 单把 key，任意启用 key 都可访问 project 级资源，审计会优先记录命中的 key 名称。
- [x] Web 已支持服务端资产库同步：当前 scope 下来自 MCP / REST / CLI / Web 的 assets 可通过服务端 list assets 接口显示，不依赖浏览器本地任务历史。
- [x] Web 已支持最小资产库视图：可显示缩略图、prompt、provider、model、status、source、task_id、asset_id、created_at、delivery links，并可 select/reject/copy/open。
- [x] Scope 管理已升级为基础控制台：workspace / project / campaign 列表显示资产数量、selected/rejected 数和最近活动，且可从控制台切换当前 scope。
- [x] Asset response 已暴露任务 `metadata_json`：MCP/REST 传入的 `source/session/batch/story/scene/target_path` 可在资产 metadata 和 Web 资产库中显示。
- [x] P1 Storage Governance 已接入：服务端可只读统计 instance/workspace/project/campaign 存储占用，REST 可返回当前 scope + 实例级 usage/counts，Web Scope 管理可展示 storage/original/thumbnail/metadata/input-files 等占用，CLI 可 dry-run 预览 rejected/generated/tmp/orphan 清理候选。
- [x] P1 Storage Governance 已完成受控执行与治理可见性：`vag storage cleanup-execute` 需要 `--execute` 加 dry-run token 或显式确认；selected/published/approved 默认 protected；执行会写本地 CLI audit；REST/Web 可读取脱敏 `storage-integrity` 摘要。
- [x] P1 Asset Production Readiness 已完成：REST/CLI 资产列表支持 limit/status/provider/model/source/session_id/batch_id/keyword/date 筛选；Web 资产库支持筛选、加载更多、lazy loading 和 metadata/parameters 摘要；`metadata_json` 标准化 source/session/run/batch/story/scene/target_path；项目级非敏感 provider profile 可保存并作为默认 provider/model 复用。
- [x] P1 Web Performance / Startup 已完成：Web 启动初始化已幂等；本地 thumbnail backfill、TaskGrid 渲染、服务端资产库保留节点、fal/custom 恢复轮询和 Scope 控制台统计都有边界；Agent/Settings/Scope/Detail/Lightbox/Mask 与 Markdown/KaTeX 样式改为按需加载。
- [x] 并发与生图性能专项已完成：Worker 支持环境变量并发覆盖，openai-compatible provider 有独立并发 cap；任务 attempts 可通过 REST/CLI/Web 查看；`vag benchmark image-generation` 可做 mock 和受控真实 provider 小样本压测。
- [x] P1 Provider Throughput & Reliability 已完成：真实 provider 默认 cap 收敛为 3，openai-compatible timeout profile 和 task attempt 阶段指标已接入；provider capability profile、同 prompt 多图拆分、diagnostic benchmark 和 batch progress 已可验证。

## Remaining Product Gaps

- [x] MVP 产品闭环与生产 hardening 缺口已清零；当前仅剩本地开发环境清理类 follow-up，不属于产品缺口。

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
- 2026-06-18: 已完成 MCP stdio entry：`/app/mcp` 支持 initialize、tools/list、tools/call；真实 smoke 跑通 create task -> get task -> select asset -> get delivery info，MCP 输出使用 generated/selected 语义。
- 2026-06-18: 已完成服务端 OpenAI-compatible provider adapter：支持 `images/generations`、`data[].b64_json`、`data[].url`、PNG 规范化、raw response/cost/parameters 记录；本地 HTTP mock 集成 smoke 跑通 `provider=openai-compatible` 任务到 ready asset。
- 2026-06-18: 已完成 Web managed mode 第一版：设置页可开启服务端托管模式并配置 API URL / workspace / project / campaign / provider；Web 创建服务端 `ImageTask`，轮询 `GET /api/tasks/{id}`，任务卡和详情页展示服务端候选图，详情页可 select/reject 并打开 original / metadata URL；Web 测试、构建、Docker Compose build 和 REST mock smoke 均通过。
- 2026-06-18: 已完成 Quality foundation：使用 `project.metadata_json.quality_profile` 保存项目级 prompt template、style preset、reference image 参数和 generation config；REST/MCP/Web 创建任务可通过 `use_project_quality_profile` 复用配置，模板渲染后的有效 prompt 和配置快照写入 `structured_input_json`；Go/Web 测试、Docker Compose build、REST profile smoke 和 MCP create smoke 均通过。
- 2026-06-18: 已完成 Best-of auto selection 第一版：`selection_mode` 进入统一 `ImageTask` 输入、`structured_input_json` 和 MCP/Web 托管创建路径；Worker 在 `auto` / `best_of` 模式下用 `local_metadata_v1` 本地启发式自动 selected 一张候选；REST smoke 验证 auto 任务恰好 1 张 selected、manual_optional 任务 0 张 selected，自动 selected 可人工 reject 覆盖。
- 2026-06-18: 已完成 Advanced managed input 第一版：REST/MCP/Web 可传 `reference_images` 扩展 descriptor、`mask_image` 和 `generation_config`；Web managed mode 不再拒绝输入图或 mask；mock REST smoke 验证 asset `parameters_json` 保留 reference/mask/generation config 快照。
- 2026-06-18: 已完成 Repair/Reconcile smoke 第一版：新增本地 `vag repair scan/requeue/verify-asset`；Docker smoke 验证模拟 `enqueue_failed` 任务可被 scan 报告、requeue 后由 Worker 处理完成，资产文件存在时 `verify-asset ok=true`，缺失 original 时报告 `missing_file`。
- 2026-06-18: 已完成 Worker retry/backoff 第一版：Worker 会先 promote Redis delayed queue，再消费主队列；provider 瞬时失败会把 attempt 标记 failed 并写入 `retry_after`，任务回到 `queued` 后自动重试；Docker smoke 验证 `mock_failure_mode=transient_once` 任务第 1 次失败、第 2 次成功。
- 2026-06-18: 已完成真实缩略图 slice：Worker/Storage 不再依赖 provider thumbnail bytes，而是基于原图生成 `.webp` 缩略图；Docker smoke 验证 `thumbnail_path` 为 `.../1.webp`、`GET /api/assets/{id}/thumbnail` 返回 `image/webp`，文件头为 `RIFF....WEBP`。
- 2026-06-18: 已完成项目级 API key + Basic Auth slice：`project.metadata_json.access_config` 保存单把项目 key 的 hash/name/preview，REST 支持 `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/access-config`，API 可选启用实例级 Basic Auth，项目级请求可要求 `X-API-Key` 或 Bearer；Docker smoke 验证未认证 `401`、仅 Basic 访问受保护 project `401`、Basic + project key 可创建任务并读取 asset，CLI `vag` 和 Web managed mode 均已支持透传鉴权。
- 2026-06-18: 已完成 Web scope selector / quick create slice：REST 新增列出/创建 workspace、project、campaign；Web 设置页可同步服务端 scope、用下拉选择当前 workspace/project/campaign，并直接快速新建后自动切换；Docker smoke 验证新建 `ws_scope_smoke -> prj_scope_smoke -> cmp_scope_smoke` 后可直接在新 campaign 下创建任务并产出 asset。
- 2026-06-18: 已完成 Managed input upload / real edit-mask slice：REST 新增 scope 内 `input-files` 上传、元数据和内容读取；Web managed mode 会先上传 reference image / mask，再创建带 `input_file_id` 的服务端任务；Docker smoke 使用本地 HTTP mock 验证 Worker 真实调用 `/images/edits` multipart（`image_count=1`、`mask_count=1`）并完成 `task_dd1a410a094e30f06fc5 -> asset_fb9f0bbe559c4c95aa88` 闭环。
- 2026-06-18: 已完成 README/demo/self-hosting 文档 slice：重建 `README.md` 为仓库入口，补齐 quickstart、mock/Web/MCP demo、自托管最小暴露面和反向代理/TLS 样例；`RUNBOOK.md` 同步补充最小自托管建议；`docker compose config`、`curl -u admin:secret http://127.0.0.1:8081/healthz` 和 `curl http://127.0.0.1:8080` 验证通过。
- 2026-06-18: 已完成 Standalone scope management slice：REST 新增 workspace/project/campaign `PATCH` / `DELETE`，列表返回 `archived`；Web 顶栏新增独立 scope 管理 modal，可 rename/archive/delete 并设为当前 scope；Docker smoke 验证 rename、archive/unarchive、非空 scope 删除报错、campaign 删除后的 `input-files` 目录清理，以及 workspace/project/campaign 最终可完整删除。
- 2026-06-18: 已完成 Remote URL + asset reuse slice：服务端 `CreateTask` 现在可解析 `reference_images[].url`、`reference_images[].asset_id`、`mask_image.url`、`mask_image.asset_id`；remote URL 会被抓取并物化为当前 scope `input-files`，`asset_id` 复用限定在当前 workspace/project；Docker smoke 验证 `GET /remote.png` 被服务端抓取、OpenAI-compatible 请求命中 `/v1/images/edits` 且 `image_count=2`，并成功完成 `task_91237d5d15aa7252bed4 -> asset_9ab0aeca719c6e9a2f66`，最终 `parameters_json.reference_images` 同时保留原始 remote URL、生成的 `input_file_id` 和复用的 `asset_id`。
- 2026-06-18: 已完成 fal provider input reuse slice：服务端新增 `provider=fal` queue + rest storage adapter，统一复用 `resolved_input_files`；Docker smoke 验证 `GET /remote.png`、两次 `POST /rest/storage/upload/initiate`、`POST /queue/openai/gpt-image-2/edit`，并成功完成 `task_0dbae47c6d0459cd8c2c -> asset_96d78f9da6b1fcdb0cca`，最终 asset `parameters_json` 保留 `request_mode=edit`、`endpoint_id=openai/gpt-image-2/edit` 和原始 reference 快照。
- 2026-06-18: 已完成 Best-of pluggable scoring slice：服务端在任务输入和项目级 quality profile 中新增 `best_of_config`，scorer registry 当前支持 `local_metadata_v1` 与 `http_judge_v1`；本地 HTTP judge smoke 验证 `task_9f3b4f5551fbdf5b8e06 -> asset_1837f5fd3e8e6977dcb3`，mock judge 收到 `POST /score` 候选缩略图 data URL，请求与 `review_event.note` 一致记录 `requested_strategy=http_judge_v1`、`applied_strategy=http_judge_v1`。
- 2026-06-18: 已完成 Best-of auto reject slice：服务端在 `best_of_config` 中新增 `auto_reject_non_selected`，自动选优后可事务式写入 `selected + rejected siblings`；本地 smoke 验证 `task_79ee5fdfe639cd532805` 生成 1 张 selected 与 2 张 rejected，并成功将 auto rejected 的 `asset_5d207d1a89b3ba6d6793` 手动重新 select 为 approved/selected。
- 2026-06-18: 已完成 HTTP rate limiting slice：服务端 `api` 进程新增 Redis 固定窗口限流，支持 `RATE_LIMIT_WINDOW_SECONDS`、`RATE_LIMIT_INSTANCE_MAX_REQUESTS`、`RATE_LIMIT_PROJECT_MAX_REQUESTS`；局部/全量 Go 测试、`npm --prefix web run build`、`docker compose config` 通过；Docker smoke 验证独立 `prj_rate_limit_smoke` 下 `POST /tasks` 命中 `201 -> 429`，实例级请求也返回了 `429` 与 `Retry-After`。
- 2026-06-18: 已完成 HTTP / API audit log slice：服务端 `api` 进程会把 `/api/*` 请求写入 `STORAGE_ROOT/audit/http-api/YYYY-MM-DD.jsonl`；局部/全量 Go 测试、`npm --prefix web run build`、`docker compose config` 通过；Docker smoke 验证 `create_task`、`get_task` 与 `404 not_found` 都会留下审计记录，并可通过 `docker compose exec api /app/vag audit list --project prj_audit_smoke --task task_071eb31a8b161a4de0b5` 查询。
- 2026-06-18: 已完成 project multi-key slice：服务端 `project.metadata_json.access_config` 已支持兼容视图 + `api_keys` 列表；全量 `go test ./...`、`npm --prefix web run build`、`docker compose config` 通过；Docker smoke 验证 `prj_multi_key_1781784728` 同时接受 `default` 与 `rollout` 两把 key，旧 key disable/delete 后 `rollout` 仍能读取 `task_fc9e1275b4dcb665e766`，并在审计里记录 `actor=rollout`。
- 2026-06-19: 已将第二阶段 P0 visibility 需求拆成可由 `/goal` 执行的 CSV 工单：`issues/next-phase-p0-visibility.csv`；范围限定为 Web 服务端资产同步、最小资产库、Scope Dashboard、source/session metadata 标准和整体回归检查。
- 2026-06-19: 已完成第二阶段 P0 visibility：通过 MCP 创建 `task_8df69831d9ae5c4aa92a -> asset_01cc40da9ee74c9f5368`，REST `list assets` 返回 `metadata_json.source=mcp`、`session_id=p0vis_open_1781800411`、`batch_id=p0vis_open_1781800411_batch` 和 delivery URLs；浏览器刷新后 Web 服务端资产库仍显示该 MCP asset，并可执行 reject/select，REST 最终状态回到 `approved`。
- 2026-06-19: 已完成 P1 Storage Governance `P1-STOR-001` 到 `P1-STOR-005`：`npm --prefix web test -- --run` 通过 17 files / 222 tests，`npm --prefix web run build` 通过且仅有既有 chunk warning，Docker `go test ./...` 通过；`GET /api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-governance` 未带 project key 返回 401，带 smoke key 返回 200，并只返回统计数字/scope id；`vag storage cleanup-preview --limit 5` 返回 `dry_run=true`，候选只包含 `generated_unselected_asset` 和 `rejected_asset`，同时报告 selected/published protected 计数；浏览器 Scope 管理已显示 `storage/original/thumbnail/metadata` 统计。
- 2026-06-19: 已完成 P1 Storage Governance `P1-STOR-006` 到 `P1-STOR-008`：新增 `vag storage cleanup-execute` 受控执行、storage-root 相对路径删除保护、draft/rejected 资产 DB 行事务删除和 CLI audit 记录；Docker fixture `ws_p1stor_smoke_1781852988/prj_p1stor_smoke_1781852988/cmp_p1stor_smoke_1781852988` 生成 3 张 mock 资产，approve 1 张、reject 1 张、保留 1 张 draft，dry-run 返回 2 个候选和 1 个 protected；无 `--execute` 执行被拒绝并写入失败 audit，带 token 执行删除 2 个候选 / 6 个文件并写入成功 audit；后续 list assets 只剩 approved 资产，二次 dry-run 为 0，`storage-integrity` 返回 `ok=true issue_count=0`，浏览器 Scope 管理显示该 workspace 的 post-cleanup 统计。
- 2026-06-19: 已完成 P1 Asset Production Readiness：新增 `AssetListQuery`、metadata 标准化、`project.metadata_json.provider_profile`、provider model override、`vag asset list` 筛选参数和 `vag project provider get/set`；Web 服务端资产库新增 status/provider/source/session/batch/keyword 筛选、limit=24 首屏、加载更多、图片 lazy loading 和折叠详情；新增 `sample-codex-pet-story-task.json` 与 `sample-codex-embedded-architecture-task.json`；`npm --prefix web test -- --run` 通过 17 files / 222 tests，`npm --prefix web run build` 通过且仅有既有 chunk warning，Docker `go test ./...` 通过；REST smoke 验证 `task_a1e1ef5a230207e4bda9 -> asset_b8ddafa3547c75aca91a`，按 `source/session_id/batch_id` 过滤可查回该资产；浏览器验证 Web 资产库首屏 24 张 lazy thumbnail，按 source/session/batch 筛选后仅显示该 P1 smoke asset。
- 2026-06-19: 已完成 P1 Web Performance / Startup：只读 profile 显示用户侧 Chrome `High memory usage` 约 1.1GB，Vite dev server 进程约 132-136 MB RSS，production preview 进程约 103 MB RSS，因此 dev/HMR 有放大但不是唯一来源；本轮未清理 IndexedDB 或资产。已新增 `initStore` in-flight/completed guard、thumbnail backfill 后台队列 120 / 每 session 48 上限、TaskGrid 首屏 60 条 + 加载更多、ServerAssetLibrary 最多保留 120 个已渲染资产、fal/custom 自动恢复 5 个 / 6 小时窗口、Scope 统计 60s 缓存和扫描边界；`App` 懒加载 Agent/Settings/Scope/Detail/Lightbox/Mask，`MarkdownRenderer` 按需加载 streamdown/KaTeX CSS。`npm --prefix web test -- --run` 通过 17 files / 224 tests，`npm --prefix web run build` 通过，主入口包由约 951 KB 降到约 711 KB；浏览器自动化在高内存页面上 `transport closed`，后续若仍复现需单独做 heap snapshot / virtualized list 专项。
- 2026-06-19: 已完成并发与生图性能专项：默认 `WORKER_CONCURRENCY=6`、`OPENAI_COMPATIBLE_MAX_CONCURRENCY=6`；task attempts REST/CLI/Web 可观测，openai-compatible `quality/moderation/output_compression` 透传，新增 `vag benchmark image-generation`。本地 mock 延迟压测：worker=1 32 任务 `mock_delay_ms=250` wall-clock 12.427s，worker=4 同配置 wall-clock 2.979s；worker=6 最大 mock 组 32 任务 / `requested_count=4` / 128 资产 wall-clock 14.239s 且全部完成。真实 provider 经用户确认后执行 c6 小样本：6 任务 4 成功、2 个 120s timeout，worker 内存峰值约 50.60MiB，API 约 26.48MiB，说明瓶颈在 provider 侧而不是本机资源。
- 2026-06-19: 已完成 P1 Provider Throughput & Reliability：默认保留 `WORKER_CONCURRENCY=6`，真实 provider 默认收敛为 `OPENAI_COMPATIBLE_MAX_CONCURRENCY=3`、`FAL_MAX_CONCURRENCY=3`、`PROVIDER_TIMEOUT_SECONDS=300`，openai-compatible 增加 connect/header/total timeout profile；`task_attempt` 新增 `queue_wait_ms/provider_first_byte_ms/provider_total_ms/response_download_ms/store_ms/thumbnail_ms/retry_count/error_stage/response_bytes`；provider profile 增加 `max_n/supports_url_result/preferred_response_format/max_concurrency/timeout_seconds`；`requested_count` 超过 provider `max_n` 时按同 task 拆分 provider 请求；新增 `GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-progress` 和 `vag batch progress`。验证：Docker `go test ./...` 通过，`npm --prefix web test -- --run` 通过 17 files / 224 tests，`npm --prefix web run build` 通过且仅有既有 chunk warning，`docker compose config` 通过；mock benchmark `bench_p1_provider_rel_batch` 3 任务 / 6 资产全部完成，batch progress 返回 `task_count=3`、`succeeded_count=3`、`asset_count=6`、`attempt_count=3`；task `task_6330ef96181ccb074ca5` attempts 可见 `queue_wait_ms/provider_total_ms/store_ms/thumbnail_ms/retry_count`；真实 provider benchmark 未运行，因为仍需费用确认。
