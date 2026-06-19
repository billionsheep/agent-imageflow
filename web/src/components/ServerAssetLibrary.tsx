import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useStore } from '../store'
import { normalizeSettings } from '../lib/apiProfiles'
import { copyTextToClipboard, getClipboardFailureMessage } from '../lib/clipboard'
import {
  listAgentImageflowAssets,
  normalizeAgentImageflowApiBaseUrl,
  rejectAgentImageflowAsset,
  selectAgentImageflowAsset,
  type AgentImageflowAssetResponse,
  type AgentImageflowAuth,
} from '../lib/agentImageflowApi'
import { CopyIcon, LinkIcon, RefreshIcon } from './icons'

function buildAuth(settings: ReturnType<typeof normalizeSettings>): AgentImageflowAuth {
  return {
    apiKey: settings.imageflowApiKey,
    basicUsername: settings.imageflowBasicUsername,
    basicPassword: settings.imageflowBasicPassword,
  }
}

function getMetadataValue(asset: AgentImageflowAssetResponse, key: string): string {
  const value = asset.metadata_json?.[key]
  return typeof value === 'string' ? value : ''
}

function formatAssetDate(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function statusClassName(status: string): string {
  if (status === 'selected') return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-200'
  if (status === 'rejected') return 'border-red-200 bg-red-50 text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200'
  return 'border-gray-200 bg-gray-50 text-gray-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300'
}

function AssetField({ label, value }: { label: string; value?: string }) {
  if (!value) return null
  return (
    <div className="min-w-0">
      <div className="text-[10px] uppercase text-gray-400 dark:text-gray-500">{label}</div>
      <div className="mt-0.5 truncate text-xs text-gray-600 dark:text-gray-300" title={value}>{value}</div>
    </div>
  )
}

export default function ServerAssetLibrary() {
  const settings = useStore((state) => state.settings)
  const showToast = useStore((state) => state.showToast)
  const normalizedSettings = useMemo(() => normalizeSettings(settings), [settings])
  const baseUrl = useMemo(
    () => normalizeAgentImageflowApiBaseUrl(normalizedSettings.imageflowApiBaseUrl),
    [normalizedSettings.imageflowApiBaseUrl],
  )
  const auth = useMemo(() => buildAuth(normalizedSettings), [normalizedSettings])
  const scope = useMemo(() => ({
    workspaceId: normalizedSettings.imageflowWorkspaceId.trim(),
    projectId: normalizedSettings.imageflowProjectId.trim(),
    campaignId: normalizedSettings.imageflowCampaignId.trim(),
  }), [
    normalizedSettings.imageflowCampaignId,
    normalizedSettings.imageflowProjectId,
    normalizedSettings.imageflowWorkspaceId,
  ])
  const scopeReady = Boolean(scope.workspaceId && scope.projectId && scope.campaignId)
  const [assets, setAssets] = useState<AgentImageflowAssetResponse[]>([])
  const [loading, setLoading] = useState(false)
  const [actionAssetId, setActionAssetId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const requestRef = useRef(0)

  const refreshAssets = useCallback(async () => {
    if (!scopeReady) {
      setAssets([])
      return
    }
    const requestId = ++requestRef.current
    setLoading(true)
    setError(null)
    try {
      const response = await listAgentImageflowAssets(baseUrl, {
        projectId: scope.projectId,
        campaignId: scope.campaignId,
      }, auth)
      if (requestRef.current !== requestId) return
      setAssets(response)
    } catch (nextError) {
      if (requestRef.current !== requestId) return
      setError(nextError instanceof Error ? nextError.message : String(nextError))
      setAssets([])
    } finally {
      if (requestRef.current === requestId) {
        setLoading(false)
      }
    }
  }, [auth, baseUrl, scope.campaignId, scope.projectId, scopeReady])

  useEffect(() => {
    void refreshAssets()
  }, [refreshAssets])

  const updateAsset = (nextAsset: AgentImageflowAssetResponse) => {
    setAssets((current) => current.map((asset) => asset.asset_id === nextAsset.asset_id ? nextAsset : asset))
  }

  const selectAsset = async (asset: AgentImageflowAssetResponse) => {
    setActionAssetId(asset.asset_id)
    try {
      updateAsset(await selectAgentImageflowAsset(baseUrl, asset.asset_id, auth))
      showToast('已标记为 selected', 'success')
    } catch (nextError) {
      showToast(nextError instanceof Error ? nextError.message : String(nextError), 'error')
    } finally {
      setActionAssetId(null)
    }
  }

  const rejectAsset = async (asset: AgentImageflowAssetResponse) => {
    setActionAssetId(asset.asset_id)
    try {
      updateAsset(await rejectAgentImageflowAsset(baseUrl, asset.asset_id, auth))
      showToast('已标记为 rejected', 'success')
    } catch (nextError) {
      showToast(nextError instanceof Error ? nextError.message : String(nextError), 'error')
    } finally {
      setActionAssetId(null)
    }
  }

  const copyText = async (text: string, label: string) => {
    try {
      await copyTextToClipboard(text)
      showToast(`已复制${label}`, 'success')
    } catch (nextError) {
      showToast(getClipboardFailureMessage('复制失败', nextError), 'error')
    }
  }

  if (!normalizedSettings.imageflowManagedMode && !scopeReady) return null

  const selectedCount = assets.filter((asset) => asset.status === 'selected').length
  const rejectedCount = assets.filter((asset) => asset.status === 'rejected').length

  return (
    <section data-no-drag-select className="mb-5 rounded-lg border border-gray-200/80 bg-white/90 p-4 shadow-sm dark:border-white/[0.08] dark:bg-white/[0.03]">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="min-w-0">
          <div className="text-sm font-semibold text-gray-800 dark:text-gray-100">服务端资产库</div>
          <div className="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
            {scopeReady ? `${scope.workspaceId} / ${scope.projectId} / ${scope.campaignId}` : '请在设置或 Scope 管理中选择完整 scope'}
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="rounded-full border border-gray-200 bg-gray-50 px-2.5 py-1 text-[11px] text-gray-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">
            {assets.length} assets · {selectedCount} selected · {rejectedCount} rejected
          </span>
          <button
            type="button"
            onClick={() => void refreshAssets()}
            disabled={!scopeReady || loading}
            className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-gray-200/80 bg-white px-3 text-xs font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300 dark:hover:border-blue-500/50 dark:hover:text-blue-300"
            title="同步服务端资产"
          >
            <RefreshIcon className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            同步
          </button>
        </div>
      </div>

      {error && (
        <div className="mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200">
          {error}
        </div>
      )}

      {!scopeReady ? (
        <div className="mt-4 rounded-lg border border-dashed border-gray-200 px-4 py-6 text-center text-xs text-gray-400 dark:border-white/[0.08] dark:text-gray-500">
          当前没有完整 scope，暂不能同步服务端资产。
        </div>
      ) : assets.length === 0 && !loading ? (
        <div className="mt-4 rounded-lg border border-dashed border-gray-200 px-4 py-6 text-center text-xs text-gray-400 dark:border-white/[0.08] dark:text-gray-500">
          当前 campaign 暂无服务端资产。MCP、REST、CLI 或 Web 托管模式生成后会显示在这里。
        </div>
      ) : (
        <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {assets.map((asset) => {
            const source = getMetadataValue(asset, 'source')
            const sessionId = getMetadataValue(asset, 'session_id')
            const batchId = getMetadataValue(asset, 'batch_id')
            const storyId = getMetadataValue(asset, 'story_id')
            const sceneId = getMetadataValue(asset, 'scene_id')
            const targetPath = getMetadataValue(asset, 'target_path')
            const busy = actionAssetId === asset.asset_id
            return (
              <article key={asset.asset_id} className="overflow-hidden rounded-lg border border-gray-200/80 bg-white dark:border-white/[0.08] dark:bg-gray-950/40">
                <div className="aspect-[4/3] bg-gray-100 dark:bg-white/[0.04]">
                  <img src={asset.delivery.thumbnail_url} alt={asset.prompt || asset.asset_id} className="h-full w-full object-cover" loading="lazy" />
                </div>
                <div className="space-y-3 p-3">
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <div className="line-clamp-2 text-sm font-medium text-gray-800 dark:text-gray-100" title={asset.prompt || asset.asset_id}>
                        {asset.prompt || asset.asset_id}
                      </div>
                      <div className="mt-1 truncate text-[11px] text-gray-500 dark:text-gray-400" title={asset.asset_id}>
                        {asset.asset_id}
                      </div>
                    </div>
                    <span className={`shrink-0 rounded-full border px-2 py-0.5 text-[10px] font-medium ${statusClassName(asset.status)}`}>
                      {asset.status}
                    </span>
                  </div>

                  <div className="grid grid-cols-2 gap-2">
                    <AssetField label="provider" value={asset.provider} />
                    <AssetField label="model" value={asset.model} />
                    <AssetField label="task" value={asset.task_id} />
                    <AssetField label="source" value={source} />
                    <AssetField label="session" value={sessionId} />
                    <AssetField label="batch" value={batchId} />
                    <AssetField label="story" value={storyId} />
                    <AssetField label="scene" value={sceneId} />
                    <AssetField label="created" value={formatAssetDate(asset.created_at)} />
                    <AssetField label="target" value={targetPath} />
                  </div>

                  <div className="flex flex-wrap items-center gap-2">
                    <button
                      type="button"
                      onClick={() => void selectAsset(asset)}
                      disabled={busy}
                      className="inline-flex h-8 items-center rounded-lg border border-emerald-200 bg-emerald-50 px-2.5 text-[11px] font-medium text-emerald-700 transition hover:bg-emerald-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-200"
                    >
                      Select
                    </button>
                    <button
                      type="button"
                      onClick={() => void rejectAsset(asset)}
                      disabled={busy}
                      className="inline-flex h-8 items-center rounded-lg border border-red-200 bg-red-50 px-2.5 text-[11px] font-medium text-red-700 transition hover:bg-red-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200"
                    >
                      Reject
                    </button>
                    <a
                      href={asset.delivery.download_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex h-8 items-center gap-1 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
                    >
                      <LinkIcon className="h-3.5 w-3.5" />
                      Original
                    </a>
                    <a
                      href={asset.delivery.metadata_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex h-8 items-center gap-1 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
                    >
                      <LinkIcon className="h-3.5 w-3.5" />
                      Metadata
                    </a>
                    <button
                      type="button"
                      onClick={() => void copyText(asset.asset_id, ' asset_id')}
                      className="inline-flex h-8 items-center gap-1 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
                      title="复制 asset_id"
                    >
                      <CopyIcon className="h-3.5 w-3.5" />
                      ID
                    </button>
                    <button
                      type="button"
                      onClick={() => void copyText(asset.delivery.download_url, ' delivery URL')}
                      className="inline-flex h-8 items-center gap-1 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
                      title="复制 delivery URL"
                    >
                      <CopyIcon className="h-3.5 w-3.5" />
                      URL
                    </button>
                  </div>
                </div>
              </article>
            )
          })}
        </div>
      )}
    </section>
  )
}
