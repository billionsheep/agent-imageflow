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
- 部署发布：GitHub Actions 构建 GHCR 私有 API/Web 镜像，服务器用 `docker-compose.prod.yml` 拉取镜像运行。

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

1. 服务器/NAS 部署演练
   - 使用 `docs/project/SERVER_DEPLOYMENT_GUIDE.md`。
   - 验证 GHCR private package 拉取权限。
   - 准备服务器 `.env.prod`，不要提交或打印。
   - 配 HTTPS 反向代理：`/api/*` 和 `/healthz` 到 API，其余路径到 Web。
   - 跑 healthz、Web、Admin、mock task、MCP `tools/list` smoke。
   - 演练一次 `IMAGE_TAG` 更新和回滚。

2. 真实试用观察
   - 用低并发真实 provider 跑小批量萌宠故事图。
   - 观察 Web 审图是否仍有闪烁、卡顿、字段噪音或操作路径太长。
   - 记录具体 URL host、Admin 登录态、scope、filter、batch/session/story/scene 和复现步骤。

3. 发布版本策略确认
   - 当前 `main` 和 `sha-<commit>` 镜像可用。
   - 是否创建 `v0.1.0` tag 作为正式第一版，需要单独确认。
   - 若创建 tag，会触发 GHCR `v0.1.0` 镜像发布。

4. 备份与恢复演练
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

目标：

- 把 GHCR 镜像部署到真实服务器/NAS。
- 完成 HTTPS 反代、Admin 登录、mock 生成、MCP tools/list、备份/回滚 smoke。
- 不运行真实 provider，除非单独确认 1 图 canary。

### P1 Trial: Pet Account Real Workflow Trial

建议文件名：

```text
issues/next-phase-p1-pet-account-real-workflow-trial.csv
```

目标：

- 用真实萌宠账号工作流跑 1 个 project、1 个 campaign、2-3 个 story scenes。
- 验证 agent 写故事 -> MCP/REST 生图 -> Web 审图 -> Production View select/reject -> JSON manifest/NAS 交付。
- 只做低频真实 provider canary，不做 benchmark。

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

