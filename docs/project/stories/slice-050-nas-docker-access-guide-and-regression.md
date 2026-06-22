# Story: 050 - NAS Docker Access Guide And Regression

## Status

- State: Done
- Created: 2026-06-22
- Updated: 2026-06-22

## Product Goal

把 Batch Story Export Foundation 第一轮收口到一个清晰的自托管交付边界：Agent ImageFlow 继续作为任务、资产、审核状态、visual context 和 manifest 的事实源；NAS、WebDAV、SMB 或 Finder 只承担大文件浏览、复制和备份，不成为新的资产状态系统。

## Source Context

- Scenario story: `docs/project/stories/slice-040-batch-story-export-scenarios.md`
- Manifest implementation: `docs/project/stories/slice-048-minimal-export-manifest.md`
- ZIP boundary: `docs/project/stories/slice-049-export-pack-zip-boundary.md`
- Project plan slice: `issues/next-phase-p1-batch-story-export-foundation.csv` / `P1-BSE-011`

## User Scenario

用户在 NAS 或内网 Docker 上运行 Agent ImageFlow，用一个 project 经营长期萌宠账号视觉上下文，用 campaign 表达一期故事或一组生产批次。外部 agent 负责写故事，另一个 agent 通过 MCP / REST / CLI 批量创建 scene tasks；Web 用 Production View 看每个 scene 的候选图、select/reject、必要时 regenerate，并导出 JSON manifest。

这一流程完成后，用户希望直接在 NAS、Finder、WebDAV 或 SMB 中查看和复制图片文件，但仍希望平台能继续知道哪些资产属于哪个 scene、哪些被选中、哪些被发布或需要保护。

## In Scope

- 文档化 Docker storage root 与 NAS bind mount / volume 的关系。
- 明确文件系统负责浏览、复制、离线备份和人工交付拷贝。
- 明确 DB / metadata 负责 task、asset、selected/rejected/published、project/campaign scope、visual context、manifest、audit 和 integrity。
- 明确 WebDAV / SMB / Finder 共享建议只读，不建议人工移动、重命名或删除平台管理的资产文件。
- 明确备份和恢复必须同时包含 Postgres dump 与 storage root 的一致快照。
- 明确 manifest 与文件系统路径的关系：manifest 使用 delivery URL、metadata URL、target_path 和逻辑 id，不输出部署机本地绝对路径。
- 同步 CSV、TASKS、PROJECT_PLAN、CHECKPOINTS、PROJECT_STATUS_MAP、RUNBOOK 和 DECISIONS。

## Out Of Scope

- 不改业务代码。
- 不实现 WebDAV / SMB server。
- 不实现服务端 ZIP。
- 不做通用 DAM、文件标签系统、复杂目录治理、多人协作或发布系统。
- 不运行真实 provider。
- 不读取、打印或处理 API key / provider key / secret / cookie / session token。

## Acceptance Criteria

- Given 用户把 Docker storage root 挂载到 NAS 路径，when 通过 Finder / SMB / WebDAV 查看文件，then 文档说明这些入口只用于浏览、复制和备份，不能代表资产状态变化。
- Given 用户已经 selected 或 published 某些资产，when 想清理磁盘，then 文档要求优先使用平台 storage governance / cleanup 入口，不手动移动、重命名或删除受保护资产文件。
- Given 外部 agent 导出 batch manifest，when 需要把文件交给下游，then manifest 提供 asset id、task id、scene/story/batch、delivery URL、metadata URL、target_path 和 visual context 摘要；文件系统路径仍由部署环境映射。
- Given 需要做灾备，when 备份或恢复，then 文档要求 Postgres dump 与 storage root 一致快照一起处理，避免 DB 指向不存在的文件或文件没有 DB metadata。
- Given 后续有人继续推进导出能力，when 评估 ZIP 或 DAM，then 项目管理文件明确 P1 第一轮已收口为 JSON manifest + NAS/filesystem，不把 WebDAV/SMB 或目录树做成应用内子系统。

## Technical Approach

本切片只更新文档和项目管理状态：

- 在 RUNBOOK 增加 NAS + Docker + WebDAV/SMB 访问指南。
- 在 DECISIONS 增加文件系统与 DB metadata 的产品边界决策。
- 将 P1-BSE-011 标记为 done，并把 Batch Story Export Foundation 第一轮状态从“下一步”改为“第一轮已完成”。
- 只运行文档安全检查，不执行 provider、认证接口或真实任务。

## Data / Interface Impact

- 无数据库迁移。
- 无 REST / MCP / CLI / Web 接口变更。
- 无 provider 配置变更。
- 文档继续沿用现有 manifest 语义：不返回本地绝对路径，不包含 provider key、project API key、cookie 或 session token。

## Verification

- `git diff --check` passed.
- 人工复核本 story、RUNBOOK、CSV 和项目管理文件边界一致性。

## Implementation Log

### 2026-06-22

- Added NAS / Docker / WebDAV / SMB access guidance.
- Confirmed filesystem is for browse/copy/backup while DB / metadata remains the source of truth for status, traceability, review, visual context and manifest.
- Confirmed WebDAV / SMB / Finder should be read-only for regular access, and platform-managed selected / published files should not be manually moved, renamed or deleted.
- Confirmed backup / restore requires Postgres dump plus a consistent storage root snapshot.
- Ran `git diff --check`; passed.
- No code changed, no real provider run, and no API key / provider key / secret / cookie / session token read or printed.
