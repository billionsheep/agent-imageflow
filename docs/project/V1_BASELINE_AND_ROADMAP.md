# V1 Baseline and Roadmap

本文档记录 Agent ImageFlow 当前第一版基线、可验收范围、剩余任务和未来方向。它用于后续部署、试用和新一轮 CSV 拆分，不替代 `TASKS.md` 或 `PROJECT_STATUS_MAP.md`。

## V1 Baseline

V1 的产品定义：

Agent ImageFlow 是 AI 可调用的图片资产生成与管理能力平台。它让 Codex、Claude、Cursor、自动化脚本、内容系统或业务后台通过 MCP / REST API / CLI / Web 创建图片资产，并拿到可追踪、可审核、可复用、可交付的正式结果。

V1 已具备：

- 多入口统一事实源：Web / MCP / REST / CLI 共用 Go application core。
- 服务端闭环：create task -> worker -> provider -> 文件落盘 -> asset/version/metadata 登记 -> thumbnail -> delivery URL。
- Scope 隔离：workspace / project / campaign / task / asset 层级已可用。
- Provider：mock、openai-compatible、fal 基础适配和输入复用已接入。
- 资产治理：select/reject、best-of、auto reject、thumbnail、metadata、repair/reconcile、storage governance、integrity view。
- 可见性：Web Recent Assets、服务端资产库、筛选、分页、lazy loading、Admin session、跨 scope 最近资产。
- Project Visual Context：project 级角色卡、reference binding、prompt recipe、quality defaults 和 task/asset 快照。
- Batch / Story / Scene：batch summary、Production View、scene asset actions、scene regenerate、JSON manifest。
- Web 审图体验：长 ID 和工程字段默认折叠，卡片优先显示图片、剧情/画面摘要、story/scene/source/created/target 和状态。
- Web 控制台入口：未登录时只显示 Admin 登录页；登录后进入完整服务器托管控制台，使用服务器配置的 provider 能力，资产库不再二次登录。
- 部署发布：GitHub Actions 构建 GHCR 私有 API/Web 镜像，服务器用 `docker-compose.prod.yml` 拉取镜像运行。
- 部署可用性：Web 生产镜像可同源代理 `/api/*` 与 `/healthz`，Admin session 可读取图片 delivery，控制台提供安全 runtime status 和 Current Scope 导航。

V1 已验证的关键真实场景：

- 萌宠账号图片资产生产最小闭环：项目视觉上下文、prompt recipe、真实 provider、落盘、缩略图、metadata、Recent Assets 和 Production View 已打通。
- MCP 工具入口真实 provider 1 图 canary：MCP `create_image_task` 已证明不只停留在 mock 路径。
- 服务器部署路径：`main` 分支 push 后 Docker Publish workflow 已构建并发布 API/Web 镜像。

## V1 Scope Boundary

V1 做：

- 图片任务、生图 provider 适配、资产落盘、缩略图、metadata、审核/选中状态。
- MCP / REST / CLI / Web 共用同一事实源。
- project 作为长期视觉生产上下文。
- batch/story/scene 的生产查看、单 scene regenerate 和 JSON manifest。
- 小团队自托管部署、Admin session、project API key、Basic Auth、限流和审计。

V1 不做：

- 小红书发布、内容日历、账号运营后台。
- 泛设计平台、白板、模板市场、通用 DAM。
- SaaS 注册登录、多租户、复杂 RBAC、计费。
- 每用户 provider key 管理。
- 内置 WebDAV/SMB server。
- 大规模真实 provider benchmark。
- 视觉质检 AI 自动裁决。
- 数据库 migration 框架。

## Immediate Remaining Tasks

这些是 V1 之后最应该先做的运维/验收任务。

