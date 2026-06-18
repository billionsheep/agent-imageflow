# Story: 001 - Web Foundation From GPT Image Playground

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

将第一版 Web 从低保真自写工作台切换为基于 `/Users/moon/Workspace/tools/gpt_image_playground` 的二开底座，先获得成熟的生图交互、设置页、Base URL/API Key、多 provider、画廊、参考图、遮罩和 Agent 模式能力。

## User Flow

1. 用户打开 `http://localhost:8080`。
2. 用户看到 Agent ImageFlow 品牌下的成熟生图工作台。
3. 用户可以进入设置配置 API URL、API Key、provider 和模型。
4. 用户可以使用参考项目已有的画廊、任务历史、参考图、遮罩和 Agent 模式能力。

## In Scope

- 回退上一版低保真 Web/API/Worker/Docker 实现。
- 将 `gpt_image_playground` 导入为 `web/`。
- 修改应用名、PWA 信息、本地存储命名空间和 Header 品牌。
- 保留原项目 MIT attribution。
- 本地运行、测试、构建验证。

## Out of Scope

- Agent ImageFlow 后端 API。
- PostgreSQL asset registry。
- MCP stdio server。
- Web 生成结果同步到服务端 `ImageTask/Asset`。
- Docker Compose 生产部署。

## Verification

```bash
npm --prefix web install
npm --prefix web test -- --run
npm --prefix web run build
npm --prefix web run dev -- --host 0.0.0.0 --port 8080
```

Browser smoke:

- Page title is `Agent ImageFlow`.
- Header shows `Agent ImageFlow`.
- Gallery / Agent switch and bottom input bar render.
- No 404 network responses after disabling fork release check.

## Implementation Log

### 2026-06-18

- Changes: Rolled back the low-fidelity implementation, removed generated Docker volumes, imported `gpt_image_playground` into `web/`, changed app identity and storage namespaces.
- Verification: 16 test files / 216 tests passed; production build passed; Vite dev server running on `http://localhost:8080`; browser smoke passed.
- Remaining gaps: Connect Web results to Agent ImageFlow service-side asset model, then implement MCP/API/CLI and Docker Compose around the stable Web base.
