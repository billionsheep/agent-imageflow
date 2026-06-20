# Runbook

当前项目已导入 `web/` 前端底座，基于 `GPT Image Playground` 二开。

## Current Commands

```bash
# Web 开发
npm --prefix web install
npm --prefix web run dev -- --host 0.0.0.0 --port 8080

# Web 验证
npm --prefix web test -- --run
npm --prefix web run build

# Web production preview（资源占用/日常试用优先用这个判断，避免 Vite dev/HMR 放大）
npm --prefix web run build
npm --prefix web run preview -- --host 127.0.0.1 --port 4173
curl -sf http://127.0.0.1:4173/

# 服务端开发 / smoke
docker compose config
docker compose build
docker compose up
docker compose exec api /app/vag task create --file /app/examples/tasks/sample-image-task.json
docker compose exec api /app/vag task get <task_id>
docker compose exec api /app/vag asset approve <asset_id> # 兼容命令，产品语义等价于 select
docker compose exec api /app/vag project access get
docker compose exec api /app/vag project access set --enabled=true --key <api_key>
docker compose exec api /app/vag project access add-key --name rollout --key <api_key>
docker compose exec api /app/vag project access update-key --id <api_key_id> --enabled=false
docker compose exec api /app/vag project access delete-key --id <api_key_id>
docker compose exec api /app/vag project provider get
docker compose exec api /app/vag project provider set --enabled=true --provider mock --model mock-image
docker compose exec api /app/vag repair scan
docker compose exec api /app/vag repair verify-asset <asset_id>
docker compose exec api /app/vag audit list --limit 20
docker compose exec api /app/vag storage cleanup-preview --workspace ws_default --project prj_xhs_anime --campaign cmp_7day_cover --limit 20
docker compose exec api /app/vag storage cleanup-execute --workspace ws_default --project prj_xhs_anime --campaign cmp_7day_cover --execute --dry-run-token <token>
curl -H 'X-API-Key: <project_key>' http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-governance
curl -H 'X-API-Key: <project_key>' http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-integrity
curl -H 'X-API-Key: <project_key>' 'http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/assets?limit=24&source=mcp&session_id=<session_id>'

# 基础限流配置（默认关闭）
RATE_LIMIT_WINDOW_SECONDS=60
RATE_LIMIT_INSTANCE_MAX_REQUESTS=0
RATE_LIMIT_PROJECT_MAX_REQUESTS=0

# 可选 best-of HTTP judge 配置
BEST_OF_HTTP_SCORER_URL=http://host.docker.internal:8789/score
BEST_OF_HTTP_SCORER_API_KEY=<optional>
BEST_OF_HTTP_SCORER_TIMEOUT_SECONDS=30

# 项目级 quality profile smoke
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/quality-profile \
  -H 'Content-Type: application/json' \
  -d '{"prompt_template":"{{prompt}}，{{channel}} 风格，清爽留白","style_preset":"anime-cover","reference_images":[{"url":"https://example.com/reference.png","role":"style"}],"generation_config":{"quality":"high"}}'

# Web scope smoke
curl http://localhost:8081/api/workspaces
curl -X POST http://localhost:8081/api/workspaces \
  -H 'Content-Type: application/json' \
  -d '{"workspace_id":"ws_scope_smoke","name":"Scope Smoke Workspace"}'
curl -X PATCH http://localhost:8081/api/workspaces/ws_scope_smoke \
  -H 'Content-Type: application/json' \
  -d '{"name":"Scope Smoke Workspace Renamed"}'
curl -X POST http://localhost:8081/api/workspaces/ws_scope_smoke/projects \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"prj_scope_smoke","name":"Scope Smoke Project"}'
curl -X PATCH http://localhost:8081/api/workspaces/ws_scope_smoke/projects/prj_scope_smoke \
  -H 'Content-Type: application/json' \
  -d '{"archived":true}'
curl -X PATCH http://localhost:8081/api/workspaces/ws_scope_smoke/projects/prj_scope_smoke \
  -H 'Content-Type: application/json' \
  -d '{"archived":false}'
curl -X POST http://localhost:8081/api/workspaces/ws_scope_smoke/projects/prj_scope_smoke/campaigns \
  -H 'Content-Type: application/json' \
  -d '{"campaign_id":"cmp_scope_smoke","name":"Scope Smoke Campaign"}'
curl -X PATCH http://localhost:8081/api/workspaces/ws_scope_smoke/projects/prj_scope_smoke/campaigns/cmp_scope_smoke \
  -H 'Content-Type: application/json' \
  -d '{"name":"Scope Smoke Campaign Renamed","archived":true}'
curl -X PATCH http://localhost:8081/api/workspaces/ws_scope_smoke/projects/prj_scope_smoke/campaigns/cmp_scope_smoke \
  -H 'Content-Type: application/json' \
  -d '{"archived":false}'
curl -X DELETE http://localhost:8081/api/workspaces/ws_scope_smoke/projects/prj_scope_smoke/campaigns/cmp_scope_smoke
curl -X DELETE http://localhost:8081/api/workspaces/ws_scope_smoke/projects/prj_scope_smoke
curl -X DELETE http://localhost:8081/api/workspaces/ws_scope_smoke

# best-of 自动选优 smoke
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Best-of smoke","prompt":"自动选优封面图","provider":"mock","requested_count":3,"selection_mode":"auto","review_required":false}'

# HTTP 基础限流 smoke
RATE_LIMIT_WINDOW_SECONDS=60 RATE_LIMIT_INSTANCE_MAX_REQUESTS=2 docker compose up -d --force-recreate api
curl http://localhost:8081/api/workspaces
curl http://localhost:8081/api/workspaces
curl http://localhost:8081/api/workspaces
docker compose up -d --force-recreate api

curl -X POST http://localhost:8081/api/workspaces/ws_default/projects \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"prj_rate_limit_smoke","name":"Rate Limit Smoke"}'
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_rate_limit_smoke/campaigns \
  -H 'Content-Type: application/json' \
  -d '{"campaign_id":"cmp_rate_limit_smoke","name":"Rate Limit Smoke Campaign"}'
RATE_LIMIT_WINDOW_SECONDS=60 RATE_LIMIT_PROJECT_MAX_REQUESTS=1 docker compose up -d --force-recreate api
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_rate_limit_smoke/campaigns/cmp_rate_limit_smoke/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Rate limit smoke 1","prompt":"验证 project 限流","provider":"mock","requested_count":1,"selection_mode":"manual_optional"}'
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_rate_limit_smoke/campaigns/cmp_rate_limit_smoke/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Rate limit smoke 2","prompt":"验证 project 限流","provider":"mock","requested_count":1,"selection_mode":"manual_optional"}'
docker compose up -d --force-recreate api

# HTTP / API 审计日志 smoke
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects \
  -H 'Content-Type: application/json' \
  -d '{"project_id":"prj_audit_smoke","name":"Audit Smoke"}'
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_audit_smoke/campaigns \
  -H 'Content-Type: application/json' \
  -d '{"campaign_id":"cmp_audit_smoke","name":"Audit Smoke Campaign"}'
TASK_ID=$(curl -s -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_audit_smoke/campaigns/cmp_audit_smoke/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Audit smoke","prompt":"验证审计日志","provider":"mock","requested_count":1,"selection_mode":"manual_optional"}' | jq -r .task_id)
curl http://localhost:8081/api/tasks/${TASK_ID}
curl http://localhost:8081/api/tasks/task_missing
docker compose exec api /app/vag audit list --limit 10 --project prj_audit_smoke
docker compose exec api /app/vag audit list --limit 5 --task ${TASK_ID}
docker compose exec api /app/vag audit list --limit 5 --task task_missing --status 404

# project access multi-key smoke
STAMP=$(date +%s)
PRJ=prj_multi_key_${STAMP}
CMP=cmp_multi_key_${STAMP}
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects \
  -H 'Content-Type: application/json' \
  -d "{\"project_id\":\"${PRJ}\",\"name\":\"Multi Key Smoke\"}"
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/${PRJ}/campaigns \
  -H 'Content-Type: application/json' \
  -d "{\"campaign_id\":\"${CMP}\",\"name\":\"Multi Key Smoke Campaign\"}"
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/${PRJ}/access-config \
  -H 'Content-Type: application/json' \
  -d '{"api_key_enabled":true,"api_key_name":"default","api_key":"proj_multi_secret_1111"}'
docker compose exec api /app/vag project access add-key --project ${PRJ} --name rollout --key proj_multi_secret_2222
KEY1_ID=$(curl -s http://localhost:8081/api/workspaces/ws_default/projects/${PRJ}/access-config | jq -r '.access_config.api_keys[] | select(.name=="default") | .id')
KEY2_ID=$(curl -s http://localhost:8081/api/workspaces/ws_default/projects/${PRJ}/access-config | jq -r '.access_config.api_keys[] | select(.name=="rollout") | .id')
TASK_ID=$(curl -s -X POST http://localhost:8081/api/workspaces/ws_default/projects/${PRJ}/campaigns/${CMP}/tasks \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: proj_multi_secret_1111' \
  -d '{"title":"Multi key smoke","prompt":"验证多 key","provider":"mock","requested_count":1,"selection_mode":"manual_optional"}' | jq -r .task_id)
curl -H 'X-API-Key: proj_multi_secret_2222' http://localhost:8081/api/tasks/${TASK_ID}
docker compose exec api /app/vag project access update-key --project ${PRJ} --id ${KEY1_ID} --enabled=false
docker compose exec api /app/vag project access delete-key --project ${PRJ} --id ${KEY1_ID}
docker compose exec api /app/vag audit list --project ${PRJ} --task ${TASK_ID}

# best-of auto reject smoke
docker compose exec api /app/vag task create --file /app/examples/tasks/sample-best-of-auto-reject-task.json

# best-of http_judge_v1 smoke
BEST_OF_HTTP_SCORER_URL=http://host.docker.internal:8789/score docker compose up -d --force-recreate api worker
docker compose exec api /app/vag task create --file /app/examples/tasks/sample-best-of-http-judge-task.json

# Worker retry/backoff smoke
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Retry smoke","prompt":"验证自动重试","provider":"mock","requested_count":1,"selection_mode":"manual_optional","generation_config":{"mock_failure_mode":"transient_once"}}'

# 真实缩略图 smoke
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Thumbnail smoke","prompt":"验证服务端真实缩略图","provider":"mock","requested_count":1,"aspect_ratio":"16:9","selection_mode":"manual_optional"}'

# 高级托管输入 descriptor smoke
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Advanced input smoke","prompt":"带参考图和 mask descriptor 的封面图","provider":"mock","requested_count":2,"selection_mode":"auto","reference_images":[{"id":"web_ref_1","role":"edit_target","source":"web-indexeddb","mime_type":"image/png"}],"mask_image":{"id":"web_mask_1","target_image_id":"web_ref_1","source":"web-mask-draft","mime_type":"image/png","has_mask":true},"generation_config":{"quality":"high"}}'

# 服务端 input-files / edit smoke
curl -u admin:secret -H 'X-API-Key: proj_smoke_secret_1234' \
  -F kind=reference \
  -F file=@/tmp/agent-imageflow-ref.png \
  http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/input-files
curl -u admin:secret -H 'X-API-Key: proj_smoke_secret_1234' \
  -F kind=mask \
  -F file=@/tmp/agent-imageflow-mask.png \
  http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/input-files

# OpenAI-compatible provider 真实 smoke，需要自行配置密钥，可能产生费用
OPENAI_COMPATIBLE_BASE_URL=https://api.openai.com/v1 \
OPENAI_COMPATIBLE_API_KEY=<secret> \
OPENAI_COMPATIBLE_MODEL=gpt-image-2 \
docker compose up -d api worker
docker compose exec api /app/vag task create --file /app/examples/tasks/sample-openai-compatible-task.json

# fal.ai provider 真实 smoke，需要自行配置密钥，可能产生费用
FAL_BASE_URL=https://queue.fal.run \
FAL_REST_BASE_URL=https://rest.fal.ai \
FAL_API_KEY=<secret> \
FAL_MODEL=openai/gpt-image-2 \
docker compose up -d api worker
docker compose exec api /app/vag task create --file /app/examples/tasks/sample-fal-task.json

# remote URL + asset reuse smoke
SEED_TASK_ID=$(curl -s -u admin:secret -H 'X-API-Key: proj_smoke_secret_1234' \
  -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Seed asset","prompt":"给后续 edit 复用的基础图","provider":"mock","requested_count":1,"selection_mode":"manual_optional"}' | jq -r .task_id)
SEED_ASSET_ID=$(curl -s -u admin:secret -H 'X-API-Key: proj_smoke_secret_1234' \
  http://localhost:8081/api/tasks/${SEED_TASK_ID} | jq -r '.asset_ids[0]')
curl -u admin:secret -H 'X-API-Key: proj_smoke_secret_1234' \
  -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d "{\"title\":\"Remote URL + asset reuse smoke\",\"prompt\":\"复用远程参考图和已有资产做 edit\",\"provider\":\"openai-compatible\",\"requested_count\":1,\"reference_images\":[{\"id\":\"remote_ref\",\"url\":\"https://example.com/reference.png\",\"role\":\"style\"},{\"id\":\"asset_ref\",\"asset_id\":\"${SEED_ASSET_ID}\",\"role\":\"edit_target\"}],\"selection_mode\":\"manual_optional\"}"

# MCP stdio smoke
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"manual-smoke"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | docker compose run -T --rm api /app/mcp

# 查看本地变更
git status --short
```

