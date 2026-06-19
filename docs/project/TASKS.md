# Tasks

## Todo

- [ ] 执行 P1 Asset Library Filters CSV：`issues/next-phase-p1-asset-library-filters.csv`。
- [ ] 执行 P1 Session / Source Tracking CSV：`issues/next-phase-p1-session-source-tracking.csv`。
- [ ] 执行 P1 Provider Profile / Cloud Safety CSV：`issues/next-phase-p1-provider-profile-cloud-safety.csv`。
- [ ] 下一轮可选：执行 P1 Web Performance / Startup CSV：`issues/next-phase-p1-web-performance-startup.csv`，用于治理打开 Web 时 CPU 偏高、首屏全量渲染、缩略图 backfill 和历史任务恢复轮询。
- [ ] 规划并拆分 Reference Library / Prompt Recipe / 账号主形象留存 P2：保留萌宠账号等场景的原始形象、参考图、prompt 和 edit lineage。
- [ ] 澄清嵌入式架构图场景：若需要 Mermaid/D2/SVG 可编辑源，需要补 Diagram source track；若只作为图片资产，可沿用当前资产闭环。

## Doing

- [ ] 下一步可选择执行 P1 Asset Library Filters、P1 Session / Source Tracking、P1 Provider Profile / Cloud Safety，或 P1 Web Performance / Startup。

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

## Acceptance Criteria For Next Step

- 下一步不要继续扩 P0 visibility；它已完成并在 `issues/next-phase-p0-visibility.csv` 记录 evidence。
- 后续优先执行 P1：Storage Governance 已完成；下一步可从资产库筛选、session/source tracking、project provider profile、云端安全或 Web 性能专项中选一个 CSV 继续。
- 若继续执行实现，选择一个 P1 CSV，用 `/goal @issues/<next>.csv` 逐条推进；涉及删除、provider key 或公网暴露策略的任务需先确认。
- 若当前已有线程在执行某个 P1 CSV，不要并行改同一 CSV；Web CPU 偏高问题已作为独立 P1 performance CSV 纳入下一次推进。
