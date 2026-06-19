# Slice 006: Quality Foundation

## Goal

让 Agent ImageFlow 在服务端保存和复用项目级质量配置，使 prompt template、style preset、reference image 参数和 generation config 能进入同一条 ImageTask 归一化链路，为后续 best-of 自动选优和更稳定的图片质量打底。

## Scope

- 使用现有 `project.metadata_json` 保存项目级 `quality_profile`，不新增数据库表和迁移。
- 新增 REST 入口读取/更新项目质量配置。
- 创建 ImageTask 时支持显式 `use_project_quality_profile`，并允许请求级字段覆盖项目默认值。
- 支持 `prompt_template` + `template_variables` 渲染最终 prompt；默认变量包含原始 prompt、title、purpose、style_preset、aspect_ratio 和 metadata 顶层字段。
- 将有效质量配置快照写入任务 `structured_input_json.metadata_json.quality_profile_snapshot`。
- Web managed mode 创建任务时默认开启项目质量配置复用开关。
- MCP `create_image_task` schema 支持质量配置相关字段。

## Non-goals

- 不接真实 reference image / mask / edit provider 参数。
- 不做 best-of 自动选优。
- 不做 Web 里的完整 Project / Campaign / Quality Profile 管理界面。
- 不新增数据库 schema，也不引入第三方模板引擎。

## Acceptance

- [x] `GET /api/workspaces/{workspace_id}/projects/{project_id}/quality-profile` 返回项目级质量配置。
- [x] `POST /api/workspaces/{workspace_id}/projects/{project_id}/quality-profile` 可以保存项目级质量配置。
- [x] 创建任务传 `use_project_quality_profile=true` 后，服务端会应用项目模板和风格默认值。
- [x] 请求级 `prompt_template`、`style_preset`、`reference_images`、`generation_config` 可以覆盖项目配置。
- [x] 任务返回和数据库里的 `structured_input_json` 包含有效质量配置快照。
- [x] Web managed mode 创建服务端任务时带上 `use_project_quality_profile`。
- [x] Go 测试、Web 测试/构建和本地 mock smoke 通过。

## Readiness Checklist

- [x] 产品边界清楚：本 slice 是质量复用基础，不扩展成设计平台或 DAM。
- [x] 数据边界清楚：复用已有 `project.metadata_json`，不做 schema 迁移。
- [x] 入口边界清楚：REST 保存/读取；REST/MCP/Web 创建任务时进入同一 application core。
- [x] 验收方式清楚：REST profile smoke + create task smoke + existing asset loop smoke。

## Implementation Status

Status: done.

Evidence:

- `go test ./...` 通过。
- `npm --prefix web test -- --run` 通过。
- `npm --prefix web run build` 通过。
- `docker compose config` 和 `docker compose build` 通过。
- REST smoke 跑通：保存 quality profile -> 创建 `use_project_quality_profile=true` 的 mock 任务 -> 模板渲染最终 prompt -> Worker 生成 2 个 asset -> select/delivery 回归通过。
- MCP smoke 跑通：`create_image_task` 传 `use_project_quality_profile=true` 后返回已渲染 prompt 和质量配置快照。
