# Product Spec

## Product Lock

- Product name: Agent ImageFlow
- One-sentence positioning: 面向 AI agent、自动化系统和内容工作流的图片资产生成、落盘、审核、复用与交付平台。
- Target user: 使用 Codex / Claude / Cursor / 自动化脚本生产内容或业务素材的独立开发者、技术创作者、内容团队和内部工具开发者。
- Core pain point: 现有生图工具大多面向网页人工操作，AI 或自动化流程即使能生成图片，也经常缺少稳定文件路径、URL、asset_id、版本、元数据、审核状态和可复用交付接口。
- Desired outcome: 外部 AI 或系统能通过结构化接口创建图片资产，并拿到可追踪、可审核、可复用、可交付的正式结果。

## Use Context

用户在写小说、做文章配图、生成海报、生产社媒卡片、维护内容系统或搭建自动化流程时，需要让 AI 或脚本批量生成图片。当前常见做法是打开网页生图工具、手工复制 prompt、下载图片、改名、移动文件、上传到目标系统，再人工记录使用位置。这条链路对 Codex / Claude / Cursor / n8n / CI 这类自动化入口很不友好。

Agent ImageFlow 作为中间层接收结构化图片任务，调用本地或云端生图后端，保存图片，登记元数据，并把结果以 MCP、REST API 或 CLI 的形式返回给调用方。

## Primary Scenarios

业务场景排序和核心流程已在 `docs/project/BUSINESS_SCENARIOS.md` 冻结。第一版核心业务流程选定为“内容系统批量生成封面图”，并以小红书/内容账号的 campaign 素材生产作为 demo 方向。

### AI agent 生成内容配图

Codex 或 Claude 在生成文章、README、小说章节、运营文案时，通过 MCP 创建图片任务，平台返回候选图、文件路径、URL 和元数据。用户审核通过后再用于正文、发布系统或素材库。

### 小说与长内容配图

用户为章节、人物、场景、物品或封面创建批量图片任务。平台保留角色设定、风格、参考图、生成版本和审核记录，避免每次从零手工整理。

### 海报与社媒卡片自动化

内容系统或自动化工作流按固定尺寸、风格和渠道生成活动海报、封面图、小红书卡片、公众号封面、Newsletter 配图等视觉资产。

### SaaS / 内部系统调用图片能力

业务后台通过 API 创建图片资产，例如运营素材、报告配图、商品图变体、客服知识卡片或活动 banner，并取得稳定 URL 与状态回调。

## MVP Scope

### Must Have

- 结构化图片任务输入，而不是只接一句 prompt。
- 至少接入一个生图后端；第一阶段先使用 mock provider，随后接一个云端 API provider。
- 生成任务异步执行，支持状态查询。
- 图片保存到本地文件系统或对象存储。
- 每张图片登记 `asset_id`、版本、来源任务、生成参数、provider、文件路径或 URL、状态和时间。
- 支持人工审核状态：draft、approved、rejected、published、deprecated。
- 提供 REST API 或 CLI 的最小入口。
- 提供 MCP tool 草案，让 Codex / Claude 后续可调用。

### Should Have If Cheap

- 生成多张候选图并选择通过其中一张。
- 缩略图预览。
- JSON metadata 导出。
- Markdown / HTML snippet 输出。
- 基础成本记录，例如 provider、模型、估算费用或本地生成标记。
- 简单 webhook 或 callback 机制。

### Later / Non-goals

- 不做泛用设计编辑器。
- 不做白板或拖拽排版工具。
- 不做模板市场。
- 不优先做 Mermaid / D2 / SVG 技术图示系统。
- 不负责正文事实编写。
- 不自动把未审核图片发布到正式内容。
- 不做企业级多租户、SSO、计费和复杂权限，除非进入公开试用阶段。

## User Flow

1. 用户创建内容账号 project，例如“小红书 AI 动漫账号”。
2. 用户创建 campaign，例如“7 天封面图计划”。
3. 用户或 AI 在 campaign 下创建图片任务，说明标题、主题、尺寸、风格、输出数量和是否需要审核。
4. 平台把任务入队，并调用配置好的生图后端。
5. Worker 下载或接收生成结果，保存原图、生成缩略图，并登记资产和版本元数据。
6. 用户在控制台或调用方看到候选图，选择 approve、reject 或 regenerate。
7. 审核通过后，平台返回稳定 URL、本地路径、缩略图、metadata 和可复用交付信息。

