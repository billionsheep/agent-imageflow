import { describe, expect, it } from 'vitest'
import { getConsoleAuthView } from './consoleAuth'

describe('getConsoleAuthView', () => {
  it('keeps the console behind a checking screen while admin session is loading', () => {
    expect(getConsoleAuthView(true, null)).toBe('checking')
  })

  it('allows the full console only after an authenticated admin session', () => {
    expect(getConsoleAuthView(false, {
      authenticated: true,
      username: 'admin',
      configured: true,
    })).toBe('console')
  })

  it('shows login when admin is configured but the user is not authenticated', () => {
    expect(getConsoleAuthView(false, {
      authenticated: false,
      configured: true,
    })).toBe('login')
  })

  it('shows an explicit unconfigured state when admin login is unavailable', () => {
    expect(getConsoleAuthView(false, {
      authenticated: false,
      configured: false,
    })).toBe('unconfigured')
  })
})

