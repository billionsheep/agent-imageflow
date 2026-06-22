# Tasks

## Todo

- [ ] 旧 P1 拆分 CSV 保留为历史参考，不再作为下一轮主入口：`issues/next-phase-p1-asset-library-filters.csv`、`issues/next-phase-p1-session-source-tracking.csv`、`issues/next-phase-p1-provider-profile-cloud-safety.csv`。
- [ ] 用 production preview 路径继续试用 Web，观察是否仍出现 Chrome `High memory usage`；若仍复现，再单独做浏览器 heap/virtualized list 专项。
- [ ] 继续执行 `issues/next-phase-p1-project-production-context.csv` 的后续项：P1-PCTX-001 到 P1-PCTX-007 已完成服务端核心和自动化 examples；下一步按 CSV 推进 P1-PCTX-008 到 P1-PCTX-009，不把范围扩成通用 DAM、账号运营系统或导出平台。
- [ ] 后续再规划 Batch / Story / Scene View、Export Pack、NAS/WebDAV/SMB 等能力；这些不进入 Project Visual Context 第一轮核心实现。
- [ ] 澄清嵌入式架构图场景：若需要 Mermaid/D2/SVG 可编辑源，需要补 Diagram source track；若只作为图片资产，可沿用当前资产闭环。

## Doing

