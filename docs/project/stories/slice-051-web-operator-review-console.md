# Slice 051: Web Operator Review Console

## Context

Project Production Context 和 Batch Story Export Foundation 已经跑通：agent 可通过 MCP/REST/CLI/Web 创建图片任务，Worker 产出 assets，Web 可看 Recent Assets、Production View、scene select/reject/regenerate 和 JSON manifest。

当前缺口不是生成链路，而是 Web 仍偏工程调试视角。Recent Assets 卡片默认暴露 workspace/project/campaign/provider/model/task/hash/session/batch/story/scene/target 等字段；Production View 仍要求用户手填长 session/batch id；Settings 里服务端连接、Project API key、Basic/Admin 和旧版浏览器直连 provider API profile 的语义容易混淆。

## Product Decision

P2 第一轮命名为 **Web Operator Review Console**。目标是让 Web 默认服务人工审图和交付确认，而不是默认展示 agent/debug metadata。

产品语义锁定：

- 真实 provider key 和真实 provider base URL 属于服务端平台配置，继续放在服务器环境变量或未来明确确认的安全存储中。
- Web Admin 登录者使用这台 Agent ImageFlow 的平台能力；当前不是“每个登录用户自带 provider key/base URL”的账号体系。
- Project API key 继续服务 MCP/CLI/REST 外部 project 调用，不作为人工 Web 查看 Recent Assets 的前置条件。
- 旧版浏览器直连 provider API profile 保留为 advanced/legacy 兼容路径，不作为 Agent ImageFlow 正式资产生产主流程。

## User Scenarios

1. Agent 调 MCP 生成一批萌宠故事图后，用户打开 Web，只想看到图、提示词、剧情/scene、状态和选择动作。
2. 用户在 Recent Assets 里点一张图，能快速进入对应 batch/story/scene Production View，不需要复制很长的 session_id 或 batch_id。
3. 用户想排查问题时，可以展开 Technical details 查看 asset_id、task_id、hash、source/session/batch/story/scene、provider/model、metadata/parameters，但这些字段默认不抢占主视图。
4. 用户登录失败或看不到历史图时，Web 能提示 localhost 与 127.0.0.1 host 混用导致的 cookie host 问题，而不是只显示 unauthorized。
5. 用户点击 manifest 导出后，能看到明确 pending/success/error 反馈。
6. 安全回归持续确保 provider key、Project API key、cookie、Authorization、password、local absolute path 不进入人工 UI 默认展示和 manifest。

## Scope

In scope:

- `issues/next-phase-p2-web-operator-review-console.csv` 作为本轮主入口。
- Web asset card 默认改为 review-first 信息层。
- Technical details 默认折叠。
- Settings 文案和分层改为 server-first。
- Production View 快速带入 recent/current batch 查询字段。
- Manifest export visible feedback。
- Host mismatch guidance。
- 最小 non-exposure regression。

Out of scope:

- 多用户账号体系、注册登录、多租户、RBAC。
- 每个登录用户保存自己的 provider key/base URL。
- 真实 provider key 迁移或展示。
- 小红书发布、内容日历、运营后台。
- 通用 DAM、模板市场、多人协作。
- 服务端 ZIP、WebDAV/SMB server、真实视觉质检 AI。

## Acceptance

- Web 默认审图体验不再要求用户阅读长 ID。
- 调试字段仍可展开查看和复制。
- 现有 MCP/REST/CLI/Web 资产查询、select/reject、Project Context reference、Production View 和 manifest 功能保持兼容。
- 更新 `TASKS.md`、`PROJECT_PLAN.md`、`PROJECT_STATUS_MAP.md`、`CHECKPOINTS.md` 和 `DECISIONS.md`。
- 不运行真实 provider；不读取、打印或处理任何 API key/provider key/secret/cookie/session。

## Implementation

Status: done.

- 新增 `web/src/lib/operatorReview.ts`，集中处理 operator review summary、technical fields、Production View seed filters 和 localhost/127.0.0.1 host mismatch warning。
- 新增 `web/src/lib/operatorReview.test.ts`，覆盖默认审图摘要不含 debug identifiers、Technical details 脱敏、host mismatch warning 和 batch filters 提取。
- `web/src/components/ServerAssetLibrary.tsx` 默认卡片改为 review-first：图片、prompt、story、scene、source、created、target、状态和核心动作优先；asset/task/scope/provider/model/hash/session/batch/metadata/parameters 放入默认折叠的 `Technical details`；资产卡新增 `Batch` 动作，把 session/batch/story/source 作为 seed 打开 Production View。
- `web/src/components/ProductionViewModal.tsx` 支持从 store seed 初始化 filters，并在 seed 变化时保留手动查询路径、清理旧 error/summary 状态；manifest 继续提供 `Exporting` pending、toast success/error 和 inline error。
- `web/src/components/SettingsModal.tsx` 将托管连接文案收束为 Agent ImageFlow 服务端连接：真实 provider key/base URL 属于服务端平台配置，Project API key 主要服务 MCP/CLI/REST 外部调用，Basic/Admin 不是 provider key。
- `web/src/store.ts` 增加 `productionViewSeed`，让 Web asset card 可以在不要求用户复制长 ID 的情况下打开对应 batch/story/scene 视图。

