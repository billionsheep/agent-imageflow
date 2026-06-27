# Slice 066: V0.2 Controlled Pet Business Regression Trial

## 背景

`v0.2.0` 的最后一个功能门槛不是再补新接口，而是证明前面已经做完的 MCP-first 生产强化在真实 provider 小样本里确实能协同工作。重点要验证的不是“图片好不好看”，而是：

- `reference_diagnostics` 是否真正对应到 provider 参考参与链路。
- `caption_lineage` 的 `speaker_character_id`、`bubble_anchor`、`tail_direction` 是否会进入任务和交付语义。
- `story_context_v1.panel_plan` 的上一格引用、状态转场和 continuity metadata 是否能稳定写回。
- `view=final_delivery` manifest 和单资产 `asset_summary` 是否已经足够让人工/agent 判断最终交付图。

## 范围

本轮只做受控低并发回归：

- 2 格情侣文案卡。
- 3 格低背景连续故事。
- 少量 caption derivative。
- 总量限制在很小样本内，不做 benchmark，不扩大到大并发真实压测。

不做：

- 服务器升级。
- HTTPS/Caddy 配置改动。
- restore / rollback 演练。
- 新增 Web 创作入口。

## 回归环境

- 本地源码重建后的 `docker compose` API/worker。
- 低并发真实 provider。
- workspace: `ws_default`
- project: `prj_xiaobai_jimao_dog_dialogue_20260626130624`
- campaign: `cmp_cp_dialogue_cards_20260626130624`

## 执行结果

### Dialogue / Caption derivative

- session: `v020_dialogue_session_20260627T124527`
- batch: `v020_dialogue_batch_20260627T124527`
- story: `v020_dialogue_story_20260627T124527`
- progress: `task_count=4`、`succeeded_count=4`、`partial_count=0`、`failed_count=0`、`asset_count=4`

关键 tasks / assets：

- `task_54c9e0a3341de92566bb -> asset_8f24aabd1f63c4b2cbb8`
- `task_3883f47c90b34ca08332 -> asset_fad52365d2ebd8bd3dd5`
- `task_6d6a843a77db9a6a9e68 -> asset_736d841a4ce928f73e63`
- `task_e9720a7d3f269941eea4 -> asset_be095eabeaa7e0d94341`

final-delivery manifest 中，两格最终交付图为：

- `scene_001 -> asset_736d841a4ce928f73e63`
- `scene_002 -> asset_be095eabeaa7e0d94341`

已验证：

- `caption_lineage.speaker_character_id`
- `caption_lineage.bubble_anchor`
- `caption_lineage.tail_direction`
- `caption_lineage.auto_select_derivative=true`
- `delivery_role=final_delivery`
- `provider_reference_participation=resolved_input_files`

### Story continuity

- session: `v020_story_session_20260627T124527`
- batch: `v020_story_batch_20260627T124527`
- story: `v020_story_20260627T124527`
- progress: `task_count=3`、`succeeded_count=3`、`partial_count=0`、`failed_count=0`、`asset_count=3`

关键 tasks / assets：

- `task_5bd73c179aa5b9bcc3fb -> asset_b260ecdb13a33f549201`
- `task_fdcb199470082694a27b -> asset_61d3a44a1f083b4b7204`
- `task_b372741024cacc443aa0 -> asset_e974a229eef0ca68aba7`

final-delivery manifest 中，三格最终交付图为：

- `scene_001 -> asset_b260ecdb13a33f549201`
- `scene_002 -> asset_61d3a44a1f083b4b7204`
- `scene_003 -> asset_e974a229eef0ca68aba7`

已验证：

- panel 2/3 保留 `previous_panel_asset_id`
- continuity 可读到 `panel_index`
- continuity 可读到 `emotion_after`
- `provider_reference_participation=resolved_input_files`

## Delivery spot check

对 5 个最终交付 assets 做了 URL spot check：

- `download_url` 全部 HTTP `200`
- `thumbnail_url` 全部 HTTP `200`
- `metadata_url` 全部 HTTP `200`

这说明 final-delivery manifest、单资产摘要和交付链接已经能被 agent 或人工稳定消费。

## 结论

`V02-MCPH-011` 可标记为完成。

本轮证明：

- `reference_diagnostics`、caption 说话人/气泡锚点、panel continuity、caption derivative final delivery 和 `view=final_delivery` manifest 已形成可用闭环。
- `v0.2.0` 的本地功能发布门槛已满足。

仍未覆盖：

- 服务器 HTTPS/Caddy 同源入口。
- 浏览器 Admin Recent Assets / original / thumbnail / metadata 的服务器侧 smoke。
- restore 与 `IMAGE_TAG` rollback。

这些保留为 `next-phase-p1-server-deployment-rehearsal.csv` 的运维门槛，不再阻塞本地代码和 GHCR 镜像发版。
