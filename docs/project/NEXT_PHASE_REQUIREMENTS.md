# Next Phase Requirements

本文档定义 MVP 之后第二批需求。目标是把 Agent ImageFlow 从“已经能生成并登记资产的服务端能力”推进到“公共图片资产能力平台的可见、可查、可治理、可集成”。

Current note 2026-06-22: 本文档中的 P0/P1 visibility、资产库、Scope Dashboard、Storage Governance、Provider Profile、Project Production Context、Batch Story Export Foundation、Web Operator Review Console、Deployment Release Pipeline 和 Web Console Auth Gate / Localization / Product Fit 主线已在后续 CSV 中完成或吸收。MCP Service Pack、Character Reference Intake、Web Review Feedback 和 Safe Delete foundation 已在独立 CSV 中推进，剩余项见 `TASKS.md`、`PROJECT_PLAN.md` 和 `PROJECT_STATUS_MAP.md`。Batch / Story / Scene View、JSON manifest、ZIP 后置边界和 NAS/Docker 文件访问的详细场景与收口记录见 `docs/project/stories/slice-040-batch-story-export-scenarios.md` 到 `docs/project/stories/slice-050-nas-docker-access-guide-and-regression.md`；V1 baseline、剩余任务和未来方向见 `docs/project/V1_BASELINE_AND_ROADMAP.md`。后续新需求需要重新确认范围并生成独立 CSV。

## V1 Follow-up Priorities

本文件以下 P0/P1/P2 章节保留为历史需求记录。V1 之后优先级已调整为：

1. Server Deployment Rehearsal：真实服务器/NAS 拉取 GHCR 镜像、配置 HTTPS 反代、跑 mock smoke、演练备份和回滚。
2. Pet Account Real Workflow Trial：低并发真实 provider 小批量跑萌宠账号业务流，观察 agent 调用和 Web 审图摩擦。
3. MCP Service Pack smoke：文档和示例已落地，下一步只需回填 tools/list、mock create task、get task、delivery info evidence。
4. Character Reference Intake 剩余项：资产卡快捷绑定和 OpenAI-compatible MIME 修复已完成；下一步补 mock pet consistency smoke、browser smoke 和人工确认后的 1 图真实参考 canary。
5. Safe Delete 产品化：CLI + Admin-only REST foundation 和 Web 当前 campaign 清理入口已完成；下一步补单 asset restore/soft delete、task/input-file reset、browser smoke 和生产备份演练。
6. Settings Information Architecture：重新设计设置页分组和凭据语义展示，先出方案再实现。
7. Usage Tracking and Edit Lineage：记录资产被哪些内容使用，以及 regenerate/edit lineage。
8. Export Pack ZIP and Multi Select：在 JSON manifest 稳定后补小批量 selected assets ZIP。
9. Deployment and Secret Hardening：补登录失败限流、Secure cookie/HTTPS 检查、服务端配置状态和 non-exposure regression。

继续后置：账号运营系统、小红书自动发布、通用 DAM、模板市场、多人协作、SaaS 注册计费、每用户 provider key 和视觉质检 AI。

## Scope

下一阶段仍然保持平台边界：

- 做公共能力：图片任务、生成、落盘、资产登记、交付、隔离、查询、同步、存储治理。
- 不做业务编排：不读取故事脚本、不拆分小红书内容、不做发布日历、不做账号运营系统。
- 不做重 DAM：只补足平台使用中必需的资产库、控制台和基础治理。
- 不把嵌入式架构图扩成完整图示编辑器；技术图源文件能力单独确认。

## Findings From Simulation

已实际模拟：

1. Web 托管模式可创建服务端任务并显示 Web 自己提交的资产。
2. `openai-compatible` provider 可用真实 base URL + key 生成图片并落盘。
3. MCP 可直接调用 `create_image_task` 创建任务，并通过 `get_asset_delivery_info` 获取交付 URL。
4. 两个独立 project 场景可隔离：
   - `prj_xhs_pet_1781798587` / `cmp_pet_posts_1781798587`
   - `prj_embedded_arch_1781798587` / `cmp_embedded_articles_1781798587`
