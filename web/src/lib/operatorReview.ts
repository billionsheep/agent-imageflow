import type { AgentImageflowAssetResponse } from './agentImageflowApi'

export interface OperatorReviewField {
  key: string
  label: string
  value: string
}

export interface OperatorReviewProductionFilters {
  sessionId: string
  batchId: string
  storyId: string
  source: string
  status: string
  includeSetup: boolean
  limit: string
}

const SENSITIVE_KEY_RE = /(?:api[_-]?key|provider[_-]?key|secret|password|authorization|cookie|token|bearer)/i
const LOCAL_PATH_KEY_RE = /^(?:local_path|file_path|thumbnail_path|metadata_path)$/i

function getStringValue(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

function getMetadataString(asset: AgentImageflowAssetResponse, key: string): string {
  return getStringValue(asset.metadata_json?.[key])
}

function collapseWhitespace(value: string): string {
  return value.replace(/\s+/g, ' ').trim()
}

function truncateText(value: string, maxLength: number): string {
  const text = collapseWhitespace(value)
  if (text.length <= maxLength) return text
  return `${text.slice(0, Math.max(0, maxLength - 3)).trimEnd()}...`
}

function isLocalhostAlias(hostname: string): hostname is 'localhost' | '127.0.0.1' {
  return hostname === 'localhost' || hostname === '127.0.0.1'
}

function parseUrl(value: string, base?: string): URL | null {
  try {
    return new URL(value, base)
  } catch {
    return null
  }
}

function isLocalAbsolutePath(value: string): boolean {
  const text = value.trim()
  if (!text) return false
  if (/^file:\/\//i.test(text)) return true
  if (/^[a-zA-Z]:[\\/]/.test(text)) return true
  return /^\/(?:Users|home|var|tmp|private|Volumes|data|app|storage|mnt|srv|opt|root|etc)(?:\/|$)/.test(text)
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function sanitizeReviewValue(value: unknown): unknown {
  if (typeof value === 'string') {
    return isLocalAbsolutePath(value) ? undefined : value
  }
  if (typeof value === 'number' || typeof value === 'boolean' || value == null) {
    return value
  }
  if (Array.isArray(value)) {
    const items = value
      .map((item) => sanitizeReviewValue(item))
      .filter((item) => item !== undefined)
    return items.length ? items : undefined
  }
  if (isRecord(value)) {
    const sanitizedRecord: Record<string, unknown> = {}
    for (const [key, item] of Object.entries(value)) {
      if (SENSITIVE_KEY_RE.test(key) || LOCAL_PATH_KEY_RE.test(key)) continue
      const sanitized = sanitizeReviewValue(item)
      if (sanitized !== undefined) sanitizedRecord[key] = sanitized
    }
    return Object.keys(sanitizedRecord).length ? sanitizedRecord : undefined
  }
  return undefined
}

function stringifySanitizedJSON(value?: Record<string, unknown>): string {
  if (!value || Object.keys(value).length === 0) return ''
  const sanitized = sanitizeReviewValue(value)
  if (!isRecord(sanitized) || Object.keys(sanitized).length === 0) return ''
  try {
    return JSON.stringify(sanitized, null, 2)
  } catch {
    return ''
  }
}

function pushField(fields: OperatorReviewField[], key: string, label: string, value?: string | number) {
  const text = value == null ? '' : String(value).trim()
  if (!text) return
  if (isLocalAbsolutePath(text)) return
  fields.push({ key, label, value: text })
}

export function getAssetReviewStatusLabel(status: string): string {
  const normalized = status.trim().toLowerCase()
  if (normalized === 'selected' || normalized === 'approved') return '已选'
  if (normalized === 'rejected') return '已拒绝'
  if (normalized === 'archived' || normalized === 'deprecated') return '已归档'
  if (normalized === 'published') return '已发布'
  if (normalized === 'failed') return '失败'
  return '待选'
}

export function getAssetReviewTitle(asset: AgentImageflowAssetResponse): string {
  const metadataSummary = [
    'scene_summary',
    'story_summary',
    'caption',
    'description',
  ].map((key) => getMetadataString(asset, key)).find(Boolean)
  if (metadataSummary) return truncateText(metadataSummary, 140)

  const prompt = asset.prompt || ''
  const firstBlock = prompt
    .split(/\n\s*\n/)
    .map((part) => collapseWhitespace(part))
    .find(Boolean) || prompt
  const withoutPrefix = firstBlock.replace(/^story\s+scene\s*:\s*/i, '')
  return truncateText(withoutPrefix || prompt || asset.asset_id, 140)
}

export function getAssetReviewSummary(asset: AgentImageflowAssetResponse): OperatorReviewField[] {
  const fields: OperatorReviewField[] = []
  pushField(fields, 'prompt', 'Prompt', getAssetReviewTitle(asset))
  pushField(fields, 'story', 'Story', getMetadataString(asset, 'story_id'))
  pushField(fields, 'scene', 'Scene', getMetadataString(asset, 'scene_id'))
  pushField(fields, 'source', 'Source', getMetadataString(asset, 'source'))
  pushField(fields, 'created', 'Created', asset.created_at)
  pushField(fields, 'target', 'Target', getMetadataString(asset, 'target_path'))
  return fields
}

export function getAssetTechnicalFields(asset: AgentImageflowAssetResponse): OperatorReviewField[] {
  const fields: OperatorReviewField[] = []
  pushField(fields, 'asset', 'Asset ID', asset.asset_id)
  pushField(fields, 'task', 'Task ID', asset.task_id)
  pushField(fields, 'workspace', 'Workspace', asset.workspace_id)
  pushField(fields, 'project', 'Project', asset.project_id)
  pushField(fields, 'campaign', 'Campaign', asset.campaign_id)
  pushField(fields, 'version', 'Version', asset.current_version)
  pushField(fields, 'provider', 'Provider', asset.provider)
  pushField(fields, 'model', 'Model', asset.model)
  pushField(fields, 'hash', 'Hash', asset.hash)
  pushField(fields, 'source', 'Source', getMetadataString(asset, 'source'))
  pushField(fields, 'session', 'Session', getMetadataString(asset, 'session_id'))
  pushField(fields, 'batch', 'Batch', getMetadataString(asset, 'batch_id'))
  pushField(fields, 'story', 'Story', getMetadataString(asset, 'story_id'))
  pushField(fields, 'scene', 'Scene', getMetadataString(asset, 'scene_id'))
  pushField(fields, 'target', 'Target', getMetadataString(asset, 'target_path'))
  pushField(fields, 'metadata', 'Metadata', stringifySanitizedJSON(asset.metadata_json))
  pushField(fields, 'parameters', 'Parameters', stringifySanitizedJSON(asset.parameters_json))
  return fields
}

export function getLocalhostMismatchWarning(pageOrigin: string, apiBaseUrl: string): string | null {
  const pageUrl = parseUrl(pageOrigin)
  const apiUrl = parseUrl(apiBaseUrl, pageUrl?.origin)
  const pageHost = pageUrl?.hostname
  const apiHost = apiUrl?.hostname
  if (!pageHost || !apiHost) return null
  if (!isLocalhostAlias(pageHost) || !isLocalhostAlias(apiHost)) return null
  if (pageHost === apiHost) return null
  return `当前页面使用 ${pageHost}，Agent ImageFlow API 使用 ${apiHost}。Admin cookie 会按 host 隔离；请统一使用 localhost 或 127.0.0.1 访问 Web 和 API 后再登录。`
}

export function getProductionFiltersFromAsset(asset: AgentImageflowAssetResponse): OperatorReviewProductionFilters | null {
  const sessionId = getMetadataString(asset, 'session_id')
  const batchId = getMetadataString(asset, 'batch_id')
  if (!sessionId && !batchId) return null
  return {
    sessionId,
    batchId,
    storyId: getMetadataString(asset, 'story_id'),
    source: getMetadataString(asset, 'source'),
    status: '',
    includeSetup: false,
    limit: '100',
  }
}
