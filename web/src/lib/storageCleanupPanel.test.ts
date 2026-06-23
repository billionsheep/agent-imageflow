import { describe, expect, it } from 'vitest'
import { AgentImageflowApiError } from './agentImageflowApi'
import {
  STORAGE_CLEANUP_CONFIRM_PHRASE,
  buildStorageCleanupExecuteInput,
  buildStorageCleanupPreviewInput,
  formatStorageCleanupError,
  maskStorageCleanupToken,
} from './storageCleanupPanel'

describe('storage cleanup panel helpers', () => {
  it('builds the default preview payload for scope cleanup', () => {
    expect(buildStorageCleanupPreviewInput()).toEqual({
      include_rejected: true,
      include_generated: true,
      include_deprecated: true,
      include_failed_task_tmp: true,
      include_orphans: true,
    })
  })

  it('builds execute payload from a dry-run token and keeps cleanup filters', () => {
    expect(buildStorageCleanupExecuteInput('cleanup_1234567890abcdef')).toEqual({
      include_rejected: true,
      include_generated: true,
      include_deprecated: true,
      include_failed_task_tmp: true,
      include_orphans: true,
      dry_run_token: 'cleanup_1234567890abcdef',
      execute: true,
    })
  })

  it('masks dry-run tokens instead of exposing the full token', () => {
    expect(maskStorageCleanupToken('cleanup_1234567890abcdef')).toBe('clea...cdef')
    expect(maskStorageCleanupToken('short')).toBe('已隐藏（5 位）')
    expect(maskStorageCleanupToken('')).toBe('未生成')
  })

  it('formats admin/permission/rate-limit errors for cleanup actions', () => {
    expect(formatStorageCleanupError(new AgentImageflowApiError('Admin session required', 401, 'admin_session_required'))).toBe('需要先登录 Admin 控制台，才能执行数据清理。')
    expect(formatStorageCleanupError(new AgentImageflowApiError('Forbidden', 403, 'admin_session_required'))).toBe('当前账号没有数据清理权限，请确认已进入 Admin 控制台。')
    expect(formatStorageCleanupError(new AgentImageflowApiError('Too Many Requests', 429, 'rate_limited', 17))).toBe('清理请求过于频繁，请在 17 秒后重试。')
    expect(formatStorageCleanupError(new Error('network unavailable'))).toBe('network unavailable')
  })

  it('keeps the required confirmation phrase stable', () => {
    expect(STORAGE_CLEANUP_CONFIRM_PHRASE).toBe('清理当前空间')
  })
})
