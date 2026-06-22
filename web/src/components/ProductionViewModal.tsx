import { type FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useStore } from '../store'
import {
  getAgentImageflowBatchManifest,
  getAgentImageflowBatchStorySummary,
  isAgentImageflowUnauthorizedError,
  normalizeAgentImageflowApiBaseUrl,
  rejectAgentImageflowAsset,
  regenerateAgentImageflowSceneTask,
  selectAgentImageflowAsset,
  type AgentImageflowAuth,
  type AgentImageflowAssetResponse,
  type AgentImageflowBatchStorySummaryResponse,
  type AgentImageflowBatchStorySummaryAsset,
  type AgentImageflowBatchStorySummaryScene,
  type AgentImageflowBatchStorySummaryTask,
  type AgentImageflowSceneRegenerationResponse,
} from '../lib/agentImageflowApi'
import { CloseIcon, LinkIcon, RefreshIcon } from './icons'

interface ProductionViewFilters {
  sessionId: string
  batchId: string
  storyId: string
  source: string
  status: string
  includeSetup: boolean
  limit: string
}

type AssetReviewAction = 'select' | 'reject'
type ManifestMode = 'selected' | 'all' | 'includeRejected'

const DEFAULT_FILTERS: ProductionViewFilters = {
  sessionId: '',
  batchId: '',
  storyId: '',
  source: '',
  status: '',
  includeSetup: false,
  limit: '100',
}

function mergeSeedFilters(seed: Partial<ProductionViewFilters> | null | undefined): ProductionViewFilters {
  if (!seed) return DEFAULT_FILTERS
  return {
    ...DEFAULT_FILTERS,
    sessionId: seed.sessionId ?? DEFAULT_FILTERS.sessionId,
    batchId: seed.batchId ?? DEFAULT_FILTERS.batchId,
    storyId: seed.storyId ?? DEFAULT_FILTERS.storyId,
    source: seed.source ?? DEFAULT_FILTERS.source,
    status: seed.status ?? DEFAULT_FILTERS.status,
    includeSetup: seed.includeSetup ?? DEFAULT_FILTERS.includeSetup,
    limit: seed.limit ?? DEFAULT_FILTERS.limit,
  }
}

function buildAuth(apiKey: string, basicUsername: string, basicPassword: string): AgentImageflowAuth {
  return {
    apiKey,
    basicUsername,
    basicPassword,
  }
}

