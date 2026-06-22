# Slice 034: P1 Web UX Task And Asset Render Containment

## Status

- State: Done
- Created: 2026-06-21
- Updated: 2026-06-21

## Product Goal

降低 TaskGrid 和服务端资产库在 select/reject、scope 切换、详情打开、筛选状态变化时的整片重绘概率，让用户点击卡片按钮时页面保持稳定、响应更顺。

## Source Context

- User feedback: Web 端点很多按钮会有屏幕闪烁，使用不流畅。
- Project plan slice: `issues/next-phase-p1-web-ux-smoothness.csv` 的 `P1-UX-008 Task and asset card render containment`。
- Current state: `TaskGrid` 每次渲染都会为每张 `TaskCard` 创建新的 handler；`TaskCard` 订阅整份 `settings`；`ServerAssetLibrary` 在 `assets.map` 内联渲染整张资产卡、现场计算 summaries 和创建 action handler。
- Existing boundary: 不运行真实 provider，不读取、不打印 API key / provider key / secret。

## User Flow

1. 用户在本地任务网格点击任务卡、复用、编辑输出、删除或多选。
2. 用户在服务端资产库点击 Select/Reject/ID/URL/Scope。
3. Web 只更新目标卡片、必要摘要和状态区，不因为父组件一次状态变化让整页卡片重新创建复杂内容。
4. 资产和任务的实际状态语义保持不变。

## In Scope

- 给 TaskGrid 单张任务卡包装一个 memo 化 row，稳定 task action handlers。
- 收窄 `TaskCard` 的全局 store 订阅，避免无关 settings 变化影响所有 task card。
- 给 ServerAssetLibrary 抽出 memo 化资产卡，稳定 select/reject/copy/scope action handlers。
- 保持现有 UI、筛选、select/reject、scope 切换和 delivery links 行为不变。
- 同步更新 Web UX Smoothness CSV 和项目管理文档。

## Out of Scope

- 不引入虚拟列表。
- 不改后端资产接口、DB 或任务状态机。
- 不做浏览器 Profiler 最终验收文档；该项仍归入 `P1-UX-009`。
- 不运行真实 provider，不读取或打印任何 secret。

## Acceptance Criteria

- Given TaskGrid 父组件因 selection/search/filter 更新而渲染，then 未变化 task 的 `TaskCard` 尽量复用稳定 props。
- Given 无关 settings 变化，then `TaskCard` 不因订阅整份 settings 而整批刷新。
- Given 服务端资产库某一张资产正在 Select/Reject，then busy 状态主要影响该资产卡和必要摘要，不重新内联计算所有资产卡内容。
- Given 资产 select/reject 返回新 asset，then 列表只替换目标 asset，状态和按钮行为保持一致。

## Technical Approach

- 在 `TaskGrid` 中新增 memo 化 `TaskGridItem`，使用 `useCallback` 封装单卡事件。
- 将 `TaskCard` 改为 `memo` 导出，并把 `settings` store 订阅收窄为 `alwaysShowRetryButton`。
- 在 `ServerAssetLibrary` 中新增 memo 化 `ServerAssetCard`，把 metadata/parameters summaries 移入卡片内部 `useMemo`。
- 将资产 action 函数改为 `useCallback`，传入稳定 handler，卡片以 `asset_id` 回调。

## Data / Interface Impact

- 无 API、数据库、storage 或外部数据契约变更。
- Web 内部渲染结构拆分为更稳定的子组件。

## Files or Subsystems Likely to Change

- `web/src/components/TaskGrid.tsx`
- `web/src/components/TaskCard.tsx`
- `web/src/components/ServerAssetLibrary.tsx`
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
curl -I http://127.0.0.1:4173/
```

## Assumptions and Risks

- `React.memo` 只有在 props 引用稳定时才有效；本 slice 会同步稳定 handler props，但不会引入更大状态模型重构。
- Browser Profiler was unavailable in this session; final manual / Network / Performance regression was closed by `P1-UX-009` using production preview and browser network observations.

## Implementation Log

### 2026-06-21

- Started implementation for `P1-UX-008 Task and asset card render containment`.
- `TaskCard` now exports `React.memo` and subscribes only to `settings.alwaysShowRetryButton` instead of the whole settings object.
- `TaskGrid` now wraps each visible card in memoized `TaskGridItem` with stable click/reuse/edit/delete handlers.
- `ServerAssetLibrary` now renders memoized `ServerAssetCard` and passes stable select/reject/copy/scope callbacks.
- Asset card metadata / parameters summaries are computed inside the memoized card with `useMemo`.
- Verification passed: `npm --prefix web test -- --run` (17 files / 224 tests), `npm --prefix web run build` (existing chunk warning only), `git diff --check`, and `curl -I http://127.0.0.1:4173/`.
- Remaining gaps: browser Profiler / Network visual regression was closed by `P1-UX-009`.
