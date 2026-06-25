# Decisions

## 2026-06-25: 单资产清理采用 Admin 归档/恢复，默认不物理删除 archived

- Decision: 单 asset 第一轮采用 Admin-only `archive/restore` 语义：归档写入现有 `deprecated` 存储状态，对外显示为 `archived`；恢复回 `generated`。`storage cleanup` 默认不再包含 archived/deprecated，只有显式 `--deprecated` 或 `include_deprecated=true` 才会物理清理归档资产。MCP 继续不提供删除、清库、workspace/project/campaign/asset destructive tools。
- Reason: 用户需要把废图从人工审图主路径移走，但不希望一次误点就物理删除；同时 agent 接入 MCP 的语义应保持“生产、查询、选择/拒绝、拿交付”，不应天然拥有破坏性数据生命周期权限。
- Impact: 删除/归档/恢复继续走 Admin Web / Admin REST / CLI，写 audit；Project API Key 仍服务外部 agent 生图和查资产，不代表清理权限。后续如果需要 task/input-file reset 或 MCP 低风险 archive tool，必须单独确认权限、dry-run、审计和误删保护。

## 2026-06-25: Runtime 诊断只暴露非敏感 build/provider/auth 摘要

- Decision: Admin runtime status 可以暴露 API/Web version、commit、image tag、provider mode/model、Admin/Basic 是否配置等非敏感状态，用于定位本地/服务器/Web 镜像不一致；缺失 build 信息显示 `unknown`。接口和 Web 不返回 provider key、project key、Basic/Auth 密码、cookie、session、本地敏感路径或完整 secret。
- Reason: 自托管部署后，用户高频遇到“我连的是哪个服务、为什么本地和服务器表现不同、为什么登录/缩略图/Provider 状态不一致”的问题；非敏感运行诊断比让用户手动看容器 env 或日志更安全。
- Impact: GitHub Actions / Docker image 会注入 build metadata；Web 控制台显示当前 Agent ImageFlow Server 状态，并提示 API/Web commit mismatch。该能力不改变账号体系，也不引入每用户 provider key。

## 2026-06-23: Scope 删除采用 Admin 受控级联删除语义

- Decision: 追加 `issues/next-phase-p1-scope-management-usability-followup.csv`。Scope 管理中的 workspace/project/campaign 删除后续应支持非空级联删除：删除 campaign 会删除其 task/attempt/asset/version/review/input-files/storage；删除 project/workspace 会递归删除所有下级 campaign/project。用户已确认该语义包含 selected/approved/published 资产。该能力仍只走 Admin Web/REST/CLI 受控链路，不向 MCP 暴露 workspace/project/campaign destructive tools。
- Reason: 真实 Web 试用中，empty-only 删除要求用户先手动清理子级 project/campaign/task/asset，和“删除这个测试空间/业务空间”的直觉不一致，也不适合持续试用产生的数据清理。Storage cleanup 默认保护 selected/published/approved 适合资产级清理；但用户明确删除整个 scope 时，应该按 scope 生命周期处理。
- Impact: 后续实现必须提供明确二次确认文案，说明会删除子级、任务、资产、文件和 selected/approved/published 结果；操作必须写 audit，并保持未授权不可删除。MCP Service Pack、RUNBOOK 和数据生命周期文档必须继续说明：agent 通过 MCP 不能硬删除空间，删除/重置走 Admin 受控入口。
- Implementation update: 已实现 Admin 受控级联删除的后端与 Web 确认文案。Postgres 显式删除 scope 下 `delivery_event`、`review_event`、`asset_version`、`asset`、`task_attempt`、`generation_task` 及下级 scope 记录，Service 层继续删除对应 storage scope 目录；ConfirmDialog 层级、ScopeManager 滚动和 InputBar `@` 误导文案也已修复。MCP 仍保持 6 个非破坏性工具，删除/重置继续走 Admin Web/REST/CLI。

## 2026-06-23: V1 稳定性以服务器/NAS 演练证据作为验收门槛

- Decision: 新增 `issues/next-phase-p1-server-deployment-rehearsal.csv` 和 `docs/project/stories/slice-053-server-deployment-rehearsal.md`，把服务器/NAS 部署演练作为 V1 baseline 之后的第一优先验收门槛。只有完成真实目标环境中的 GHCR pull、prod compose first boot、HTTPS 同源入口、Admin/Web mock task、MCP smoke、Postgres + storage 备份恢复和 `IMAGE_TAG` 回滚后，才把 V1 视为适合长期自托管运行。
- Reason: `slice-052` 已证明发布流水线和生产 compose 存在，但“部署文件存在”不等于“目标服务器可恢复、可回滚、可连续运行”。V1 当前最主要风险是运维可靠性，而不是缺新功能。
- Impact: 后续默认先执行 server deployment rehearsal，再进入 pet account real workflow trial。演练默认只跑 mock，不运行真实 provider；1 图真实 provider canary 必须另行确认费用。演练证据只能记录非敏感状态、task/asset id、服务健康和回滚结果，不记录 `.env.prod`、GHCR token、provider key、project key、Basic/Auth/Admin cookie 或 session。

## 2026-06-22: Cleanup REST 采用 Admin session only

- Decision: `storage-cleanup-preview` 和 `storage-cleanup-execute` 第一版只允许 Admin session 访问；即使实例开启 Basic Auth，或请求带有 Project API Key，也不能把它们当作清理/重置权限使用。CLI 仍作为本机/服务器运维入口，MCP 不新增 destructive tools。
- Reason: Project API Key 的语义是让 agent / 自动化系统创建任务、查资产和拿交付链接，不应天然拥有删除数据、重置试用环境或释放存储文件的权限。Basic Auth 只是实例级入口保护，也不等于已进入 Admin 控制台。把清理能力收束到 Admin session 可以降低误删和远程 agent 误操作风险。
- Impact: Web/REST 数据生命周期能力必须走 Admin 登录态；未登录或只带 Basic Auth 的浏览器/脚本会得到 `admin_session_required`，且不会触发浏览器原生 Basic Auth 弹窗。文档、MCP Service Pack 和 RUNBOOK 都必须继续说明：删除和试用重置走 Admin Web/REST/CLI，不把 Admin cookie、cleanup token、provider key 或真实 project key 写入 MCP 示例。

## 2026-06-22: Character Reference Intake 成为 P1 角色一致性补强切片

- Decision: 追加 `issues/next-phase-p1-character-reference-intake-consistency.csv`，把“角色卡图像入库、绑定、参与生成和可视化验收”作为独立 P1 切片，而不是继续把它归入已经完成的 Project Visual Context 第一版。
- Reason: 真实试用证明当前只能算 MCP 文生图链路跑通：角色卡只有文字描述，没有 `primary_asset_id` / `reference_asset_ids`；用户上传和裁切的图停留在 campaign `input-files`，没有沉淀为正式 asset，也没有绑定进 Project Visual Context；绕过参考图后的成功任务不能证明角色一致性。
- Impact: 后续验收必须区分三层结果：平台链路成功、参考图参与成功、角色一致性人工判断通过。第一轮补 input-file promote、character primary/reference asset binding、Web 角色卡缩略图、MCP 接入边界、provider reference canary/诊断和 pet character consistency smoke；不做通用 DAM、AI 自动视觉质检或 provider key 下发。

