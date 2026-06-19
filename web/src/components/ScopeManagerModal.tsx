import { createPortal } from 'react-dom'
import { type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useStore } from '../store'
import { useCloseOnEscape } from '../hooks/useCloseOnEscape'
import { usePreventBackgroundScroll } from '../hooks/usePreventBackgroundScroll'
import { normalizeSettings } from '../lib/apiProfiles'
import {
  deleteAgentImageflowCampaign,
  deleteAgentImageflowProject,
  deleteAgentImageflowWorkspace,
  listAgentImageflowCampaigns,
  listAgentImageflowAssets,
  listAgentImageflowProjects,
  getAgentImageflowStorageGovernance,
  getAgentImageflowStorageIntegrity,
  listAgentImageflowWorkspaces,
  normalizeAgentImageflowApiBaseUrl,
  updateAgentImageflowCampaign,
  updateAgentImageflowProject,
  updateAgentImageflowWorkspace,
  type AgentImageflowAuth,
  type AgentImageflowAssetResponse,
  type AgentImageflowCampaign,
  type AgentImageflowProject,
  type AgentImageflowStorageGovernanceCountSnapshot,
  type AgentImageflowStorageIntegrityResponse,
  type AgentImageflowStorageUsageSnapshot,
  type AgentImageflowWorkspace,
} from '../lib/agentImageflowApi'
import type { AppSettings } from '../types'
import { ArchiveIcon, CloseIcon, CollectionManageIcon, EditIcon, RefreshIcon, TrashIcon } from './icons'

type ScopeSettings = Pick<
  AppSettings,
  'imageflowApiBaseUrl' |
  'imageflowApiKey' |
  'imageflowBasicUsername' |
  'imageflowBasicPassword' |
  'imageflowWorkspaceId' |
  'imageflowProjectId' |
  'imageflowCampaignId' |
  'imageflowManagedMode'
>

function buildManagedScopeAuth(settings: ScopeSettings): AgentImageflowAuth {
  return {
    apiKey: settings.imageflowApiKey,
    basicUsername: settings.imageflowBasicUsername,
    basicPassword: settings.imageflowBasicPassword,
  }
}

function pickPreferredId<T extends { archived?: boolean }>(
  items: T[],
  preferredId: string,
  getId: (item: T) => string,
): string {
  const trimmed = preferredId.trim()
  if (trimmed && items.some((item) => getId(item) === trimmed)) return trimmed
  const firstActive = items.find((item) => !item.archived)
  if (firstActive) return getId(firstActive)
  return items[0] ? getId(items[0]) : ''
}

function sortScopesByArchived<T extends { archived?: boolean }>(items: T[]): T[] {
  const active = items.filter((item) => !item.archived)
  const archived = items.filter((item) => item.archived)
  return [...active, ...archived]
}

type ScopeStats = {
  projectCount: number
  campaignCount: number
  assetCount: number
  selectedCount: number
  rejectedCount: number
  failedCount: number
  taskCount: number
  failedTaskCount: number
  integrityIssueCount: number
  integrityErrorCount: number
  integrityWarningCount: number
  storageBytes: number
  storageFileCount: number
  originalBytes: number
  thumbnailBytes: number
  metadataBytes: number
  inputFilesBytes: number
  auditBytes: number
  tmpBytes: number
  orphanBytes: number
  latestActivity: string
}

type ScopeDashboardStats = {
  workspaces: Record<string, ScopeStats>
  projects: Record<string, ScopeStats>
  campaigns: Record<string, ScopeStats>
}

const EMPTY_SCOPE_STATS: ScopeStats = {
  projectCount: 0,
  campaignCount: 0,
  assetCount: 0,
  selectedCount: 0,
  rejectedCount: 0,
  failedCount: 0,
  taskCount: 0,
  failedTaskCount: 0,
  integrityIssueCount: 0,
  integrityErrorCount: 0,
  integrityWarningCount: 0,
  storageBytes: 0,
  storageFileCount: 0,
  originalBytes: 0,
  thumbnailBytes: 0,
  metadataBytes: 0,
  inputFilesBytes: 0,
  auditBytes: 0,
  tmpBytes: 0,
  orphanBytes: 0,
  latestActivity: '',
}

const EMPTY_DASHBOARD_STATS: ScopeDashboardStats = {
  workspaces: {},
  projects: {},
  campaigns: {},
}

function cloneStats(stats: ScopeStats): ScopeStats {
  return { ...stats }
}

function getCampaignKey(projectID: string, campaignID: string): string {
  return `${projectID}/${campaignID}`
}

function addAssetsToStats(stats: ScopeStats, assets: AgentImageflowAssetResponse[]) {
  stats.assetCount += assets.length
  for (const asset of assets) {
    if (asset.status === 'selected') stats.selectedCount += 1
    if (asset.status === 'rejected') stats.rejectedCount += 1
    if (asset.status === 'failed') stats.failedCount += 1
    if (asset.created_at && (!stats.latestActivity || asset.created_at > stats.latestActivity)) {
      stats.latestActivity = asset.created_at
    }
  }
}

