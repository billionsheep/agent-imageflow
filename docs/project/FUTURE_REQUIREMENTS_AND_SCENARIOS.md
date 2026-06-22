# Future Requirements and Scenario Notes

本文档记录 MVP 试用过程中发现的后续需求、问题和真实业务场景。它不改变第一版 MVP 的冻结边界，而是作为下一阶段拆分 CSV / vertical slice 的输入。

## Current Test Findings

### MCP 生成资产不会自动出现在 Web 画廊

已验证：

- MCP 通过 `create_image_task` 可以创建任务。
- Worker 可以调用 `openai-compatible` provider 生成图片。
- 服务端能登记 `Asset`，并可通过 `get_asset_delivery_info` 返回 `download_url`、`thumbnail_url` 和 `metadata_url`。
- Web 刷新后仍不会自动显示 MCP 创建的任务或资产。

结论：

当前后端事实源已经统一，但 Web 仍主要展示 Web 自己提交并写入前端状态的托管任务。后续需要 Web 具备“同步当前服务端 scope / campaign 资产”的能力。

建议后续能力：

- 在 Web 增加“服务端资产库 / 当前 Campaign 同步”视图。
- 支持从服务端读取当前 `workspace_id / project_id / campaign_id` 下的所有 assets。
- 将 MCP / CLI / REST / Web 创建的任务统一显示。
- 支持手动刷新和后续自动轮询或订阅。

### Web 托管模式仍受 legacy API 配置守卫影响

已验证：

- 开启服务端托管模式后，主按钮一开始仍显示“请先配置 API”。
- 填入 legacy API URL 和占位 key 后，按钮才变为“生成图像”。
- 真实生成仍走服务端托管链路，不走浏览器直连 provider。

结论：

这是前端 UX / 状态守卫问题。托管模式开启且服务端 API / scope / provider 已配置时，不应要求 legacy provider API key。

### Provider base URL 需要明确 OpenAI-compatible 路径规则

已验证：

- `https://cpa.041212.xyz` 被 adapter 拼成 `/images/generations`，返回 `404 Not Found`。
- `https://cpa.041212.xyz/v1` 被 adapter 拼成 `/v1/images/generations` 后成功。

后续需要：

- 在配置说明中明确 `OPENAI_COMPATIBLE_BASE_URL` 应包含 API version path，例如 `/v1`。
- Web / Runbook 中给出错误排查提示：`404` 通常优先检查 base URL 是否缺少 `/v1`。

## Asset Library and Storage Governance

下一阶段需要从“生成工作台”升级出一个“资产库和治理视图”。

### Asset Library View

目标：

- Web 能显示服务端当前 scope 下的所有 assets，而不是只显示当前浏览器创建的任务。
- 所有入口创建的资产都能统一查看：Web、MCP、REST、CLI。

建议筛选维度：

- workspace
- project
- campaign
- session / run / source thread
- provider
- model
- status: generated / selected / rejected / published / deprecated
- source: web / mcp / rest / cli / automation
- created_at
- prompt keyword
- task_id / asset_id

### Storage Visualization

目标：

- 在 Web 管理视图中看到当前实例和每个 scope 的存储占用。

建议统计：

- 总占用。
- 原图占用。
- 缩略图占用。
- metadata 占用。
- input-files 占用。
- audit log 占用。
- asset 数量、task 数量、failed task 数量。
- 最近 7 / 30 天增长。

### Retention and Expiration

需要支持按项目或 campaign 配置资产生命周期。

建议策略：

- `selected` 默认长期保留。
- `published` 默认长期保留。
- `rejected` 可短期过期或批量清理。
- `generated` 且未被选择的候选可按策略过期。
- failed task 的临时文件应可清理。
- orphan files 需要先 reconcile 再清理。

需要避免：

- 未经确认删除 selected / published 资产。
- 只删文件不删数据库记录。
- 只删数据库记录不处理文件。

### Bulk Operations

建议支持：

- 批量 select / reject。
- 批量删除 rejected。
- 批量导出 selected。
- 按 campaign 导出素材包。
- 批量移动资产到另一个 campaign。
- 批量查看和复用 prompt / reference。