## 2026-06-22: Web Review Feedback and Stability 作为真实试用 UX follow-up

- Decision: 追加 `issues/next-phase-p1-web-review-feedback-stability.csv`，专门处理 Select/Reject 状态反馈不明显、Production View 局部状态不清晰、下拉菜单切换造成整页闪烁和请求风暴的问题。
- Reason: P1 Web UX Smoothness 已完成一轮启动和刷新稳定性修复，但真实使用中仍复现两个高频摩擦：审图动作只有 toast 不足以让用户确认状态，scope/filter/recipe 等下拉变更仍可能触发大范围重绘或空态闪烁。
- Impact: 后续 Web 验收必须包含可见状态变化、optimistic update、失败回滚、局部 summary 更新、下拉切换保留旧内容、请求去重/节流和 browser regression evidence。该切片不重新设计 Settings 信息架构，不引入新设计系统，不运行真实 provider。

## 2026-06-22: Safe Delete and Trial Reset 作为后续 P1 数据生命周期切片

- Decision: 追加 `issues/next-phase-p1-safe-delete-and-trial-reset.csv`，目标是补受控删除、归档和试用重置能力，解决本地/服务器试用中 task、asset、batch、campaign 只增不减的问题。
- Reason: 当前 MCP 只暴露 `create_image_task`、`get_image_task`、`list_image_assets`、`select_image_asset`、`reject_image_asset` 和 `get_asset_delivery_info`，没有删除类 tool；现有 scope delete 更适合空 scope，Storage Governance 更偏底层清理。用户真实试用时需要按废图、失败任务、batch/session/campaign 安全清理，而不是重置整个 Docker volume。
- Impact: 第一轮删除能力优先放在 Admin Web/REST/CLI，采用 dry-run、二次确认、selected/published/approved 保护、审计和恢复说明；MCP 不开放 workspace/project/campaign 硬删除。若未来要给 MCP 增加删除能力，只考虑低风险 archive/reject 类动作，并单独确认 schema、权限和误删保护。

## 2026-06-22: MCP Service Pack 作为后续 P1 agent onboarding 小切片

- Decision: 在 Settings 信息架构之前或并行追加 `issues/next-phase-p1-mcp-service-pack.csv`，目标是把现有 MCP 能力整理成低成本接入服务包，而不是扩展新的 MCP 协议能力。
- Reason: 后续用户会频繁新开线程或让不同 agent 调 Agent ImageFlow 生图。若每个 agent 都要理解完整项目、scope、key、tools、参数和返回值，接入成本太高；一份 guide + config + pet scene 示例 + mock smoke 可以用很小改动换来很高复用收益。
- Impact: P1-MCP-SVC-001 只新增文档和示例，明确 Project API Key、Basic Auth、Admin Login、provider key 的边界；不做远程 HTTP MCP、新账号系统、多用户权限、provider key 下发、tool schema 大改或真实 provider 默认 smoke。

## 2026-06-22: Web 控制台采用前置 Admin 登录页

- Decision: Web 控制台默认是服务器托管的图片资产生产平台入口。浏览器未通过 Admin session 时，只显示全局登录页，不渲染 Header、InputBar、资产库、Production View、Project Context 或 Settings 主体；登录后用户使用服务器环境变量中配置好的 provider 能力，退出后回到登录页。
- Reason: 用户明确希望把原先藏在资产库局部的登录前置化，避免未登录用户误以为 Web 是“每个浏览器用户自带 provider key/base URL 的生图前端”。这也降低了公网部署时误操作旧 provider 路径、看到无意义长 ID 或误解凭据边界的风险。
- Impact: `Admin Login` 是进入 Web 控制台的门；`Project API Key` 继续服务 MCP/CLI/REST 外部 project 调用；`Basic Auth` 继续作为实例级保护；`provider key/base URL` 继续只属于服务器环境变量或受控服务端配置。Settings 信息架构仍需后续独立设计，本轮只做 server-first 文案收敛和主路径中文化。

## 2026-06-22: 生产 Web 入口采用同源 Admin delivery

- Decision: 服务器部署后的正式浏览器入口采用 Web 同源模式：Web 镜像代理 `/api/*` 与 `/healthz` 到内部 API，`PUBLIC_BASE_URL` 指向 Web/HTTPS 公开 origin；Admin session cookie 可授权 asset thumbnail/original/metadata 读取。实例级 Basic Auth 继续作为外部 API/运维保护，但图片 delivery 的未授权响应不返回浏览器原生 Basic challenge。
- Reason: 公网部署后 Web 与 API 分端口会形成不同 origin，`<img>` 无法携带 Web settings 里的 Authorization header，导致缩略图 401 和浏览器弹出 `http://host:api_port` 登录框。用户需要的是登录 Web 后直接审图，而不是在图片资源请求上处理 Basic Auth。
- Impact: Web 默认 Agent ImageFlow API URL 改为浏览器当前 origin 优先，本地非浏览器环境回退 `http://localhost:8081`；新增 Admin runtime status 只读接口和控制台安全摘要；资产库增加 Current Scope 导航并把 Project Context 提升为 project 工作区入口。Basic Auth、Admin Login、Project API Key、provider key 继续是四种不同概念，provider key 不进入前端 bundle、localStorage、响应 JSON 或日志。

## 2026-06-22: 当前 main 作为 V1 baseline，后续只按独立 CSV 继续推进

- Decision: 当前 `main` 可暂定为 Agent ImageFlow V1 baseline。V1 已包含核心资产生产、Project Visual Context、Batch/Story/Scene Production View、Web Operator Review Console、JSON manifest、NAS/Docker 文件访问边界和 GHCR 发布流水线。后续不再复开已完成的 P1/P2 CSV，而是按 `docs/project/V1_BASELINE_AND_ROADMAP.md` 重新拆独立 CSV。
- Reason: 项目已经从“能跑通”进入“可部署、可试用、可运营维护”的阶段。继续在旧 CSV 上追加会增加回归风险和范围混乱；按部署演练、真实试用、usage/edit lineage、export pack、deployment secret hardening 分开推进更容易验收。
- Impact: 服务器/NAS 部署演练、真实萌宠账号试用和后续 P2 能力必须单独确认范围、成本、provider 调用和验收标准。小红书发布、内容日历、通用 DAM、SaaS 注册计费、每用户 provider key、视觉质检 AI 和无目标 benchmark 继续后置。

## 2026-06-22: 生产发布采用 GHCR 私有镜像 + 服务器拉取运行

