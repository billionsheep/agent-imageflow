# Story Slice Index

`docs/project/stories/` 保存 Agent ImageFlow 从早期实现到 V1 baseline 的历史 slice 记录。它们是实现日志、验收证据和设计追溯材料，不是日常必读入口。

## 使用方式

- 新会话默认先读 `docs/project/README.md`、`PROJECT_STATUS_MAP.md`、`TASKS.md` 和 `RUNBOOK.md`。
- 只有需要追溯某个能力的设计或验收证据时，再按 slice 编号查看本目录。
- 不建议在新任务中复开已完成 slice；后续新增能力应新建独立 CSV、story 或 roadmap 条目。

## 编号范围

- `slice-001` 到 `slice-023`：MVP、Web 托管、provider、鉴权、审计和多 key 基础。
- `slice-024` 到 `slice-029`：Storage Governance、资产生产就绪、Web 性能、并发与控制台可见性。
- `slice-030` 到 `slice-039`：Project Production Context、Web UX Smoothness 和萌宠故事 mock 回归。
- `slice-040` 到 `slice-050`：Batch / Story / Scene、Production View、regenerate、manifest 和 NAS 文件访问边界。
- `slice-051` 到 `slice-053`：Web Operator Review Console、Deployment Release Pipeline 和 Server Deployment Rehearsal。

## 维护规则

- 历史 slice 默认保留，不移动、不删除。
- 如果某个 slice 的结论被新决策替代，在 `DECISIONS.md` 或 `PROJECT_STATUS_MAP.md` 中记录当前结论，不直接改写历史证据。
- 新增 slice 时保持编号递增，并在本 README 的编号范围中补一行说明。