- [ ] 当前建议主线推进 `issues/next-phase-p1-project-production-context.csv` 的 P1-PCTX-008 到 P1-PCTX-009，完成最小 Web context 面板和萌宠故事多 scene 回归。

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
- [x] 实现 MCP stdio server，复用现有服务端 application core。
- [x] 实现 `create_image_task` / `get_image_task` / `list_image_assets` / `select_image_asset` / `reject_image_asset` / `get_asset_delivery_info` MCP tool schema。
- [x] 迁移第一个真实 provider 到服务端 Worker：OpenAI-compatible `images/generations` adapter。
- [x] 将 Web 现有生成交互接入服务端 `ImageTask/Asset` 托管模式。
- [x] 实现 Web 候选图详情页 select/reject 交互，当前使用设置页中的 workspace / project / campaign scope。
- [x] 增加 prompt 模板、style preset、reference image 参数和 generation config 的服务端保存/复用策略。
- [x] Web 托管模式创建任务时可通过 `use_project_quality_profile` 复用服务端项目级质量配置。
- [x] 实现服务端 best-of 自动选优：`selection_mode=auto` / `best_of` 时 Worker 自动 selected 一张候选。
- [x] 将 Web/MCP/REST 的 reference image、mask/edit 描述符和更多生成参数迁移到服务端托管任务；当前先保存/传递 descriptor，不上传原图或执行真实 edit/mask provider 调用。
- [x] 增加本地 `vag repair scan` / `vag repair requeue` / `vag repair verify-asset`，可扫描可恢复任务、重入队 `enqueue_failed` 任务，并校验资产文件一致性。
- [x] 增强 Worker 自动 retry/backoff：provider 瞬时失败会写入 `task_attempt.retry_after`、回到 delayed queue，并由 Worker 自动重试。
- [x] 增强缩略图处理为服务端真实 resize / `.webp` 输出，统一缩略图路径与 MIME。
- [x] 增加项目级 API key、实例级 Basic Auth、CLI/Web 鉴权透传和自托管配置样例。
- [x] 完善 Web scope 选择与快速新建：设置页可从服务端加载 workspace / project / campaign，并直接创建新 scope，不再主要依赖手填 seed scope。
- [x] 接入服务端输入文件上传/取回与第一版真实 edit/mask 闭环：Web 托管模式会先上传 reference image / mask，再由 OpenAI-compatible provider 走 `/images/edits` multipart。
- [x] 补 README/demo 和更完整的自托管部署说明（反向代理/TLS、最小暴露面）。
- [x] 补独立 Web scope 管理页与基础 rename/delete/归档体验。
- [x] 补远程 URL 抓取与 asset reuse：服务端创建任务可直接解析匿名 `http/https` 图片 URL 或当前项目 `asset_id`，并由 OpenAI-compatible 继续走 `/images/edits` multipart。
- [x] 把 remote URL / asset reuse 的 edit/mask 输入复用扩到第二条真实 provider：`provider=fal` 现已可消费 `input-files`、匿名 remote URL 和当前项目 `asset_id`，并通过 fal queue/storage 完成 edit 闭环。
- [x] 实现 best-of 可插拔评分：任务/quality profile 可传 `best_of_config`，服务端支持 `local_metadata_v1` 与 `http_judge_v1`，外部 judge 失败时回退本地启发式。
- [x] 实现 best-of 可选 auto reject：`best_of_config.auto_reject_non_selected=true` 时，自动选优后会把未入选候选标记为 rejected，且 auto rejected 候选仍可人工重新 select。
- [x] 增加第一版实例级 / project 级 HTTP 基础限流：支持 `RATE_LIMIT_WINDOW_SECONDS`、`RATE_LIMIT_INSTANCE_MAX_REQUESTS`、`RATE_LIMIT_PROJECT_MAX_REQUESTS`，命中后返回 `429` 和 `Retry-After`。
- [x] 补第一版 HTTP / API 审计日志：`/api/*` 请求会写入本地 JSONL 审计事件，`vag audit list` 可按 project / task / asset / status 过滤查询。
- [x] 扩展项目级更多 key 策略：`access-config` 已支持 `api_keys` 列表、add/update/delete 动作、任意启用 key 鉴权和命中的 key 名称审计。
- [x] 补本地 Web `.vite/` 生成目录 ignore 规则，避免运行态缓存长期出现在 `git status`。
- [x] 记录 MVP 试用后的后续需求和真实场景：资产库治理、MCP/Web 同步、云端安全、对外注册、萌宠账号和嵌入式架构图场景，详见 `docs/project/FUTURE_REQUIREMENTS_AND_SCENARIOS.md`。
- [x] 生成第二批需求文档：`docs/project/NEXT_PHASE_REQUIREMENTS.md`，明确 P0 为 Web 服务端资产同步、最小资产库、控制台 / Scope Dashboard、source/session metadata 标准。
- [x] 生成第二阶段 P0 visibility CSV 工单：`issues/next-phase-p0-visibility.csv`，用于后续 `/goal` 逐条执行和验收。
- [x] 完成第二阶段 P0 visibility 工单：Web 已可同步显示 MCP/REST/CLI/Web 创建的服务端资产；新增最小服务端资产库；Scope 管理升级为带统计的控制台；asset response 暴露任务 `metadata_json`，资产库可显示 source/session/batch/story/scene/target_path；已用 MCP smoke 和浏览器刷新验证。
- [x] 生成下一阶段 P1 CSV 工单：`issues/next-phase-p1-storage-governance.csv`、`issues/next-phase-p1-asset-library-filters.csv`、`issues/next-phase-p1-session-source-tracking.csv`、`issues/next-phase-p1-provider-profile-cloud-safety.csv`。
- [x] 将 Web 打开 CPU 偏高问题纳入下一轮 P1，生成性能专项工单：`issues/next-phase-p1-web-performance-startup.csv`。
- [x] 完成 P1 Storage Governance：新增 storage usage scanner、只读 storage-governance API、Web Scope 管理存储占用展示、`vag storage cleanup-preview` dry-run、受控 `vag storage cleanup-execute` 执行、CLI 审计日志，以及只读 `storage-integrity` 治理视图。
- [x] 生成合并后的 P1 Asset Production Readiness CSV：`issues/next-phase-p1-asset-production-readiness.csv`，将资产库筛选、session/source 追踪、最小 provider profile 和首屏性能保护收束为下一轮主线。
- [x] 新增项目状态可视化文档：`docs/project/PROJECT_STATUS_MAP.md`，用脑图和表格说明已完成、未完成、暂缓和不做的场景。
- [x] 完成 P1 Asset Production Readiness：REST/CLI 资产列表支持 limit/status/provider/model/source/session_id/batch_id/keyword/date 筛选；Web 服务端资产库支持筛选、加载更多、图片 lazy loading 和 metadata/parameters 摘要；`metadata_json` 标准化 source/session/run/batch/story/scene/target_path；新增非敏感 project provider profile；补 Codex 批量资产生产示例。
- [x] 完成 P1 Web Performance / Startup：`initStore` 幂等、thumbnail backfill 预算、本地任务画廊渲染预算、服务端资产库节点上限、恢复轮询上限、Scope 统计缓存/边界、Markdown/Agent/Modal 懒加载均已接入；前端测试和 production build 通过。
- [x] 完成并发与生图性能专项基线：验证 `WORKER_CONCURRENCY=6` 下平台本地 worker/storage 吞吐正常；新增 task attempts API/CLI/Web 展示、openai-compatible 参数透传和 `vag benchmark image-generation`。mock 延迟压测验证 worker=6 可稳定处理 32 任务 / 128 mock 资产；真实 openai-compatible c6 小样本 6 任务中 4 成功、2 个 120s timeout，说明本机资源不是瓶颈，provider 侧 6 并发不稳定。
- [x] 完成 P1 Provider Throughput & Reliability：默认保留 `WORKER_CONCURRENCY=6`，真实 provider 默认收敛为 `OPENAI_COMPATIBLE_MAX_CONCURRENCY=3`、`FAL_MAX_CONCURRENCY=3`、`PROVIDER_TIMEOUT_SECONDS=300`；openai-compatible 增加 connect/header/total timeout profile；task attempts 新增 queue/provider/download/store/thumbnail 阶段指标；provider profile 增加 `max_n` 等非敏感 capability；`requested_count` 超过 `max_n` 时按同 task 拆分 provider 请求；benchmark 报告增强并新增 `vag batch progress`。
- [x] 完成 P1 Web Console Auth & Asset Visibility：新增轻量 Admin session、`/api/admin/assets/recent` 跨 scope 最近资产、Web Admin 登录态和未授权/空列表/筛选无结果状态区分；Web 登录后不需要手填 project API key 即可查看 MCP/CLI/REST 生成的最近资产，资产卡显示 workspace/project/campaign 并支持一键切换 scope；project API key 仍保留给外部 REST/MCP/CLI。
- [x] 生成下一阶段 P1 Project Production Context 核心 CSV：`issues/next-phase-p1-project-production-context.csv`，范围限定为 Character/Mascot Profile、Project Reference Library、Prompt Recipe / Quality Profile 2.0、Task Input Integration、最小 Web 面板和萌宠故事 mock 回归；Batch/Export/NAS 能力后置。
- [x] 完成 P1 Project Production Context 基线和契约确认：只读复核 AGENTS、docs/project、CSV 和现有源码后，确认本轮主入口为 `issues/next-phase-p1-project-production-context.csv`；第一版 Project Visual Context 复用 `project.metadata_json.visual_context`，字段覆盖 character profiles、reference bindings、prompt recipes、quality defaults 和任务快照规则，未触发新增表、迁移或破坏性接口变更。
- [x] 完成 P1 Project Production Context 服务端核心闭环 P1-PCTX-003 到 P1-PCTX-006：`project.metadata_json.visual_context` 已支持 Character/Mascot Profile、Project Reference Library、Prompt Recipe / Quality Profile 2.0 和 CreateTask visual context expansion；REST `GET/POST /visual-context`、CLI `vag project context get/set`、MCP `create_image_task` visual context 字段、`structured_input_json` / asset `parameters_json` 快照均已接入；mock smoke 验证 `task_099adb62c6c7d7cb25cb -> asset_84172190f72d61f701c2` 和 MCP `task_582c7440fd46425545cc -> asset_50d7842428660394d469`。
- [x] 完成 P1-PCTX-007 MCP REST CLI contract and examples：新增 `examples/tasks/project-visual-context-usage.md`、scene 002/003 CLI task 示例、REST create task body 示例和 JSONL-compatible MCP `tools/call` 示例；JSON parse 验证 6 个示例文件通过；mock smoke 在 `prj_pctx_smoke_1781964012 / cmp_pctx_smoke_1781964012` 下创建 CLI `task_d1bdf01c13450287d6d2`、REST `task_bf41909b09cdb53c7387`、MCP `task_7efdece0e104a504f613`，batch progress 返回 `task_count=4`、`succeeded_count=4`、`asset_count=8`；默认 project 正确要求 project API key，本轮未读取或打印任何 key。
- [x] 生成 P1 Web UX Smoothness 专项 CSV：`issues/next-phase-p1-web-ux-smoothness.csv`。子代理只读审视指出闪烁高概率来自 `settings/scope/filter` 与资产库刷新耦合、scope 分步写入、刷新时清空 assets、筛选请求无 debounce、lazy modal 空 fallback 和局部更新重绘，本 CSV 已拆分 P0/P1 修复与验收。
- [x] 完成 P1 Web UX Smoothness 第一批资产库稳定性修复：`ServerAssetLibrary` 已避免订阅整份 settings；刷新/error/scope incomplete 不再直接清空现有资产；文本筛选增加 300ms debounce 并继续用 request 序号忽略旧响应；资产卡 Scope 切换改为一次性写入必要 scope 字段。`npm --prefix web test -- --run` 通过 17 files / 224 tests，`npm --prefix web run build` 通过且仅有既有 chunk warning；本轮未运行真实 provider，浏览器自动化因工具元数据错误未完成视觉验收。
- [x] 完成 P1-UX-004 Atomic scope switching：Settings 托管 scope selector 和手动兜底 ID 输入现在只在本地 draft 中保存不完整 workspace/project/campaign，完整后才一次性提交到全局 settings；关闭 Settings 时会保留上一份完整 scope，避免把空 project/campaign 归一化为默认示例 scope；Scope 管理 modal 的“设为当前托管 scope”只写入必要的完整 scope 字段。`npm --prefix web test -- --run` 通过 17 files / 224 tests，`npm --prefix web run build` 通过且仅有既有 chunk warning；浏览器 Network/Performance 视觉回归已由 P1-UX-009 收口。
- [x] 完成 P1-UX-007 Stable lazy modal fallback and preload：`App` 的 AgentWorkspace、Detail、Lightbox、Settings、ScopeManager、MaskEditor lazy fallback 不再为 `null`，首次加载会显示稳定 overlay/skeleton；新增 lazy module preload helper，Header Settings/Scope/Agent、TaskGrid/AgentWorkspace 任务卡、InputBar 图片预览/Mask/无配置提交入口会在 hover/focus/pointerdown 时预加载对应 chunk。`npm --prefix web test -- --run` 通过 17 files / 224 tests，`npm --prefix web run build` 通过且 lazy chunks 仍拆分，仅有既有 chunk warning；`curl -I http://127.0.0.1:4173/` 返回 200；最终视觉回归已由 P1-UX-009 收口。
- [x] 完成 P1-UX-006 Scope dashboard stats background loading：Scope 管理 modal 的层级加载和 dashboard stats 现在使用独立 request id；层级列表渲染后延迟 180ms 启动统计，关闭或重新加载时取消 timer 并忽略旧统计写回；统计扫描资产列表使用明确 `limit=24`，继续保留 60s 缓存和扫描上限。`npm --prefix web test -- --run` 通过 17 files / 224 tests，`npm --prefix web run build` 通过且仅有既有 chunk warning；`git diff --check` 通过；`curl -I http://127.0.0.1:4173/` 返回 200；浏览器 Network/Performance 视觉回归已由 P1-UX-009 收口。
- [x] 完成 P1-UX-008 Task and asset card render containment：`TaskCard` 改为 `React.memo` 并只订阅 `settings.alwaysShowRetryButton`；`TaskGrid` 把每张可见卡片包进 memo 化 `TaskGridItem`，稳定 click/reuse/edit/delete handlers；`ServerAssetLibrary` 抽出 memo 化 `ServerAssetCard`，select/reject/copy/scope action callbacks 稳定，metadata/parameters 摘要在卡片内 `useMemo`。`npm --prefix web test -- --run` 通过 17 files / 224 tests，`npm --prefix web run build` 通过且仅有既有 chunk warning；`git diff --check` 通过；`curl -I http://127.0.0.1:4173/` 返回 200；浏览器 Profiler/Network 证据已由 P1-UX-009 补充。
- [x] 完成 P1-UX-009 UX smoothness regression and docs：`npm --prefix web test -- --run` 通过 17 files / 224 tests，`npm --prefix web run build` 通过且仅有既有 chunk warning，`git diff --check` 通过，`curl -I http://127.0.0.1:4173/` 返回 200；`agent-browser-cli` 回归确认 production preview root 与服务端资产库可见，Settings 打开不触发 `/api/*` 请求，Scope 管理打开主框架保持可见且首轮只观察到 `/api/workspaces`，Recent + 同步保持 root/library 可见且只有 1 个 recent 请求；当前浏览器 Recent Assets 显示明确 `unauthorized`，更符合 Admin session/API host 状态问题而非渲染空白。

