# Agent ImageFlow Web

本目录是 Agent ImageFlow 的前端工作台，当前基于 `GPT Image Playground` 做二开。

## 当前策略

- 保留参考项目成熟的画廊、任务历史、设置页、Base URL、API Key、多 provider、参考图、遮罩和 Agent 模式基础。
- 将应用品牌、本地存储命名空间和 PWA 信息改为 `Agent ImageFlow`。
- 第一阶段先以前端能力为底座继续二开，再逐步接入 Agent ImageFlow 的服务端资产登记、MCP、审核和交付模型。

## Commands

```bash
npm install
npm run dev
npm run build
npm test
```

## Attribution

This web app is modified based on the open-source project [GPT Image Playground](https://github.com/CookSleep/gpt_image_playground), licensed under MIT. Keep the original license and attribution notices when redistributing modified versions.
