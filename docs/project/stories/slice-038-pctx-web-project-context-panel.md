# Slice 038: P1-PCTX-008 Web Project Context Panel

## Status

- State: Done
- Created: 2026-06-22
- Updated: 2026-06-22

## Product Goal

让 Web 成为当前 project 长期视觉上下文的最小管理入口：管理员可以在同一个事实源里维护角色卡、项目参考图绑定和 Prompt Recipe，并在 Web managed task 创建时选择 recipe / characters / references，把 context ids 交给服务端展开。

## In Scope

- 顶栏 Project Context modal 入口，复用稳定 lazy fallback / preload。
- 读取当前 `workspace/project` 的 `visual_context`，区分 loading / empty / unauthorized / error。
- 新增、编辑、归档 character profile。
- 新增、编辑、归档 project reference binding；asset card 可带入 `asset_id` 打开 Mark as reference。
- 新增、编辑、归档 prompt recipe，支持角色块、风格块、镜头块、渠道块、negative prompt 和 generation_config。
- Web managed task selector 可选择 recipe、characters、references，并在创建任务时发送 `character_ids`、`reference_asset_ids`、`prompt_recipe_id`、`use_project_visual_context`。

## Out Of Scope

- 不做 P1-PCTX-009 萌宠故事完整回归。
- 不做 Batch / Story / Scene 新 UI。
- 不做 Export Pack、NAS/WebDAV/SMB、Usage Tracking。
- 不做小红书发布、内容日历、账号运营后台、通用 DAM、模板市场或多人协作。
- 不运行真实 provider，不读取或打印任何 API key / provider key / secret。

## Acceptance

- Admin session 下可通过 Web 打开 Project Context modal 查看当前 project visual context 摘要。
- 未登录时显示明确 `unauthorized / login required`，不伪装为空数据。
- 空 project 仍显示 characters / references / prompt recipes 的创建入口。
- character / recipe 可新增、编辑、归档；reference binding 可从面板或资产卡创建，不改变 asset 文件或状态。
- Web managed task request body 携带 context selector 字段，并由服务端负责展开最终 prompt / snapshot。
- 长字段使用 `break-words` / textarea / scroll 容器，窄屏不应横向溢出。

## Technical Notes

- Web API client 新增 `getAgentImageflowProjectVisualContext` / `updateAgentImageflowProjectVisualContext`，继续调用既有 `GET/POST /api/workspaces/{workspace_id}/projects/{project_id}/visual-context`。
- Project Context modal 使用 Admin cookie session / Basic fallback，不要求 Web 日常管理手填 project API key。
- Context selector 状态保存在 Web settings：`imageflowUseProjectVisualContext`、`imageflowCharacterIds`、`imageflowReferenceAssetIds`、`imageflowPromptRecipeId`。
- Web 不在前端拼最终 prompt，只把 ids 传给服务端。

## Verification

- `npm --prefix web test -- --run src/lib/agentImageflowApi.test.ts src/lib/apiProfiles.test.ts src/store.test.ts` passed：3 files / 98 tests。
- `npm --prefix web test -- --run` passed：17 files / 226 tests。
- `npm --prefix web run build` passed，仅保留既有 Vite chunk size warning。
- `git diff --check` passed。
- `curl -I http://127.0.0.1:4173/` returned HTTP 200。
- `agent-browser-cli` production preview smoke：刷新 `http://127.0.0.1:4173/` 后 root 可见；点击顶栏 Project Context 后 modal 出现，显示 `ws_default / prj_xhs_anime` 摘要和明确 `unauthorized / login required` 状态，主框架未空白。

## Implementation Log

- Added Project Visual Context web API types, URL builder, GET/POST client and URL test.
- Added settings normalization and store persistence for managed task context selectors.
- Added `ProjectContextModal` with Characters / References / Prompt Recipes sections.
- Added Header entry, lazy module preload, App Suspense wiring and SupportPrompt modal priority.
- Added Server Asset Library `Reference` action to open Project Context with a prefilled `asset_id`.
- Added InputBar managed mode Project Context selector and store request-body wiring.
- Added store regression coverage for managed task context selector fields without requiring a frontend API profile key.
