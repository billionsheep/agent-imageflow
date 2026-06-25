import { describe, expect, it } from 'vitest'
import { AgentImageflowApiError } from '../lib/agentImageflowApi'
import { getLoginErrorMessage } from './ConsoleLoginScreen'

describe('getLoginErrorMessage', () => {
  it('explains admin login failures without asking for provider keys', () => {
    expect(getLoginErrorMessage(new AgentImageflowApiError('invalid', 401, 'admin_login_invalid'))).toContain('服务器 Admin 登录')
    expect(getLoginErrorMessage(new AgentImageflowApiError('missing', 503, 'admin_not_configured'))).toContain('控制台登录未配置')
    expect(getLoginErrorMessage(new AgentImageflowApiError('too many', 429, 'rate_limited'))).toContain('请求过于频繁')
    expect(getLoginErrorMessage(new AgentImageflowApiError('not found', 404, 'not_found'))).toContain('版本或地址不匹配')
  })
})