## Verification

- `npm --prefix web test -- --run src/lib/operatorReview.test.ts`: 4 tests passed。
- `npm --prefix web test -- --run`: 18 files / 232 tests passed。
- `npm --prefix web run build`: passed，仅保留既有 Vite chunk size warning。
- `git diff --check`: passed。
- Browser smoke on `http://localhost:4173/`: Admin/Recent Assets 页面可见；默认 asset card 文本不含 `Asset ID` / `Task ID` / `Hash` / `Provider` / `Model`；`Technical details` 存在且默认折叠；资产卡 `Batch` 入口可带入 Production View 的 batch/session filters；页面未显示 secret-like labels。

本轮未运行真实 provider，未读取、打印或处理任何 API key/provider key/secret/cookie/session。

## Follow-up: Compact Review And Real MCP Canary

Status: done.

用户试用后确认两个补充点：`Project Context` 选择器默认占用太多视觉空间，Recent Assets 卡片标题仍可能像长 prompt/debug 文本；另外需要精确跑一次 MCP + 真实 provider 的 1 图 smoke，确认 agent 工具入口也走真实 provider。该 follow-up 仍保持 P2 Web Operator Review Console 范围，不扩展账号系统、provider key 托管、复杂 benchmark 或真实视觉质检。

Implementation:

- `InputBar` 的 `Project Context` 改为默认折叠摘要：保留启用 checkbox、选中数量、recipe/characters/references 摘要、`Expand/Collapse`、`Manage` 和 `Clear`；展开后才显示 recipe/characters/references 详细选择器。
- 新增 `web/src/lib/projectContextPanel.ts` 和测试，集中生成 `No context selected`、`Project defaults enabled`、`Recipe · Characters · references` 等紧凑摘要。
- `ServerAssetLibrary` 卡片标题改用 `getAssetReviewTitle`：优先 `scene_summary/story_summary/caption/description`，否则取 prompt 第一段、移除 `Story scene:` 前缀并截断，减少风格块、渠道块和技术 prompt 对审图首屏的干扰。
- 默认卡片动作进一步收敛为 `Select`、`Reject`、`Original`、`Batch`；`Metadata`、`ID`、`URL`、`Scope`、`Reference` 继续放在默认折叠的 `Technical details` 中。

Real provider canary:

- 只执行 1 图 MCP canary，不做真实 provider benchmark 或并发压测。
- Project/Campaign: `prj_mcp_real_pet_canary_1782109391 / cmp_mcp_real_pet_canary_1782109391`。
- Session/Batch/Story: `mcp_real_pet_canary_session_1782109391 / mcp_real_pet_canary_batch_1782109391 / mcp_real_pet_canary_story_1782109391`。
- MCP `create_image_task` 使用 `provider=openai-compatible`、model `gpt-image-2`、`requested_count=1`，生成 `task_b5256922a91e424850d3` 和 selected asset `asset_7c706c1a1cea00490a40`。
- Batch progress: `task_count=1`、`succeeded_count=1`、`failed_count=0`、`asset_count=1`、`selected_asset_count=1`。
- Task attempt: `attempt_no=1`、`status=completed`、`retry_count=0`、`latency_ms=85569`、`provider_first_byte_ms=68826`、`provider_total_ms=85449`、`response_download_ms=16215`。
- Metadata 保留 `visual_context_snapshot`；thumbnail URL 返回 `image/webp`，original URL 返回 `image/png`；Admin Recent Assets 可用 `source=mcp/session_id/batch_id` 查回该资产。

Verification:

- Red-green: 新增测试前，focused test 因缺少 `getAssetReviewTitle` / `projectContextPanel` 失败；实现后 focused tests 通过。
- `npm --prefix web test -- --run src/lib/operatorReview.test.ts src/lib/projectContextPanel.test.ts`: 2 files / 7 tests passed。
- `npm --prefix web test -- --run`: 19 files / 235 tests passed。
- `npm --prefix web run build`: passed，仅保留既有 Vite chunk size warning。
- Browser smoke on `http://localhost:4173/`: `Project Context` 默认折叠，仅显示摘要、计数、`Expand` 和 `Manage`；默认资产卡动作只保留核心操作；`Technical details` 默认折叠；默认文本不显示 recipe style/channel 长块。

本 follow-up 使用现有服务端 provider 配置完成真实 provider canary，但未读取、打印或处理任何 API key/provider key/secret/cookie/session。