### Integrity and Repair View

已有 CLI 能力：

- `vag repair scan`
- `vag repair requeue`
- `vag repair verify-asset`

后续 Web 管理视图可显示：

- 数据库有记录但文件缺失。
- 文件存在但数据库无记录。
- thumbnail 缺失。
- metadata 缺失。
- 长时间 running 的任务。
- provider 失败任务。

## Isolation Model Extension

当前已冻结模型：

```text
Workspace -> Project -> Campaign -> ImageTask -> Asset
```

下一阶段需要考虑会话或运行维度：

```text
Workspace -> Project -> Campaign -> Session/Run -> ImageTask -> Asset
```

第一版不一定需要新增数据库表，可以先通过 `metadata_json` 保存：

- `session_id`
- `run_id`
- `source_thread_id`
- `source_agent`
- `source_channel`
- `content_account`
- `content_series`

用途：

- Codex 某个 thread 生成的一批图片能独立回看。
- n8n / GitHub Actions 某次自动化运行能独立追踪。
- 同一 campaign 下不同会话的试验稿不会混在一起。

## Prompt, Reference, and Lineage Retention

用户明确希望后续修改的原始形象和 prompt 留存。

当前已有：

- `structured_input_json` 保存任务输入快照。
- `asset_version.parameters_json` 保存 provider 参数、reference / mask / generation config 快照。
- `project.metadata_json.quality_profile` 保存项目级质量配置。

后续需要强化：

- Prompt Recipe：可复用 prompt 配方。
- Prompt version：同一配方的修改历史。
- Reference Library：项目内角色、萌宠、品牌、产品、场景参考图。
- Character / Mascot Profile：稳定角色或账号主形象设定。
- Asset lineage：一张图从哪个 reference、哪个 prompt、哪个 seed / provider request 生成而来。
- Edit lineage：一张图被哪次 edit/mask 任务修改成新版本或新资产。

## Project Production Context

本节记录 2026-06-20 的产品讨论结论：下一阶段不优先扩展 workspace 或完整账号系统，而是完善 `project` 这一层，让它承载一个账号、IP、产品线或客户项目的长期视觉生产记忆。

推荐语义：

```text
workspace = 个人/团队/客户/业务大边界
project   = 长期经营对象，例如萌宠账号、嵌入式架构图账号、品牌 IP、产品线
campaign  = 一次具体生产批次，例如一期故事、一组四格漫画、一周封面图
asset     = 生成或上传后可复用、可交付的图片资产
```

对“萌宠账号”场景，推荐结构是：

```text
workspace: ws_personal_media
  project: prj_two_dogs_xhs
    character cards:
      - char_doudou
      - char_mimi
    project references:
      - character reference assets
      - style reference assets
      - scene reference assets
    campaigns:
      - cmp_daily_four_panel_001
      - cmp_origin_story_001
```

需求方向：

- Project Dashboard：查看当前 project 下的 campaigns、assets、characters、references 和基础统计。
- Character Card：项目级角色卡，保存角色外观、性格、禁止项、参考图和主视觉资产。
- Project Reference Library：把现有 asset 标记为项目级参考图，并区分 `character`、`style`、`scene` 等用途。
- Project-level Asset View：跨 campaign 查看当前 project 的资产，避免每次必须进入某个 campaign。
- Task Input Integration：创建任务时允许传 `character_ids` 和 `reference_asset_ids`，服务端展开为 `reference_images` 并写入 metadata / parameters 快照。

第一版建议边界：

- 做 project 级视觉生产上下文，不做通用 DAM。
- 做角色卡和参考图绑定，不做复杂 IP 管理系统。
- 做 project 内复用，不做跨 workspace 素材市场。
- 做服务端 provider key 固定配置，不把 provider key 放到 Web 前端。
- 不做小红书运营、内容日历、自动发布或账号增长分析。

2026-06-22 场景补全：

