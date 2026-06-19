export interface AgentImageflowScope {
  workspaceId: string
  projectId: string
  campaignId: string
}

export interface AgentImageflowAuth {
  apiKey?: string
  basicUsername?: string
  basicPassword?: string
}

export interface AgentImageflowWorkspace {
  workspace_id: string
  name: string
  archived?: boolean
}

export interface AgentImageflowProject {
  workspace_id: string
  project_id: string
  name: string
  description?: string
  archived?: boolean
}

export interface AgentImageflowCampaign {
  workspace_id: string
  project_id: string
  campaign_id: string
  name: string
  description?: string
  archived?: boolean
}

export interface AgentImageflowWorkspaceListResponse {
  workspaces: AgentImageflowWorkspace[]
}

export interface AgentImageflowProjectListResponse {
  workspace_id: string
  projects: AgentImageflowProject[]
}

export interface AgentImageflowCampaignListResponse {
  workspace_id: string
  project_id: string
  campaigns: AgentImageflowCampaign[]
}

export interface AgentImageflowCreateWorkspaceInput {
  workspace_id: string
  name: string
}

export interface AgentImageflowCreateProjectInput {
  project_id: string
  name: string
  description?: string
}

export interface AgentImageflowCreateCampaignInput {
  campaign_id: string
  name: string
  description?: string
}

export interface AgentImageflowUpdateWorkspaceInput {
  name?: string
  archived?: boolean
}

export interface AgentImageflowUpdateProjectInput {
  name?: string
  archived?: boolean
}

export interface AgentImageflowUpdateCampaignInput {
  name?: string
  archived?: boolean
}

export interface AgentImageflowTaskInput {
  idempotency_key?: string
  title: string
  purpose?: string
  prompt: string
  negative_prompt?: string
  style_preset?: string
  prompt_template?: string
  template_variables?: Record<string, unknown>
  reference_images?: AgentImageflowReferenceImage[]
  best_of_config?: AgentImageflowBestOfConfig
  mask_image?: AgentImageflowMaskImage
  generation_config?: Record<string, unknown>
  use_project_quality_profile?: boolean
  aspect_ratio?: string
  output_format?: 'png' | 'jpeg' | 'webp' | string
  requested_count?: number
  provider?: string
  selection_mode?: 'manual_optional' | 'auto' | 'best_of' | string
  review_required?: boolean
  metadata_json?: Record<string, unknown>
}

export interface AgentImageflowReferenceImage {
  id?: string
  url?: string
  asset_id?: string
  input_file_id?: string
  role?: string
  source?: string
  mime_type?: string
  width?: number
  height?: number
  weight?: number
}

export interface AgentImageflowMaskImage {
  id?: string
  url?: string
  asset_id?: string
  input_file_id?: string
  target_image_id?: string
  source?: string
  mime_type?: string
  width?: number
  height?: number
  has_mask?: boolean
}

export interface AgentImageflowBestOfConfig {
  strategy?: 'local_metadata_v1' | 'http_judge_v1' | string
  judge_prompt?: string
  auto_reject_non_selected?: boolean
}

export interface AgentImageflowInputFileResponse {
  input_file_id: string
  workspace_id: string
  project_id: string
  campaign_id: string
  kind: string
  original_filename: string
  mime_type: string
  width?: number
  height?: number
  size_bytes: number
  download_url: string
  metadata_url: string
}

export interface AgentImageflowUploadInputFileOptions {
  kind?: 'reference' | 'mask' | string
  file: Blob
  filename?: string
  mimeType?: string
}

export interface AgentImageflowQualityProfile {
  prompt_template?: string
  negative_prompt?: string
  style_preset?: string
  reference_images?: AgentImageflowReferenceImage[]
  best_of_config?: AgentImageflowBestOfConfig
  generation_config?: Record<string, unknown>
}

export interface AgentImageflowQualityProfileResponse {
  workspace_id: string
  project_id: string
  quality_profile: AgentImageflowQualityProfile
}