1. 服务器/NAS 部署复验
   - 使用 `docs/project/SERVER_DEPLOYMENT_GUIDE.md`。
   - 当前已新增 `issues/next-phase-p1-server-deployment-rehearsal.csv` 和 `docs/project/stories/slice-053-server-deployment-rehearsal.md` 作为演练入口。
   - Volcengine 旧服务已完成 GHCR `main` 更新、升级前备份、临时 HTTP health/Web、MCP `tools/list` 和 mock benchmark smoke。
   - 准备服务器 `.env.prod`，不要提交或打印。
   - 推荐 `PUBLIC_BASE_URL` 指向 Web/HTTPS origin；Web 镜像可代理 `/api/*` 与 `/healthz` 到内部 API。
   - 继续补 HTTPS/Caddy 正式同源入口、浏览器 Admin、Recent Assets 缩略图、original/metadata delivery smoke。
   - 演练一次 restore 和 `IMAGE_TAG` 回滚。

2. 真实试用观察
   - 用低并发真实 provider 跑小批量萌宠故事图。
   - 观察 Web 审图是否仍有闪烁、卡顿、字段噪音或操作路径太长。
   - 记录具体 URL host、Admin 登录态、scope、filter、batch/session/story/scene 和复现步骤。

3. MCP Service Pack smoke
   - 接入 guide、MCP config 示例、萌宠 scene 示例和 smoke 说明已落地。
   - 下一步只需回填 `tools/list`、mock create task、get task 和 delivery info 的实际 evidence。
   - 继续明确 Project API Key、Basic Auth、Admin Login 和 provider key 的边界。

4. Character Reference Intake and Consistency
   - 后端已支持把 campaign input-files 提升为正式 project reference asset。
   - 已支持角色主图/参考图字段、任务自动带参考资产、provider 参考参与 metadata 和参考图失败诊断。
   - Web 角色卡已显示主图/参考图缩略图和缺图警告。
   - 从资产卡打开 Project Context 时，已可把当前 asset 设为角色主图、加入角色参考图，或保存为项目参考图。
   - 下一步补完整 mock pet consistency smoke、browser smoke 和人工确认后的 1 图真实参考 canary。

5. Web Review Feedback and Stability
   - Web 前端和 tests 范围已完成：Select / Reject 后卡片、scene header、coverage count 和按钮状态会局部变化。
   - 下拉和筛选刷新保留旧内容，审图请求已有去重、旧响应丢弃和 429 友好提示。
   - 下一步只建议补一次 browser smoke evidence。

6. Safe Delete and Trial Reset
   - CLI + Admin-only REST foundation 已完成：cleanup preview/execute 可按 asset/task/session/batch/story/campaign 定位候选。
   - Web Scope 管理已提供当前 campaign 的数据清理入口，可 preview/execute，并要求确认短语。
   - 继续要求 dry-run token 或显式确认，保护 selected/published/approved 并写 audit。
   - 单 asset archive/restore 已补到 Admin REST/Web/CLI；archived 默认不进 cleanup。
   - 下一步补 task/input-file reset、完整 browser smoke 和生产备份演练；MCP 第一轮不删除 workspace/project/campaign/asset。

7. 发布版本策略确认
   - 当前 `main` 和 `sha-<commit>` 镜像可用。
   - 是否创建 `v0.1.0` tag 作为正式第一版，需要单独确认。
   - 若创建 tag，会触发 GHCR `v0.1.0` 镜像发布。

8. 备份与恢复演练
   - Postgres dump。
   - storage root / NAS snapshot。
   - `.env.prod` 安全备份。
   - 验证恢复后的 asset original/thumbnail/metadata URL。

## Recommended Next CSV Slices

### P1 Ops: Server Deployment Rehearsal

建议文件名：

```text
issues/next-phase-p1-server-deployment-rehearsal.csv
```

当前状态：

- CSV 与 `docs/project/stories/slice-053-server-deployment-rehearsal.md` 已新增。
- `docs/project/SERVER_DEPLOYMENT_GUIDE.md` 已补部署演练证据模板。
- 本地发布材料仍通过静态检查；Volcengine 旧服务已完成 `main` 更新、升级前备份、临时 HTTP health/Web、MCP `tools/list` 和 mock benchmark smoke；HTTPS/Caddy 正式入口、浏览器 Admin delivery smoke、restore 和回滚仍待执行。

目标：

- 把 GHCR 镜像部署到真实服务器/NAS。
- 完成 HTTPS 反代、Admin 登录、mock 生成、MCP tools/list、备份/回滚 smoke。
- 不运行真实 provider，除非单独确认 1 图 canary。