## Repository

- Remote: `git@github.com:billionsheep/agent-imageflow.git`
- Local branch: `main`
- Initial commit 已推送，`main` 跟踪 `origin/main`。

## Local Run Target

服务端当前可通过 Docker Compose 启动：

```bash
docker compose up
```

默认部署目标：

```text
Docker Compose
  api
  worker
  postgres
  redis
  storage volume -> /data/agent-imageflow
```

最小 smoke test：

```bash
docker compose exec api /app/vag task create --file /app/examples/tasks/sample-image-task.json
docker compose exec api /app/vag task get <task_id>
docker compose exec api /app/vag asset approve <asset_id> # 兼容命令，产品语义等价于 select
curl http://localhost:8081/api/assets/<asset_id>
```

MCP 和 Web managed mode 优先使用 `select_image_asset` / `select` 命名；当前 Runbook 保留 `approve` 是为了匹配已实现 CLI。

## Web Performance / Startup

若浏览器提示 `High memory usage`，先区分开发模式和生产模式：

```bash
npm --prefix web test -- --run
npm --prefix web run build
npm --prefix web run preview -- --host 127.0.0.1 --port 4173
curl -sf http://127.0.0.1:4173/
```

当前 P1 Web Performance / Startup 的验证结论：

- Vite dev server 进程约 132-136 MB RSS，production preview 进程约 103 MB RSS；Vite dev/HMR 会放大资源占用，但不是浏览器约 1.1GB 提示的唯一来源。
- Web 启动已避免 React StrictMode 或重复 mount 下重复执行 `initStore` 重活。
- 本地 thumbnail backfill 有后台队列和每次启动处理上限；不会在启动时无限补建历史缩略图。
- 本地 TaskGrid 首屏只挂载有限任务卡，并通过加载更多继续查看历史记录。
- 服务端资产库默认分页/lazy loading，并最多保留 120 个已渲染资产节点。
- fal/custom 历史恢复轮询限制为最多 5 个、6 小时内任务。
- Scope 控制台统计仅在打开时加载，带 60s 缓存、扫描上限和关闭后的结果写回保护。
- Agent workspace、Settings、Scope、Detail、Lightbox、Mask editor、Markdown/KaTeX 样式按需加载。

