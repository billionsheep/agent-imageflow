# Issues CSV Index

本目录保存按阶段拆分的 CSV 工单。V1 之后，CSV 作为计划、验收和历史证据使用；已完成 CSV 不复开，新需求应新建独立 CSV。

## Active

- `next-phase-p1-server-deployment-rehearsal.csv`：服务器/NAS 部署演练，当前第一优先。
- `next-phase-p1-pet-account-real-workflow-trial.csv`：服务器/NAS 或本地低并发萌宠账号真实业务试用，记录角色一致性、场景漂移、Web 审图和 manifest/NAS 交付摩擦。
- `next-phase-p1-story-continuity-comic-workflow.csv`：Story Bible、Panel Plan、reference roles、Story Review 和连续故事生产工作流。
- `next-phase-p1-caption-edit-lineage.csv`：基于固定 asset 加字 edit 的派生资产谱系、Web 加字入口和批量 caption 工作流。
- `next-phase-p1-settings-information-architecture.csv`：待创建，用于 Settings 信息架构重整。

## Maintenance / Partially Open

- `next-phase-p1-runtime-auth-accessibility-lifecycle-closure.csv`：本地验收已完成；部署环境 replay、Basic Auth 复核和服务器证据回填仍待做。
- `next-phase-p1-character-reference-intake-consistency.csv`：本地 mock + 1 图真实 canary 已完成；部署环境复放和更完整真实业务观察仍待做。
- `next-phase-p1-safe-delete-and-trial-reset.csv`：单 asset archive/restore 已完成；cleanup panel browser smoke、task/input-file reset 和生产备份演练仍待做。
- `next-phase-p2-usage-lineage.csv`：待创建，用于更完整的 usage tracking、edit lineage 和交付使用记录；当前 P1 caption lineage 先解决加字派生最小闭环。

## Completed / Historical

- `next-phase-p0-visibility.csv`
- `next-phase-p0-p1-deployment-auth-scope-project-console.csv`
- `next-phase-p1-asset-library-filters.csv`
- `next-phase-p1-asset-production-readiness.csv`
- `next-phase-p1-batch-story-export-foundation.csv`
- `next-phase-p1-deployment-release-pipeline.csv`
- `next-phase-p1-mcp-service-pack.csv`
- `next-phase-p1-project-production-context.csv`
- `next-phase-p1-provider-profile-cloud-safety.csv`
- `next-phase-p1-provider-throughput-reliability.csv`
- `next-phase-p1-scope-management-usability-followup.csv`
- `next-phase-p1-session-source-tracking.csv`
- `next-phase-p1-storage-governance.csv`
- `next-phase-p1-web-console-auth-localization-product-fit.csv`
- `next-phase-p1-web-console-auth-visibility.csv`
- `next-phase-p1-web-performance-startup.csv`
- `next-phase-p1-web-review-feedback-stability.csv`
- `next-phase-p1-web-ux-smoothness.csv`
- `next-phase-p2-web-operator-review-console.csv`

## Rules

- 不把 CSV 当成长期 backlog 无限追加；完成后只补证据，不复开 scope。
- 新产品方向、新部署演练、新业务试用或新维护专项都新建 CSV。
- CSV evidence 不记录 provider key、project key、Basic/Auth 密码、Admin cookie、session 或完整 secret。
- 若未来移动历史 CSV 到 archive，需要先做引用检查并单独执行 repo cleanup 计划。
