import { describe, expect, it } from 'vitest'
import { buildCascadeScopeDeleteMessage } from './scopeManagerUsability'

describe('buildCascadeScopeDeleteMessage', () => {
  it('warns that deleting a workspace removes nested scopes, tasks, assets and files', () => {
    const message = buildCascadeScopeDeleteMessage('workspace', '萌宠账号', {
      projectCount: 2,
      campaignCount: 3,
      assetCount: 8,
      selectedCount: 2,
    })

    expect(message).toContain('萌宠账号')
    expect(message).toContain('project')
    expect(message).toContain('campaign')
    expect(message).toContain('任务')
    expect(message).toContain('资产')
    expect(message).toContain('文件')
    expect(message).toContain('已选')
    expect(message).toContain('已发布')
    expect(message).not.toContain('只有当')
  })

  it('keeps campaign copy focused on the selected campaign scope', () => {
    const message = buildCascadeScopeDeleteMessage('campaign', '第 1 期故事', {
      assetCount: 4,
      selectedCount: 1,
    })

    expect(message).toContain('campaign')
    expect(message).toContain('第 1 期故事')
    expect(message).toContain('该 campaign 下的任务、资产、缩略图、metadata 和原图文件')
    expect(message).not.toContain('workspace 下没有')
  })
})