诊断时不要为了复现性能差异直接清空 IndexedDB、删除资产或执行 storage cleanup。若 production preview 仍复现高内存，再单独做浏览器 heap snapshot、虚拟列表或历史数据规模专项。

## HTTP / API 审计日志

第一版 HTTP / API 审计日志只覆盖 `/api/*` 请求，不包含 `/healthz` 与 `OPTIONS` 预检：

- 落盘目录：`STORAGE_ROOT/audit/http-api/YYYY-MM-DD.jsonl`
- 写入时机：请求返回后
- 当前关键字段：`method`、`path`、`route`、`action`、`status_code`、`duration_ms`、`auth_mode`、`actor`、`workspace_id`、`project_id`、`campaign_id`、`task_id`、`asset_id`、`input_file_id`、`error_code`、`error_message`
- 查询入口：`docker compose exec api /app/vag audit list --limit 20`

常见过滤示例：

```bash
docker compose exec api /app/vag audit list --project prj_xhs_anime --limit 20
docker compose exec api /app/vag audit list --task <task_id>
docker compose exec api /app/vag audit list --asset <asset_id>
docker compose exec api /app/vag audit list --status 404
```

## Storage governance

P1 Storage Governance 已经接入只读存储统计、清理候选 dry-run、受控本地 CLI 清理执行和只读 integrity 摘要。

只读 REST 统计接口：

```bash
curl -H 'X-API-Key: <project_key>' \
  http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-governance
```

只读 REST integrity 摘要：

```bash
curl -H 'X-API-Key: <project_key>' \
  http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-integrity
```

`storage-integrity` 会返回当前 scope 内的 `missing_file`、`empty_file`、`invalid_current_version`、`stale_queued`、`stale_running`、`enqueue_failed` 摘要；响应不包含本地绝对路径，文件问题只暴露 `file_kind`。

返回内容包含：

- `usage.instance` / `usage.workspace` / `usage.project` / `usage.campaign`
- `original`、`thumbnail`、`metadata`、`input_files`、`audit`、`tmp`、`orphan` 等分类的 `bytes` 与 `file_count`
- `counts.*.task_count`、`failed_task_count`、`asset_count`、`generated_asset_count`、`selected_asset_count`、`rejected_asset_count`、`published_asset_count`

安全边界：

- 该接口不接受任意本地路径参数。
- 该接口不返回本地绝对路径，只返回 scope id、计数和字节数。
- 路由沿用当前 project/campaign 的 project API key 规则；启用 project key 后，未带 key 会返回 `401`。

本地 dry-run 预览：

```bash
docker compose exec api /app/vag storage cleanup-preview \
  --workspace ws_default \
  --project prj_xhs_anime \
  --campaign cmp_7day_cover \
  --limit 20
```

说明：

- `cleanup-preview` 是只读 dry-run，不删除文件，不更新数据库。
- 默认候选包括 `rejected` 资产、`generated/draft` 未选中资产、临时文件和 orphan final files。
- `selected/approved` 与 `published` 默认 protected，不进入清理候选；响应只返回 protected 计数。
- 文件明细使用 storage root 下的相对 `storage_key`，不暴露宿主机绝对路径。

受控本地执行：

```bash
docker compose exec api /app/vag storage cleanup-execute \
  --workspace ws_default \
  --project prj_xhs_anime \
  --campaign cmp_7day_cover \
  --limit 20 \
  --execute \
  --dry-run-token <token_from_cleanup_preview> \
  --actor <operator_name>
```

执行边界：

- 没有 `--execute` 时禁止删除。
- 带 `--dry-run-token` 时必须匹配当前 dry-run 候选集；token 不匹配即使带 `--confirm` 也拒绝。
- 无 token 的本地执行必须同时传 `--execute --confirm`，用于明确人工确认；建议常规操作仍先使用 dry-run token。
- 第一版只提供 CLI 执行入口，不暴露匿名远程清理 REST。
- 默认只清理 `rejected`、`generated/draft` 未选中资产、`tmp` 和明确 orphan files；`selected/approved`、`published`、`deprecated` 默认 protected。
- 资产清理会先在数据库事务内删除 `review_event` / `delivery_event` / `asset_version` / `asset` 行，再删除对应 storage files；若文件删除失败，执行报告会标记失败，数据库不会继续引用已清理资产。
- 每次执行或拒绝执行都会写入本地 audit，`source=cli`、`action=storage_cleanup_execute`。

查看清理审计：

```bash
docker compose exec api /app/vag audit list \
  --workspace ws_default \
  --project prj_xhs_anime \
  --campaign cmp_7day_cover \
  --action storage_cleanup_execute \
  --limit 20
```

## Self-hosting Minimum

`docker-compose.yml` 当前保留开发友好的端口映射：

- `8081:8081`
- `5432:5432`
- `6379:6379`

这适合本地启动和 smoke，但不应原样作为公网部署拓扑。

生产或公网自托管的最小建议：

1. 只对外暴露反向代理入口，例如 `443`。
2. `api` 放在私有网络、容器网络，或仅绑定 `127.0.0.1`。
3. `postgres`、`redis` 不对公网开放；如需宿主机排障，也应只绑定到内网或临时开放。
4. 设置 `PUBLIC_BASE_URL=https://your-domain.example`，保证 asset/original/thumbnail/metadata URL 返回正确公网地址。
5. 持久化 `postgres-data` 和 `asset-storage`，不要把图片与数据库写到临时盘。
6. 启用实例级 Basic Auth，并按 project 启用 API key。
7. 如果要开放 Web UI，构建 `web/dist` 后由静态文件服务或反向代理托管。

一个最小反向代理思路：

```text
Internet
  -> TLS reverse proxy
      -> /api/* -> Agent ImageFlow API
      -> /      -> web/dist 静态文件（可选）
```

Caddy 示例：

```caddyfile
imageflow.example.com {
  encode zstd gzip

  handle /api/* {
    reverse_proxy 127.0.0.1:8081
  }

  root * /srv/agent-imageflow/web/dist
  try_files {path} /index.html
  file_server
}
```

如果当前实例只给 MCP、CLI、自动化脚本或内网后台使用，可以不公开 Web UI，只反代 `/api/*`。

## Project API key / Basic Auth

服务端当前支持两层最小鉴权：

- 实例级 Basic Auth：通过 `BASIC_AUTH_USERNAME` / `BASIC_AUTH_PASSWORD` 保护整个 HTTP 入口。
- 项目级 API key：通过 `project.metadata_json.access_config` 保存兼容视图和 `api_keys` 列表，可同时维护多把命名 key。

Docker Compose 启用示例：

```bash
BASIC_AUTH_USERNAME=admin BASIC_AUTH_PASSWORD=secret docker compose up -d --force-recreate api worker
```

配置 project key：

```bash
curl -u admin:secret -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/access-config \
  -H 'Content-Type: application/json' \
  -d '{"api_key_enabled":true,"api_key_name":"smoke","api_key":"proj_smoke_secret_1234"}'
```