export interface AgentImageflowStorageUsageCategoryStat {
  category: string
  file_count: number
  bytes: number
}

export interface AgentImageflowStorageUsageSnapshot {
  scope_type: string
  workspace_id?: string
  project_id?: string
  campaign_id?: string
  file_count: number
  bytes: number
  categories: AgentImageflowStorageUsageCategoryStat[]
}

export interface AgentImageflowStorageUsageScopes {
  instance: AgentImageflowStorageUsageSnapshot
  workspace: AgentImageflowStorageUsageSnapshot
  project: AgentImageflowStorageUsageSnapshot
  campaign: AgentImageflowStorageUsageSnapshot
}

export interface AgentImageflowStorageGovernanceCountSnapshot {
  task_count: number
  failed_task_count: number
  asset_count: number
  generated_asset_count: number
  selected_asset_count: number
  rejected_asset_count: number
  published_asset_count: number
}

export interface AgentImageflowStorageGovernanceCounts {
  instance: AgentImageflowStorageGovernanceCountSnapshot
  workspace: AgentImageflowStorageGovernanceCountSnapshot
  project: AgentImageflowStorageGovernanceCountSnapshot
  campaign: AgentImageflowStorageGovernanceCountSnapshot
}

export interface AgentImageflowStorageGovernanceResponse {
  generated_at: string
  scope: {
    WorkspaceID?: string
    ProjectID?: string
    CampaignID?: string
    workspace_id?: string
    project_id?: string
    campaign_id?: string
  }
  usage: AgentImageflowStorageUsageScopes
  counts: AgentImageflowStorageGovernanceCounts
}

export interface AgentImageflowStorageIntegrityIssue {
  kind: string
  severity: string
  task_id?: string
  asset_id?: string
  version_id?: string
  status?: string
  file_kind?: string
  message: string
  repair_hint?: string
}

export interface AgentImageflowStorageIntegritySummary {
  issue_count: number
  by_kind: Record<string, number>
}

export interface AgentImageflowStorageIntegrityResponse {
  checked_at: string
  scope: {
    WorkspaceID?: string
    ProjectID?: string
    CampaignID?: string
    workspace_id?: string
    project_id?: string
    campaign_id?: string
  }
  ok: boolean
  summary: AgentImageflowStorageIntegritySummary
  issues: AgentImageflowStorageIntegrityIssue[]
}

export interface AgentImageflowTaskResponse {
  task_id: string
  status: string
  asset_ids: string[]
  assets?: AgentImageflowAssetListEntry[]
  selection_mode?: string
  error_code?: string | null
  error_message?: string | null
}

export interface AgentImageflowAssetListEntry {
  asset_id: string
  status: string
  thumbnail_url: string
  metadata_url: string
}