5. 两个 project 的资产列表和文件路径按 workspace / project / campaign 隔离。

主要缺口：

- Web 不会自动显示 MCP / REST / CLI 创建的资产。
- 当前 Scope 管理只管理 workspace / project / campaign，不显示资产数量、任务数量、存储占用和最近活动。
- 外部批量工作流可以由 Codex 完成，但平台需要保留 `source/session/batch/story/scene/target_path` 等通用追踪字段。
- 云端和对外开放还需要更明确的控制台、key、配额、provider profile 和安全部署路径。

## Priority P0

### 1. Web Server Asset Sync

目标：

Web 能看到当前服务端 scope 下所有资产，不管资产来自 Web、MCP、REST 还是 CLI。

建议实现：

- 在 Web 增加“同步服务端资产”动作。
- 读取当前 `workspace_id / project_id / campaign_id` 下的 assets。
- 将服务端 assets 显示到资产库视图，避免只依赖浏览器本地任务状态。
- 支持刷新后仍能看到服务端资产。

验收：

- 通过 MCP 创建一张图。
- Web 点击同步或刷新资产库后能看到这张图。
- 图卡显示 provider、status、prompt、task_id、asset_id、created_at。
- 可打开 original / thumbnail / metadata。

### 2. Asset Library Minimal View

目标：

提供最小资产库视图，让平台用户知道“我有哪些图、来自哪里、现在什么状态、怎么交付”。

字段：

- thumbnail
- prompt
- provider
- model
- status
- source
- task_id
- asset_id
- project_id
- campaign_id
- created_at
- delivery links

操作：

- select
- reject
- open original
- open metadata
- copy asset_id
- copy delivery URL

暂不做：

- 高级 DAM 标签系统。
- 复杂权限。
- 多人协作审核。

### 3. Platform Console / Scope Dashboard

用户担心忘记曾经创建过哪些 workspace / project / campaign。当前 Scope 管理能列出并切换 scope，但不是控制台。

目标：

在 Web 提供一个控制台式视图，展示所有空间和基础统计。

建议内容：

- workspace 列表。
- 每个 workspace 下的 project 数量。
- 每个 project 下的 campaign 数量。
- 每个 project 的 asset 数量、task 数量、最近活动时间。
- 每个 campaign 的 asset 数量、selected/rejected/failed 数量。
- 当前 scope 标识。
- archived 状态。

验收：

- 用户打开控制台能发现之前创建过的萌宠账号 project 和嵌入式架构图 project。
- 可以从控制台切换当前 scope。
- 可以看出哪个空间最近有生成活动。

### 4. Source / Session Metadata Standard

目标：

Codex 或外部脚本批量生成时，可以把 story、scene、batch 等通用上下文留在资产平台里。

第一版不新增业务表，先标准化 `metadata_json` 字段。

建议字段：

```json
{
  "source": "mcp|web|rest|cli|automation",
  "source_agent": "codex",
  "source_thread_id": "thread_xxx",
  "session_id": "pet_story_session_001",
  "batch_id": "pet_story_batch_2026_07_01",
  "story_id": "story_001",
  "scene_id": "scene_003",
  "target_path": "assets/xhs-pet/story-001/scene-003.png"
}
```

验收：

- MCP / REST 创建任务时传入这些字段。
- `GET /api/tasks/{id}` 和 asset metadata 能保留这些字段。
- Web 资产库能显示或筛选 `source/session/batch`。

## Priority P1

### 5. Storage Usage Minimal Dashboard

目标：

让用户知道当前服务用了多少存储。

建议统计：

- 当前实例总存储。
- 当前 workspace / project / campaign 存储。
- original 总大小。
- thumbnail 总大小。
- metadata 总大小。
- input-files 总大小。
- asset 数量。
- failed task 数量。

