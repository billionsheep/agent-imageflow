# Slice 067: Final Delivery NAS Readable Mirror

## Status

- State: Done
- Created: 2026-06-27
- Updated: 2026-06-27

## Product Goal

在不改 canonical storage、不中断既有 manifest/cleanup/integrity/audit 语义的前提下，给人工复盘和 NAS/Finder 浏览补一层真正“看得懂”的批次目录。

## Source Context

- Delivery export CSV: `issues/next-phase-p1-final-delivery-nas-readable-export.csv`
- Final delivery contract slice: `docs/project/stories/slice-065-final-delivery-manifest-contract.md`
- NAS governance slice: `docs/project/stories/slice-063-nas-storage-adaptation-and-delivery-governance.md`

## In Scope

- 新增 batch-first readable mirror materialize 能力。
- 默认 mirror root 为 `STORAGE_ROOT/final-delivery-mirror`，可通过 `FINAL_DELIVERY_MIRROR_ROOT` 覆盖。
- mirror 目录结构固定为：

```text
<mirror-root>/
  workspaces/<workspace_id>/
    projects/<project_id>/
      batches/<batch_id>/
        manifest.final.json
        final/<target_path or fallback>
        thumbnails/<target_path>.webp
```

- 只复制 `delivery_role=final_delivery` 的 originals 和现有 thumbnails。
- 优先复用 final asset 的 `target_path`；缺失时回退为 `stories/<story_id>/<scene_id>.<ext>`。
- 新增本地维护命令 `vag storage mirror-final`。
- 新增 Admin 受控 REST：`POST /api/workspaces/{workspace_id}/projects/{project_id}/campaigns/{campaign_id}/final-delivery-mirror`。

## Out Of Scope

- 不改 canonical storage。
- 不新增 story/batch ZIP。
- 不做 project delivery defaults。
- 不做后台自动同步或零触发刷新。
- 不做 WebDAV/SMB server。
- 不把 mirror 目录重新提升为事实源。

## Acceptance Criteria

- 运维可对一个 batch 显式 materialize readable mirror。
- mirror 中必须有 `manifest.final.json`、final originals、thumbnails。
- mirror 只包含 final/selected 交付图，不复制所有候选图。
- `target_path` 允许形成业务子目录，但必须拒绝 path traversal。
- 删除或重建 mirror 不影响 canonical storage、DB、audit、cleanup 或 integrity 语义。

## Verification

- 容器化 `go test ./internal/app ./internal/config ./internal/httpapi ./cmd/vag`
- `git diff --check`
- focused unit tests 覆盖：
  - 目录层级和文件复制
  - `target_path` 复用
  - fallback 命名
  - path traversal 拦截

## Implementation Notes

- 继续复用 `view=final_delivery` 的 `batch-manifest` 作为 mirror 的 manifest 文件内容。
- mirror materialize 采用 staging 目录后再替换 batch 目录，避免半写入目录。
- 第一版显式 materialize 比后台自动刷新更稳，避免把 select/reject 变成隐藏写盘副作用。