读取 project key 配置（不会返回明文 key）：

```bash
curl -u admin:secret http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/access-config
```

新增第二把 key：

```bash
curl -u admin:secret -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/access-config \
  -H 'Content-Type: application/json' \
  -d '{"action":"add_key","api_key_name":"rollout","api_key":"proj_rollout_secret_5678"}'
```

禁用或删除某把 key：

```bash
curl -u admin:secret -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/access-config \
  -H 'Content-Type: application/json' \
  -d '{"action":"update_key","api_key_id":"<api_key_id>","api_key_enabled":false}'

curl -u admin:secret -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/access-config \
  -H 'Content-Type: application/json' \
  -d '{"action":"delete_key","api_key_id":"<api_key_id>"}'
```

带鉴权创建任务：

```bash
curl -u admin:secret -H 'X-API-Key: proj_smoke_secret_1234' \
  -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Auth smoke","prompt":"带鉴权的服务端任务","provider":"mock","requested_count":1,"selection_mode":"auto","review_required":false}'
```

CLI 透传环境变量：

```bash
AGENT_IMAGEFLOW_BASIC_USER=admin \
AGENT_IMAGEFLOW_BASIC_PASS=secret \
AGENT_IMAGEFLOW_API_KEY=proj_smoke_secret_1234 \
docker compose exec -T api /app/vag task get <task_id>
```

CLI 多 key 管理示例：

```bash
docker compose exec -T api /app/vag project access add-key --project prj_xhs_anime --name rollout --key proj_rollout_secret_5678
docker compose exec -T api /app/vag project access update-key --project prj_xhs_anime --id <api_key_id> --enabled=false
docker compose exec -T api /app/vag project access delete-key --project prj_xhs_anime --id <api_key_id>
```

注意：

- 当前 scope 管理接口属于实例级管理能力。
- 如果启用了 `BASIC_AUTH_USERNAME` / `BASIC_AUTH_PASSWORD`，这些接口只要求 Basic Auth。
- workspace / project / campaign 的 rename、archive/unarchive、delete 与 list/create 一样，都属于实例级管理能力。
- 当前不要求 project API key 来列出或创建 workspace/project/campaign；更细权限控制留给后续 hardening。
- `input-files` 接口属于 project/campaign 级资源；如果 project API key 已启用，上传、读取 metadata 和读取 content 都要求 project API key。
- 当前不会追踪单把 key 的 usage/last_used；轮换和清理依赖管理员自行确认。

## Managed input files / real edit

第一版服务端可访问输入文件路径通过当前 scope 下的 `input-files` 接口提供：

```bash
POST /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/input-files
GET  /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/input-files/{input_file_id}
GET  /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/input-files/{input_file_id}/content
```

行为说明：

- 上传使用 `multipart/form-data`，字段名固定为 `file`，可选 `kind=reference|mask`。
- 服务端把文件落到当前 scope 的本地存储，并返回 `input_file_id`、MIME、尺寸、metadata URL 和 content URL。
- 创建任务时，`reference_images[].input_file_id` 和 `mask_image.input_file_id` 会在服务端解析为内部 `resolved_input_files`。
- 当前第一版还支持两类额外输入来源：匿名 `http/https` 远程 URL 会在创建任务时抓取并物化到当前 scope 的 `input-files`；同 workspace/project 下已有的 `asset_id` 会复用其原图文件作为输入。
- 远程 URL 解析成功后，任务快照会同时保留原始 `url` 和服务端生成的 `input_file_id`；资产复用会保留原始 `asset_id`。
- 当前 `openai-compatible` provider 在存在已解析输入文件时会走 `/images/edits` multipart；没有已解析输入文件时继续走 `/images/generations`。
- 当前 `fal` provider 在存在已解析输入文件时会先把本地文件上传到 fal storage，再走 queue `/edit`；没有已解析输入文件时继续走 queue 文生图 endpoint。
- 输入文件删除、配额治理和带鉴权远程抓取仍留给后续 slice。

本地 HTTP mock smoke：

```bash
# 1. 在宿主机启动本地 openai-compatible edit mock
python3 /path/to/local/mock-server.py

# 2. 用 Docker Compose 指向 mock
BASIC_AUTH_USERNAME=admin \
BASIC_AUTH_PASSWORD=secret \
OPENAI_COMPATIBLE_BASE_URL=http://host.docker.internal:18081 \
OPENAI_COMPATIBLE_API_KEY=test-key \
OPENAI_COMPATIBLE_MODEL=image-model \
docker compose up -d --force-recreate api worker

# 3. 上传 reference / mask
REF_ID=$(curl -s -u admin:secret -H 'X-API-Key: proj_smoke_secret_1234' \
  -F kind=reference \
  -F file=@/tmp/agent-imageflow-ref.png \
  http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/input-files | jq -r .input_file_id)

MASK_ID=$(curl -s -u admin:secret -H 'X-API-Key: proj_smoke_secret_1234' \
  -F kind=mask \
  -F file=@/tmp/agent-imageflow-mask.png \
  http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/input-files | jq -r .input_file_id)

# 4. 创建 edit 任务
curl -u admin:secret -H 'X-API-Key: proj_smoke_secret_1234' \
  -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d "{\"title\":\"Edit smoke\",\"prompt\":\"带遮罩编辑的服务端任务\",\"provider\":\"openai-compatible\",\"requested_count\":1,\"reference_images\":[{\"id\":\"web_ref_1\",\"input_file_id\":\"${REF_ID}\",\"role\":\"edit_target\"}],\"mask_image\":{\"input_file_id\":\"${MASK_ID}\",\"target_image_id\":\"web_ref_1\",\"has_mask\":true},\"selection_mode\":\"manual_optional\",\"review_required\":false}"

# 5. 先生成一个可复用 asset，再用 remote URL + asset_id 创建第二条 edit 任务
SEED_TASK_ID=$(curl -s -u admin:secret -H 'X-API-Key: proj_smoke_secret_1234' \
  -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Seed asset","prompt":"为 asset reuse 生成基础图","provider":"mock","requested_count":1,"selection_mode":"manual_optional","review_required":false}' | jq -r .task_id)

SEED_ASSET_ID=$(curl -s -u admin:secret -H 'X-API-Key: proj_smoke_secret_1234' \
  http://localhost:8081/api/tasks/${SEED_TASK_ID} | jq -r '.asset_ids[0]')

curl -u admin:secret -H 'X-API-Key: proj_smoke_secret_1234' \
  -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d "{\"title\":\"Remote URL + asset reuse edit smoke\",\"prompt\":\"复用远程参考图和已有资产的服务端任务\",\"provider\":\"openai-compatible\",\"requested_count\":1,\"reference_images\":[{\"id\":\"remote_ref\",\"url\":\"http://host.docker.internal:8787/remote.png\",\"role\":\"style\"},{\"id\":\"asset_ref\",\"asset_id\":\"${SEED_ASSET_ID}\",\"role\":\"edit_target\"}],\"selection_mode\":\"manual_optional\",\"review_required\":false}"

# 6. 切到 fal mock，验证同一套 remote URL + asset reuse 输入复用
BASIC_AUTH_USERNAME=admin \
BASIC_AUTH_PASSWORD=secret \
FAL_BASE_URL=http://host.docker.internal:8788/queue \
FAL_REST_BASE_URL=http://host.docker.internal:8788/rest \
FAL_API_KEY=test-key \
FAL_MODEL=openai/gpt-image-2 \
FAL_POLL_INTERVAL_MS=100 \
docker compose up -d --force-recreate api worker

curl -u admin:secret -H 'X-API-Key: proj_smoke_secret_1234' \
  -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d "{\"title\":\"fal Remote URL + asset reuse smoke\",\"prompt\":\"复用远程参考图和已有资产的 fal edit 任务\",\"provider\":\"fal\",\"requested_count\":1,\"reference_images\":[{\"id\":\"remote_ref\",\"url\":\"http://host.docker.internal:8788/remote.png\",\"role\":\"style\"},{\"id\":\"asset_ref\",\"asset_id\":\"${SEED_ASSET_ID}\",\"role\":\"edit_target\"}],\"selection_mode\":\"manual_optional\",\"review_required\":false}"
```