验收：

- 在控制台中能看到每个 project/campaign 的粗略存储占用。
- 统计不需要实时毫秒级准确，可以按需刷新。

### 6. Retention and Cleanup Minimal Policy

目标：

提供最小清理能力，但保护重要资产。

建议：

- `selected` 默认保护。
- `published` 默认保护。
- `rejected` 可清理。
- failed task 临时文件可清理。
- generated 未选中候选可按用户确认清理。

验收：

- 用户能看到哪些资产可以清理。
- 删除前明确提示影响。
- 清理后数据库记录和文件状态一致。

### 7. Project Provider Profile

目标：

把 provider 配置从全局环境变量逐步升级为 project 可选配置，方便不同 project 使用不同 provider。

建议先做：

- project 级 provider profile metadata。
- 支持 provider name、base URL、model、key preview。
- 完整 key 不回显。

暂缓：

- 复杂 secret manager。
- 计费系统。
- 多用户权限模型。

## Priority P2

### 8. External Onboarding

目标：

如果未来把平台能力暴露给别人，需要有管理员开通路径。

最小管理员流程：

1. 创建 workspace。
2. 创建 project。
3. 创建 campaign。
4. 添加 project API key。
5. 设置 provider 策略。
6. 给调用方 API URL、scope 和 key。

后续再考虑自助注册。

### 9. Batch Manifest Export

目标：

Codex 负责读取故事脚本和循环生成；平台只负责资产记录和导出 manifest。

建议输出：

```json
{
  "batch_id": "pet_story_batch_2026_07_01",
  "assets": [
    {
      "story_id": "story_001",
      "scene_id": "scene_003",
      "asset_id": "asset_xxx",
      "download_url": "...",
      "metadata_url": "...",
      "target_path": "assets/xhs-pet/story-001/scene-003.png"
    }
  ]
}
```

2026-06-22 更新：Batch Manifest Export 已在 `issues/next-phase-p1-batch-story-export-foundation.csv` 中完成第一版 JSON manifest。服务端 ZIP 已后置，只有在明确确认“小批量 selected assets 一键包”时才另开切片；不做外部发布推送。

### 10. Diagram Source Track

仅当确认嵌入式架构图需要可编辑源文件时再做。

目标：

- 保留 Mermaid / D2 / SVG / draw.io source。
- 渲染输出仍进入 Asset Registry。
- 源文件和渲染图建立 lineage。

暂不纳入 P0。

## Scenario Simulation: Cute Pet Xiaohongshu Account

外部编排：

```text
Codex 读取故事脚本
  -> 拆出 scene prompt
  -> 循环调用 MCP create_image_task
  -> 等待 selected asset
  -> 获取 delivery info
  -> 下载图片到内容仓库
  -> 写 manifest
```

Agent ImageFlow 负责：

```text
接任务
生成图
落盘
登记 asset
保留 prompt/metadata
返回 URL
```

当前可用：

- 结构化任务。
- MCP 调用。
- project/campaign 隔离。
- asset 落盘和交付。

缺口：

- Web 看不到 MCP 创建的资产。
- 没有 batch/session/source 的标准查询视图。
- 没有 manifest 导出。

## Scenario Simulation: Embedded Architecture Diagram Account

外部编排：

```text
Codex 根据技术文章生成配图 prompt
  -> 调 MCP 生成风格化技术封面图
  -> 获取 asset delivery info
  -> 写入文章仓库
```

当前可用：

- 作为 raster image asset 生成和管理。
- project/campaign 隔离。
- metadata 和 prompt 留存。

缺口：

- 如果需要技术图源文件，当前不支持 source retention。
- 如果要求事实准确，需要外部流程或后续 diagram source track 校验。

## Out of Scope For Next Phase

- 故事脚本读取。
- 分镜规划。
- 小红书发布。
- 账号运营系统。
- 完整内容日历。
- 完整企业 DAM。
- 技术图编辑器。