- P1-PCTX-008 的 Web 第一版入口只服务当前 project 的 visual context 维护：查看 empty / unauthorized / error 状态，维护 characters、reference bindings 和 prompt recipes，从 asset card 标记 reference，并在 Web managed task 创建时选择 recipe / characters / references。
- P1-PCTX-009 的回归只验证萌宠账号图片资产生产链路：clean project/campaign/session/batch、两只狗和一只橘猫角色卡、style reference、`pet_story_cover` recipe、2-3 个 scene task、batch progress、asset metadata 和 Web 可见性。
- 这两个任务的详细场景和最小功能设计以 `docs/project/stories/slice-037-pctx-web-panel-and-pet-story-scenarios.md` 为准；它们不包含小红书发布、内容日历、通用 DAM、Batch / Story / Scene 新 UI、Export Pack、NAS/WebDAV/SMB 或 Usage Tracking。

2026-06-22 Batch Story Export Foundation 收口：

- P1-PCTX-001 到 P1-PCTX-009 完成后，`issues/next-phase-p1-batch-story-export-foundation.csv` 已完成 P1-BSE-001 到 P1-BSE-011。
- 场景设计见 `docs/project/stories/slice-040-batch-story-export-scenarios.md`，收口记录见 `docs/project/stories/slice-050-nas-docker-access-guide-and-regression.md`。
- 第一轮已解决外部 agent 批量生成故事图后的生产查看和交付：batch/story/scene grouped view、scene 级 retry/regenerate、selected-only review、JSON manifest、ZIP 后置边界，以及 NAS/Docker 文件系统访问说明。
- NAS/WebDAV/SMB 判断：第一轮由文件系统和部署环境承担浏览、拷贝、备份；Agent ImageFlow 的 DB / metadata / manifest 继续承担 asset id、状态、prompt、visual context snapshot、scene/batch/story 追踪和审计。暂不在应用内实现 WebDAV/SMB server。

后续若继续推进，需要重新定义 P2 CSV，例如：

```text
issues/next-phase-p2-production-operations.csv
```

## Provider and Credential Model

需要继续区分两类 key：

- Provider key：服务端出站调用外部生图 provider。
- Project API key：外部系统访问 Agent ImageFlow 项目资源。

当前 provider key 通过环境变量进入服务端，不写入仓库。

后续如果对外开放，需要决定：

1. 平台统一提供 provider key。
   - 优点：用户开箱即用。
   - 风险：需要成本、额度、账单、风控。
2. 用户自带 provider key。
   - 优点：平台成本风险低。
   - 风险：需要 project/provider profile、安全保存、加密、轮换和权限说明。

建议下一阶段先支持 project 级 provider profile，但不要把完整 key 明文展示给前端。

## Cloud Deployment and Security

当前 `docker-compose.yml` 是本地开发友好配置，不应直接作为公网部署配置。

已观察：

- API 暴露 `8081`。
- PostgreSQL 暴露 `5432`。
- Redis 暴露 `6379`。

云端自托管建议：

- 只暴露反向代理入口，例如 `443`。
- API 放在内网或仅反向代理访问。
- PostgreSQL / Redis 绝不公网暴露。
- 启用 Basic Auth。
- 每个 project 启用 project API key。
- 启用 rate limit。
- 保留 audit log。
- provider key 只在服务端环境变量或安全配置中保存。

## External User Registration

当前没有自助注册系统。

现阶段对外给别人使用，只能由管理员手动开通：

1. 创建 workspace。
2. 创建 project。
3. 创建 campaign。
4. 给 project 添加 API key。
5. 给用户 API URL、project key 和 scope 信息。

后续如果产品化开放，需要补：

- 注册 / 登录。
- 创建默认 workspace / project / campaign。
- 生成 project API key。
- 配额、限流、过期策略。
- provider key 使用策略：平台 key 或用户自带 key。
- 用量和成本可视化。
- project 级权限和成员管理。

## Scenario A: Xiaohongshu Cute Pet Account

目标：

用户运营一个关于萌宠的小红书账号，需要持续生成封面图、头像、贴纸风格图、栏目图和活动图。

建议空间模型：

