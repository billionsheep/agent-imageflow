# P2 Web Operator Review Console 前端任务 1-4 报告

## 状态

DONE_WITH_CONCERNS

任务 1-4 已按限定范围实现；唯一 concern 是用户指定的 focused test 命令 `npm --prefix web test -- --run web/src/lib/operatorReview.test.ts` 在当前 npm/Vitest 执行方式下会把 Vitest root 设为 `web/`，因此 `web/src/...` filter 匹配不到文件并返回 `No test files found`。等价的 `src/lib/operatorReview.test.ts` filter 可正常运行并通过。

## 改动文件

- `web/src/lib/operatorReview.ts`
- `web/src/lib/operatorReview.test.ts`
- `web/src/components/ServerAssetLibrary.tsx`
- `web/src/components/ProductionViewModal.tsx`
- `web/src/components/SettingsModal.tsx`
- `web/src/store.ts`
- `.superpowers/sdd/task-frontend-report.md`

## 任务完成情况

### Task 1: Operator Review Helpers

- 新增 `getAssetReviewSummary(asset)`：
  - 输出 review-first 字段：prompt、story、scene、source、created、target。
  - 默认摘要不输出 asset_id、task_id、hash、local absolute path。
- 新增 `getAssetTechnicalFields(asset)`：
  - 输出 asset/task/scope/provider/model/hash/source/session/batch/story/scene/target。
  - metadata/parameters 会递归 scrub secret-like keys 和 local absolute path。
  - 不读取、打印、处理真实 secret；测试只使用假值。
- 新增 `getLocalhostMismatchWarning(pageOrigin, apiBaseUrl)`：
  - 覆盖 `localhost` 与 `127.0.0.1` 混用提示。
- 新增 `getProductionFiltersFromAsset(asset)`：
  - 从 asset metadata 提取 session/batch/story/source seed。
  - 缺少 session_id 与 batch_id 时返回 `null`。

### Task 2: Recent Assets Review Card

- `ServerAssetLibrary` asset card 默认展示缩略图、prompt、story/scene/source/created/target 和状态。
- workspace/project/campaign/provider/model/task/hash/session/batch/story/scene/target、metadata、parameters 改入默认折叠的 `Technical details`。
- 保留 Select、Reject、Original、Metadata、Copy ID、Copy URL、Scope、Reference 动作。
- 对有 session 或 batch 的 asset 增加 `Batch` 动作。

### Task 3: Settings Server-First Copy And Host Guidance

- `SettingsModal` 服务端托管区域改为 `Agent ImageFlow 服务端连接`。
- 文案明确：
  - Web 使用服务端平台能力。
  - 真实 provider key / provider base URL 属于服务端配置。
  - Project API key 是 MCP/CLI/REST 外部 project 调用或 fallback，不是 Web Recent Assets 日常查看前置。
  - Basic auth 是自托管/反向代理 fallback，Admin 登录负责 Web 控制台 session。
  - API tab 是旧版/高级浏览器直连 provider 配置。
- `SettingsModal` 和 `ServerAssetLibrary` unauthorized/login 区域加入 localhost/127.0.0.1 host mismatch guidance。

### Task 4: Production View Quick Batch Entry And Feedback

- `ServerAssetLibrary` 的 `Batch` 动作会：
  - 从 asset metadata 提取 Production View filters。
  - 切到 asset 所在 project/campaign scope。
  - 打开 Production View 并通过 store seed 预填 session/batch/story/source。
- `ProductionViewModal` 新增 store seed 初始化路径；手填 session_id/batch_id 的高级路径保留。
- manifest pending/error/toast 反馈保持原有实现，没有退化；build 通过验证该路径仍可编译。

## TDD 与验证

红灯：

- `npm --prefix web test -- --run src/lib/operatorReview.test.ts`
  - 失败原因：`Cannot find module './operatorReview'`，符合 helper 尚未实现的预期。

绿灯与补充验证：

- `npm --prefix web test -- --run src/lib/operatorReview.test.ts`
  - 1 file / 4 tests passed。
- `npm --prefix web test -- --run`
  - 18 files / 232 tests passed。
- `npm --prefix web run build`
  - passed；仅保留既有 Vite large chunk warning。

用户指定命令的 concern：

- `npm --prefix web test -- --run web/src/lib/operatorReview.test.ts`
  - 失败：`No test files found`。
  - 原因：`npm --prefix web` 下 Vitest root 是 `/Users/moon/Workspace/tools/agent-imageflow/web`，filter 需写成 `src/lib/operatorReview.test.ts` 才能匹配当前测试文件。

## 风险与疑问

- 未运行真实 provider，符合约束。
- 未读取、打印、处理任何真实 API key/provider key/secret/cookie/session/password/Authorization。
- 未新增第三方依赖、未提交 git、未推送、未改数据库。
- Production View 的 seed 行为为“预填 filters + 用户点击 Query”，没有自动发请求，避免打开资产卡时隐式触发更多 API 调用。
- 当前 helper 会从 metadata/parameters 中移除 secret-like key 和 local absolute path；如果未来要展示某些名称包含 `token` 但非敏感的业务字段，需要单独确认白名单。
