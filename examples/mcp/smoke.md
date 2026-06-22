# MCP Mock Smoke

本文件记录人工 smoke 的最短路径，只验证本地 `stdio` MCP + `mock` provider。

## 前提

先启动服务：

```bash
docker compose up -d postgres redis api worker
curl -sf http://localhost:8081/healthz
docker compose ps
```

建议在仓库根目录执行以下命令。

## 1. tools/list

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":"list","method":"tools/list","params":{}}' \
  | docker compose exec -T api /app/mcp
```

期望：

- 返回 6 个工具
- 至少包含 `create_image_task`
- 至少包含 `get_asset_delivery_info`

## 2. create mock task

```bash
cat examples/mcp/create-pet-scene.json \
  | docker compose exec -T api /app/mcp
```

记录返回里的 `task_id`。

期望：

- `structuredContent.task_id` 存在
- `structuredContent.status` 初始通常为 `queued`

## 3. get task

把 `<task_id>` 替换成上一步结果，等待 2 到 5 秒后再查：

```bash
printf '%s\n' "{\"jsonrpc\":\"2.0\",\"id\":\"get-task\",\"method\":\"tools/call\",\"params\":{\"name\":\"get_image_task\",\"arguments\":{\"task_id\":\"<task_id>\"}}}" \
  | docker compose exec -T api /app/mcp
```

期望：

- `status` 最终为 `completed`
- 返回里至少有 1 个 `asset_id`

## 4. list assets

```bash
printf '%s\n' '{"jsonrpc":"2.0","id":"list-assets","method":"tools/call","params":{"name":"list_image_assets","arguments":{"project_id":"prj_pet_account","campaign_id":"cmp_pet_story_batch_001","source":"codex","session_id":"pet_story_session_local_demo","batch_id":"pet_story_batch_local_demo","limit":10}}}' \
  | docker compose exec -T api /app/mcp
```

记录目标 `asset_id`。

期望：

- 返回至少 1 张资产
- 资产 metadata 能对上 `source/session_id/batch_id/story_id/scene_id`

## 5. get delivery info

把 `<asset_id>` 替换成上一步结果：

```bash
printf '%s\n' "{\"jsonrpc\":\"2.0\",\"id\":\"delivery\",\"method\":\"tools/call\",\"params\":{\"name\":\"get_asset_delivery_info\",\"arguments\":{\"asset_id\":\"<asset_id>\"}}}" \
  | docker compose exec -T api /app/mcp
```

期望：

- 返回 `original_url`
- 返回 `thumbnail_url`
- 返回 `metadata_url`

## 6. 可选：select 一张候选图

如果第 3 步返回多张候选图，可选做：

```bash
printf '%s\n' "{\"jsonrpc\":\"2.0\",\"id\":\"select\",\"method\":\"tools/call\",\"params\":{\"name\":\"select_image_asset\",\"arguments\":{\"asset_id\":\"<asset_id>\"}}}" \
  | docker compose exec -T api /app/mcp
```

期望：

- 返回状态映射为 `selected`

## 失败时先看哪里

- `docker compose ps`
- `curl -sf http://localhost:8081/healthz`
- `docker compose logs --tail=100 worker`
- `docker compose logs --tail=100 api`

## 这轮 smoke 不做什么

- 不运行真实 provider
- 不写入真实 project API key / provider key / Basic Auth
- 不验证远程 HTTP MCP
