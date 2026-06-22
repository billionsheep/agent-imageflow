# Slice 032: P1 Web UX Stable Lazy Modal Fallback

## Status

- State: Done
- Created: 2026-06-21
- Updated: 2026-06-21

## Product Goal

减少 Web 首次打开 Settings、Scope、Detail、Lightbox 和 Mask editor 时的空白帧，让用户点击按钮后立即看到稳定的 modal loading shell，而不是整屏短暂闪烁或无响应。

## Source Context

- User feedback: Web 端点很多按钮会有屏幕闪烁，使用不流畅。
- Project plan slice: `issues/next-phase-p1-web-ux-smoothness.csv` 的 `P1-UX-007 Stable lazy modal fallback and preload`。
- Current state: P1-UX-001 到 P1-UX-005 已完成，`App.tsx` 的大模块 lazy loading 仍使用 `Suspense fallback={null}`。
- Existing boundary: 不运行真实 provider，不读取、不打印 API key / provider key / secret。

## User Flow

1. 用户点击 Settings、Scope 管理、任务卡详情、图片预览或 Mask editor。
2. 如果对应 chunk 还没加载，页面立即显示稳定 overlay / skeleton。
3. chunk 加载完成后切换到真实 modal，overlay 尺寸稳定，不造成整屏空白。
4. 用户 hover 或 pointerdown 高频入口时，Web 提前预加载对应 chunk，降低首次打开等待。

## In Scope

- 为 lazy modal 增加稳定 fallback shell。
- 把 Settings、Scope、Detail、Lightbox、Mask editor 和 Agent workspace 的动态 import 收束到可预加载 helper。
- Header 的 Settings / Scope 入口支持 hover / focus / pointerdown 预加载。
- TaskGrid 和 AgentWorkspace 的任务详情入口支持 hover / pointerdown 预加载。
- InputBar 的图片预览、Mask editor 和无配置时打开 Settings 的入口支持预加载。
- 同步更新 Web UX Smoothness CSV 和项目管理文档。

## Out of Scope

- 不改变 modal 内部业务逻辑。
- 不处理 Scope dashboard stats 后台加载。
- 不做 TaskCard / AssetCard memo 化和局部重绘。
- 不做浏览器真实 provider 生成，不读取或打印任何 secret。
- 不新增第三方依赖。

## Acceptance Criteria

- Given Settings / Scope / Detail chunk 首次加载，when 用户点击入口，then 页面显示稳定 overlay/skeleton fallback，而不是 `null` 空帧。
- Given 用户 hover、focus 或 pointerdown Settings / Scope 入口，then 对应 chunk 会提前开始加载。
- Given 用户 hover 或 pointerdown 任务卡，then DetailModal chunk 会提前开始加载。
- Given 用户 hover 或 pointerdown 输入图片或遮罩编辑入口，then Lightbox / MaskEditor chunk 会提前开始加载。
- Given Web build，then lazy chunks 仍然拆分，且没有新增依赖或破坏 production build。

## Technical Approach

- 新增 `web/src/lib/lazyModules.ts`，导出统一的 `load*` 和 `preload*` 函数。
- `App.tsx` 使用 `lazy(load*)`，并为大 modal `Suspense` 提供稳定 fallback。
- 预加载入口只调用 `void import(...)` helper，不改 store 状态。
- fallback 使用现有 Tailwind 样式，不引入 CSS 动画依赖。

## Data / Interface Impact

- 无 API、数据库或外部数据契约变更。
- 仅改变 Web 首次打开懒加载 modal 时的过渡体验。

## Files or Subsystems Likely to Change

- `web/src/App.tsx`
- `web/src/lib/lazyModules.ts`
- `web/src/components/Header.tsx`
- `web/src/components/TaskGrid.tsx`
- `web/src/components/AgentWorkspace.tsx`
- `web/src/components/InputBar.tsx`
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

- Final Network/Performance browser regression was closed by `P1-UX-009`.
- Preload only starts loading chunks; it must not open modal or alter settings.

## Implementation Log

### 2026-06-21

- Changes: 新增 `web/src/lib/lazyModules.ts` 统一 lazy 动态 import 和 preload helper；`App.tsx` 为 AgentWorkspace、Detail、Lightbox、Settings、ScopeManager、MaskEditor 增加稳定 overlay/skeleton fallback，不再使用 `fallback={null}`；Header Settings/Scope/Agent、TaskGrid/AgentWorkspace 任务卡、InputBar 图片预览/Mask/无配置提交入口在 hover/focus/pointerdown 时预加载对应 chunk。
- Verification: `npm --prefix web test -- --run` passed 17 files / 224 tests；`npm --prefix web run build` passed with lazy chunks still split and existing chunk-size warning only；`curl -I http://127.0.0.1:4173/` returned 200；`git diff --check` passed；no real provider was run and no API key / provider key / secret was read or printed.
- Remaining gaps: Final visual / Network / Performance regression was closed by `P1-UX-009`. Render containment was completed in slice 034; Scope dashboard stats background loading was completed in slice 033.
