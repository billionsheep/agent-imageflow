import { lazy, Suspense, useEffect } from 'react'
import { initStore } from './store'
import { useStore } from './store'
import { activateFirstImportedProfile, buildSettingsFromUrlParams, clearUrlSettingParams, hasUrlSettingParams } from './lib/urlSettings'
import { isDefaultConfigOnlyEnabled, mergeImportedSettings } from './lib/apiProfiles'
import { getCustomProviderConfigUrl, loadCustomProviderSettingsFromUrl } from './lib/customProviderConfigUrl'
import { useDockerApiUrlMigrationNotice } from './hooks/useDockerApiUrlMigrationNotice'
import type { AppSettings } from './types'
import Header from './components/Header'
import SearchBar from './components/SearchBar'
import ServerAssetLibrary from './components/ServerAssetLibrary'
import TaskGrid from './components/TaskGrid'
import InputBar from './components/InputBar'
import ConfirmDialog from './components/ConfirmDialog'
import Toast from './components/Toast'
import ImageContextMenu from './components/ImageContextMenu'
import SupportPromptModal from './components/SupportPromptModal'
import { FavoriteCollectionPickerModal, FavoriteCollectionsView, ManageCollectionsModal } from './components/FavoriteCollections'
import { useGlobalClickSuppression } from './lib/clickSuppression'
import {
  loadAgentWorkspace,
  loadDetailModal,
  loadLightbox,
  loadMaskEditorModal,
  loadProjectContextModal,
  loadProductionViewModal,
  loadScopeManagerModal,
  loadSettingsModal,
} from './lib/lazyModules'

const AgentWorkspace = lazy(loadAgentWorkspace)
const DetailModal = lazy(loadDetailModal)
const Lightbox = lazy(loadLightbox)
const SettingsModal = lazy(loadSettingsModal)
const ScopeManagerModal = lazy(loadScopeManagerModal)
const ProjectContextModal = lazy(loadProjectContextModal)
const ProductionViewModal = lazy(loadProductionViewModal)
const MaskEditorModal = lazy(loadMaskEditorModal)

let customProviderConfigUrlImportStarted = false

