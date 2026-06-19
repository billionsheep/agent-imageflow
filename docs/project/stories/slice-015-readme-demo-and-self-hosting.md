# Story: Slice 015 - README, Demo, and Self-hosting Guidance

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

把当前已经能跑通的 MVP 能力整理成一份对外可读、可上手、可自托管的说明，让新接手的人能在 5 到 10 分钟内理解产品定位、跑起最小闭环、完成一个 demo，并知道自托管时最小暴露面和反向代理/TLS 的基本做法。

## Source Context

- Tasks: 当前唯一 Todo 是“补 README/demo 和更完整的自托管部署说明（反向代理/TLS、最小暴露面）”。
- Project plan: 当前服务端资产闭环、Web 托管、真实 edit/mask、鉴权和 hardening 基础都已完成，下一步是文档收尾。
- Runbook: 已有大量命令，但偏工程内部记录，缺少面向新用户的 quickstart/demo/self-hosting 汇总。
- README: 当前 README 仍保留多处阶段性表述，且下一阶段优先级已过时。

## User Flow

1. 新读者打开仓库首页。
2. 读者快速理解 Agent ImageFlow 是什么、现在已经能做什么、当前还有什么边界。
3. 读者按 Quickstart 启动 Docker Compose 和 Web。
4. 读者按 Demo 流程创建一条 mock 或 managed task，看到资产闭环。
5. 若准备自托管，读者能看到反向代理/TLS、Basic Auth、project API key、最小暴露面和持久化建议。

## In Scope

- 更新 `README.md`，补定位、能力现状、Quickstart、Demo 流程和当前边界。
- 补一段更完整的自托管说明，明确：
  - 对外只暴露反向代理入口
  - 不公开暴露 PostgreSQL/Redis
  - 启用 Basic Auth 和 project API key
  - 存储卷持久化
  - 反向代理/TLS 的最小样例
- 同步 `RUNBOOK.md`、`TASKS.md`、`PROJECT_PLAN.md`、`CHECKPOINTS.md` 和故事文件。

## Out of Scope

- 不修改 `docker-compose.yml` 的端口映射默认值。
- 不新增生产部署脚本、systemd、Helm chart、Terraform、Kubernetes 清单。
- 不改 CI/CD、数据库、镜像发布或真实线上环境。
- 不补新的代码能力，只整理说明和 demo 路径。

## Acceptance Criteria

- Given 新读者只看 `README.md`，when 按 quickstart 操作，then 能知道先启动哪些服务、打开哪个地址、如何跑最小闭环。
- Given 新读者想验证核心能力，when 按 demo 说明执行，then 能完成 mock 任务、managed mode 或 MCP 的最小演示路径。
- Given 运维或开发者想自托管，when 阅读 README / RUNBOOK，then 能知道哪些端口不该暴露、如何放到反向代理后、如何启用 Basic Auth 和 project API key。
- Given 当前项目已支持 scope input-files 和真实 edit/mask 边界，when 阅读 README / RUNBOOK，then 文档不再停留在旧状态描述。

## Technical Approach

- 以 README 为“外部入口”，Runbook 为“内部操作手册”。
- 在 README 中增加：
  - What it is / What ships today
  - Quickstart
  - Demo flows
  - Self-hosting guidance
  - Current known gaps
- 在 Runbook 中补更偏操作的自托管建议与注意事项。
- 不引入新文件格式或外部文档站，保持仓库内可读。

## Data / Interface Impact

- 无运行时代码接口变化。
- 仅更新说明文档与 story / plan / tasks / checkpoints。

## Files or Subsystems Likely to Change

- `README.md`
- `docs/project/RUNBOOK.md`
- `docs/project/TASKS.md`
- `docs/project/PROJECT_PLAN.md`
- `docs/project/CHECKPOINTS.md`
- `docs/project/DECISIONS.md`
- `docs/project/stories/slice-015-readme-demo-and-self-hosting.md`

## Verification Plan

```bash
docker compose config
curl -u admin:secret http://localhost:8081/healthz
curl http://localhost:8080
```

手工核对：

- README Quickstart、Demo、Self-hosting 内容与当前实现一致。
- RUNBOOK 与 README 不互相冲突。
- TODO / Doing / Current Phase 已切到下一条真实剩余 gap。

## Assumptions and Risks

- 本片不碰真实部署配置，只补说明，因此主要风险是文档与现状不一致。
- 本地 `docker compose.yml` 仍保留开发友好的端口映射；文档必须明确“生产不要这样暴露”。
- 本地 `.vite/` 仍可能存在，是运行产物，不应被误写成源码要求。

## Implementation Log

### 2026-06-18

- Changes:
  - 重建 `README.md`，补产品定位、当前能力、Quickstart、mock/Web/MCP demo、自托管最小暴露面和反向代理/TLS 样例。
  - 更新 `RUNBOOK.md`，补开发友好端口暴露与生产最小暴露面的边界说明。
  - 同步 `TASKS.md`、`PROJECT_PLAN.md`、`CHECKPOINTS.md`、`TECH_SPEC.md` 和 `DECISIONS.md`，把当前 slice 标记为 done，并显式切到下一条 pending slice。
- Verification:
  - `docker compose config`
  - `curl -u admin:secret http://127.0.0.1:8081/healthz`
  - `curl http://127.0.0.1:8080`
- Remaining gaps:
  - 独立 Web scope 管理页、rename/delete/archive 仍待下一片实现。
  - 远程 URL 抓取、asset reuse 和更多 provider 的 edit/mask 仍未接入。
  - best-of 仍是本地启发式，尚未升级到视觉/LLM 打分。
