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
docker compose exec api /app/vag project context get
docker compose exec api /app/vag project context set --file /app/examples/tasks/sample-project-visual-context.json
docker compose exec api /app/vag repair scan
docker compose exec api /app/vag repair verify-asset <asset_id>
docker compose exec api /app/vag audit list --limit 20
docker compose exec api /app/vag storage cleanup-preview --workspace ws_default --project prj_xhs_anime --campaign cmp_7day_cover --limit 20
docker compose exec api /app/vag storage cleanup-execute --workspace ws_default --project prj_xhs_anime --campaign cmp_7day_cover --execute --dry-run-token <token>
curl -H 'X-API-Key: <project_key>' http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-governance
curl -H 'X-API-Key: <project_key>' http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-integrity
curl -H 'X-API-Key: <project_key>' 'http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/assets?limit=24&source=mcp&session_id=<session_id>'
curl -H 'X-API-Key: <project_key>' 'http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/batch-summary?session_id=<session_id>&batch_id=<batch_id>&limit=100'

# Project Visual Context smoke（mock provider，无外部费用）
STAMP=$(date +%s)
PRJ=prj_pctx_smoke_${STAMP}
CMP=cmp_pctx_smoke_${STAMP}
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects \
  -H 'Content-Type: application/json' \
  -d "{\"project_id\":\"${PRJ}\",\"name\":\"PCTX Smoke Project\"}"
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/${PRJ}/campaigns \
  -H 'Content-Type: application/json' \
  -d "{\"campaign_id\":\"${CMP}\",\"name\":\"PCTX Smoke Campaign\"}"
docker compose exec -T api /app/vag project context set \
  --project ${PRJ} \
  --file /app/examples/tasks/sample-project-visual-context.json
docker compose exec -T api /app/vag task create \
  --project ${PRJ} \
  --campaign ${CMP} \
  --file /app/examples/tasks/sample-pet-story-visual-context-task.json

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

## Local packaged review 标准路径

`V02-MCPH-010` 第一版把“本地审图复放”固定成一条标准命令链，不再默认依赖 Vite dev/HMR 或临时手工猜步骤：

```bash
# 1. 起 API / worker（默认只跑 mock）
docker compose up -d postgres redis api worker
curl -sf http://127.0.0.1:8081/healthz

# 2. 起 Web packaged review
npm --prefix web run build
npm --prefix web run preview -- --host 127.0.0.1 --port 4173
curl -sf http://127.0.0.1:4173/
```

Admin 临时账号边界：

- 只通过本地环境变量、受控 `.env` 或部署环境注入 `ADMIN_USERNAME` / `ADMIN_PASSWORD`。
- 不把真实账号、密码、cookie、session 写进脚本、文档、示例 JSON 或聊天记录。
- 本地 preview 推荐始终用同一个 host 打开 Web 和 API，例如统一使用 `127.0.0.1`，避免 Admin cookie 因 `localhost` / `127.0.0.1` 分裂。

Replay 指南：

1. 打开 `http://127.0.0.1:4173` 并完成 Admin 登录。
2. 先用 REST 或 CLI 确认目标批次：
   - `GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-summary?session_id=<session_id>&batch_id=<batch_id>&story_id=<story_id>&limit=100`
   - `GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-manifest?session_id=<session_id>&batch_id=<batch_id>&story_id=<story_id>&selected_only=true`
   - `GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-manifest?session_id=<session_id>&batch_id=<batch_id>&story_id=<story_id>&selected_only=true&view=final_delivery`
3. 在 Web Recent Assets 中填同一组 `session_id` / `batch_id` / `story_id` 过滤条件，确认 selected assets 与 manifest 对齐。
4. 从任一资产卡进入 Production View，核对 scene continuity、selected 状态、`delivery_role`、`asset_summary` 和 manifest replay；如需给 PM/运营/NAS 复盘，优先再导出一次 `Final delivery manifest`。

这条路径的目标是“让新会话稳定复放审图证据”，不是新增 Web 创作入口，也不是把 story/batch 发布组织内建到平台存储结构里。

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

## Web UX Smoothness

用户反馈按钮点击时若出现屏幕闪烁，优先用 production preview 复现，而不是 Vite dev/HMR：

```bash
npm --prefix web test -- --run
npm --prefix web run build
npm --prefix web run preview -- --host 127.0.0.1 --port 4173
curl -I http://127.0.0.1:4173/
curl -sf http://localhost:8081/healthz
```

当前 P1 Web UX Smoothness P1-UX-001 到 P1-UX-009 已完成：

- 服务端资产库只订阅 Agent ImageFlow 相关 settings 字段，避免无关设置变化触发资产列表重拉。
- Recent Assets / Scope 资产刷新、普通错误和 scope incomplete 路径不会再直接 `setAssets([])` 把已有列表闪空；刷新中会保留旧列表并显示状态。
- provider/source/session/batch/keyword 文本筛选使用 300ms debounce；旧请求通过 request 序号忽略，避免晚返回覆盖新结果。
- 资产卡 `Scope` 操作会一次性写入必要 workspace/project/campaign 字段，不再把整份 settings 展开写回。
- Settings 托管 scope selector 和手动兜底 ID 输入只在本地 draft 中保留不完整 workspace/project/campaign；只有三段完整后才提交全局 settings，关闭 Settings 时会保留上一份完整 scope。
- Scope 管理 modal 的“设为当前托管 scope”只写入 `imageflowManagedMode`、workspace、project 和 campaign 字段。
- Scope 管理 modal 的层级请求和 dashboard stats 请求已分离；层级列表先渲染，stats 延迟 180ms 后台启动，关闭或刷新时会取消/忽略旧 stats 写回，资产统计扫描显式使用 `limit=24`。
- AgentWorkspace、Detail、Lightbox、Settings、ScopeManager、MaskEditor 的 lazy `Suspense` fallback 不再为 `null`，首次加载会显示稳定 overlay/skeleton。
- Header Settings/Scope/Agent、TaskGrid/AgentWorkspace 任务卡、InputBar 图片预览/Mask/无配置提交入口会在 hover/focus/pointerdown 时预加载对应 chunk。
- `TaskCard` 使用 `React.memo` 并只订阅 `settings.alwaysShowRetryButton`；`TaskGrid` 的单卡事件由 memo 化 `TaskGridItem` 稳定；服务端资产库使用 memo 化 `ServerAssetCard` 和稳定 select/reject/copy/scope callbacks，减少单卡操作牵动整页卡片重绘。
- production preview/browser 回归已完成：Settings 打开不触发 `/api/*` 请求；Scope 管理打开保持主框架可见，首轮只观察到 `/api/workspaces`；Recent + 同步保持 root/library 可见，请求数量受控。

