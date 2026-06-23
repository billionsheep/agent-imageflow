# Slice 053: Server Deployment Rehearsal

## Context

V1 baseline 已形成，`slice-052` 已完成 GHCR 私有镜像发布流、生产 compose、Web 镜像、`.env.example.prod`、部署静态检查和服务器交接文档。当前仍缺真实服务器/NAS 环境里的上线证据：能否拉取 GHCR 镜像、能否通过 HTTPS 同源入口登录 Web、能否跑 mock task 和 MCP smoke、能否备份恢复、能否通过 `IMAGE_TAG` 回滚。

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

- 实际连接用户服务器或 NAS。
- 创建、读取、打印或提交真实 `.env.prod`。
- 读取、打印或迁移 GHCR token、provider key、project API key、Basic/Auth/Admin cookie/session。
- 默认运行真实 provider。
- 创建 `v0.1.0` tag、推送远程分支或发布版本。
- 数据库 schema migration 框架。
- Kubernetes、Terraform、Helm、自动证书申请或托管数据库。

## Acceptance Criteria

- CSV 明确拆出 GHCR pull、HTTPS 同源入口、Admin Web mock smoke、MCP smoke、备份恢复、回滚和可选真实 provider canary。
- 部署指南告诉执行者应该记录哪些非敏感证据，以及哪些内容禁止粘贴到文档、issue 或聊天中。
- 项目任务状态明确：本轮完成的是部署演练准备包，真实服务器上线仍待执行。
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

Status: prepared / waiting for real server execution.

- 新增 P1 Server Deployment Rehearsal CSV，拆出 8 个验收项。
- 新增本 story，明确本轮只做部署演练准备包，不执行真实服务器上线。
- 在服务器部署指南中增加 evidence template 和禁止记录项。
- 同步项目管理文档，让下一步默认进入真实服务器/NAS 演练，而不是继续扩功能。

## Verification

Local checks passed:

- `python3 scripts/check_deployment_release.py`
- `docker compose -f docker-compose.prod.yml --env-file .env.example.prod config`
- CSV parse: `issues/next-phase-p1-server-deployment-rehearsal.csv` has 8 rows and required columns.
- `git diff --check`

Manual checks still pending external environment:

- GHCR pull on target server.
- HTTPS reverse proxy smoke.
- Admin Web mock task smoke.
- MCP stdio mock smoke on deployed image.
- Postgres + storage restore rehearsal.
- `IMAGE_TAG` update/rollback rehearsal.

## Assumptions and Risks

- 真实服务器、GHCR token、域名、反向代理和 `.env.prod` 属于外部状态，本地无法代替验收。
- 备份恢复演练可能影响数据，必须使用测试环境或先由用户确认窗口和范围。
- 如果后续版本包含 schema change，回滚不再只是改 `IMAGE_TAG`，必须先补 migration/backup plan。
- 真实 provider canary 会产生费用，只能在用户明确确认后执行 1 图低频验证。