```text
Workspace: personal_creator
Project: xhs_cute_pet_account
Campaign:
  - 2026_07_daily_posts
  - avatar_and_mascot_design
  - sticker_pack_v1
  - campaign_summer_pet_care
```

需要保留：

- 账号主形象设定，例如猫、狗、兔子的固定风格。
- 原始参考图。
- prompt recipe。
- 每次修改的 edit lineage。
- selected 封面和 rejected 候选。
- 已发布到哪篇笔记的 usage tracking。

当前能走通：

- 创建 project / campaign。
- Web / MCP / REST 创建图片任务。
- 使用 prompt、reference image、mask/edit descriptor。
- 生成并落盘资产。
- selected / rejected。
- 获取 original / thumbnail / metadata。

当前不完整：

- Web 还不能统一显示 MCP / CLI / REST 创建的所有 assets。
- 没有正式 Reference Library / Mascot Profile 视图。
- 没有 session/run 隔离。
- 没有发布使用记录。
- 没有存储占用、过期策略和批量清理。

## Scenario B: Embedded Project Architecture Diagrams

目标：

另一个账号或项目专门生产嵌入式项目架构图，例如板卡、传感器、MCU、RTOS、数据流、通信链路、模块关系。

建议空间模型：

```text
Workspace: personal_creator 或 engineering_docs
Project: embedded_architecture_diagrams
Campaign:
  - product_a_board_architecture
  - rtos_data_flow
  - sensor_pipeline_article_series
  - release_notes_diagrams
```

需要澄清：

- 如果目标是“AI 生成一张图片风格的架构图”，当前平台可以作为图片资产生成和管理平台承接。
- 如果目标是“可编辑、可 diff、可复用的 Mermaid / D2 / SVG 架构图”，这仍然不是第一版重点，当前产品不完整。
- 嵌入式架构图通常需要事实准确性、模块命名、连线正确和可编辑源文件，不能只依赖生图模型自由发挥。

建议后续能力：

- Diagram source retention：保留 Mermaid / D2 / SVG / Excalidraw / draw.io 源。
- Rendered asset：把源文件渲染成图片资产并进入同一 Asset Registry。
- Prompt + source 双轨：AI 可辅助生成图，但最终以结构化 source 为准。
- 技术图示 review：对标签、连线和模块关系做人工或规则校验。

当前能走通：

- 作为“图片资产”创建、落盘、选优、交付。
- 作为某个 project / campaign 隔离管理。

当前不完整：

- 不保证技术图事实准确。
- 不支持 Mermaid / D2 / SVG 源文件管理。
- 不支持图示语义校验。
- Web 没有技术图源文件编辑和预览工作流。

## Proposed Next Planning Slices

建议后续不要零散改，先按以下 slice 拆分：

1. Web server asset sync
   - Web 能同步当前 scope 下所有服务端 assets。
   - MCP / CLI / REST 创建的资产能在 Web 看到。

2. Asset library filters
   - 按 workspace / project / campaign / provider / status / source / keyword 筛选。

3. Session and source tracking
   - 在任务输入中标准化 `session_id`、`source_thread_id`、`source_agent`。
   - Web 支持按 session/run 查看。

4. Storage usage dashboard
   - 显示总占用、原图、缩略图、metadata、input-files、audit。

5. Retention and cleanup policy
   - 支持 rejected/generated 过期策略。
   - selected/published 默认保护。

6. Reference library and prompt recipes
   - 支持项目级参考图、主形象、prompt recipe 和版本留存。

7. Provider profile management
   - 支持 project 级 provider 配置。
   - 明确平台 key / 用户自带 key 的策略。

8. Cloud deployment hardening
   - 生产 compose override。
   - 关闭 DB/Redis 公网端口。
   - 反向代理/TLS/鉴权/限流/审计默认模板。

9. External onboarding
   - 管理员手动创建用户空间或未来自助注册。
   - API key 发放、禁用、轮换、配额。

10. Diagram source track
    - 如果确认嵌入式架构图要做可编辑技术图，补 source retention 和 renderer，而不是只做 raster image。
