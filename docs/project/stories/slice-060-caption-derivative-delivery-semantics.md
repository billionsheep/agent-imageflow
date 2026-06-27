# Slice 060: Caption Derivative Delivery Semantics

## 背景

真实萌宠试跑证明了 caption derivative 可以生成，但也暴露了交付摩擦：如果运营先选中了 base selected asset，再做 caption edit，最终 `selected_only` manifest 仍然可能落回无字 base 图，除非再手动 select 一次带字派生图。

本切片的目标不是新增 caption UI 或交付状态机，而是在不改数据库 schema、不新增 MCP 工具的前提下，把 caption 派生交付的第一版 contract 固定下来。

## 本片范围

- `metadata_json.caption_lineage` 新增 `auto_select_derivative`
- 服务端继续把归一化后的 `caption_lineage` 写入 `structured_input_json.caption_lineage`
- `manual_optional` caption derivative task 在 `auto_select_derivative=true` 时自动选中第一张派生图
- `batch summary` / `batch manifest` 新增只读 `delivery_role`
- `selected_only` manifest 折叠 scene 内的 `base_original`、`caption_derivative`、`final_delivery`

不做：

- 不改 `asset.status` 枚举
- 不新增数据库表或 migration
- 不新增 MCP delete/setup 工具
- 不做 caption renderer、批量加字 UI 或双分镜创作

## Contract

第一版 `delivery_role` 只读语义：

- `base_original`
- `caption_derivative`
- `final_delivery`

行为规则：

1. base asset 的事实状态继续独立保存，不因 caption derivative 成功而被覆盖。
2. caption derivative 默认只是派生候选。
3. 当 `auto_select_derivative=true` 且任务完成时，第一张 caption derivative 会自动进入 `selected`。
4. 当 caption derivative 已 `selected` 时，`selected_only` manifest 优先交付该派生图。
5. `all-assets` manifest 继续同时保留 base + derivative 事实，便于复盘。

## 实现摘要

- `domain.CaptionLineageSummary` 增加 `AutoSelectDerivative`
- `service.ProcessTask` 在既有 `auto/best_of` 选优之后，补一层 caption derivative 自动选中逻辑
- `service` 在 scene 级别计算 `delivery_role`
- `buildBatchManifest` 在 `selected_only` 路径上只保留 `final_delivery`
- MCP schema、provider parameters 与 caption edit 示例同步 `auto_select_derivative`

## 验证

- 容器化 `go test ./internal/domain ./internal/app ./internal/provider ./internal/mcp`
- `npm --prefix web test -- src/lib/agentImageflowApi.test.ts`
- `python3 -m json.tool examples/mcp/create-caption-edit-task.json`

本轮未运行真实 provider，未读取或打印任何 key、cookie、session 或 secret。
