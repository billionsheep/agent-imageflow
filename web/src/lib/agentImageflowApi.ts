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

export interface AgentImageflowAdminLoginInput {
  username: string
  password: string
}

export interface AgentImageflowAdminSessionResponse {
  authenticated: boolean
  username?: string
  expires_at?: string
  configured: boolean
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
  character_ids?: string[]
  reference_asset_ids?: string[]
  prompt_recipe_id?: string
  use_project_visual_context?: boolean
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

export interface AgentImageflowProviderProfile {
  enabled?: boolean
  provider?: string
  model?: string
  base_url?: string
  generation_config?: Record<string, unknown>
  use_project_quality_profile?: boolean
  api_mode?: string
  stream?: boolean
  partial_images?: number
  max_n?: number
  supports_url_result?: boolean
  preferred_response_format?: string
  max_concurrency?: number
  timeout_seconds?: number
}

export interface AgentImageflowProviderProfileResponse {
  workspace_id: string
  project_id: string
  provider_profile: AgentImageflowProviderProfile
}

export interface AgentImageflowAssetListQuery {
  limit?: number
  offset?: number
  status?: string
  provider?: string
  model?: string
  source?: string
  sessionId?: string
  batchId?: string
  keyword?: string
  createdFrom?: string
  createdTo?: string
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

export interface AgentImageflowTaskAttempt {
  attempt_id: string
  task_id: string
  attempt_no: number
  status: string
  provider: string
  provider_request_id?: string
  request_mode?: string
  api_mode?: string
  stream?: boolean
  partial_image_count?: number
  started_at: string
  finished_at?: string
  latency_ms?: number
  queue_wait_ms?: number
  provider_first_byte_ms?: number
  provider_total_ms?: number
  response_download_ms?: number
  store_ms?: number
  thumbnail_ms?: number
  retry_count?: number
  error_stage?: string
  response_bytes?: number
  retry_after?: string
  error_code?: string
  error_message?: string
}

export interface AgentImageflowTaskAttemptsResponse {
  task_id: string
  attempts: AgentImageflowTaskAttempt[]
}

export interface AgentImageflowBatchProgressQuery {
  sessionId?: string
  batchId?: string
  limit?: number
}

export interface AgentImageflowBatchProgressTask {
  task_id: string
  status: string
  asset_count: number
  attempt_count: number
  retrying: boolean
  error_stage?: string
  error_code?: string
  error_message?: string
  created_at: string
  updated_at: string
}

export interface AgentImageflowBatchProgressResponse {
  generated_at: string
  project_id: string
  campaign_id: string
  session_id?: string
  batch_id?: string
  counts: {
    task_count: number
    queued_count: number
    running_count: number
    succeeded_count: number
    partial_count: number
    failed_count: number
    retrying_count: number
    asset_count: number
    attempt_count: number
  }
  tasks: AgentImageflowBatchProgressTask[]
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

export class AgentImageflowApiError extends Error {
  status: number
  errorCode?: string