## Repair / reconcile

本地维护命令：

```bash
docker compose exec api /app/vag repair scan
docker compose exec api /app/vag repair scan --limit 20 --stale-after 1h
docker compose exec api /app/vag repair requeue <task_id>
docker compose exec api /app/vag repair requeue --dry-run <task_id>
docker compose exec api /app/vag repair verify-asset <asset_id>
```

说明：

- `repair scan` 是只读操作，输出结构化 JSON。
- `repair requeue` 可将 `enqueue_failed`、长时间 `queued`、长时间 `running` 的任务重新标记为 `queued` 并写入 Redis queue。
- `repair verify-asset` 检查当前版本的 original、thumbnail、metadata 文件是否存在且非空。
- 该能力直接读取 `DATABASE_URL`、`REDIS_URL`、`STORAGE_ROOT`，是本地自托管维护命令，不暴露为 REST/MCP 管理接口。

第一版 smoke：

```bash
docker compose up -d postgres redis api
docker compose stop worker

# 创建一个任务后，模拟入队失败/队列丢失
TASK_ID=$(curl -s -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Repair smoke task","prompt":"repair smoke image","provider":"mock","requested_count":1,"selection_mode":"manual_optional","review_required":false}' \
  | jq -r .task_id)
docker compose exec redis redis-cli DEL queue:image_generation
docker compose exec postgres psql -U agent -d agent_imageflow \
  -c "UPDATE generation_task SET status='enqueue_failed', error_code='enqueue_failed', error_message='repair smoke simulated enqueue failure', updated_at=now() WHERE id='${TASK_ID}'"

docker compose exec api /app/vag repair scan --limit 20 --stale-after 1h
docker compose exec api /app/vag repair requeue "${TASK_ID}"
docker compose up -d worker
docker compose exec api /app/vag task get "${TASK_ID}"
```

## Worker retry / backoff

Worker 当前会先 promote Redis delayed queue，再消费主队列 `queue:image_generation`。

相关环境变量：

```bash
WORKER_MAX_RETRIES=3
WORKER_RETRY_BASE_DELAY_SECONDS=15
```

行为说明：

- 仅对 provider 瞬时失败做自动重试，例如超时、429、5xx、`temporary_unavailable`。
- 每次失败都会写一条 `task_attempt`，并在需要重试时写入 `retry_after`。
- 任务状态会回到 `queued`，保留最近一次 provider 错误信息，随后由 delayed queue 自动重试。
- 超过最大重试次数后，任务进入 `failed`。
- 当前不对资产落盘/缩略图处理失败做自动重试。

smoke：

```bash
TASK_ID=$(curl -s -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Retry smoke task","prompt":"验证自动重试","provider":"mock","requested_count":1,"selection_mode":"manual_optional","generation_config":{"mock_failure_mode":"transient_once"}}' \
  | jq -r .task_id)

watch -n 2 "curl -s http://localhost:8081/api/tasks/${TASK_ID} | jq '{task_id,status,error_code,updated_at}'"

docker compose exec postgres psql -U agent -d agent_imageflow -P pager=off \
  -c \"select attempt_no,status,error_code,retry_after from task_attempt where task_id='${TASK_ID}' order by attempt_no;\"
```

## Worker concurrency / provider reliability / benchmark

当前 Docker Compose 默认使用 6 个 worker goroutine，但真实 provider 默认 cap 更保守：

```bash
WORKER_CONCURRENCY=6
OPENAI_COMPATIBLE_MAX_CONCURRENCY=3
FAL_MAX_CONCURRENCY=3
PROVIDER_TIMEOUT_SECONDS=300
OPENAI_COMPATIBLE_CONNECT_TIMEOUT_SECONDS=30
OPENAI_COMPATIBLE_RESPONSE_HEADER_TIMEOUT_SECONDS=300
OPENAI_COMPATIBLE_TOTAL_TIMEOUT_SECONDS=300
```

含义：

- `WORKER_CONCURRENCY` 控制 Worker 同时消费 Redis 任务的 goroutine 数。
- `OPENAI_COMPATIBLE_MAX_CONCURRENCY` 控制同时进入 `provider=openai-compatible` 的请求数；设为 `0` 可禁用该 cap。
- `FAL_MAX_CONCURRENCY` 控制同时进入 `provider=fal` 的请求数；设为 `0` 可禁用该 cap。
- `PROVIDER_TIMEOUT_SECONDS` 仍作为旧配置兼容项，并作为 openai-compatible header/total timeout 的默认回退。
- `OPENAI_COMPATIBLE_CONNECT_TIMEOUT_SECONDS` 区分连接阶段；`OPENAI_COMPATIBLE_RESPONSE_HEADER_TIMEOUT_SECONDS` 区分等待 headers/首字节阶段；`OPENAI_COMPATIBLE_TOTAL_TIMEOUT_SECONDS` 控制整体 provider 调用上限。
- 当前平台代码层面没有固定只能 1 的硬上限；实际可用并发需要结合 provider 429/timeout、PostgreSQL、Redis 和本地 storage 压测确认。

切换本地 worker 并发：

```bash
WORKER_CONCURRENCY=6 OPENAI_COMPATIBLE_MAX_CONCURRENCY=2 docker compose up -d --build worker
docker compose logs --tail=20 worker
```

查看 task attempts：

```bash
docker compose exec api /app/vag task attempts "${TASK_ID}"
curl -s "http://localhost:8081/api/tasks/${TASK_ID}/attempts" | jq
```

attempts 中的诊断字段：

| 字段 | 含义 |
| --- | --- |
| `queue_wait_ms` | task 创建到当前 attempt 开始之间的等待时间 |
| `provider_first_byte_ms` | openai-compatible 从发起请求到首字节/headers 的耗时 |
| `provider_total_ms` | provider adapter 调用总耗时；拆分请求时会累计 |
| `response_download_ms` | 读取 provider JSON body 或结果下载阶段耗时 |
| `store_ms` | 资产文件、metadata 和缩略图写入的总耗时 |
| `thumbnail_ms` | 缩略图生成耗时 |
| `retry_count` | 当前 attempt 前已经重试过的次数 |
| `error_stage` | `connect`、`provider_first_byte`、`provider_total`、`response_download`、`response_parse`、`store` 等诊断阶段 |
| `response_bytes` | provider 响应体大小，主要用于排查异常大响应 |

mock benchmark 不产生 provider 费用：

```bash
docker compose exec api /app/vag benchmark image-generation \
  --provider mock \
  --tasks 32 \
  --requested-count 1 \
  --mock-delay-ms 250 \
  --poll-interval 250ms \
  --timeout 120s \
  --concurrency-label worker-4-delay250
```

benchmark 会写入 `metadata_json.session_id` 和 `metadata_json.batch_id`，值等于 `--run-id`。运行后可以按批次查看进度：

