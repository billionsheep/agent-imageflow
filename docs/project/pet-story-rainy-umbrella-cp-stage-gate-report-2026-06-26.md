# 明确情侣向雨天共伞阶段门报告（2026-06-26）

## 结论摘要

本轮已完成一次 **MCP/API-first、6 幅、明确情侣向、复杂雨天场景、真实 provider、顺序引用上一格 selected asset** 的真实业务试跑，结论为 **有条件通过**。

这次真正被证明的是：

- 平台可以支持外部 Story Continuity Agent 先产出完整 `story_context_v1`，再通过 MCP 顺序创建 6 幅真实任务。
- `panel 2 -> panel 6` 的 `previous_panel_asset_id` 链条完整，角色参考图和上一格 selected asset 都真实参与了后续生成。
- 单格顺序生图主链和“后置组合成上下两格发布版”都能跑通。
- 最终 6 幅 selected assets 的 `original / thumbnail / metadata` 交付 URL 都可访问。

这次尚未被完全证明的是：

- 6 幅复杂场景是否已经稳定到可量产。
- 环境连续性是否足够强；当前仍然缺少独立环境 reference。
- fresh Web preview 下，运营是否能在不额外处理登录态的情况下直接高效 replay 这一批故事。

PM 可直接使用的决策句：

- 当前能力已证明 **6 幅情侣向连续故事在低并发、人工选图条件下业务可用**，但还不能直接承诺“稳定量产”。
- 若继续推进下一阶段，**优先补环境 reference 与 Web 审图 replay 条件**，不要误开工 Web 主创作入口。
- “上下两格”目前已证明 **交付组织可行**，但**不是原生双分镜创作能力**。

## 当前阶段判断

判定：**有条件通过**

原因：

- 6 幅全部完成，且每格都有 selected asset。
- `previous_panel_asset_id` 链条从 panel 2 到 panel 6 完整。
- 实际总图量 12 张，未超过本轮预算。
- 角色 identity 和情侣关系整体稳定，复杂元素如雨、风、伞偏转、路灯倒影基本成立。
- 但 panel 3 发生一次需要 regenerate 的 continuity 偏移；panel 1 和 panel 3 的首个 task 都出现 provider partial 返回。
- selected manifest 的 continuity 信息可读，但 `delivery` 字段没有直接给出最终 URL，本轮需要额外依赖 `get_asset_delivery_info` / asset endpoints 做 delivery 确认。

## 本轮执行范围

- 运行方式：MCP/API-first
- 审图原则：Web 只做审图/管理，不做主创作
- provider cap：1
- 每格候选上限：2
- 实际 task 数：7
- 实际产出图量：12
- regenerate：1 次，仅用于 panel 3
- caption edit：未执行

业务标识：

- `project_id`: `prj_xiaobai_jimao_ref_20260623094956`
- `campaign_id`: `cmp_pet_story_rainy_umbrella_cp_trial_20260626115435`
- `story_id`: `pet_rainy_umbrella_cp_story`
- `session_id`: `pet_story_rainy_umbrella_cp_session_20260626115435`
- `batch_id`: `pet_story_rainy_umbrella_cp_batch_20260626115435`
- `story_revision`: `rev_001`
- `story_plan_hash`: `sha256:d392353ec86b4cdb9c9e19ea5302a6c168d6e6306c2ab19f993973fab47d64ff`

故事标题：

- `小白和鸡毛的雨天共伞靠近`

## 本轮脚本

6 幅脚本固定为单一因果链：

1. 小白在便利店门口等雨，鸡毛举伞跑来接它。
2. 鸡毛把伞举到小白头顶，明确“来接你”变成“照顾你”。
3. 小白主动更靠近一点，进入同一把伞下。
4. 风吹来后，鸡毛把伞偏向小白，自己多淋一点。
5. 两只停下对视，小白明显心动。
6. 两只共伞并肩走远，甜蜜收尾。

这轮对白是简单、直接、明确情侣向的短句，不写模糊猜测式台词。

## 已验证能力

### 1. 真实 project / 视觉上下文准备

- 已复用真实萌宠 project。
- 真实角色 reference 参与：
  - `鸡毛` primary/reference
  - `小白` primary/reference
- 已复用 prompt recipe：`cute_duo_interaction`
- `provider_reference_participation=resolved_input_files`
- panel 2 以后额外加入 `previous_panel_reference=<上一格 selected asset>`

结论：

- **真正使用了 reference assets**
- 但 **没有独立环境 reference asset**
- 本轮必须继续标记为：**角色 reference-assisted + 环境 text-constrained**

### 2. Story Continuity Agent 产物

