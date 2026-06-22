# Slice 035: P1 Web UX Smoothness Regression

## Status

- State: Done
- Created: 2026-06-21
- Updated: 2026-06-21

## Product Goal

把 Web UX Smoothness 专项从修复阶段收口到可试用状态，记录 Settings、Scope、Recent Assets、筛选和卡片交互的验证证据，避免后续功能开发再次引入整屏闪烁、空列表误导或请求风暴。

## Source Context

- Project plan slice: `issues/next-phase-p1-web-ux-smoothness.csv` 的 `P1-UX-009 UX smoothness regression and docs`。
- Current state: P1-UX-001 到 P1-UX-008 已完成，剩余工作是最终回归和项目管理固化。
- Existing boundary: 不运行真实 provider，不读取、不打印 API key / provider key / secret。

## User Flow

1. 用户打开 production preview。
2. 打开 Settings，不应触发资产库重拉或整屏空白。
3. 打开 Scope 管理，层级先出现，统计请求不阻塞主 UI。
4. 切换 Recent/Scope、同步资产、输入筛选时，页面状态应区分 unauthorized/empty/no result，不闪成误导性空列表。
5. Select/Reject 等卡片操作应由前序 memo/局部更新治理，后续如仍复现需记录具体路径。

## In Scope

- 运行前端测试、production build、diff check 和 preview HTTP smoke。
- 用本地浏览器工具对 production preview 做无密钥 UI/Network 观察。
- 更新 Web UX Smoothness CSV、TASKS、PROJECT_PLAN、PROJECT_STATUS_MAP、CHECKPOINTS 和 RUNBOOK。

## Out of Scope

- 不再扩大 Web UX 修复范围。
- 不运行真实 provider。
- 不读取 cookie、API key、provider key 或任何 secret。
- 不解决当前浏览器 Admin session / host 不一致导致的 unauthorized，只记录为试用排障路径。

## Acceptance Criteria

- Settings 打开后主框架仍存在，且不触发 `/api/*` 请求。
- Scope 管理打开后主框架仍存在，且首轮只观察到层级请求，没有因为统计扫描造成空白。
- Recent + 同步操作后主框架和服务端资产库仍存在，请求数量受控。
- 当前未授权状态清楚显示为 `unauthorized`，不是误导成真实空列表。
- 项目管理文件记录专项完成状态和已知剩余风险。

## Technical Approach

- 使用 `npm --prefix web test -- --run`、`npm --prefix web run build`、`git diff --check` 和 `curl -I http://127.0.0.1:4173/` 做基础验证。
- 使用 `agent-browser-cli` 连接已有 `http://127.0.0.1:4173/` 标签页，执行只读 DOM / performance entries 检查和按钮点击。
- 不读取 cookies，不打印认证 token，不触碰 provider 配置。

## Data / Interface Impact

- 无代码、API、数据库或 storage 行为变更。
- 仅文档和 CSV 状态收口。

## Files or Subsystems Changed

- `issues/next-phase-p1-web-ux-smoothness.csv`
- `docs/project/TASKS.md`
- `docs/project/PROJECT_PLAN.md`
- `docs/project/PROJECT_STATUS_MAP.md`
- `docs/project/CHECKPOINTS.md`
- `docs/project/RUNBOOK.md`
- `docs/project/stories/slice-035-web-ux-smoothness-regression.md`

## Verification Evidence

- `npm --prefix web test -- --run`: 17 files / 224 tests passed.
- `npm --prefix web run build`: passed, with existing Vite chunk size warning only.
- `git diff --check`: passed.
- `curl -I http://127.0.0.1:4173/`: HTTP 200.
- Browser baseline on `http://127.0.0.1:4173/`: title `Agent ImageFlow`, root and 服务端资产库 visible.
- Browser Settings regression: opening Settings kept root visible and produced 0 `/api/*` resource entries.
- Browser Scope regression: opening Scope 管理 kept root visible and observed `/api/workspaces` as the only `/api/*` entry during the first check.
- Browser Recent/sync regression: Recent + 同步 kept root and 服务端资产库 visible and observed 1 recent request; current status remained explicit `unauthorized`.

## Assumptions and Risks

- Browser JS injection could change an input value but did not reliably simulate React typing in one step; filter debounce was already covered by code review and previous tests/build, while browser-level final evidence records the controlled request after sync.
- Current browser session shows Recent Assets `unauthorized`; this is consistent with host/session mismatch or expired Admin session, not a render regression. Login/session handling remains covered by Web Console Auth docs and RUNBOOK.
- React Profiler was not available through the current browser tooling; P1-UX-008 was verified by code structure, tests, build and production preview availability.

## Implementation Log

### 2026-06-21

- Completed P1 Web UX Smoothness final regression and documentation close-out.