```bash
docker compose exec api /app/vag batch progress \
  --project prj_xhs_anime \
  --campaign cmp_7day_cover \
  --session-id bench_p1_provider_rel_batch \
  --batch-id bench_p1_provider_rel_batch \
  --limit 100
```

真实 provider benchmark 可能产生费用，默认会被 CLI 拒绝。确认小样本后再显式开启：

```bash
docker compose exec api /app/vag benchmark image-generation \
  --provider openai-compatible \
  --tasks 8 \
  --requested-count 1 \
  --poll-interval 2s \
  --timeout 30m \
  --concurrency-label worker-2-provider-cap-2 \
  --allow-paid-provider
```

推荐压测顺序：

| 阶段 | Worker | Provider cap | 任务数 | 目标 |
| --- | ---: | ---: | ---: | --- |
| mock 基线 | 1 | n/a | 32 | 验证平台串行 queue wait |
| mock 并发 | 4 | n/a | 32 | 验证平台自身吞吐 |
| real 小样本 | 1 | 1 | 8 | 真实 provider 单请求耗时基线 |
| real 小样本 | 2 | 2 | 8 | 推荐起步并发 |
| real 小样本 | 3 | 3 | 8 | 当前默认建议档，观察 timeout/429 |
| real 小样本 | 4 | 4 | 8 | 仅在 cap=3 稳定时尝试 |

推荐判定：

- 成功率 `>= 90%`。
- timeout `<= 10%`。
- 无持续 `429`。
- 若 timeout 或 429 超过 `25%`，停止更高并发档。

本地 mock 结果（2026-06-19）：

| 场景 | 任务 | requested_count | mock_delay_ms | wall-clock | P95 queue wait | 结果 |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| worker=1 | 32 | 1 | 250 | 12.427s | 10.987s | 32/32 completed |
| worker=4 | 32 | 1 | 250 | 2.979s | 2.464s | 32/32 completed |
| worker=4 | 8 | 4 | 250 | 1.318s | 0.527s | 8/8 completed |
| worker=6 | 32 | 1 | 1000 | 8.204s | 4.534s | 32/32 completed |
| worker=6 | 16 | 4 | 1000 | 4.112s | 2.564s | 16/16 completed |
| worker=6 | 32 | 4 | 2000 | 14.239s | 9.138s | 32/32 completed |
| P1 provider reliability | 3 | 2 | 50 | 0.221s | 0.007s | 3/3 completed，6 assets，0 retry/timeout |

真实 provider 小样本结果（2026-06-19，prompt 为萌宠，临时 `WORKER_MAX_RETRIES=0` 防止失败自动重复调用）：

| 场景 | 任务 | requested_count | Provider cap | wall-clock | P50 task | P95 task | 成功率 | timeout |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| openai-compatible worker=6 | 6 | 1 | 6 | 120.628s | 56.250s | 120.024s | 4/6 | 2/6 |

资源峰值采样：

| 容器 | CPU 峰值 | 内存峰值 |
| --- | ---: | ---: |
| api | 0.92% | 26.48MiB |
| worker | 1.08% | 50.60MiB |
| postgres | 2.17% | 99.86MiB |
| redis | 4.05% | 13.36MiB |

结论：平台自身在 mock worker=6 下吞吐正常，真实生图 worker=6/provider cap=6 时本机 CPU/内存仍很低；瓶颈在 provider 侧，当前 provider 6 并发出现 33.3% timeout，不满足成功率 `>= 90%`、timeout `<= 10%` 的推荐标准。因此运行默认已调整为 worker=6、provider cap=3、provider total timeout=300s。真实 provider 后续建议按 cap `2 -> 3 -> 4` 小样本确认稳定档，不要直接把 worker 并发等同于 provider 并发。

## Real thumbnail output

服务端当前会基于原图统一生成缩略图，而不是直接保存 provider 返回的 thumbnail bytes。

相关环境变量：

```bash
THUMBNAIL_MAX_WIDTH=720
THUMBNAIL_MAX_HEIGHT=720
```

行为说明：

- 缩略图最终固定输出到 `thumbnails/{asset_id}/1.webp`。
- 缩略图会保持原图比例，并受最大宽高约束。
- `GET /api/assets/{asset_id}/thumbnail` 返回 `image/webp`。
- Docker runtime image 已内置 `cwebp`；如果直接在宿主机运行二进制，需要确保 PATH 中存在同名命令。

smoke：

```bash
TASK_ID=$(curl -s -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Thumbnail smoke task","prompt":"验证服务端真实缩略图","provider":"mock","requested_count":1,"aspect_ratio":"16:9","selection_mode":"manual_optional"}' \
  | jq -r .task_id)

ASSET_ID=""
for _ in $(seq 1 20); do
  ASSET_ID=$(curl -s "http://localhost:8081/api/tasks/${TASK_ID}" | jq -r 'if .status=="completed" or .status=="partially_completed" then (.asset_ids[0] // "") else "" end')
  [ -n "${ASSET_ID}" ] && break
  sleep 1
done
test -n "${ASSET_ID}"

curl -I "http://localhost:8081/api/assets/${ASSET_ID}/thumbnail"

docker compose exec postgres psql -U agent -d agent_imageflow -P pager=off -t -A \
  -c "select thumbnail_path from asset_version where asset_id='${ASSET_ID}' order by created_at desc limit 1;"
```

## Web managed mode

Web 当前保留 legacy playground mode，同时新增服务端托管模式。

启动服务端：

```bash
docker compose up -d postgres redis api worker
```

启动 Web：

```bash
npm --prefix web run dev -- --host 0.0.0.0 --port 8080
```

在 Web 设置页的“习惯配置”中开启“服务端托管模式”，默认配置为：

```text
API URL: http://localhost:8081
Project API Key: (optional)
Basic 用户名: (optional)
Basic 密码: (optional)
Workspace: ws_default
Project: prj_xhs_anime
Campaign: cmp_7day_cover
Provider: mock
Use project quality profile: on
Selection mode: auto
```

开启后：

- Web 会优先尝试从服务端同步已有 workspace / project / campaign，并在设置页里通过下拉选择当前 scope。
- 如果没有合适的业务空间，可以直接在设置页里快速新建 workspace / project / campaign；创建成功后会自动切换到新 scope。
- 输入框提交 prompt 会创建服务端 `ImageTask`，不走浏览器直连 provider。
- 如果输入框带参考图或 mask，Web 会把本地图片 ID、来源、MIME、角色和 mask target 作为 descriptor 提交到服务端；原图/mask 二进制仍留在浏览器 IndexedDB。
- 如果“使用项目质量配置”开启，创建任务时会传 `use_project_quality_profile=true`，服务端会应用项目级 prompt template / style preset / reference 参数 / generation config。
- 托管模式默认传 `selection_mode=auto`；多候选任务完成后，服务端会按任务输入或项目级 quality profile 中的 `best_of_config` 自动 selected 一张候选。
- 如果服务端启用了 Basic Auth 或项目级 API key，Web 会自动附带 `Authorization: Basic ...` 和 `X-API-Key`。
- Web 会轮询 `GET /api/tasks/{task_id}` 并展示服务端候选资产。
- 任务详情页可以对当前候选资产执行 Select / Reject。
- Original / Metadata 按钮会打开服务端 delivery URL。

限制：

- Web 已有独立 scope 管理 modal，可 rename/archive/delete 并设为当前 scope；但还没有完整的项目级 quality profile / scorer 配置 UI。
- Web 托管模式当前会优先把 reference image / mask 上传到服务端 `input-files`，再由 OpenAI-compatible 或 fal 消费统一的 `resolved_input_files`；浏览器本地二进制路径仅保留给 legacy playground mode。
- `best_of_config` 已可透传到服务端，但 Web 侧暂未提供显式的 scorer 策略或 auto reject 开关控件；自动 selected 和 auto rejected 候选仍可以被用户手动改选。
- REST 底层当前仍使用 `/approve` 兼容入口；Web 展示语义映射为 `selected`。