  constructor(message: string, status: number, errorCode?: string) {
    super(message)
    this.name = 'AgentImageflowApiError'
    this.status = status
    this.errorCode = errorCode
  }
}

export function isAgentImageflowUnauthorizedError(error: unknown): boolean {
  return error instanceof AgentImageflowApiError && (error.status === 401 || error.status === 403)
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

export function buildAgentImageflowTaskAttemptsUrl(baseUrl: string, taskId: string): string {
  return `${buildAgentImageflowTaskStatusUrl(baseUrl, taskId)}/attempts`
}

export function buildAgentImageflowWorkspacesUrl(baseUrl: string): string {
  return `${normalizeAgentImageflowApiBaseUrl(baseUrl)}/api/workspaces`
}

export function buildAgentImageflowAdminLoginUrl(baseUrl: string): string {
  return `${normalizeAgentImageflowApiBaseUrl(baseUrl)}/api/admin/login`
}

export function buildAgentImageflowAdminMeUrl(baseUrl: string): string {
  return `${normalizeAgentImageflowApiBaseUrl(baseUrl)}/api/admin/me`
}

export function buildAgentImageflowAdminLogoutUrl(baseUrl: string): string {
  return `${normalizeAgentImageflowApiBaseUrl(baseUrl)}/api/admin/logout`
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

export function buildAgentImageflowAssetsUrl(
  baseUrl: string,
  scope: Pick<AgentImageflowScope, 'projectId' | 'campaignId'>,
  query?: AgentImageflowAssetListQuery,
): string {
  const base = normalizeAgentImageflowApiBaseUrl(baseUrl)
  const url = [
    base,
    'api',
    'projects',
    encodeURIComponent(scope.projectId),
    'campaigns',
    encodeURIComponent(scope.campaignId),
    'assets',
  ].join('/')
  const params = buildAssetListSearchParams(query)
  return params ? `${url}?${params}` : url
}

export function buildAgentImageflowRecentAssetsUrl(
  baseUrl: string,
  query?: AgentImageflowAssetListQuery,
): string {
  const url = `${normalizeAgentImageflowApiBaseUrl(baseUrl)}/api/admin/assets/recent`
  const params = buildAssetListSearchParams(query)
  return params ? `${url}?${params}` : url
}

export function buildAgentImageflowBatchProgressUrl(
  baseUrl: string,
  scope: Pick<AgentImageflowScope, 'projectId' | 'campaignId'>,
  query?: AgentImageflowBatchProgressQuery,
): string {
  const base = normalizeAgentImageflowApiBaseUrl(baseUrl)
  const url = [
    base,
    'api',
    'projects',
    encodeURIComponent(scope.projectId),
    'campaigns',
    encodeURIComponent(scope.campaignId),
    'batch-progress',
  ].join('/')
  const params = new URLSearchParams()
  if (query?.sessionId?.trim()) params.set('session_id', query.sessionId.trim())
  if (query?.batchId?.trim()) params.set('batch_id', query.batchId.trim())
  if (query?.limit && Number.isFinite(query.limit)) params.set('limit', String(Math.max(1, Math.trunc(query.limit))))
  const text = params.toString()
  return text ? `${url}?${text}` : url
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

export function buildAgentImageflowProviderProfileUrl(baseUrl: string, scope: Pick<AgentImageflowScope, 'workspaceId' | 'projectId'>): string {
  const base = normalizeAgentImageflowApiBaseUrl(baseUrl)
  return [
    base,
    'api',
    'workspaces',
    encodeURIComponent(scope.workspaceId),
    'projects',
    encodeURIComponent(scope.projectId),
    'provider-profile',
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

function buildAssetListSearchParams(query?: AgentImageflowAssetListQuery): string {
  if (!query) return ''
  const params = new URLSearchParams()
  const appendString = (key: string, value?: string) => {
    const trimmed = value?.trim()
    if (trimmed) params.set(key, trimmed)
  }
  if (Number.isFinite(query.limit) && query.limit && query.limit > 0) {
    params.set('limit', String(Math.trunc(query.limit)))
  }
  if (Number.isFinite(query.offset) && query.offset && query.offset > 0) {
    params.set('offset', String(Math.trunc(query.offset)))
  }
  appendString('status', query.status)
  appendString('provider', query.provider)
  appendString('model', query.model)
  appendString('source', query.source)
  appendString('session_id', query.sessionId)
  appendString('batch_id', query.batchId)
  appendString('keyword', query.keyword)
  appendString('created_from', query.createdFrom)
  appendString('created_to', query.createdTo)
  return params.toString()
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

export async function loginAgentImageflowAdmin(
  baseUrl: string,
  input: AgentImageflowAdminLoginInput,
): Promise<AgentImageflowAdminSessionResponse> {
  return requestJson<AgentImageflowAdminSessionResponse>(buildAgentImageflowAdminLoginUrl(baseUrl), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
}

export async function getAgentImageflowAdminMe(baseUrl: string): Promise<AgentImageflowAdminSessionResponse> {
  return requestJson<AgentImageflowAdminSessionResponse>(buildAgentImageflowAdminMeUrl(baseUrl))
}

export async function logoutAgentImageflowAdmin(baseUrl: string): Promise<AgentImageflowAdminSessionResponse> {
  return requestJson<AgentImageflowAdminSessionResponse>(buildAgentImageflowAdminLogoutUrl(baseUrl), {
    method: 'POST',
  })
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

export async function getAgentImageflowTaskAttempts(baseUrl: string, taskId: string, auth?: AgentImageflowAuth): Promise<AgentImageflowTaskAttemptsResponse> {
  return requestJson<AgentImageflowTaskAttemptsResponse>(buildAgentImageflowTaskAttemptsUrl(baseUrl, taskId), {
    headers: buildAgentImageflowHeaders(auth),
  })
}

export async function listAgentImageflowAssets(
  baseUrl: string,
  scope: Pick<AgentImageflowScope, 'projectId' | 'campaignId'>,
  auth?: AgentImageflowAuth,
  query?: AgentImageflowAssetListQuery,
): Promise<AgentImageflowAssetResponse[]> {
  const response = await requestJson<AgentImageflowAssetResponse[]>(buildAgentImageflowAssetsUrl(baseUrl, scope, query), {
    headers: buildAgentImageflowHeaders(auth),
  })
  return normalizeAgentImageflowAssetListResponse(response)
}

export async function listAgentImageflowRecentAssets(
  baseUrl: string,
  query?: AgentImageflowAssetListQuery,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowAssetResponse[]> {
  const response = await requestJson<AgentImageflowAssetResponse[]>(buildAgentImageflowRecentAssetsUrl(baseUrl, query), {
    headers: buildAgentImageflowHeaders(auth),
  })
  return normalizeAgentImageflowAssetListResponse(response)
}

export async function getAgentImageflowBatchProgress(
  baseUrl: string,
  scope: Pick<AgentImageflowScope, 'projectId' | 'campaignId'>,
  auth?: AgentImageflowAuth,
  query?: AgentImageflowBatchProgressQuery,
): Promise<AgentImageflowBatchProgressResponse> {
  return requestJson<AgentImageflowBatchProgressResponse>(buildAgentImageflowBatchProgressUrl(baseUrl, scope, query), {
    headers: buildAgentImageflowHeaders(auth),
  })
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

export async function getAgentImageflowProviderProfile(
  baseUrl: string,
  scope: Pick<AgentImageflowScope, 'workspaceId' | 'projectId'>,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowProviderProfileResponse> {
  return requestJson<AgentImageflowProviderProfileResponse>(buildAgentImageflowProviderProfileUrl(baseUrl, scope), {
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

export async function updateAgentImageflowProviderProfile(
  baseUrl: string,
  scope: Pick<AgentImageflowScope, 'workspaceId' | 'projectId'>,
  profile: AgentImageflowProviderProfile,
  auth?: AgentImageflowAuth,
): Promise<AgentImageflowProviderProfileResponse> {
  return requestJson<AgentImageflowProviderProfileResponse>(buildAgentImageflowProviderProfileUrl(baseUrl, scope), {
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
  const response = await fetch(url, { credentials: 'include', ...init })
  const text = await response.text()
  const payload = text ? JSON.parse(text) : null
  if (!response.ok) {
    const message = typeof payload?.error_message === 'string' ? payload.error_message : `HTTP ${response.status}`
    const errorCode = typeof payload?.error_code === 'string' ? payload.error_code : undefined
    throw new AgentImageflowApiError(message, response.status, errorCode)
  }
  return payload as T
}

async function requestEmpty(url: string, init?: RequestInit): Promise<void> {
  const response = await fetch(url, { credentials: 'include', ...init })
  const text = await response.text()
  const payload = text ? JSON.parse(text) : null
  if (!response.ok) {
    const message = typeof payload?.error_message === 'string' ? payload.error_message : `HTTP ${response.status}`
    const errorCode = typeof payload?.error_code === 'string' ? payload.error_code : undefined
    throw new AgentImageflowApiError(message, response.status, errorCode)
  }
}
