# Input / Output Spec v0.1

本文件冻结 Agent ImageFlow 第一版输入、输出和业务隔离边界。后续可以扩展，但第一版实现不应突破这里定义的核心形状。

## Product Boundary

Agent ImageFlow 第一版不是泛设计工具，也不是单纯网页生图工具。它的核心是：

```text
输入图片任务 -> 调用生成后端 -> 保存图片文件 -> 生成缩略图 -> 登记资产 -> 轻量选优/状态标记 -> 对外提供可获取结果
```

第一版允许未来业务丰富，但不把海报设计、小说配图、小红书运营、CMS 发布等全部做成重功能。它只保留足够的业务隔离和扩展钩子。

## Frozen Inputs

第一版只保留 4 类输入入口。

### 1. MCP Input

给 Codex、Claude、Cursor 等 AI agent 使用。

定位：核心 agent 入口。

要求：

- 输入必须是结构化 JSON。
- 输出必须是结构化 JSON。
- 不依赖 AI 读取网页或手动操作 UI。
- 第一版至少设计 `create_image_task` / `get_task` / `get_asset` / `list_assets` 的 tool schema。

### 2. REST API Input

给外部系统、SaaS、CMS、n8n、自动化脚本使用。

定位：系统集成入口。

要求：

- 支持创建任务、查询任务、获取资产、标记 selected / rejected。
- 所有 API 都应带业务隔离字段或从路径上下文继承业务隔离。
- 第一版不做复杂用户系统；当前实现支持可选实例级 Basic Auth，以及 project 维度的 API key（`X-API-Key` 或 Bearer）。
- 当自托管实例配置了基础限流阈值时，REST API 可返回 `429`、`Retry-After` 和结构化错误 JSON；调用方应把它视为可恢复错误。
- 当调用方需要服务端真实执行 edit/mask 时，当前可使用三类输入来源：
  - 先把输入图片上传到当前 scope 的 `input-files`，再在 `ImageTask.reference_images[].input_file_id` / `mask_image.input_file_id` 中引用。
  - 直接传匿名 `http/https` 远程图片 URL，由服务端在创建任务时抓取并物化到当前 scope。
  - 直接传当前 workspace/project 下已有的 `asset_id`，复用已有资产原图作为 reference / mask 输入。

### 3. CLI Input

给本地批处理、开发调试、GitHub Actions 和脚本任务使用。

定位：开发者入口。

要求：

- 可以从 JSON 文件创建任务。
- 可以查询任务和资产状态。
- 可以查询任务 attempts，用于定位 provider latency、timeout、retry_after 和失败原因。
- 可以运行有限 benchmark；mock benchmark 不产生费用，真实 provider benchmark 必须显式确认费用风险。
- 可以触发 select / reject；当前实现中的 approve / reject 作为兼容命名保留。
- 第一版可作为 smoke test 工具，不要求完整交互体验。

### 4. Web UI Input

给人工创建、修改、预览、选优任务使用。

定位：人工操作与选优入口。

要求：

- 重点是任务台、候选图预览、选优视图和资产库。
- 不做 Midjourney 式聊天生图。
- 不做复杂设计画布。
- 第一版 Web UI 可以非常轻，但必须支持缩略图预览和 selected / rejected 状态标记。

## Unified Input Object

不管来自 MCP、REST、CLI 还是 Web UI，最终都归一成 `ImageTask`。

第一版 `ImageTask` 必须包含：

```text
ImageTask:
- id
- workspace_id
- project_id
- campaign_id
- title
- purpose
- prompt
- negative_prompt
- style_preset
- prompt_template
- template_variables
- reference_images
- character_ids
- reference_asset_ids
- prompt_recipe_id
- use_project_visual_context
- mask_image
- generation_config
- use_project_quality_profile
- aspect_ratio
- output_format
- requested_count
- provider
- selection_mode
- best_of_config
- metadata_json
- status
- created_by
- created_at
- updated_at
```

说明：

- `workspace_id`、`project_id`、`campaign_id` 用于业务隔离。
- `metadata_json` 用于未来扩展，例如小红书账号、小说角色、海报渠道、客户项目、参考图、发布计划等。当前 P1 约定一组跨入口通用字段：`source`、`source_agent`、`source_thread_id`、`session_id`、`run_id`、`batch_id`、`story_id`、`scene_id`、`target_path`。这些字段用于 Codex / MCP / REST / CLI / Web 的来源、会话、批次和目标落盘追踪；未知业务字段继续保留，不新增业务表。
- `selection_mode` 默认 `manual_optional`，传 `auto` 或 `best_of` 时服务端可在多候选生成完成后自动标记一张 selected；这不表示强制人工审核。
- `best_of_config` 用于控制自动选优策略，可由任务输入直接提供，也可通过项目级 quality profile 复用；当前支持 `local_metadata_v1`、`http_judge_v1` 和 `auto_reject_non_selected`。当 `auto_reject_non_selected=true` 且 `selection_mode=auto` / `best_of` 时，服务端会在自动选出推荐图后把其他候选标记为 rejected；这些候选后续仍可被人工重新 select。
- `prompt_template`、`template_variables`、`reference_images`、`mask_image`、`generation_config` 用于质量复用和 provider 参数扩展；第一版可保存/传递这些参数，其中 `openai-compatible` 和 `fal` 已支持通过服务端上传输入文件、匿名远程 URL 和当前项目资产复用来真实执行 edit/mask。
- `character_ids`、`reference_asset_ids`、`prompt_recipe_id`、`use_project_visual_context` 用于复用 project 级长期视觉生产上下文；当前第一版从 `project.metadata_json.visual_context` 展开角色卡、reference binding 和 prompt recipe，展开快照写入 `structured_input_json.visual_context_snapshot` 和 asset `parameters_json.visual_context_snapshot`。
- `reference_images` descriptor 可包含 `id`、`url`、`asset_id`、`input_file_id`、`role`、`source`、`mime_type`、`width`、`height`、`weight`。
- `mask_image` descriptor 可包含 `id`、`url`、`asset_id`、`input_file_id`、`target_image_id`、`source`、`mime_type`、`width`、`height`、`has_mask`；当前 `openai-compatible` 和 `fal` 已支持三类输入来源，Web managed mode 默认仍优先先上传 reference / mask 到服务端 `input-files`，再提交 `input_file_id`。
- 第一版不追求把所有业务字段都结构化成数据库列，避免过早膨胀。

