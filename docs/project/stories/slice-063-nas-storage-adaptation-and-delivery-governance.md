# Slice 063: NAS Storage Adaptation And Delivery Governance

## 背景

真实试跑后，产品侧已经确认未来很可能把生成环境落到 NAS 或内网自托管主机上。当前平台物理存储按 `asset_id` 落盘，这对平台事实模型是对的，但如果运维或人工直接把“故事分组”需求投射到物理目录，就很容易破坏 DB / metadata / manifest 的一致性。

## 本片范围

- 固定第一版 NAS/self-host 口径：
  - DB / metadata / manifest 是事实源
  - storage root 通过 bind mount 指向持久目录或 NAS 路径
  - 人工/NAS 访问以只读 SMB/WebDAV/Finder 浏览和复制交付件为主
  - 备份必须同时包含 Postgres dump 与 storage root 一致快照
  - 禁止手动移动/重命名平台管理目录
- 同步 `RUNBOOK.md` 与 `SERVER_DEPLOYMENT_GUIDE.md`

不做：

- 不新增 NAS 子系统
- 不内置 WebDAV/SMB server
- 不把物理目录树改造成 story/campaign 事实源

## 实现摘要

- `RUNBOOK.md` 的 NAS / Docker / WebDAV / SMB access guide 明确了职责边界、备份恢复和 target_path 语义
- `SERVER_DEPLOYMENT_GUIDE.md` 增补 NAS bind mount 的治理口径和只读访问建议
- `docker-compose.prod.yml` 继续复用 `AGENT_IMAGEFLOW_STORAGE_ROOT`、`AGENT_IMAGEFLOW_POSTGRES_DATA`、`AGENT_IMAGEFLOW_REDIS_DATA`

## 验证

- `docker compose -f docker-compose.prod.yml config --quiet`
- 文档 review：确认 bind mount、只读访问、快照一致性和“不要手动改目录”口径齐全

本轮未运行真实 provider，未读取或打印任何 key、cookie、session 或 secret。
