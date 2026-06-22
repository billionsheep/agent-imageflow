import { memo, type FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useStore } from '../store'
import { copyTextToClipboard, getClipboardFailureMessage } from '../lib/clipboard'
import {
  getAgentImageflowAdminMe,
  AgentImageflowApiError,
  isAgentImageflowUnauthorizedError,
  listAgentImageflowAssets,
  listAgentImageflowRecentAssets,
  loginAgentImageflowAdmin,
  logoutAgentImageflowAdmin,
  normalizeAgentImageflowApiBaseUrl,
  rejectAgentImageflowAsset,
  selectAgentImageflowAsset,
  type AgentImageflowAssetListQuery,
  type AgentImageflowAssetResponse,
  type AgentImageflowAuth,
  type AgentImageflowAdminSessionResponse,
} from '../lib/agentImageflowApi'
import { CopyIcon, LinkIcon, RefreshIcon } from './icons'

const ASSET_PAGE_SIZE = 24
const MAX_RENDERED_SERVER_ASSETS = 120
const ASSET_FILTER_DEBOUNCE_MS = 300
type AssetLibraryMode = 'recent' | 'scope'

interface AssetFilters {
  status: string
  provider: string
  source: string
  sessionId: string
  batchId: string
  keyword: string
}

const EMPTY_ASSET_FILTERS: AssetFilters = {
  status: '',
  provider: '',
  source: '',
  sessionId: '',
  batchId: '',
  keyword: '',
}