function addStorageToStats(
  stats: ScopeStats,
  usage: AgentImageflowStorageUsageSnapshot,
  counts: AgentImageflowStorageGovernanceCountSnapshot,
) {
  stats.taskCount += counts.task_count
  stats.failedTaskCount += counts.failed_task_count
  stats.storageBytes += usage.bytes
  stats.storageFileCount += usage.file_count
  for (const category of usage.categories) {
    if (category.category === 'original') stats.originalBytes += category.bytes
    if (category.category === 'thumbnail') stats.thumbnailBytes += category.bytes
    if (category.category === 'metadata') stats.metadataBytes += category.bytes
    if (category.category === 'input_files') stats.inputFilesBytes += category.bytes
    if (category.category === 'audit') stats.auditBytes += category.bytes
    if (category.category === 'tmp') stats.tmpBytes += category.bytes
    if (category.category === 'orphan') stats.orphanBytes += category.bytes
  }
}

function addIntegrityToStats(stats: ScopeStats, integrity: AgentImageflowStorageIntegrityResponse) {
  stats.integrityIssueCount += integrity.summary.issue_count
  for (const issue of integrity.issues) {
    if (issue.severity === 'error') stats.integrityErrorCount += 1
    if (issue.severity === 'warning') stats.integrityWarningCount += 1
  }
}

function formatLatestActivity(value: string): string {
  if (!value) return '无活动'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = value
  let unitIndex = 0
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex += 1
  }
  const precision = unitIndex === 0 || size >= 10 ? 0 : 1
  return `${size.toFixed(precision)} ${units[unitIndex]}`
}

function renderStorageBreakdown(stats: ScopeStats): string {
  const parts = [
    stats.originalBytes ? `original ${formatBytes(stats.originalBytes)}` : '',
    stats.thumbnailBytes ? `thumbnail ${formatBytes(stats.thumbnailBytes)}` : '',
    stats.metadataBytes ? `metadata ${formatBytes(stats.metadataBytes)}` : '',
    stats.inputFilesBytes ? `input ${formatBytes(stats.inputFilesBytes)}` : '',
    stats.auditBytes ? `audit ${formatBytes(stats.auditBytes)}` : '',
    stats.tmpBytes ? `tmp ${formatBytes(stats.tmpBytes)}` : '',
    stats.orphanBytes ? `orphan ${formatBytes(stats.orphanBytes)}` : '',
  ].filter(Boolean)
  return parts.length > 0 ? parts.join(' · ') : 'storage 0 B'
}

function renderStatsLine(stats?: ScopeStats): string {
  if (!stats) return '统计待同步'
  const parts = [
    stats.projectCount ? `${stats.projectCount} projects` : '',
    stats.campaignCount ? `${stats.campaignCount} campaigns` : '',
    `${stats.assetCount} assets`,
    `${stats.taskCount} tasks`,
    `${stats.selectedCount} selected`,
    `${stats.rejectedCount} rejected`,
    stats.failedTaskCount || stats.failedCount ? `${stats.failedTaskCount + stats.failedCount} failed` : '',
    stats.integrityIssueCount ? `${stats.integrityIssueCount} integrity` : '',
    `storage ${formatBytes(stats.storageBytes)}`,
  ].filter(Boolean)
  return `${parts.join(' · ')} · ${renderStorageBreakdown(stats)} · ${formatLatestActivity(stats.latestActivity)}`
}