### P0/P1 Closed: Deployment Auth, Scope and Project Console

记录文件：

```text
issues/next-phase-p0-p1-deployment-auth-scope-project-console.csv
```

结果：

- 已修复生产 Web/API 不同 origin 导致的缩略图鉴权问题。
- 已明确 Basic Auth、Admin Login、Project API Key、provider key 四种凭据语义。
- 已增加 Current Scope 导航和 Project Context 工作区摘要，降低手填长 ID 的操作负担。

### P1 Trial: Pet Account Real Workflow Trial

建议文件名：

```text
issues/next-phase-p1-pet-account-real-workflow-trial.csv
```

目标：

- 用真实萌宠账号工作流跑 1 个 project、1 个 campaign、2-3 个 story scenes。
- 验证 agent 写故事 -> MCP/REST 生图 -> Web 审图 -> Production View select/reject -> JSON manifest/NAS 交付。
- 只做低频真实 provider canary，不做 benchmark。

### P1 Agent Onboarding: MCP Service Pack

建议文件名：

```text
issues/next-phase-p1-mcp-service-pack.csv
```

当前状态：

- 已新增 `docs/project/MCP_SERVICE_GUIDE.md`、`examples/mcp/agent-imageflow.local.json`、`examples/mcp/create-pet-scene.json` 和 `examples/mcp/smoke.md`。
- 已明确 Project API Key、Basic Auth、Admin Login 和 provider key 的边界；MCP 配置示例不写真实 secret，也不需要 Admin cookie 或 provider key。
- 已补删除边界：MCP 继续负责 create/get/list/select/reject/delivery，删除和试用重置走 Admin Web/REST/CLI。

剩余目标：

- 跑 `tools/list`、mock create task、get task、list assets 和 delivery info smoke。
- 把 smoke evidence 回填到 CSV / CHECKPOINTS。
- 暂不做远程 HTTP MCP、新账号系统、多用户权限、provider key 下发、新工具 schema 大改或真实 provider 默认 smoke。

### P1 Visual Consistency: Character Reference Intake

建议文件名：

```text
issues/next-phase-p1-character-reference-intake-consistency.csv
```

当前状态：

- 后端已支持 input-file promote 为正式 reference asset，并写入 asset/version/thumbnail/metadata。
- Character Profile 已支持 `primary_asset_id`、`reference_asset_ids`、`reference_policy` 和 `appearance_lock_notes`。
- CreateTask 使用 `character_ids` 与 `use_project_visual_context=true` 时可自动展开角色参考资产，并在 task/asset metadata 中记录参考参与诊断。
- Web Project Context 角色卡已展示主图/参考图缩略图和缺图警告；从资产卡带 `asset_id` 打开时，已提供“设为主图 / 加入参考图 / 保存为项目参考图”的快捷绑定区。

剩余目标：

- 跑完整 mock pet character consistency smoke。
- 在人工确认费用后跑 1 图真实参考 canary；失败时明确告诉用户参考图没有参与生成。
- 补完整 browser smoke，确认角色缩略图、快捷绑定、参考参与 metadata 和 Web 状态展示不断链。
- 验收分三层：平台链路成功、参考图参与成功、角色一致性人工判断通过。
- 暂不做通用 DAM、模板市场、AI 自动视觉质检、provider key 下发或大规模真实 provider benchmark。

### P1 Web UX Follow-up: Review Feedback and Stability

建议文件名：

```text
issues/next-phase-p1-web-review-feedback-stability.csv
```

当前状态：

- Web 前端与 tests 范围已完成：`ServerAssetLibrary` 与 `ProductionViewModal` 已有 per-asset pending、optimistic status/style、失败回滚、局部 counts 更新、旧内容保留、请求去重/旧响应丢弃和 429 友好提示。

剩余目标：

- 增加 browser smoke，把点击选择/拒绝、切换下拉和 Production View 状态变化纳入回归证据。
- 这是 P1 Web UX Smoothness 之后的真实试用 follow-up，不复开旧 CSV，不大改 Settings 信息架构。

### P1 Data Lifecycle: Safe Delete and Trial Reset

建议文件名：

```text
issues/next-phase-p1-safe-delete-and-trial-reset.csv
```

