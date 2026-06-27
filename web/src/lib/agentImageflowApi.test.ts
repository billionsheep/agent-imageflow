import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  buildAgentImageflowHeaders,
  buildAgentImageflowAssetReviewUrl,
  buildAgentImageflowAssetLifecycleUrl,
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
  executeAgentImageflowStorageCleanup,
  previewAgentImageflowStorageCleanup,
  regenerateAgentImageflowSceneTask,
  archiveAgentImageflowAsset,
  restoreAgentImageflowAsset,
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
    expect(buildAgentImageflowAssetLifecycleUrl('http://localhost:8081', 'asset_1', 'archive')).toBe('http://localhost:8081/api/assets/asset_1/archive')
    expect(buildAgentImageflowAssetLifecycleUrl('http://localhost:8081', 'asset_1', 'restore')).toBe('http://localhost:8081/api/assets/asset_1/restore')
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
    expect(buildAgentImageflowBatchManifestUrl('http://localhost:8081/', {
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    }, {
      sessionId: 'session_1',
      selectedOnly: true,
      includeRejected: false,
      view: 'final_delivery',
    } as unknown as Parameters<typeof buildAgentImageflowBatchManifestUrl>[2])).toBe('http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/batch-manifest?session_id=session_1&selected_only=true&include_rejected=false&view=final_delivery')
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
    expect(normalizeAgentImageflowAssetStatus('deprecated')).toBe('archived')
  })

  it('keeps story continuity summary fields on normalized batch summaries', () => {
    const response = normalizeAgentImageflowBatchStorySummaryResponse({
      generated_at: '2026-06-25T12:00:00Z',
      project_id: 'prj_demo',
      campaign_id: 'cmp_demo',
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
      stories: [{ story_id: 'pet_story', scene_count: 1, selected_scene_count: 1, scenes: ['scene_001'] }],
      scenes: [{
        story_id: 'pet_story',
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
        continuity: {
          story_revision: 'rev_001',
          story_plan_hash: 'sha256:story-plan',
          generation_mode: 'sequential_previous_panel',
          panel_index: 1,
          narrative_role: 'setup',
          dialogue: '才没有等你',
          emotion_before: '嘴硬但期待',
          emotion_after: '偷偷开心',
          pose_change: '从抱书防御变成稍微转头看门口',
          relationship_shift: '从独自等待变成准备迎接鸡毛',
          must_change: ['眼神从书上转向门口'],
          must_not_keep: ['不能把门口空间完全挡住'],
          state_transition_notes: '第一格先建立等待感，不要过早同框。',
          previous_panel_asset_id: '',
          provider_reference_participation: 'resolved_input_files',
          resolved_reference_assets: [
            { role: 'character_reference', asset_id: 'asset_milo_primary' },
          ],
        },
        tasks: [],
        assets: [{
          asset_id: 'asset_selected',
          task_id: 'task_001',
          status: 'approved',
          download_url: '/api/assets/asset_selected/original',
          thumbnail_url: '/api/assets/asset_selected/thumbnail',
          metadata_url: '/api/assets/asset_selected/metadata',
        }],
      }],
    })

    expect(response.scenes[0].assets[0].status).toBe('selected')
    expect(response.scenes[0].continuity?.panel_index).toBe(1)
    expect(response.scenes[0].continuity?.emotion_before).toBe('嘴硬但期待')
    expect(response.scenes[0].continuity?.must_change?.[0]).toBe('眼神从书上转向门口')
    expect(response.scenes[0].continuity?.resolved_reference_assets?.[0]?.asset_id).toBe('asset_milo_primary')
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
    const taskResponse = normalizeAgentImageflowTaskResponse({
      task_id: 'task_1',
      status: 'completed',
      asset_ids: ['asset_1'],
      requested_count: 2,
      delivered_count: 1,
      partial_success_reason: 'delivered_count_below_requested',
      provider_error_summary: 'provider returned only one candidate',
      assets: [{ asset_id: 'asset_1', status: 'approved', thumbnail_url: '/thumb', metadata_url: '/meta' }],
    })
    expect(taskResponse.assets?.[0]?.status).toBe('selected')
    expect(taskResponse.delivered_count).toBe(1)
    expect(taskResponse.partial_success_reason).toBe('delivered_count_below_requested')

    const assetResponse = normalizeAgentImageflowAssetResponse({
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
      delivery_role: 'final_delivery',
      asset_summary: {
        story_id: 'story_1',
        scene_id: 'scene_001',
        panel_index: 1,
        dialogue: '因为喜欢你',
        asset_status: 'approved',
        delivery_role: 'final_delivery',
      },
      delivery: {
        local_path: '/tmp/a.png',
        download_url: '/original',
        thumbnail_url: '/thumbnail',
        metadata_url: '/metadata',
      },
      created_at: '2026-06-19T00:00:00Z',
    })
    expect(assetResponse.status).toBe('generated')
    expect(assetResponse.asset_summary?.asset_status).toBe('selected')
    expect(assetResponse.asset_summary?.delivery_role).toBe('final_delivery')

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

    const summaryResponse = normalizeAgentImageflowBatchStorySummaryResponse({
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
        tasks: [{
          task_id: 'task_1',
          status: 'partially_completed',
          requested_count: 2,
          delivered_count: 1,
          partial_success_reason: 'delivered_count_below_requested',
          provider_error_summary: 'provider returned only one candidate',
          asset_count: 1,
          attempt_count: 1,
          retrying: false,
        }],
        assets: [{
          asset_id: 'asset_3',
          task_id: 'task_1',
          status: 'approved',
          download_url: '/original',
          thumbnail_url: '/thumbnail',
          metadata_url: '/metadata',
        }],
      }],
    })
    expect(summaryResponse.scenes[0].assets[0].status).toBe('selected')
    expect(summaryResponse.scenes[0].tasks[0].requested_count).toBe(2)
    expect(summaryResponse.scenes[0].tasks[0].delivered_count).toBe(1)
    expect(summaryResponse.scenes[0].tasks[0].provider_error_summary).toBe('provider returned only one candidate')
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
        delivery_role: 'final_delivery',
        created_at: '2026-06-22T00:00:00Z',
        caption_lineage: {
          derived_from_asset_id: 'asset_original',
          derivation_type: 'caption_edit',
          caption_text: '因为喜欢你',
          speaker_character_id: 'dog_xiaobai',
          bubble_anchor: 'top_right',
          tail_direction: 'toward_left',
          caption_intent: 'confession',
          auto_select_derivative: true,
          avoid_covering_subjects: true,
        },
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
    expect(response.assets[0].delivery_role).toBe('final_delivery')
    expect(response.assets[0].caption_lineage?.speaker_character_id).toBe('dog_xiaobai')
    expect(response.assets[0].caption_lineage?.bubble_anchor).toBe('top_right')
    expect(response.assets[0].caption_lineage?.auto_select_derivative).toBe(true)
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

  it('normalizes final-delivery manifest blocks and requests the final_delivery view', async () => {
    const manifest = {
      generated_at: '2026-06-27T00:00:00Z',
      project_id: 'prj_xhs_anime',
      campaign_id: 'cmp_7day_cover',
      session_id: 'session_1',
      batch_id: 'batch_1',
      manifest_view: 'final_delivery',
      selected_only: true,
      include_rejected: false,
      counts: {},
      tasks: [],
      assets: [],
      scenes: [],
      stories: [],
      final_delivery: {
        counts: {
          story_count: 1,
          scene_count: 1,
          scene_with_final_asset_count: 1,
          scene_missing_final_asset_count: 0,
          final_asset_count: 1,
        },
        stories: [{
          story_id: 'story_1',
          scene_count: 1,
          final_asset_count: 1,
          scenes: [{
            story_id: 'story_1',
            scene_id: 'scene_001',
            target_path: 'stories/story_1/scene_001-caption.png',
            latest_task_id: 'task_1',
            primary_selected_asset_id: 'asset_final',
            final_assets: [{
              asset_id: 'asset_final',
              task_id: 'task_1',
              story_id: 'story_1',
              scene_id: 'scene_001',
              status: 'approved',
              delivery_role: 'final_delivery',
              derived_from_asset_id: 'asset_base',
              derivation_type: 'caption_edit',
              download_url: '/api/assets/asset_final/original',
              thumbnail_url: '/api/assets/asset_final/thumbnail',
              metadata_url: '/api/assets/asset_final/metadata',
              target_path: 'stories/story_1/scene_001-caption.png',
              created_at: '2026-06-27T00:00:00Z',
            }],
          }],
          final_assets: [{
            asset_id: 'asset_final',
            task_id: 'task_1',
            story_id: 'story_1',
            scene_id: 'scene_001',
            status: 'approved',
            delivery_role: 'final_delivery',
            derived_from_asset_id: 'asset_base',
            derivation_type: 'caption_edit',
            download_url: '/api/assets/asset_final/original',
            thumbnail_url: '/api/assets/asset_final/thumbnail',
            metadata_url: '/api/assets/asset_final/metadata',
            target_path: 'stories/story_1/scene_001-caption.png',
            created_at: '2026-06-27T00:00:00Z',
          }],
        }],
        scenes: [{
          story_id: 'story_1',
          scene_id: 'scene_001',
          target_path: 'stories/story_1/scene_001-caption.png',
          latest_task_id: 'task_1',
          primary_selected_asset_id: 'asset_final',
          final_assets: [{
            asset_id: 'asset_final',
            task_id: 'task_1',
            story_id: 'story_1',
            scene_id: 'scene_001',
            status: 'approved',
            delivery_role: 'final_delivery',
            derived_from_asset_id: 'asset_base',
            derivation_type: 'caption_edit',
            download_url: '/api/assets/asset_final/original',
            thumbnail_url: '/api/assets/asset_final/thumbnail',
            metadata_url: '/api/assets/asset_final/metadata',
            target_path: 'stories/story_1/scene_001-caption.png',
            created_at: '2026-06-27T00:00:00Z',
          }],
        }],
        final_assets: [{
          asset_id: 'asset_final',
          task_id: 'task_1',
          story_id: 'story_1',
          scene_id: 'scene_001',
          status: 'approved',
          delivery_role: 'final_delivery',
          derived_from_asset_id: 'asset_base',
          derivation_type: 'caption_edit',
          download_url: '/api/assets/asset_final/original',
          thumbnail_url: '/api/assets/asset_final/thumbnail',
          metadata_url: '/api/assets/asset_final/metadata',
          target_path: 'stories/story_1/scene_001-caption.png',
          created_at: '2026-06-27T00:00:00Z',
        }],
      },
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
      view: 'final_delivery',
    } as unknown as Parameters<typeof getAgentImageflowBatchManifest>[3])

    expect(response.manifest_view).toBe('final_delivery')
    expect(response.final_delivery?.final_assets[0].status).toBe('selected')
    expect(response.final_delivery?.scenes[0].final_assets[0].status).toBe('selected')
    expect(response.final_delivery?.stories[0].final_assets[0].status).toBe('selected')
    const [url] = fetchMock.mock.calls[0]
    expect(url).toBe('http://localhost:8081/api/projects/prj_xhs_anime/campaigns/cmp_7day_cover/batch-manifest?session_id=session_1&selected_only=true&include_rejected=false&view=final_delivery')
  })

  it('posts cleanup preview and execute payloads to the admin-only cleanup endpoints', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify({
        generated_at: '2026-06-23T00:00:00Z',
        dry_run: true,
        dry_run_token: 'cleanup_1234567890abcdef',
        scope: {},
        summary: {
          candidate_count: 2,
          file_count: 6,
          bytes: 4096,
          by_reason: {
            rejected_asset: 1,
            orphan_file: 1,
          },
        },
        candidates: [],
        protected: {
          selected_asset_count: 1,
          published_asset_count: 1,
        },
      }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({
        generated_at: '2026-06-23T00:00:01Z',
        dry_run: false,
        executed: true,
        dry_run_token: 'cleanup_1234567890abcdef',
        scope: {},
        summary: {
          candidate_count: 2,
          deleted_candidate_count: 2,
          skipped_candidate_count: 0,
          failed_candidate_count: 0,
          file_count: 6,
          deleted_file_count: 6,
          bytes: 4096,
          deleted_bytes: 4096,
          by_reason: {
            rejected_asset: 1,
            orphan_file: 1,
          },
        },
        results: [],
        protected: {
          selected_asset_count: 1,
          published_asset_count: 1,
        },
      }), { status: 200 }))

    const scope = {
      workspaceId: 'ws_default',
      projectId: 'prj_xhs_anime',
      campaignId: 'cmp_7day_cover',
    }

    await previewAgentImageflowStorageCleanup('http://localhost:8081/', scope, {
      include_rejected: true,
      include_generated: true,
      include_deprecated: true,
      include_failed_task_tmp: true,
      include_orphans: true,
    })

    await executeAgentImageflowStorageCleanup('http://localhost:8081/', scope, {
      include_rejected: true,
      include_generated: true,
      include_deprecated: true,
      include_failed_task_tmp: true,
      include_orphans: true,
      dry_run_token: 'cleanup_1234567890abcdef',
      execute: true,
    })

    expect(fetchMock).toHaveBeenCalledTimes(2)
    const [previewUrl, previewInit] = fetchMock.mock.calls[0]
    expect(previewUrl).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-cleanup-preview')
    expect(previewInit).toMatchObject({
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
    })
    expect(JSON.parse(String(previewInit?.body))).toEqual({
      include_rejected: true,
      include_generated: true,
      include_deprecated: true,
      include_failed_task_tmp: true,
      include_orphans: true,
    })

    const [executeUrl, executeInit] = fetchMock.mock.calls[1]
    expect(executeUrl).toBe('http://localhost:8081/api/workspaces/ws_default/projects/prj_xhs_anime/campaigns/cmp_7day_cover/storage-cleanup-execute')
    expect(executeInit).toMatchObject({
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
    })
    expect(JSON.parse(String(executeInit?.body))).toEqual({
      include_rejected: true,
      include_generated: true,
      include_deprecated: true,
      include_failed_task_tmp: true,
      include_orphans: true,
      dry_run_token: 'cleanup_1234567890abcdef',
      execute: true,
    })
  })

  it('posts archive and restore actions to admin session asset lifecycle endpoints', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(new Response(JSON.stringify({
        asset_id: 'asset_1',
        status: 'deprecated',
        delivery: {
          download_url: '/original',
          thumbnail_url: '/thumbnail',
          metadata_url: '/metadata',
        },
      }), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({
        asset_id: 'asset_1',
        status: 'draft',
        delivery: {
          download_url: '/original',
          thumbnail_url: '/thumbnail',
          metadata_url: '/metadata',
        },
      }), { status: 200 }))

    await expect(archiveAgentImageflowAsset('http://localhost:8081/', 'asset_1')).resolves.toMatchObject({ status: 'archived' })
    await expect(restoreAgentImageflowAsset('http://localhost:8081/', 'asset_1')).resolves.toMatchObject({ status: 'generated' })

    expect(fetchMock.mock.calls.map(([url, init]) => [url, init?.method])).toEqual([
      ['http://localhost:8081/api/assets/asset_1/archive', 'POST'],
      ['http://localhost:8081/api/assets/asset_1/restore', 'POST'],
    ])
  })
})