function ScopeBadge({ children, tone = 'default' }: { children: string; tone?: 'default' | 'info' | 'warning' }) {
  const className = tone === 'warning'
    ? 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-200'
    : tone === 'info'
      ? 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-200'
      : 'border-gray-200 bg-gray-50 text-gray-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300'
  return (
    <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[10px] font-medium ${className}`}>
      {children}
    </span>
  )
}

type ScopeListSectionProps<T> = {
  title: string
  description: string
  items: T[]
  selectedId: string
  getId: (item: T) => string
  getLabel: (item: T) => string
  getSubLabel?: (item: T) => string | undefined
  getMeta?: (item: T) => ReactNode
  onSelect: (id: string) => void
  emptyText: string
  disabled?: boolean
  currentId?: string
}

function ScopeListSection<T extends { archived?: boolean }>({
  title,
  description,
  items,
  selectedId,
  getId,
  getLabel,
  getSubLabel,
  getMeta,
  onSelect,
  emptyText,
  disabled,
  currentId,
}: ScopeListSectionProps<T>) {
  return (
    <section className="flex min-h-[240px] flex-col rounded-2xl border border-gray-200/80 bg-white/80 dark:border-white/[0.08] dark:bg-white/[0.03]">
      <div className="border-b border-gray-200/70 px-4 py-3 dark:border-white/[0.08]">
        <div className="text-sm font-semibold text-gray-800 dark:text-gray-100">{title}</div>
        <div className="mt-1 text-[11px] leading-relaxed text-gray-500 dark:text-gray-400">{description}</div>
      </div>
      <div className="flex-1 overflow-y-auto px-2 py-2">
        {disabled ? (
          <div className="flex h-full items-center justify-center px-3 text-center text-xs text-gray-400 dark:text-gray-500">
            请先选择上一级 scope
          </div>
        ) : items.length === 0 ? (
          <div className="flex h-full items-center justify-center px-3 text-center text-xs text-gray-400 dark:text-gray-500">
            {emptyText}
          </div>
        ) : (
          <div className="space-y-1">
            {items.map((item) => {
              const id = getId(item)
              const selected = id === selectedId
              const current = currentId?.trim() === id
              const subLabel = getSubLabel?.(item)
              const meta = getMeta?.(item)
              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => onSelect(id)}
                  className={`flex w-full items-start justify-between gap-3 rounded-xl px-3 py-2 text-left transition ${
                    selected
                      ? 'bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-100'
                      : 'text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-white/[0.04]'
                  } ${item.archived ? 'opacity-70' : ''}`}
                >
                  <div className="min-w-0">
                    <div className="truncate text-sm font-medium">{getLabel(item) || id}</div>
                    <div className="mt-0.5 truncate text-[11px] text-gray-500 dark:text-gray-400">{subLabel || id}</div>
                    {meta && <div className="mt-1 text-[10px] leading-relaxed text-gray-400 dark:text-gray-500">{meta}</div>}
                  </div>
                  <div className="flex shrink-0 flex-col items-end gap-1 pt-0.5">
                    {current && <ScopeBadge tone="info">当前</ScopeBadge>}
                    {item.archived && <ScopeBadge tone="warning">已归档</ScopeBadge>}
                  </div>
                </button>
              )
            })}
          </div>
        )}
      </div>
    </section>
  )
}

type ScopeActionPanelProps = {
  kindLabel: string
  selectedName: string
  archived?: boolean
  current?: boolean
  nameDraft: string
  setNameDraft: (value: string) => void
  onSaveName: () => void
  onToggleArchive: () => void
  onDelete: () => void
  onApplyCurrent?: () => void
  busy?: boolean
  disabled?: boolean
}

function ScopeActionPanel({
  kindLabel,
  selectedName,
  archived,
  current,
  nameDraft,
  setNameDraft,
  onSaveName,
  onToggleArchive,
  onDelete,
  onApplyCurrent,
  busy,
  disabled,
}: ScopeActionPanelProps) {
  return (
    <div className="border-t border-gray-200/70 px-4 py-3 dark:border-white/[0.08]">
      {disabled ? (
        <div className="text-[11px] text-gray-400 dark:text-gray-500">请选择一个 {kindLabel.toLowerCase()} 后再操作。</div>
      ) : (
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <div className="min-w-0 flex-1 truncate text-xs font-medium text-gray-700 dark:text-gray-200">
              {selectedName || `未选择 ${kindLabel}`}
            </div>
            {current && <ScopeBadge tone="info">当前托管 scope</ScopeBadge>}
            {archived && <ScopeBadge tone="warning">已归档</ScopeBadge>}
          </div>
          <div className="flex items-center gap-2">
            <input
              value={nameDraft}
              onChange={(event) => setNameDraft(event.target.value)}
              placeholder={`重命名 ${kindLabel}`}
              disabled={busy}
              className="min-w-0 flex-1 rounded-lg border border-gray-200/70 bg-white px-2.5 py-2 text-xs text-gray-700 outline-none transition focus:border-blue-300 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-200 dark:focus:border-blue-500/50"
            />
            <button
              type="button"
              onClick={onSaveName}
              disabled={busy}
              className="inline-flex h-8 items-center gap-1.5 rounded-lg border border-gray-200/70 bg-white px-2.5 text-[11px] font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300 dark:hover:border-blue-500/50 dark:hover:text-blue-400"
            >
              <EditIcon className="h-3.5 w-3.5" />
              保存
            </button>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {onApplyCurrent && (
              <button
                type="button"
                onClick={onApplyCurrent}
                disabled={busy || archived}
                className="inline-flex h-8 items-center rounded-lg border border-blue-200 bg-blue-50 px-2.5 text-[11px] font-medium text-blue-700 transition hover:border-blue-300 hover:bg-blue-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-200 dark:hover:border-blue-500/40 dark:hover:bg-blue-500/15"
              >
                设为当前 scope
              </button>
            )}
            <button
              type="button"
              onClick={onToggleArchive}
              disabled={busy}
              className="inline-flex h-8 items-center gap-1.5 rounded-lg border border-gray-200/70 bg-white px-2.5 text-[11px] font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300 dark:hover:border-blue-500/50 dark:hover:text-blue-400"
            >
              <ArchiveIcon className="h-3.5 w-3.5" />
              {archived ? '恢复' : '归档'}
            </button>
            <button
              type="button"
              onClick={onDelete}
              disabled={busy}
              className="inline-flex h-8 items-center gap-1.5 rounded-lg border border-red-200 bg-red-50 px-2.5 text-[11px] font-medium text-red-700 transition hover:border-red-300 hover:bg-red-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200 dark:hover:border-red-500/40 dark:hover:bg-red-500/15"
            >
              <TrashIcon className="h-3.5 w-3.5" />
              删除
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

export default function ScopeManagerModal() {
  const open = useStore((state) => state.showScopeManager)
  const setShowScopeManager = useStore((state) => state.setShowScopeManager)
  const settings = useStore((state) => state.settings)
  const setSettings = useStore((state) => state.setSettings)
  const setShowSettings = useStore((state) => state.setShowSettings)
  const showToast = useStore((state) => state.showToast)
  const setConfirmDialog = useStore((state) => state.setConfirmDialog)
  const modalRef = useRef<HTMLDivElement>(null)
  const requestRef = useRef(0)

  const [loading, setLoading] = useState(false)
  const [busyAction, setBusyAction] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [workspaces, setWorkspaces] = useState<AgentImageflowWorkspace[]>([])
  const [projects, setProjects] = useState<AgentImageflowProject[]>([])
  const [campaigns, setCampaigns] = useState<AgentImageflowCampaign[]>([])
  const [dashboardStats, setDashboardStats] = useState<ScopeDashboardStats>(EMPTY_DASHBOARD_STATS)
  const [dashboardLoading, setDashboardLoading] = useState(false)
  const [dashboardError, setDashboardError] = useState<string | null>(null)
  const [selectedWorkspaceId, setSelectedWorkspaceId] = useState('')
  const [selectedProjectId, setSelectedProjectId] = useState('')
  const [selectedCampaignId, setSelectedCampaignId] = useState('')
  const [workspaceNameDraft, setWorkspaceNameDraft] = useState('')
  const [projectNameDraft, setProjectNameDraft] = useState('')
  const [campaignNameDraft, setCampaignNameDraft] = useState('')

  const normalizedSettings = useMemo(() => normalizeSettings(settings), [settings])
  const baseUrl = useMemo(
    () => normalizeAgentImageflowApiBaseUrl(normalizedSettings.imageflowApiBaseUrl),
    [normalizedSettings.imageflowApiBaseUrl],
  )
  const auth = useMemo(
    () => buildManagedScopeAuth(normalizedSettings),
    [
      normalizedSettings.imageflowApiKey,
      normalizedSettings.imageflowBasicPassword,
      normalizedSettings.imageflowBasicUsername,
    ],
  )

  const close = useCallback(() => setShowScopeManager(false), [setShowScopeManager])

  useCloseOnEscape(open, close)
  usePreventBackgroundScroll(open, modalRef)

  const selectedWorkspace = workspaces.find((workspace) => workspace.workspace_id === selectedWorkspaceId) ?? null
  const selectedProject = projects.find((project) => project.project_id === selectedProjectId) ?? null
  const selectedCampaign = campaigns.find((campaign) => campaign.campaign_id === selectedCampaignId) ?? null

  const loadDashboardStats = useCallback(async (workspaceItems: AgentImageflowWorkspace[], requestId: number) => {
    setDashboardLoading(true)
    setDashboardError(null)
    try {
      const nextStats: ScopeDashboardStats = {
        workspaces: {},
        projects: {},
        campaigns: {},
      }
      let partialStatsFailed = false

      for (const workspace of workspaceItems) {
        const workspaceStats = cloneStats(EMPTY_SCOPE_STATS)
        let projectResponse: Awaited<ReturnType<typeof listAgentImageflowProjects>>
        try {
          projectResponse = await listAgentImageflowProjects(baseUrl, workspace.workspace_id, auth)
        } catch {
          partialStatsFailed = true
          nextStats.workspaces[workspace.workspace_id] = workspaceStats
          continue
        }
        if (requestRef.current !== requestId) return

        workspaceStats.projectCount = projectResponse.projects.length
        for (const project of projectResponse.projects) {
          const projectStats = cloneStats(EMPTY_SCOPE_STATS)
          let campaignResponse: Awaited<ReturnType<typeof listAgentImageflowCampaigns>>
          try {
            campaignResponse = await listAgentImageflowCampaigns(baseUrl, {
              workspaceId: workspace.workspace_id,
              projectId: project.project_id,
            }, auth)
          } catch {
            partialStatsFailed = true
            nextStats.projects[project.project_id] = projectStats
            continue
          }
          if (requestRef.current !== requestId) return

          projectStats.campaignCount = campaignResponse.campaigns.length
          workspaceStats.campaignCount += campaignResponse.campaigns.length

          for (const campaign of campaignResponse.campaigns) {
            const campaignStats = cloneStats(EMPTY_SCOPE_STATS)
            let assets: AgentImageflowAssetResponse[] = []
            try {
              assets = await listAgentImageflowAssets(baseUrl, {
                projectId: project.project_id,
                campaignId: campaign.campaign_id,
              }, auth)
            } catch {
              partialStatsFailed = true
            }
            if (requestRef.current !== requestId) return

            addAssetsToStats(campaignStats, assets)
            addAssetsToStats(projectStats, assets)
            addAssetsToStats(workspaceStats, assets)
            try {
              const governance = await getAgentImageflowStorageGovernance(baseUrl, {
                workspaceId: workspace.workspace_id,
                projectId: project.project_id,
                campaignId: campaign.campaign_id,
              }, auth)
              addStorageToStats(campaignStats, governance.usage.campaign, governance.counts.campaign)
              addStorageToStats(projectStats, governance.usage.campaign, governance.counts.campaign)
              addStorageToStats(workspaceStats, governance.usage.campaign, governance.counts.campaign)
            } catch {
              partialStatsFailed = true
            }
            try {
              const integrity = await getAgentImageflowStorageIntegrity(baseUrl, {
                workspaceId: workspace.workspace_id,
                projectId: project.project_id,
                campaignId: campaign.campaign_id,
              }, auth)
              addIntegrityToStats(campaignStats, integrity)
              addIntegrityToStats(projectStats, integrity)
              addIntegrityToStats(workspaceStats, integrity)
            } catch {
              partialStatsFailed = true
            }
            if (requestRef.current !== requestId) return
            nextStats.campaigns[getCampaignKey(project.project_id, campaign.campaign_id)] = campaignStats
          }

          nextStats.projects[project.project_id] = projectStats
        }

        nextStats.workspaces[workspace.workspace_id] = workspaceStats
      }

      if (requestRef.current !== requestId) return
      setDashboardStats(nextStats)
      setDashboardError(partialStatsFailed ? '部分 project/campaign 需要 API key，已跳过对应统计。' : null)
    } catch (nextError) {
      if (requestRef.current !== requestId) return
      setDashboardError(nextError instanceof Error ? nextError.message : String(nextError))
      setDashboardStats(EMPTY_DASHBOARD_STATS)
    } finally {
      if (requestRef.current === requestId) {
        setDashboardLoading(false)
      }
    }
  }, [auth, baseUrl])

  useEffect(() => {
    setWorkspaceNameDraft(selectedWorkspace?.name ?? '')
  }, [selectedWorkspace?.name, selectedWorkspace?.workspace_id])

  useEffect(() => {
    setProjectNameDraft(selectedProject?.name ?? '')
  }, [selectedProject?.name, selectedProject?.project_id])

  useEffect(() => {
    setCampaignNameDraft(selectedCampaign?.name ?? '')
  }, [selectedCampaign?.name, selectedCampaign?.campaign_id])

  const reloadHierarchy = useCallback(async (
    preferredWorkspaceId?: string,
    preferredProjectId?: string,
    preferredCampaignId?: string,
  ) => {
    const requestId = ++requestRef.current
    setLoading(true)
    setError(null)

    try {
      const requestedWorkspaceId = preferredWorkspaceId ?? normalizedSettings.imageflowWorkspaceId
      const requestedProjectId = preferredProjectId ?? normalizedSettings.imageflowProjectId
      const requestedCampaignId = preferredCampaignId ?? normalizedSettings.imageflowCampaignId

      const workspaceResponse = await listAgentImageflowWorkspaces(baseUrl, auth)
      if (requestRef.current !== requestId) return

      const nextWorkspaces = sortScopesByArchived(workspaceResponse.workspaces)
      setWorkspaces(nextWorkspaces)
      void loadDashboardStats(nextWorkspaces, requestId)

      const nextWorkspaceId = pickPreferredId(nextWorkspaces, requestedWorkspaceId, (item) => item.workspace_id)
      setSelectedWorkspaceId(nextWorkspaceId)

      if (!nextWorkspaceId) {
        setProjects([])
        setCampaigns([])
        setSelectedProjectId('')
        setSelectedCampaignId('')
        return
      }

      const projectResponse = await listAgentImageflowProjects(baseUrl, nextWorkspaceId, auth)
      if (requestRef.current !== requestId) return

      const nextProjects = sortScopesByArchived(projectResponse.projects)
      setProjects(nextProjects)

      const nextProjectId = pickPreferredId(nextProjects, requestedProjectId, (item) => item.project_id)
      setSelectedProjectId(nextProjectId)

      if (!nextProjectId) {
        setCampaigns([])
        setSelectedCampaignId('')
        return
      }

      const campaignResponse = await listAgentImageflowCampaigns(baseUrl, {
        workspaceId: nextWorkspaceId,
        projectId: nextProjectId,
      }, auth)
      if (requestRef.current !== requestId) return

      const nextCampaigns = sortScopesByArchived(campaignResponse.campaigns)
      setCampaigns(nextCampaigns)
      setSelectedCampaignId(pickPreferredId(nextCampaigns, requestedCampaignId, (item) => item.campaign_id))
    } catch (nextError) {
      if (requestRef.current !== requestId) return
      setError(nextError instanceof Error ? nextError.message : String(nextError))
      setWorkspaces([])
      setProjects([])
      setCampaigns([])
      setDashboardStats(EMPTY_DASHBOARD_STATS)
      setSelectedWorkspaceId('')
      setSelectedProjectId('')
      setSelectedCampaignId('')
    } finally {
      if (requestRef.current === requestId) {
        setLoading(false)
      }
    }
  }, [
    auth,
    baseUrl,
    loadDashboardStats,
    normalizedSettings.imageflowCampaignId,
    normalizedSettings.imageflowProjectId,
    normalizedSettings.imageflowWorkspaceId,
  ])

  useEffect(() => {
    if (!open) return
    void reloadHierarchy(
      normalizedSettings.imageflowWorkspaceId,
      normalizedSettings.imageflowProjectId,
      normalizedSettings.imageflowCampaignId,
    )
  }, [
    normalizedSettings.imageflowApiBaseUrl,
    normalizedSettings.imageflowApiKey,
    normalizedSettings.imageflowBasicPassword,
    normalizedSettings.imageflowBasicUsername,
    normalizedSettings.imageflowCampaignId,
    normalizedSettings.imageflowProjectId,
    normalizedSettings.imageflowWorkspaceId,
    open,
    reloadHierarchy,
  ])

  const openSettings = () => {
    setShowScopeManager(false)
    setShowSettings(true, 'general')
  }

  const setCurrentManagedScope = () => {
    if (!selectedWorkspace || !selectedProject || !selectedCampaign) {
      showToast('请先选择完整的 workspace / project / campaign', 'error')
      return
    }
    if (selectedWorkspace.archived || selectedProject.archived || selectedCampaign.archived) {
      showToast('归档中的 scope 不能设为当前托管 scope', 'error')
      return
    }
    setSettings(normalizeSettings({
      ...settings,
      imageflowWorkspaceId: selectedWorkspace.workspace_id,
      imageflowProjectId: selectedProject.project_id,
      imageflowCampaignId: selectedCampaign.campaign_id,
    }))
    showToast('已设为当前托管 scope', 'success')
  }

  const runAction = async (actionKey: string, action: () => Promise<void>) => {
    setBusyAction(actionKey)
    try {
      await action()
    } catch (nextError) {
      showToast(nextError instanceof Error ? nextError.message : String(nextError), 'error')
    } finally {
      setBusyAction(null)
    }
  }

  const saveWorkspaceName = () => {
    if (!selectedWorkspace) return
    const nextName = workspaceNameDraft.trim()
    if (!nextName) {
      showToast('请输入 workspace 名称', 'error')
      return
    }
    if (nextName === selectedWorkspace.name) return
    void runAction('workspace-rename', async () => {
      await updateAgentImageflowWorkspace(baseUrl, selectedWorkspace.workspace_id, { name: nextName }, auth)
      showToast(`已更新 workspace「${nextName}」`, 'success')
      await reloadHierarchy(selectedWorkspace.workspace_id, selectedProjectId, selectedCampaignId)
    })
  }

  const saveProjectName = () => {
    if (!selectedWorkspace || !selectedProject) return
    const nextName = projectNameDraft.trim()
    if (!nextName) {
      showToast('请输入 project 名称', 'error')
      return
    }
    if (nextName === selectedProject.name) return
    void runAction('project-rename', async () => {
      await updateAgentImageflowProject(baseUrl, {
        workspaceId: selectedWorkspace.workspace_id,
        projectId: selectedProject.project_id,
      }, { name: nextName }, auth)
      showToast(`已更新 project「${nextName}」`, 'success')
      await reloadHierarchy(selectedWorkspace.workspace_id, selectedProject.project_id, selectedCampaignId)
    })
  }

  const saveCampaignName = () => {
    if (!selectedWorkspace || !selectedProject || !selectedCampaign) return
    const nextName = campaignNameDraft.trim()
    if (!nextName) {
      showToast('请输入 campaign 名称', 'error')
      return
    }
    if (nextName === selectedCampaign.name) return
    void runAction('campaign-rename', async () => {
      await updateAgentImageflowCampaign(baseUrl, {
        workspaceId: selectedWorkspace.workspace_id,
        projectId: selectedProject.project_id,
        campaignId: selectedCampaign.campaign_id,
      }, { name: nextName }, auth)
      showToast(`已更新 campaign「${nextName}」`, 'success')
      await reloadHierarchy(selectedWorkspace.workspace_id, selectedProject.project_id, selectedCampaign.campaign_id)
    })
  }

  const toggleWorkspaceArchive = () => {
    if (!selectedWorkspace) return
    const nextArchived = !selectedWorkspace.archived
    void runAction('workspace-archive', async () => {
      await updateAgentImageflowWorkspace(baseUrl, selectedWorkspace.workspace_id, { archived: nextArchived }, auth)
      showToast(nextArchived ? 'workspace 已归档' : 'workspace 已恢复', 'success')
      await reloadHierarchy(selectedWorkspace.workspace_id, selectedProjectId, selectedCampaignId)
    })
  }

  const toggleProjectArchive = () => {
    if (!selectedWorkspace || !selectedProject) return
    const nextArchived = !selectedProject.archived
    void runAction('project-archive', async () => {
      await updateAgentImageflowProject(baseUrl, {
        workspaceId: selectedWorkspace.workspace_id,
        projectId: selectedProject.project_id,
      }, { archived: nextArchived }, auth)
      showToast(nextArchived ? 'project 已归档' : 'project 已恢复', 'success')
      await reloadHierarchy(selectedWorkspace.workspace_id, selectedProject.project_id, selectedCampaignId)
    })
  }

  const toggleCampaignArchive = () => {
    if (!selectedWorkspace || !selectedProject || !selectedCampaign) return
    const nextArchived = !selectedCampaign.archived
    void runAction('campaign-archive', async () => {
      await updateAgentImageflowCampaign(baseUrl, {
        workspaceId: selectedWorkspace.workspace_id,
        projectId: selectedProject.project_id,
        campaignId: selectedCampaign.campaign_id,
      }, { archived: nextArchived }, auth)
      showToast(nextArchived ? 'campaign 已归档' : 'campaign 已恢复', 'success')
      await reloadHierarchy(selectedWorkspace.workspace_id, selectedProject.project_id, selectedCampaign.campaign_id)
    })
  }

  const confirmDeleteWorkspace = () => {
    if (!selectedWorkspace) return
    setConfirmDialog({
      title: '删除 Workspace',
      message: `确定要删除 workspace「${selectedWorkspace.name || selectedWorkspace.workspace_id}」吗？\n\n只有当 workspace 下没有 project 时才能删除。`,
      tone: 'danger',
      confirmText: '删除',
      action: () => {
        void runAction('workspace-delete', async () => {
          await deleteAgentImageflowWorkspace(baseUrl, selectedWorkspace.workspace_id, auth)
          if (settings.imageflowWorkspaceId.trim() === selectedWorkspace.workspace_id) {
            setSettings(normalizeSettings({
              ...settings,
              imageflowWorkspaceId: '',
              imageflowProjectId: '',
              imageflowCampaignId: '',
            }))
          }
          showToast('workspace 已删除', 'success')
          await reloadHierarchy('', '', '')
        })
      },
    })
  }

  const confirmDeleteProject = () => {
    if (!selectedWorkspace || !selectedProject) return
    setConfirmDialog({
      title: '删除 Project',
      message: `确定要删除 project「${selectedProject.name || selectedProject.project_id}」吗？\n\n只有当 project 下没有 campaign 时才能删除。`,
      tone: 'danger',
      confirmText: '删除',
      action: () => {
        void runAction('project-delete', async () => {
          await deleteAgentImageflowProject(baseUrl, {
            workspaceId: selectedWorkspace.workspace_id,
            projectId: selectedProject.project_id,
          }, auth)
          if (
            settings.imageflowWorkspaceId.trim() === selectedWorkspace.workspace_id &&
            settings.imageflowProjectId.trim() === selectedProject.project_id
          ) {
            setSettings(normalizeSettings({
              ...settings,
              imageflowProjectId: '',
              imageflowCampaignId: '',
            }))
          }
          showToast('project 已删除', 'success')
          await reloadHierarchy(selectedWorkspace.workspace_id, '', '')
        })
      },
    })
  }

  const confirmDeleteCampaign = () => {
    if (!selectedWorkspace || !selectedProject || !selectedCampaign) return
    setConfirmDialog({
      title: '删除 Campaign',
      message: `确定要删除 campaign「${selectedCampaign.name || selectedCampaign.campaign_id}」吗？\n\n只有当 campaign 下没有任务或资产时才能删除。`,
      tone: 'danger',
      confirmText: '删除',
      action: () => {
        void runAction('campaign-delete', async () => {
          await deleteAgentImageflowCampaign(baseUrl, {
            workspaceId: selectedWorkspace.workspace_id,
            projectId: selectedProject.project_id,
            campaignId: selectedCampaign.campaign_id,
          }, auth)
          if (
            settings.imageflowWorkspaceId.trim() === selectedWorkspace.workspace_id &&
            settings.imageflowProjectId.trim() === selectedProject.project_id &&
            settings.imageflowCampaignId.trim() === selectedCampaign.campaign_id
          ) {
            setSettings(normalizeSettings({
              ...settings,
              imageflowCampaignId: '',
            }))
          }
          showToast('campaign 已删除', 'success')
          await reloadHierarchy(selectedWorkspace.workspace_id, selectedProject.project_id, '')
        })
      },
    })
  }

  if (!open) return null

  const currentScopeSummary = normalizedSettings.imageflowWorkspaceId && normalizedSettings.imageflowProjectId && normalizedSettings.imageflowCampaignId
    ? `${normalizedSettings.imageflowWorkspaceId} / ${normalizedSettings.imageflowProjectId} / ${normalizedSettings.imageflowCampaignId}`
    : '尚未设置当前托管 scope'

  return createPortal(
    <div data-no-drag-select className="fixed inset-0 z-[110] flex items-center justify-center p-4" onClick={close}>
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm animate-overlay-in" />
      <div
        ref={modalRef}
        className="relative z-10 flex h-[88vh] w-full max-w-6xl flex-col overflow-hidden rounded-3xl border border-white/50 bg-white/95 shadow-2xl ring-1 ring-black/5 animate-modal-in dark:border-white/[0.08] dark:bg-gray-900/95 dark:ring-white/10"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="border-b border-gray-200/70 px-6 py-5 dark:border-white/[0.08]">
          <button
            type="button"
            onClick={close}
            className="absolute right-5 top-5 rounded-full p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-white/[0.06] dark:hover:text-gray-200"
            aria-label="关闭"
          >
            <CloseIcon className="h-5 w-5" />
          </button>
          <div className="flex items-start gap-3 pr-10">
            <div className="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-300">
              <CollectionManageIcon className="h-5 w-5" />
            </div>
            <div className="min-w-0">
              <h2 className="text-lg font-semibold text-gray-800 dark:text-gray-100">Scope 管理</h2>
              <p className="mt-1 text-sm leading-relaxed text-gray-500 dark:text-gray-400">
                在这里查看并维护服务端的 workspace / project / campaign，支持 rename、归档、删除，以及把某个 campaign 设为当前托管 scope。
              </p>
            </div>
          </div>
        </div>

        <div className="border-b border-gray-200/70 px-6 py-3 dark:border-white/[0.08]">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div className="min-w-0">
              <div className="text-[11px] uppercase tracking-wide text-gray-400 dark:text-gray-500">Current Managed Scope</div>
              <div className="mt-1 truncate text-sm font-medium text-gray-700 dark:text-gray-200">{currentScopeSummary}</div>
              <div className="mt-1 text-[11px] text-gray-500 dark:text-gray-400">
                服务端地址：{baseUrl}
                {!normalizedSettings.imageflowManagedMode ? ' · 当前未开启托管模式' : ''}
                {dashboardLoading ? ' · 统计同步中' : ''}
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={openSettings}
                className="inline-flex h-9 items-center rounded-xl border border-gray-200/70 bg-white px-3 text-xs font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300 dark:hover:border-blue-500/50 dark:hover:text-blue-400"
              >
                打开设置
              </button>
              <button
                type="button"
                onClick={() => void reloadHierarchy(selectedWorkspaceId, selectedProjectId, selectedCampaignId)}
                disabled={loading}
                className="inline-flex h-9 items-center gap-1.5 rounded-xl border border-gray-200/70 bg-white px-3 text-xs font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300 dark:hover:border-blue-500/50 dark:hover:text-blue-400"
              >
                <RefreshIcon className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                刷新
              </button>
            </div>
          </div>
        </div>

        {error && (
          <div className="mx-6 mt-4 rounded-xl border border-red-200 bg-red-50/80 px-3 py-2 text-xs text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200">
            {error}
          </div>
        )}
        {dashboardError && (
          <div className="mx-6 mt-4 rounded-xl border border-amber-200 bg-amber-50/80 px-3 py-2 text-xs text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-200">
            控制台统计提示：{dashboardError}
          </div>
        )}

        <div className="flex-1 overflow-hidden px-6 py-4">
          <div className="grid h-full grid-cols-1 gap-4 xl:grid-cols-3">
            <div className="flex min-h-0 flex-col">
              <ScopeListSection
                title="Workspace"
                description="实例级业务空间。"
                items={workspaces}
                selectedId={selectedWorkspaceId}
                getId={(item) => item.workspace_id}
                getLabel={(item) => item.name || item.workspace_id}
                getMeta={(item) => renderStatsLine(dashboardStats.workspaces[item.workspace_id])}
                onSelect={(workspaceId) => void reloadHierarchy(workspaceId, '', '')}
                emptyText="暂无 workspace"
                currentId={normalizedSettings.imageflowWorkspaceId}
              />
              <ScopeActionPanel
                kindLabel="Workspace"
                selectedName={selectedWorkspace?.name || selectedWorkspace?.workspace_id || ''}
                archived={selectedWorkspace?.archived}
                current={normalizedSettings.imageflowWorkspaceId.trim() === selectedWorkspace?.workspace_id}
                nameDraft={workspaceNameDraft}
                setNameDraft={setWorkspaceNameDraft}
                onSaveName={saveWorkspaceName}
                onToggleArchive={toggleWorkspaceArchive}
                onDelete={confirmDeleteWorkspace}
                busy={Boolean(busyAction)}
                disabled={!selectedWorkspace}
              />
            </div>

            <div className="flex min-h-0 flex-col">
              <ScopeListSection
                title="Project"
                description="内容账号或业务项目。"
                items={projects}
                selectedId={selectedProjectId}
                getId={(item) => item.project_id}
                getLabel={(item) => item.name || item.project_id}
                getSubLabel={(item) => item.description}
                getMeta={(item) => renderStatsLine(dashboardStats.projects[item.project_id])}
                onSelect={(projectId) => void reloadHierarchy(selectedWorkspaceId, projectId, '')}
                emptyText="当前 workspace 下暂无 project"
                disabled={!selectedWorkspaceId}
                currentId={normalizedSettings.imageflowProjectId}
              />
              <ScopeActionPanel
                kindLabel="Project"
                selectedName={selectedProject?.name || selectedProject?.project_id || ''}
                archived={selectedProject?.archived}
                current={normalizedSettings.imageflowProjectId.trim() === selectedProject?.project_id}
                nameDraft={projectNameDraft}
                setNameDraft={setProjectNameDraft}
                onSaveName={saveProjectName}
                onToggleArchive={toggleProjectArchive}
                onDelete={confirmDeleteProject}
                busy={Boolean(busyAction)}
                disabled={!selectedProject}
              />
            </div>

            <div className="flex min-h-0 flex-col">
              <ScopeListSection
                title="Campaign"
                description="任务与资产归属的实际投放批次。"
                items={campaigns}
                selectedId={selectedCampaignId}
                getId={(item) => item.campaign_id}
                getLabel={(item) => item.name || item.campaign_id}
                getSubLabel={(item) => item.description}
                getMeta={(item) => renderStatsLine(dashboardStats.campaigns[getCampaignKey(item.project_id, item.campaign_id)])}
                onSelect={setSelectedCampaignId}
                emptyText="当前 project 下暂无 campaign"
                disabled={!selectedProjectId}
                currentId={normalizedSettings.imageflowCampaignId}
              />
              <ScopeActionPanel
                kindLabel="Campaign"
                selectedName={selectedCampaign?.name || selectedCampaign?.campaign_id || ''}
                archived={selectedCampaign?.archived}
                current={
                  normalizedSettings.imageflowWorkspaceId.trim() === selectedWorkspace?.workspace_id &&
                  normalizedSettings.imageflowProjectId.trim() === selectedProject?.project_id &&
                  normalizedSettings.imageflowCampaignId.trim() === selectedCampaign?.campaign_id
                }
                nameDraft={campaignNameDraft}
                setNameDraft={setCampaignNameDraft}
                onSaveName={saveCampaignName}
                onToggleArchive={toggleCampaignArchive}
                onDelete={confirmDeleteCampaign}
                onApplyCurrent={setCurrentManagedScope}
                busy={Boolean(busyAction)}
                disabled={!selectedCampaign}
              />
            </div>
          </div>
        </div>
      </div>
    </div>,
    document.body,
  )
}
