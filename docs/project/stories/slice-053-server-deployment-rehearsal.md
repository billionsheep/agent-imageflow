# Slice 053: Server Deployment Rehearsal

## Context

V1 baseline 已形成，`slice-052` 已完成 GHCR 私有镜像发布流、生产 compose、Web 镜像、`.env.example.prod`、部署静态检查和服务器交接文档。当前仍缺完整服务器/NAS 验收闭环：能否通过 HTTPS 同源入口登录 Web、能否在浏览器里查看 Recent Assets delivery、能否恢复备份、能否通过 `IMAGE_TAG` 回滚。

2026-06-23 已在 Volcengine VPS 对旧服务执行一次更新演练：升级前备份、GHCR `main` pull/up、临时 HTTP health/Web smoke、MCP `tools/list` 和 mock benchmark smoke 已通过。

## Product Goal

把 V1 从“已有生产部署材料”推进到“真实服务器/NAS 部署演练可执行、可验收、可回滚”的状态。

## User Flow

1. 运维者在服务器准备 `/opt/agent-imageflow`。
2. 运维者放置 `docker-compose.prod.yml` 和只在服务器保存的 `.env.prod`。
3. 运维者用只读 package token 登录 GHCR。
4. 运维者执行 prod compose `config`、`pull`、`up -d` 和 `ps`。
5. 运维者通过 HTTPS 反向代理打开 Web，并完成 Admin 登录。
6. 运维者用 mock provider 创建 1 张测试图片，确认 Recent Assets、thumbnail、original、metadata 和 Production View 可用。
7. 运维者执行 MCP `tools/list`，并在安全条件下跑 mock create/query/delivery smoke。
8. 运维者演练 Postgres dump + storage root/NAS 快照恢复。
9. 运维者演练 `IMAGE_TAG` 更新和回滚。

## Scope

In scope:

- 新增 `issues/next-phase-p1-server-deployment-rehearsal.csv`。
- 记录服务器部署演练 story。
- 在部署指南中增加非敏感 evidence template。
- 同步 `TASKS.md`、`PROJECT_PLAN.md`、`CHECKPOINTS.md`、`RUNBOOK.md`、`V1_BASELINE_AND_ROADMAP.md` 和 `PROJECT_STATUS_MAP.md`。
- 本地运行部署静态检查和 prod compose example config。

Out of scope:

- 配置正式域名、Caddy、Cloudflare 或 HTTPS 证书。
- 创建、读取、打印或提交真实 `.env.prod`。
- 读取、打印或迁移 GHCR token、provider key、project API key、Basic/Auth/Admin cookie/session。
- 默认运行真实 provider。
- 创建 `v0.1.0` tag、推送远程分支或发布版本。
- 执行 restore 或 `IMAGE_TAG` 回滚。
- 数据库 schema migration 框架。
- Kubernetes、Terraform、Helm、自动证书申请或托管数据库。

## Acceptance Criteria

- CSV 明确拆出 GHCR pull、HTTPS 同源入口、Admin Web mock smoke、MCP smoke、备份恢复、回滚和可选真实 provider canary。
- 部署指南告诉执行者应该记录哪些非敏感证据，以及哪些内容禁止粘贴到文档、issue 或聊天中。
- 项目任务状态明确：准备包已完成，Volcengine 旧服务更新 smoke 已完成；正式 HTTPS、浏览器 Admin delivery、restore 和回滚仍待执行。
- 本地部署发布材料通过静态检查。
- prod compose 可使用 `.env.example.prod` 渲染配置，不要求真实 secret。

## Technical Approach

- 保持 `docker-compose.prod.yml`、`.env.example.prod` 和 `scripts/check_deployment_release.py` 的现有部署契约不变。
- 不新增脚本或依赖；本轮只补项目管理和运行手册。
- 真实服务器命令继续集中在 `docs/project/SERVER_DEPLOYMENT_GUIDE.md`，避免在多个文档复制长命令后漂移。
- CSV 作为执行入口，story 作为范围与实现日志，RUNBOOK/V1/状态图作为导航。

## Data / Interface Impact

无数据库、API、MCP、Web 或 CLI 接口变更。

## Files Changed

- `issues/next-phase-p1-server-deployment-rehearsal.csv`
- `docs/project/stories/slice-053-server-deployment-rehearsal.md`
- `docs/project/SERVER_DEPLOYMENT_GUIDE.md`
- `docs/project/RUNBOOK.md`
- `docs/project/TASKS.md`
- `docs/project/PROJECT_PLAN.md`
- `docs/project/CHECKPOINTS.md`
- `docs/project/V1_BASELINE_AND_ROADMAP.md`
- `docs/project/PROJECT_STATUS_MAP.md`

## Implementation Log

Status: partial / Volcengine update smoke completed.

- 新增 P1 Server Deployment Rehearsal CSV，拆出 8 个验收项。
- 新增本 story，先明确部署演练准备包；后续继续记录真实服务器更新 evidence。
- 在服务器部署指南中增加 evidence template 和禁止记录项。
- 同步项目管理文档，让下一步默认补齐服务器/NAS 正式验收，而不是继续扩功能。
- 2026-06-23：在 Volcengine `/opt/huoshan/apps/agent-imageflow` 完成升级前备份、`main` 镜像拉取和 `api/worker/web` 原地重启。
- 2026-06-23：新镜像为 API `9637de835ac6`、Web `5e237d3f11f1`；API/Postgres/Redis healthy，Worker/Web running。
- 2026-06-23：公网临时入口 `http://163.7.5.68:18080/healthz` 与 `http://163.7.5.68:18081/healthz` 返回 200/ok，Web 首页返回 200。
- 2026-06-23：MCP `tools/list` 返回 6 个工具；mock benchmark `vps_update_smoke_20260623T062530Z` 完成 `task_c395196e3db93a6788ea -> asset_0a0efbb532fa943e0f92`。

## Verification

Local checks passed:

- `python3 scripts/check_deployment_release.py`
- `docker compose -f docker-compose.prod.yml --env-file .env.example.prod config`
- CSV parse: `issues/next-phase-p1-server-deployment-rehearsal.csv` has 8 rows and required columns.
- `git diff --check`

Server checks passed on Volcengine temporary HTTP deployment:

- GHCR `main` pull and compose `up -d api worker web`.
- `docker compose --env-file .env.prod ps`: API/Postgres/Redis healthy, Worker/Web running.
- Local API health: `status=200`, body `{"status":"ok"}`.
- Public Web health and API health: both `status=200`, body `{"status":"ok"}`.
- Public Web home: `status=200`, `content_type=text/html`.
- MCP stdio `initialize -> tools/list`: 6 tools.
- Mock benchmark: 1 task completed, 1 asset, 0 failed.
- Read-only DB asset check: generated asset version ready, thumbnail and metadata registered.

Manual checks still pending:

- HTTPS reverse proxy smoke.
- Admin Web mock task smoke.
- Postgres + storage restore rehearsal.
- `IMAGE_TAG` update/rollback rehearsal.

## Assumptions and Risks

- 域名、反向代理、restore 环境和 `.env.prod` 仍属于外部状态，临时 HTTP smoke 不能代替正式 HTTPS 同源验收。
- 备份恢复演练可能影响数据，必须使用测试环境或先由用户确认窗口和范围。
- 如果后续版本包含 schema change，回滚不再只是改 `IMAGE_TAG`，必须先补 migration/backup plan。
- 真实 provider canary 会产生费用，只能在用户明确确认后执行 1 图低频验证。