- Decision: Agent ImageFlow 的正式自托管发布流采用 GitHub Actions 构建并推送 GHCR 私有镜像，服务器只通过 `docker compose -f docker-compose.prod.yml --env-file .env.prod pull/up` 拉取和运行镜像，不在服务器构建 Go 或 Web。后端镜像名为 `ghcr.io/billionsheep/agent-imageflow-api`，Web 镜像名为 `ghcr.io/billionsheep/agent-imageflow-web`；版本使用 `main`、`vX.Y.Z` 和 `sha-<short_sha>`。
- Reason: 当前产品已经进入可自托管试用阶段，本地源码 build 适合开发但不适合长期服务器部署。镜像发布流能让上线、回滚、审计和服务器环境保持稳定，同时避免把本地 dirty worktree、构建工具链或 provider key 带到服务器。
- Impact: 新增 `docker-compose.prod.yml`、`Dockerfile.web`、`docker/nginx-web.conf`、`.github/workflows/docker-publish.yml`、`.env.example.prod`、`scripts/check_deployment_release.py` 和 `docs/project/SERVER_DEPLOYMENT_GUIDE.md`。`docker-compose.yml` 保持开发模式；`.env.prod` 只放服务器且继续被忽略；Postgres/Redis/storage root 不直接公网暴露；HTTPS/证书/反向代理由部署环境负责。后续任何包含数据库 schema 变化的版本，必须另开 migration/backup 计划。

## 2026-06-22: 真实 provider 验证默认采用低频 canary

- Decision: 后续真实 provider 验证默认采用低频 canary：独立 scope、少量图片、明确入口、明确停止条件；不把真实 provider benchmark 当作日常验证手段。MCP/REST/Web/CLI 入口是否真实走 provider，可通过 1 图 canary、batch progress、task attempts、metadata、thumbnail/original URL 和 Recent Assets 可见性来验收。
- Reason: 项目已经具备 mock benchmark、provider stage metrics 和 batch progress，真实 provider 的主要风险从“平台是否能并发”转为“费用、外部限流和单次调用可靠性”。用户当前目标是打通萌宠账号资产生产工作流，而不是压测 provider；低频 canary 能证明真实链路不断，又避免费用失控。
- Impact: 本轮已用 MCP `create_image_task` 执行 1 图真实 canary，生成 `task_b5256922a91e424850d3 -> asset_7c706c1a1cea00490a40`，provider 为 `openai-compatible / gpt-image-2`。后续如需真实 benchmark，必须另行确认费用预算、并发、样本量、provider cap 和中止条件；不得读取、打印或迁移 provider key/API key/secret/cookie/session。

## 2026-06-22: P2 Web Operator Review Console 采用 server-first 审图语义

- Decision: 下一阶段 P2 主线命名为 Web Operator Review Console，默认服务人工审图与交付确认；Recent Assets 和 Production View 默认展示图片、prompt/画面描述、story/scene、状态和核心动作，asset/task/scope/provider/hash/session/batch 等工程字段放入折叠 Technical details。真实 provider key/base URL 继续属于服务端平台配置；Admin 登录者使用平台能力；Project API key 继续服务 MCP/CLI/REST 外部调用；旧版浏览器直连 provider profile 保留为 advanced/legacy 兼容路径。
- Reason: P1 已证明 Project Visual Context、batch/story/scene 聚合、scene action/regenerate 和 JSON manifest 闭环可用，但 Web 默认信息层仍像调试面板，用户在萌宠账号图片资产生产中只需要快速看图、理解剧情、选择/拒绝和导出。若不先收束凭据语义和 UI 信息层，后续功能越多越难用，也容易误导为“每个登录用户自带 provider key”的账号体系。
- Impact: 已新增并关闭 `issues/next-phase-p2-web-operator-review-console.csv`，实现记录写入 `docs/project/stories/slice-051-web-operator-review-console.md`；本轮只做 Web 操作体验和最小安全回归，不引入多用户账号、注册、多租户、RBAC、发布系统、DAM、服务端 ZIP、WebDAV/SMB server 或真实视觉质检 AI；不读取、打印、迁移或处理任何真实 key/secret/cookie/session。

## 2026-06-20: Project Visual Context 第一版复用 project.metadata_json

- Decision: P1 Project Production Context 第一版不新增数据库表或迁移，统一使用 `project.metadata_json.visual_context` 保存 Character/Mascot Profile、Project Reference Library binding 和 Prompt Recipe；`CreateTask` 新增 `character_ids`、`reference_asset_ids`、`prompt_recipe_id`、`use_project_visual_context`，在入队前展开为 `structured_input_json.visual_context_snapshot`，并进入 asset `parameters_json`。
- Reason: 当前目标是让单个 project 承载长期视觉生产上下文，而不是建设通用 DAM、模板市场、账号运营系统或复杂权限体系。复用既有 metadata_json 能与 `quality_profile`、`provider_profile`、`access_config` 的策略保持一致，避免为了第一版上下文引入高风险 schema 迁移。
- Impact: REST 新增 `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/visual-context`，CLI 新增 `vag project context get/set`，MCP `create_image_task` 可传 visual context 引用字段。角色卡和 reference binding 只能引用同 workspace/project 下的 asset；删除或归档 binding 不删除原 asset；provider secret 仍只走服务端环境变量，不进入 visual context、响应 JSON 或日志。

## 2026-06-20: Web 控制台采用轻量 Admin session + Recent Assets

- Decision: Web 控制台新增轻量 Admin session，提供 `login/me/logout` 和跨 scope `Recent Assets` 读取路径；Web 登录后可查看 MCP/CLI/REST/Web 生成的最近资产，不再把 project API key 当作人工查看资产的前置条件。Project API key 继续保留给 MCP、CLI、REST 等外部 project 级调用。
- Reason: 项目已经具备 project API key 和 scope 资产列表，但 Web 资产库仍容易困在当前 scope 或 401 状态，用户忘记 scope 或不想手填 key 时很难发现外部 agent 生成的资产。当前目标是自托管小团队/单人控制台，不需要完整账号系统、SaaS 注册、租户、RBAC 或 OAuth。
- Impact: 新增 Admin cookie session 和 `/api/admin/assets/recent`，Web 服务端资产库默认走 Recent Assets 并区分 unauthorized、空列表、筛选无结果和加载错误。Provider key 仍只在服务端环境变量中，不能进入前端 bundle、localStorage、响应 JSON 或日志；生产公网暴露仍需反向代理/TLS、强 Admin 密码、Basic/project key、限流和审计。

## 2026-06-18: MVP 后续阶段转向资产库治理和真实场景验证

- Decision: 当前 MVP 产品闭环不再继续盲目扩功能，后续先围绕真实场景试用收集需求，并把下一阶段聚焦到 Web 服务端资产同步、资产库治理、存储可视化、session/source tracking、参考图/prompt 留存、云端安全和对外开通路径。新增 `docs/project/FUTURE_REQUIREMENTS_AND_SCENARIOS.md` 作为后续拆分 CSV / vertical slices 的输入。
- Reason: 实测 MCP 可以创建并交付服务端资产，但 Web 不会自动显示 MCP / REST / CLI 创建的任务；同时用户明确提出萌宠小红书账号、嵌入式架构图账号、存储可视化、过期策略、项目/会话隔离、prompt 和原始形象留存等真实使用需求。继续零散加功能会造成资产库、治理、安全和业务空间模型遗漏。
- Impact: 第一版 MVP 边界不变；下一阶段优先从“生成能力”转向“资产库可见、可治理、可隔离、可追踪”。嵌入式架构图场景如果要求 Mermaid / D2 / SVG 可编辑源，需要作为 Diagram source track 单独确认，不能默认混入生图 MVP。

## 2026-06-18: 项目级 access-config 在不改路由的前提下扩展为多 key 策略

