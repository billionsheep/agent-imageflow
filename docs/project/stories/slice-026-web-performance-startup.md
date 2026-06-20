# Slice 026: Web Performance Startup

## Product Goal

治理 Web 首屏和长时间打开后的资源占用，避免本地试用时触发浏览器 `High memory usage`，同时不改变 Agent ImageFlow 的产品边界和资产生产能力。

## User Flow

1. 用户打开 `http://localhost:8080/`。
2. Web 只执行一次必要启动初始化。
3. 本地历史任务和服务端资产库按预算渲染，图片和缩略图按需加载。
4. 恢复轮询和 Scope 控制台统计只在有明确需要时运行，并且有数量上限。
5. 用户需要 Agent/Markdown 功能时才加载重模块。

## In Scope

- dev / production preview 的资源基线和差异记录。
- `initStore` 幂等，降低 React StrictMode 双调用放大。
- IndexedDB 缩略图 backfill 队列和每次启动处理上限。
- 本地任务画廊首屏渲染预算和加载更多。
- fal/custom 恢复轮询数量和时间窗口上限。
- Scope 控制台统计缓存、关闭取消和扫描边界。
- 服务端资产库前端保留节点上限。
- Markdown/Math 样式和模块首屏懒加载。

## Out Of Scope

- 不清空 IndexedDB。
- 不删除用户资产。
- 不执行 storage cleanup。
- 不推进 Reference Library、Mascot Profile、Prompt Recipe 或 edit lineage。
- 不修改 provider key、Docker secret 或公网部署策略。

## Acceptance Criteria

- 重复调用 `initStore()` 时，同一 in-flight 初始化只执行一次，失败后仍可重试。
- 启动时后台缩略图补建不会无限制处理所有历史图片。
- 本地任务画廊不会一次挂载全部历史任务卡。
- 恢复轮询不会对大量历史 recoverable task 同时发起请求。
- Scope 控制台关闭后不继续把结果写回 UI，并有短缓存和扫描边界。
- 服务端资产库加载更多不会无限保留图片节点。
- 首屏不静态加载 Markdown/Math 重 CSS。

## Technical Approach

- 仅改 Web 前端，保持服务端 API 和数据模型不变。
- 优先使用模块级启动 guard、渲染分页、现有 `requestIdleCallback`、短缓存和上限常量。
- 不引入虚拟列表依赖；若后续仍高内存，再单独评估虚拟化。

## Data / Interface Impact

- 无服务端接口变化。
- 无数据迁移。
- Web 本地启动策略变化：旧 recoverable 任务不会自动无限恢复，需要用户手动 retry/repair。

## Verification

- `npm --prefix web test -- --run store`
- `npm --prefix web test -- --run`
- `npm --prefix web run build`
- dev server 和 production preview 的进程/页面资源对比。

## Assumptions And Risks

- 当前 1.1GB 提示更可能来自浏览器 renderer、dev/HMR 和大量本地 UI 节点组合，而不是 Vite dev server 本身。
- 没有清理用户数据，因此 before/after 只看同一数据集下的启动和渲染预算变化。
- 若用户本地历史任务极多，当前分页能降低首屏压力，但完整虚拟滚动仍属于后续专项。

## Implementation Log

- Baseline: Vite dev server 进程约 132 MB RSS；Codex renderer 约 846 MB RSS；浏览器自动化大采样在当前页面上断开，说明页面已处于较重状态。
- Added `initStore` in-flight guard and test reset helper.
- Added background thumbnail backfill queue/session budget.
- Added local TaskGrid render budget and load-more control.
- Added startup recovery cap and age window for fal/custom recoverable tasks.
- Added ServerAssetLibrary rendered asset cap.
- Added Scope dashboard cache, scan bounds and close cancellation.
- Moved Markdown/Math CSS loading from app entry to MarkdownRenderer lazy path.
- Lazy-loaded AgentWorkspace, Settings, Scope manager, Detail, Lightbox and Mask editor from the app shell.
- Production preview served successfully at `http://127.0.0.1:4173/`; preview node process was about 103 MB RSS while dev Vite was about 136 MB RSS.
- `npm --prefix web test -- --run` passed: 17 files / 224 tests.
- `npm --prefix web run build` passed; main entry chunk reduced from about 951 KB to about 711 KB, with remaining chunk warning left as a follow-up rather than a broad rewrite.
