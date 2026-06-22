# Server Deployment Guide

本文件用于把 Agent ImageFlow 部署到服务器、NAS 或内网 Docker 主机。目标是让服务器只拉取 GitHub Actions 已构建好的 GHCR 私有镜像并运行 Docker Compose，不在服务器构建 Go 或 Web。

## 部署边界

- 镜像仓库：`ghcr.io/billionsheep/agent-imageflow-api` 和 `ghcr.io/billionsheep/agent-imageflow-web`。
- 版本选择：通过 `.env.prod` 中的 `IMAGE_TAG` 指定 `main`、`vX.Y.Z` 或 `sha-xxxxxxx`。
- Secret 位置：真实 `.env.prod` 只放服务器，不提交 Git，不复制到 issue、日志、聊天或 CI。
- Provider key：只作为服务器环境变量进入 `api` / `worker` 容器，不进入镜像、不进入仓库、不进入 GitHub Actions。
- 反向代理：HTTPS、域名、证书由外部 Caddy/Nginx/Traefik/NAS 反代负责，本项目不自动申请证书。
- 数据持久化：Postgres、Redis、asset storage 必须持久化；storage 可使用 Docker volume，也可 bind mount 到 NAS 路径。

## 服务器前置条件

1. Docker Engine 与 Docker Compose v2 可用。
2. 已有域名或内网域名，例如 `https://imageflow.example.com`。
3. 已有 HTTPS 反向代理入口。
4. GitHub 账号或 deploy token 可读取 GHCR private package，权限至少包含 `read:packages`。
5. 准备一处持久化目录，若使用 NAS 例如：

```bash
/volume1/agent-imageflow/storage
/volume1/agent-imageflow/postgres
/volume1/agent-imageflow/redis
```

## 准备部署目录

服务器不需要构建源码，但需要保留生产 compose 和环境文件。可以选择最小复制，也可以 clone 仓库只使用部署文件。

推荐目录：

```bash
sudo mkdir -p /opt/agent-imageflow
sudo chown "$USER":"$USER" /opt/agent-imageflow
cd /opt/agent-imageflow
```

把以下文件放到该目录：

- `docker-compose.prod.yml`
- `.env.example.prod`

然后生成真实环境文件：

```bash
cp .env.example.prod .env.prod
chmod 600 .env.prod
```

用编辑器填写 `.env.prod`。不要用会打印内容的命令展示真实文件。

必须设置：

- `IMAGE_TAG`
- `PUBLIC_BASE_URL`
- `POSTGRES_PASSWORD`
- `DATABASE_URL`
- `ADMIN_USERNAME`
- `ADMIN_PASSWORD`
- `ADMIN_SESSION_SECRET`

按需设置：

- `BASIC_AUTH_USERNAME`
- `BASIC_AUTH_PASSWORD`
- `OPENAI_COMPATIBLE_BASE_URL`
- `OPENAI_COMPATIBLE_API_KEY`
- `OPENAI_COMPATIBLE_MODEL`
- `FAL_API_KEY`
- `AGENT_IMAGEFLOW_STORAGE_ROOT`
- `AGENT_IMAGEFLOW_POSTGRES_DATA`
- `AGENT_IMAGEFLOW_REDIS_DATA`

如果使用 NAS bind mount，示例：

```dotenv
AGENT_IMAGEFLOW_STORAGE_ROOT=/volume1/agent-imageflow/storage
AGENT_IMAGEFLOW_POSTGRES_DATA=/volume1/agent-imageflow/postgres
AGENT_IMAGEFLOW_REDIS_DATA=/volume1/agent-imageflow/redis
```

## 登录 GHCR

在服务器上执行：

```bash
docker login ghcr.io -u <github_username>
```

按提示粘贴只具备 package 读取权限的 token。不要把 token 写进 shell history、文档或日志。

## 首次启动

```bash
cd /opt/agent-imageflow
docker compose -f docker-compose.prod.yml --env-file .env.prod config
docker compose -f docker-compose.prod.yml --env-file .env.prod pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
docker compose -f docker-compose.prod.yml --env-file .env.prod ps
```

`docker-compose.prod.yml` 默认只把 API 和 Web 绑定到宿主机本地：

- API: `127.0.0.1:${API_BIND_PORT:-8081}`
- Web: `127.0.0.1:${WEB_BIND_PORT:-8080}`

Postgres 和 Redis 不暴露宿主机端口。

## 反向代理与同源入口

推荐生产入口只暴露 Web/HTTPS origin。Web 镜像内置 Nginx 会把 `/api/*` 和 `/healthz` 代理到 compose 内部 `api:8081`，因此外部反向代理可以简单转发到 Web 宿主机端口。

简单 Caddy 示例：

```caddyfile
imageflow.example.com {
  encode zstd gzip
  reverse_proxy 127.0.0.1:8080
}
```

如果希望外部反向代理直接分流 API 和 Web，也可以使用下面的高级模式：

高级 Caddy 示例：

```caddyfile
imageflow.example.com {
  encode zstd gzip

  handle /healthz {
    reverse_proxy 127.0.0.1:8081
  }

  handle /api/* {
    reverse_proxy 127.0.0.1:8081
  }

  handle {
    reverse_proxy 127.0.0.1:8080
  }
}
```