- Decision: 保持 `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/access-config` 路由不变，在 `project.metadata_json.access_config` 中加入 `api_keys` 列表；顶层 `api_key_enabled` / `api_key_name` / `api_key_preview` / `api_key_hash` 继续作为兼容视图保留。`POST` 通过 `action=add_key|update_key|delete_key` 和 `api_key_id` 完成最小多 key 管理动作。
- Reason: 当前 MVP 已经具备项目级鉴权，但只有单把活动 key，无法支撑低风险轮换、双写切换或按系统分发凭据。项目规则又不希望为了这一片引入数据库迁移、复杂 secret manager 或新的管理页面。
- Impact: REST、CLI 和审计层现在都能识别多把命名 key；HTTP API 对项目级资源接受任意一把启用 key，并会在审计里优先记录命中的 key 名称。MVP 范围内的生产 hardening 缺口由此清零。

## 2026-06-18: HTTP / API 审计日志第一版采用本地 JSONL + CLI 查询

- Decision: HTTP / API 第一版结构化审计日志仅挂在 `api` 入口，针对 `/api/*` 请求写入 `STORAGE_ROOT/audit/http-api/YYYY-MM-DD.jsonl`；同时提供本地 `vag audit list` 查询入口，支持按 project / task / asset / status 等条件过滤。审计写入异常时记录日志并 fail-open，不把审计故障升级为请求失败。
- Reason: 当前项目已经进入生产 hardening 阶段，需要先补一层“谁在什么时候对哪个资源做了什么”的可追踪性；但项目规则不希望这一片直接引入数据库迁移、Web 审计页、远程日志系统或更复杂 retention 治理。
- Impact: `api` 进程会为常见 HTTP 请求记录 method、route、action、status、duration、auth mode、actor 和关键 scope/resource id；运维可直接用 `vag audit list` 做本地排查。当前主要剩余 hardening 缺口从“审计、多 key 策略”收敛为“多 key 策略”。

## 2026-06-18: HTTP API 第一版限流采用 Redis 固定窗口并 fail-open

- Decision: HTTP API 第一版限流仅挂在 `api` 入口，复用现有 Redis 做固定窗口计数；同时支持实例级和 project 级阈值，命中时返回 `429`、结构化错误 JSON 和 `Retry-After`。Redis 限流后端异常时记录日志并 fail-open，不把限流组件故障升级为整站不可用。
- Reason: 当前项目已经进入自托管 / 对公网暴露前的生产 hardening 阶段，需要先补一层基础放大保护；但本片不适合引入新的数据库表、复杂配额系统、IP 级策略或多节点治理。
- Impact: 配置面增加 `RATE_LIMIT_WINDOW_SECONDS`、`RATE_LIMIT_INSTANCE_MAX_REQUESTS`、`RATE_LIMIT_PROJECT_MAX_REQUESTS`；MCP stdio、Worker 和服务端核心状态机不受直接影响。当前主要剩余 hardening 缺口从“基础限流”转移为“审计、多 key 策略”。

## 2026-06-18: Best-of 第三版加入可选 auto reject，但保留人工改选

- Decision: 在已存在的 `best_of_config` 上增加 `auto_reject_non_selected` 开关；当 `selection_mode=auto` / `best_of` 自动选出推荐图后，可由服务端自动把其他候选标记为 rejected。同时保留 `rejected -> approved` 的人工 override 路径，不让 auto reject 变成不可逆闸门。
- Reason: 小团队/单体平台的实际摩擦已经从“怎么选出推荐图”转向“怎么自动清理未入选候选状态”。如果只做 auto reject 而不保留人工改选，则会把轻量状态流重新拉回强审核心智，不符合 MVP 定位。
- Impact: `best_of_config` 现在同时表达 scorer 和 auto reject 策略；服务端会事务式应用 `selected + rejected siblings`，当前主要剩余缺口从“自动 reject 非推荐候选”转移为“限流、审计、多 key 策略”等生产 hardening。

## 2026-06-18: Best-of 第二版采用可插拔 scorer + HTTP judge adapter

- Decision: 在不改变当前 `generated -> selected/rejected` 轻量状态模型的前提下，给任务输入和项目级 quality profile 新增 `best_of_config`；服务端 best-of scorer 采用可注册策略，默认保留 `local_metadata_v1`，并新增可选 `http_judge_v1` 通过外部 HTTP judge 做视觉/LLM 打分。
- Reason: 当前需要把自动选优从写死的本地元数据启发式升级为可扩展的视觉/LLM judge 能力，但又不希望把 MVP 绑定到某一家付费模型、引入新数据库迁移，或让外部评分成为自动选优的单点故障。
- Impact: REST/MCP/Web 托管入口与项目级 quality profile 都可表达 scorer 选择；`http_judge_v1` 会消费服务端缩略图 data URL 和结构化任务信息，失败时自动回退到 `local_metadata_v1`。MVP 主要剩余缺口从“可插拔评分”转移为“生产 hardening”。

## 2026-06-18: fal.ai provider adapter 采用 queue + rest storage HTTP 协议，并复用统一 resolved input 链路

- Decision: 服务端 `provider=fal` 采用 fal queue + rest storage 的标准 HTTP 协议实现，不引入 Go SDK 新依赖；继续直接消费 `CreateTask` 统一生成的 `resolved_input_files`，不新增第二套 provider 专用输入协议或状态机。
- Reason: 当前已经有 OpenAI-compatible 的真实输入复用闭环，下一步需要把同一套 `input-files`、匿名 remote URL 和当前项目 `asset_id` 扩到第二条真实 provider 路径；保持 HTTP 实现更透明，也更容易用本地 mock 做集成 smoke。
- Impact: `provider=fal` 现已支持 queue 文生图，以及基于 remote URL / `asset_id` / `input_file_id` 的 edit 输入复用；MVP 的主要剩余缺口从“更多 provider 输入复用”转移为“生产 hardening”。

## 2026-06-18: Remote URL 物化到 scope input-files，asset reuse 限定在同 workspace/project

- Decision: 服务端在创建任务时支持三类 edit/mask 输入来源: scope `input-files`、匿名 `http/https` 远程 URL、当前 workspace/project 下已有 `asset_id`。远程 URL 会在创建任务时抓取并物化到当前 scope 的 `input-files`；`asset_id` 复用限定在同 workspace/project，不新增数据库表或第二套输入状态机。
- Reason: 当前需要把“真实 provider edit/mask”从只支持上传文件扩展到更实用的复用路径，同时项目规则不希望在这一片引入新的输入索引表、缓存表或跨项目资产引用复杂度。
- Impact: `CreateTask` 现在可以统一把三类来源解析为 `resolved_input_files`，OpenAI-compatible 继续消费同一条 `/images/edits` 路径；任务快照会保留远程 `url`、生成的 `input_file_id` 和原始 `asset_id`，后续更多 provider 应复用这条输入解析链路。

## 2026-06-18: Scope 管理第二版进入独立 modal，并先复用 metadata 存 archive 状态

