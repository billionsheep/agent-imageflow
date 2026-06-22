import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useStore } from '../store'
import { copyTextToClipboard, getClipboardFailureMessage } from '../lib/clipboard'
import {
  getAgentImageflowAdminMe,
  getAgentImageflowRuntimeStatus,
  AgentImageflowApiError,
  isAgentImageflowUnauthorizedError,
  listAgentImageflowCampaigns,
  listAgentImageflowAssets,
  listAgentImageflowProjects,
  listAgentImageflowRecentAssets,
  listAgentImageflowWorkspaces,
  normalizeAgentImageflowApiBaseUrl,
  rejectAgentImageflowAsset,
  resolveAgentImageflowDeliveryUrl,
  selectAgentImageflowAsset,
  type AgentImageflowAssetListQuery,
  type AgentImageflowAssetResponse,
  type AgentImageflowAuth,
  type AgentImageflowAdminSessionResponse,
  type AgentImageflowCampaign,
  type AgentImageflowProject,
  type AgentImageflowRuntimeStatusResponse,
  type AgentImageflowWorkspace,
} from '../lib/agentImageflowApi'
import {
  getAssetReviewTitle,
  getAssetReviewSummary,
  getAssetTechnicalFields,
  getLocalhostMismatchWarning,
  getProductionFiltersFromAsset,
} from '../lib/operatorReview'
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

function displayScopeName(item?: { name?: string }, fallback?: string): string {
  return item?.name?.trim() || fallback || ''
}

function pickScopeId<T>(items: T[], preferredId: string, getId: (item: T) => string, isArchived: (item: T) => boolean | undefined): string {
  const preferred = preferredId.trim()
  if (preferred && items.some((item) => getId(item) === preferred)) return preferred
  const fallback = items.find((item) => !isArchived(item)) ?? items[0]
  return fallback ? getId(fallback) : ''
}

function getRateLimitFriendlyMessage(error: unknown): string {
  if (error instanceof AgentImageflowApiError && error.status === 429) {
    return '请求太快，服务端正在限流。请稍后重试，或缩小筛选/刷新频率。'
  }
  return error instanceof Error ? error.message : String(error)
}

interface ServerAssetCardProps {
  asset: AgentImageflowAssetResponse
  baseUrl: string
  busy: boolean
  onSelectAsset: (asset: AgentImageflowAssetResponse) => void
  onRejectAsset: (asset: AgentImageflowAssetResponse) => void
  onMarkAsReference: (asset: AgentImageflowAssetResponse) => void
  onOpenProductionView: (asset: AgentImageflowAssetResponse) => void
  onCopyText: (text: string, label: string) => void
  onSwitchToAssetScope: (asset: AgentImageflowAssetResponse) => void
}