## Capability Platform Definition

Agent ImageFlow 是能力平台，不是单一网页工具。它应提供：

- MCP capability: 给 Codex、Claude、Cursor 等 agent 使用。
- API capability: 给 SaaS、CMS、自动化系统、n8n、脚本使用。
- CLI capability: 给本地项目、批处理和 CI 使用。
- Provider adapter capability: 接 ComfyUI、fal.ai、Replicate、OpenAI Images 或其他后端。
- Asset registry capability: 管理资产身份、版本、文件、URL、状态和来源。
- Review workflow capability: 把生成与正式使用分开。
- Delivery capability: 输出本地路径、对象存储 URL、snippet、metadata 和回调。

## Input / Output Lock v0.1

第一版输入和输出已经冻结，详细规格见 `docs/project/INPUT_OUTPUT_SPEC.md`。

### Frozen Inputs

- MCP input: 给 Codex、Claude、Cursor 等 AI agent 使用。
- REST API input: 给外部系统、SaaS、CMS、n8n、自动化脚本使用。
- CLI input: 给本地批处理、开发调试、GitHub Actions 使用。
- Web UI input: 给人工创建、预览和审核任务使用。

### Frozen Outputs

- Task output: `task_id`、状态、错误信息和关联资产。
- Asset identity output: `asset_id`、版本、hash、provider、prompt、业务归属和状态。
- Original file output: 指定资产原图获取。
- Thumbnail output: 缩略图预览。
- Metadata output: JSON metadata。
- Delivery output: 本地路径、下载 URL、缩略图 URL 和 metadata URL。

## Business Isolation Lock v0.1

第一版业务隔离采用：

```text
Workspace -> Project -> Campaign -> ImageTask -> Asset
```

第一版可以只有一个默认 workspace，但任务和资产必须保留 `workspace_id`、`project_id`、`campaign_id`。这样小红书账号、小说项目、海报活动、技术博客等业务不会混在同一个资产池里。

未来可以扩展 Brand Profile、Style Preset Library、Content Calendar、Reference Library、Prompt Recipe、Publishing Status、Usage Tracking 和 Export Pack，但第一版只保留 `project`、`campaign`、`style_preset` 和 `metadata_json`。

## Acceptance Criteria

- Given 一个结构化图片任务，when 调用生成接口，then 系统创建任务并返回 `task_id`。
- Given 任务执行完成，when 查询任务结果，then 可以看到至少一张已落盘的图片资产和对应 `asset_id`。
- Given 一张 draft 图片，when 用户审核通过，then 资产状态变为 approved，并可返回稳定路径或 URL。
- Given Codex/Claude 后续通过 MCP 调用，when 创建图片任务，then 返回结构化 JSON，而不是只返回自然语言说明。
- Given 不同 project / campaign，when 创建任务和资产，then 文件、列表、缩略图和 metadata 默认按业务空间隔离。
- Given 一个内容账号 campaign，when 批量创建封面图任务，then 每个任务都能生成候选图、缩略图和可审核资产。

## Assumptions

- 用户更关心自动化闭环和资产交付，而不是网页端生图体验本身。
- 第一版以内容系统封面图批量生成为核心业务流程，其他业务场景作为后续扩展。
- 第一版的价值来自“AI/系统能可靠拿到图片资产句柄”，不是来自自研模型。
- 先接一个 provider 足够验证产品价值。
- 本地优先或轻量自托管会比一开始做云 SaaS 更适合验证。

## Open Questions

- 审核通过后是否需要直接推送到某个目标系统，例如 Notion、GitHub repo、CMS？

## Implementation Defaults

- 第一阶段 provider 使用 `mock`，后续接一个云端生图 API provider。
- MVP 不考虑本地 GPU 或 ComfyUI。
- 开发环境存储根目录默认使用项目内 `./storage`。
- 部署环境通过 Docker volume 映射到 `/data/agent-imageflow`。
- 第一阶段只提供本地 API 下载 URL、缩略图 URL 和 metadata URL，不直接推送外部系统。