## Acceptance Criteria For Next Step

- 下一步不要继续扩 P0 visibility；它已完成并在 `issues/next-phase-p0-visibility.csv` 记录 evidence。
- 合并后的 P1 Asset Production Readiness 已完成，下一步试用资产库筛选、分页、provider profile 和 batch progress。
- P1 Web Performance / Startup 已完成，日常试用优先用 production preview 判断真实资源占用；Vite dev/HMR 只用于开发。
- P1 Provider Throughput & Reliability 已完成，当前推荐默认是 `WORKER_CONCURRENCY=6`、真实 provider cap `3`、provider timeout `300s`；真实 provider 后续按 cap `2 -> 3 -> 4` 小样本复测稳定档，且必须先确认费用。
- P1 Web Console Auth & Asset Visibility 已完成，Web 控制台建议用 Admin session 查看 Recent Assets；project API key 继续用于 MCP/CLI/REST，不应放进 Web 作为日常查看资产的前置条件。
- P1 Web UX Smoothness 已完成，日常试用时若仍复现整屏闪烁，应记录具体按钮、URL host、Admin 登录状态、当前 mode/filter/scope 和是否正在刷新资产，再另起针对性 follow-up；不要把已关闭的 P1-UX-001 到 P1-UX-009 重复排期。
- 若继续执行 P2，建议先重新生成独立 CSV；涉及 provider key、公网暴露策略或真实 secret 存储的任务需先确认。
- 若当前已有线程在执行某个 P1 CSV，不要并行改同一 CSV；Web CPU 偏高问题已作为独立 P1 performance CSV 纳入下一次推进。
- 旧 P1 拆分 CSV 保留用于溯源，不再作为下一轮主入口；项目全局状态优先查看 `docs/project/PROJECT_STATUS_MAP.md`。