- Decision: Web scope 管理从“仅设置页内联同步/新建”升级为独立 modal；workspace / project / campaign 的 archive 状态先写入各自 `metadata_json`，不新增数据库迁移；delete 只允许删除空 scope。
- Reason: 现有设置页足够完成托管模式的“查 + 选 + 新建”，但当用户需要持续维护业务空间时，缺少独立管理入口和 rename/archive/delete 会明显拖慢日常使用；同时当前项目规则不希望为了这一片引入新的 scope 状态表或复杂迁移。
- Impact: REST 新增 workspace/project/campaign 的 `PATCH` / `DELETE`；Web 顶栏可直接打开独立 scope 管理入口；设置页 selector 默认过滤 archived scope；scope 管理仍保持实例级 Basic Auth 管理能力，project API key 不参与这一层。

## 2026-06-18: 项目定位为 Agent ImageFlow

- Decision: 项目定位从 DiagramOps / 技术图示网关调整为 Agent ImageFlow / AI 图片资产生成与管理平台。
- Reason: 用户明确表示不希望第一版重点做 SVG、Mermaid 或技术图示，因为 Codex 已能完成部分图示生成；更想解决生图自动化、海报设计、小说配图和其他图片资产场景。
- Impact: 第一版优先验证生图、落盘、轻量选优、复用、MCP/API 调用闭环；暂不优先做技术图示 DSL。

## 2026-06-18: 降级人工审核为轻量选优/状态标记

- Decision: 第一版不把“每张图片必须人工审核通过”作为默认流程，改为 `generated -> selected/rejected/published` 的轻量选优和状态标记。当前代码中的 `draft/approved`、`approve/reject` 保留为兼容命名，产品语义上分别映射为 `generated/selected`、`select/reject`。
- Reason: 项目当前面向单体平台或小团队，强人工审核会增加使用成本；质量优先通过 prompt 优化、style preset、参考图、模板复用和多候选 best-of 逻辑保证。
- Impact: 后续 MCP、Web managed mode 和计划文档优先使用 `select_image_asset`、候选图选优视图和自动选优策略；强审核只作为未来项目级可选策略，不进入 MVP 默认路径。

## 2026-06-18: 当前阶段只写项目文档，不实现代码

- Decision: 初始化项目上下文和产品规格文档，暂不创建应用代码。
- Reason: 产品能力和定位还要再次规格定义，过早实现容易固化错误边界。
- Impact: 后续需要产品/MVP lock 后再进入实现。

## 2026-06-18: 能力平台而非网页生图工具

- Decision: 产品应作为 MCP/API/CLI 能力平台，而不是只面向人工点击的网页工具。
- Reason: 核心痛点是 AI 和自动化系统无法稳定拿到图片资产句柄、元数据、候选图状态和交付路径。
- Impact: 未来接口设计优先考虑结构化输入输出、任务状态和资产登记。

## 2026-06-18: 冻结输入与输出 v0.1

- Decision: 第一版输入固定为 MCP、REST API、CLI、Web UI 四类；输出固定为任务结果、资产身份、原图文件、缩略图、metadata JSON、交付信息六类。
- Reason: 产品需要先稳定契约，避免继续扩散到过多调用方式和输出形态。
- Impact: 后续接口、MCP tools、CLI 和 UI 都围绕 `ImageTask`、`Asset`、原图、缩略图、metadata 展开。

## 2026-06-18: 冻结业务隔离模型 v0.1

- Decision: 第一版采用 `Workspace -> Project -> Campaign -> ImageTask -> Asset` 的业务隔离模型。
- Reason: 小红书账号、小说配图、海报活动、技术博客等业务不能混在同一个资产池里；同时项目不应过早扩展成完整 DAM。
- Impact: 任务和资产必须带 `workspace_id`、`project_id`、`campaign_id`；未来业务能力通过 metadata、adapter 和模块扩展，而不是第一版写死所有业务字段。

## 2026-06-18: 选择内容系统批量封面图作为核心业务流程

- Decision: 第一版核心业务流程选定为“内容系统批量生成封面图”，demo 方向使用小红书/内容账号 campaign 素材生产。
- Reason: 这个流程最能验证批量任务、业务隔离、候选图、缩略图、选优、文件获取和 metadata 交付，同时比小说角色一致性和电商商品海报更轻。
- Impact: 首个 vertical slice 围绕内容账号 project、7 天封面图 campaign、批量 ImageTask、候选 Asset 和选优交付展开。

## 2026-06-18: 冻结第一版架构方向

- Decision: 第一版采用“小核心 + 多入口 + 多适配器 + 大存储”的模块化单体架构。入口层包含 MCP、REST API、CLI、Web UI；核心层统一处理 Workspace / Project / Campaign / ImageTask / Asset；Worker 异步调用 provider；资产处理层负责原图、缩略图、hash 和 metadata；存储层优先本地大磁盘，后续扩展 MinIO/S3。
- Reason: 这个架构能支撑 AI/自动化调用和未来业务扩展，同时避免过早拆微服务或做成纯网页生图工具。
- Impact: 实现阶段应优先打通 API/Worker/Postgres/Redis/File Storage 的核心管线；Provider、Storage、Delivery 都以 adapter 方式设计。

## 2026-06-18: 合并架构评审为最终架构指导

- Decision: 将 `ARCHITECTURE_REVIEW.md` 中的状态模型、幂等重试、一致性边界、文件访问隔离、provider 失败模型、可观测性和演进触发条件合并进 `ARCHITECTURE.md`，并把 `ARCHITECTURE.md` 作为后续实现主准绳。
- Reason: 原架构方向正确，但如果缺少任务/资产/版本状态拆分、重复消费处理、文件与数据库一致性和 provider 失败结构化，第一版很容易退化成不可靠的生图 API wrapper。
- Impact: 实现阶段必须优先验证 mock provider 全链路，并在首个 vertical slice 中纳入 `idempotency_key`、`task_attempt`、`AssetVersion.status`、归属校验和结构化错误；`ARCHITECTURE_REVIEW.md` 保留为评审输入，不再作为并列实现规范。

## 2026-06-18: 锁定第一阶段实施栈和部署方式

- Decision: 第一阶段采用 Go + PostgreSQL + Redis + 本地文件系统 + Docker Compose；先实现 mock provider 闭环，后续再接一个云端 API provider；MVP 不考虑本地 GPU 或 ComfyUI。
- Reason: 用户明确希望只用 API、不考虑 GPU；Go 适合 API、Worker、CLI、MCP 共享 domain code；Docker Compose 能覆盖本地开发和第一版自托管部署。
- Impact: 下一步直接进入实施骨架：Docker Compose、Go API、Go Worker、CLI smoke test、PostgreSQL schema、Redis queue、local storage 和 mock provider。

## 2026-06-18: 回退低保真 Web，改为基于 GPT Image Playground 二开

- Decision: 撤回自写低保真 Web/API/Worker 实现，当前前端改为直接导入 `/Users/moon/Workspace/tools/gpt_image_playground` 到 `web/` 并二开。
- Reason: 用户明确反馈自写 Web 质量不足，且参考项目已经具备成熟的生图工作台、设置页、Base URL/API Key、多 provider、画廊、参考图、遮罩和 Agent 模式。
- Impact: 第一实现步骤改为先稳定 Web 底座，再把 Agent ImageFlow 的服务端资产登记、轻量选优、MCP 和交付模型接入进去；原架构方向保留，但实施顺序调整为 Web-first。