export interface AgentImageflowAssetResponse {
  asset_id: string
  workspace_id?: string
  project_id?: string
  campaign_id?: string
  task_id?: string
  current_version?: number
  status: string
  hash?: string
  provider?: string
  model?: string
  prompt?: string
  parameters_json?: Record<string, unknown>
  metadata_json?: Record<string, unknown>
  delivery: {
    local_path: string
    download_url: string
    thumbnail_url: string
    metadata_url: string
  }
  created_at?: string
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

export function buildAgentImageflowWorkspacesUrl(baseUrl: string): string {
  return `${normalizeAgentImageflowApiBaseUrl(baseUrl)}/api/workspaces`
}

export function buildAgentImageflowProjectsUrl(baseUrl: string, workspaceId: string): string {
  return `${buildAgentImageflowWorkspacesUrl(baseUrl)}/${encodeURIComponent(workspaceId)}/projects`
}

export function buildAgentImageflowWorkspaceUrl(baseUrl: string, workspaceId: string): string {
  return `${buildAgentImageflowWorkspacesUrl(baseUrl)}/${encodeURIComponent(workspaceId)}`
}

export function buildAgentImageflowCampaignsUrl(baseUrl: string, scope: Pick<AgentImageflowScope, 'workspaceId' | 'projectId'>): string {
  return `${buildAgentImageflowProjectsUrl(baseUrl, scope.workspaceId)}/${encodeURIComponent(scope.projectId)}/campaigns`
}

export function buildAgentImageflowProjectUrl(baseUrl: string, scope: Pick<AgentImageflowScope, 'workspaceId' | 'projectId'>): string {
  return `${buildAgentImageflowProjectsUrl(baseUrl, scope.workspaceId)}/${encodeURIComponent(scope.projectId)}`
}

export function buildAgentImageflowCampaignUrl(baseUrl: string, scope: AgentImageflowScope): string {
  return `${buildAgentImageflowCampaignsUrl(baseUrl, scope)}/${encodeURIComponent(scope.campaignId)}`
}

export function buildAgentImageflowStorageGovernanceUrl(baseUrl: string, scope: AgentImageflowScope): string {
  return `${buildAgentImageflowCampaignUrl(baseUrl, scope)}/storage-governance`
}

export function buildAgentImageflowStorageIntegrityUrl(baseUrl: string, scope: AgentImageflowScope): string {
  return `${buildAgentImageflowCampaignUrl(baseUrl, scope)}/storage-integrity`
}

export function buildAgentImageflowInputFilesUrl(baseUrl: string, scope: AgentImageflowScope): string {
  return `${buildAgentImageflowCampaignsUrl(baseUrl, scope)}/${encodeURIComponent(scope.campaignId)}/input-files`
}

export function buildAgentImageflowAssetsUrl(baseUrl: string, scope: Pick<AgentImageflowScope, 'projectId' | 'campaignId'>): string {
  const base = normalizeAgentImageflowApiBaseUrl(baseUrl)
  return [
    base,
    'api',
    'projects',
    encodeURIComponent(scope.projectId),
    'campaigns',
    encodeURIComponent(scope.campaignId),
    'assets',
  ].join('/')
}

export function buildAgentImageflowQualityProfileUrl(baseUrl: string, scope: Pick<AgentImageflowScope, 'workspaceId' | 'projectId'>): string {
  const base = normalizeAgentImageflowApiBaseUrl(baseUrl)
  return [
    base,
    'api',
    'workspaces',
    encodeURIComponent(scope.workspaceId),
    'projects',
    encodeURIComponent(scope.projectId),
    'quality-profile',
  ].join('/')
}

export function buildAgentImageflowAssetUrl(baseUrl: string, assetId: string): string {
  return `${normalizeAgentImageflowApiBaseUrl(baseUrl)}/api/assets/${encodeURIComponent(assetId)}`
}

export function normalizeAgentImageflowAssetStatus(status: string): string {
  if (status === 'draft') return 'generated'
  if (status === 'approved') return 'selected'
  return status
}

export function normalizeAgentImageflowTaskResponse(response: AgentImageflowTaskResponse): AgentImageflowTaskResponse {
  return {
    ...response,
    assets: response.assets?.map((asset) => ({
      ...asset,
      status: normalizeAgentImageflowAssetStatus(asset.status),
    })),
  }
}

export function normalizeAgentImageflowAssetResponse(response: AgentImageflowAssetResponse): AgentImageflowAssetResponse {
  return {
    ...response,
    status: normalizeAgentImageflowAssetStatus(response.status),
  }
}

export function normalizeAgentImageflowAssetListResponse(response: AgentImageflowAssetResponse[]): AgentImageflowAssetResponse[] {
  return response.map(normalizeAgentImageflowAssetResponse)
}

export function buildAgentImageflowHeaders(
  auth?: AgentImageflowAuth,
  headers: Record<string, string> = {},
): Record<string, string> {
  const nextHeaders = { ...headers }
  const apiKey = auth?.apiKey?.trim()
  if (apiKey) {
    nextHeaders['X-API-Key'] = apiKey
  }
  if (auth?.basicUsername || auth?.basicPassword) {
    nextHeaders.Authorization = `Basic ${encodeBasicCredentials(auth.basicUsername ?? '', auth.basicPassword ?? '')}`
  }
  return nextHeaders
}

export async function createAgentImageflowTask(
  baseUrl: string,
  scope: AgentImageflowScope,
  input: AgentImageflowTaskInput,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowTaskResponse> {
  const response = await requestJson<AgentImageflowTaskResponse>(buildAgentImageflowTaskUrl(baseUrl, scope), {
    method: 'POST',
    headers: buildAgentImageflowHeaders(auth, { 'Content-Type': 'application/json' }),
    body: JSON.stringify(input),
  })
  return normalizeAgentImageflowTaskResponse(response)
}

export async function listAgentImageflowWorkspaces(baseUrl: string, auth?: AgentImageflowAuth): Promise<AgentImageflowWorkspaceListResponse> {
  return requestJson<AgentImageflowWorkspaceListResponse>(buildAgentImageflowWorkspacesUrl(baseUrl), {
    headers: buildAgentImageflowHeaders(auth),
  })
}

export async function createAgentImageflowWorkspace(
  baseUrl: string,
  input: AgentImageflowCreateWorkspaceInput,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowWorkspace> {
  return requestJson<AgentImageflowWorkspace>(buildAgentImageflowWorkspacesUrl(baseUrl), {
    method: 'POST',
    headers: buildAgentImageflowHeaders(auth, { 'Content-Type': 'application/json' }),
    body: JSON.stringify(input),
  })
}

export async function updateAgentImageflowWorkspace(
  baseUrl: string,
  workspaceId: string,
  input: AgentImageflowUpdateWorkspaceInput,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowWorkspace> {
  return requestJson<AgentImageflowWorkspace>(buildAgentImageflowWorkspaceUrl(baseUrl, workspaceId), {
    method: 'PATCH',
    headers: buildAgentImageflowHeaders(auth, { 'Content-Type': 'application/json' }),
    body: JSON.stringify(input),
  })
}

export async function deleteAgentImageflowWorkspace(
  baseUrl: string,
  workspaceId: string,
  auth?: AgentImageflowAuth,
): Promise<void> {
  await requestEmpty(buildAgentImageflowWorkspaceUrl(baseUrl, workspaceId), {
    method: 'DELETE',
    headers: buildAgentImageflowHeaders(auth),
  })
}

export async function listAgentImageflowProjects(
  baseUrl: string,
  workspaceId: string,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowProjectListResponse> {
  return requestJson<AgentImageflowProjectListResponse>(buildAgentImageflowProjectsUrl(baseUrl, workspaceId), {
    headers: buildAgentImageflowHeaders(auth),
  })
}

export async function createAgentImageflowProject(
  baseUrl: string,
  workspaceId: string,
  input: AgentImageflowCreateProjectInput,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowProject> {
  return requestJson<AgentImageflowProject>(buildAgentImageflowProjectsUrl(baseUrl, workspaceId), {
    method: 'POST',
    headers: buildAgentImageflowHeaders(auth, { 'Content-Type': 'application/json' }),
    body: JSON.stringify(input),
  })
}

export async function updateAgentImageflowProject(
  baseUrl: string,
  scope: Pick<AgentImageflowScope, 'workspaceId' | 'projectId'>,
  input: AgentImageflowUpdateProjectInput,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowProject> {
  return requestJson<AgentImageflowProject>(buildAgentImageflowProjectUrl(baseUrl, scope), {
    method: 'PATCH',
    headers: buildAgentImageflowHeaders(auth, { 'Content-Type': 'application/json' }),
    body: JSON.stringify(input),
  })
}

export async function deleteAgentImageflowProject(
  baseUrl: string,
  scope: Pick<AgentImageflowScope, 'workspaceId' | 'projectId'>,
  auth?: AgentImageflowAuth,
): Promise<void> {
  await requestEmpty(buildAgentImageflowProjectUrl(baseUrl, scope), {
    method: 'DELETE',
    headers: buildAgentImageflowHeaders(auth),
  })
}

export async function listAgentImageflowCampaigns(
  baseUrl: string,
  scope: Pick<AgentImageflowScope, 'workspaceId' | 'projectId'>,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowCampaignListResponse> {
  return requestJson<AgentImageflowCampaignListResponse>(buildAgentImageflowCampaignsUrl(baseUrl, scope), {
    headers: buildAgentImageflowHeaders(auth),
  })
}

export async function createAgentImageflowCampaign(
  baseUrl: string,
  scope: Pick<AgentImageflowScope, 'workspaceId' | 'projectId'>,
  input: AgentImageflowCreateCampaignInput,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowCampaign> {
  return requestJson<AgentImageflowCampaign>(buildAgentImageflowCampaignsUrl(baseUrl, scope), {
    method: 'POST',
    headers: buildAgentImageflowHeaders(auth, { 'Content-Type': 'application/json' }),
    body: JSON.stringify(input),
  })
}

export async function updateAgentImageflowCampaign(
  baseUrl: string,
  scope: AgentImageflowScope,
  input: AgentImageflowUpdateCampaignInput,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowCampaign> {
  return requestJson<AgentImageflowCampaign>(buildAgentImageflowCampaignUrl(baseUrl, scope), {
    method: 'PATCH',
    headers: buildAgentImageflowHeaders(auth, { 'Content-Type': 'application/json' }),
    body: JSON.stringify(input),
  })
}

export async function deleteAgentImageflowCampaign(
  baseUrl: string,
  scope: AgentImageflowScope,
  auth?: AgentImageflowAuth,
): Promise<void> {
  await requestEmpty(buildAgentImageflowCampaignUrl(baseUrl, scope), {
    method: 'DELETE',
    headers: buildAgentImageflowHeaders(auth),
  })
}

export async function uploadAgentImageflowInputFile(
  baseUrl: string,
  scope: AgentImageflowScope,
  options: AgentImageflowUploadInputFileOptions,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowInputFileResponse> {
  const formData = new FormData()
  formData.append('file', options.file, options.filename ?? 'input.png')
  if (options.kind?.trim()) {
    formData.append('kind', options.kind.trim())
  }
  if (options.mimeType?.trim()) {
    formData.append('mime_type', options.mimeType.trim())
  }
  return requestJson<AgentImageflowInputFileResponse>(buildAgentImageflowInputFilesUrl(baseUrl, scope), {
    method: 'POST',
    headers: buildAgentImageflowHeaders(auth),
    body: formData,
  })
}

export async function getAgentImageflowTask(baseUrl: string, taskId: string, auth?: AgentImageflowAuth): Promise<AgentImageflowTaskResponse> {
  const response = await requestJson<AgentImageflowTaskResponse>(buildAgentImageflowTaskStatusUrl(baseUrl, taskId), {
    headers: buildAgentImageflowHeaders(auth),
  })
  return normalizeAgentImageflowTaskResponse(response)
}

export async function listAgentImageflowAssets(
  baseUrl: string,
  scope: Pick<AgentImageflowScope, 'projectId' | 'campaignId'>,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowAssetResponse[]> {
  const response = await requestJson<AgentImageflowAssetResponse[]>(buildAgentImageflowAssetsUrl(baseUrl, scope), {
    headers: buildAgentImageflowHeaders(auth),
  })
  return normalizeAgentImageflowAssetListResponse(response)
}

export async function getAgentImageflowQualityProfile(
  baseUrl: string,
  scope: Pick<AgentImageflowScope, 'workspaceId' | 'projectId'>,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowQualityProfileResponse> {
  return requestJson<AgentImageflowQualityProfileResponse>(buildAgentImageflowQualityProfileUrl(baseUrl, scope), {
    headers: buildAgentImageflowHeaders(auth),
  })
}

export async function getAgentImageflowStorageGovernance(
  baseUrl: string,
  scope: AgentImageflowScope,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowStorageGovernanceResponse> {
  return requestJson<AgentImageflowStorageGovernanceResponse>(buildAgentImageflowStorageGovernanceUrl(baseUrl, scope), {
    headers: buildAgentImageflowHeaders(auth),
  })
}

export async function getAgentImageflowStorageIntegrity(
  baseUrl: string,
  scope: AgentImageflowScope,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowStorageIntegrityResponse> {
  return requestJson<AgentImageflowStorageIntegrityResponse>(buildAgentImageflowStorageIntegrityUrl(baseUrl, scope), {
    headers: buildAgentImageflowHeaders(auth),
  })
}

export async function updateAgentImageflowQualityProfile(
  baseUrl: string,
  scope: Pick<AgentImageflowScope, 'workspaceId' | 'projectId'>,
  profile: AgentImageflowQualityProfile,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowQualityProfileResponse> {
  return requestJson<AgentImageflowQualityProfileResponse>(buildAgentImageflowQualityProfileUrl(baseUrl, scope), {
    method: 'POST',
    headers: buildAgentImageflowHeaders(auth, { 'Content-Type': 'application/json' }),
    body: JSON.stringify(profile),
  })
}

export async function getAgentImageflowAsset(baseUrl: string, assetId: string, auth?: AgentImageflowAuth): Promise<AgentImageflowAssetResponse> {
  const response = await requestJson<AgentImageflowAssetResponse>(buildAgentImageflowAssetUrl(baseUrl, assetId), {
    headers: buildAgentImageflowHeaders(auth),
  })
  return normalizeAgentImageflowAssetResponse(response)
}

export async function selectAgentImageflowAsset(baseUrl: string, assetId: string, auth?: AgentImageflowAuth): Promise<AgentImageflowAssetResponse> {
  const response = await requestJson<AgentImageflowAssetResponse>(`${buildAgentImageflowAssetUrl(baseUrl, assetId)}/approve`, {
    method: 'POST',
    headers: buildAgentImageflowHeaders(auth),
  })
  return normalizeAgentImageflowAssetResponse(response)
}

export async function approveAgentImageflowAsset(baseUrl: string, assetId: string, auth?: AgentImageflowAuth): Promise<AgentImageflowAssetResponse> {
  return selectAgentImageflowAsset(baseUrl, assetId, auth)
}

export async function rejectAgentImageflowAsset(baseUrl: string, assetId: string, auth?: AgentImageflowAuth): Promise<AgentImageflowAssetResponse> {
  const response = await requestJson<AgentImageflowAssetResponse>(`${buildAgentImageflowAssetUrl(baseUrl, assetId)}/reject`, {
    method: 'POST',
    headers: buildAgentImageflowHeaders(auth),
  })
  return normalizeAgentImageflowAssetResponse(response)
}

function encodeBasicCredentials(username: string, password: string): string {
  const input = `${username}:${password}`
  if (typeof globalThis.btoa === 'function') {
    return globalThis.btoa(input)
  }
  const globalBuffer = (globalThis as { Buffer?: { from(value: string, encoding?: string): { toString(encoding: string): string } } }).Buffer
  if (globalBuffer) {
    return globalBuffer.from(input, 'utf-8').toString('base64')
  }
  throw new Error('Basic auth encoding is not available in this runtime')
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

async function requestEmpty(url: string, init?: RequestInit): Promise<void> {
  const response = await fetch(url, init)
  const text = await response.text()
  const payload = text ? JSON.parse(text) : null
  if (!response.ok) {
    const message = typeof payload?.error_message === 'string' ? payload.error_message : `HTTP ${response.status}`
    throw new Error(message)
  }
}
