import { AgentImageflowApiError, type AgentImageflowCleanupRequest } from './agentImageflowApi'

export const STORAGE_CLEANUP_CONFIRM_PHRASE = '清理当前空间'

export function buildStorageCleanupPreviewInput(): AgentImageflowCleanupRequest {
  return {
    include_rejected: true,
    include_generated: true,
    include_deprecated: false,
    include_failed_task_tmp: true,
    include_orphans: true,
  }
}

export function buildStorageCleanupExecuteInput(dryRunToken: string): AgentImageflowCleanupRequest {
  return {
    ...buildStorageCleanupPreviewInput(),
    dry_run_token: dryRunToken,
    execute: true,
  }
}

export function maskStorageCleanupToken(token: string): string {
  const trimmed = token.trim()
  if (!trimmed) return '未生成'
  if (trimmed.length <= 8) return `已隐藏（${trimmed.length} 位）`
  return `${trimmed.slice(0, 4)}...${trimmed.slice(-4)}`
}

export function formatStorageCleanupError(error: unknown): string {
  if (error instanceof AgentImageflowApiError) {
    if (error.status === 401 && error.errorCode === 'admin_session_required') {
      return '需要先登录 Admin 控制台，才能执行数据清理。'
    }
    if (error.status === 403) {
      return '当前账号没有数据清理权限，请确认已进入 Admin 控制台。'
    }
    if (error.status === 429) {
      return error.retryAfterSeconds
        ? `清理请求过于频繁，请在 ${error.retryAfterSeconds} 秒后重试。`
        : '清理请求过于频繁，请稍后重试。'
    }
  }
  return error instanceof Error ? error.message : String(error)
}
