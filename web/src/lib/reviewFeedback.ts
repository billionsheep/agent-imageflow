import {
  AgentImageflowApiError,
  type AgentImageflowAssetResponse,
  type AgentImageflowBatchStorySummaryAsset,
  type AgentImageflowBatchStorySummaryResponse,
} from './agentImageflowApi'

export type ReviewableAssetStatus = 'selected' | 'rejected'

export interface PendingAssetReview {
  nextStatus: ReviewableAssetStatus
}

export type PendingAssetReviewMap = Record<string, PendingAssetReview>

function getTimestamp(value?: string): number {
  if (!value) return 0
  const timestamp = new Date(value).getTime()
  return Number.isFinite(timestamp) ? timestamp : 0
}

function pickPrimarySelectedAssetId(assets: AgentImageflowBatchStorySummaryAsset[]): string | undefined {
  const selectedAssets = assets.filter((asset) => asset.status === 'selected')
  if (selectedAssets.length === 0) return undefined
  return [...selectedAssets].sort((left, right) => {
    const timeDelta = getTimestamp(right.created_at) - getTimestamp(left.created_at)
    if (timeDelta !== 0) return timeDelta
    return right.asset_id.localeCompare(left.asset_id)
  })[0]?.asset_id
}

function applySceneReviewStatus<T extends { asset_id: string; status: string }>(
  assets: T[],
  targetAssetId: string,
  nextStatus: ReviewableAssetStatus,
): T[] {
  return assets.map((asset) => {
    if (asset.asset_id === targetAssetId) {
      return { ...asset, status: nextStatus }
    }
    if (nextStatus === 'selected' && asset.status === 'selected') {
      return { ...asset, status: 'generated' }
    }
    return asset
  })
}

export function applyPendingReviewsToAssetList(
  assets: AgentImageflowAssetResponse[],
  pendingReviews: PendingAssetReviewMap,
  activeStatusFilter = '',
): AgentImageflowAssetResponse[] {
  const nextAssets = assets.map((asset) => {
    const pending = pendingReviews[asset.asset_id]
    return pending ? { ...asset, status: pending.nextStatus } : asset
  })
  const statusFilter = activeStatusFilter.trim()
  return statusFilter ? nextAssets.filter((asset) => asset.status === statusFilter) : nextAssets
}

export function applyPendingReviewsToBatchSummary(
  summary: AgentImageflowBatchStorySummaryResponse,
  pendingReviews: PendingAssetReviewMap,
): AgentImageflowBatchStorySummaryResponse {
  if (Object.keys(pendingReviews).length === 0) return summary

  const scenes = summary.scenes.map((scene) => {
    let nextAssets = scene.assets
    for (const asset of scene.assets) {
      const pending = pendingReviews[asset.asset_id]
      if (!pending) continue
      nextAssets = applySceneReviewStatus(nextAssets, asset.asset_id, pending.nextStatus)
    }
    const selectedAssetCount = nextAssets.filter((asset) => asset.status === 'selected').length
    const rejectedAssetCount = nextAssets.filter((asset) => asset.status === 'rejected').length
    return {
      ...scene,
      primary_selected_asset_id: pickPrimarySelectedAssetId(nextAssets),
      counts: {
        ...scene.counts,
        asset_count: nextAssets.length,
        selected_asset_count: selectedAssetCount,
        rejected_asset_count: rejectedAssetCount,
      },
      assets: nextAssets,
    }
  })

  const stories = summary.stories.map((story) => {
    const storyScenes = scenes.filter((scene) => scene.story_id === story.story_id)
    return {
      ...story,
      selected_scene_count: storyScenes.filter((scene) => scene.counts.selected_asset_count > 0).length,
    }
  })

  const allAssets = scenes.flatMap((scene) => scene.assets)
  const sceneWithSelectedCount = scenes.filter((scene) => scene.counts.selected_asset_count > 0).length

  return {
    ...summary,
    counts: {
      ...summary.counts,
      scene_with_selected_count: sceneWithSelectedCount,
      scene_missing_selected_count: Math.max(summary.counts.scene_count - sceneWithSelectedCount, 0),
      asset_count: allAssets.length,
      generated_asset_count: allAssets.filter((asset) => asset.status === 'generated').length,
      selected_asset_count: allAssets.filter((asset) => asset.status === 'selected').length,
      rejected_asset_count: allAssets.filter((asset) => asset.status === 'rejected').length,
    },
    stories,
    scenes,
  }
}

function toSummaryAsset(
  current: AgentImageflowBatchStorySummaryAsset,
  response: AgentImageflowAssetResponse,
): AgentImageflowBatchStorySummaryAsset {
  return {
    ...current,
    status: response.status,
    provider: response.provider ?? current.provider,
    model: response.model ?? current.model,
    prompt: response.prompt ?? current.prompt,
    download_url: response.delivery?.download_url ?? current.download_url,
    thumbnail_url: response.delivery?.thumbnail_url ?? current.thumbnail_url,
    metadata_url: response.delivery?.metadata_url ?? current.metadata_url,
    created_at: response.created_at ?? current.created_at,
  }
}

export function reconcileReviewedAssetInBatchSummary(
  summary: AgentImageflowBatchStorySummaryResponse,
  sceneKey: string,
  assetId: string,
  response: AgentImageflowAssetResponse,
): AgentImageflowBatchStorySummaryResponse {
  const pendingApplied = applyPendingReviewsToBatchSummary(summary, {
    [assetId]: {
      nextStatus: response.status === 'rejected' ? 'rejected' : 'selected',
    },
  })

  const scenes = pendingApplied.scenes.map((scene) => {
    if (`${scene.story_id}:${scene.scene_id}` !== sceneKey) return scene
    return {
      ...scene,
      assets: scene.assets.map((asset) => asset.asset_id === assetId ? toSummaryAsset(asset, response) : asset),
    }
  })

  return {
    ...pendingApplied,
    scenes,
  }
}

export function getReviewFriendlyErrorMessage(error: unknown): string {
  if (error instanceof AgentImageflowApiError && error.status === 429) {
    if (error.retryAfterSeconds && error.retryAfterSeconds > 0) {
      return `请求太快，请等待 ${error.retryAfterSeconds} 秒后再试。当前内容已保留。`
    }
    return '请求太快，服务端正在限流。请稍后再试。当前内容已保留。'
  }
  return error instanceof Error ? error.message : String(error)
}
