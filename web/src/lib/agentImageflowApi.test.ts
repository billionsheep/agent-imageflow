import { describe, expect, it } from 'vitest'
import {
  buildAgentImageflowAssetUrl,
  buildAgentImageflowTaskStatusUrl,
  buildAgentImageflowTaskUrl,
  normalizeAgentImageflowApiBaseUrl,
} from './agentImageflowApi'

describe('agentImageflowApi', () => {
  it('normalizes the service base URL', () => {
    expect(normalizeAgentImageflowApiBaseUrl('http://localhost:8081///')).toBe('http://localhost:8081')
    expect(normalizeAgentImageflowApiBaseUrl('')).toBe('http://localhost:8081')
  })

  it('builds the server-side ImageTask URL from scope ids', () => {
    expect(buildAgentImageflowTaskUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/tasks')
  })

  it('builds task and asset lookup URLs', () => {
    expect(buildAgentImageflowTaskStatusUrl('http://localhost:8081', 'task_1')).toBe('http://localhost:8081/api/tasks/task_1')
    expect(buildAgentImageflowAssetUrl('http://localhost:8081', 'asset_1')).toBe('http://localhost:8081/api/assets/asset_1')
  })
})

