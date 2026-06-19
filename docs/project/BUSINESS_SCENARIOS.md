# Business Scenarios v0.1

本文件用于冻结 Agent ImageFlow 第一版业务场景排序，并选出核心业务流程。它不替代 `INPUT_OUTPUT_SPEC.md`，而是回答“先服务哪个真实业务场景”。

## Scenario Ranking

### P0: 其他服务把图片任务交给 Agent ImageFlow

定位：平台基础能力，不算单独业务场景。

原因：

- MCP、REST API、CLI、Web UI 都围绕这个能力展开。
- 没有这个入口，产品会退化成网页生图工具。
- 它是所有业务场景的底层前提。

第一版必须支持：外部调用方提交结构化图片任务，并拿到 `task_id`、`asset_id`、原图、缩略图和 metadata。

### P1: 内容系统批量生成封面图

定位：第一版核心业务流程。

适合第一版的原因：

- 需求清晰：文章、Newsletter、小红书笔记、博客、活动页都需要封面图。
- 输入相对简单：标题、主题、渠道、尺寸、风格、数量。
- 输出稳定：原图、缩略图、下载 URL、metadata。
- 很适合验证 Project / Campaign 隔离。
- 很适合展示批量生成、候选图选优、交付。
- 不会过早引入小说角色一致性、电商商品合规、复杂排版等重问题。

第一版建议以“小红书/内容账号封面图批量生成”作为 demo 业务。

### P2: 自动化工作流按计划生成图片

定位：P1 的自然扩展，不作为 MVP 第一条主线。

原因：

- 它本质上是在 P1 之上增加 schedule、cron、webhook 和重试。
- 适合未来接 n8n、GitHub Actions、定时内容计划。
- 需要先有稳定任务、资产、选优状态和交付能力。

第一版只预留 `metadata_json` 和接口，不做完整调度系统。

### P3: 小说平台生成章节插图

定位：高潜力场景，适合第二阶段。

原因：

- 业务想象力强，适合章节插图、角色设定、场景图、封面图。
- 但会很快要求角色一致性、世界观设定、参考图库、风格连续性。
- 如果第一版直接做小说平台，容易把产品复杂度拉高。

第一版可以把它作为 Project / Campaign 示例，但不做角色库和连续性保证。

### P4: 电商后台生成商品海报

定位：商业价值高，但第一版不优先。

原因：

- 需要商品信息、品牌约束、价格/促销文字、平台规范、合规审核。
- 很可能需要模板、图文排版和商品图融合。
- 容易把产品拖向设计编辑器或营销素材 SaaS。

第一版只保留未来扩展空间，不做电商模板和商品数据接入。

## Core Business Flow v0.1

第一版核心业务流程选定为：

```text
内容系统批量生成封面图
```

更具体地说：

```text
创建 Project: 小红书/内容账号
  -> 创建 Campaign: 7 天封面图计划
  -> 提交一批 ImageTask
  -> 每个任务生成多张候选图
  -> 保存原图和缩略图
  -> 人工或自动选优，标记推荐图
  -> 对外提供 asset_id、原图、缩略图、metadata、下载 URL
```

## Core Function Order

第一版核心功能按以下顺序实现和验证。

1. Project / Campaign 隔离
   - 能创建或使用默认 workspace。
   - 能创建内容账号 project。
   - 能创建一批内容素材 campaign。

2. ImageTask 创建
   - 支持 MCP / REST / CLI / Web UI 中至少一种入口先跑通。
   - 任务必须带 `workspace_id`、`project_id`、`campaign_id`。
   - 任务包含标题、用途、prompt、尺寸、输出数量、provider。

3. 候选图生成
   - 每个任务可以生成 1 到多张候选图。
   - 第一版允许使用 mock provider 或单一真实 provider。

4. 资产落盘
   - 保存原图。
   - 生成缩略图。
   - 写入 metadata。
   - 文件路径按 project / campaign 隔离。