function buildAuth(apiKey: string, basicUsername: string, basicPassword: string): AgentImageflowAuth {
  return {
    apiKey,
    basicUsername,
    basicPassword,
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

function AssetFilterInput({ label, value, placeholder, onChange }: { label: string; value: string; placeholder?: string; onChange: (value: string) => void }) {
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

function buildAssetListQuery(filters: AssetFilters, offset: number): AgentImageflowAssetListQuery {
  return {
    limit: ASSET_PAGE_SIZE,
    offset,
    status: filters.status,
    provider: filters.provider,
    source: filters.source,
    sessionId: filters.sessionId,
    batchId: filters.batchId,
    keyword: filters.keyword,
  }
}

function mergeAssets(current: AgentImageflowAssetResponse[], next: AgentImageflowAssetResponse[]): AgentImageflowAssetResponse[] {
  const seen = new Set(current.map((asset) => asset.asset_id))
  const merged = [...current]
  for (const asset of next) {
    if (seen.has(asset.asset_id)) continue
    seen.add(asset.asset_id)
    merged.push(asset)
  }
  return merged
}

function formatJSONSummary(value?: Record<string, unknown>): string {
  if (!value || Object.keys(value).length === 0) return ''
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return ''
  }
}

function safeLocalPathLabel(value?: string): string {
  if (!value) return ''
  const pieces = value.split(/[\\/]+/).filter(Boolean)
  const filename = pieces[pieces.length - 1]
  return filename ? `stored file: ${filename}` : 'stored file available'
}

interface ServerAssetCardProps {
  asset: AgentImageflowAssetResponse
  busy: boolean
  onSelectAsset: (asset: AgentImageflowAssetResponse) => void
  onRejectAsset: (asset: AgentImageflowAssetResponse) => void
  onCopyText: (text: string, label: string) => void
  onSwitchToAssetScope: (asset: AgentImageflowAssetResponse) => void
}

const ServerAssetCard = memo(function ServerAssetCard({
  asset,
  busy,
  onSelectAsset,
  onRejectAsset,
  onCopyText,
  onSwitchToAssetScope,
}: ServerAssetCardProps) {
  const source = getMetadataValue(asset, 'source')
  const sessionId = getMetadataValue(asset, 'session_id')
  const batchId = getMetadataValue(asset, 'batch_id')
  const storyId = getMetadataValue(asset, 'story_id')
  const sceneId = getMetadataValue(asset, 'scene_id')
  const targetPath = getMetadataValue(asset, 'target_path')
  const metadataSummary = useMemo(() => formatJSONSummary(asset.metadata_json), [asset.metadata_json])
  const parametersSummary = useMemo(() => formatJSONSummary(asset.parameters_json), [asset.parameters_json])
  const safeLocalPath = useMemo(() => safeLocalPathLabel(asset.delivery.local_path), [asset.delivery.local_path])

  return (
    <article className="overflow-hidden rounded-lg border border-gray-200/80 bg-white dark:border-white/[0.08] dark:bg-gray-950/40">
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
          <AssetField label="workspace" value={asset.workspace_id} />
          <AssetField label="project" value={asset.project_id} />
          <AssetField label="campaign" value={asset.campaign_id} />
          <AssetField label="provider" value={asset.provider} />
          <AssetField label="model" value={asset.model} />
          <AssetField label="task" value={asset.task_id} />
          <AssetField label="hash" value={asset.hash} />
          <AssetField label="source" value={source} />
          <AssetField label="session" value={sessionId} />
          <AssetField label="batch" value={batchId} />
          <AssetField label="story" value={storyId} />
          <AssetField label="scene" value={sceneId} />
          <AssetField label="created" value={formatAssetDate(asset.created_at)} />
          <AssetField label="target" value={targetPath} />
        </div>

        {(metadataSummary || parametersSummary || safeLocalPath) && (
          <details className="rounded-lg border border-gray-200 bg-gray-50/70 p-2 dark:border-white/[0.08] dark:bg-white/[0.03]">
            <summary className="cursor-pointer text-[11px] font-medium text-gray-500 dark:text-gray-300">Details</summary>
            <div className="mt-2 space-y-2">
              {safeLocalPath && (
                <div className="break-all text-[11px] text-gray-500 dark:text-gray-400">{safeLocalPath}</div>
              )}
              {metadataSummary && (
                <pre className="max-h-28 overflow-auto whitespace-pre-wrap break-words rounded-md bg-white p-2 text-[11px] text-gray-600 dark:bg-gray-950/50 dark:text-gray-300">{metadataSummary}</pre>
              )}
              {parametersSummary && (
                <pre className="max-h-28 overflow-auto whitespace-pre-wrap break-words rounded-md bg-white p-2 text-[11px] text-gray-600 dark:bg-gray-950/50 dark:text-gray-300">{parametersSummary}</pre>
              )}
            </div>
          </details>
        )}

        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => void onSelectAsset(asset)}
            disabled={busy}
            className="inline-flex h-8 items-center rounded-lg border border-emerald-200 bg-emerald-50 px-2.5 text-[11px] font-medium text-emerald-700 transition hover:bg-emerald-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-200"
          >
            Select
          </button>
          <button
            type="button"
            onClick={() => void onRejectAsset(asset)}
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
            onClick={() => void onCopyText(asset.asset_id, ' asset_id')}
            className="inline-flex h-8 items-center gap-1 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
            title="复制 asset_id"
          >
            <CopyIcon className="h-3.5 w-3.5" />
            ID
          </button>
          <button
            type="button"
            onClick={() => void onCopyText(asset.delivery.download_url, ' delivery URL')}
            className="inline-flex h-8 items-center gap-1 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
            title="复制 delivery URL"
          >
            <CopyIcon className="h-3.5 w-3.5" />
            URL
          </button>
          <button
            type="button"
            onClick={() => onSwitchToAssetScope(asset)}
            className="inline-flex h-8 items-center rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
            title="切换到该资产所在 scope"
          >
            Scope
          </button>
        </div>
      </div>
    </article>
  )
})

export default function ServerAssetLibrary() {
  const imageflowApiBaseUrl = useStore((state) => state.settings.imageflowApiBaseUrl)
  const imageflowApiKey = useStore((state) => state.settings.imageflowApiKey)
  const imageflowBasicUsername = useStore((state) => state.settings.imageflowBasicUsername)
  const imageflowBasicPassword = useStore((state) => state.settings.imageflowBasicPassword)
  const imageflowWorkspaceId = useStore((state) => state.settings.imageflowWorkspaceId)
  const imageflowProjectId = useStore((state) => state.settings.imageflowProjectId)
  const imageflowCampaignId = useStore((state) => state.settings.imageflowCampaignId)
  const setSettings = useStore((state) => state.setSettings)
  const showToast = useStore((state) => state.showToast)
  const baseUrl = useMemo(
    () => normalizeAgentImageflowApiBaseUrl(imageflowApiBaseUrl),
    [imageflowApiBaseUrl],
  )
  const auth = useMemo(
    () => buildAuth(imageflowApiKey, imageflowBasicUsername, imageflowBasicPassword),
    [imageflowApiKey, imageflowBasicPassword, imageflowBasicUsername],
  )
  const consoleAuth = useMemo<AgentImageflowAuth>(() => ({
    basicUsername: imageflowBasicUsername,
    basicPassword: imageflowBasicPassword,
  }), [imageflowBasicPassword, imageflowBasicUsername])
  const scope = useMemo(() => ({
    workspaceId: imageflowWorkspaceId.trim(),
    projectId: imageflowProjectId.trim(),
    campaignId: imageflowCampaignId.trim(),
  }), [
    imageflowCampaignId,
    imageflowProjectId,
    imageflowWorkspaceId,
  ])
  const scopeReady = Boolean(scope.workspaceId && scope.projectId && scope.campaignId)
  const [assets, setAssets] = useState<AgentImageflowAssetResponse[]>([])
  const [loading, setLoading] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [actionAssetId, setActionAssetId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [unauthorized, setUnauthorized] = useState(false)
  const [hasMore, setHasMore] = useState(false)
  const [mode, setMode] = useState<AssetLibraryMode>('recent')
  const [adminSession, setAdminSession] = useState<AgentImageflowAdminSessionResponse | null>(null)
  const [adminChecking, setAdminChecking] = useState(false)
  const [adminUsername, setAdminUsername] = useState(imageflowBasicUsername || 'admin')
  const [adminPassword, setAdminPassword] = useState('')
  const [adminLoginBusy, setAdminLoginBusy] = useState(false)
  const [draftFilters, setDraftFilters] = useState<AssetFilters>(EMPTY_ASSET_FILTERS)
  const [filters, setFilters] = useState<AssetFilters>(EMPTY_ASSET_FILTERS)
  const requestRef = useRef(0)

  useEffect(() => {
    if (adminUsername || !imageflowBasicUsername) return
    setAdminUsername(imageflowBasicUsername)
  }, [adminUsername, imageflowBasicUsername])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setFilters(draftFilters)
    }, ASSET_FILTER_DEBOUNCE_MS)
    return () => window.clearTimeout(timer)
  }, [draftFilters])

  useEffect(() => {
    let cancelled = false
    setAdminChecking(true)
    void getAgentImageflowAdminMe(baseUrl)
      .then((session) => {
        if (cancelled) return
        setAdminSession(session)
        if (session.authenticated) setUnauthorized(false)
      })
      .catch((nextError) => {
        if (cancelled) return
        const configured = !(nextError instanceof AgentImageflowApiError && nextError.errorCode === 'admin_not_configured')
        setAdminSession({ authenticated: false, configured })
      })
      .finally(() => {
        if (!cancelled) setAdminChecking(false)
      })
    return () => {
      cancelled = true
    }
  }, [baseUrl])

  const loadAssets = useCallback(async (loadMode: 'replace' | 'append', offset: number) => {
    setError(null)
    if (loadMode === 'replace') setUnauthorized(false)
    const libraryMode = mode
    if (libraryMode === 'scope' && !scopeReady) {
      requestRef.current += 1
      setHasMore(false)
      setLoading(false)
      setLoadingMore(false)
      return
    }
    const requestId = ++requestRef.current
    if (loadMode === 'append') {
      setLoadingMore(true)
    } else {
      setLoading(true)
    }
    try {
      const query = buildAssetListQuery(filters, offset)
      const response = libraryMode === 'recent'
        ? await listAgentImageflowRecentAssets(baseUrl, query, consoleAuth)
        : await listAgentImageflowAssets(baseUrl, {
          projectId: scope.projectId,
          campaignId: scope.campaignId,
        }, auth, query)
      if (requestRef.current !== requestId) return
      const loadedCount = loadMode === 'append'
        ? Math.min(offset + response.length, MAX_RENDERED_SERVER_ASSETS)
        : Math.min(response.length, MAX_RENDERED_SERVER_ASSETS)
      setAssets((current) => {
        const nextAssets = loadMode === 'append' ? mergeAssets(current, response) : response
        return nextAssets.slice(0, MAX_RENDERED_SERVER_ASSETS)
      })
      setHasMore(response.length === ASSET_PAGE_SIZE && loadedCount < MAX_RENDERED_SERVER_ASSETS)
      setUnauthorized(false)
    } catch (nextError) {
      if (requestRef.current !== requestId) return
      if (isAgentImageflowUnauthorizedError(nextError)) {
        setUnauthorized(true)
        setError(null)
      } else {
        setError(nextError instanceof Error ? nextError.message : String(nextError))
      }
      if (loadMode === 'replace') {
        setHasMore(false)
      }
    } finally {
      if (requestRef.current === requestId) {
        setLoading(false)
        setLoadingMore(false)
      }
    }
  }, [auth, baseUrl, consoleAuth, filters, mode, scope.campaignId, scope.projectId, scopeReady])

  const refreshAssets = useCallback(async () => {
    await loadAssets('replace', 0)
  }, [loadAssets])

  useEffect(() => {
    void refreshAssets()
  }, [refreshAssets])

  const updateAsset = useCallback((nextAsset: AgentImageflowAssetResponse) => {
    setAssets((current) => current.map((asset) => asset.asset_id === nextAsset.asset_id ? nextAsset : asset))
  }, [])

  const selectAsset = useCallback(async (asset: AgentImageflowAssetResponse) => {
    setActionAssetId(asset.asset_id)
    try {
      updateAsset(await selectAgentImageflowAsset(baseUrl, asset.asset_id, auth))
      showToast('已标记为 selected', 'success')
    } catch (nextError) {
      showToast(nextError instanceof Error ? nextError.message : String(nextError), 'error')
    } finally {
      setActionAssetId(null)
    }
  }, [auth, baseUrl, showToast, updateAsset])

  const rejectAsset = useCallback(async (asset: AgentImageflowAssetResponse) => {
    setActionAssetId(asset.asset_id)
    try {
      updateAsset(await rejectAgentImageflowAsset(baseUrl, asset.asset_id, auth))
      showToast('已标记为 rejected', 'success')
    } catch (nextError) {
      showToast(nextError instanceof Error ? nextError.message : String(nextError), 'error')
    } finally {
      setActionAssetId(null)
    }
  }, [auth, baseUrl, showToast, updateAsset])

  const copyText = useCallback(async (text: string, label: string) => {
    try {
      await copyTextToClipboard(text)
      showToast(`已复制${label}`, 'success')
    } catch (nextError) {
      showToast(getClipboardFailureMessage('复制失败', nextError), 'error')
    }
  }, [showToast])

  const handleAdminLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const username = adminUsername.trim()
    if (!username || !adminPassword) {
      showToast('请输入 Admin 用户名和密码', 'error')
      return
    }
    setAdminLoginBusy(true)
    try {
      const session = await loginAgentImageflowAdmin(baseUrl, { username, password: adminPassword })
      setAdminSession(session)
      setAdminPassword('')
      setUnauthorized(false)
      showToast('已登录控制台', 'success')
      await loadAssets('replace', 0)
    } catch (nextError) {
      showToast(nextError instanceof Error ? nextError.message : String(nextError), 'error')
    } finally {
      setAdminLoginBusy(false)
    }
  }

  const handleAdminLogout = async () => {
    setAdminLoginBusy(true)
    try {
      const session = await logoutAgentImageflowAdmin(baseUrl)
      setAdminSession(session)
      setAssets([])
      setHasMore(false)
      setUnauthorized(mode === 'recent')
      showToast('已退出控制台', 'success')
    } catch (nextError) {
      showToast(nextError instanceof Error ? nextError.message : String(nextError), 'error')
    } finally {
      setAdminLoginBusy(false)
    }
  }

  const switchToAssetScope = useCallback((asset: AgentImageflowAssetResponse) => {
    const workspaceId = asset.workspace_id?.trim()
    const projectId = asset.project_id?.trim()
    const campaignId = asset.campaign_id?.trim()
    if (!workspaceId || !projectId || !campaignId) {
      showToast('该资产缺少完整 scope，无法切换', 'error')
      return
    }
    setSettings({
      imageflowManagedMode: true,
      imageflowWorkspaceId: workspaceId,
      imageflowProjectId: projectId,
      imageflowCampaignId: campaignId,
    })
    setMode((current) => current === 'scope' ? current : 'scope')
    showToast('已切换到该资产所在 scope', 'success')
  }, [setSettings, showToast])

  const setTextFilter = (key: Exclude<keyof AssetFilters, 'status'>, value: string) => {
    setDraftFilters((current) => ({ ...current, [key]: value }))
  }

  const setStatusFilter = (status: string) => {
    const nextFilters = { ...draftFilters, status }
    setDraftFilters(nextFilters)
    setFilters(nextFilters)
  }

  const clearFilters = () => {
    setDraftFilters(EMPTY_ASSET_FILTERS)
    setFilters(EMPTY_ASSET_FILTERS)
  }

  const selectedCount = assets.filter((asset) => asset.status === 'selected').length
  const rejectedCount = assets.filter((asset) => asset.status === 'rejected').length
  const filtersActive = Object.values(draftFilters).some((value) => value.trim())
  const adminConfigured = adminSession?.configured !== false
  const adminAuthenticated = Boolean(adminSession?.authenticated)
  const summaryText = unauthorized
    ? 'unauthorized'
    : `${assets.length} shown · ${selectedCount} selected · ${rejectedCount} rejected${loading && assets.length > 0 ? ' · refreshing' : ''}`
  const emptyText = filtersActive
    ? '当前筛选没有匹配资产。'
    : mode === 'recent'
      ? '最近资产为空。MCP、REST、CLI 或 Web 托管模式生成后会显示在这里。'
      : '当前 campaign 暂无服务端资产。MCP、REST、CLI 或 Web 托管模式生成后会显示在这里。'

  return (
    <section data-no-drag-select className="mb-5 rounded-lg border border-gray-200/80 bg-white/90 p-4 shadow-sm dark:border-white/[0.08] dark:bg-white/[0.03]">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="min-w-0">
          <div className="text-sm font-semibold text-gray-800 dark:text-gray-100">服务端资产库</div>
          <div className="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
            {mode === 'recent'
              ? 'Recent Assets · 跨 workspace / project / campaign'
              : scopeReady ? `${scope.workspaceId} / ${scope.projectId} / ${scope.campaignId}` : '请在设置或 Scope 管理中选择完整 scope'}
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="inline-flex h-9 overflow-hidden rounded-lg border border-gray-200 bg-white text-xs dark:border-white/[0.08] dark:bg-white/[0.04]">
            <button
              type="button"
              onClick={() => setMode((current) => current === 'recent' ? current : 'recent')}
              className={`px-3 font-medium transition ${mode === 'recent' ? 'bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-200' : 'text-gray-500 hover:text-blue-600 dark:text-gray-300'}`}
            >
              Recent
            </button>
            <button
              type="button"
              onClick={() => setMode((current) => current === 'scope' ? current : 'scope')}
              className={`border-l border-gray-200 px-3 font-medium transition dark:border-white/[0.08] ${mode === 'scope' ? 'bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-200' : 'text-gray-500 hover:text-blue-600 dark:text-gray-300'}`}
            >
              Scope
            </button>
          </div>
          <span className="rounded-full border border-gray-200 bg-gray-50 px-2.5 py-1 text-[11px] text-gray-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">
            {summaryText}
          </span>
          {adminChecking ? (
            <span className="rounded-full border border-gray-200 bg-gray-50 px-2.5 py-1 text-[11px] text-gray-500 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">
              checking session
            </span>
          ) : adminAuthenticated ? (
            <button
              type="button"
              onClick={() => void handleAdminLogout()}
              disabled={adminLoginBusy}
              className="h-9 rounded-lg border border-gray-200 bg-white px-3 text-xs font-medium text-gray-500 transition hover:border-blue-300 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
              title={`已登录 ${adminSession?.username ?? 'admin'}`}
            >
              Logout
            </button>
          ) : null}
          {!adminConfigured && (
            <span className="rounded-full border border-amber-200 bg-amber-50 px-2.5 py-1 text-[11px] text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-200">
              admin disabled
            </span>
          )}
          <button
            type="button"
            onClick={() => void refreshAssets()}
            disabled={(mode === 'scope' && !scopeReady) || loading}
            className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-gray-200/80 bg-white px-3 text-xs font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300 dark:hover:border-blue-500/50 dark:hover:text-blue-300"
            title="同步服务端资产"
          >
            <RefreshIcon className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            同步
          </button>
        </div>
      </div>

      <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-6">
        <label className="min-w-0 text-[11px] text-gray-500 dark:text-gray-400">
          <span className="mb-1 block uppercase">status</span>
          <select
            value={draftFilters.status}
            onChange={(event) => setStatusFilter(event.target.value)}
            className="h-9 w-full min-w-0 rounded-lg border border-gray-200 bg-white px-2.5 text-xs text-gray-700 outline-none transition focus:border-blue-300 dark:border-white/[0.08] dark:bg-gray-950/50 dark:text-gray-100 dark:focus:border-blue-500/60"
          >
            <option value="">All</option>
            <option value="generated">Generated</option>
            <option value="selected">Selected</option>
            <option value="rejected">Rejected</option>
            <option value="published">Published</option>
          </select>
        </label>
        <AssetFilterInput label="provider" value={draftFilters.provider} placeholder="mock" onChange={(value) => setTextFilter('provider', value)} />
        <AssetFilterInput label="source" value={draftFilters.source} placeholder="mcp / web" onChange={(value) => setTextFilter('source', value)} />
        <AssetFilterInput label="session" value={draftFilters.sessionId} placeholder="session_id" onChange={(value) => setTextFilter('sessionId', value)} />
        <AssetFilterInput label="batch" value={draftFilters.batchId} placeholder="batch_id" onChange={(value) => setTextFilter('batchId', value)} />
        <AssetFilterInput label="keyword" value={draftFilters.keyword} placeholder="prompt / id" onChange={(value) => setTextFilter('keyword', value)} />
      </div>
      {filtersActive && (
        <div className="mt-2 flex justify-end">
          <button
            type="button"
            onClick={clearFilters}
            className="h-8 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] font-medium text-gray-500 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
          >
            清除筛选
          </button>
        </div>
      )}

      {error && (
        <div className="mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200">
          {error}
        </div>
      )}
      {loading && assets.length > 0 && (
        <div className="mt-3 rounded-lg border border-blue-200 bg-blue-50 px-3 py-2 text-xs text-blue-700 dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-200">
          正在刷新服务端资产，当前列表会保留到新结果返回。
        </div>
      )}

      {unauthorized ? (
        <div className="mt-4 rounded-lg border border-amber-200 bg-amber-50/70 p-3 dark:border-amber-500/20 dark:bg-amber-500/10">
          {!adminConfigured ? (
            <div className="text-xs text-amber-800 dark:text-amber-100">
              控制台 Admin 登录未配置。请在服务端设置 ADMIN_USERNAME / ADMIN_PASSWORD，或复用 BASIC_AUTH_USERNAME / BASIC_AUTH_PASSWORD。
            </div>
          ) : (
            <form onSubmit={handleAdminLogin} className="grid gap-2 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] md:items-end">
              <label className="min-w-0 text-[11px] text-amber-800 dark:text-amber-100">
                <span className="mb-1 block uppercase">admin</span>
                <input
                  value={adminUsername}
                  onChange={(event) => setAdminUsername(event.target.value)}
                  className="h-9 w-full min-w-0 rounded-lg border border-amber-200 bg-white px-2.5 text-xs text-gray-800 outline-none transition focus:border-amber-400 dark:border-amber-500/30 dark:bg-gray-950/70 dark:text-gray-100"
                  autoComplete="username"
                />
              </label>
              <label className="min-w-0 text-[11px] text-amber-800 dark:text-amber-100">
                <span className="mb-1 block uppercase">password</span>
                <input
                  type="password"
                  value={adminPassword}
                  onChange={(event) => setAdminPassword(event.target.value)}
                  className="h-9 w-full min-w-0 rounded-lg border border-amber-200 bg-white px-2.5 text-xs text-gray-800 outline-none transition focus:border-amber-400 dark:border-amber-500/30 dark:bg-gray-950/70 dark:text-gray-100"
                  autoComplete="current-password"
                />
              </label>
              <button
                type="submit"
                disabled={adminLoginBusy}
                className="h-9 rounded-lg border border-amber-300 bg-white px-3 text-xs font-medium text-amber-700 transition hover:border-amber-400 hover:text-amber-800 disabled:cursor-not-allowed disabled:opacity-50 dark:border-amber-500/30 dark:bg-white/[0.04] dark:text-amber-100"
              >
                Login
              </button>
            </form>
          )}
        </div>
      ) : mode === 'scope' && !scopeReady ? (
        <div className="mt-4 rounded-lg border border-dashed border-gray-200 px-4 py-6 text-center text-xs text-gray-400 dark:border-white/[0.08] dark:text-gray-500">
          当前没有完整 scope，暂不能同步服务端资产。
        </div>
      ) : assets.length === 0 && loading ? (
        <div className="mt-4 rounded-lg border border-dashed border-gray-200 px-4 py-6 text-center text-xs text-gray-400 dark:border-white/[0.08] dark:text-gray-500">
          正在同步服务端资产...
        </div>
      ) : assets.length === 0 && !loading ? (
        <div className="mt-4 rounded-lg border border-dashed border-gray-200 px-4 py-6 text-center text-xs text-gray-400 dark:border-white/[0.08] dark:text-gray-500">
          {emptyText}
        </div>
      ) : (
        <>
          <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {assets.map((asset) => (
              <ServerAssetCard
                key={asset.asset_id}
                asset={asset}
                busy={actionAssetId === asset.asset_id}
                onSelectAsset={selectAsset}
                onRejectAsset={rejectAsset}
                onCopyText={copyText}
                onSwitchToAssetScope={switchToAssetScope}
              />
            ))}
          </div>
          {hasMore && (
            <div className="mt-4 flex justify-center">
              <button
                type="button"
                onClick={() => void loadAssets('append', assets.length)}
                disabled={loadingMore}
                className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-gray-200/80 bg-white px-3 text-xs font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300 dark:hover:border-blue-500/50 dark:hover:text-blue-300"
              >
                <RefreshIcon className={`h-4 w-4 ${loadingMore ? 'animate-spin' : ''}`} />
                加载更多
              </button>
            </div>
          )}
        </>
      )}
    </section>
  )
}