## 2026-06-18: 完成第一条服务端 mock 资产闭环

- Decision: 在当前 Web 底座之外新增 Go API、Worker、CLI、PostgreSQL、Redis、本地文件系统和 Docker Compose 骨架，并用 mock provider 跑通 `ImageTask -> Asset -> AssetVersion -> 状态事件 -> DeliveryInfo`。
- Reason: 产品核心价值必须来自稳定 `task_id`、`asset_id`、落盘文件、metadata、候选图状态和交付 URL，而不是浏览器端临时图片结果。
- Impact: 第一版已有 REST/CLI smoke 能力；Web 后续应通过新增的服务端 API client 进入托管模式。MCP、真实云端 provider、MinIO/S3、权限计费仍保持 out of scope。
- Implementation note: Go 依赖 `pgx` 和 `go-redis` 作为 PostgreSQL/Redis 驱动；API 默认监听 `http://localhost:8081`；Docker volume 挂载到 `/data/agent-imageflow`。

## 2026-06-18: Web 和服务端最终收敛到同一资产核心

- Decision: `web/`、MCP、REST API 和 CLI 最终都应作为入口调用同一个服务端 application core；不长期维护“浏览器直连 provider”和“服务端 Worker provider”两套正式生图系统。
- Reason: Agent ImageFlow 的产品定义是可追踪、可选优、可交付的图片资产平台。浏览器直连 provider 可以提供成熟交互和迁移经验，但不能作为 MCP/自动化系统的正式事实源。
- Impact: MCP stdio server 已补齐；后续优先迁移服务端真实 provider adapter，再进入 Web 托管模式。原 Web 的 OpenAI-compatible、fal.ai、自定义 HTTP provider 逻辑作为服务端 provider adapter 的参考来源。

## 2026-06-18: MCP stdio 入口复用现有服务端核心

- Decision: 新增 Go `cmd/mcp` 和 `internal/mcp`，用标准库实现 MCP stdio JSON-RPC 薄封装，不新增 MCP SDK 依赖。MCP tools 直接调用现有 `app.Service`。
- Reason: 当前 slice 只需要本地 stdio、tools/list 和 tools/call；标准库实现能减少依赖和迁移成本，并确保 MCP、REST、CLI 共用同一套任务/资产状态机。
- Impact: Docker image 新增 `/app/mcp`；MCP 对外暴露 `generated/selected` 语义，底层 `draft/approved` 继续作为兼容命名保留。下一步可以在不改 MCP 协议层的情况下迁移真实 provider adapter。

## 2026-06-18: 第一版真实 provider 采用 OpenAI-compatible adapter

- Decision: 服务端新增 `provider=openai-compatible`，通过 `OPENAI_COMPATIBLE_BASE_URL`、`OPENAI_COMPATIBLE_API_KEY`、`OPENAI_COMPATIBLE_MODEL` 和 `PROVIDER_TIMEOUT_SECONDS` 配置，调用同步 `images/generations` 并解析 `data[].b64_json` 或 `data[].url`。
- Reason: OpenAI-compatible 形态与现有 Web 参考项目和多家中转/云端生图服务最接近，能最小成本验证“真实云端 provider -> Worker -> asset processor -> storage -> delivery”闭环。
- Impact: 默认 provider 仍是 `mock`，未配置密钥时不会启用真实 provider；真实 API smoke 需要用户自行配置密钥。当前 slice 不包含 fal.ai、异步 polling、provider routing、reference image 或 edit/mask。

## 2026-06-18: Web managed mode 先采用设置驱动的最小托管入口

- Decision: Web 第一版托管模式通过设置页开启，并配置服务端 API URL、workspace、project、campaign 和 provider；提交 prompt 时创建服务端 `ImageTask`，轮询任务状态，展示服务端候选 `Asset`，在详情页执行 select/reject。原浏览器直连 provider 路径保留为 legacy playground mode。
- Reason: 当前阶段的优先目标是让 Web 进入服务端 application core，验证 `ImageTask/Asset` 托管闭环，而不是同时实现完整 project/campaign 管理、reference/mask 参数迁移和模板系统。
- Impact: Web 已能参与正式资产流；prompt template、style preset、reference/mask descriptor 的服务端保存/传递已补齐，下一步应补真实 edit/mask provider 边界和更完整的 workspace/project/campaign 管理体验。

## 2026-06-18: Quality profile 先复用 project metadata

- Decision: 第一版服务端质量复用使用 `project.metadata_json.quality_profile` 保存 prompt template、negative prompt、style preset、reference image 参数和 generation config；创建任务时通过 `use_project_quality_profile` 显式复用，并把有效配置快照写入 `structured_input_json`。
- Reason: 当前 MVP 需要稳定质量输入和复用策略，但还不需要独立模板表、版本化模板库或完整 Web 管理界面；复用既有 `project.metadata_json` 可以避免 schema 迁移和过早复杂化。
- Impact: REST/MCP/Web 托管入口已能共用项目级质量配置；best-of 自动选优和 reference/mask descriptor 已在后续 slice 补齐，真实 provider edit/mask 调用仍待后续实现。

## 2026-06-18: Best-of 第一版采用本地启发式

- Decision: 第一版 best-of 自动选优通过 `selection_mode=auto` / `best_of` 触发，Worker 在资产登记后先使用 `local_metadata_v1` 本地启发式选出一张候选并标记 selected；当前第二版已在此基础上补 `best_of_config` 和 `http_judge_v1`，当前第三版再补可选 auto reject，但仍不改变轻量资产状态模型。
- Reason: 当前目标是削弱逐张人工审核成本并跑通自动推荐状态流；先有本地启发式基线，再补可插拔视觉/LLM 打分和可选 auto reject，更适合 MVP 的渐进演进顺序。
- Impact: Web managed mode 默认传 `selection_mode=auto`，多候选任务完成后会出现推荐资产；该推荐和 auto rejected 候选都仍可被用户手动覆盖。后续重点从 best-of 状态流转向限流、审计和多 key 等生产 hardening。

## 2026-06-18: Advanced managed input 先迁移 descriptor，不上传原图

- Decision: Web/MCP/REST 第一版高级托管输入先迁移 reference image、mask/edit descriptor 和 generation config；服务端把这些字段写入 `structured_input_json` 和 asset `parameters_json`，但暂不上传 Web IndexedDB 原图/mask，也不执行真实 provider edit/mask 请求。
- Reason: 当前目标是让高级输入不再被 Web managed mode 拦截，并确保服务端资产闭环不丢上下文；直接引入文件上传、服务端取回和各 provider edit/mask API 会扩大 schema、存储和安全边界。
- Impact: 后续真实 edit/mask provider slice 可以复用已保存 descriptor，但还需要新增服务端可访问的输入图片/遮罩存储或上传路径。

## 2026-06-18: Repair/reconcile 管理能力先放在本地 CLI