`PUBLIC_BASE_URL` 应设置为用户浏览器打开的 Web/HTTPS origin，例如 `https://imageflow.example.com`。Web Settings 里的 Agent ImageFlow API URL 留空时会使用当前 Web origin；高级场景也应填写同一个公开 HTTPS 域名。不要在远程浏览器里保留 `http://localhost:8081`，否则浏览器会访问操作者本机的 localhost。

API 独立端口只给同机反向代理、运维或受控内网使用，不建议直接暴露给浏览器用户。

## Smoke 验收

上线后先只跑 mock，不跑真实 provider。

```bash
curl -fsS https://imageflow.example.com/healthz
curl -fsSI https://imageflow.example.com/
docker compose -f docker-compose.prod.yml --env-file .env.prod ps
```

Admin 登录后重点检查：

- Recent Assets 缩略图能显示，不弹出浏览器原生 Basic Auth 登录框。
- asset original / thumbnail / metadata URL 使用 Web/HTTPS 同源入口。
- Settings 中的 Agent ImageFlow API URL 可留空，且不是 provider base URL。

MCP stdio 只读工具列表：

```bash
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"deploy-smoke"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | docker compose -f docker-compose.prod.yml --env-file .env.prod run -T --rm api /app/mcp
```

Web 验收：

1. 打开 `https://imageflow.example.com`。
2. 使用 Admin Console 登录。
3. 确认 Recent Assets 能打开。
4. 使用 mock provider 创建 1 张图片任务。
5. 确认任务完成后有 original、thumbnail、metadata URL。
6. 确认 Production View / Recent Assets 能看到该资产。

真实 provider canary 只在你明确接受费用时执行：

- `DEFAULT_PROVIDER` 切到对应 provider，或单次任务指定 provider。
- 并发保持低值，例如 provider cap `1`。
- 只生成 1 图。
- 记录 task、asset、batch id。
- 完成后恢复默认配置或保留低并发。

## 更新版本

修改 `.env.prod` 中的 `IMAGE_TAG`：

```dotenv
IMAGE_TAG=v0.1.1
```

然后执行：

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
curl -fsS https://imageflow.example.com/healthz
```

## 回滚

把 `IMAGE_TAG` 改回上一版，例如：

```dotenv
IMAGE_TAG=sha-previous
```

然后执行：

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
curl -fsS https://imageflow.example.com/healthz
```

如果版本包含数据库 schema 变化，不要直接回滚；需要先确认 migration/backup 计划。当前项目还没有独立 migration 框架。

## 备份

上线前、升级前、真实批量生产前建议备份：

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod exec -T postgres \
  pg_dump -U agent -d agent_imageflow > agent_imageflow_$(date +%Y%m%d_%H%M%S).sql
```

同时备份：

- `AGENT_IMAGEFLOW_STORAGE_ROOT` 或 Docker `asset-storage` volume。
- `.env.prod`，单独安全保存。
- NAS 快照，尽量与 Postgres dump 时间接近。

## 常见问题

GHCR 拉取失败：

- 检查是否已 `docker login ghcr.io`。
- 检查 token 是否有 `read:packages` 权限。
- 检查 package 是否对该账号或组织可见。

Web 能打开但 API 不通：

- 检查反向代理是否转发 `/api/*` 和 `/healthz`。
- 检查 Web Settings 的 API URL 是否为公开域名，而不是远程浏览器自己的 `localhost`。
- 检查 `PUBLIC_BASE_URL` 是否与访问域名一致。

Admin 登录后 Recent Assets 为空：

- 检查是否混用 `localhost`、`127.0.0.1` 和域名。
- 检查 Admin cookie 是否在同一 host 下产生。
- 检查 scope/filter 是否过窄。

真实 provider 没有生效：

- 检查 provider key 是否只在服务器 `.env.prod` 中配置。
- 检查 `DEFAULT_PROVIDER` 或任务 provider。
- 检查 worker 是否重启。
- 不要打印 provider key；只检查变量名是否配置。

NAS 路径权限错误：

- 检查 Docker 进程是否能写入 bind mount。
- 检查 storage 目录是否持久化。
- 不要手工修改 DB 中的 asset 状态；文件系统只负责浏览、复制和备份。

## 给新线程的部署提示词

```text
你现在负责把 Agent ImageFlow 部署到服务器/NAS Docker 环境。

请严格遵守：
- 不读取、不打印、不回显任何 API key、provider key、GitHub token、.env.prod、cookie 或 session。
- 服务器只运行 Docker Compose，不在服务器构建 Go/Web。
- 使用 GHCR 私有镜像：ghcr.io/billionsheep/agent-imageflow-api 和 ghcr.io/billionsheep/agent-imageflow-web。
- 按 docs/project/SERVER_DEPLOYMENT_GUIDE.md 执行。
- 先跑 mock smoke，不要默认调用真实 provider。
- 如需真实 provider canary，必须先让我确认费用和 1 图范围。

目标：
1. 准备 /opt/agent-imageflow。
2. 放置 docker-compose.prod.yml 和 .env.prod。
3. docker login ghcr.io。
4. docker compose config / pull / up -d。
5. 配好 HTTPS 反向代理：/api/* 和 /healthz 到 API，其他路径到 Web。
6. 完成 healthz、Web、Admin、mock task、MCP tools/list smoke。
7. 给出最终部署状态、访问地址、版本 tag、容器状态、smoke 结果和剩余风险。
```