const ServerAssetCard = memo(function ServerAssetCard({
  asset,
  baseUrl,
  busy,
  onSelectAsset,
  onRejectAsset,
  onMarkAsReference,
  onOpenProductionView,
  onCopyText,
  onSwitchToAssetScope,
}: ServerAssetCardProps) {
  const reviewSummary = useMemo(() => getAssetReviewSummary(asset), [asset])
  const reviewTitle = useMemo(() => getAssetReviewTitle(asset), [asset])
  const technicalFields = useMemo(() => getAssetTechnicalFields(asset), [asset])
  const productionFilters = useMemo(() => getProductionFiltersFromAsset(asset), [asset])
  const visibleReviewFields = reviewSummary.filter((field) => field.key !== 'prompt')
  const thumbnailUrl = resolveAgentImageflowDeliveryUrl(baseUrl, asset.delivery.thumbnail_url)
  const originalUrl = resolveAgentImageflowDeliveryUrl(baseUrl, asset.delivery.download_url)
  const metadataUrl = resolveAgentImageflowDeliveryUrl(baseUrl, asset.delivery.metadata_url)

  return (
    <article className="overflow-hidden rounded-lg border border-gray-200/80 bg-white dark:border-white/[0.08] dark:bg-gray-950/40">
      <div className="aspect-[4/3] bg-gray-100 dark:bg-white/[0.04]">
        <img src={thumbnailUrl} alt={reviewTitle || asset.asset_id} className="h-full w-full object-cover" loading="lazy" />
      </div>
      <div className="space-y-3 p-3">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <div className="line-clamp-2 text-sm font-medium text-gray-800 dark:text-gray-100" title={reviewTitle || asset.asset_id}>
              {reviewTitle || asset.asset_id}
            </div>
          </div>
          <span className={`shrink-0 rounded-full border px-2 py-0.5 text-[10px] font-medium ${statusClassName(asset.status)}`}>
            {asset.status}
          </span>
        </div>

        {visibleReviewFields.length > 0 && (
          <div className="grid grid-cols-2 gap-2 rounded-lg border border-gray-200 bg-gray-50/70 p-2 dark:border-white/[0.08] dark:bg-white/[0.03]">
            {visibleReviewFields.map((field) => (
              <AssetField
                key={field.key}
                label={field.label}
                value={field.key === 'created' ? formatAssetDate(field.value) : field.value}
              />
            ))}
          </div>
        )}

        {technicalFields.length > 0 && (
          <details className="rounded-lg border border-gray-200 bg-gray-50/70 p-2 dark:border-white/[0.08] dark:bg-white/[0.03]">
            <summary className="cursor-pointer text-[11px] font-medium text-gray-500 dark:text-gray-300">技术详情</summary>
            <div className="mt-2 grid grid-cols-2 gap-2">
              {technicalFields.map((field) => (
                field.key === 'metadata' || field.key === 'parameters' ? (
                  <div key={field.key} className="col-span-2 min-w-0">
                    <div className="text-[10px] uppercase text-gray-400 dark:text-gray-500">{field.label}</div>
                    <pre className="mt-1 max-h-28 overflow-auto whitespace-pre-wrap break-words rounded-md bg-white p-2 text-[11px] text-gray-600 dark:bg-gray-950/50 dark:text-gray-300">{field.value}</pre>
                  </div>
                ) : (
                  <AssetField key={field.key} label={field.label} value={field.value} />
                )
              ))}
            </div>
            <div className="mt-3 flex flex-wrap items-center gap-2 border-t border-gray-200 pt-2 dark:border-white/[0.08]">
              <a
                href={metadataUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex h-8 items-center gap-1 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
              >
                <LinkIcon className="h-3.5 w-3.5" />
                元数据
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
                onClick={() => void onCopyText(originalUrl, ' delivery URL')}
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
                切换空间
              </button>
              <button
                type="button"
                onClick={() => onMarkAsReference(asset)}
                className="inline-flex h-8 items-center rounded-lg border border-blue-200 bg-blue-50 px-2.5 text-[11px] font-medium text-blue-700 transition hover:border-blue-300 hover:bg-blue-100 dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-200"
                title="标记为当前 project 的参考图"
              >
                参考图
              </button>
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
            选中
          </button>
          <button
            type="button"
            onClick={() => void onRejectAsset(asset)}
            disabled={busy}
            className="inline-flex h-8 items-center rounded-lg border border-red-200 bg-red-50 px-2.5 text-[11px] font-medium text-red-700 transition hover:bg-red-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200"
          >
            拒绝
          </button>
          <a
            href={originalUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex h-8 items-center gap-1 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
          >
            <LinkIcon className="h-3.5 w-3.5" />
            原图
          </a>
          {productionFilters && (
            <button
              type="button"
              onClick={() => onOpenProductionView(asset)}
              className="inline-flex h-8 items-center rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
              title="用该资产的 session / batch 打开批次生产视图"
            >
              批次
            </button>
          )}
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
  const setShowScopeManager = useStore((state) => state.setShowScopeManager)
  const setShowProjectContext = useStore((state) => state.setShowProjectContext)
  const setShowProductionView = useStore((state) => state.setShowProductionView)
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
  const hostMismatchWarning = useMemo(() => (
    typeof window === 'undefined'
      ? null
      : getLocalhostMismatchWarning(window.location.origin, baseUrl)
  ), [baseUrl])
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
  const [draftFilters, setDraftFilters] = useState<AssetFilters>(EMPTY_ASSET_FILTERS)
  const [filters, setFilters] = useState<AssetFilters>(EMPTY_ASSET_FILTERS)
  const [runtimeStatus, setRuntimeStatus] = useState<AgentImageflowRuntimeStatusResponse | null>(null)
  const [runtimeError, setRuntimeError] = useState<string | null>(null)
  const [scopeLoading, setScopeLoading] = useState(false)
  const [scopeError, setScopeError] = useState<string | null>(null)
  const [workspaces, setWorkspaces] = useState<AgentImageflowWorkspace[]>([])
  const [projects, setProjects] = useState<AgentImageflowProject[]>([])
  const [campaigns, setCampaigns] = useState<AgentImageflowCampaign[]>([])
  const requestRef = useRef(0)
  const scopeRequestRef = useRef(0)

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

  useEffect(() => {
    if (!adminSession?.authenticated) {
      setRuntimeStatus(null)
      setRuntimeError(null)
      return
    }
    let cancelled = false
    setRuntimeError(null)
    void getAgentImageflowRuntimeStatus(baseUrl)
      .then((status) => {
        if (!cancelled) setRuntimeStatus(status)
      })
      .catch((nextError) => {
        if (!cancelled) {
          setRuntimeStatus(null)
          setRuntimeError(getRateLimitFriendlyMessage(nextError))
        }
      })
    return () => {
      cancelled = true
    }
  }, [adminSession?.authenticated, baseUrl])

  const commitScope = useCallback((workspaceId: string, projectId: string, campaignId: string) => {
    const nextWorkspaceId = workspaceId.trim()
    const nextProjectId = projectId.trim()
    const nextCampaignId = campaignId.trim()
    if (!nextWorkspaceId || !nextProjectId || !nextCampaignId) return
    const current = useStore.getState().settings
    if (
      current.imageflowWorkspaceId.trim() === nextWorkspaceId &&
      current.imageflowProjectId.trim() === nextProjectId &&
      current.imageflowCampaignId.trim() === nextCampaignId &&
      current.imageflowManagedMode
    ) {
      return
    }
    setSettings({
      imageflowManagedMode: true,
      imageflowWorkspaceId: nextWorkspaceId,
      imageflowProjectId: nextProjectId,
      imageflowCampaignId: nextCampaignId,
    })
  }, [setSettings])

  const reloadScopeHierarchy = useCallback(async (preferredWorkspaceId = scope.workspaceId, preferredProjectId = scope.projectId, preferredCampaignId = scope.campaignId) => {
    if (!adminSession?.authenticated && !consoleAuth.basicUsername && !consoleAuth.basicPassword) return
    const requestId = ++scopeRequestRef.current
    setScopeLoading(true)
    setScopeError(null)
    try {
      const workspaceResponse = await listAgentImageflowWorkspaces(baseUrl, consoleAuth)
      if (scopeRequestRef.current !== requestId) return
      const nextWorkspaces = workspaceResponse.workspaces ?? []
      setWorkspaces(nextWorkspaces)
      const nextWorkspaceId = pickScopeId(nextWorkspaces, preferredWorkspaceId, (item) => item.workspace_id, (item) => item.archived)
      if (!nextWorkspaceId) {
        setProjects([])
        setCampaigns([])
        return
      }

      const projectResponse = await listAgentImageflowProjects(baseUrl, nextWorkspaceId, consoleAuth)
      if (scopeRequestRef.current !== requestId) return
      const nextProjects = projectResponse.projects ?? []
      setProjects(nextProjects)
      const nextProjectId = pickScopeId(nextProjects, preferredProjectId, (item) => item.project_id, (item) => item.archived)
      if (!nextProjectId) {
        setCampaigns([])
        return
      }

      const campaignResponse = await listAgentImageflowCampaigns(baseUrl, {
        workspaceId: nextWorkspaceId,
        projectId: nextProjectId,
      }, consoleAuth)
      if (scopeRequestRef.current !== requestId) return
      const nextCampaigns = campaignResponse.campaigns ?? []
      setCampaigns(nextCampaigns)
      const nextCampaignId = pickScopeId(nextCampaigns, preferredCampaignId, (item) => item.campaign_id, (item) => item.archived)
      commitScope(nextWorkspaceId, nextProjectId, nextCampaignId)
    } catch (nextError) {
      if (scopeRequestRef.current !== requestId) return
      setScopeError(getRateLimitFriendlyMessage(nextError))
    } finally {
      if (scopeRequestRef.current === requestId) setScopeLoading(false)
    }
  }, [adminSession?.authenticated, baseUrl, commitScope, consoleAuth, scope.campaignId, scope.projectId, scope.workspaceId])

  useEffect(() => {
    void reloadScopeHierarchy()
  }, [reloadScopeHierarchy])

  const handleWorkspaceChange = useCallback(async (workspaceId: string) => {
    await reloadScopeHierarchy(workspaceId, '', '')
  }, [reloadScopeHierarchy])

  const handleProjectChange = useCallback(async (projectId: string) => {
    await reloadScopeHierarchy(scope.workspaceId, projectId, '')
  }, [reloadScopeHierarchy, scope.workspaceId])

  const handleCampaignChange = useCallback((campaignId: string) => {
    commitScope(scope.workspaceId, scope.projectId, campaignId)
  }, [commitScope, scope.projectId, scope.workspaceId])

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
        setError(getRateLimitFriendlyMessage(nextError))
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
      showToast('已标记为选中', 'success')
    } catch (nextError) {
      showToast(getRateLimitFriendlyMessage(nextError), 'error')
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
      showToast(getRateLimitFriendlyMessage(nextError), 'error')
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

  const markAsReference = useCallback((asset: AgentImageflowAssetResponse) => {
    const assetProjectId = asset.project_id?.trim()
    if (assetProjectId && assetProjectId !== scope.projectId) {
      setSettings({
        imageflowManagedMode: true,
        imageflowWorkspaceId: asset.workspace_id?.trim() || scope.workspaceId,
        imageflowProjectId: assetProjectId,
        imageflowCampaignId: asset.campaign_id?.trim() || scope.campaignId,
      })
    }
    setShowProjectContext(true, asset.asset_id)
  }, [scope.campaignId, scope.projectId, scope.workspaceId, setSettings, setShowProjectContext])

  const openProductionViewFromAsset = useCallback((asset: AgentImageflowAssetResponse) => {
    const filters = getProductionFiltersFromAsset(asset)
    if (!filters) {
      showToast('该资产缺少 session_id 或 batch_id，无法打开批次生产视图', 'error')
      return
    }
    const workspaceId = asset.workspace_id?.trim()
    const projectId = asset.project_id?.trim()
    const campaignId = asset.campaign_id?.trim()
    if (!projectId || !campaignId) {
      showToast('该资产缺少 project / campaign，无法打开批次生产视图', 'error')
      return
    }
    setSettings({
      imageflowManagedMode: true,
      ...(workspaceId ? { imageflowWorkspaceId: workspaceId } : {}),
      imageflowProjectId: projectId,
      imageflowCampaignId: campaignId,
    })
    setMode((current) => current === 'scope' ? current : 'scope')
    setShowProductionView(true, filters)
  }, [setSettings, setShowProductionView, showToast])

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
  const currentWorkspace = workspaces.find((item) => item.workspace_id === scope.workspaceId)
  const currentProject = projects.find((item) => item.project_id === scope.projectId)
  const currentCampaign = campaigns.find((item) => item.campaign_id === scope.campaignId)
  const runtimeProvider = runtimeStatus?.default_provider?.trim() || 'unknown'
  const openAIStatus = runtimeStatus?.providers?.openai_compatible
  const falStatus = runtimeStatus?.providers?.fal
  const providerSummary = runtimeStatus
    ? `${runtimeProvider}${openAIStatus?.configured ? ` · openai-compatible ${openAIStatus.model || ''}`.trimEnd() : falStatus?.configured ? ` · fal ${falStatus.model || ''}`.trimEnd() : ' · 未配置真实 provider key'}`
    : runtimeError || '状态不可用'
  const summaryText = unauthorized
    ? '未授权'
    : `${assets.length} 张 · ${selectedCount} 已选中 · ${rejectedCount} 已拒绝${loading && assets.length > 0 ? ' · 刷新中' : ''}`
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
              ? '最近资产 · 跨 workspace / project / campaign'
              : scopeReady ? `${scope.workspaceId} / ${scope.projectId} / ${scope.campaignId}` : '请在设置或业务空间管理中选择完整 scope'}
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="inline-flex h-9 overflow-hidden rounded-lg border border-gray-200 bg-white text-xs dark:border-white/[0.08] dark:bg-white/[0.04]">
            <button
              type="button"
              onClick={() => setMode((current) => current === 'recent' ? current : 'recent')}
              className={`px-3 font-medium transition ${mode === 'recent' ? 'bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-200' : 'text-gray-500 hover:text-blue-600 dark:text-gray-300'}`}
            >
              最近
            </button>
            <button
              type="button"
              onClick={() => setMode((current) => current === 'scope' ? current : 'scope')}
              className={`border-l border-gray-200 px-3 font-medium transition dark:border-white/[0.08] ${mode === 'scope' ? 'bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-200' : 'text-gray-500 hover:text-blue-600 dark:text-gray-300'}`}
            >
              当前空间
            </button>
          </div>
          <span className="rounded-full border border-gray-200 bg-gray-50 px-2.5 py-1 text-[11px] text-gray-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">
            {summaryText}
          </span>
          {adminChecking ? (
            <span className="rounded-full border border-gray-200 bg-gray-50 px-2.5 py-1 text-[11px] text-gray-500 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">
              检查登录
            </span>
          ) : adminAuthenticated ? (
            <span
              className="rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-1 text-[11px] text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-200"
              title={`已登录 ${adminSession?.username ?? 'admin'}`}
            >
              已登录
            </span>
          ) : null}
          {!adminConfigured && (
            <span className="rounded-full border border-amber-200 bg-amber-50 px-2.5 py-1 text-[11px] text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-200">
              控制台未配置
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

      <div className="mt-3 rounded-lg border border-gray-200 bg-gray-50/70 p-3 dark:border-white/[0.08] dark:bg-white/[0.03]">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div className="grid min-w-0 flex-1 gap-2 sm:grid-cols-3">
            <label className="min-w-0 text-[11px] text-gray-500 dark:text-gray-400">
              <span className="mb-1 block uppercase">工作区</span>
              <select
                value={scope.workspaceId}
                onChange={(event) => void handleWorkspaceChange(event.target.value)}
                disabled={scopeLoading || workspaces.length === 0}
                className="h-9 w-full min-w-0 rounded-lg border border-gray-200 bg-white px-2.5 text-xs text-gray-700 outline-none transition focus:border-blue-300 disabled:cursor-not-allowed disabled:opacity-60 dark:border-white/[0.08] dark:bg-gray-950/50 dark:text-gray-100 dark:focus:border-blue-500/60"
              >
                {workspaces.length === 0 ? (
                  <option value={scope.workspaceId}>{scope.workspaceId || '暂无工作区'}</option>
                ) : workspaces.map((workspace) => (
                  <option key={workspace.workspace_id} value={workspace.workspace_id}>
                    {displayScopeName(workspace, workspace.workspace_id)}{workspace.archived ? '（已归档）' : ''}
                  </option>
                ))}
              </select>
            </label>
            <label className="min-w-0 text-[11px] text-gray-500 dark:text-gray-400">
              <span className="mb-1 block uppercase">项目</span>
              <select
                value={scope.projectId}
                onChange={(event) => void handleProjectChange(event.target.value)}
                disabled={scopeLoading || projects.length === 0}
                className="h-9 w-full min-w-0 rounded-lg border border-gray-200 bg-white px-2.5 text-xs text-gray-700 outline-none transition focus:border-blue-300 disabled:cursor-not-allowed disabled:opacity-60 dark:border-white/[0.08] dark:bg-gray-950/50 dark:text-gray-100 dark:focus:border-blue-500/60"
              >
                {projects.length === 0 ? (
                  <option value={scope.projectId}>{scope.projectId || '暂无项目'}</option>
                ) : projects.map((project) => (
                  <option key={project.project_id} value={project.project_id}>
                    {displayScopeName(project, project.project_id)}{project.archived ? '（已归档）' : ''}
                  </option>
                ))}
              </select>
            </label>
            <label className="min-w-0 text-[11px] text-gray-500 dark:text-gray-400">
              <span className="mb-1 block uppercase">批次</span>
              <select
                value={scope.campaignId}
                onChange={(event) => handleCampaignChange(event.target.value)}
                disabled={scopeLoading || campaigns.length === 0}
                className="h-9 w-full min-w-0 rounded-lg border border-gray-200 bg-white px-2.5 text-xs text-gray-700 outline-none transition focus:border-blue-300 disabled:cursor-not-allowed disabled:opacity-60 dark:border-white/[0.08] dark:bg-gray-950/50 dark:text-gray-100 dark:focus:border-blue-500/60"
              >
                {campaigns.length === 0 ? (
                  <option value={scope.campaignId}>{scope.campaignId || '暂无批次'}</option>
                ) : campaigns.map((campaign) => (
                  <option key={campaign.campaign_id} value={campaign.campaign_id}>
                    {displayScopeName(campaign, campaign.campaign_id)}{campaign.archived ? '（已归档）' : ''}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={() => setShowScopeManager(true)}
              className="h-9 rounded-lg border border-gray-200 bg-white px-3 text-xs font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
            >
              业务空间
            </button>
            <button
              type="button"
              onClick={() => setShowProjectContext(true)}
              disabled={!scope.workspaceId || !scope.projectId}
              className="h-9 rounded-lg border border-blue-200 bg-blue-50 px-3 text-xs font-medium text-blue-700 transition hover:bg-blue-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-200"
            >
              视觉上下文
            </button>
          </div>
        </div>
        <div className="mt-2 flex flex-wrap gap-2 text-[11px] text-gray-500 dark:text-gray-400">
          <span className="rounded-full border border-gray-200 bg-white px-2 py-0.5 dark:border-white/[0.08] dark:bg-white/[0.04]">
            {scopeLoading ? '正在加载业务空间' : `${displayScopeName(currentWorkspace, scope.workspaceId)} / ${displayScopeName(currentProject, scope.projectId)} / ${displayScopeName(currentCampaign, scope.campaignId)}`}
          </span>
          <span className="rounded-full border border-gray-200 bg-white px-2 py-0.5 dark:border-white/[0.08] dark:bg-white/[0.04]">
            provider：{providerSummary}
          </span>
          {runtimeStatus && (
            <span className="rounded-full border border-gray-200 bg-white px-2 py-0.5 dark:border-white/[0.08] dark:bg-white/[0.04]">
              Admin {runtimeStatus.admin_configured ? '已配置' : '未启用'} · Basic Auth {runtimeStatus.basic_auth_configured ? '已开启' : '未开启'}
            </span>
          )}
        </div>
        {scopeError && (
          <div className="mt-2 rounded-lg border border-amber-200 bg-amber-50 px-2.5 py-2 text-[11px] text-amber-800 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-100">
            {scopeError}
          </div>
        )}
      </div>

      <div className="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-6">
        <label className="min-w-0 text-[11px] text-gray-500 dark:text-gray-400">
          <span className="mb-1 block uppercase">状态</span>
          <select
            value={draftFilters.status}
            onChange={(event) => setStatusFilter(event.target.value)}
            className="h-9 w-full min-w-0 rounded-lg border border-gray-200 bg-white px-2.5 text-xs text-gray-700 outline-none transition focus:border-blue-300 dark:border-white/[0.08] dark:bg-gray-950/50 dark:text-gray-100 dark:focus:border-blue-500/60"
          >
            <option value="">全部</option>
            <option value="generated">已生成</option>
            <option value="selected">已选中</option>
            <option value="rejected">已拒绝</option>
            <option value="published">已发布</option>
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
          {hostMismatchWarning && (
            <div className="mb-3 rounded-lg border border-amber-300 bg-white/70 px-2.5 py-2 text-[11px] leading-relaxed text-amber-800 dark:border-amber-500/30 dark:bg-gray-950/30 dark:text-amber-100">
              {hostMismatchWarning}
            </div>
          )}
          {!adminConfigured ? (
            <div className="text-xs text-amber-800 dark:text-amber-100">
              控制台 Admin 登录未配置。请在服务端设置 ADMIN_USERNAME / ADMIN_PASSWORD，或复用 BASIC_AUTH_USERNAME / BASIC_AUTH_PASSWORD。
            </div>
          ) : (
            <div className="text-xs leading-relaxed text-amber-800 dark:text-amber-100">
              控制台登录状态已失效。请退出后重新登录，或刷新页面回到登录页。
            </div>
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
                baseUrl={baseUrl}
                busy={actionAssetId === asset.asset_id}
                onSelectAsset={selectAsset}
                onRejectAsset={rejectAsset}
                onMarkAsReference={markAsReference}
                onOpenProductionView={openProductionViewFromAsset}
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