统一写入了 `story_context_v1`，包含：

- `story_bible`
- 6 幅完整 `panel_plan`
- `reference_bindings`
- `resolved_reference_assets`
- `continuity_policy.mode=sequential_previous_panel`

### 3. MCP 顺序生产闭环

| Panel | Scene | Task | Selected Asset | Previous Panel Asset | 实际候选 | 耗时 | 结果 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `scene_001` | `task_8981b5543e36819aa37f` | `asset_b51c304625ba1f1bae6d` | - | 1/2 | 127s | partial 成功；另一候选 `http_524` |
| 2 | `scene_002` | `task_fbc2cb6cb03e9d0ef47d` | `asset_7e99bd54b26aa250398d` | `asset_b51c304625ba1f1bae6d` | 2/2 | 87s | completed |
| 3 | `scene_003` attempt 1 | `task_0b84dfda8b6e003163e1` | - | `asset_7e99bd54b26aa250398d` | 1/2 | 83s | partial；候选 `asset_45d79efa84d68b79aef5` 因无故出现 cupcake 被 reject |
| 3 | `scene_003` attempt 2 | `task_87719cb68a84fbd1036c` | `asset_cebeb538195aa88da860` | `asset_7e99bd54b26aa250398d` | 2/2 | 80s | completed；本轮唯一 regenerate |
| 4 | `scene_004` | `task_7fa7fe33bb5a046c5a28` | `asset_859797f0d544c2dcc66c` | `asset_cebeb538195aa88da860` | 2/2 | 80s | completed |
| 5 | `scene_005` | `task_a0eb92768fd0f4674fd3` | `asset_0602f95a1da2d12c3436` | `asset_859797f0d544c2dcc66c` | 2/2 | 79s | completed |
| 6 | `scene_006` | `task_d008362900cba74d23dd` | `asset_99e1ea0ddcf8e9b01072` | `asset_0602f95a1da2d12c3436` | 2/2 | 87s | completed |

统计：

- selected panels：6/6
- failed tasks：0
- partial tasks：2
- rejected assets：1
- regenerate：1
- 实际总图量：12

## 主观评估

### 角色一致性

结论：**及格偏上**

- 小白 / 鸡毛的身份、主轮廓、主体颜色和基本站位整体稳定。
- “明确 CP 情侣”在 panel 1、3、5、6 都能读出来，不是朋友式并排共处。
- panel 4 因风和动作张力拉开了部分身体距离，但“保护优先级”是清楚的。

### 场景 / 道具连续性

结论：**及格**

- 主道具链条稳定：透明雨伞始终保留。
- 雨、便利店门口、暖黄店内光、湿地面倒影、街边路灯大多能延续。
- 但镜头距离、路灯是否突出、便利店与街边的相对位置仍有轻微漂移。
- 这次不能宣传为“环境 reference-assisted continuity 成功”，因为环境仍然是 text-constrained。

### 因果清晰度

结论：**通过**

整体观感符合：

- `arrival`
- `care`
- `closeness`
- `protective turn`
- `heart racing`
- `sweet walkoff`

panel 3 首次尝试失败后，第二次通过 regenerate 把因果动作拉回来了。

### 明确情侣语义

结论：**通过**

- panel 1 的“来接你”、panel 3 的主动靠近、panel 4 的伞偏向保护、panel 5 的心动、panel 6 的并肩走远，已经足够构成明确情侣读感。
- 这轮不是暧昧猜测路线，而是明确甜蜜路线。

## 上下两格交付测试

本轮没有把“上下两格”当成原生创作模式，而是作为 **交付组织测试** 执行：

- 先完成 6 幅单格顺序生图主链。
- 再将 selected assets 组合成 3 张上下两格发布版。

结果：

- **成功组合出 3 张上下两格发布版**
- 证明当前平台产出的 selected assets 可以被组织成小红书式发布稿
- **不能据此宣称平台已原生支持双分镜单图创作**

## Manifest / Delivery 结论

### 已确认的部分

- selected manifest 的 `assets[]` continuity 中包含：
  - `panel_index`
  - `previous_panel_asset_id`
  - `provider_reference_participation`
  - `caption_lineage` 本轮为空
- 6 个 selected assets 的以下 URL 均已验证可访问：
  - `download_url`
  - `thumbnail_url`
  - `metadata_url`

### 本轮暴露的问题

- selected manifest 的 `assets[]` 中，`delivery` 字段本轮返回为 `null`
- 也就是说：
  - continuity 摘要在 manifest 里有
  - 但最终交付 URL 不能只靠 selected manifest 一次拿齐
  - 本轮需要额外依赖 `get_asset_delivery_info` 或已知 asset endpoints 做补充确认