仍待后续：

- 本专项已关闭。若手测仍复现闪烁，应记录具体按钮、当前 URL host、Admin 登录状态、筛选条件、mode、scope 和是否正在刷新资产，再新建针对性 follow-up。
- 若 Recent Assets 显示 `unauthorized`，优先检查 Web URL host 与 API/admin session cookie host 是否一致，以及是否已用同一 host 登录 Admin Console；不要把该状态误判为历史图片丢失或资产库空列表。

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
  --batch-id <optional_batch_id> \
  --session-id <optional_session_id> \
  --limit 20
```

说明：

- `cleanup-preview` 是只读 dry-run，不删除文件，不更新数据库。
- 默认候选包括 `rejected` 资产、`generated/draft` 未选中资产、临时文件和 orphan final files；`deprecated/archived` 资产默认保留，用于支持单资产归档后的恢复。
- 如需物理清理已归档资产，必须显式传 `--deprecated`（REST payload 为 `include_deprecated=true`），并先确认这些资产不需要恢复。
- `selected/approved` 与 `published` 默认 protected，不进入清理候选；响应只返回 protected 计数。
- 文件明细使用 storage root 下的相对 `storage_key`，不暴露宿主机绝对路径。
- 可选过滤：`--asset-id`、`--task-id`、`--session-id`、`--batch-id`、`--story-id`。这些过滤只限制当前 scope 内候选，不允许跨 workspace/project/campaign 清理。

受控本地执行：

```bash
docker compose exec api /app/vag storage cleanup-execute \
  --workspace ws_default \
  --project prj_xhs_anime \
  --campaign cmp_7day_cover \
  --batch-id <optional_batch_id> \
  --session-id <optional_session_id> \
  --limit 20 \
  --execute \
  --dry-run-token <token_from_cleanup_preview> \
  --actor <operator_name>
```

执行边界：

- 没有 `--execute` 时禁止删除。
- 带 `--dry-run-token` 时必须匹配当前 dry-run 候选集；token 不匹配即使带 `--confirm` 也拒绝。
- 无 token 的本地执行必须同时传 `--execute --confirm`，用于明确人工确认；建议常规操作仍先使用 dry-run token。
- 第一版提供 CLI 和 Admin-only REST 执行入口，不暴露匿名远程清理 REST，也不向 MCP 暴露 hard delete。
- 默认只清理 `rejected`、`generated/draft` 未选中资产、`tmp` 和明确 orphan files；`archived/deprecated` 只有显式 `--deprecated` / `include_deprecated=true` 时才会进入物理清理候选；`selected/approved` 与 `published` 默认 protected。
- 资产清理会先在数据库事务内删除 `review_event` / `delivery_event` / `asset_version` / `asset` 行，再删除对应 storage files；若文件删除失败，执行报告会标记失败，数据库不会继续引用已清理资产。
- 每次执行或拒绝执行都会写入本地 audit，`source=cli`、`action=storage_cleanup_execute`。

Admin REST dry-run 预览：

```bash
curl -X POST \
  -H 'Content-Type: application/json' \
  -b '<admin_session_cookie>' \
  http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-cleanup-preview \
  -d '{"batch_id":"<optional_batch_id>","limit":20}'
```

Admin REST 受控执行：

```bash
curl -X POST \
  -H 'Content-Type: application/json' \
  -b '<admin_session_cookie>' \
  http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-cleanup-execute \
  -d '{"batch_id":"<same_batch_id>","limit":20,"execute":true,"dry_run_token":"<token_from_preview>"}'
```

REST 边界：

- 需要 Admin session；不要把 Admin cookie、cleanup token 或任何 key 写进项目文档、MCP 配置或聊天记录。
- Project API Key 继续服务外部 MCP/REST/CLI 正常生图和查资产，不作为 cleanup 执行凭据。
- REST cleanup 与 CLI 使用同一候选和保护规则；执行前仍建议先做 Postgres dump 与 storage root 快照。

查看清理审计：

```bash
docker compose exec api /app/vag audit list \
  --workspace ws_default \
  --project prj_xhs_anime \
  --campaign cmp_7day_cover \
  --action storage_cleanup_execute \
  --limit 20
```

### Scope 级联删除边界

Scope 管理里的 workspace / project / campaign 删除现在是 Admin 受控级联删除，不再要求 scope 为空：

- 删除 `campaign` 会清理该 campaign 下的 task、task attempt、asset、asset version、review/delivery event 和对应 storage scope 目录。
- 删除 `project` 会递归清理下级 campaign，再删除 project scope storage。
- 删除 `workspace` 会递归清理下级 project/campaign，再删除 workspace scope storage。
- 这是“删除整个业务空间/测试空间”的生命周期动作。用户确认后，`selected/approved/published` 资产也会随 scope 删除。
- 资产级 `storage-cleanup-preview/execute` 仍默认保护 `selected/approved/published`；不要把这两条链路混用。
- MCP 仍不提供 workspace/project/campaign/asset 硬删除工具；agent 如需清理，只能标记 `reject_image_asset` 或请求人类/运维通过 Admin Web/REST/CLI 处理。
- 执行真实删除前建议先备份 Postgres 与 storage root/NAS 快照；浏览器确认框会提示影响范围，但它不是备份机制。

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

## Production Image Deployment

正式部署推荐使用 GitHub Actions 发布到 GHCR 的私有镜像，服务器只拉取镜像运行，不在服务器构建 Go 或 Web。

如果要把任务交给新线程或服务器运维执行，优先使用独立交接文档：`docs/project/SERVER_DEPLOYMENT_GUIDE.md`。

当前 V1 后的部署演练入口是：

```text
issues/next-phase-p1-server-deployment-rehearsal.csv
docs/project/stories/slice-053-server-deployment-rehearsal.md
```

本轮已准备部署演练工单和证据模板；真实服务器/NAS 上线仍需在目标环境执行。默认先跑 mock smoke，不运行真实 provider；如果需要 1 图真实 provider canary，必须先单独确认费用、provider、scope 和停止条件。演练证据只记录 `IMAGE_TAG`、服务状态、health/Web/Admin/mock/MCP/备份/回滚结果，不记录 `.env.prod`、GHCR token、provider key、project key、Basic/Auth/Admin cookie 或 session。

默认镜像：

```text
ghcr.io/billionsheep/agent-imageflow-api:${IMAGE_TAG}
ghcr.io/billionsheep/agent-imageflow-web:${IMAGE_TAG}
```

生产 compose：

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
docker compose -f docker-compose.prod.yml --env-file .env.prod ps
```

