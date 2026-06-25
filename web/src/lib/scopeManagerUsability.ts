export type ScopeDeleteKind = 'workspace' | 'project' | 'campaign'

export interface CascadeScopeDeleteImpact {
  projectCount?: number
  campaignCount?: number
  taskCount?: number
  assetCount?: number
  selectedCount?: number
  publishedCount?: number
}

function formatCount(label: string, value?: number): string | null {
  if (!value || value <= 0) return null
  return `${value} ${label}`
}

function formatImpact(impact: CascadeScopeDeleteImpact): string {
  const parts = [
    formatCount('project', impact.projectCount),
    formatCount('campaign', impact.campaignCount),
    formatCount('任务', impact.taskCount),
    formatCount('资产', impact.assetCount),
    formatCount('已选', impact.selectedCount),
    formatCount('已发布', impact.publishedCount),
  ].filter(Boolean)
  return parts.length > 0 ? `当前统计约包含：${parts.join('、')}。` : '当前统计可能仍在同步，请以确认后的删除结果为准。'
}

export function buildCascadeScopeDeleteMessage(
  kind: ScopeDeleteKind,
  displayName: string,
  impact: CascadeScopeDeleteImpact = {},
): string {
  const name = displayName.trim() || kind
  if (kind === 'campaign') {
    return `确定要删除 campaign「${name}」吗？\n\n此操作会删除该 campaign 下的任务、资产、缩略图、metadata 和原图文件。已选 / 已发布资产也会随 scope 删除。\n\n${formatImpact(impact)}\n\n该操作不可撤销，建议先确认不再需要这个测试或生产批次。`
  }
  if (kind === 'project') {
    return `确定要删除 project「${name}」吗？\n\n此操作会删除该 project 下的所有 campaign、任务、资产、缩略图、metadata 和原图文件。已选 / 已发布资产也会随 scope 删除。\n\n${formatImpact(impact)}\n\n该操作不可撤销，建议先确认不再需要这个项目/IP/产品线。`
  }
  return `确定要删除 workspace「${name}」吗？\n\n此操作会删除该 workspace 下的所有 project、campaign、任务、资产、缩略图、metadata 和原图文件。已选 / 已发布资产也会随 scope 删除。\n\n${formatImpact(impact)}\n\n该操作不可撤销，建议先确认不再需要这个团队/客户/业务空间。`
}
