# Slice 033: P1 Web UX Scope Stats Background Loading

## Status

- State: Done
- Created: 2026-06-21
- Updated: 2026-06-21

## Product Goal

让 Scope 管理 modal 打开和切换 workspace/project/campaign 时优先保持可交互，统计扫描慢或失败不能拖慢层级列表、点击切换和设为当前 scope。

## Source Context

- User feedback: Web 端点很多按钮会有屏幕闪烁，使用不流畅。
- Project plan slice: `issues/next-phase-p1-web-ux-smoothness.csv` 的 `P1-UX-006 Scope dashboard stats background loading`。
- Current state: Scope dashboard stats 已有缓存和扫描上限，但统计加载和层级加载共用 request id，快速切换会频繁取消/重启统计；统计中的资产列表查询没有显式低 limit。
- Existing boundary: 不运行真实 provider，不读取、不打印 API key / provider key / secret。

## User Flow

1. 用户打开 Scope 管理。
2. Web 先加载 workspace/project/campaign 层级并立即允许点击。
3. 统计在后台延迟启动，加载中只影响每行的统计摘要，不阻塞层级选择。
4. 用户快速切换 workspace/project 时，层级请求继续响应最新选择，旧统计请求被取消或忽略。
5. 统计失败只显示提示，不影响 scope 切换、rename/archive/delete 和设为当前 scope。

## In Scope

- Scope hierarchy request 和 dashboard stats request 使用独立 request id。
- Dashboard stats 启动增加短延迟和缓存复用，避免打开 modal 时与层级首屏争抢。
- 快速切换层级时取消/忽略旧统计结果。
- Dashboard 资产列表查询使用明确小 limit，避免无分页拉大资产列表。
- 同步更新 Web UX Smoothness CSV 和项目管理文档。

## Out of Scope

- 不改变后端统计接口。
- 不改变 scope rename/archive/delete 行为。
- 不做 TaskCard / AssetCard render containment。
- 不做最终浏览器 Network/Performance 回归文档。
- 不运行真实 provider，不读取或打印任何 secret。

## Acceptance Criteria

- Given 打开 Scope 管理，when workspace 列表返回，then modal 立即渲染层级列表，统计显示待同步/同步中。
- Given dashboard stats 正在加载，when 用户点击 workspace/project/campaign，then 点击不等待统计完成。
- Given 用户连续切换 scope，when 旧统计请求晚返回，then 不覆盖最新选择的统计状态。
- Given 某个 campaign 的统计接口失败，then 只显示统计提示，scope 切换仍可继续。
- Given dashboard 拉 asset 摘要，then 使用明确小 limit，避免为了统计扫描大资产列表。

## Technical Approach

- 在 ScopeManagerModal 中拆分 hierarchy request 和 dashboard request refs。
- 新增 stats debounce timer，层级列表稳定后再启动 stats。
- `scheduleDashboardStats` 负责取消旧 timer、递增 stats request id，并调用 `loadDashboardStats`。
- `loadDashboardStats` 只检查 stats request id；关闭 modal 时取消 timer 并终止 stats 写回。
- `listAgentImageflowAssets` 在 dashboard stats 中传 `limit` 查询参数。

## Data / Interface Impact

- 无 API、数据库或外部数据契约变更。
- Web dashboard 统计从“跟层级刷新绑定”调整为“后台延迟同步”。

## Files or Subsystems Likely to Change

- `web/src/components/ScopeManagerModal.tsx`
- `issues/next-phase-p1-web-ux-smoothness.csv`
- `docs/project/TASKS.md`
- `docs/project/PROJECT_PLAN.md`
- `docs/project/PROJECT_STATUS_MAP.md`
- `docs/project/CHECKPOINTS.md`
- `docs/project/RUNBOOK.md`

## Verification Plan

```bash
npm --prefix web test -- --run
npm --prefix web run build
git diff --check
```

## Assumptions and Risks

- Dashboard stats 是辅助信息，不是 scope 管理动作的事实源；因此后台化后允许统计短暂显示旧缓存或待同步文案。
- Final visual / Network / Performance regression was closed by `P1-UX-009`.

## Implementation Log

### 2026-06-21

- Started implementation for `P1-UX-006 Scope dashboard stats background loading`.
- Split `ScopeManagerModal` hierarchy loading and dashboard stats loading into independent request lifecycles.
- Added delayed stats scheduling so workspace/project/campaign hierarchy renders before stats scans begin.
- Stats scans now cancel pending timers and ignore stale results when the modal closes or hierarchy reloads.
- Dashboard asset scans now request a bounded asset list with `limit=24`.
- Verification passed: `npm --prefix web test -- --run` (17 files / 224 tests), `npm --prefix web run build` (existing chunk warning only), `git diff --check`, and `curl -I http://127.0.0.1:4173/`.
- Remaining gaps: browser Network/Performance visual regression was closed by `P1-UX-009`; Select/Reject render containment was completed in `P1-UX-008`.
