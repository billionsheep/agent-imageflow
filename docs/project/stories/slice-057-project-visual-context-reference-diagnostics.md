# Slice 057 - Project Visual Context Reference Diagnostics

## 背景

真实萌宠业务试跑表明，平台虽然已经能接住 project visual context、reference asset 和 prompt recipe，但在真正创建任务前，并不会明确告诉调用方：

- 当前到底是 `image_backed` 还是 `text_constrained`
- 有没有缺环境参考
- recipe negative prompt 对物种漂移有没有基本覆盖
- 当前 project context 是不是属于“很容易狗变熊”的弱配置

这会让调用方在失败后把问题笼统归给 provider，而不是先修 reference 配置。

## 目标

在不新增数据库表或 migration 的前提下，把 Project Visual Context 的 readiness diagnostics 补进：

- GET project visual context 响应
- task metadata
- task structured input
- batch summary / manifest
- Web Project Context 诊断展示

## 实现

### 后端 contract

新增 `ProjectVisualContextReferenceDiagnostics`，第一版输出：

- `primary_readiness`
- `labels`
- `summary`
- `active_character_count`
- `character_with_image_count`
- `missing_character_image_count`
- `missing_character_ids`
- `active_reference_count`
- `environment_reference_count`
- `image_reference_count`
- `negative_prompt_covers_species_drift`
- `identity_signal_present`
- `provider_reference_participation_risk`

响应与透传规则：

- GET project visual context 顶层返回 `reference_diagnostics`
- 当任务使用 project visual context 时，metadata 和 `structured_input_json` 保留 `project_visual_context_diagnostics`
- batch summary / manifest 通过 `visual_context.reference_diagnostics` 继续暴露

### 诊断规则

第一版只做规则型配置诊断，不做 AI 自动视觉质检：

- `image_backed`: 当前选择或 project 中存在真实角色/参考资产
- `text_constrained`: 没有图像参考，只能依赖文字描述
- `missing_environment_reference`: 没有 `purpose=scene` 的环境参考
- `weak_species_lock`: 角色 identity signal 或 recipe negative prompt 的 species drift 覆盖偏弱
- `provider_reference_participation_risk`: 用于预估当前更像 `likely_resolved_input_files` 还是 `descriptor_only_risk`

## Web

`ProjectContextModal` 新增只读诊断卡：

- 主判断
- 中文标签
- 角色/参考计数
- species drift / identity 检查
- provider 参考参与风险
- 缺图和缺环境参考提示

不改变现有角色卡、reference binding 和 prompt recipe 的编辑流程。

## 测试

后端：

- `internal/app/visual_context_test.go`
- `internal/store/story_continuity_test.go`
- `internal/app/batch_manifest_test.go`

前端：

- `web/src/lib/projectContextPanel.test.ts`
- `web/src/lib/agentImageflowApi.test.ts`

验证命令：

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25.3 go test ./internal/app ./internal/store ./internal/httpapi ./internal/provider ./cmd/api ./cmd/vag
npm --prefix web test -- src/lib/projectContextPanel.test.ts src/lib/agentImageflowApi.test.ts
npm --prefix web run build
```

## 结果

`V02-MCPH-003` 现在已经把“reference 是否足够强、是否缺环境图、是否容易物种漂移”的问题前置成可见诊断，而不是等真实 provider 出图后再倒推。