## Advanced managed input

服务端任务输入支持：

```json
{
  "reference_images": [
    {
      "id": "web_ref_1",
      "role": "edit_target",
      "source": "web-indexeddb",
      "mime_type": "image/png"
    }
  ],
  "mask_image": {
    "id": "web_mask_1",
    "target_image_id": "web_ref_1",
    "source": "web-mask-draft",
    "mime_type": "image/png",
    "has_mask": true
  },
  "generation_config": {
    "quality": "high"
  }
}
```

这些字段会进入 `generation_task.structured_input_json`，并在 provider 生成的 asset version `parameters_json` 中保留快照。当前 mock provider、OpenAI-compatible provider 和 fal provider 都会记录这些参数；当存在已解析输入文件时，OpenAI-compatible 会走 `/images/edits` multipart，fal 会走 storage upload + queue `/edit`。

## Best-of auto selection

创建任务时可传：

```json
{
  "requested_count": 3,
  "selection_mode": "auto",
  "best_of_config": {
    "strategy": "http_judge_v1",
    "judge_prompt": "优先选择更适合作为内容封面图的候选",
    "auto_reject_non_selected": true
  }
}
```

支持值：

- `manual_optional`: 默认值；候选保持 generated，由用户、agent 或外部系统 select/reject。
- `auto`: Worker 生成并登记候选后自动 selected 一张推荐图。
- `best_of`: 与 `auto` 等价，保留给更明确的调用语义。

当前 scorer：

- `local_metadata_v1`: 默认策略，使用服务端本地 metadata：版本 ready、图片面积、目标比例接近度和 hash 稳定排序。
- `http_judge_v1`: 可选外部 judge；服务端会把候选缩略图、任务信息和可选 `judge_prompt` 组织成结构化 JSON 发送到外部 HTTP endpoint。
- `auto_reject_non_selected`: 可选开关；若为 `true`，服务端会在自动 selected 后把其他候选标记为 rejected。

自动选择事件写入 `review_event`，`reviewer=auto-best-of`；note 会记录 `requested_strategy`、`applied_strategy`，外部 scorer 失败时还会带 fallback 信息。若 `http_judge_v1` 不可用或调用失败，服务端会自动回退 `local_metadata_v1`。若开启 `auto_reject_non_selected`，未入选候选会自动写入 rejected review_event，但后续仍可通过 `POST /api/assets/{id}/approve` 或 MCP `select_image_asset` 手动重新 selected。

### `http_judge_v1` scorer

启用配置：

```bash
BEST_OF_HTTP_SCORER_URL=http://host.docker.internal:8789/score
BEST_OF_HTTP_SCORER_API_KEY=<optional>
BEST_OF_HTTP_SCORER_TIMEOUT_SECONDS=30
```

请求体会包含：

- `strategy`
- `judge_prompt`
- `task`
- `candidates`

其中 `candidates` 默认携带服务端缩略图生成的 `data:` URL；若缩略图缺失，则回退发送原图。响应可以直接返回 `selected_asset_id`，也可以返回带分数的 `scores[]`。仓库示例任务见 `/app/examples/tasks/sample-best-of-http-judge-task.json`。

## Asset list query

服务端资产库接口保持返回 asset array，旧客户端可以继续读取；P1 起支持筛选和分页：

```bash
curl -H 'X-API-Key: <project_key>' \
  'http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/assets?limit=24&status=selected&source=mcp&session_id=session_001&batch_id=batch_001&keyword=cover'
```

支持参数：

- `limit`: 默认 `50`，最大 `100`。
- `offset`: 用于加载更多。
- `status`: `generated` / `selected` 会兼容映射到当前内部 `draft` / `approved`，也接受 `rejected` / `published` 等内部状态。
- `provider` / `model`: 按当前 asset version 过滤。
- `source` / `session_id` / `batch_id`: 按 `generation_task.structured_input_json.metadata_json` 过滤。
- `keyword`: 在 asset id、task id、asset name、prompt 和 task title 中搜索。
- `created_from` / `created_to`: RFC3339 时间。

Web 服务端资产库第一屏显式传 `limit=24`，图片使用 lazy loading，并通过“加载更多”追加读取。select/reject 后只更新当前 asset，不需要全量重拉。

## Quality profile

项目级质量配置保存在 `project.metadata_json.quality_profile`，当前通过 REST 读取和更新：

```bash
curl http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/quality-profile

curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/quality-profile \
  -H 'Content-Type: application/json' \
  -d '{
    "prompt_template": "{{prompt}}，{{channel}} 风格，清爽留白，统一封面视觉",
    "negative_prompt": "low quality, blurry, watermark",
    "style_preset": "anime-cover",
    "reference_images": [{"url": "https://example.com/reference.png", "role": "style", "weight": 0.8}],
    "generation_config": {"quality": "high", "seed_strategy": "stable"},
    "best_of_config": {"strategy": "local_metadata_v1", "auto_reject_non_selected": true}
  }'
```

创建任务时启用复用：

```json
{
  "prompt": "普通人如何用 AI 做第一张动漫头像",
  "use_project_quality_profile": true,
  "metadata_json": {
    "channel": "xiaohongshu"
  }
}
```

服务端会先渲染 `prompt_template`，再把有效配置快照写入 `structured_input_json.metadata_json.quality_profile_snapshot`。

## Provider profile

项目级 provider profile 保存在 `project.metadata_json.provider_profile`，第一版只保存非敏感默认值：

```json
{
  "enabled": true,
  "provider": "openai-compatible",
  "model": "gpt-image-2",
  "base_url": "https://api.openai.com/v1",
  "generation_config": {
    "quality": "high"
  },
  "use_project_quality_profile": true,
  "max_n": 4,
  "supports_url_result": true,
  "preferred_response_format": "url",
  "max_concurrency": 3,
  "timeout_seconds": 300
}
```

读取和更新：

```bash
curl -H 'X-API-Key: <project_key>' \
  http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/provider-profile

curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/provider-profile \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: <project_key>' \
  -d '{"enabled":true,"provider":"mock","model":"mock-image","generation_config":{"quality":"high"},"use_project_quality_profile":true,"max_n":4,"preferred_response_format":"url"}'

docker compose exec api /app/vag project provider set \
  --provider mock \
  --model mock-image \
  --generation-config '{"quality":"high"}' \
  --max-n 4 \
  --preferred-response-format url
```

边界：

- 不保存或回显真实 provider secret。
- 未配置 profile 时继续使用服务端环境变量中的默认 provider。
- 创建任务没有显式 `provider` 时，服务端会优先使用启用的项目 provider profile。
- `provider_profile.model` 当前可覆盖 `openai-compatible` 的 model 和 `fal` 的 endpoint id；`base_url` 第一版只作为非敏感项目默认配置保存，真实 endpoint/key 存储策略需要单独确认。
- `max_n` 表示单次 provider 请求建议承载的同 prompt 变体数，默认 4，服务端上限 10；`requested_count` 超过 `max_n` 时会拆成多次 provider 请求并保留同一个 task。
- `supports_url_result`、`preferred_response_format`、`max_concurrency`、`timeout_seconds` 是项目经验配置，不代表 provider 一定支持；openai-compatible adapter 默认 URL 优先，会省略 `response_format`，仅在显式配置 `preferred_response_format=b64_json` 时请求 Base64 响应。