结论：

- **Manifest / delivery 基本可交付**
- 但 manifest 的 delivery block 还不够完整，属于需要 PM 看见的结构性问题

## Web 审图结果

### 本轮能确认的部分

- Web preview 可正常打开登录页。
- 登录页明确区分了 Web Admin 登录与 provider key / Project API Key。

### 本轮未能完成的部分

- fresh preview 下没有现成 Admin session。
- 在严格遵守“不读取、不打印、不复用任何凭据”的前提下，本轮未进入控制台内部。
- 因此无法在 fresh session 下实际复看 `Recent Assets / Production View` 对这次 6 幅故事的展示效果。

### Web 审图摩擦

- **P1 审图 replay 摩擦**：无现成登录态时，外部 agent 只能停在登录页。
- **P2 审图可观察性摩擦**：这一轮无法在 Web 内直接验证 `story / scene / panel / selected / continuity` 的页面可读性，只能用 API/manifest 侧证。

结论：

- 这更像 **Web 审图体验 + 部署复验入口问题**
- 不是图片生产主链失败

## 本地落盘 / NAS 分组现状

本轮核查确认：

- 当前真实物理落盘仍按 `workspace / project / campaign / originals|thumbnails|metadata / <asset_id> / version` 的方式组织。
- 这次故事属于哪一组，当前主要靠：
  - `session_id`
  - `batch_id`
  - `story_id`
  - `scene_id`
  - `target_path`
  - manifest / metadata / batch summary

结论：

- **组别语义主要在 metadata / manifest 层，不在物理文件夹层级**
- 这和平台作为“资产事实源”的定位是一致的

对未来 NAS 的产品建议：

- 优先考虑 **发布镜像目录 / manifest-safe 导出目录**
- 不建议把平台内部事实存储改成依赖 story 文件夹表达状态

## 风险分级

### P0 生产阻塞

- 无

### P1 影响量产稳定性

- panel 1、panel 3 首 task 都发生 provider partial 返回，说明低并发也会遇到不完整候选。
- panel 3 首次生成无故引入 cupcake，说明 6 幅长链下仍有道具漂移风险。
- 当前没有独立环境 reference，环境连续性仍偏 text-constrained。
- selected manifest 没有直接给出 delivery block，需要额外调用补齐。

### P2 影响运营效率

- fresh Web preview 无登录态时无法直接 replay 本轮故事。
- 上下两格当前只能在交付端后处理组织，不是平台内原生生产能力。

## 问题分类

### 平台资产模型问题

- selected manifest 的 continuity 信息在 asset rows 中可读，但 `delivery` 字段为 `null`，需要额外调用 `get_asset_delivery_info` 补足。
- 物理目录仍按 asset_id 分组，story 组别语义不在文件夹层级；这本身不是 bug，但 PM 需要明确认知。

### Story Continuity Agent 问题

- panel 3 首次 prompt 对“禁止新增食物道具”约束不够强，导致需要单格 regenerate。
- 后续复杂故事脚本应更明确谁负责动作、什么道具绝不能新增。

### Provider 能力问题

- panel 1 遇到 `http_524`
- panel 3 首尝试遇到 `http_504`
- 复杂场景下 provider 会用心形点缀、镜头拉近等方式补“甜感”，但不总是严格服从动作责任

### Web 审图体验问题

- 新开 preview 时只能停在登录页，无法在无凭据前提下完成 Production View replay。

### 部署 / NAS 运维问题

- 当前内部存储和未来 NAS 运营分组心智不一致：前者按 asset 事实存储，后者更想按发布组别浏览。
- 建议后续通过导出镜像目录解决，而不是重写内部资产目录事实。

## 下一阶段建议

应继续做什么：

1. 下一轮只新增 **环境 reference** 这一项变量，验证是否能改善背景锚点和镜头连续性。
2. 在已登录 Web 环境中 replay 这次 `campaign / story / batch`，只验证审图可读性，不改 Web 主创作边界。
3. 再跑一轮 5-6 幅复杂故事，但保持 `cap=1`、每格最多 2 候选、不做 benchmark。

应后置什么：

1. Web 主创作入口
2. 批量 caption
3. ZIP / renderer
4. benchmark / 高并发
5. 原生双分镜单图创作

不应误开工什么：

1. 不要把这轮“上下两格可组合”误报成平台已支持原生双分镜创作。
2. 不要因为 Web replay 摩擦就把主线改成 Web 创作工具。
3. 不要在环境 reference 价值还没验证前，同时混入太多新变量。
