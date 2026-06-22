import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  buildAgentImageflowHeaders,
  buildAgentImageflowAssetReviewUrl,
  buildAgentImageflowAssetsUrl,
  buildAgentImageflowAssetUrl,
  buildAgentImageflowBatchProgressUrl,
  buildAgentImageflowBatchManifestUrl,
  buildAgentImageflowAdminLoginUrl,
  buildAgentImageflowAdminLogoutUrl,
  buildAgentImageflowAdminMeUrl,
  buildAgentImageflowRuntimeStatusUrl,
  buildAgentImageflowCampaignsUrl,
  buildAgentImageflowCampaignUrl,
  buildAgentImageflowBatchStorySummaryUrl,
  buildAgentImageflowInputFilesUrl,
  buildAgentImageflowInputFilePromoteUrl,
  buildAgentImageflowProjectUrl,
  buildAgentImageflowProjectsUrl,
  buildAgentImageflowProviderProfileUrl,
  buildAgentImageflowProjectVisualContextUrl,
  buildAgentImageflowQualityProfileUrl,
  buildAgentImageflowSceneRegenerationsUrl,
  buildAgentImageflowRecentAssetsUrl,
  buildAgentImageflowStorageGovernanceUrl,
  buildAgentImageflowStorageIntegrityUrl,
  buildAgentImageflowStorageCleanupExecuteUrl,
  buildAgentImageflowStorageCleanupPreviewUrl,
  buildAgentImageflowTaskAttemptsUrl,
  buildAgentImageflowTaskStatusUrl,
  buildAgentImageflowTaskUrl,
  buildAgentImageflowWorkspaceUrl,
  buildAgentImageflowWorkspacesUrl,
  normalizeAgentImageflowAssetResponse,
  normalizeAgentImageflowAssetListResponse,
  normalizeAgentImageflowAssetStatus,
  normalizeAgentImageflowBatchStorySummaryResponse,
  normalizeAgentImageflowTaskResponse,
  normalizeAgentImageflowApiBaseUrl,
  resolveAgentImageflowDeliveryUrl,
  getAgentImageflowBatchManifest,
  regenerateAgentImageflowSceneTask,
} from './agentImageflowApi'

