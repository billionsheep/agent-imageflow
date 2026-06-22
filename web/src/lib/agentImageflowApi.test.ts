import { describe, expect, it } from 'vitest'
import {
  buildAgentImageflowHeaders,
  buildAgentImageflowAssetsUrl,
  buildAgentImageflowAssetUrl,
  buildAgentImageflowBatchProgressUrl,
  buildAgentImageflowAdminLoginUrl,
  buildAgentImageflowAdminLogoutUrl,
  buildAgentImageflowAdminMeUrl,
  buildAgentImageflowCampaignsUrl,
  buildAgentImageflowCampaignUrl,
  buildAgentImageflowInputFilesUrl,
  buildAgentImageflowProjectUrl,
  buildAgentImageflowProjectsUrl,
  buildAgentImageflowProviderProfileUrl,
  buildAgentImageflowQualityProfileUrl,
  buildAgentImageflowRecentAssetsUrl,
  buildAgentImageflowStorageGovernanceUrl,
  buildAgentImageflowStorageIntegrityUrl,
  buildAgentImageflowTaskAttemptsUrl,
  buildAgentImageflowTaskStatusUrl,
  buildAgentImageflowTaskUrl,
  buildAgentImageflowWorkspaceUrl,
  buildAgentImageflowWorkspacesUrl,
  normalizeAgentImageflowAssetResponse,
  normalizeAgentImageflowAssetListResponse,
  normalizeAgentImageflowAssetStatus,
  normalizeAgentImageflowTaskResponse,
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
    expect(buildAgentImageflowTaskAttemptsUrl('http://localhost:8081', 'task_1')).toBe('http://localhost:8081/api/tasks/task_1/attempts')
    expect(buildAgentImageflowAssetUrl('http://localhost:8081', 'asset_1')).toBe('http://localhost:8081/api/assets/asset_1')
    expect(buildAgentImageflowAdminLoginUrl('http://localhost:8081/')).toBe('http://localhost:8081/api/admin/login')
    expect(buildAgentImageflowAdminMeUrl('http://localhost:8081/')).toBe('http://localhost:8081/api/admin/me')
    expect(buildAgentImageflowAdminLogoutUrl('http://localhost:8081/')).toBe('http://localhost:8081/api/admin/logout')
    expect(buildAgentImageflowWorkspacesUrl('http://localhost:8081/')).toBe('http://localhost:8081/api/workspaces')
    expect(buildAgentImageflowWorkspaceUrl('http://localhost:8081/', 'ws_default')).toBe('http://localhost:8081/api/workspaces/ws_default')
    expect(buildAgentImageflowProjectsUrl('http://localhost:8081/', 'ws_default')).toBe('http://localhost:8081/api/workspaces/ws_default/projects')
    expect(buildAgentImageflowProjectUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime')
    expect(buildAgentImageflowCampaignsUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns')
    expect(buildAgentImageflowCampaignUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover')
    expect(buildAgentImageflowInputFilesUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/input-files')
    expect(buildAgentImageflowAssetsUrl('http://localhost:8081/', {
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    })).toBe('http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/assets')
    expect(buildAgentImageflowAssetsUrl('http://localhost:8081/', {
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    }, {
      limit: 24,
      offset: 48,
      status: 'selected',
      provider: 'mock',
      source: 'mcp',
      sessionId: 'session_1',
      batchId: 'batch_1',
      keyword: 'cover',
    })).toBe('http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/assets?limit=24&offset=48&status=selected&provider=mock&source=mcp&session_id=session_1&batch_id=batch_1&keyword=cover')
    expect(buildAgentImageflowRecentAssetsUrl('http://localhost:8081/', {
      limit: 24,
      offset: 24,
      source: 'mcp',
      sessionId: 'session_1',
    })).toBe('http://localhost:8081/api/admin/assets/recent?limit=24&offset=24&source=mcp&session_id=session_1')
    expect(buildAgentImageflowBatchProgressUrl('http://localhost:8081/', {
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    }, {
      sessionId: 'session_1',
      batchId: 'batch_1',
      limit: 50,
    })).toBe('http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/batch-progress?session_id=session_1&batch_id=batch_1&limit=50')
    expect(buildAgentImageflowQualityProfileUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/quality-profile')
    expect(buildAgentImageflowProviderProfileUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/provider-profile')
    expect(buildAgentImageflowStorageGovernanceUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-governance')
    expect(buildAgentImageflowStorageIntegrityUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-integrity')
  })

  it('maps compatible asset statuses to product language', () => {
    expect(normalizeAgentImageflowAssetStatus('draft')).toBe('generated')
    expect(normalizeAgentImageflowAssetStatus('approved')).toBe('selected')
    expect(normalizeAgentImageflowAssetStatus('rejected')).toBe('rejected')
  })

  it('builds auth headers for managed mode requests', () => {
    expect(buildAgentImageflowHeaders({
      apiKey: 'project-secret',
      basicUsername: 'admin',
      basicPassword: 'secret',
    }, { 'Content-Type': 'application/json' })).toEqual({
      'Content-Type': 'application/json',
      'X-API-Key': 'project-secret',
      Authorization: 'Basic YWRtaW46c2VjcmV0',
    })
  })

  it('normalizes task and asset response statuses', () => {
    expect(normalizeAgentImageflowTaskResponse({
      task_id: 'task_1',
      status: 'completed',
      asset_ids: ['asset_1'],
      assets: [{ asset_id: 'asset_1', status: 'approved', thumbnail_url: '/thumb', metadata_url: '/meta' }],
    }).assets?.[0]?.status).toBe('selected')

    expect(normalizeAgentImageflowAssetResponse({
      asset_id: 'asset_1',
      workspace_id: 'ws_default',
      project_id: 'prj_xhs_anime',
      campaign_id: 'cmp_7day_cover',
      task_id: 'task_1',
      status: 'draft',
      provider: 'mock',
      model: 'mock-image',
      prompt: 'pet cafe',
      metadata_json: {
        source: 'mcp',
        session_id: 'session_1',
      },
      delivery: {
        local_path: '/tmp/a.png',
        download_url: '/original',
        thumbnail_url: '/thumbnail',
        metadata_url: '/metadata',
      },
      created_at: '2026-06-19T00:00:00Z',
    }).status).toBe('generated')

    expect(normalizeAgentImageflowAssetListResponse([{
      asset_id: 'asset_2',
      status: 'approved',
      delivery: {
        local_path: '/tmp/b.png',
        download_url: '/original-b',
        thumbnail_url: '/thumbnail-b',
        metadata_url: '/metadata-b',
      },
    }])[0].status).toBe('selected')
  })
})
