import type { AgentImageflowAdminSessionResponse } from './agentImageflowApi'

export type ConsoleAuthView = 'checking' | 'login' | 'unconfigured' | 'console'

export function getConsoleAuthView(
  checking: boolean,
  session: AgentImageflowAdminSessionResponse | null,
): ConsoleAuthView {
  if (checking) return 'checking'
  if (session?.authenticated) return 'console'
  if (session && session.configured === false) return 'unconfigured'
  return 'login'
}