describe('agentImageflowApi', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('normalizes the service base URL', () => {
    expect(normalizeAgentImageflowApiBaseUrl('http://localhost:8081///')).toBe('http://localhost:8081')
    expect(normalizeAgentImageflowApiBaseUrl('')).toBe('http://localhost:8081')
  })

  it('uses the browser origin as the empty service base URL when available', () => {
    const originalWindow = globalThis.window
    vi.stubGlobal('window', { location: { origin: 'https://imageflow.example.com' } })
    try {
      expect(normalizeAgentImageflowApiBaseUrl('')).toBe('https://imageflow.example.com')
    } finally {
      vi.stubGlobal('window', originalWindow)
    }
  })

  it('falls back to a host-matched local API for Vite dev and preview origins', () => {
    const originalWindow = globalThis.window
    vi.stubGlobal('window', { location: { origin: 'http://127.0.0.1:4173' } })
    try {
      expect(normalizeAgentImageflowApiBaseUrl('')).toBe('http://127.0.0.1:8081')
    } finally {
      vi.stubGlobal('window', originalWindow)
    }

    vi.stubGlobal('window', { location: { origin: 'http://localhost:5173' } })
    try {
      expect(normalizeAgentImageflowApiBaseUrl('')).toBe('http://localhost:8081')
    } finally {
      vi.stubGlobal('window', originalWindow)
    }
  })

  it('keeps saved local API settings on the same host as the current local page', () => {
    const originalWindow = globalThis.window
    vi.stubGlobal('window', { location: { origin: 'http://127.0.0.1:4173' } })
    try {
      expect(normalizeAgentImageflowApiBaseUrl('http://localhost:8081')).toBe('http://127.0.0.1:8081')
    } finally {
      vi.stubGlobal('window', originalWindow)
    }

    vi.stubGlobal('window', { location: { origin: 'http://localhost:4173' } })
    try {
      expect(normalizeAgentImageflowApiBaseUrl('http://127.0.0.1:8081')).toBe('http://localhost:8081')
    } finally {
      vi.stubGlobal('window', originalWindow)
    }
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
    expect(buildAgentImageflowAssetReviewUrl('http://localhost:8081', 'asset_1', 'select')).toBe('http://localhost:8081/api/assets/asset_1/approve')
    expect(buildAgentImageflowAssetReviewUrl('http://localhost:8081', 'asset_1', 'reject')).toBe('http://localhost:8081/api/assets/asset_1/reject')
    expect(buildAgentImageflowAdminLoginUrl('http://localhost:8081/')).toBe('http://localhost:8081/api/admin/login')
    expect(buildAgentImageflowAdminMeUrl('http://localhost:8081/')).toBe('http://localhost:8081/api/admin/me')
    expect(buildAgentImageflowAdminLogoutUrl('http://localhost:8081/')).toBe('http://localhost:8081/api/admin/logout')
    expect(buildAgentImageflowRuntimeStatusUrl('http://localhost:8081/')).toBe('http://localhost:8081/api/admin/runtime-status')
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
    expect(buildAgentImageflowInputFilePromoteUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    }, 'inp_1')).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/input-files/inp_1/promote-asset')
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
    expect(buildAgentImageflowBatchManifestUrl('http://localhost:8081/', {
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    }, {
      sessionId: 'session_1',
      batchId: 'batch_1',
      storyId: 'story_1',
      source: 'codex',
      status: 'completed',
      includeSetup: true,
      limit: 100,
      selectedOnly: false,
      includeRejected: true,
    })).toBe('http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/batch-manifest?session_id=session_1&batch_id=batch_1&story_id=story_1&source=codex&status=completed&include_setup=true&limit=100&selected_only=false&include_rejected=true')
    expect(buildAgentImageflowBatchStorySummaryUrl('http://localhost:8081/', {
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    }, {
      sessionId: 'session_1',
      batchId: 'batch_1',
      storyId: 'story_1',
      source: 'codex',
      status: 'completed',
      includeSetup: true,
      limit: 100,
    })).toBe('http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/batch-summary?session_id=session_1&batch_id=batch_1&story_id=story_1&source=codex&status=completed&include_setup=true&limit=100')
    expect(buildAgentImageflowSceneRegenerationsUrl('http://localhost:8081/', {
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    })).toBe('http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/scene-regenerations')
    expect(buildAgentImageflowQualityProfileUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/quality-profile')
    expect(buildAgentImageflowProviderProfileUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/provider-profile')
    expect(buildAgentImageflowProjectVisualContextUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/visual-context')
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
    expect(buildAgentImageflowStorageCleanupPreviewUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-cleanup-preview')
    expect(buildAgentImageflowStorageCleanupExecuteUrl('http://localhost:8081/', {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    })).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-cleanup-execute')
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

  it('rewrites asset delivery URLs to the active same-origin API base', () => {
    expect(resolveAgentImageflowDeliveryUrl('https://imageflow.example.com', '/api/assets/asset_1/thumbnail')).toBe('https://imageflow.example.com/api/assets/asset_1/thumbnail')
    expect(resolveAgentImageflowDeliveryUrl('https://imageflow.example.com', 'http://163.7.5.68:18081/api/assets/asset_1/thumbnail')).toBe('https://imageflow.example.com/api/assets/asset_1/thumbnail')
    expect(resolveAgentImageflowDeliveryUrl('https://imageflow.example.com', 'https://cdn.example.com/public/asset_1.webp')).toBe('https://cdn.example.com/public/asset_1.webp')
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

    expect(normalizeAgentImageflowBatchStorySummaryResponse({
      generated_at: '2026-06-22T00:00:00Z',
      project_id: 'prj_xhs_anime',
      campaign_id: 'cmp_7day_cover',
      counts: {
        story_count: 1,
        scene_count: 1,
        scene_with_selected_count: 1,
        scene_missing_selected_count: 0,
        task_count: 1,
        queued_count: 0,
        running_count: 0,
        succeeded_count: 1,
        partial_count: 0,
        failed_count: 0,
        retrying_count: 0,
        asset_count: 1,
        generated_asset_count: 0,
        selected_asset_count: 1,
        rejected_asset_count: 0,
        attempt_count: 1,
        excluded_setup_task_count: 0,
      },
      stories: [],
      scenes: [{
        story_id: 'story_1',
        scene_id: 'scene_001',
        status: 'completed',
        counts: {
          task_count: 1,
          succeeded_count: 1,
          failed_count: 0,
          asset_count: 1,
          selected_asset_count: 1,
          rejected_asset_count: 0,
          attempt_count: 1,
        },
        tasks: [],
        assets: [{
          asset_id: 'asset_3',
          task_id: 'task_1',
          status: 'approved',
          download_url: '/original',
          thumbnail_url: '/thumbnail',
          metadata_url: '/metadata',
        }],
      }],
    }).scenes[0].assets[0].status).toBe('selected')
  })

  it('posts a scene regeneration payload to the project campaign action URL', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({
      task_id: 'task_new',
      status: 'queued',
      regenerated_from_task_id: 'task_old',
      regenerate_no: 2,
      project_id: 'prj_xhs_anime',
      campaign_id: 'cmp_7day_cover',
      session_id: 'session_1',
      batch_id: 'batch_1',
      story_id: 'story_1',
      scene_id: 'scene_001',
      copied_visual_context_snapshot: {
        character_ids: ['dog_mochi'],
        reference_asset_ids: ['asset_ref'],
        prompt_recipe_id: 'pet_story_cover',
        character_count: 1,
        reference_count: 1,
        has_prompt_recipe: true,
      },
      warnings: [{
        code: 'selected_asset_preserved',
        message: 'Existing selected assets were not changed.',
      }],
    }), { status: 200 }))

    const response = await regenerateAgentImageflowSceneTask('http://localhost:8081/', {
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    }, {
      apiKey: 'project-secret',
    }, {
      source_task_id: 'task_old',
      regenerate_reason: 'scene failed',
      created_by: 'web',
    })

    expect(response.task_id).toBe('task_new')
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/scene-regenerations')
    expect(init).toMatchObject({
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': 'project-secret',
      },
    })
    expect(JSON.parse(String(init?.body))).toEqual({
      source_task_id: 'task_old',
      regenerate_reason: 'scene failed',
      created_by: 'web',
    })
  })

  it('gets a batch manifest with auth headers on the project campaign URL', async () => {
    const manifest = {
      generated_at: '2026-06-22T00:00:00Z',
      project_id: 'prj_xhs_anime',
      campaign_id: 'cmp_7day_cover',
      session_id: 'session_1',
      batch_id: 'batch_1',
      selected_only: true,
      include_rejected: false,
      counts: { asset_count: 1 },
      tasks: [],
      assets: [{
        asset_id: 'asset_1',
        task_id: 'task_1',
        story_id: 'story_1',
        scene_id: 'scene_001',
        status: 'approved',
        provider: 'mock',
        model: 'mock-image',
        prompt: 'pet cafe',
        download_url: '/api/assets/asset_1/original',
        thumbnail_url: '/api/assets/asset_1/thumbnail',
        metadata_url: '/api/assets/asset_1/metadata',
        target_path: 'stories/story_1/scene_001.png',
        created_at: '2026-06-22T00:00:00Z',
        visual_context: {
          character_ids: ['dog_mochi'],
        },
      }],
      scenes: [],
      stories: [],
    }
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify(manifest), { status: 200 }))

    const response = await getAgentImageflowBatchManifest('http://localhost:8081/', {
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    }, {
      apiKey: 'project-secret',
    }, {
      sessionId: 'session_1',
      selectedOnly: true,
      includeRejected: false,
    })

    expect(response.assets[0].status).toBe('selected')
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/batch-manifest?session_id=session_1&selected_only=true&include_rejected=false')
    expect(init).toMatchObject({
      method: 'GET',
      credentials: 'include',
      headers: {
        'X-API-Key': 'project-secret',
      },
    })
  })
})
