# Slice 061: Provider Partial Success Product Semantics

## 背景

真实业务试跑里，多次出现“请求 2 张候选，但 provider 只回 1 张”的情况。平台底层其实已经把成功返回的 asset 落盘了，但上层调用方和审图者很容易把这类情况误判成“平台不稳定”或“整次任务失败”。

## 本片范围

- 在 task 响应上补齐 `requested_count`、`delivered_count`、`partial_success_reason`、`provider_error_summary`
- 在 batch summary / batch manifest 上补齐同名只读字段
- 固定语义：
  - `completed`：`delivered_count == requested_count`
  - `partially_completed`：`0 < delivered_count < requested_count`
  - `failed`：`delivered_count == 0`
- Web 技术详情直接消费这些字段，不再靠 error message 自己猜

不做：

- 不改 task 状态机方向
- 不新增数据库表或 migration
- 不把 provider 少回候选误判为平台全失败

## 实现摘要

- `domain.Task`、`BatchStorySummaryTask`、`BatchManifestTask` 增加 partial-success 只读字段
- `service.taskResponse` / `enrichBatchStorySummaryTaskRuntimeSemantics` 用实际 asset 数补齐 runtime semantics
- `store.GetBatchStorySummary` 把 `requested_count` 带入 summary 事实源
- Web `agentImageflowApi` 与 `operatorReview` 直接消费这些字段

## 验证

- 容器化 `go test ./internal/domain ./internal/app ./internal/provider ./internal/mcp`
- `npm --prefix web test -- src/lib/agentImageflowApi.test.ts src/lib/operatorReview.test.ts`
- `npm --prefix web run build`

本轮未运行真实 provider，未读取或打印任何 key、cookie、session 或 secret。