第一次上线步骤：

1. 在服务器执行 `docker login ghcr.io`，使用只具备 `read:packages` 权限的 GitHub token 或 deploy token。
2. 从 `.env.example.prod` 复制出 `.env.prod`，只在服务器编辑真实值。
3. 设置 `IMAGE_TAG`，可用 `main`、`vX.Y.Z` 或 `sha-xxxxxxx`。
4. 设置 `PUBLIC_BASE_URL=https://your-domain.example`，推荐指向 Web/HTTPS 公开入口，而不是 API 独立端口。
5. 设置 `DATABASE_URL` 与 `POSTGRES_PASSWORD`；如果密码包含 URL 特殊字符，`DATABASE_URL` 中需要 URL encode。
6. 设置 `ADMIN_USERNAME`、`ADMIN_PASSWORD`、`ADMIN_SESSION_SECRET`。
7. 按需设置 `BASIC_AUTH_USERNAME`、`BASIC_AUTH_PASSWORD` 和 project API key。
8. 按需设置 `OPENAI_COMPATIBLE_*` 或 `FAL_*`；未设置 key 时不会默认调用真实 provider。
9. 如使用 NAS，把 `AGENT_IMAGEFLOW_STORAGE_ROOT` 设置为宿主机/NAS 路径；留空则使用 Docker named volume。
10. 执行 pull/up，并通过反向代理开放 HTTPS。

生产浏览器入口推荐保持同源：

- Web 镜像已代理 `/api/*` 与 `/healthz` 到 compose 内部 API；外部反向代理可以只把公开域名转发到 Web 宿主机端口。
- `PUBLIC_BASE_URL` 应与用户浏览器打开的 Web origin 一致，保证 thumbnail/original/metadata URL 不跳到 API 独立端口。
- API 宿主机端口只给同机反向代理或运维使用，不应作为日常 Web 用户入口。

如果外部反向代理选择绕过 Web 镜像直接分流，也必须保持这个边界：

- `/api/*` 转发到 API，例如宿主机 `127.0.0.1:8081`。
- `/healthz` 转发到 API，便于上线 smoke 和外部健康检查。
- 其他路径转发到 Web 镜像，例如宿主机 `127.0.0.1:8080`。

Web Settings 里的 Agent ImageFlow API URL 留空时会默认使用当前 Web origin。高级场景可以填写同一个公开 HTTPS 域名，例如 `https://your-domain.example`，不要在远程浏览器里保留 `http://localhost:8081`；否则浏览器会访问操作者本机的 localhost，而不是服务器。

版本更新步骤：

```bash
# 1. 修改 .env.prod 中的 IMAGE_TAG，例如 v0.1.1 或 sha-xxxxxxx
# 2. 拉取并重建容器
docker compose -f docker-compose.prod.yml --env-file .env.prod pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d

# 3. 验证
curl -fsS https://your-domain.example/healthz
docker compose -f docker-compose.prod.yml --env-file .env.prod ps
```

回滚步骤：

```bash
# 把 IMAGE_TAG 改回上一版
docker compose -f docker-compose.prod.yml --env-file .env.prod pull
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

生产 smoke 建议：

```bash
curl -fsS https://your-domain.example/healthz
curl -fsSI https://your-domain.example/
docker compose -f docker-compose.prod.yml --env-file .env.prod exec api /app/vag audit list --limit 5
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"deploy-smoke"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | docker compose -f docker-compose.prod.yml --env-file .env.prod run -T --rm api /app/mcp
```

备份建议：

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod exec -T postgres \
  pg_dump -U agent -d agent_imageflow > agent_imageflow_$(date +%Y%m%d_%H%M%S).sql
```

- 同时对 `asset-storage` 或 `AGENT_IMAGEFLOW_STORAGE_ROOT` 做一致快照。
- `.env.prod` 单独安全备份，不提交 Git。
- 恢复时要让 Postgres dump 和 storage root 尽量来自同一时间点。

当前不引入数据库 migration 框架；后续如果某个版本包含 schema 变化，必须先另开 migration/backup 计划。

## Project API key / Basic Auth / Admin Console

服务端当前支持两层最小鉴权：

- 实例级 Basic Auth：通过 `BASIC_AUTH_USERNAME` / `BASIC_AUTH_PASSWORD` 保护整个 HTTP 入口。
- 项目级 API key：通过 `project.metadata_json.access_config` 保存兼容视图和 `api_keys` 列表，可同时维护多把命名 key。
- 轻量 Admin Console session：通过 `ADMIN_USERNAME` / `ADMIN_PASSWORD` 登录 Web 控制台，使用 HttpOnly cookie 查看控制台安全资源；未单独配置时可回退复用 Basic Auth 用户名和密码。

Docker Compose 启用示例：

```bash
BASIC_AUTH_USERNAME=admin BASIC_AUTH_PASSWORD=secret \
ADMIN_USERNAME=admin ADMIN_PASSWORD=secret \
docker compose up -d --force-recreate api worker
```

Admin session 可选环境变量：

```text
ADMIN_USERNAME=admin
ADMIN_PASSWORD=<console_password>
ADMIN_SESSION_SECRET=<long_random_secret>
ADMIN_SESSION_TTL_SECONDS=43200
```

说明：

- Web 控制台现在采用前置 Admin 登录页：未登录时只显示登录页，不展示 Header、InputBar、资产库、Production View、Project Context 或 Settings 主体。
- Admin session 只用于 Web 控制台和管理读取路径，不替代 project API key。
- Provider API key 固定在服务端环境变量中，不返回给 Web、不写入 localStorage、不进入响应 JSON。
- MCP / CLI / REST 外部 project 级调用继续使用 project API key，也可按需叠加 Basic Auth。
- 如果启用了 Admin credentials，scope 管理、Recent Assets 和控制台读取路径需要 Admin session 或 Basic Auth。
- Web 登录者使用的是服务器配置好的平台能力，不需要也不应该在 Web 里填写 provider key 或 provider base URL。
- 本地浏览器手测时不要混用 `127.0.0.1` Web origin 和 `localhost` API base；Admin cookie 绑定 host。生产/preview 推荐使用同一个 Web origin，让 `/api/*` 和图片 delivery 都走同源入口。
- 本地 Vite preview/dev 常用端口 `4173` / `5173` 没有 Web 镜像里的 Nginx `/api` 代理，Web 留空 API URL 时会按当前页面 host 自动回退到 `http://127.0.0.1:8081` 或 `http://localhost:8081`；生产 Web 镜像或正式反代仍保持同源 `/api`。
- 已登录 Admin session 可读取 asset thumbnail/original/metadata；未登录仍返回 401，但图片类 delivery 不返回会触发浏览器原生 Basic Auth 弹窗的 challenge。
- 如果用户看到旧的 provider/base URL 配置，它属于高级/旧模式兼容路径；正式资产生产优先走服务端托管模式和服务器配置的 provider。

