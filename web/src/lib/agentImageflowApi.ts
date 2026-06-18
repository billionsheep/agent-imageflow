export interface AgentImageflowScope {
  workspaceId: string
  projectId: string
  campaignId: string
}

export interface AgentImageflowTaskInput {
  idempotency_key?: string
  title: string
  purpose?: string
  prompt: string
  negative_prompt?: string
  style_preset?: string
  aspect_ratio?: string
  output_format?: 'png' | 'jpeg' | 'webp' | string
  requested_count?: number
  provider?: string
  review_required?: boolean
  metadata_json?: Record<string, unknown>
}

export interface AgentImageflowTaskResponse {
  task_id: string
  status: string
  asset_ids: string[]
  assets?: Array<{
    asset_id: string
    status: string
    thumbnail_url: string
    metadata_url: string
  }>
  error_code?: string | null
  error_message?: string | null
}

export interface AgentImageflowAssetResponse {
  asset_id: string
  status: string
  delivery: {
    local_path: string
    download_url: string
    thumbnail_url: string
    metadata_url: string
  }
}

export function normalizeAgentImageflowApiBaseUrl(baseUrl: string): string {
  return (baseUrl || 'http://localhost:8081').trim().replace(/\/+$/, '')
}

export function buildAgentImageflowTaskUrl(baseUrl: string, scope: AgentImageflowScope): string {
  const base = normalizeAgentImageflowApiBaseUrl(baseUrl)
  return [
    base,
    'api',
    'workspaces',
    encodeURIComponent(scope.workspaceId),
    'projects',
    encodeURIComponent(scope.projectId),
    'campaigns',
    encodeURIComponent(scope.campaignId),
    'tasks',
  ].join('/')
}

export function buildAgentImageflowTaskStatusUrl(baseUrl: string, taskId: string): string {
  return `${normalizeAgentImageflowApiBaseUrl(baseUrl)}/api/tasks/${encodeURIComponent(taskId)}`
}

export function buildAgentImageflowAssetUrl(baseUrl: string, assetId: string): string {
  return `${normalizeAgentImageflowApiBaseUrl(baseUrl)}/api/assets/${encodeURIComponent(assetId)}`
}

export async function createAgentImageflowTask(
  baseUrl: string,
  scope: AgentImageflowScope,
  input: AgentImageflowTaskInput,
): Promise<AgentImageflowTaskResponse> {
  return requestJson<AgentImageflowTaskResponse>(buildAgentImageflowTaskUrl(baseUrl, scope), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
}

export async function getAgentImageflowTask(baseUrl: string, taskId: string): Promise<AgentImageflowTaskResponse> {
  return requestJson<AgentImageflowTaskResponse>(buildAgentImageflowTaskStatusUrl(baseUrl, taskId))
}

export async function approveAgentImageflowAsset(baseUrl: string, assetId: string): Promise<AgentImageflowAssetResponse> {
  return requestJson<AgentImageflowAssetResponse>(`${buildAgentImageflowAssetUrl(baseUrl, assetId)}/approve`, {
    method: 'POST',
  })
}

export async function rejectAgentImageflowAsset(baseUrl: string, assetId: string): Promise<AgentImageflowAssetResponse> {
  return requestJson<AgentImageflowAssetResponse>(`${buildAgentImageflowAssetUrl(baseUrl, assetId)}/reject`, {
    method: 'POST',
  })
}

async function requestJson<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, init)
  const text = await response.text()
  const payload = text ? JSON.parse(text) : null
  if (!response.ok) {
    const message = typeof payload?.error_message === 'string' ? payload.error_message : `HTTP ${response.status}`
    throw new Error(message)
  }
  return payload as T
}

