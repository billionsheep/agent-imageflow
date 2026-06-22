import { type FormEvent, useEffect, useMemo, useState } from 'react'
import { AgentImageflowApiError, getAgentImageflowAdminMe, loginAgentImageflowAdmin, normalizeAgentImageflowApiBaseUrl, type AgentImageflowAdminSessionResponse } from '../lib/agentImageflowApi'
import { getConsoleAuthView } from '../lib/consoleAuth'
import { useStore } from '../store'

interface ConsoleLoginScreenProps {
  checking: boolean
  session: AgentImageflowAdminSessionResponse | null
  onSessionChange: (session: AgentImageflowAdminSessionResponse) => void
  onCheckingChange: (checking: boolean) => void
}

function getLoginErrorMessage(error: unknown): string {
  if (error instanceof AgentImageflowApiError) {
    if (error.status === 401 || error.status === 403) return '用户名或密码不正确'
    if (error.errorCode === 'admin_not_configured') return '控制台登录未配置，请先在服务器设置 Admin 账号。'
    if (error.status === 429) return '请求过于频繁，请稍后再试。'
    return error.message
  }
  return error instanceof Error ? error.message : String(error)
}

export default function ConsoleLoginScreen({
  checking,
  session,
  onSessionChange,
  onCheckingChange,
}: ConsoleLoginScreenProps) {
  const settings = useStore((state) => state.settings)
  const showToast = useStore((state) => state.showToast)
  const baseUrl = useMemo(() => normalizeAgentImageflowApiBaseUrl(settings.imageflowApiBaseUrl), [settings.imageflowApiBaseUrl])
  const authView = getConsoleAuthView(checking, session)
  const [username, setUsername] = useState(settings.imageflowBasicUsername || 'admin')
  const [password, setPassword] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (username || !settings.imageflowBasicUsername) return
    setUsername(settings.imageflowBasicUsername)
  }, [settings.imageflowBasicUsername, username])

  const refreshSession = async () => {
    onCheckingChange(true)
    setError(null)
    try {
      const nextSession = await getAgentImageflowAdminMe(baseUrl)
      onSessionChange(nextSession)
    } catch (nextError) {
      const configured = !(nextError instanceof AgentImageflowApiError && nextError.errorCode === 'admin_not_configured')
      onSessionChange({ authenticated: false, configured })
      setError(getLoginErrorMessage(nextError))
    } finally {
      onCheckingChange(false)
    }
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const nextUsername = username.trim()
    if (!nextUsername || !password) {
      setError('请输入控制台用户名和密码。')
      return
    }
    setBusy(true)
    setError(null)
    try {
      const nextSession = await loginAgentImageflowAdmin(baseUrl, { username: nextUsername, password })
      onSessionChange(nextSession)
      setPassword('')
      showToast('已进入控制台', 'success')
    } catch (nextError) {
      setError(getLoginErrorMessage(nextError))
    } finally {
      setBusy(false)
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-gray-50 px-4 py-10 text-gray-900 dark:bg-gray-950 dark:text-gray-100">
      <section className="w-full max-w-md rounded-2xl border border-gray-200/80 bg-white p-6 shadow-xl ring-1 ring-black/5 dark:border-white/[0.08] dark:bg-gray-900 dark:ring-white/10">
        <div className="mb-6">
          <div className="text-sm font-semibold text-blue-600 dark:text-blue-300">Agent ImageFlow</div>
          <h1 className="mt-2 text-2xl font-bold tracking-normal">控制台登录</h1>
          <p className="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
            服务器托管的图片资产生产平台。登录后可查看资产、管理项目视觉上下文，并使用服务器配置好的 provider 能力创建图片任务。
          </p>
        </div>

        {authView === 'checking' ? (
          <div className="rounded-xl border border-gray-200 bg-gray-50 px-4 py-5 text-sm text-gray-500 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">
            正在检查控制台登录状态...
          </div>
        ) : authView === 'unconfigured' ? (
          <div className="space-y-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-4 text-sm text-amber-800 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-100">
            <p className="font-medium">控制台登录未配置</p>
            <p>请在服务器环境变量中配置 `ADMIN_USERNAME`、`ADMIN_PASSWORD` 和 `ADMIN_SESSION_SECRET`，或临时复用 Basic Auth 凭据。</p>
            <button
              type="button"
              onClick={() => void refreshSession()}
              disabled={checking}
              className="h-9 rounded-lg border border-amber-300 bg-white px-3 text-xs font-medium text-amber-700 transition hover:border-amber-400 disabled:cursor-not-allowed disabled:opacity-60 dark:border-amber-500/30 dark:bg-white/[0.04] dark:text-amber-100"
            >
              重新检查
            </button>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-200">
              <span className="mb-1.5 block">用户名</span>
              <input
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                autoComplete="username"
                className="h-11 w-full rounded-xl border border-gray-200 bg-white px-3 text-sm outline-none transition focus:border-blue-300 dark:border-white/[0.08] dark:bg-gray-950 dark:text-gray-100 dark:focus:border-blue-500/60"
              />
            </label>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-200">
              <span className="mb-1.5 block">密码</span>
              <input
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                autoComplete="current-password"
                className="h-11 w-full rounded-xl border border-gray-200 bg-white px-3 text-sm outline-none transition focus:border-blue-300 dark:border-white/[0.08] dark:bg-gray-950 dark:text-gray-100 dark:focus:border-blue-500/60"
              />
            </label>
            {error && (
              <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-100">
                {error}
              </div>
            )}
            <button
              type="submit"
              disabled={busy}
              className="h-11 w-full rounded-xl bg-blue-600 px-4 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {busy ? '正在登录...' : '进入控制台'}
            </button>
            <p className="text-xs leading-5 text-gray-400 dark:text-gray-500">
              这里使用的是 Web 控制台 Admin 登录，不是 provider key，也不是 Project API Key。
            </p>
          </form>
        )}
      </section>
    </main>
  )
}