Runtime status smoke：

```bash
curl -s -b /tmp/agent-imageflow-admin.cookie \
  http://localhost:8081/api/admin/runtime-status
```

说明：

- 该接口只返回 Admin 已登录状态下可见的非敏感摘要：API build/version/commit/image tag、provider mode/model、Admin/Basic 是否配置、限流/并发摘要。
- 本地开发或旧镜像缺失 build metadata 时会显示 `unknown`，不影响使用。
- Web 控制台会同时显示 Web build 和 API build；如果 commit 不一致，应优先确认浏览器入口、镜像 tag 和服务器 compose 是否都已更新。
- 响应不得包含 provider key、project key、Basic/Auth 密码、cookie、session、本地绝对路径或完整 secret。

Admin login smoke：

```bash
curl -i -c /tmp/agent-imageflow-admin.cookie \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"secret"}' \
  http://localhost:8081/api/admin/login

curl -b /tmp/agent-imageflow-admin.cookie \
  http://localhost:8081/api/admin/me

curl -b /tmp/agent-imageflow-admin.cookie \
  'http://localhost:8081/api/admin/assets/recent?limit=24'
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
- 如果启用了 `ADMIN_USERNAME` / `ADMIN_PASSWORD`，Web 控制台优先使用 Admin session；Basic Auth 仍可作为实例级管理调用方式。
- workspace / project / campaign 的 rename、archive/unarchive、delete 与 list/create 一样，都属于实例级管理能力。
- 当前不要求 project API key 来列出或创建 workspace/project/campaign；更细权限控制留给后续 hardening。
- `input-files` 接口属于 project/campaign 级资源；如果 project API key 已启用，上传、读取 metadata 和读取 content 都要求 project API key。
- 当前不会追踪单把 key 的 usage/last_used；轮换和清理依赖管理员自行确认。

## Project Visual Context

P1 Project Production Context 第一版使用 `project.metadata_json.visual_context`，不新增数据库表，也不保存 provider secret。它只服务 project 级长期视觉生产上下文：

- `characters`: 角色/主形象卡，包含 `id/name/status/role/appearance/personality/forbidden/primary_asset_id/reference_asset_ids`。
- `references`: 复用已有 asset 的 reference binding，包含 `asset_id/purpose/label/weight/notes/character_id/status`；删除或归档 binding 不删除原 asset。
- `prompt_recipes`: prompt recipe，包含 `prompt_blocks/negative_prompt/default_aspect_ratio/default_output_format/default_provider/default_model/generation_config`。

REST 入口：

```bash
curl http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/visual-context
curl -X POST http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/visual-context \
  -H 'Content-Type: application/json' \
  -d '{"visual_context":{"characters":[],"references":[],"prompt_recipes":[]}}'
```

CLI 入口：

```bash
docker compose exec -T api /app/vag project context get
docker compose exec -T api /app/vag project context set --file /app/examples/tasks/sample-project-visual-context.json
docker compose exec -T api /app/vag task create --file /app/examples/tasks/sample-pet-story-visual-context-task.json
docker compose exec -T api /app/vag task create --file /app/examples/tasks/sample-pet-story-visual-context-scene-002.json
docker compose exec -T api /app/vag task create --file /app/examples/tasks/sample-pet-story-visual-context-scene-003.json
```

任务创建时可传：

```json
{
  "character_ids": ["dog_mochi"],
  "reference_asset_ids": ["asset_existing"],
  "prompt_recipe_id": "pet_story_cover",
  "use_project_visual_context": true
}
```

服务端会在 `CreateTask` 阶段展开角色、reference 和 recipe，写入 `structured_input_json.visual_context_snapshot`，并让 asset `parameters_json.visual_context_snapshot` 保留关键快照。显式任务字段优先于 recipe/project 默认值；跨 workspace/project 的 `asset_id` 会被拒绝。

示例文件：

- `examples/tasks/project-visual-context-usage.md`: CLI / REST / MCP 使用说明。
- `examples/tasks/sample-project-visual-context.json`: 三个萌宠角色和 `pet_story_cover` recipe。
- `examples/tasks/sample-pet-story-visual-context-task.json`: CLI scene 001。
- `examples/tasks/sample-pet-story-visual-context-scene-002.json`: CLI scene 002。
- `examples/tasks/sample-pet-story-visual-context-scene-003.json`: CLI scene 003。
- `examples/tasks/sample-pet-story-visual-context-rest-create-task.json`: REST create task body。
- `examples/tasks/sample-pet-story-visual-context-mcp-call.json`: JSONL-compatible MCP `tools/call` 示例。

REST 示例：

```bash
curl -X POST \
  http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks \
  -H 'Content-Type: application/json' \
  --data @examples/tasks/sample-pet-story-visual-context-rest-create-task.json
