# Runbook

当前项目进入实施准备阶段，暂无应用代码、运行命令或测试命令。

## Current Commands

```bash
# 查看项目文档
find docs/project -maxdepth 2 -type f -print

# 查看 Git remote
git remote -v

# 查看本地变更
git status --short
```

## Repository

- Remote: `git@github.com:billionsheep/agent-imageflow.git`
- Local branch: `main`
- Initial commit 已推送，`main` 跟踪 `origin/main`。

## Future Local Run Target

实现阶段建议优先提供：

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

以及至少一个 smoke test：

```bash
vag task create --file examples/tasks/sample-image-task.json
vag task get <task_id>
```

## Ports

未定。进入实现阶段后记录 API、Web UI、MinIO 等端口。

## Debug Notes

- 如果接云端 provider，必须避免提交 API key。
- 如果未来接 ComfyUI，需记录本地 ComfyUI URL、模型要求和输出目录；MVP 不接本地 GPU。
- 如果接 MCP，优先从 stdio 本地模式开始。
