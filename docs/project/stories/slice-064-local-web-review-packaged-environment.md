# Slice 064: Local Web Review Packaged Environment

## 背景

本地开发时，API/worker 能起并不代表“审图复放”路径清晰。过去经常需要额外手工判断该用 Vite dev 还是 preview、该先起哪些服务、该怎样按 batch/story replay，这对 PM 验收和问题回放都不稳定。

## 本片范围

- 固定第一版本地 packaged review 命令链：
  - `docker compose up -d postgres redis api worker`
  - `curl -sf http://127.0.0.1:8081/healthz`
  - `npm --prefix web run build`
  - `npm --prefix web run preview -- --host 127.0.0.1 --port 4173`
- 补充 Admin 临时账号边界
- 补充按 `session_id` / `batch_id` / `story_id` replay Recent Assets / Production View / manifest 的步骤

不做：

- 不新增 Web 创作入口
- 不写入真实账号、cookie、session 或 secret
- 不做浏览器自动登录脚本

## 实现摘要

- `RUNBOOK.md` 新增 Local packaged review 标准路径
- 明确本地 preview 推荐统一用 `127.0.0.1` 或统一用 `localhost`，避免 Admin cookie 因 host 分裂
- 将 replay 入口固定到 batch summary、batch manifest、Recent Assets filters 和 Production View

## 验证

- `npm --prefix web run build`
- `curl -sf http://127.0.0.1:4173/`（在 preview 进程运行时）
- 文档 review：确认新会话可按固定步骤进入审图态

本轮未运行真实 provider，未读取或打印任何 key、cookie、session 或 secret。
