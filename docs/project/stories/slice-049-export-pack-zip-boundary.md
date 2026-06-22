# Story: 049 - Export Pack ZIP Boundary

## Status

- State: Done
- Created: 2026-06-22
- Updated: 2026-06-22

## Product Goal

明确第一轮是否要实现服务端 ZIP 打包，避免在已经有 JSON manifest 和 NAS/filesystem 文件访问路径的情况下，把 Agent ImageFlow 过早扩成复杂导出系统或通用 DAM。

## Source Context

- Manifest implementation: `docs/project/stories/slice-048-minimal-export-manifest.md`
- Scenario story: `docs/project/stories/slice-040-batch-story-export-scenarios.md`
- Project plan slice: `issues/next-phase-p1-batch-story-export-foundation.csv` / `P1-BSE-010`

## Decision

P1 第一轮不实现服务端 ZIP export。

当前保留：

- REST / CLI / Web JSON manifest。
- NAS / Docker storage root 文件系统访问。
- Web asset original / thumbnail / metadata links。

后续只有在用户明确需要“小批量 selected assets 一键打包”时，才另开 ZIP 实现切片。

## Reasoning

- 真实目标是萌宠账号的图片资产生产和交付追踪，不是做复杂导出系统。
- Manifest 已经提供 agent 可读的资产清单、delivery URL、metadata URL、target_path 和 visual context 摘要。
- NAS / WebDAV / SMB / Finder 更适合承担大文件浏览、复制、备份和批量移动。
- 服务端 ZIP 会引入资源保护问题：文件数量、总大小、临时文件清理、路径安全、并发打包、下载中断和低资源 NAS 压力。
- 过早做 ZIP 容易把平台拉向通用 DAM，而不是专注图片生产事实源。

## Future ZIP Boundary

若未来确认实现 ZIP，应另开切片并满足：

- 只支持小批量 selected assets。
- 默认依赖 manifest 的 selected asset list，不重新发明筛选。
- ZIP 内只包含 original、thumbnail、metadata JSON 和 manifest JSON。
- 文件名必须安全，路径只能来自 manifest-safe `target_path` 或 fallback。
- 必须有数量和大小上限。
- 必须避免本地绝对路径、provider key、project API key、cookie、session token 进入 ZIP。
- 大批量导出仍提示使用 NAS/filesystem + manifest。

## Verification

- Product boundary review only.
- No code changed.
- No tests required for ZIP because no ZIP implementation was added.
- `git diff --check` should remain clean.

## Implementation Log

### 2026-06-22

- Confirmed P1-BSE-010 as a boundary decision: defer service-side ZIP and keep JSON manifest + filesystem access as the first delivery path.
- No real provider run and no API key / provider key / secret / cookie / session token read or printed.