```

MCP 示例：

```bash
docker compose exec -T api /app/mcp < examples/tasks/sample-pet-story-visual-context-mcp-call.json
```

注意：

- 如果 project 已启用 project API key，REST/CLI 调用方需要自行提供现有 `X-API-Key` / Bearer；不要把 key 写进示例文件或日志。
- MCP stdio 逐行读取 JSON-RPC，因此 `sample-pet-story-visual-context-mcp-call.json` 保持为单行 JSON。
- 若正在使用旧的已运行 Docker image，新增 examples 需要 rebuild 后才会出现在 `/app/examples/tasks/`；未 rebuild 时可以用 host-side REST 示例或临时 `docker compose cp` 到容器内验证。

### Web Project Context panel

P1-PCTX-008 已完成最小 Web Project Context 入口，第一版仍只服务当前 project 的长期视觉上下文，不扩展成通用 DAM 或运营后台。

Web 行为：

- 顶栏 `Project Context` 按钮会懒加载 modal，并读取当前 `workspace_id / project_id` 的 `visual_context`。
- Web 使用 Admin session / Basic fallback 读取和保存 visual context；不要求人工在 Web 中填写 project API key，也不在前端接触 provider key。
- modal 展示 characters、references、prompt recipes 三类数据，并区分 loading、empty、unauthorized/login required、error 状态。
- character / reference binding / prompt recipe 支持最小新增、编辑、归档/恢复；保存方式是写回完整 `visual_context` 文档，保存后重新拉取服务端状态。
- 服务端资产卡的 `Reference` 动作只打开 Project Context modal 并带入 `asset_id`，用于写 reference binding；不会复制文件、改 asset status 或改原始 asset。
- Web 托管模式输入框可选择 recipe、characters 和 references；创建任务时只发送 `character_ids`、`reference_asset_ids`、`prompt_recipe_id`、`use_project_visual_context`，最终 prompt 展开仍由服务端完成。

如果打开 modal 后看到 `unauthorized / login required`，先用同一 host 完成 Admin 登录，再刷新或重新打开 Project Context。不要把 Admin cookie、Basic 密码、project API key 或 provider key 写入日志、截图、文档 evidence。

### P1-PCTX-009 regression evidence

P1-PCTX-008/009 的场景矩阵和最小功能设计见：

```text
docs/project/stories/slice-037-pctx-web-panel-and-pet-story-scenarios.md
```

P1-PCTX-008 的实现 evidence 见：

```text
docs/project/stories/slice-038-pctx-web-project-context-panel.md
```

P1-PCTX-009 clean 萌宠故事 mock 回归已完成。验收用独立 project/campaign、同 project style reference、三角色、`pet_story_cover` recipe 和 scene-only batch，验证 project context 到 task / asset metadata、batch progress、asset list 和 Admin Recent Assets 不断链。

验收 ID：

```text
workspace: ws_default
project: prj_pet_story_pctx009_1782094416
campaign: cmp_pet_story_pctx009_1782094416
style_reference_task: task_4dfbbd870dbb99f2e9fc
style_reference_asset: asset_b8d5272e4afa0e249e5f
scene_session: pet_story_pctx009_scene_session_1782094416
scene_batch: pet_story_pctx009_scene_batch_1782094416
scene_story: pet_story_pctx009_scene_story_1782094416
scene_tasks: task_6e6cf178fcf656386d62, task_2fca7b06875125b64d01, task_108de9bcdaafc384302f
scene_assets: asset_0aaa4b95e0914ba51c6d, asset_1735fa3123d84caea912, asset_743f12cb989a42fe002a, asset_4b174cc3330569378cc0, asset_b4232047c2b6df882314, asset_c486950784aef846a424
```

结果：

- Visual Context readback 返回 3 个角色、1 个同 project style reference 和 `pet_story_cover` recipe。
- Final scene-only batch progress 返回 `task_count=3`、`succeeded_count=3`、`failed_count=0`、`asset_count=6`、`attempt_count=3`。
- REST/CLI asset list 通过 `source=codex`、`session_id=pet_story_pctx009_scene_session_1782094416`、`batch_id=pet_story_pctx009_scene_batch_1782094416` 查回 6 张 assets，覆盖 `scene_001`、`scene_002`、`scene_003`。
- Admin Recent Assets 使用同一组 filters 查回 6 张 assets，scope 为 `ws_default/prj_pet_story_pctx009_1782094416/cmp_pet_story_pctx009_1782094416`。
- 每个 scene task 的 `structured_input_json` 保留 `character_ids`、`reference_asset_ids`、`prompt_recipe_id`、`visual_context_snapshot` 和 story/scene/batch metadata。
- 每个 scene asset 的 `parameters_json.visual_context_snapshot` / `metadata_json.visual_context_snapshot` 保留 character ids、reference asset ids、recipe id，并保留 `story_id`、`scene_id`、`batch_id`。

复跑注意：

- style reference 准备任务可以复用同 project asset，但不要把 reference setup task 混进最终 scene batch；最终验收建议单独使用 scene-only `session_id/batch_id`，避免 `task_count` 和 `asset_count` 被 setup task 污染。
- 若需要验证 Web UI，优先用 Admin Recent Assets 数据源或 production preview；不要读取或打印 Admin cookie/session token。

验收边界：

- 不运行真实 provider；mock provider 足够验证任务、资产、缩略图、metadata 和 Web 查看闭环。
- 不读取、打印或处理任何 API key / provider key / secret；如果 project key 已启用，只验证未授权状态或由调用方在不输出的环境变量里提供。
- 不做小红书发布、内容日历、账号运营后台、通用 DAM、模板市场、多人协作、RBAC、Batch / Story / Scene 新 UI、Export Pack 或 NAS/WebDAV/SMB。
- P1-PCTX-009 已验证 project context 到 task/asset metadata/Web 可见性不断链；后续如果继续 Batch / Story / Scene UI 或导出能力，需要重新确认范围并拆独立 CSV。

### Batch Story Export Foundation planning

下一阶段入口：

```text
issues/next-phase-p1-batch-story-export-foundation.csv
```

场景和边界：

```text
docs/project/stories/slice-040-batch-story-export-scenarios.md
```

第一轮只做：

- batch/story/scene grouped view。
- `batch-summary` 第一版契约见 `docs/project/stories/slice-041-batch-story-summary-contract.md`，实现记录见 `docs/project/stories/slice-042-batch-story-summary-api.md`。
- MCP `list_image_assets` 与 REST/CLI/Admin Recent Assets 的 session/batch filter parity。
- scene 级 retry/regenerate：设计见 `docs/project/stories/slice-046-scene-regenerate-design.md`，实现记录见 `docs/project/stories/slice-047-scene-regenerate-implementation.md`。第一版采用 `create-new-task-as-regeneration`，保留原 scene metadata，不覆盖旧 task/assets，不自动改变 selected/rejected。
- selected-only review。
- JSON manifest export：REST `GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-manifest`、CLI `vag batch manifest`、Web Production View manifest buttons 已完成。
- 可选 ZIP export 已在 `docs/project/stories/slice-049-export-pack-zip-boundary.md` 明确后置；第一轮默认通过 manifest + NAS/filesystem 访问交付文件。
- NAS/Docker 文件系统访问说明已完成，见 `docs/project/stories/slice-050-nas-docker-access-guide-and-regression.md`。

Scene regenerate 使用/维护注意：

- 优先输入 `source_task_id`；也可用 `session_id/batch_id/story_id/scene_id` + `task_selector=latest` 解析 source task。
- REST action: `POST /api/projects/{project_id}/campaigns/{campaign_id}/scene-regenerations`。
- Web Production View 第一版使用 scene `latest_task_id` / `source_task_id` 和可选 reason 创建 regeneration task。
- 新 task 必须保持同一 project/campaign/session/batch/story/scene，并在 `structured_input_json.metadata_json` 记录 `regenerated_from_task_id`、`regenerate_no`、`regenerate_reason`、`regeneration_overrides` 和 request source。
- 可覆盖字段只限 prompt、negative prompt、prompt recipe、character/reference ids、reference descriptors、requested_count、selection_mode、aspect/output/provider/model 和非敏感 generation config；不得接受 provider key、API key、cookie、session token 或任意本地路径。
- `batch-summary`、`batch-progress`、asset list 和未来 manifest 应通过 metadata lineage 读到 regeneration；selected-only manifest 不因 regenerate 自动替换旧 selected asset。
- CLI/MCP regenerate command 和 Web prompt/recipe override UI 仍后置；当前外部 agent 可先调用 REST action。

Batch manifest 使用注意：

- 至少传 `session_id` 或 `batch_id`。
- `view=engineering` 为默认值；`view=final_delivery` 会在兼容保留旧顶层 `counts/tasks/assets/scenes/stories` 的同时，额外返回 `manifest_view=final_delivery` 和 `final_delivery` block，适合人工按 `story/scene/batch` 复盘最终交付图。
- `selected_only=true` 只导出 selected assets；`selected_only=false` 默认导出 generated + selected；`include_rejected=true` 才导出 rejected assets。
- `final_delivery.final_assets` 的判定规则固定为 scene 内 `delivery_role=final_delivery` 的资产；如果 caption derivative 已被 selected 或 auto-selected，最终交付会指向派生图，否则仍指向 base selected。
- `target_path` 采用 asset 优先、scene 兜底；manifest 继续只输出公开 delivery URL、thumbnail URL、metadata URL 和逻辑交付路径，不输出 `local_path`、宿主机绝对路径、provider key、project API key、cookie 或 session token。
- Manifest 只包含公开 delivery URL、metadata URL、target_path、scene/story/task id 和 visual context 摘要；不得加入 `local_path`、provider key、project API key、cookie 或 session token。
- ZIP、多文件下载和服务端打包能力已在 P1-BSE-010 决定后置；未另行确认前不实现。
- 已新增 batch-first NAS readable mirror：运维可把 final/selected originals、thumbnails 和 `manifest.final.json` materialize 到 `workspaces/<workspace>/projects/<project>/batches/<batch>`，方便人工通过 Finder/NAS 按一组图复盘，而无需翻 `asset_id` 目录。

常用查询方式：

- REST 工程视图：`GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-manifest?session_id=<session_id>&batch_id=<batch_id>&selected_only=true`
- REST final delivery 视图：`GET /api/projects/{project_id}/campaigns/{campaign_id}/batch-manifest?session_id=<session_id>&batch_id=<batch_id>&selected_only=true&view=final_delivery`
- CLI final delivery 视图：`vag batch manifest --project-id <project_id> --campaign-id <campaign_id> --session-id <session_id> --batch-id <batch_id> --selected-only --view final_delivery`
- 本地 readable mirror：`vag storage mirror-final --workspace <workspace_id> --project <project_id> --campaign <campaign_id> --session-id <session_id> --batch-id <batch_id>`
- Admin 受控 REST readable mirror：`POST /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/final-delivery-mirror`

### NAS / Docker / WebDAV / SMB access guide

第一版自托管交付推荐采用：

```text
Agent ImageFlow DB / metadata / manifest
        +