function formatDate(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function formatCount(value: unknown): string {
  if (typeof value === 'string') return value
  return typeof value === 'number' && Number.isFinite(value) ? String(value) : '0'
}

function statusClassName(status: string): string {
  if (status === 'completed') return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-200'
  if (status === 'selected') return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-200'
  if (status === 'running' || status === 'retrying') return 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-200'
  if (status === 'queued') return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-200'
  if (status === 'failed' || status === 'rejected') return 'border-red-200 bg-red-50 text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200'
  if (status === 'partial') return 'border-orange-200 bg-orange-50 text-orange-700 dark:border-orange-500/20 dark:bg-orange-500/10 dark:text-orange-200'
  return 'border-gray-200 bg-gray-50 text-gray-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300'
}

function resolveApiUrl(baseUrl: string, value?: string): string {
  if (!value) return ''
  if (/^https?:\/\//i.test(value)) return value
  if (value.startsWith('/')) return `${baseUrl}${value}`
  return value
}

function getTaskErrorSummary(tasks: AgentImageflowBatchStorySummaryTask[]): string {
  const errorTask = tasks.find((task) => task.error_message || task.error_code || task.error_stage)
  if (!errorTask) return ''
  return [errorTask.error_stage, errorTask.error_code, errorTask.error_message].filter(Boolean).join(' · ')
}

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

function getManifestModeLabel(mode: ManifestMode): string {
  if (mode === 'selected') return 'Selected manifest'
  if (mode === 'includeRejected') return 'All + rejected'
  return 'All manifest'
}

function toSafeManifestFilePart(value: string): string {
  return value.trim().replace(/[^a-zA-Z0-9._-]+/g, '-').replace(/^-+|-+$/g, '') || 'batch'
}

function downloadManifestJson(payload: unknown, fileName: string) {
  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = fileName
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
  window.setTimeout(() => URL.revokeObjectURL(url), 0)
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

function updateSummaryWithReviewedAsset(
  current: AgentImageflowBatchStorySummaryResponse,
  sceneKey: string,
  assetId: string,
  response: AgentImageflowAssetResponse,
): AgentImageflowBatchStorySummaryResponse {
  const scenes = current.scenes.map((scene) => {
    if (`${scene.story_id}:${scene.scene_id}` !== sceneKey) return scene
    const assets = scene.assets.map((asset) => asset.asset_id === assetId ? toSummaryAsset(asset, response) : asset)
    const selectedAssetCount = assets.filter((asset) => asset.status === 'selected').length
    const rejectedAssetCount = assets.filter((asset) => asset.status === 'rejected').length
    return {
      ...scene,
      primary_selected_asset_id: pickPrimarySelectedAssetId(assets),
      counts: {
        ...scene.counts,
        asset_count: assets.length,
        selected_asset_count: selectedAssetCount,
        rejected_asset_count: rejectedAssetCount,
      },
      assets,
    }
  })
  const sceneWithSelectedCount = scenes.filter((scene) => scene.counts.selected_asset_count > 0).length
  const allAssets = scenes.flatMap((scene) => scene.assets)
  const stories = current.stories.map((story) => {
    const storyScenes = scenes.filter((scene) => scene.story_id === story.story_id)
    return {
      ...story,
      selected_scene_count: storyScenes.filter((scene) => scene.counts.selected_asset_count > 0).length,
    }
  })
  return {
    ...current,
    counts: {
      ...current.counts,
      scene_with_selected_count: sceneWithSelectedCount,
      scene_missing_selected_count: Math.max(current.counts.scene_count - sceneWithSelectedCount, 0),
      asset_count: allAssets.length,
      generated_asset_count: allAssets.filter((asset) => asset.status === 'generated').length,
      selected_asset_count: allAssets.filter((asset) => asset.status === 'selected').length,
      rejected_asset_count: allAssets.filter((asset) => asset.status === 'rejected').length,
    },
    stories,
    scenes,
  }
}

function SummaryStat({ label, value, tone = 'default' }: { label: string; value: unknown; tone?: 'default' | 'good' | 'bad' }) {
  const toneClass = tone === 'good'
    ? 'text-emerald-700 dark:text-emerald-200'
    : tone === 'bad'
      ? 'text-red-700 dark:text-red-200'
      : 'text-gray-800 dark:text-gray-100'
  return (
    <div className="min-w-0 rounded-lg border border-gray-200 bg-white px-3 py-2 dark:border-white/[0.08] dark:bg-white/[0.04]">
      <div className="truncate text-[10px] uppercase text-gray-400 dark:text-gray-500">{label}</div>
      <div className={`mt-1 truncate text-sm font-semibold ${toneClass}`}>{formatCount(value)}</div>
    </div>
  )
}

function Field({ label, value }: { label: string; value?: string | number }) {
  if (value == null || value === '') return null
  const text = String(value)
  return (
    <div className="min-w-0">
      <div className="text-[10px] uppercase text-gray-400 dark:text-gray-500">{label}</div>
      <div className="mt-0.5 truncate text-xs text-gray-600 dark:text-gray-300" title={text}>{text}</div>
    </div>
  )
}

function FilterInput({ label, value, placeholder, onChange }: { label: string; value: string; placeholder?: string; onChange: (value: string) => void }) {
  return (
    <label className="min-w-0 text-[11px] text-gray-500 dark:text-gray-400">
      <span className="mb-1 block uppercase">{label}</span>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        className="h-9 w-full min-w-0 rounded-lg border border-gray-200 bg-white px-2.5 text-xs text-gray-700 outline-none transition placeholder:text-gray-400 focus:border-blue-300 dark:border-white/[0.08] dark:bg-gray-950/50 dark:text-gray-100 dark:focus:border-blue-500/60"
      />
    </label>
  )
}

function SceneCard({
  actionErrors,
  baseUrl,
  pendingAssetIds,
  pendingSceneKeys,
  regenerationErrors,
  regenerationReasons,
  regenerationResults,
  scene,
  onRegenerateReasonChange,
  onRegenerateScene,
  onReviewAsset,
}: {
  actionErrors: Record<string, string>
  baseUrl: string
  pendingAssetIds: Record<string, boolean>
  pendingSceneKeys: Record<string, boolean>
  regenerationErrors: Record<string, string>
  regenerationReasons: Record<string, string>
  regenerationResults: Record<string, AgentImageflowSceneRegenerationResponse>
  scene: AgentImageflowBatchStorySummaryScene
  onRegenerateReasonChange: (sceneKey: string, value: string) => void
  onRegenerateScene: (scene: AgentImageflowBatchStorySummaryScene) => void
  onReviewAsset: (scene: AgentImageflowBatchStorySummaryScene, asset: AgentImageflowBatchStorySummaryAsset, action: AssetReviewAction) => void
}) {
  const sceneKey = `${scene.story_id}:${scene.scene_id}`
  const selectedCoverage = `${scene.counts.selected_asset_count}/${Math.max(scene.counts.asset_count, 0)}`
  const errorSummary = getTaskErrorSummary(scene.tasks)
  const thumbnailAsset = scene.assets.find((asset) => asset.asset_id === scene.primary_selected_asset_id) ?? scene.assets[0]
  const hasSelectedAsset = scene.counts.selected_asset_count > 0
  const regeneratePending = Boolean(pendingSceneKeys[sceneKey])
  const regenerateDisabled = regeneratePending || !scene.latest_task_id
  const regenerateReason = regenerationReasons[sceneKey] ?? ''
  const regenerateResult = regenerationResults[sceneKey]

  return (
    <article className="overflow-hidden rounded-lg border border-gray-200/80 bg-white dark:border-white/[0.08] dark:bg-gray-950/40">
      <div className="grid gap-0 md:grid-cols-[220px_minmax(0,1fr)]">
        <div className="aspect-[4/3] bg-gray-100 dark:bg-white/[0.04] md:aspect-auto md:min-h-full">
          {thumbnailAsset?.thumbnail_url ? (
            <img
              src={resolveApiUrl(baseUrl, thumbnailAsset.thumbnail_url)}
              alt={scene.scene_id}
              className="h-full w-full object-cover"
              loading="lazy"
            />
          ) : (
            <div className="flex h-full min-h-40 items-center justify-center px-4 text-center text-xs text-gray-400 dark:text-gray-500">
              no thumbnail
            </div>
          )}
        </div>
        <div className="min-w-0 space-y-3 p-3">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
            <div className="min-w-0">
              <div className="flex min-w-0 flex-wrap items-center gap-2">
                <span className="truncate text-sm font-semibold text-gray-800 dark:text-gray-100" title={`${scene.story_id} / ${scene.scene_id}`}>
                  {scene.scene_id}
                </span>
                <span className={`shrink-0 rounded-full border px-2 py-0.5 text-[10px] font-medium ${statusClassName(scene.status)}`}>
                  {scene.status}
                </span>
                <span className={`shrink-0 rounded-full border px-2 py-0.5 text-[10px] font-medium ${hasSelectedAsset ? statusClassName('selected') : statusClassName('queued')}`}>
                  {hasSelectedAsset ? 'selected asset' : 'missing selected'}
                </span>
              </div>
              <div className="mt-1 truncate text-xs text-gray-500 dark:text-gray-400" title={scene.story_id}>
                {scene.story_id}
              </div>
            </div>
            <div className="grid min-w-[180px] grid-cols-3 gap-2 text-center">
              <SummaryStat label="tasks" value={scene.counts.task_count} />
              <SummaryStat label="assets" value={scene.counts.asset_count} />
              <SummaryStat label="selected" value={selectedCoverage} tone={scene.counts.selected_asset_count > 0 ? 'good' : 'bad'} />
            </div>
          </div>

          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
            <Field label="order" value={scene.scene_order} />
            <Field label="latest task" value={scene.latest_task_id} />
            <Field label="primary selected" value={scene.primary_selected_asset_id} />
            <Field label="target" value={scene.target_path} />
            <Field label="succeeded" value={scene.counts.succeeded_count} />
            <Field label="failed" value={scene.counts.failed_count} />
            <Field label="attempts" value={scene.counts.attempt_count} />
            <Field label="rejected" value={scene.counts.rejected_asset_count} />
          </div>

          {scene.visual_context && (
            <div className="grid gap-2 rounded-lg border border-gray-200 bg-gray-50/70 p-2 dark:border-white/[0.08] dark:bg-white/[0.03] sm:grid-cols-3">
              <Field label="characters" value={scene.visual_context.character_ids?.join(', ')} />
              <Field label="references" value={scene.visual_context.reference_asset_ids?.join(', ')} />
              <Field label="recipe" value={scene.visual_context.prompt_recipe_id} />
            </div>
          )}

          {errorSummary && (
            <div className="break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200">
              {errorSummary}
            </div>
          )}

          <div className="grid gap-2 rounded-lg border border-gray-200 bg-gray-50/70 p-2 dark:border-white/[0.08] dark:bg-white/[0.03] sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
            <label className="min-w-0 text-[11px] text-gray-500 dark:text-gray-400">
              <span className="mb-1 block uppercase">regenerate reason</span>
              <input
                value={regenerateReason}
                onChange={(event) => onRegenerateReasonChange(sceneKey, event.target.value)}
                placeholder={scene.latest_task_id ? 'optional' : 'no latest task'}
                disabled={!scene.latest_task_id || regeneratePending}
                className="h-8 w-full min-w-0 rounded-lg border border-gray-200 bg-white px-2.5 text-xs text-gray-700 outline-none transition placeholder:text-gray-400 disabled:cursor-not-allowed disabled:opacity-60 focus:border-blue-300 dark:border-white/[0.08] dark:bg-gray-950/50 dark:text-gray-100 dark:focus:border-blue-500/60"
              />
              {hasSelectedAsset && (
                <span className="mt-1 block truncate text-[11px] text-emerald-700 dark:text-emerald-200">
                  selected preserved; regenerate will not auto replace it
                </span>
              )}
            </label>
            <button
              type="button"
              onClick={() => onRegenerateScene(scene)}
              disabled={regenerateDisabled}
              title={scene.latest_task_id ? `Regenerate from ${scene.latest_task_id}` : 'No latest task to regenerate'}
              className="inline-flex h-8 min-w-[112px] items-center justify-center rounded-lg border border-blue-200 bg-blue-50 px-3 text-xs font-medium text-blue-700 transition hover:bg-blue-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-200 dark:hover:bg-blue-500/20"
            >
              {regeneratePending ? 'Creating' : 'Regenerate'}
            </button>
            {!scene.latest_task_id && (
              <div className="break-words text-[11px] text-gray-400 dark:text-gray-500 sm:col-span-2">
                No latest task is available for this scene.
              </div>
            )}
            {regenerateResult && (
              <div className="min-w-0 rounded-lg border border-emerald-200 bg-emerald-50 px-2 py-1.5 text-[11px] text-emerald-800 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-100 sm:col-span-2">
                <div className="truncate" title={`${regenerateResult.task_id} / ${regenerateResult.status}`}>
                  new task {regenerateResult.task_id} · {regenerateResult.status}
                </div>
                {regenerateResult.warnings && regenerateResult.warnings.length > 0 && (
                  <div className="mt-1 line-clamp-2 break-words text-emerald-700 dark:text-emerald-200">
                    {regenerateResult.warnings.map((warning) => warning.message || warning.code).join(' · ')}
                  </div>
                )}
              </div>
            )}
            {regenerationErrors[sceneKey] && (
              <div className="break-words rounded-lg border border-red-200 bg-red-50 px-2 py-1.5 text-[11px] text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200 sm:col-span-2">
                {regenerationErrors[sceneKey]}
              </div>
            )}
          </div>

          <div className="space-y-2">
            {scene.tasks.map((task) => (
              <div key={task.task_id} className="grid gap-2 rounded-lg border border-gray-200 bg-gray-50/70 p-2 dark:border-white/[0.08] dark:bg-white/[0.03] sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
                <div className="min-w-0">
                  <div className="truncate text-xs font-medium text-gray-700 dark:text-gray-200" title={task.task_id}>{task.task_id}</div>
                  <div className="mt-0.5 truncate text-[11px] text-gray-500 dark:text-gray-400">
                    {formatDate(task.created_at)} · assets {task.asset_count} · attempts {task.attempt_count}
                  </div>
                </div>
                <span className={`w-fit rounded-full border px-2 py-0.5 text-[10px] font-medium ${statusClassName(task.retrying ? 'retrying' : task.status)}`}>
                  {task.retrying ? 'retrying' : task.status}
                </span>
              </div>
            ))}
          </div>

          {scene.assets.length > 0 && (
            <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
              {scene.assets.map((asset) => (
                <div key={asset.asset_id} className="min-w-0 rounded-lg border border-gray-200 bg-gray-50/70 p-2 dark:border-white/[0.08] dark:bg-white/[0.03]">
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <div className="truncate text-xs font-medium text-gray-700 dark:text-gray-200" title={asset.asset_id}>{asset.asset_id}</div>
                      <div className="mt-0.5 truncate text-[11px] text-gray-500 dark:text-gray-400" title={asset.prompt}>{asset.prompt || asset.provider || '-'}</div>
                    </div>
                    <span className={`shrink-0 rounded-full border px-2 py-0.5 text-[10px] font-medium ${statusClassName(asset.status)}`}>
                      {asset.status}
                    </span>
                  </div>
                  <div className="mt-2 flex flex-wrap gap-2">
                    <button
                      type="button"
                      onClick={() => onReviewAsset(scene, asset, 'select')}
                      disabled={pendingAssetIds[asset.asset_id] || asset.status === 'selected'}
                      className="inline-flex h-7 min-w-[74px] items-center justify-center rounded-lg border border-emerald-200 bg-emerald-50 px-2 text-[11px] font-medium text-emerald-700 transition hover:bg-emerald-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-200 dark:hover:bg-emerald-500/20"
                    >
                      {pendingAssetIds[asset.asset_id] ? 'Saving' : 'Select'}
                    </button>
                    <button
                      type="button"
                      onClick={() => onReviewAsset(scene, asset, 'reject')}
                      disabled={pendingAssetIds[asset.asset_id] || asset.status === 'rejected'}
                      className="inline-flex h-7 min-w-[74px] items-center justify-center rounded-lg border border-red-200 bg-red-50 px-2 text-[11px] font-medium text-red-700 transition hover:bg-red-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200 dark:hover:bg-red-500/20"
                    >
                      {pendingAssetIds[asset.asset_id] ? 'Saving' : 'Reject'}
                    </button>
                    {asset.download_url && (
                      <a href={resolveApiUrl(baseUrl, asset.download_url)} target="_blank" rel="noopener noreferrer" className="inline-flex h-7 items-center gap-1 rounded-lg border border-gray-200 bg-white px-2 text-[11px] text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">
                        <LinkIcon className="h-3 w-3" />
                        Original
                      </a>
                    )}
                    {asset.metadata_url && (
                      <a href={resolveApiUrl(baseUrl, asset.metadata_url)} target="_blank" rel="noopener noreferrer" className="inline-flex h-7 items-center gap-1 rounded-lg border border-gray-200 bg-white px-2 text-[11px] text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">
                        <LinkIcon className="h-3 w-3" />
                        Metadata
                      </a>
                    )}
                  </div>
                  {actionErrors[asset.asset_id] && (
                    <div className="mt-2 break-words rounded-lg border border-red-200 bg-red-50 px-2 py-1.5 text-[11px] text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200">
                      {actionErrors[asset.asset_id]}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </article>
  )
}

export default function ProductionViewModal() {
  const imageflowApiBaseUrl = useStore((state) => state.settings.imageflowApiBaseUrl)
  const imageflowApiKey = useStore((state) => state.settings.imageflowApiKey)
  const imageflowBasicUsername = useStore((state) => state.settings.imageflowBasicUsername)
  const imageflowBasicPassword = useStore((state) => state.settings.imageflowBasicPassword)
  const imageflowWorkspaceId = useStore((state) => state.settings.imageflowWorkspaceId)
  const imageflowProjectId = useStore((state) => state.settings.imageflowProjectId)
  const imageflowCampaignId = useStore((state) => state.settings.imageflowCampaignId)
  const productionViewSeed = useStore((state) => state.productionViewSeed)
  const setShowProductionView = useStore((state) => state.setShowProductionView)
  const showToast = useStore((state) => state.showToast)
  const baseUrl = useMemo(() => normalizeAgentImageflowApiBaseUrl(imageflowApiBaseUrl), [imageflowApiBaseUrl])
  const auth = useMemo(
    () => buildAuth(imageflowApiKey, imageflowBasicUsername, imageflowBasicPassword),
    [imageflowApiKey, imageflowBasicPassword, imageflowBasicUsername],
  )
  const scope = useMemo(() => ({
    workspaceId: imageflowWorkspaceId.trim(),
    projectId: imageflowProjectId.trim(),
    campaignId: imageflowCampaignId.trim(),
  }), [imageflowCampaignId, imageflowProjectId, imageflowWorkspaceId])
  const scopeReady = Boolean(scope.workspaceId && scope.projectId && scope.campaignId)
  const [filters, setFilters] = useState<ProductionViewFilters>(() => mergeSeedFilters(productionViewSeed))
  const [summary, setSummary] = useState<AgentImageflowBatchStorySummaryResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [unauthorized, setUnauthorized] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [pendingAssetIds, setPendingAssetIds] = useState<Record<string, boolean>>({})
  const [actionErrors, setActionErrors] = useState<Record<string, string>>({})
  const [regenerationReasons, setRegenerationReasons] = useState<Record<string, string>>({})
  const [pendingSceneKeys, setPendingSceneKeys] = useState<Record<string, boolean>>({})
  const [regenerationErrors, setRegenerationErrors] = useState<Record<string, string>>({})
  const [regenerationResults, setRegenerationResults] = useState<Record<string, AgentImageflowSceneRegenerationResponse>>({})
  const [pendingManifestModes, setPendingManifestModes] = useState<Partial<Record<ManifestMode, boolean>>>({})
  const [manifestError, setManifestError] = useState<string | null>(null)
  const requestRef = useRef(0)

  useEffect(() => {
    if (!productionViewSeed) return
    setFilters(mergeSeedFilters(productionViewSeed))
    setSummary(null)
    setError(null)
    setUnauthorized(false)
    setManifestError(null)
  }, [productionViewSeed])

  const queryReady = Boolean(filters.sessionId.trim() || filters.batchId.trim())
  const filtersActive = Object.values(filters).some((value) => typeof value === 'boolean' ? value : value.trim())

  const updateFilter = <K extends keyof ProductionViewFilters>(key: K, value: ProductionViewFilters[K]) => {
    setFilters((current) => ({ ...current, [key]: value }))
  }

  const loadSummary = useCallback(async () => {
    setError(null)
    setUnauthorized(false)
    if (!scopeReady) {
      setError('请先在设置或 Scope 管理中选择完整 workspace / project / campaign。')
      return
    }
    if (!queryReady) {
      setError('请输入 session_id 或 batch_id 后再查询。')
      return
    }
    const requestId = ++requestRef.current
    setLoading(true)
    try {
      const limit = Number.parseInt(filters.limit, 10)
      const response = await getAgentImageflowBatchStorySummary(baseUrl, {
        projectId: scope.projectId,
        campaignId: scope.campaignId,
      }, auth, {
        sessionId: filters.sessionId,
        batchId: filters.batchId,
        storyId: filters.storyId,
        source: filters.source,
        status: filters.status,
        includeSetup: filters.includeSetup,
        limit: Number.isFinite(limit) && limit > 0 ? limit : undefined,
      })
      if (requestRef.current !== requestId) return
      setSummary(response)
    } catch (nextError) {
      if (requestRef.current !== requestId) return
      if (isAgentImageflowUnauthorizedError(nextError)) {
        setUnauthorized(true)
        setError(null)
      } else {
        setError(nextError instanceof Error ? nextError.message : String(nextError))
      }
    } finally {
      if (requestRef.current === requestId) setLoading(false)
    }
  }, [auth, baseUrl, filters, queryReady, scope.campaignId, scope.projectId, scopeReady])

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    void loadSummary()
  }

  const clearFilters = () => {
    setFilters(DEFAULT_FILTERS)
    setSummary(null)
    setError(null)
    setUnauthorized(false)
    setPendingAssetIds({})
    setActionErrors({})
    setRegenerationReasons({})
    setPendingSceneKeys({})
    setRegenerationErrors({})
    setRegenerationResults({})
    setPendingManifestModes({})
    setManifestError(null)
  }

  const handleExportManifest = useCallback(async (mode: ManifestMode) => {
    setManifestError(null)
    setUnauthorized(false)
    if (!scopeReady) {
      setManifestError('请先在设置或 Scope 管理中选择完整 workspace / project / campaign。')
      return
    }
    if (!queryReady) {
      setManifestError('请输入 session_id 或 batch_id 后再导出 manifest。')
      return
    }
    const limit = Number.parseInt(filters.limit, 10)
    const selectedOnly = mode === 'selected'
    const includeRejected = mode === 'includeRejected'
    setPendingManifestModes((current) => ({ ...current, [mode]: true }))
    try {
      const manifest = await getAgentImageflowBatchManifest(baseUrl, {
        projectId: scope.projectId,
        campaignId: scope.campaignId,
      }, auth, {
        sessionId: filters.sessionId,
        batchId: filters.batchId,
        storyId: filters.storyId,
        source: filters.source,
        status: filters.status,
        includeSetup: filters.includeSetup,
        limit: Number.isFinite(limit) && limit > 0 ? limit : undefined,
        selectedOnly,
        includeRejected,
      })
      const scopePart = toSafeManifestFilePart(filters.batchId || filters.sessionId)
      const modePart = mode === 'includeRejected' ? 'include-rejected' : mode
      downloadManifestJson(manifest, `agent-imageflow-manifest-${scopePart}-${modePart}.json`)
      showToast(`${getManifestModeLabel(mode)} 已导出`, 'success')
    } catch (nextError) {
      const message = isAgentImageflowUnauthorizedError(nextError)
        ? 'unauthorized / login required'
        : nextError instanceof Error ? nextError.message : String(nextError)
      if (isAgentImageflowUnauthorizedError(nextError)) setUnauthorized(true)
      setManifestError(message)
      showToast(message, 'error')
    } finally {
      setPendingManifestModes((current) => {
        const next = { ...current }
        delete next[mode]
        return next
      })
    }
  }, [auth, baseUrl, filters, queryReady, scope.campaignId, scope.projectId, scopeReady, showToast])

  const handleReviewAsset = useCallback(async (
    scene: AgentImageflowBatchStorySummaryScene,
    asset: AgentImageflowBatchStorySummaryAsset,
    action: AssetReviewAction,
  ) => {
    const sceneKey = `${scene.story_id}:${scene.scene_id}`
    setPendingAssetIds((current) => ({ ...current, [asset.asset_id]: true }))
    setActionErrors((current) => {
      const next = { ...current }
      delete next[asset.asset_id]
      return next
    })
    setUnauthorized(false)
    try {
      const response = action === 'select'
        ? await selectAgentImageflowAsset(baseUrl, asset.asset_id, auth)
        : await rejectAgentImageflowAsset(baseUrl, asset.asset_id, auth)
      setSummary((current) => current ? updateSummaryWithReviewedAsset(current, sceneKey, asset.asset_id, response) : current)
      showToast(action === 'select' ? '已标记为 selected' : '已标记为 rejected', 'success')
    } catch (nextError) {
      const message = isAgentImageflowUnauthorizedError(nextError)
        ? 'unauthorized / login required'
        : nextError instanceof Error ? nextError.message : String(nextError)
      if (isAgentImageflowUnauthorizedError(nextError)) setUnauthorized(true)
      setActionErrors((current) => ({ ...current, [asset.asset_id]: message }))
      showToast(message, 'error')
    } finally {
      setPendingAssetIds((current) => {
        const next = { ...current }
        delete next[asset.asset_id]
        return next
      })
    }
  }, [auth, baseUrl, showToast])

  const handleRegenerateReasonChange = useCallback((sceneKey: string, value: string) => {
    setRegenerationReasons((current) => ({ ...current, [sceneKey]: value }))
  }, [])

  const handleRegenerateScene = useCallback(async (scene: AgentImageflowBatchStorySummaryScene) => {
    const sceneKey = `${scene.story_id}:${scene.scene_id}`
    const sourceTaskId = scene.latest_task_id?.trim()
    if (!sourceTaskId) return
    setPendingSceneKeys((current) => ({ ...current, [sceneKey]: true }))
    setRegenerationErrors((current) => {
      const next = { ...current }
      delete next[sceneKey]
      return next
    })
    setUnauthorized(false)
    try {
      const reason = regenerationReasons[sceneKey]?.trim()
      const response = await regenerateAgentImageflowSceneTask(baseUrl, {
        projectId: scope.projectId,
        campaignId: scope.campaignId,
      }, auth, {
        source_task_id: sourceTaskId,
        ...(reason ? { regenerate_reason: reason } : {}),
        created_by: 'web',
      })
      setRegenerationResults((current) => ({ ...current, [sceneKey]: response }))
      showToast(`已创建 regenerate task ${response.task_id}`, 'success')
      await loadSummary()
    } catch (nextError) {
      const message = isAgentImageflowUnauthorizedError(nextError)
        ? 'unauthorized / login required'
        : nextError instanceof Error ? nextError.message : String(nextError)
      if (isAgentImageflowUnauthorizedError(nextError)) setUnauthorized(true)
      setRegenerationErrors((current) => ({ ...current, [sceneKey]: message }))
      showToast(message, 'error')
    } finally {
      setPendingSceneKeys((current) => {
        const next = { ...current }
        delete next[sceneKey]
        return next
      })
    }
  }, [auth, baseUrl, loadSummary, regenerationReasons, scope.campaignId, scope.projectId, showToast])

  const selectedCoverage = summary
    ? `${summary.counts.scene_with_selected_count}/${Math.max(summary.counts.scene_count, 0)}`
    : '0/0'

  return (
    <div className="fixed inset-0 z-[110] bg-black/35 p-3 backdrop-blur-sm sm:p-4" role="dialog" aria-modal="true" aria-label="Production View">
      <div className="mx-auto flex h-full max-w-7xl flex-col overflow-hidden rounded-lg border border-gray-200/80 bg-white shadow-2xl ring-1 ring-black/5 dark:border-white/[0.08] dark:bg-gray-950 dark:ring-white/10">
        <div className="flex min-h-14 items-center justify-between gap-3 border-b border-gray-100 px-4 dark:border-white/[0.08]">
          <div className="min-w-0">
            <div className="truncate text-sm font-semibold text-gray-800 dark:text-gray-100">Production View</div>
            <div className="mt-0.5 truncate text-xs text-gray-500 dark:text-gray-400">
              {scopeReady ? `${scope.workspaceId} / ${scope.projectId} / ${scope.campaignId}` : 'scope incomplete'}
            </div>
          </div>
          <button
            type="button"
            onClick={() => setShowProductionView(false)}
            className="rounded-lg p-2 text-gray-500 transition hover:bg-gray-100 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-white/[0.06] dark:hover:text-gray-100"
            aria-label="关闭"
          >
            <CloseIcon className="h-5 w-5" />
          </button>
        </div>

        <div className="min-h-0 flex-1 overflow-auto p-4">
          <form onSubmit={handleSubmit} className="rounded-lg border border-gray-200/80 bg-gray-50/70 p-3 dark:border-white/[0.08] dark:bg-white/[0.03]">
            <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-7">
              <FilterInput label="session_id" value={filters.sessionId} placeholder="session_id" onChange={(value) => updateFilter('sessionId', value)} />
              <FilterInput label="batch_id" value={filters.batchId} placeholder="batch_id" onChange={(value) => updateFilter('batchId', value)} />
              <FilterInput label="story_id" value={filters.storyId} placeholder="optional" onChange={(value) => updateFilter('storyId', value)} />
              <FilterInput label="source" value={filters.source} placeholder="codex / mcp" onChange={(value) => updateFilter('source', value)} />
              <label className="min-w-0 text-[11px] text-gray-500 dark:text-gray-400">
                <span className="mb-1 block uppercase">status</span>
                <select
                  value={filters.status}
                  onChange={(event) => updateFilter('status', event.target.value)}
                  className="h-9 w-full min-w-0 rounded-lg border border-gray-200 bg-white px-2.5 text-xs text-gray-700 outline-none transition focus:border-blue-300 dark:border-white/[0.08] dark:bg-gray-950/50 dark:text-gray-100 dark:focus:border-blue-500/60"
                >
                  <option value="">All</option>
                  <option value="queued">Queued</option>
                  <option value="running">Running</option>
                  <option value="completed">Completed</option>
                  <option value="partially_completed">Partial</option>
                  <option value="failed">Failed</option>
                  <option value="enqueue_failed">Enqueue failed</option>
                </select>
              </label>
              <FilterInput label="limit" value={filters.limit} placeholder="100" onChange={(value) => updateFilter('limit', value)} />
              <label className="flex min-w-0 items-end gap-2 rounded-lg border border-gray-200 bg-white px-2.5 py-2 text-xs text-gray-600 dark:border-white/[0.08] dark:bg-gray-950/50 dark:text-gray-300">
                <input
                  type="checkbox"
                  checked={filters.includeSetup}
                  onChange={(event) => updateFilter('includeSetup', event.target.checked)}
                  className="mb-0.5"
                />
                <span className="truncate">include setup</span>
              </label>
            </div>
            <div className="mt-3 flex flex-wrap items-center justify-between gap-2">
              <div className="min-w-0 truncate text-xs text-gray-500 dark:text-gray-400">
                At least one of session_id or batch_id is required.
              </div>
              <div className="flex flex-wrap items-center justify-end gap-2">
                <div className="flex min-w-0 flex-wrap items-center justify-end gap-2">
                  {(['selected', 'all', 'includeRejected'] as ManifestMode[]).map((mode) => (
                    <button
                      key={mode}
                      type="button"
                      onClick={() => void handleExportManifest(mode)}
                      disabled={!scopeReady || !queryReady || Boolean(pendingManifestModes[mode])}
                      className="inline-flex h-9 min-w-[128px] items-center justify-center rounded-lg border border-gray-200 bg-white px-3 text-xs font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
                    >
                      {pendingManifestModes[mode] ? 'Exporting' : getManifestModeLabel(mode)}
                    </button>
                  ))}
                </div>
                {filtersActive && (
                  <button
                    type="button"
                    onClick={clearFilters}
                    className="h-9 rounded-lg border border-gray-200 bg-white px-3 text-xs font-medium text-gray-500 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
                  >
                    Clear
                  </button>
                )}
                <button
                  type="submit"
                  disabled={loading || !scopeReady}
                  className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-blue-200 bg-blue-50 px-3 text-xs font-medium text-blue-700 transition hover:bg-blue-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-200"
                >
                  <RefreshIcon className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                  Query
                </button>
              </div>
            </div>
          </form>

          {manifestError && (
            <div className="mt-3 break-words rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200">
              {manifestError}
            </div>
          )}
          {error && (
            <div className="mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200">
              {error}
            </div>
          )}
          {unauthorized && (
            <div className="mt-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-100">
              unauthorized / login required. 请先登录 Admin session，或在设置中配置可访问当前 project 的 Basic Auth / project API key。
            </div>
          )}
          {loading && summary && (
            <div className="mt-3 rounded-lg border border-blue-200 bg-blue-50 px-3 py-2 text-xs text-blue-700 dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-200">
              正在刷新 production summary，当前结果会保留到新结果返回。
            </div>
          )}

          {!summary && loading ? (
            <div className="mt-4 rounded-lg border border-dashed border-gray-200 px-4 py-10 text-center text-xs text-gray-400 dark:border-white/[0.08] dark:text-gray-500">
              正在加载 batch / story / scene summary...
            </div>
          ) : !summary && !loading && !unauthorized && !error ? (
            <div className="mt-4 rounded-lg border border-dashed border-gray-200 px-4 py-10 text-center text-xs text-gray-400 dark:border-white/[0.08] dark:text-gray-500">
              {queryReady ? '已带入 batch / session filters，点击 Query 查看 production summary。' : '输入 session_id 或 batch_id 后查询 production summary。'}
            </div>
          ) : summary ? (
            <div className="mt-4 space-y-4">
              <div className="grid gap-2 sm:grid-cols-3 lg:grid-cols-8">
                <SummaryStat label="stories" value={summary.counts.story_count} />
                <SummaryStat label="scenes" value={summary.counts.scene_count} />
                <SummaryStat label="selected coverage" value={selectedCoverage} tone={summary.counts.scene_missing_selected_count === 0 ? 'good' : 'bad'} />
                <SummaryStat label="tasks" value={summary.counts.task_count} />
                <SummaryStat label="assets" value={summary.counts.asset_count} />
                <SummaryStat label="running" value={summary.counts.running_count} />
                <SummaryStat label="failed" value={summary.counts.failed_count} tone={summary.counts.failed_count > 0 ? 'bad' : 'default'} />
                <SummaryStat label="excluded setup" value={summary.counts.excluded_setup_task_count} />
              </div>

              <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_280px]">
                <div className="space-y-3">
                  {summary.scenes.length === 0 ? (
                    <div className="rounded-lg border border-dashed border-gray-200 px-4 py-8 text-center text-xs text-gray-400 dark:border-white/[0.08] dark:text-gray-500">
                      当前查询没有 scene production rows。
                    </div>
                  ) : summary.scenes.map((scene) => (
                    <SceneCard
                      key={`${scene.story_id}:${scene.scene_id}`}
                      actionErrors={actionErrors}
                      baseUrl={baseUrl}
                      pendingAssetIds={pendingAssetIds}
                      pendingSceneKeys={pendingSceneKeys}
                      regenerationErrors={regenerationErrors}
                      regenerationReasons={regenerationReasons}
                      regenerationResults={regenerationResults}
                      scene={scene}
                      onRegenerateReasonChange={handleRegenerateReasonChange}
                      onRegenerateScene={handleRegenerateScene}
                      onReviewAsset={handleReviewAsset}
                    />
                  ))}
                </div>
                <aside className="min-w-0 space-y-3">
                  <div className="rounded-lg border border-gray-200/80 bg-white p-3 dark:border-white/[0.08] dark:bg-white/[0.04]">
                    <div className="text-xs font-semibold text-gray-800 dark:text-gray-100">Batch</div>
                    <div className="mt-3 grid gap-2">
                      <Field label="generated" value={formatDate(summary.generated_at)} />
                      <Field label="project" value={summary.project_id} />
                      <Field label="campaign" value={summary.campaign_id} />
                      <Field label="session" value={summary.session_id} />
                      <Field label="batch" value={summary.batch_id} />
                      <Field label="source" value={summary.source} />
                      <Field label="story filter" value={summary.story_id} />
                    </div>
                  </div>
                  <div className="rounded-lg border border-gray-200/80 bg-white p-3 dark:border-white/[0.08] dark:bg-white/[0.04]">
                    <div className="text-xs font-semibold text-gray-800 dark:text-gray-100">Stories</div>
                    <div className="mt-3 space-y-2">
                      {summary.stories.length === 0 ? (
                        <div className="text-xs text-gray-400 dark:text-gray-500">no stories</div>
                      ) : summary.stories.map((story) => (
                        <div key={story.story_id} className="min-w-0 rounded-lg border border-gray-200 bg-gray-50/70 p-2 dark:border-white/[0.08] dark:bg-white/[0.03]">
                          <div className="truncate text-xs font-medium text-gray-700 dark:text-gray-200" title={story.story_id}>{story.story_id}</div>
                          <div className="mt-1 text-[11px] text-gray-500 dark:text-gray-400">
                            {story.selected_scene_count}/{story.scene_count} selected
                          </div>
                          <div className="mt-1 line-clamp-3 break-words text-[11px] text-gray-400 dark:text-gray-500" title={story.scenes.join(', ')}>
                            {story.scenes.join(', ')}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </aside>
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}
