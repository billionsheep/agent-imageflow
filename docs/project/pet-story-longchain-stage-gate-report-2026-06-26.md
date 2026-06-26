# 萌宠账号连续故事阶段门报告（2026-06-26）

## 结论摘要

本轮已在本地完成一次 **MCP/API-first、5 格、真实 provider、顺序引用上一格 selected asset** 的萌宠连续故事试跑，结论为 **有条件通过**。

这次真正被证明的是：

- 平台可以支持外部 Story Continuity Agent 先产出 `story_context_v1`，再通过 MCP 顺序创建 5 格真实任务。
- 每格都能保留 `panel_index`、`previous_panel_asset_id`、`provider_reference_participation` 和 continuity 摘要。
- selected manifest 和 delivery URL 可以作为正式交付事实源。

这次**尚未被完全证明**的是：

- 5-6 格长链是否已经稳定到可量产。
- Web 审图是否能在真实运营环境中稳定、高效地完成整条故事审看。
- 补环境 reference 后，环境连续性是否能明显进一步提升。

PM 可直接使用的决策句：

- 当前能力已证明 **5 格长链在低并发、人工选图条件下业务可用**，但还不能直接承诺“稳定量产”。
- 若要继续推进下一阶段，**优先补环境 continuity 和 Web 审图复放条件**，不要误开工 Web 主创作入口。
- 若下一轮 6 格或双故事 replay 失稳，优先回到 **环境 reference / 单格 regenerate / 审图登录复放**，而不是扩 caption UI、ZIP、批量运营能力。

## 当前阶段判断

判定：**有条件通过**

原因：

- 5 格全部完成，且每格都有 selected asset。
- `previous_panel_asset_id` 链条完整。
- selected manifest / delivery 可读、可交付。
- 视觉上已达到“业务及格”，但 panel 3 已出现一次轻微动作责任漂移，说明长链稳定性仍然有边界。
- 新开 Web preview 时没有可复用的 Admin session，导致本轮无法在未输入凭据的前提下完成 Production View 审图 replay。

## 本轮执行范围

- 运行方式：MCP/API-first
- 审图原则：Web 只做审图/管理，不做主创作
- provider cap：1
- 每格候选：2
- 总图量：10 张
- regenerate：0
- caption edit：未执行

业务标识：

- `project_id`: `prj_xiaobai_jimao_ref_20260623094956`
- `campaign_id`: `cmp_pet_story_longchain_trial_20260626104430`
- `story_id`: `pet_cupcake_star_finish_story`
- `session_id`: `pet_story_longchain_session_20260626104430`
- `batch_id`: `pet_story_longchain_batch_20260626104430`

故事标题：

- `小白和鸡毛一起放上最后的星星糖`

## 已验证能力

### 1. 生产上下文准备

- 已复用真实萌宠 project，存在真实角色参考资产：
  - `鸡毛` primary/reference
  - `小白` primary/reference
- 已复用 prompt recipe：`cute_duo_interaction`
- 已确认真实参考资产确实进入 provider 侧参与，manifest / metadata 中为 `provider_reference_participation=resolved_input_files`
- **未发现独立环境 reference asset**
  - 本轮必须标记为：**角色 reference-assisted + 环境 text-constrained**
  - 不能把本轮结果宣传为“完整 reference-assisted continuity”

### 2. Story Continuity Agent 产物

已生成并写入统一 `story_context_v1`，包含：

- `story_revision=rev_001`
- `story_plan_hash=sha256:e27709b5dd58c1b5353a8a28933587ac5e2403f247f2e102e63b497146b6bfb6`
- `generation_mode=sequential_previous_panel`
- 5 格完整 panel plan
- `reference_bindings`
- `resolved_reference_assets`
- continuity policy

### 3. MCP 顺序生产闭环

| Panel | Scene | Task | Selected Asset | Previous Panel Asset | 候选数 | 耗时 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `scene_001` | `task_b540b5b4a62533d27f23` | `asset_0336a2f32acc59559332` | - | 2 | 57s | 起始格稳定 |
| 2 | `scene_002` | `task_1c14b311378b553337a6` | `asset_b157000833e9979cee58` | `asset_0336a2f32acc59559332` | 2 | 63s | 因果推进清楚 |
| 3 | `scene_003` | `task_0d3d99f27afce3d9129d` | `asset_bf127ae61fb2e7008483` | `asset_b157000833e9979cee58` | 2 | 69s | 出现轻微动作责任漂移 |
| 4 | `scene_004` | `task_ccd1c2c869df90af086b` | `asset_7f5c8b86e2de89d5f70c` | `asset_bf127ae61fb2e7008483` | 2 | 60s | 后半段动作责任恢复 |
| 5 | `scene_005` | `task_8ff539f8fa8ab3e9246f` | `asset_56c1ec7cc5fab2e3598c` | `asset_7f5c8b86e2de89d5f70c` | 2 | 60s | 收尾明确 |

统计：

- 成功 task：5
- 失败 task：0
- selected asset：5
- generated-but-not-selected asset：5
- regenerate：0