Docker storage root on NAS-backed volume
        +
NAS / Finder / WebDAV / SMB read-only file access
```

职责边界：

- 文件系统负责：浏览原图、缩略图和 metadata 文件；复制交付文件；做 NAS 快照、离线备份和人工归档；通过部署环境把 Docker storage root 映射到 NAS 路径。
- DB / metadata 负责：workspace/project/campaign/task/asset 归属，task 状态，asset `generated/selected/rejected/published` 状态，visual context snapshot，scene/story/batch 追踪，manifest，audit log，storage governance 和 integrity 视图。
- Manifest 负责连接两边：它输出 asset id、task id、session/batch/story/scene、delivery URL、thumbnail URL、metadata URL、target_path 和 visual context 摘要；它不输出宿主机本地绝对路径。

部署建议：

- Docker Compose 的 storage root 应挂载到持久目录；在 NAS 上运行时，优先使用 NAS 本地路径或 bind mount 作为该目录。
- NAS / WebDAV / SMB / Finder 面向人和外部 agent 的常规访问建议只读。需要复制交付件时，从共享目录复制出去，不在共享目录内移动、重命名或删除平台管理文件。
- 不要把 storage root 直接暴露到公网；公网访问走 Web/API 的 HTTPS 反向代理和现有鉴权。
- 容器运行用户、NAS 共享用户和备份任务需要有清晰权限：服务端进程可写，普通浏览/交付账号只读。

不要手动改动平台管理文件：

- 不要手动移动、重命名或删除已 selected / approved / published 的资产文件。
- 不要通过文件夹重命名来表达 project、campaign、scene 或 asset 状态；这些状态只由 DB / metadata 表达。
- 如果绕过平台删除文件，DB 仍会保留 asset/task/status 记录，delivery URL 可能失效，storage-integrity 才能发现不一致。
- 清理磁盘优先使用 storage governance / cleanup dry-run 和 execute 流程；它会保护 selected / published / approved 资产。

备份与恢复：

- 备份必须同时包含 Postgres dump 和 storage root 一致快照；只备份数据库会丢图片文件，只备份文件会丢 task/asset/status/visual context/manifest 追踪。
- 恢复时先恢复数据库和 storage root，再用 storage-integrity 或 repair verify 类命令做只读校验。
- 如果 NAS 提供快照，建议在数据库 dump 前后记录时间点，并让 storage root 快照与 dump 时间尽量一致。

Restore drill 最小步骤：

1. 停止 `api` / `worker`，必要时也停止 `web`，避免恢复中继续写入。
2. 恢复同一时间窗的 Postgres dump 与 storage root / NAS 快照。
3. 重新启动 `postgres`、`redis`、`api`、`worker`，确认 `/healthz` 正常。
4. 运行 `storage-integrity` 或 `vag repair verify-asset <asset_id>`，抽查 1-3 个 selected assets。
5. 重新获取一次 `selected_only` manifest，确认 `delivery_url` / `thumbnail_url` / `metadata_url` 仍可访问。
6. 最后再用 Web Recent Assets / Production View 做人工 replay，不要把“能翻到文件”当成恢复完成的唯一标准。

Manifest 与文件系统路径：

- `target_path` 是交付逻辑路径，用于下游组织文件名或目录，不等于宿主机绝对路径。
- `delivery_url` / `thumbnail_url` / `metadata_url` 是平台交付入口，适合 agent、Web 和外部系统读取。
- 文件系统实际路径由 Docker volume / NAS mount 决定，不写入 manifest；不同机器恢复后路径可以不同，只要 storage root 内容和 DB metadata 一致即可。

Manifest -> NAS 复制交付最小流程：

1. 人工复盘/NAS 复制时优先取 `selected_only=true&view=final_delivery` 的 batch manifest，确认最终交付使用的是哪组 asset、`delivery_role` 是什么、每个 scene/story 对应哪个 `target_path`；如果需要更完整工程事实，再补看默认 `engineering` 视图。
2. 如果只需要平台交付链接，直接把 `delivery_url` / `thumbnail_url` / `metadata_url` 交给下游，不必翻物理目录。
3. 如果希望平台直接生成给人看的批次目录，先执行一次 readable mirror materialize；平台会在 mirror root 下写出 `manifest.final.json`、`final/<target_path>` 和 `thumbnails/<target_path>.webp`。
4. 如果还需要人工从 NAS/共享目录复制交付件，再从 readable mirror 或只读共享复制到共享外的新目录或外部交付目录。
5. 不要试图通过改平台内部 `asset_id` 目录名来表达 story、scene 或发布分组；这类业务语义只由 manifest / metadata 表达。

Readable mirror 约束：

- mirror 默认根目录是 `STORAGE_ROOT/final-delivery-mirror`；可通过 `FINAL_DELIVERY_MIRROR_ROOT` 覆盖。
- mirror 只复制 `delivery_role=final_delivery` 的 originals 和现有 thumbnails，不复制所有 generated/rejected 候选图。
- 目录优先复用 final asset 的 `target_path`；如果缺失，则回退为 `stories/<story_id>/<scene_id>`。
- mirror 是派生视图，不是事实源。删除、清空或重建 mirror 不会改变 DB、canonical storage、audit、cleanup 或 integrity 语义。

NAS 只读共享 checklist：

- 平台服务账号：对 `AGENT_IMAGEFLOW_STORAGE_ROOT` 保持读写。
- 人工浏览/交付账号：默认只读；需要复制时，复制到共享外的新目录，不回写平台管理目录。
- 备份任务账号：可读 storage root，并在 dump/快照窗口内执行一致性备份。
- 恢复演练后重新挂载共享时，先做 `storage-integrity` 或 `repair verify-asset` 校验，再开放给人工浏览。

避免变成复杂 DAM：

- Reference Library 只保留 `character/style/scene/prop` 等项目视觉生产用途标记，不扩展成通用标签体系、文件夹 taxonomy 或企业资产库。
- WebDAV / SMB server 不在 Agent ImageFlow 应用内实现，由 NAS 或操作系统提供。
- 不做内容日历、发布状态运营、账号后台、多租户文件权限或多人协作工作流。
- 后续若需要 ZIP，只能另开小批量 selected assets 切片，并复用 manifest，不重新发明导出筛选。

边界：

- 不运行真实 provider，除非用户单独确认费用。
- 不读取、打印或处理任何 API key / provider key / secret / cookie / session token。
- 不做小红书发布、内容日历、账号运营后台、通用 DAM、WebDAV/SMB server、多人协作、Usage Tracking 或 AI 视觉质检。
- NAS/WebDAV/SMB 第一轮由部署环境承担文件浏览、拷贝和备份；Agent ImageFlow 负责 DB metadata、状态、delivery URL、manifest 和审计。

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
  --model gpt-image-2 \
  --poll-interval 2s \
  --timeout 30m \
  --concurrency-label worker-2-provider-cap-2 \
  --allow-paid-provider
```

