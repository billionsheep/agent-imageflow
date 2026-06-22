# Slice 052: Deployment Release Pipeline

## Context

Agent ImageFlow 已具备 Web/MCP/REST/CLI 多入口、真实 provider canary、Web 审图控制台、batch/story/scene 视图和 NAS/Docker 文件访问边界。当前缺口是发布方式仍停留在本地开发 compose：`api` 和 `worker` 通过 `build: .` 从源码构建，Web 需要手动构建 `web/dist` 并由外部静态服务托管。

## Product Goal

把项目升级为可持续自托管发布流：GitHub Actions 构建并推送 GHCR 私有镜像，服务器只拉取镜像和运行生产 compose，不在服务器构建 Go/Web。

## Scope

In scope:

- GHCR 私有镜像发布契约。
- 后端 API/Worker/CLI/MCP 镜像。
- 独立 Web 静态镜像。
- GitHub Actions 测试、构建、发布。
- `docker-compose.prod.yml` 生产运行拓扑。
- `.env.example.prod` 与 secret 边界。
- 上线、更新、回滚、备份和 smoke 文档。
- 项目管理文件同步。

Out of scope:

- Kubernetes、Terraform、Helm、云厂商托管数据库。
- 自动证书申请和反向代理自动部署。
- 多租户账号、RBAC、SaaS 注册/计费。
- 迁移 provider key 到云 secret manager。
- 真实 provider benchmark 或默认 CI canary。
- 数据库 schema migration 框架；未来 schema change 另开计划。

## Acceptance

- `docker-compose.yml` 继续作为开发模式，不强行改成生产模式。
- `docker-compose.prod.yml` 使用 `image:`，不含 `build:`。
- `api` 与 `worker` 使用同一后端镜像，通过 command 区分 `/app/api` 和 `/app/worker`。
- Web 有独立镜像，runtime 只托管 `web/dist`。
- GitHub Actions 在 PR/push/tag 上跑 Web tests/build、容器化 Go tests、deployment static check 和 Docker build。
- main/tag 推送 GHCR；PR 不推镜像。
- `.env.example.prod` 不包含真实 API key/provider key/password/session。
- Postgres/Redis 在 prod compose 中不映射公网 ports。
- 文档包含首次上线、更新、回滚、备份和 smoke。

## Technical Approach

- 后端复用现有 `Dockerfile`，镜像名为 `ghcr.io/billionsheep/agent-imageflow-api`。
- 新增 `Dockerfile.web`，使用 Node 构建 `web/dist`，Nginx runtime 监听 `8080` 并支持 SPA fallback。
- 新增 `docker-compose.prod.yml`，通过 `IMAGE_TAG` 选择版本；默认 `main`，发布时可指定 `vX.Y.Z` 或 `sha-xxxxxxx`。
- 生产数据通过 Docker named volumes 或 `AGENT_IMAGEFLOW_*` bind mount 进入 Postgres/Redis/storage。
- 服务器 `.env.prod` 不提交；仓库只提交 `.env.example.prod`。

## Implementation Log

Status: done.

- 新增 `scripts/check_deployment_release.py`，静态校验 prod compose、Web Dockerfile、Nginx config、GitHub Actions 和 prod env 示例的关键约束。
- 新增 `docker-compose.prod.yml`，API/Web 仅绑定 `127.0.0.1`，Postgres/Redis 不暴露 ports，storage 支持 named volume 或 NAS bind mount。
- 新增 `Dockerfile.web` 和 `docker/nginx-web.conf`。
- 新增 `.github/workflows/docker-publish.yml`，PR 不推镜像，main/tag 推 API/Web 镜像到 GHCR。
- 新增 `.env.example.prod`，并调整 `.gitignore` 允许追踪该示例文件。
- 新增 `issues/next-phase-p1-deployment-release-pipeline.csv`。
- 更新 README、RUNBOOK、TASKS、PROJECT_PLAN、CHECKPOINTS、DECISIONS。

## Verification

- `python3 scripts/check_deployment_release.py`
- `docker compose config`
- `docker compose -f docker-compose.prod.yml --env-file .env.example.prod config`
- `npm --prefix web test -- --run`
- `npm --prefix web run build`
- 容器化 `go test ./...`
- `docker build -t agent-imageflow-api:deploy-smoke .`
- `docker build -f Dockerfile.web -t agent-imageflow-web:deploy-smoke .`
- Web image smoke: run container and `curl /`
- API health smoke on existing local stack: `curl http://localhost:8081/healthz`
- MCP stdio smoke on existing local stack: `initialize` -> `tools/list`

本 slice 不读取、打印或处理任何 provider key / API key / secret / cookie / session；不运行真实 provider。
