# Slice 031: P1 Web UX Atomic Scope Switching

## Status

- State: Done
- Created: 2026-06-21
- Updated: 2026-06-21

## Product Goal

减少 Web 端点击 Settings、Scope 和资产卡 scope 操作时的闪烁，让资产库不要看到临时空 scope 或默认示例 scope，从而避免空态、重复刷新和错误列表跳变。

## Source Context

- User feedback: Web 端点很多按钮会有屏幕闪烁，使用不流畅。
- Project plan slice: `issues/next-phase-p1-web-ux-smoothness.csv` 的 `P1-UX-004 Atomic scope switching`。
- Current state: `ServerAssetLibrary` 已完成第一批资产库稳定性修复，但 Settings / ScopeManager 仍可能分步写入 workspace/project/campaign。
- Existing boundary: 不运行真实 provider，不读取、不打印 API key / provider key / secret。

## User Flow

1. 用户在 Settings 中选择 workspace。
2. Web 先在本地 draft 中清空 project/campaign 并加载可选 project/campaign。
3. 只有 workspace/project/campaign 都完整后，Web 才一次性写入全局 settings。
4. 资产库只看到完整 scope 的最终变化，不再经历空 project/campaign 或默认示例 scope。
5. 用户在 Scope Manager 或资产卡 Scope 中设为当前 scope 时，也一次性写入完整 workspace/project/campaign。

## In Scope

- Settings 托管 scope selector 改为本地 draft 选择，完整后再提交。
- Settings 手动兜底 ID 输入避免把不完整 workspace/project/campaign 写入全局 settings。
- Settings 托管 scope 自动补齐逻辑避免提交不完整 scope。
- ScopeManager 的“设为当前托管 scope”只提交必要的完整 scope 字段。
- 同步更新 Web UX Smoothness CSV 和项目管理文档。

## Out of Scope

- 不处理 Scope dashboard stats 后台加载。
- 不处理 lazy modal fallback / preload。
- 不做 TaskCard / AssetCard memo 化和局部重绘。
- 不做真实 provider 生成，不读取或打印任何 secret。
- 不改变后端 scope API、数据库结构或 provider 配置。

## Acceptance Criteria

- Given 用户在 Settings 选择 workspace，when project/campaign 尚未完成选择，then 全局 settings 不写入空 project/campaign。
- Given 用户在 Settings 选择 project，when campaign 尚未完成选择，then 全局 settings 不写入空 campaign。
- Given 用户在 Settings 选择 campaign，when workspace/project/campaign 完整，then 一次性提交完整 scope。
- Given 用户手动输入 workspace/project/campaign，when 三段 scope 未完整，then 只更新本地 draft；when 三段完整并 blur/关闭 Settings，then 一次性提交完整 scope。
- Given 用户在 Scope Manager 设当前 scope，then 只写入 `imageflowManagedMode`、`imageflowWorkspaceId`、`imageflowProjectId`、`imageflowCampaignId`。

## Technical Approach

- 在 SettingsModal 增加 managed scope draft helper，允许本地 draft 保留空 project/campaign，但阻止这些空值进入全局 settings。
- `loadManagedScopes` 使用本地 draft 自动补齐，只有补齐到完整 scope 时才提交。
- create workspace/project 不立即提交不完整 scope，create campaign 保持完整提交。
- Select 和手动兜底输入复用同一套 complete-scope guard。
- ScopeManager set current 改为 partial settings update，避免 spread 整份 settings。

## Data / Interface Impact

- 无 API、数据库或外部数据契约变更。
- Web settings 行为变更：不完整托管 scope 只存在于 Settings 本地 draft，不再进入全局 store。

## Files or Subsystems Likely to Change

- `web/src/components/SettingsModal.tsx`
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

- `normalizeSettings` 仍会为全局 settings 提供默认示例 scope；本 slice 通过阻止不完整 scope 进入全局 settings 来规避闪烁，不重构全局 settings schema。
- 浏览器自动化若仍不可用，本轮以代码路径审查、测试和 build 作为最小验证；最终视觉回归留给 `P1-UX-009`。

## Implementation Log

### 2026-06-21

- Changes: Settings 托管 scope selector、手动兜底 ID 输入和快速创建 workspace/project 现在只在本地 draft 保留不完整 scope；完整 workspace/project/campaign 后才一次性提交全局 settings；关闭 Settings 时会保留上一份完整 scope，避免空 project/campaign 被 `normalizeSettings` 补成默认示例 scope；连续切换 scope 时会忽略旧的下级选项加载结果。ScopeManager 的“设为当前托管 scope”改为 partial settings update，只写入 `imageflowManagedMode`、workspace、project 和 campaign。
- Verification: `npm --prefix web test -- --run` passed 17 files / 224 tests；`npm --prefix web run build` passed with existing chunk-size warning；no real provider was run and no API key / provider key / secret was read or printed.
- Remaining gaps: P1-UX-007 lazy modal fallback/preload 已在后续切片完成；Scope dashboard stats background loading、Task/Asset card render containment 和最终浏览器 Network/Performance 视觉回归仍按 `P1-UX-006`、`P1-UX-008`、`P1-UX-009` 后续推进。