对照 Responses API streaming 时，用任务级 `--model` 覆盖环境中的 Images 模型，避免 `/responses` 被显式的 `OPENAI_COMPATIBLE_MODEL=gpt-image-2` 路由失败：

```bash
docker compose exec api /app/vag benchmark image-generation \
  --provider openai-compatible \
  --tasks 5 \
  --requested-count 1 \
  --api-mode responses \
  --stream true \
  --partial-images 1 \
  --model gpt-5.5 \
  --timeout-seconds 600 \
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
API URL: (留空时使用当前 Web origin；本地 CLI/测试环境回退 http://localhost:8081)
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
- 托管模式输入框会显示 Project Context selector，可选择当前 project 的 prompt recipe、characters 和 references；提交任务时会传 `character_ids`、`reference_asset_ids`、`prompt_recipe_id`、`use_project_visual_context`，前端不展开最终 prompt。
- 托管模式默认传 `selection_mode=auto`；多候选任务完成后，服务端会按任务输入或项目级 quality profile 中的 `best_of_config` 自动 selected 一张候选。
- 如果服务端启用了 Basic Auth 或项目级 API key，Web 会自动附带 `Authorization: Basic ...` 和 `X-API-Key`。
- Web 服务端资产库默认展示 Recent Assets；用户使用 Admin 登录后，不需要手填 project API key 也可以看到 MCP / CLI / REST / Web 生成的最近资产。
- 生产部署时，Agent ImageFlow API URL 不是 provider base URL；provider key/base URL 只来自服务器环境变量。
- Web Current Scope 可通过 workspace / project / campaign 三级下拉切换业务空间，Recent Assets、Production View 和 Project Context 会跟随当前 scope。
- 资产卡会显示 workspace / project / campaign，并可点击 `Scope` 切换当前 scope；`Scope` 视图仍可查看当前 campaign 的资产。
- 资产卡的 `Reference` 动作会打开 Project Context modal，用当前 asset 建立 character/style/scene/prop reference binding；该动作不改变 asset 文件或 selected/rejected 状态。
- 未登录或 401 会显示 unauthorized/login 状态，不再和真实空列表、筛选无结果混成 `0 shown`。
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

Web 控制台 Recent Assets 使用 Admin session 读取跨 scope 最近资产：

```bash
curl -b /tmp/agent-imageflow-admin.cookie \
  'http://localhost:8081/api/admin/assets/recent?limit=24&source=rest&session_id=session_001'