## 主观评估

### 角色一致性

结论：**及格偏上**

- 小白 / 鸡毛的身份、主轮廓、左右站位整体稳定。
- 角色外观没有明显崩坏，也没有出现新增无关角色。
- panel 3 出现一次“谁拿着星星糖”的责任轻微漂移，但 panel 4 又被 prompt 拉回。

### 场景与道具连续性

结论：**及格**

- 主道具链条完整：同一个杯子蛋糕、最后一颗星星糖一直延续到 payoff。
- 背景锚点整体保持在同类暖色室内角落、粉色地毯、浅蓝窗形块面。
- 但环境仍然更像“弱连续同风格背景”，不是强环境 reference continuity。

### 因果清晰度

结论：**通过**

五格的观感基本符合：

- setup：只剩一颗星星糖
- complication：一起比位置
- decision：一起决定怎么放
- execution：真的开始放
- payoff：完成后一起看成品

虽然 panel 3 的具体动作责任有轻微漂移，但整体因果仍能读出来。

## Manifest / Delivery 结论

selected manifest 已验证：

- `scene_count=5`
- `scene_with_selected_count=5`
- `selected_asset_count=5`
- 每格都有 `panel_index`
- panel 2-5 都有 `previous_panel_asset_id`
- 每格都有 `provider_reference_participation=resolved_input_files`
- 每格都存在 `download_url`、`thumbnail_url`、`metadata_url`
- 本轮没有 `caption_lineage`

交付可读性结论：

- **Manifest / delivery 可交付**
- 适合作为 NAS / 文件系统外部交付前的只读事实源

## Web 审图结果

### 本轮能确认的部分

- Web preview 可正常打开登录页。
- 登录页文案明确区分了 `Admin Login` 与 `provider key / Project API Key`。
- 新开 preview 页面时，`/api/admin/me` 返回 `401 unauthorized`，行为与“无现成 Admin session”一致。

### 本轮未能完成的部分

- 因为严格遵守“不读取、不打印、不复用任何凭据”的边界，本轮未输入 Admin 密码。
- 因此无法在 fresh preview 会话中进入 Recent Assets / Production View，不能把这次 5 格故事在 Web 中完整 replay 一遍。

### Web 审图摩擦

- **P1 审图复放摩擦**：没有现成 Admin session 时，外部 agent 无法直接落到 Production View，只能停在登录页。
- **P2 可观察性摩擦**：这次无法在 fresh session 下复放，不代表 Production View 能力不存在，但会影响真实试用复验效率。

判断：

- 这更像 **部署 / 审图入口复放问题**，不是底层资产链路失败。

## 风险分级

### P0 生产阻塞

- 无

### P1 影响量产稳定性

- 当前只有角色 reference，缺少独立环境 reference，环境连续性仍偏 text-constrained。
- panel 3 已出现一次轻微动作责任漂移，说明 5-6 格稳定性还没有被充分证明。
- fresh Web preview 没有可复用 Admin session，真实运营 replay 成本偏高。

### P2 影响运营效率

- Web 审图依赖登录态复放；如果运营同学频繁丢失会话，会影响“按 batch/story 快速复看”的效率。
- 当前 evidence 仍主要依赖 MCP/API + 本地图片人工判断，尚未形成稳定的“登录后即见这一批故事”的操作节奏。

## 问题分类

### 平台资产模型问题

- 本轮未发现阻塞性资产模型缺口。
- `story_context_v1`、`panel_index`、`previous_panel_asset_id`、manifest continuity 摘要都能支撑这次试跑。

### Story Continuity Agent 问题

- panel 3 prompt 虽然保住了合作关系，但动作责任没有完全锁住，说明下一轮故事设计还要更强调“谁做什么”的动作锚点。

### Provider 能力问题

- provider 在 5 格中整体表现合格，但中段出现一次轻微责任漂移，说明长链依然存在语义漂移风险。

### Web 审图体验问题

- fresh preview 下没有可复用登录态时，只能停留在控制台登录页，无法快速 replay 本轮 batch/story。

### 部署 / NAS 运维问题

- 本轮未进入 NAS 只读交付环境复核。
- 但 Web 审图 replay 受登录态影响，属于部署与运营入口可复验性问题。

## 下一阶段建议

应继续做什么：

1. 再跑一轮 **5-6 格长链**，继续保持 `MCP/API-first`、`cap=1`、每格最多 2 候选。
2. 下一轮只新增 **环境 reference** 这一项变量，验证是否能明显改善背景/空间锚点连续性。
3. 在同源、已登录的真实审图环境下 replay 一次本轮 batch/story，确认 Recent Assets / Production View 的可读性和摩擦点。

应后置什么：

1. Web 主创作入口
2. 批量 caption
3. ZIP / renderer
4. benchmark / 高并发

不应误开工什么：

1. 不要把这次 5 格通过直接解读为“可以稳定量产更长故事”。
2. 不要因为 Web replay 摩擦就把主线改成 Web 创作工具。
3. 不要在还没验证环境 reference 价值前，同时混入太多新变量。
