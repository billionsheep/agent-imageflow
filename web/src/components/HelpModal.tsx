import { useState, useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import type { AppMode } from '../types'
import { useCloseOnEscape } from '../hooks/useCloseOnEscape'
import { usePreventBackgroundScroll } from '../hooks/usePreventBackgroundScroll'

interface HelpModalProps {
  appMode: AppMode
  isFavoriteCollectionOverview?: boolean
  onClose: () => void
}

function useIsMobile() {
  const [isMobile, setIsMobile] = useState(window.innerWidth < 640)
  useEffect(() => {
    const onResize = () => setIsMobile(window.innerWidth < 640)
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])
  return isMobile
}

export default function HelpModal({ appMode, isFavoriteCollectionOverview = false, onClose }: HelpModalProps) {
  const isMobile = useIsMobile()
  const modalRef = useRef<HTMLDivElement>(null)
  const isAgentMode = appMode === 'agent'
  useCloseOnEscape(true, onClose)
  usePreventBackgroundScroll(true, modalRef)

  return createPortal(
    <div
      data-no-drag-select
      className="fixed inset-0 z-[100] flex items-center justify-center p-4"
      onClick={onClose}
    >
      <div className="absolute inset-0 bg-black/30 backdrop-blur-sm animate-overlay-in" />
      <div
        ref={modalRef}
        className="relative z-10 w-full max-w-md rounded-3xl border border-white/50 bg-white/95 p-5 shadow-2xl ring-1 ring-black/5 animate-modal-in dark:border-white/[0.08] dark:bg-gray-900/95 dark:ring-white/10 flex flex-col max-h-[85vh] custom-scrollbar"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-5 flex items-center justify-between gap-4">
          <h3 className="text-base font-semibold text-gray-800 dark:text-gray-100 flex items-center gap-2">
            <svg className="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round" viewBox="0 0 24 24">
              <circle cx="12" cy="12" r="10" />
              <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
              <path d="M12 17h.01" />
            </svg>
            操作指南
          </h3>
          <div className="flex items-center gap-3">
            <button
              onClick={onClose}
              className="rounded-full p-1 text-gray-400 transition hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-white/[0.06] dark:hover:text-gray-200"
              aria-label="关闭"
            >
              <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto overscroll-contain mb-6 text-sm text-gray-600 dark:text-gray-300 space-y-6 custom-scrollbar pr-2">
          <section>
            <h4 className="mb-4 text-sm font-medium text-gray-800 dark:text-gray-200 flex items-center gap-1.5">
              <svg className="w-4 h-4 text-gray-400 dark:text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h7" />
              </svg>
              Agent ImageFlow 控制台
            </h4>
            <div className="space-y-4">
              <ul className="list-disc pl-4 space-y-2">
                <li>登录后进入控制台，先确认当前 workspace、project、campaign，确保任务和资产进入正确业务空间。</li>
                <li>优先使用服务端托管生图：Web 创建服务端 ImageTask，由 Worker 调用 provider/model，并保存原图、缩略图和 metadata。</li>
                <li>服务端会返回可追踪的 `task_id` 和 `asset_id`；这些 ID 可用于 API 查询、复用、交付、manifest 或外部系统引用。</li>
                <li>MCP/REST/CLI/API 创建的资产会出现在最近资产，可继续查看、选中、拒绝、打开原图或 metadata。</li>
              </ul>
            </div>
          </section>

          <section>
            <h4 className="mb-4 text-sm font-medium text-gray-800 dark:text-gray-200 flex items-center gap-1.5">
              <svg className="w-4 h-4 text-gray-400 dark:text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              批次生产与视觉上下文
            </h4>
            <div className="space-y-4">
              <ul className="list-disc pl-4 space-y-2">
                <li>批次生产视图用于按 story/scene 查看一批资产，检查每个画面是否已有选中图，并对单张候选执行选中/拒绝。</li>
                <li>项目视觉上下文用于维护角色/主形象、参考图和 Prompt Recipe；创建任务时可复用这些上下文，让同一 project 的视觉风格更稳定。</li>
                <li>选中表示推荐使用，拒绝表示排除候选；这是轻量选优状态，不是强制人工审核闸门。</li>
              </ul>
            </div>
          </section>

          {(isAgentMode || isFavoriteCollectionOverview || isMobile) && (
            <section>
              <h4 className="mb-4 text-sm font-medium text-gray-800 dark:text-gray-200 flex items-center gap-1.5">
                <svg className="w-4 h-4 text-gray-400 dark:text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v3.75m0 3.75h.01M4.93 19.07a10 10 0 1114.14 0 10 10 0 01-14.14 0z" />
                </svg>
                高级/旧模式
              </h4>
              <div className="space-y-4">
                <p>Agent 对话、浏览器本地任务多选、收藏夹和下载选中来自参考项目的 playground 能力，适合临时整理或探索。正式生产与交付请优先使用服务端资产、最近资产、批次生产视图和 JSON manifest。</p>
              </div>
            </section>
          )}
        </div>

        <div className="pt-4 border-t border-gray-200 dark:border-white/[0.08] flex justify-center">
          <a
            href="https://github.com/billionsheep/agent-imageflow"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 text-sm font-medium text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100 transition-colors group"
          >
            <svg className="w-5 h-5 group-hover:scale-110 transition-transform" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
            </svg>
            Agent ImageFlow
          </a>
        </div>
      </div>
    </div>,
    document.body
  )
}
