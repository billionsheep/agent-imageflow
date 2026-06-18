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

# 服务端开发 / smoke
docker compose build
docker compose up
docker compose exec api /app/vag task create --file /app/examples/tasks/sample-image-task.json
docker compose exec api /app/vag task get <task_id>
docker compose exec api /app/vag asset approve <asset_id> # 兼容命令，产品语义等价于 select

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

后续新增 MCP 和 Web managed mode 时优先使用 `select_image_asset` / `select` 命名；当前 Runbook 保留 `approve` 是为了匹配已实现 CLI。

## Ports

- Web dev server: `http://localhost:8080`
- API: `http://localhost:8081`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`
- Worker: 无 HTTP 端口，消费 Redis 队列 `queue:image_generation`。

## Debug Notes

- 如果接云端 provider，必须避免提交 API key。
- 如果未来接 ComfyUI，需记录本地 ComfyUI URL、模型要求和输出目录；MVP 不接本地 GPU。
- 如果接 MCP，优先从 stdio 本地模式开始。
- 本机 shell 当前没有 `go` 命令；Go 测试和格式化可通过 Docker 执行：

```bash
docker run --rm -v "$PWD":/src -w /src golang:1.25.3-alpine sh -lc '/usr/local/go/bin/gofmt -w cmd internal && /usr/local/go/bin/go test ./...'
```
