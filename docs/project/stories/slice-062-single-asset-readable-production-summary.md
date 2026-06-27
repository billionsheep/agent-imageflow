# Slice 062: Single Asset Readable Production Summary

## 背景

在 batch manifest 里，story/scene/panel、caption lineage 和 continuity 已经可读；但一旦回到单个 asset detail，信息仍然深埋在 `metadata_json` 和 `structured_input_json` 里，PM/运营很难快速看懂“这张图是什么、从哪来、是不是最终交付”。

## 本片范围

- 在单资产 `asset` / `metadata` 响应补齐只读 `asset_summary`
- 第一版 `asset_summary` 固定包含：
  - `story_id` / `scene_id` / `panel_index`
  - `dialogue` / `caption_text`
  - `derived_from_asset_id` / `derivation_type`
  - `previous_panel_asset_id`
  - `provider_reference_participation`
  - `provider` / `model`
  - `asset_status`
  - `delivery_role`
- Web 标题、概览和技术详情优先消费 `asset_summary`

不做：

- 不暴露 `local_path`
- 不让 Web 从深层 metadata 二次推理业务语义
- 不新增编辑或批量管理入口

## 实现摘要

- `domain.AssetResponse` / `AssetMetadataResponse` 增加 `asset_summary`
- `service.assetResponse` / `assetMetadataResponse` 统一构建 `asset_summary`
- Web `operatorReview` helper 优先显示 `asset_summary` 与 `caption_lineage`
- `asset_summary.asset_status` 在前端继续映射到 `generated/selected/archived` 的产品语义

## 验证

- 容器化 `go test ./internal/domain ./internal/app ./internal/provider ./internal/mcp`
- `npm --prefix web test -- src/lib/agentImageflowApi.test.ts src/lib/operatorReview.test.ts`
- `npm --prefix web run build`

本轮未运行真实 provider，未读取或打印任何 key、cookie、session 或 secret。
