# Agent ImageFlow

面向 AI agent、自动化系统和内容工作流的图片资产生成、落盘、选优、复用与交付平台。

当前仓库已经能把 `Web / MCP / REST / CLI` 收敛到同一套 Go 服务端核心，围绕 `Workspace -> Project -> Campaign -> ImageTask -> Asset` 跑通图片任务、候选资产、选优状态、文件落盘和交付 URL。

## 当前已可验证的能力

- Go API、Worker、CLI、MCP 共用同一套 application core。
- PostgreSQL + Redis + 本地文件系统的服务端资产闭环已跑通。
- `provider=mock` 可用于本地闭环；`provider=openai-compatible` 已支持 `/images/generations`，以及基于 scope `input-files`、匿名远程 URL 和当前项目资产复用的 `/images/edits`；`provider=fal` 已支持 queue 文生图和同一套输入复用下的 edit 路径。
- Web 托管模式可创建服务端 `ImageTask`、轮询候选 `Asset`、查看缩略图/原图并执行 `select/reject`。
- Web 已有独立的 scope 管理入口，可查看 workspace / project / campaign，并执行 rename、归档、删除和“设为当前 scope”。
- 项目级 quality profile、`selection_mode=auto` / `best_of`、`best_of_config` 可插拔评分与可选 auto reject、服务端缩略图 `.webp`、repair/requeue、本地 retry/backoff 已接入。
- 第一版最小鉴权已接入：实例级 Basic Auth + 项目级多 key API key。
- HTTP API 基础限流已接入：支持实例级 / project 级阈值、`429` 结构化错误和 `Retry-After`，默认关闭。
- HTTP / API 第一版结构化审计日志已接入：`/api/*` 请求会写入本地 JSONL 审计事件，并可通过 `vag audit list` 查询。

## 架构收敛方向

```text
Web UI / MCP / REST / CLI
        |
        v
同一个 Go Application Core
        |
        v
ImageTask / Provider Adapter / Asset Registry / Selection / Delivery
        |
        v
PostgreSQL + Redis + Local Storage
```

`web/` 仍保留来自 `GPT Image Playground` 的成熟交互，但正式资产流的事实源已经转到服务端。

## Quickstart

### 1. 启动服务端

```bash
docker compose up -d --build postgres redis api worker
```

服务端默认地址：

- API: `http://localhost:8081`
- 默认 workspace/project/campaign:
  - `ws_default`
  - `prj_xhs_anime`
  - `cmp_7day_cover`

可选健康检查：

```bash
curl http://localhost:8081/healthz
```

### 2. 启动 Web

```bash
npm --prefix web install
npm --prefix web run dev -- --host 0.0.0.0 --port 8080
```

打开：

- Web: `http://localhost:8080`
- API health: `http://localhost:8081/healthz`

### 3. 最小 mock 资产闭环

直接走 REST 创建一条 mock 任务：

```bash
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"README smoke","prompt":"夏日动漫封面图","provider":"mock","requested_count":2,"selection_mode":"auto","review_required":false}'
```

返回里会带 `task_id`。随后查询：

```bash
curl http://localhost:8081/api/tasks/<task_id>
```

任务完成后可看到候选资产、`selected/rejected` 状态、`original_url`、`thumbnail_url` 和 `metadata_url`。

## Demo 路径

### Demo 1: Web 托管模式

1. 打开 `http://localhost:8080`。
2. 进入设置页，开启服务端托管模式。
3. 填入：
   - API URL: `http://localhost:8081`
   - Workspace: `ws_default`
   - Project: `prj_xhs_anime`
   - Campaign: `cmp_7day_cover`
   - Provider: `mock`
4. 提交 prompt，等待任务完成。
5. 在详情页查看候选图，并执行 `select/reject`。
6. 如需调整业务空间，可通过顶栏的 `Scope 管理` 入口独立管理 workspace / project / campaign。

如果你已经启用了项目级 quality profile，Web 托管模式也可以直接复用服务端模板、style preset、reference 参数和 generation config。

### Demo 2: CLI / 运维命令

```bash
docker compose exec api /app/vag task create --file /app/examples/tasks/sample-image-task.json
docker compose exec api /app/vag task get <task_id>
docker compose exec api /app/vag asset approve <asset_id>
docker compose exec api /app/vag project access add-key --name rollout --key <api_key>
docker compose exec api /app/vag audit list --limit 10
docker compose exec api /app/vag repair scan
```

说明：CLI 里仍保留 `approve` 兼容命名，产品语义等价于 `select`。

### Demo 3: MCP stdio smoke

```bash
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"manual-smoke"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | docker compose run -T --rm api /app/mcp
```

当前 MCP tools 已包含：

- `create_image_task`
- `get_image_task`
- `list_image_assets`
- `select_image_asset`
- `reject_image_asset`
- `get_asset_delivery_info`

### Demo 4: 可选真实 edit/mask 路径

如果你要验证服务端真实 edit/mask：