- Decision: 第一版 repair/reconcile 通过 `vag repair scan/requeue/verify-asset` 本地维护命令提供，直接读取 `DATABASE_URL`、`REDIS_URL` 和 `STORAGE_ROOT`；暂不暴露 REST/MCP 管理接口。
- Reason: repair 能力会扫描文件系统、重置任务状态并重新入队，属于本地自托管运维动作；在尚未实现项目级 API key 和权限模型前，不宜开放为远程 HTTP 能力。
- Impact: Docker image 中的 `/app/vag` 可用于 smoke 和人工修复；未来如果要远程化 repair，需要先完成鉴权、审计和更严格的操作范围控制。

## 2026-06-18: 第一版项目鉴权采用实例级 Basic Auth + 项目级单 key

- Decision: HTTP API 第一版使用两层可选鉴权：实例入口可配置 `BASIC_AUTH_USERNAME` / `BASIC_AUTH_PASSWORD`，project 维度在 `project.metadata_json.access_config` 中维护单把活动 API key 的 `name/preview/hash`；客户端优先通过 `X-API-Key` 传项目 key，也兼容 Bearer token。
- Reason: 当前产品面向单体平台和小团队自托管，先需要“能挡住公开入口、能按 project 隔离凭据、能让 Web/CLI/脚本继续用”的最小方案，而不是完整用户体系、RBAC 或多 key 管理。
- Impact: REST 新增 `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/access-config`；Web managed mode 新增 project key、Basic user、Basic password 设置；`vag` 通过 `AGENT_IMAGEFLOW_BASIC_USER`、`AGENT_IMAGEFLOW_BASIC_PASS`、`AGENT_IMAGEFLOW_API_KEY` 透传鉴权。第一版只保留一把活动 key，并使用标准库 `sha256` + 常量时间比较，后续如果暴露到更复杂环境，再升级到审计、限流和更强密钥管理。

## 2026-06-18: Web scope 管理第一版放在设置页完成

- Decision: 第一版 Web scope 管理不新开独立页面，而是在设置页的托管模式区域补“服务端同步 + 下拉选择 + 快速新建”；scope REST 接口新增列出/创建 workspace、project、campaign，并视为实例级管理操作，若启用了 Basic Auth 则只要求 Basic Auth。
- Reason: 当前最主要摩擦是托管模式仍依赖手填 seed scope；先把“查 + 选 + 新建”做进已有设置页，可以最小改动地让日常使用更顺，而不会把 slice 扩张成完整后台系统。
- Impact: Web 已不再主要依赖手写 `workspace_id/project_id/campaign_id`；后续如果要补独立 scope 管理页、rename/delete/archive 或更细权限，再在这个基础上继续演进。

## 2026-06-18: 服务端缩略图统一由 cwebp 生成

- Decision: 服务端缩略图不再依赖 provider 返回的 thumbnail bytes，而是统一基于原图在 asset processor 中生成 `.webp`；运行时镜像安装 `libwebp-tools`，通过 `cwebp` 完成 resize 和 WebP 编码。
- Reason: 架构和输入输出规格都要求稳定的服务端 thumbnail output，并明确建议 `thumbnails/{asset_id}/{version}.webp`；Go 标准库没有 WebP 编码器，而当前项目规则又不希望为此引入新的 Go 第三方图像依赖。
- Impact: `thumbnail_path` 的扩展名切换为 `.webp`，`GET /api/assets/{id}/thumbnail` 返回 `image/webp`；Docker Compose 继续作为标准运行环境，若直接运行本地二进制则需要系统 PATH 中存在 `cwebp`。

## 2026-06-18: 第一版服务端输入文件先落 scope 内本地存储，不入数据库

- Decision: 第一版 reference image / mask 的服务端可访问路径通过 scope 内 `input-files` 实现，文件和 metadata 直接落到本地文件系统；`CreateTask` 只接受公开的 `input_file_id`，并在服务端把它解析成内部 `resolved_input_files`，不新增数据库表。
- Reason: 当前 slice 需要尽快把 Web 托管模式和真实 provider edit/mask 接起来，但项目规则不希望在这一片引入数据库迁移、额外依赖和更宽的索引面。scope 内文件系统 metadata 足够支撑 MVP 闭环。
- Impact: Web managed mode 现在会先上传 reference image / mask，再创建带 `input_file_id` 的任务；OpenAI-compatible provider 在存在已解析输入文件时走 `/images/edits` multipart。后续如果需要远程 URL 抓取、asset reuse、输入文件治理或跨实例管理，再考虑独立表或对象存储索引。

## 2026-06-18: 生产入口默认交给反向代理，保持 compose 为开发友好

- Decision: 保持当前 `docker-compose.yml` 为本地开发友好配置；生产自托管默认通过反向代理/TLS 暴露 UI/API，对外只开放反向代理入口，不直接公网暴露 PostgreSQL 和 Redis。
- Reason: 本片目标是补齐 README/demo/自托管说明，而不是重写部署拓扑。开发环境需要低摩擦启动，生产环境则需要更小暴露面和更清晰的域名入口。
- Impact: README 与 Runbook 明确要求在公网部署时设置 `PUBLIC_BASE_URL`、启用 Basic Auth 和 project API key，并通过反向代理承接 `/api/*`；后续如果需要更强生产安全能力，再继续补限流、审计和多 key 策略。

## 2026-06-19: 并发专项先优化 Images API 正式资产路径

- Decision: 本轮把优化目标锁定为平台吞吐和可控并发：保留 openai-compatible 的 `/images/generations` / `/images/edits` 正式资产路径，新增 Worker 并发覆盖、provider 级 backpressure、attempt 可观测和 benchmark 工具；暂不实现 Responses API image_generation adapter 或 streaming partial images 状态机。
- Reason: 当前慢的主要根因是 worker 实际串行、同步 provider timeout/retry/backoff 放大总耗时，以及缺少 attempt/benchmark 数据。Images API 路径更直接、成本结构更清晰、落盘闭环更简单，适合作为正式资产生产基线；Responses/streaming 更适合交互体感和进度预览，需要另行设计半成品状态机。
- Impact: `WORKER_CONCURRENCY` 可按环境变量调到 2/4；真实 provider 建议先从 provider cap=2 小样本压测，成功率、timeout 和 429 稳定后再评估更高档。后续 P1 Provider Throughput & Reliability 已将自托管默认收敛为 provider cap=3。

## 2026-06-19: P1 Provider Throughput & Reliability 收紧真实 provider 默认值

- Decision: 保留 `WORKER_CONCURRENCY=6` 作为平台吞吐默认，但将真实 provider 默认入口 cap 调整为 `OPENAI_COMPATIBLE_MAX_CONCURRENCY=3`、`FAL_MAX_CONCURRENCY=3`，并将 `PROVIDER_TIMEOUT_SECONDS` 默认提升到 `300`；openai-compatible 额外拆分 connect/header/total timeout profile，task attempt 写入 queue/provider/download/store/thumbnail 阶段指标，benchmark run 自动带 `session_id/batch_id` 以便 batch progress 查询。
- Reason: mock benchmark 已证明平台本地 worker/storage 不是主要瓶颈；真实 provider cap=6 小样本出现 2/6 timeout，说明把 worker 并发直接等同于 provider 并发风险较高。cap=3 + 300s total timeout 更适合作为自托管默认安全档，后续真实 provider 应按 cap `2 -> 3 -> 4` 小样本确认。
- Impact: provider 生产吞吐会更保守，但 timeout/429 风险降低；需要更高吞吐时应使用 `vag benchmark image-generation --allow-paid-provider` 在用户确认费用后验证。仍不做 streaming、partial images、adaptive backpressure 或复杂 provider probe。

