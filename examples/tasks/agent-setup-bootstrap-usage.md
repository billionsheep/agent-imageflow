# Agent-Friendly Bootstrap Usage

本文示例说明在不使用 Admin cookie/session、不直连 DB、也不读取 provider key 的前提下，如何为新 agent 准备 `workspace/project/campaign/visual-context`。

## 前提

- 服务端已配置 `AGENT_SETUP_TOKEN`。
- 调用方只把它当成非 destructive bootstrap token。
- 该 token 不可用于 task create、asset select/reject/delivery、archive/restore、cleanup 或 delete。

## 1. 用 REST 列出或创建 workspace

```bash
curl -sS \
  -H "X-Agent-Setup-Token: $AGENT_SETUP_TOKEN" \
  http://localhost:8081/api/workspaces
```

```bash
curl -sS \
  -X POST \
  -H "Content-Type: application/json" \
  -H "X-Agent-Setup-Token: $AGENT_SETUP_TOKEN" \
  http://localhost:8081/api/workspaces \
  -d '{
    "workspace_id": "ws_cp_cards",
    "name": "CP Dialogue Cards"
  }'
```

## 2. 在 workspace 下创建 project

```bash
curl -sS \
  -X POST \
  -H "Content-Type: application/json" \
  -H "X-Agent-Setup-Token: $AGENT_SETUP_TOKEN" \
  http://localhost:8081/api/workspaces/ws_cp_cards/projects \
  -d '{
    "project_id": "prj_xiaobai_jimao_dog_dialogue",
    "name": "Xiaobai Jimao Dog Dialogue",
    "description": "Pet CP dialogue cards"
  }'
```

## 3. 在 project 下创建 campaign

```bash
curl -sS \
  -X POST \
  -H "Content-Type: application/json" \
  -H "X-Agent-Setup-Token: $AGENT_SETUP_TOKEN" \
  http://localhost:8081/api/workspaces/ws_cp_cards/projects/prj_xiaobai_jimao_dog_dialogue/campaigns \
  -d '{
    "campaign_id": "cmp_cp_dialogue_cards_trial",
    "name": "CP Dialogue Cards Trial"
  }'
```

## 4. 读取 project visual context

```bash
curl -sS \
  -H "X-Agent-Setup-Token: $AGENT_SETUP_TOKEN" \
  http://localhost:8081/api/workspaces/ws_cp_cards/projects/prj_xiaobai_jimao_dog_dialogue/visual-context
```

## 5. 更新 project visual context

可直接复用现有示例：

- `examples/tasks/sample-project-visual-context.json`

```bash
curl -sS \
  -X POST \
  -H "Content-Type: application/json" \
  -H "X-Agent-Setup-Token: $AGENT_SETUP_TOKEN" \
  http://localhost:8081/api/workspaces/ws_cp_cards/projects/prj_xiaobai_jimao_dog_dialogue/visual-context \
  --data @examples/tasks/sample-project-visual-context.json
```

## 6. 用 `vag` 自动转发 setup token

`vag` 不需要额外 flag。只要设置环境变量 `AGENT_IMAGEFLOW_SETUP_TOKEN`，它会自动附带 `X-Agent-Setup-Token`。

```bash
export AGENT_IMAGEFLOW_SETUP_TOKEN="$AGENT_SETUP_TOKEN"

docker compose exec -T api /app/vag project context get \
  --workspace ws_cp_cards \
  --project prj_xiaobai_jimao_dog_dialogue
```

```bash
docker compose exec -T api /app/vag project context set \
  --workspace ws_cp_cards \
  --project prj_xiaobai_jimao_dog_dialogue \
  --file /app/examples/tasks/sample-project-visual-context.json
```

## 7. 生产任务仍走原有项目权限

bootstrap 完成后，再使用现有 MCP 6 工具或 project API key 路径去创建任务、查资产、select/reject 和获取 delivery。

不要把 setup token 误当成：

- project API key
- Admin login
- cleanup token
- provider key