1. 按需配置 OpenAI-compatible 或 fal 环境变量，例如 `OPENAI_COMPATIBLE_*` 或 `FAL_*`。
2. 任选一种输入来源：先上传到当前 scope 的 `input-files`，或直接传匿名远程 URL，或复用当前项目已有 `asset_id`。
3. 再创建服务端任务，服务端会把这些输入统一解析为 provider 可消费的 edit/mask 输入。
4. 仓库已提供 `/app/examples/tasks/sample-fal-task.json` 作为 fal 文生图 smoke 示例。

完整命令见 [docs/project/RUNBOOK.md](docs/project/RUNBOOK.md) 的 `Managed input files / real edit`。

## 自托管最小建议

当前 `docker-compose.yml` 为本地开发友好配置，默认直接暴露 `8081`、`5432`、`6379`。生产或公网环境不要原样暴露。

最小建议：

1. 只对外暴露反向代理入口（通常是 `443`，可附带 `80 -> 443` 跳转）。
2. `api` 放在私有网络或仅监听 `127.0.0.1`。
3. `postgres` 和 `redis` 不对公网暴露；在生产环境去掉端口映射，或仅绑定到内网。
4. 把 `PUBLIC_BASE_URL` 改成用户真实可访问的域名，例如 `https://imageflow.example.com`。
5. 启用实例级 `BASIC_AUTH_USERNAME` / `BASIC_AUTH_PASSWORD`。
6. 给需要隔离的 project 启用 project API key。
7. 持久化 `postgres-data` 与 `asset-storage` 卷，避免重启后丢库或丢图。
8. 如果需要 Web UI，推荐使用生产 Web 镜像；也可以自行构建 `web/dist` 并用静态文件服务或反向代理托管。

### Production image deployment

正式部署推荐使用 GHCR 私有镜像，服务器只拉取镜像运行，不在服务器上构建 Go 或 Web。

默认镜像：

```text
ghcr.io/billionsheep/agent-imageflow-api:${IMAGE_TAG}
ghcr.io/billionsheep/agent-imageflow-web:${IMAGE_TAG}
```

第一次上线：

```bash
docker login ghcr.io
cp .env.example.prod .env.prod
# 编辑 .env.prod，设置 PUBLIC_BASE_URL、DATABASE_URL、Admin、Basic、provider 和 storage 变量
docker compose -f docker-compose.prod.yml --env-file .env.prod pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
docker compose -f docker-compose.prod.yml --env-file .env.prod ps
curl https://imageflow.example.com/healthz
```

版本更新：

```bash
# 修改 .env.prod 中的 IMAGE_TAG，例如 v0.1.1 或 sha-xxxxxxx
docker compose -f docker-compose.prod.yml --env-file .env.prod pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

回滚：

```bash
# 把 IMAGE_TAG 改回上一版
docker compose -f docker-compose.prod.yml --env-file .env.prod pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

部署前后建议备份：

- PostgreSQL dump。
- `asset-storage` / NAS storage root 快照。
- `.env.prod` 单独安全备份。

`docker-compose.prod.yml` 默认只把 API 和 Web 绑定到 `127.0.0.1`，交给宿主机反向代理转发；Postgres、Redis 和 storage root 不应直接暴露到公网。

反向代理需要把 `/api/*` 和 `/healthz` 转到 API，把其他路径转到 Web 镜像；Web Settings 里的 Agent ImageFlow API URL 建议填写同一个公开域名，例如 `https://imageflow.example.com`，避免远程浏览器误连自己的 `localhost:8081`。

启用 Basic Auth 示例：

```bash
BASIC_AUTH_USERNAME=admin \
BASIC_AUTH_PASSWORD=secret \
docker compose up -d --force-recreate api worker
```

项目级 API key 配置与鉴权透传示例见 [docs/project/RUNBOOK.md](docs/project/RUNBOOK.md) 的 `Project API key / Basic Auth`。

### 一个最小的反向代理示例

下面示例假设：

- API 在宿主机 `127.0.0.1:8081`
- Web 镜像在宿主机 `127.0.0.1:8080`

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

如果你不需要公开 Web，只想给自动化系统或 MCP/CLI 使用，也可以只暴露 `/api/*`，把 UI 留在内网。

## 当前已知边界

- `openai-compatible` 和 `fal` 的真实输入复用闭环都已跑通；自定义 HTTP provider、Replicate 等更多 provider 仍未接入。
- best-of 当前已支持 `local_metadata_v1`、可选 `http_judge_v1` 和 `auto_reject_non_selected`；auto rejected 候选仍可手动重新 select。
- 项目级多 key、基础限流和本地审计日志也已接入，当前 MVP 没有未完成的强制产品硬化缺口。

## 文档入口

- [产品规格](docs/project/PRODUCT_SPEC.md)
- [项目计划](docs/project/PROJECT_PLAN.md)
- [技术规格](docs/project/TECH_SPEC.md)
- [输入输出规格](docs/project/INPUT_OUTPUT_SPEC.md)
- [运行手册](docs/project/RUNBOOK.md)
- [服务器部署指导](docs/project/SERVER_DEPLOYMENT_GUIDE.md)
- [检查点](docs/project/CHECKPOINTS.md)