## 2026-06-22: Project Context Web 面板先按最小可打通场景实现

- Decision: P1-PCTX-008 和 P1-PCTX-009 已按 `docs/project/stories/slice-037-pctx-web-panel-and-pet-story-scenarios.md` 的边界落地：Web 第一版只做当前 project 的 Visual Context 维护入口、asset card Mark as reference、Web managed task context selector；clean 萌宠故事 mock 回归已验证角色、参考图、recipe、scene metadata、batch progress 和 Web 可见性。暂不提前实现独立 Batch / Story / Scene View、Export Pack、NAS/WebDAV/SMB、Usage Tracking、通用 DAM 或运营后台。
- Reason: 服务端和 MCP/REST/CLI 已经能处理 Project Visual Context，当前最大断点是用户在 Web 上无法维护和使用这些长期上下文。如果先做批量故事视图或导出系统，会把产品重心从“project 级视觉生产上下文”拉到更复杂的生产/交付平台，增加返工风险。
- Impact: Web Project Context modal 使用 Admin session / Basic fallback 读取和保存完整 `project.metadata_json.visual_context`，不新增数据库表或迁移；Web 只把 context ids 传给服务端，不在前端拼最终 prompt；asset card Reference 只写 reference binding，不复制文件或改变 asset 状态。P1-PCTX-009 只运行了 mock provider，evidence 已写回 CSV、TASKS、PROJECT_PLAN、PROJECT_STATUS_MAP、CHECKPOINTS 和 RUNBOOK。

## 2026-06-22: Batch Story Export Foundation 作为下一阶段主线

- Decision: 下一阶段正式入口为 `issues/next-phase-p1-batch-story-export-foundation.csv`。第一轮只做 batch/story/scene 聚合查看、MCP asset filter parity、scene 级 retry/regenerate、selected-only review、JSON manifest / 可选 ZIP 和 NAS/Docker 文件访问说明；不做小红书发布、内容日历、账号运营后台、通用 DAM、内置 WebDAV/SMB server、Usage Tracking 或 AI 视觉质检。
- Reason: Project Visual Context 已证明角色、参考图和 recipe 可以贯穿 task/asset metadata；下一个真实断点是用户批量生成一组故事图后，无法在 Web 中按 scene 看进度、选图、重试和交付。直接做完整导出系统或运营后台会扩大范围，而 summary API + grouped view + manifest 可以先打通生产闭环。
- Impact: 第一版继续使用 `metadata_json.session_id/batch_id/story_id/scene_id/target_path` 作为 grouping contract，优先不新增数据库表；NAS/WebDAV/SMB 由部署环境提供文件访问，Agent ImageFlow 继续负责 DB metadata、状态、delivery URL、manifest 和审计。

## 2026-06-22: Batch summary route contract

- Decision: P1-BSE-002 选择 `GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-summary` 作为第一版批量故事聚合路由，而不是 `story-summary`。`story_id` 作为可选 query filter，默认仍以 `session_id/batch_id` 表达一次生产批次。
- Reason: 现有平台已有 `batch-progress` 和 `vag batch progress`，使用 `batch-summary` 可以复用用户心智，并保持 story 是 batch 内的业务维度。
- Impact: 后续 P1-BSE-003 API、P1-BSE-005 Web production view 和 P1-BSE-009 manifest 都应复用 `slice-041` 的 metadata-only grouping、setup exclusion、scene ordering 和 regenerate metadata contract。

## 2026-06-22: Scene regenerate 第一版采用新建 task 的 metadata lineage

- Decision: P1-BSE-007 确认 scene regenerate 第一版采用 `create-new-task-as-regeneration`：不 retry 原 task，不 mutate 原 task，不覆盖旧 assets，不自动改变 selected/rejected。新 task 复用同一 `project_id/campaign_id/session_id/batch_id/story_id/scene_id`，通过 `structured_input_json.metadata_json` 和 asset `parameters_json` 记录 `regenerated_from_task_id`、`regenerate_no`、reason、overrides 和 visual context snapshot 摘要。
- Reason: 用户在萌宠故事批量生产中需要针对单个失败或效果差的 scene 重新生成，同时保留旧候选、旧错误、已选图和审看痕迹。把 regenerate 设计成原 task retry 或覆盖旧资产会破坏 batch summary、manifest 和 selected-only 交付语义，也会让 agent 难以追溯哪次任务产生了哪张图。
- Impact: P1-BSE-008 已按该决策实现第一版 Web/REST scene-level action，内部复用 create task / worker / asset registry 现有链路，且不需要 schema migration。`batch-summary`、`batch-progress`、asset list 和未来 manifest 必须继续通过 metadata lineage 识别 regenerated scene attempts；selected-only manifest 只有在用户显式 select 新 asset 后才切换。CLI/MCP regenerate command 和复杂 Web override UI 仍后置。

## 2026-06-22: P1 第一轮不实现服务端 ZIP export

- Decision: P1-BSE-010 确认第一轮不实现服务端 ZIP 打包。交付路径优先采用 JSON manifest + NAS/filesystem 访问；ZIP 仅在后续明确确认“小批量 selected assets 一键包”时另开实现切片。
- Reason: 当前真实工作流需要的是图片资产生产、审看、追踪和交付清单，而不是复杂导出系统。JSON manifest 已能让 agent 和人拿到 delivery URL、metadata URL、target_path、scene/story/task 和 visual context；NAS/WebDAV/SMB/Finder 更适合承担大文件浏览、复制和备份。服务端 ZIP 会引入数量/大小保护、临时文件、路径安全、并发打包、下载中断和低资源 NAS 压力。
- Impact: P1-BSE-011 应文档化 Docker storage root、只读共享和 manifest/file-system 分工；不要在应用内实现 WebDAV/SMB server 或 ZIP。未来 ZIP 必须限制小批量 selected assets，并加入数量、大小、路径和 secret 保护。

## 2026-06-22: NAS / Docker 文件访问不成为资产事实源

- Decision: P1-BSE-011 确认 NAS、WebDAV、SMB 和 Finder 只作为部署层文件访问能力，主要承担浏览、复制、交付拷贝和备份；Agent ImageFlow 的 DB / metadata / manifest 继续作为 task、asset、selected/rejected/published、visual context、scene/story/batch 追踪和审计的事实源。
- Reason: 自托管和 NAS 场景下，直接访问图片目录很高效，但如果把文件夹移动、重命名或删除当成资产状态变化，会破坏 delivery URL、manifest、storage-integrity、selected/published 保护和跨入口一致性。
- Impact: RUNBOOK 明确建议 NAS / WebDAV / SMB / Finder 常规访问只读；不要手动移动、重命名或删除平台管理的 selected / approved / published 资产；备份/恢复必须同时处理 Postgres dump 和 storage root 一致快照。应用内不实现 WebDAV/SMB server，也不把目录树扩成通用 DAM。
