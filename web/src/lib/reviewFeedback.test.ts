import { describe, expect, it } from 'vitest'
import {
  AgentImageflowApiError,
  type AgentImageflowAssetResponse,
  type AgentImageflowBatchStorySummaryAsset,
  type AgentImageflowBatchStorySummaryResponse,
  type AgentImageflowBatchStorySummaryScene,
} from './agentImageflowApi'
import {
  applyPendingReviewsToAssetList,
  applyPendingReviewsToBatchSummary,
  getReviewFriendlyErrorMessage,
  reconcileReviewedAssetInBatchSummary,
} from './reviewFeedback'

function createLibraryAsset(assetId: string, status: string): AgentImageflowAssetResponse {
  return {
    asset_id: assetId,
    workspace_id: 'ws',
    project_id: 'prj',
    campaign_id: 'cmp',
    task_id: `task-${assetId}`,
    status,
    provider: 'mock',
    model: 'mock-image',
    prompt: `${assetId} prompt`,
    delivery: {
      local_path: `/tmp/${assetId}.png`,
      download_url: `/api/assets/${assetId}/original`,
      thumbnail_url: `/api/assets/${assetId}/thumbnail`,
      metadata_url: `/api/assets/${assetId}/metadata`,
    },
    created_at: '2026-06-22T10:00:00Z',
  }
}

function createSummaryAsset(assetId: string, status: string): AgentImageflowBatchStorySummaryAsset {
  return {
    asset_id: assetId,
    task_id: `task-${assetId}`,
    status,
    provider: 'mock',
    model: 'mock-image',
    prompt: `${assetId} prompt`,
    download_url: `/api/assets/${assetId}/original`,
    thumbnail_url: `/api/assets/${assetId}/thumbnail`,
    metadata_url: `/api/assets/${assetId}/metadata`,
    created_at: assetId === 'asset-b' ? '2026-06-22T10:01:00Z' : '2026-06-22T10:00:00Z',
  }
}

function createScene(assets: AgentImageflowBatchStorySummaryAsset[]): AgentImageflowBatchStorySummaryScene {
  const selectedCount = assets.filter((asset) => asset.status === 'selected').length
  const rejectedCount = assets.filter((asset) => asset.status === 'rejected').length
  return {
    story_id: 'story-1',
    scene_id: 'scene-1',
    status: 'completed',
    latest_task_id: 'task-scene-1',
    primary_selected_asset_id: assets.find((asset) => asset.status === 'selected')?.asset_id,
    counts: {
      task_count: 1,
      succeeded_count: 1,
      failed_count: 0,
      asset_count: assets.length,
      selected_asset_count: selectedCount,
      rejected_asset_count: rejectedCount,
      attempt_count: 1,
    },
    tasks: [
      {
        task_id: 'task-scene-1',
        status: 'completed',
        asset_count: assets.length,
        attempt_count: 1,
        retrying: false,
      },
    ],
    assets,
  }
}

function createSummary(scene: AgentImageflowBatchStorySummaryScene): AgentImageflowBatchStorySummaryResponse {
  const selectedSceneCount = scene.counts.selected_asset_count > 0 ? 1 : 0
  const generatedCount = scene.assets.filter((asset) => asset.status === 'generated').length
  const selectedCount = scene.assets.filter((asset) => asset.status === 'selected').length
  const rejectedCount = scene.assets.filter((asset) => asset.status === 'rejected').length
  return {
    generated_at: '2026-06-22T10:02:00Z',
    project_id: 'prj',
    campaign_id: 'cmp',
    session_id: 'session-1',
    batch_id: 'batch-1',
    source: 'mcp',
    counts: {
      story_count: 1,
      scene_count: 1,
      scene_with_selected_count: selectedSceneCount,
      scene_missing_selected_count: selectedSceneCount === 0 ? 1 : 0,
      task_count: 1,
      queued_count: 0,
      running_count: 0,
      succeeded_count: 1,
      partial_count: 0,
      failed_count: 0,
      retrying_count: 0,
      asset_count: scene.assets.length,
      generated_asset_count: generatedCount,
      selected_asset_count: selectedCount,
      rejected_asset_count: rejectedCount,
      attempt_count: 1,
      excluded_setup_task_count: 0,
    },
    stories: [
      {
        story_id: 'story-1',
        scene_count: 1,
        selected_scene_count: selectedSceneCount,
        scenes: ['scene-1'],
      },
    ],
    scenes: [scene],
  }
}