5. 候选图选优
   - 候选图默认是 generated。
   - 用户、agent 或后续自动策略可以将候选图标记为 selected / rejected。
   - selected 表示推荐使用，不是默认交付闸门；小团队场景不要求每张图人工审核。

6. 指定文件获取
   - 按 `asset_id` 获取原图。
   - 按 `asset_id` 获取缩略图。
   - 按 `asset_id` 获取 metadata。

7. 交付信息
   - 返回 `asset_id`、本地路径、下载 URL、缩略图 URL、metadata URL。
   - 后续再扩展 export pack、public URL、signed URL、CMS push。

## MVP Non-goals For Business Scenarios

- 不做完整内容日历。
- 不做自动发布到小红书。
- 不做角色一致性系统。
- 不做电商商品模板。
- 不做复杂图文排版编辑器。
- 不做跨平台投放管理。
- 不做商业账号数据分析。

## Future Expansion Space

第一版保留以下扩展点：

- Brand Profile: 账号定位、风格、禁用元素、常用尺寸。
- Style Preset: 动漫、海报、写实、产品图、公众号封面等风格预设。
- Content Calendar: 按天/周/月生成内容图片计划。
- Reference Library: 角色、品牌、产品、场景参考图。
- Prompt Recipe: 可复用 prompt 配方。
- Publishing Status: 未使用、已用于某篇内容、已发布。
- Usage Tracking: 一张图被哪些内容使用。
- Export Pack: 按 campaign 导出素材包。

## Scenario Clarifications After MVP Trial

以下内容记录真实试用后的后续澄清，不改变第一版核心流程。

### 小红书萌宠账号

如果用户后续运营一个关于萌宠的小红书账号，建议把它作为独立 project，而不是和其他账号混在一起。

建议空间：

```text
Workspace: personal_creator
Project: xhs_cute_pet_account
Campaign:
  - avatar_and_mascot_design
  - 2026_07_daily_posts
  - sticker_pack_v1
  - campaign_summer_pet_care
```

这个场景需要重点保留：

- 账号主形象，例如固定猫/狗/兔子角色。
- 原始参考图和后续修改图。
- prompt recipe 和 prompt 修改历史。
- 每次 edit/mask 的 lineage。
- selected / rejected 候选状态。
- 图片被哪篇小红书笔记使用的 usage tracking。

当前产品可以完成：

- project / campaign 隔离。
- Web / MCP / REST / CLI 创建图片任务。
- 生成、落盘、缩略图、metadata、selected/rejected。
- reference image、mask/edit descriptor 和真实 provider 输入复用。

当前还不完整：

- Web 不能统一显示 MCP / REST / CLI 创建的全部资产。
- 没有正式 Reference Library、Mascot Profile、Prompt Recipe 和 edit lineage 视图。
- 没有 session/run/source_thread 隔离。
- 没有发布使用记录、存储占用和过期策略。

### 嵌入式项目架构图账号

如果另一个账号专门做嵌入式项目架构图，也建议作为独立 project。它和萌宠账号共享平台能力，但业务资产、prompt、参考图和交付目标不同，不能混在同一个 project。

建议空间：

```text
Workspace: engineering_docs 或 personal_creator
Project: embedded_architecture_diagrams
Campaign:
  - product_a_board_architecture
  - rtos_data_flow
  - sensor_pipeline_article_series
  - release_notes_diagrams
```

需要澄清：

- 如果只是生成“图片风格的技术架构图封面/插图”，当前图片资产闭环可以承接。
- 如果需要可编辑、可 diff、可长期维护的 Mermaid / D2 / SVG / draw.io 源文件，当前产品还不完整。
- 嵌入式架构图对事实准确性、模块命名、连线方向和接口关系要求更高，不能只依赖生图模型自由发挥。

后续如果确认要支持该方向，应补：

- Diagram source retention：保存 Mermaid / D2 / SVG / draw.io 等源文件。
- Rendered asset：把图示源渲染成图片资产并进入 Asset Registry。
- Prompt + source 双轨：prompt 用于辅助生成，正式交付以结构化 source 为准。
- 技术图示 review：对标签、连线和模块关系做校验。