```

该接口返回同一套 asset response shape，但按 `created_at desc` 跨 workspace/project/campaign 排序。它用于控制台发现资产，不改变 project API key 作为外部 MCP/CLI/REST project 级访问凭据的规则。

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
    "quality": "high",
    "api_mode": "images",
    "stream": false,
    "partial_images": 0
  },
  "use_project_quality_profile": true,
  "api_mode": "images",
  "stream": false,
  "partial_images": 0,
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
  -d '{"enabled":true,"provider":"mock","model":"mock-image","generation_config":{"quality":"high"},"use_project_quality_profile":true,"api_mode":"images","stream":false,"partial_images":0,"max_n":4,"preferred_response_format":"url"}'

docker compose exec api /app/vag project provider set \
  --provider mock \
  --model mock-image \
  --generation-config '{"quality":"high"}' \
  --api-mode images \
  --stream false \
  --partial-images 0 \
  --max-n 4 \
  --preferred-response-format url
```

边界：

- 不保存或回显真实 provider secret。
- 未配置 profile 时继续使用服务端环境变量中的默认 provider。
- 创建任务没有显式 `provider` 时，服务端会优先使用启用的项目 provider profile。
- `provider_profile.model` 当前可覆盖 `openai-compatible` 的 model 和 `fal` 的 endpoint id；`base_url` 第一版只作为非敏感项目默认配置保存，真实 endpoint/key 存储策略需要单独确认。
- `api_mode` 可选 `images` / `responses`；`stream` 控制是否请求 SSE；`partial_images` 控制 partial image event 数量，范围 0-3。任务 `generation_config` 中同名字段优先于项目 provider profile。
- `max_n` 表示单次 provider 请求建议承载的同 prompt 变体数，默认值按 provider 保守选择：mock 为 4，openai-compatible 为 1，服务端上限 10；`requested_count` 超过 `max_n` 时会拆成多次 provider 请求并保留同一个 task。
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
OPENAI_COMPATIBLE_MODEL=
OPENAI_COMPATIBLE_MAX_CONCURRENCY=3
PROVIDER_TIMEOUT_SECONDS=300
OPENAI_COMPATIBLE_CONNECT_TIMEOUT_SECONDS=30
OPENAI_COMPATIBLE_RESPONSE_HEADER_TIMEOUT_SECONDS=300
OPENAI_COMPATIBLE_TOTAL_TIMEOUT_SECONDS=300
```

说明：

- 默认不启用真实 provider；未配置 base URL 或 API key 时，`provider=openai-compatible` 会在创建任务时返回明确错误。
- adapter 支持 Images API 同步、Images API streaming 和 Responses API `image_generation` streaming。Images API 默认模型为 `gpt-image-2`；当未显式配置 `OPENAI_COMPATIBLE_MODEL` 且 `api_mode=responses` 时，Responses API 默认模型为 `gpt-5.5`。
- adapter 调用 `{OPENAI_COMPATIBLE_BASE_URL}/images/generations`，默认省略 `response_format` 并解析 `data[].url` 或 `data[].b64_json`；显式配置 `preferred_response_format=b64_json` 时会请求 Base64 响应。
- 当任务存在已解析 reference/mask 输入时，adapter 调用 `{OPENAI_COMPATIBLE_BASE_URL}/images/edits`。
- 当 `api_mode=responses` 时，adapter 调用 `{OPENAI_COMPATIBLE_BASE_URL}/responses`，使用 `tools:[{"type":"image_generation"}]` 和 `tool_choice:"required"`。
- adapter 会从 `generation_config` 白名单透传 `quality`、`moderation`、`output_compression`；`api_mode`、`stream`、`partial_images`、`preferred_response_format`、`timeout_seconds`、`max_n` 优先使用任务 `generation_config`，再使用项目 provider profile。
- openai-compatible 默认 `max_n=1`，多图会走多个受 provider cap 限制的并发 `n=1` 请求；只有在目标 provider 已验证 `n>1` 会返回完整多图时，才建议显式配置 `max_n`。
- 返回图片会在服务端规范化为 PNG，再进入现有 asset processor / storage / delivery。
- 自动化验证使用本地 HTTP mock，不会触发真实外部 API。真实 smoke 需要用户自行配置密钥，并自行承担 provider 成本。

### Reference image 1 图 canary checklist

仅在用户明确确认费用和使用真实 provider 后执行。默认 CI、mock smoke、Web browser smoke 都不要跑真实 provider。

目标：验证固定角色参考图真的进入 openai-compatible `/images/edits` 链路，而不是绕过参考图后纯文生图成功。

前置条件：

- 服务器已部署包含 `fix: set edit multipart image content types` 的镜像或 commit。
- `.env.prod` 中的真实 provider key 只保存在服务器环境变量，不写入仓库、MCP 配置、截图或聊天。
- 使用一个独立 test project/campaign/session/batch，避免污染正式批次。
- 准备 1-3 张同 project 的角色主图/参考图 asset，并在 Project Context 中绑定到角色。

执行验收只记录这些非敏感证据：

- task id / asset id / project id / campaign id / session id / batch id。
- provider 和 model 名称。
- task/asset metadata 中 `reference_asset_count > 0`。
- `provider_reference_participation` 为 `resolved_input_files` 或等价成功状态。
- task 使用 edits/reference 路径；如果失败，错误需要说明“参考图未参与生成”和 MIME/content-type。
- thumbnail/original/metadata delivery URL 返回 200。
- Web Project Context 能看到角色主图/参考图缩略图；Recent Assets 能看到生成资产。

禁止记录：

- provider key、project API key、Basic Auth、Admin cookie、session token、cleanup token。
- 本地绝对文件路径。
- 真实 provider 响应中的敏感 header。

如果 canary 失败：

- 不要自动重试多张图。
- 先保存非敏感 task/attempt error、metadata 摘要和 provider/model。
- 检查 multipart image/mask part 是否有正确 `Content-Type`，以及任务 metadata 是否显示参考图已解析。

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

MCP 新 agent 接入优先参考：

- `docs/project/MCP_SERVICE_GUIDE.md`
- `examples/mcp/agent-imageflow.local.json`
- `examples/mcp/create-pet-scene.json`
- `examples/mcp/smoke.md`

当前 MCP Service Pack 已完成文档和示例落地，并通过 JSON parse 与静态检查；人工 mock smoke evidence 尚未回填，因此项目管理文档里暂不标 done。

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