## Codex batch asset production examples

Agent ImageFlow 不读取故事脚本、不拆分内容、不发布小红书，也不维护内容日历。Codex、MCP client、REST 脚本或其他外部编排工具应先把脚本拆成图片任务，再把标准 `metadata_json` 传进来。

萌宠账号示例：

```bash
docker compose exec api /app/vag task create \
  --project prj_pet_account \
  --campaign cmp_pet_story_batch_001 \
  --file /app/examples/tasks/sample-codex-pet-story-task.json
```

嵌入式文章插图示例：

```bash
docker compose exec api /app/vag task create \
  --project prj_embedded_diagrams \
  --campaign cmp_rtos_sensor_pipeline \
  --file /app/examples/tasks/sample-codex-embedded-architecture-task.json
```

MCP `create_image_task` 的核心参数形状：

```json
{
  "workspace_id": "ws_default",
  "project_id": "prj_pet_account",
  "campaign_id": "cmp_pet_story_batch_001",
  "prompt": "一只圆眼睛的小橘猫在窗边看雨，画面温暖、干净、适合小红书萌宠故事封面。",
  "provider": "mock",
  "requested_count": 2,
  "selection_mode": "auto",
  "metadata_json": {
    "source": "codex",
    "source_agent": "codex-cli",
    "source_thread_id": "thread_pet_story_demo",
    "session_id": "pet_story_session_2026_06_19",
    "run_id": "run_scene_batch_001",
    "batch_id": "pet_story_batch_001",
    "story_id": "rainy_window_cat",
    "scene_id": "scene_001",
    "target_path": "assets/pet-story/rainy-window-cat/scene-001.png"
  }
}
```

生成后可按批次查询：

```bash
curl -H 'X-API-Key: <project_key>' \
  'http://localhost:8081/api/projects/prj_pet_account/campaigns/cmp_pet_story_batch_001/assets?limit=24&source=codex&session_id=pet_story_session_2026_06_19&batch_id=pet_story_batch_001'
```

批量执行中可先看进度，不必等所有任务完成：

```bash
curl -H 'X-API-Key: <project_key>' \
  'http://localhost:8081/api/projects/prj_pet_account/campaigns/cmp_pet_story_batch_001/batch-progress?session_id=pet_story_session_2026_06_19&batch_id=pet_story_batch_001&limit=100'

docker compose exec api /app/vag batch progress \
  --project prj_pet_account \
  --campaign cmp_pet_story_batch_001 \
  --session-id pet_story_session_2026_06_19 \
  --batch-id pet_story_batch_001
```

## OpenAI-compatible provider

服务端 Worker 当前支持第一个真实 provider adapter：

```text
provider=openai-compatible
```

配置项：

```bash
OPENAI_COMPATIBLE_BASE_URL=https://api.openai.com/v1
OPENAI_COMPATIBLE_API_KEY=<secret>
OPENAI_COMPATIBLE_MODEL=gpt-image-2
OPENAI_COMPATIBLE_MAX_CONCURRENCY=3
PROVIDER_TIMEOUT_SECONDS=300
OPENAI_COMPATIBLE_CONNECT_TIMEOUT_SECONDS=30
OPENAI_COMPATIBLE_RESPONSE_HEADER_TIMEOUT_SECONDS=300
OPENAI_COMPATIBLE_TOTAL_TIMEOUT_SECONDS=300
```

说明：

- 默认不启用真实 provider；未配置 base URL 或 API key 时，`provider=openai-compatible` 会在创建任务时返回明确错误。
- adapter 调用 `{OPENAI_COMPATIBLE_BASE_URL}/images/generations`，默认省略 `response_format` 并解析 `data[].url` 或 `data[].b64_json`；显式配置 `preferred_response_format=b64_json` 时会请求 Base64 响应。
- 当任务存在已解析 reference/mask 输入时，adapter 调用 `{OPENAI_COMPATIBLE_BASE_URL}/images/edits`。
- adapter 会从 `generation_config` 白名单透传 `quality`、`moderation`、`output_compression`；当前不透传 `stream` / `partial_images`。
- 返回图片会在服务端规范化为 PNG，再进入现有 asset processor / storage / delivery。
- 自动化验证使用本地 HTTP mock，不会触发真实外部 API。真实 smoke 需要用户自行配置密钥，并自行承担 provider 成本。

## fal.ai provider

服务端 Worker 当前也支持：

```text
provider=fal
```

配置项：

```bash
FAL_BASE_URL=https://queue.fal.run
FAL_REST_BASE_URL=https://rest.fal.ai
FAL_API_KEY=<secret>
FAL_MODEL=openai/gpt-image-2
FAL_POLL_INTERVAL_MS=1000
FAL_MAX_CONCURRENCY=3
PROVIDER_TIMEOUT_SECONDS=300
```

说明：

- 未配置 `FAL_API_KEY` 时，`provider=fal` 会在创建或执行任务时返回明确错误。
- 无输入图时，adapter 调用 `${FAL_BASE_URL}/{model}` 的 queue 文生图 endpoint。
- 有已解析输入图时，adapter 会先向 `${FAL_REST_BASE_URL}/storage/upload/initiate?storage_type=fal-cdn-v3` 申请上传，再把本地 resolved input 上传到 fal storage，最后调用 `${FAL_BASE_URL}/{model}/edit`。
- queue 完成后，服务端会下载结果图并统一规范化为 PNG，再进入现有 asset processor / storage / delivery。
- 第一版采用 queue + rest storage HTTP 协议，不引入 Go SDK 新依赖；本地 Docker smoke 已验证 queue `/edit`、storage upload 和 remote URL + `asset_id` 复用输入闭环。

## Ports

- Web dev server: `http://localhost:8080`
- API: `http://localhost:8081`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`
- Worker: 无 HTTP 端口，消费 Redis 队列 `queue:image_generation`。
- Delayed retry queue: Redis sorted set `queue:image_generation:scheduled`。

## MCP stdio

当前 Docker image 内包含 MCP server binary：

```bash
docker compose run -T --rm api /app/mcp
```

MCP server 通过 stdin/stdout 收发换行分隔 JSON-RPC。它会复用与 API 相同的环境变量、PostgreSQL、Redis、默认 workspace/project/campaign 和本地存储配置。

已暴露 tools：

```text
create_image_task
get_image_task
list_image_assets
select_image_asset
reject_image_asset
get_asset_delivery_info
```

`select_image_asset` 在底层调用当前兼容的 approve 状态迁移，但 MCP 输出会把 `approved` 映射为产品语义 `selected`，把 `draft` 映射为 `generated`。

## Debug Notes

- 如果接云端 provider，必须避免提交 API key。
- 如果未来接 ComfyUI，需记录本地 ComfyUI URL、模型要求和输出目录；MVP 不接本地 GPU。
- MCP 已支持 stdio 本地模式；后续如果要做远程 MCP，再单独评估 Streamable HTTP。
- 本地 Web 开发当前会生成未忽略的 `.vite/` 目录；它属于运行态缓存，不影响当前 slice 验收，但后续应补 ignore/清理规则，避免污染 `git status`。
- Docker runtime 现在依赖 `libwebp-tools` 提供 `cwebp` 生成 `.webp` 缩略图；若直接运行本地 Go 二进制，需要自行安装同等命令。
- 本机 shell 当前没有 `go` 命令；Go 测试和格式化可通过 Docker 执行：

```bash
docker run --rm -v "$PWD":/src -w /src golang:1.25.3-alpine sh -lc '/usr/local/go/bin/gofmt -w cmd internal && /usr/local/go/bin/go test ./...'
```