function LazyModalFallback({ variant = 'panel' }: { variant?: 'panel' | 'lightbox' | 'fullscreen' }) {
  if (variant === 'lightbox') {
    return (
      <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/80 p-4 backdrop-blur-sm" role="status" aria-label="Loading">
        <div className="h-[min(72vh,720px)] w-full max-w-5xl rounded-lg bg-white/10 shadow-2xl ring-1 ring-white/15">
          <div className="h-full w-full animate-pulse rounded-lg bg-white/10" />
        </div>
      </div>
    )
  }

  if (variant === 'fullscreen') {
    return (
      <div className="fixed inset-0 z-[80] bg-gray-50 p-4 dark:bg-gray-900" role="status" aria-label="Loading">
        <div className="mx-auto flex h-full max-w-6xl flex-col gap-4">
          <div className="h-12 rounded-lg bg-gray-200/80 dark:bg-white/[0.08]" />
          <div className="grid flex-1 grid-cols-[minmax(0,1fr)_180px] gap-4 max-md:grid-cols-1">
            <div className="animate-pulse rounded-lg bg-gray-200/70 dark:bg-white/[0.06]" />
            <div className="animate-pulse rounded-lg bg-gray-200/70 dark:bg-white/[0.06]" />
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 z-[110] flex items-center justify-center bg-black/30 p-4 backdrop-blur-sm" role="status" aria-label="Loading">
      <div className="w-full max-w-4xl overflow-hidden rounded-lg border border-gray-200/80 bg-white shadow-2xl ring-1 ring-black/5 dark:border-white/[0.08] dark:bg-gray-950 dark:ring-white/10">
        <div className="flex h-14 items-center justify-between border-b border-gray-100 px-4 dark:border-white/[0.08]">
          <div className="h-3 w-36 animate-pulse rounded-full bg-gray-200 dark:bg-white/[0.08]" />
          <div className="h-8 w-8 animate-pulse rounded-lg bg-gray-100 dark:bg-white/[0.06]" />
        </div>
        <div className="grid min-h-[min(68vh,620px)] grid-cols-[220px_minmax(0,1fr)] gap-0 max-sm:grid-cols-1">
          <div className="space-y-3 border-r border-gray-100 p-4 dark:border-white/[0.08] max-sm:hidden">
            <div className="h-9 animate-pulse rounded-lg bg-gray-100 dark:bg-white/[0.06]" />
            <div className="h-9 animate-pulse rounded-lg bg-gray-100 dark:bg-white/[0.06]" />
            <div className="h-9 animate-pulse rounded-lg bg-gray-100 dark:bg-white/[0.06]" />
          </div>
          <div className="space-y-4 p-4">
            <div className="h-24 animate-pulse rounded-lg bg-gray-100 dark:bg-white/[0.06]" />
            <div className="grid grid-cols-2 gap-3 max-sm:grid-cols-1">
              <div className="h-28 animate-pulse rounded-lg bg-gray-100 dark:bg-white/[0.06]" />
              <div className="h-28 animate-pulse rounded-lg bg-gray-100 dark:bg-white/[0.06]" />
            </div>
            <div className="h-36 animate-pulse rounded-lg bg-gray-100 dark:bg-white/[0.06]" />
          </div>
        </div>
      </div>
    </div>
  )
}

export default function App() {
  const setSettings = useStore((s) => s.setSettings)
  const appMode = useStore((s) => s.appMode)
  const filterFavorite = useStore((s) => s.filterFavorite)
  const activeFavoriteCollectionId = useStore((s) => s.activeFavoriteCollectionId)
  const detailTaskId = useStore((s) => s.detailTaskId)
  const lightboxImageId = useStore((s) => s.lightboxImageId)
  const showSettings = useStore((s) => s.showSettings)
  const showScopeManager = useStore((s) => s.showScopeManager)
  const showProjectContext = useStore((s) => s.showProjectContext)
  const showProductionView = useStore((s) => s.showProductionView)
  const maskEditorImageId = useStore((s) => s.maskEditorImageId)
  useDockerApiUrlMigrationNotice()
  useGlobalClickSuppression()

  useEffect(() => {
    const searchParams = new URLSearchParams(window.location.search)
    const customProviderConfigUrl = getCustomProviderConfigUrl()
    const defaultConfigOnly = isDefaultConfigOnlyEnabled()

    const applyUrlSettings = (baseSettings: Partial<AppSettings>) => {
      const nextSettings = buildSettingsFromUrlParams(baseSettings, searchParams)
      return Object.keys(nextSettings).length ? nextSettings : baseSettings
    }

    const clearAppliedUrlSettings = () => {
      if (!hasUrlSettingParams(searchParams)) return

      clearUrlSettingParams(searchParams)

      const nextSearch = searchParams.toString()
      const nextUrl = `${window.location.pathname}${nextSearch ? `?${nextSearch}` : ''}${window.location.hash}`
      window.history.replaceState(null, '', nextUrl)
    }

    if (customProviderConfigUrl && defaultConfigOnly && !customProviderConfigUrlImportStarted) {
      customProviderConfigUrlImportStarted = true
      void loadCustomProviderSettingsFromUrl(customProviderConfigUrl)
        .then((importedSettings) => {
          const state = useStore.getState()
          const baseSettings = importedSettings
            ? activateFirstImportedProfile(mergeImportedSettings(state.settings, importedSettings), importedSettings)
            : state.settings
          state.setSettings(applyUrlSettings(baseSettings))
          clearAppliedUrlSettings()
        })
        .catch((error) => {
          console.warn('Failed to import custom provider config URL:', error)
          const state = useStore.getState()
          state.setSettings(applyUrlSettings(state.settings))
          clearAppliedUrlSettings()
        })

      initStore()
      return
    }

    const nextSettings = buildSettingsFromUrlParams(useStore.getState().settings, searchParams)

    setSettings(nextSettings)

    clearAppliedUrlSettings()

    if (customProviderConfigUrl && !customProviderConfigUrlImportStarted) {
      customProviderConfigUrlImportStarted = true
      void loadCustomProviderSettingsFromUrl(customProviderConfigUrl)
        .then((importedSettings) => {
          if (!importedSettings) return
          const state = useStore.getState()
          state.setSettings(mergeImportedSettings(state.settings, importedSettings))
        })
        .catch((error) => {
          console.warn('Failed to import custom provider config URL:', error)
        })
    }

    initStore()
  }, [setSettings])

  useEffect(() => {
    const preventPageImageDrag = (e: DragEvent) => {
      if ((e.target as HTMLElement | null)?.closest('img')) {
        e.preventDefault()
      }
    }

    document.addEventListener('dragstart', preventPageImageDrag)
    return () => document.removeEventListener('dragstart', preventPageImageDrag)
  }, [])

  return (
    <>
      <Header />
      {appMode === 'agent' ? (
        <Suspense fallback={<LazyModalFallback variant="fullscreen" />}>
          <AgentWorkspace />
        </Suspense>
      ) : (
        <main data-home-main data-drag-select-surface className="pb-48">
          <div className="safe-area-x max-w-7xl mx-auto">
            <SearchBar />
            <ServerAssetLibrary />
            {filterFavorite && !activeFavoriteCollectionId ? <FavoriteCollectionsView /> : <TaskGrid />}
          </div>
        </main>
      )}
      <InputBar />
      {detailTaskId && (
        <Suspense fallback={<LazyModalFallback />}>
          <DetailModal />
        </Suspense>
      )}
      {lightboxImageId && (
        <Suspense fallback={<LazyModalFallback variant="lightbox" />}>
          <Lightbox />
        </Suspense>
      )}
      {showSettings && (
        <Suspense fallback={<LazyModalFallback />}>
          <SettingsModal />
        </Suspense>
      )}
      {showScopeManager && (
        <Suspense fallback={<LazyModalFallback />}>
          <ScopeManagerModal />
        </Suspense>
      )}
      {showProjectContext && (
        <Suspense fallback={<LazyModalFallback />}>
          <ProjectContextModal />
        </Suspense>
      )}
      {showProductionView && (
        <Suspense fallback={<LazyModalFallback />}>
          <ProductionViewModal />
        </Suspense>
      )}
      <ConfirmDialog />
      <SupportPromptModal />
      <FavoriteCollectionPickerModal />
      <ManageCollectionsModal />
      <Toast />
      {maskEditorImageId && (
        <Suspense fallback={<LazyModalFallback variant="fullscreen" />}>
          <MaskEditorModal />
        </Suspense>
      )}
      <ImageContextMenu />
    </>
  )
}