## Business Isolation Model

第一版业务隔离固定为三层。

```text
Workspace
  -> Project
      -> Campaign
          -> ImageTask
              -> Asset
```

### Workspace

代表用户或团队空间。

第一版可以只有一个默认 workspace，但数据模型必须保留 `workspace_id`。

### Project

代表一个业务空间，例如：

- 小红书 AI 动漫账号
- 小说项目
- 技术博客
- 电商店铺
- 客户素材项目

Project 负责隔离资产库、任务列表、风格配置和导出范围。

### Campaign

代表一批有共同目标的生成任务，例如：

- 7 天小红书内容计划
- 一组动漫头像
- 第一卷章节插图
- 某个活动海报素材
- 某篇文章配图

Campaign 负责组织批量任务和资产集合。

## Isolation Requirements

第一版必须满足：

- 任务和资产都带 `workspace_id`、`project_id`、`campaign_id`。
- Web UI 默认只显示当前 project / campaign 的内容。
- 文件路径按 project / campaign 隔离。
- 获取文件时必须能校验资产归属。
- 导出素材时可以按 project / campaign 单独导出。

建议文件路径：

```text
storage/
  workspaces/{workspace_id}/
    projects/{project_id}/
      campaigns/{campaign_id}/
        input-files/
        originals/
        thumbnails/
        metadata/
```

## Frozen Outputs

第一版输出不是“返回一张图片”，而是返回图片资产包。

### 1. Task Output

用于表示生成任务状态。

必须包含：

```text
task_id
status
created_at
updated_at
error_message
asset_ids
```

### 2. Asset Identity Output

用于追踪和复用资产。

必须包含：

```text
asset_id
workspace_id
project_id
campaign_id
current_version
status
hash
provider
model
prompt
parameters_json
created_at
```

### 3. Original File Output

用于获取原图。

必须支持：

```text
GET /assets/{asset_id}/original
```

后续可扩展：

```text
GET /assets/{asset_id}/versions/{version}/original
GET /files/{file_id}
```

### 4. Thumbnail Output

用于选优视图、列表页和快速预览。

必须支持：

```text
GET /assets/{asset_id}/thumbnail
```

第一版至少生成一个缩略图版本。后续可扩展多尺寸缩略图。

### 5. Metadata Output

用于机器继续处理。

必须支持：

```text
GET /assets/{asset_id}
```

返回资产 JSON，包括文件、版本、状态、provider、prompt、业务归属和可交付链接。

### 6. Delivery Output

用于外部系统使用资产。

第一版至少返回：

```text
local_path
download_url
thumbnail_url
metadata_url
```

后续可扩展：

```text
public_url
signed_url
cdn_url
markdown_snippet
html_snippet
webhook_payload
export_pack
```

## Extension Points

为了未来支持小红书账号、小说配图、海报设计和其他业务，第一版保留以下扩展点，但不立即做重功能。

### Provider Adapter

未来可接：

- ComfyUI
- fal.ai
- Replicate
- OpenAI Images
- 自定义 HTTP provider
- 本地 mock provider

当前已接入 `mock`、`openai-compatible`、`fal`；后续可继续补其他 provider。

### Storage Adapter

未来可接：

- 本地文件系统
- MinIO
- S3
- CDN

第一版优先本地文件系统。

### Delivery Adapter

未来可接：

- GitHub repo
- Notion
- CMS
- 小红书运营素材包
- 静态网站
- 对象存储公开链接

第一版只提供文件获取和 JSON metadata。

### Business Modules

未来可扩展：

- Brand Profile
- Style Preset Library
- Content Calendar
- Reference Library
- Prompt Recipe
- Publishing Status
- Usage Tracking
- Export Pack

第一版只保留 `project`、`campaign`、`style_preset` 和 `metadata_json`，不做完整业务模块。

## First Version Non-goals

- 不做泛用设计编辑器。
- 不做图片细节编辑器。
- 不做复杂多租户权限。
- 不做模板市场。
- 不做发布平台。
- 不自动发布到小红书或其他外部平台。
- 不把所有业务场景都做成固定数据库字段。
- 不把未经人工审核作为默认禁止交付条件；正式交付可直接使用 generated/selected 资产。强审核策略留作未来配置。
