# Input / Output Spec v0.1

本文件冻结 Agent ImageFlow 第一版输入、输出和业务隔离边界。后续可以扩展，但第一版实现不应突破这里定义的核心形状。

## Product Boundary

Agent ImageFlow 第一版不是泛设计工具，也不是单纯网页生图工具。它的核心是：

```text
输入图片任务 -> 调用生成后端 -> 保存图片文件 -> 生成缩略图 -> 登记资产 -> 审核状态 -> 对外提供可获取结果
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

- 支持创建任务、查询任务、获取资产、审核资产。
- 所有 API 都应带业务隔离字段或从路径上下文继承业务隔离。
- 第一版不要求复杂鉴权，但接口设计必须预留项目级 API key。

### 3. CLI Input

给本地批处理、开发调试、GitHub Actions 和脚本任务使用。

定位：开发者入口。

要求：

- 可以从 JSON 文件创建任务。
- 可以查询任务和资产状态。
- 可以触发 approve / reject。
- 第一版可作为 smoke test 工具，不要求完整交互体验。

### 4. Web UI Input

给人工创建、修改、预览、审核任务使用。

定位：人工操作与审核入口。

要求：

- 重点是任务台、审核台和资产库。
- 不做 Midjourney 式聊天生图。
- 不做复杂设计画布。
- 第一版 Web UI 可以非常轻，但必须支持缩略图预览和审核动作。

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
- aspect_ratio
- output_format
- requested_count
- provider
- review_required
- metadata_json
- status
- created_by
- created_at
- updated_at
```

说明：

- `workspace_id`、`project_id`、`campaign_id` 用于业务隔离。
- `metadata_json` 用于未来扩展，例如小红书账号、小说角色、海报渠道、客户项目、参考图、发布计划等。
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

用于审核台、列表页和快速预览。

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

第一版只接一个真实 provider 或一个 mock provider。

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
- 不支持未经审核的图片直接进入正式交付流程。