当前状态：

- CLI `vag storage cleanup-preview/execute` 已支持 `--asset-id`、`--task-id`、`--session-id`、`--batch-id`、`--story-id` 和 `--deprecated`。
- Admin-only REST `storage-cleanup-preview/execute` 已接入，必须使用 Admin session；Project API Key 和 Basic Auth 不授予清理权限。
- Web `ScopeManagerModal` 已提供当前 campaign 的数据清理面板，支持 cleanup preview/execute，展示候选统计、保护计数和脱敏 dry-run token，execute 前必须输入确认短语。
- 清理继续复用 dry-run token / confirm、selected/published/approved 保护和 audit；MCP 不开放 destructive tools。

剩余目标：

- 单 asset archive/restore 已完成；补 task/input-file 级 reset、完整 browser smoke 和生产备份演练。
- 补完整 browser smoke 和 RUNBOOK 运维命令整理，确认 Web 清理入口、CLI/REST foundation、审计和备份策略一致。
- 不做通用 DAM、回收站复杂权限、多用户审批流、MCP 大范围 destructive tool 或绕过备份的生产清库。

### P1 Product: Settings Information Architecture

建议文件名：

```text
issues/next-phase-p1-settings-information-architecture.csv
```

目标：

- 重新设计设置页分组：控制台状态、业务空间、服务端能力、高级/旧模式、数据管理、关于。
- 明确 Admin Login、Basic Auth、Project API Key、provider key/base URL 的边界。
- 先做结构和文案方案，再进入实现，避免继续在旧 playground 设置页上补丁式堆功能。

### P2 Product: Usage Tracking and Edit Lineage

建议文件名：

```text
issues/next-phase-p2-usage-lineage.csv
```

目标：

- 记录资产被用于哪个内容、文章、笔记、封面或导出包。
- 记录 edit/regenerate lineage，明确一张图从哪个 prompt/reference/source task 发展而来。
- 不做小红书发布，不做内容日历。

### P2 Delivery: Export Pack ZIP and Multi Select

建议文件名：

```text
issues/next-phase-p2-export-pack-zip.csv
```

目标：

- 在 JSON manifest 稳定后，支持小批量 selected assets ZIP。
- 限制数量、总大小和路径。
- ZIP 内包含 original、thumbnail、metadata、manifest。
- 不扩展成通用 DAM 下载中心。

### P2 Safety: Deployment and Secret Hardening

建议文件名：

```text
issues/next-phase-p2-deployment-secret-hardening.csv
```

目标：

- Web 显示当前登录态、API host、server-first provider 状态和 safe config 摘要。
- 登录失败限流、Secure cookie / HTTPS 反代检查指南。
- non-exposure regression 覆盖 provider key、project key、cookie/session、local path。
- 不实现每用户 provider key。

### P3 Quality: Visual Consistency Review Aid

建议文件名：

```text
issues/next-phase-p3-visual-consistency-review-aid.csv
```

目标：

- 先做人工审图辅助，不做自动裁决。
- 对角色、style、scene 的 prompt/reference 快照做对比展示。
- 未来再评估 AI 视觉质检。

## Future Direction

产品方向继续保持：

```text
workspace = 个人/团队/客户/业务大边界
project   = 长期经营对象，例如萌宠账号、品牌 IP、产品线、技术图账号
campaign  = 一次具体生产批次，例如一期故事、一组四格漫画、一周封面图
asset     = 生成或上传后可追踪、可审核、可复用、可交付的图片资产
```

下一阶段不要扩成账号运营系统，而是沿着三条线推进：

- 部署线：服务器/NAS 上线、备份、回滚、HTTPS、GHCR 权限。
- 试用线：用真实小批量业务流发现 Web 审图和 agent 调用中的摩擦。
- 产品线：usage tracking、edit lineage、导出包、deployment secret hardening。

继续后置：

- SaaS 化、多租户、复杂 RBAC。
- 小红书自动发布、内容日历、账号运营后台。
- 通用 DAM、模板市场、多人协作审核流。
- 内置 WebDAV/SMB server。
- 大规模 provider benchmark。
- 过早视觉质检 AI。