describe('applyPendingReviewsToAssetList', () => {
  it('shows optimistic status immediately and rolls back when pending state clears', () => {
    const assets = [createLibraryAsset('asset-a', 'generated'), createLibraryAsset('asset-b', 'selected')]

    const optimistic = applyPendingReviewsToAssetList(assets, {
      'asset-a': { nextStatus: 'selected' },
    })

    expect(optimistic.map((asset) => asset.status)).toEqual(['selected', 'selected'])
    expect(assets.map((asset) => asset.status)).toEqual(['generated', 'selected'])

    const rolledBack = applyPendingReviewsToAssetList(assets, {})
    expect(rolledBack.map((asset) => asset.status)).toEqual(['generated', 'selected'])
  })

  it('removes assets that no longer match the active status filter after optimistic review', () => {
    const assets = [createLibraryAsset('asset-a', 'selected'), createLibraryAsset('asset-b', 'selected')]

    const optimistic = applyPendingReviewsToAssetList(assets, {
      'asset-a': { nextStatus: 'rejected' },
    }, 'selected')

    expect(optimistic.map((asset) => asset.asset_id)).toEqual(['asset-b'])
  })
})

describe('applyPendingReviewsToBatchSummary', () => {
  it('updates scene header, story coverage and top counts during optimistic select', () => {
    const summary = createSummary(createScene([
      createSummaryAsset('asset-a', 'selected'),
      createSummaryAsset('asset-b', 'generated'),
    ]))

    const optimistic = applyPendingReviewsToBatchSummary(summary, {
      'asset-b': { nextStatus: 'selected' },
    })

    expect(optimistic.scenes[0].primary_selected_asset_id).toBe('asset-b')
    expect(optimistic.scenes[0].assets.map((asset) => ({ id: asset.asset_id, status: asset.status }))).toEqual([
      { id: 'asset-a', status: 'generated' },
      { id: 'asset-b', status: 'selected' },
    ])
    expect(optimistic.scenes[0].counts.selected_asset_count).toBe(1)
    expect(optimistic.stories[0].selected_scene_count).toBe(1)
    expect(optimistic.counts.scene_with_selected_count).toBe(1)
    expect(optimistic.counts.selected_asset_count).toBe(1)
    expect(optimistic.counts.generated_asset_count).toBe(1)
  })

  it('drops selected coverage immediately when rejecting the selected asset', () => {
    const summary = createSummary(createScene([
      createSummaryAsset('asset-a', 'selected'),
      createSummaryAsset('asset-b', 'generated'),
    ]))

    const optimistic = applyPendingReviewsToBatchSummary(summary, {
      'asset-a': { nextStatus: 'rejected' },
    })

    expect(optimistic.scenes[0].primary_selected_asset_id).toBeUndefined()
    expect(optimistic.scenes[0].counts.selected_asset_count).toBe(0)
    expect(optimistic.stories[0].selected_scene_count).toBe(0)
    expect(optimistic.counts.scene_with_selected_count).toBe(0)
    expect(optimistic.counts.scene_missing_selected_count).toBe(1)
    expect(optimistic.counts.rejected_asset_count).toBe(1)
  })
})

describe('reconcileReviewedAssetInBatchSummary', () => {
  it('keeps confirmed local summary aligned after select succeeds', () => {
    const summary = createSummary(createScene([
      createSummaryAsset('asset-a', 'selected'),
      createSummaryAsset('asset-b', 'generated'),
    ]))

    const reconciled = reconcileReviewedAssetInBatchSummary(
      summary,
      'story-1:scene-1',
      'asset-b',
      createLibraryAsset('asset-b', 'selected'),
    )

    expect(reconciled.scenes[0].primary_selected_asset_id).toBe('asset-b')
    expect(reconciled.scenes[0].assets.map((asset) => ({ id: asset.asset_id, status: asset.status }))).toEqual([
      { id: 'asset-a', status: 'generated' },
      { id: 'asset-b', status: 'selected' },
    ])
    expect(reconciled.counts.selected_asset_count).toBe(1)
  })
})

describe('getReviewFriendlyErrorMessage', () => {
  it('includes retry-after guidance for 429 responses', () => {
    const error = new AgentImageflowApiError('HTTP 429', 429, undefined, 7)
    expect(getReviewFriendlyErrorMessage(error)).toBe('请求太快，请等待 7 秒后再试。当前内容已保留。')
  })
})
