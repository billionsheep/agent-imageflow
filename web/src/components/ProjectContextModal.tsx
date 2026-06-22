import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { useStore } from '../store'
import { useCloseOnEscape } from '../hooks/useCloseOnEscape'
import { usePreventBackgroundScroll } from '../hooks/usePreventBackgroundScroll'
import {
  getAgentImageflowProjectVisualContext,
  isAgentImageflowUnauthorizedError,
  normalizeAgentImageflowApiBaseUrl,
  resolveAgentImageflowDeliveryUrl,
  updateAgentImageflowProjectVisualContext,
  type AgentImageflowAuth,
  type AgentImageflowCharacterProfile,
  type AgentImageflowProjectReferenceBinding,
  type AgentImageflowProjectVisualContext,
  type AgentImageflowPromptBlock,
  type AgentImageflowPromptRecipe,
  type AgentImageflowReferencePurpose,
} from '../lib/agentImageflowApi'
import { ArchiveIcon, CloseIcon, PlusIcon, RefreshIcon } from './icons'

type ProjectContextTab = 'characters' | 'references' | 'recipes'

interface CharacterDraft {
  id: string
  name: string
  status: string
  role: string
  appearance: string
  personality: string
  forbiddenText: string
  primaryAssetId: string
  referenceAssetIdsText: string
  referencePolicy: string
  appearanceLockNotes: string
}

interface ReferenceDraft {
  assetId: string
  purpose: AgentImageflowReferencePurpose
  label: string
  characterId: string
  weight: string
  notes: string
}

interface RecipeDraft {
  id: string
  name: string
  status: string
  characterBlock: string
  styleBlock: string
  cameraBlock: string
  channelBlock: string
  negativePrompt: string
  defaultAspectRatio: string
  defaultOutputFormat: string
  defaultProvider: string
  defaultModel: string
  generationConfigText: string
}

const EMPTY_CONTEXT: Required<Pick<AgentImageflowProjectVisualContext, 'characters' | 'references' | 'prompt_recipes'>> = {
  characters: [],
  references: [],
  prompt_recipes: [],
}

const EMPTY_CHARACTER_DRAFT: CharacterDraft = {
  id: '',
  name: '',
  status: 'active',
  role: '',
  appearance: '',
  personality: '',
  forbiddenText: '',
  primaryAssetId: '',
  referenceAssetIdsText: '',
  referencePolicy: 'primary_plus_references',
  appearanceLockNotes: '',
}

const EMPTY_REFERENCE_DRAFT: ReferenceDraft = {
  assetId: '',
  purpose: 'style',
  label: '',
  characterId: '',
  weight: '1',
  notes: '',
}

const EMPTY_RECIPE_DRAFT: RecipeDraft = {
  id: '',
  name: '',
  status: 'active',
  characterBlock: '',
  styleBlock: '',
  cameraBlock: '',
  channelBlock: '',
  negativePrompt: '',
  defaultAspectRatio: '',
  defaultOutputFormat: 'png',
  defaultProvider: '',
  defaultModel: '',
  generationConfigText: '{}',
}

function buildConsoleAuth(settings: { imageflowBasicUsername: string; imageflowBasicPassword: string }): AgentImageflowAuth {
  return {
    basicUsername: settings.imageflowBasicUsername,
    basicPassword: settings.imageflowBasicPassword,
  }
}

function normalizeContext(context: AgentImageflowProjectVisualContext | null) {
  return {
    ...context,
    characters: context?.characters ?? EMPTY_CONTEXT.characters,
    references: context?.references ?? EMPTY_CONTEXT.references,
    prompt_recipes: context?.prompt_recipes ?? EMPTY_CONTEXT.prompt_recipes,
  }
}

function splitList(text: string): string[] {
  const seen = new Set<string>()
  return text
    .split(/[\n,]+/)
    .map((item) => item.trim())
    .filter((item) => {
      if (!item || seen.has(item)) return false
      seen.add(item)
      return true
    })
}

function joinList(values?: string[]): string {
  return (values ?? []).join('\n')
}

function slugPart(value: string): string {
  return value.trim().toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_+|_+$/g, '')
}

function createVisualContextId(prefix: string, seed: string): string {
  return `${prefix}_${slugPart(seed) || Date.now().toString(36)}`
}

function formatDate(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function compactText(value?: string): string {
  return value?.trim() || '-'
}

function activeItems<T extends { status?: string }>(items: T[]): T[] {
  return items.filter((item) => item.status !== 'archived')
}

function compactListPreview(values: string[]): string {
  const cleaned = values.map((value) => value.trim()).filter(Boolean)
  if (cleaned.length === 0) return '-'
  const visible = cleaned.slice(0, 3).join(', ')
  return cleaned.length > 3 ? `${visible} +${cleaned.length - 3}` : visible
}

function parseJsonObject(text: string, label: string): Record<string, unknown> | undefined {
  const trimmed = text.trim()
  if (!trimmed || trimmed === '{}') return undefined
  const parsed = JSON.parse(trimmed)
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error(`${label} 必须是 JSON object`)
  }
  return parsed as Record<string, unknown>
}

function stringifyJsonObject(value?: Record<string, unknown>): string {
  if (!value || Object.keys(value).length === 0) return '{}'
  return JSON.stringify(value, null, 2)
}

function characterToDraft(character: AgentImageflowCharacterProfile): CharacterDraft {
  return {
    id: character.id,
    name: character.name,
    status: character.status || 'active',
    role: character.role ?? '',
    appearance: character.appearance ?? '',
    personality: character.personality ?? '',
    forbiddenText: joinList(character.forbidden),
    primaryAssetId: character.primary_asset_id ?? '',
    referenceAssetIdsText: joinList(character.reference_asset_ids),
    referencePolicy: character.reference_policy ?? 'primary_plus_references',
    appearanceLockNotes: character.appearance_lock_notes ?? '',
  }
}

function recipeToDraft(recipe: AgentImageflowPromptRecipe): RecipeDraft {
  const blocks = recipe.prompt_blocks ?? []
  const byRole = (roles: string[]) => blocks.find((block) => roles.includes((block.role ?? '').toLowerCase()))?.text ?? ''
  return {
    id: recipe.id,
    name: recipe.name,
    status: recipe.status || 'active',
    characterBlock: byRole(['character', 'characters']),
    styleBlock: byRole(['style']),
    cameraBlock: byRole(['camera', 'lens', 'shot']),
    channelBlock: byRole(['channel']),
    negativePrompt: recipe.negative_prompt ?? '',
    defaultAspectRatio: recipe.default_aspect_ratio ?? '',
    defaultOutputFormat: recipe.default_output_format ?? 'png',
    defaultProvider: recipe.default_provider ?? '',
    defaultModel: recipe.default_model ?? '',
    generationConfigText: stringifyJsonObject(recipe.generation_config),
  }
}

function draftToPromptBlocks(draft: RecipeDraft): AgentImageflowPromptBlock[] {
  return [
    { role: 'character', text: draft.characterBlock.trim() },
    { role: 'style', text: draft.styleBlock.trim() },
    { role: 'camera', text: draft.cameraBlock.trim() },
    { role: 'channel', text: draft.channelBlock.trim() },
  ].filter((block) => block.text)
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="min-w-0 text-[11px] font-medium uppercase text-gray-500 dark:text-gray-400">
      <span className="mb-1 block">{label}</span>
      {children}
    </label>
  )
}

const inputClass = 'w-full min-w-0 rounded-lg border border-gray-200/70 bg-white/80 px-2.5 py-2 text-xs text-gray-700 outline-none transition focus:border-blue-300 disabled:cursor-not-allowed disabled:opacity-60 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-100 dark:focus:border-blue-500/50'
const textareaClass = `${inputClass} min-h-20 resize-y whitespace-pre-wrap break-words`

function SummaryPill({ label, value }: { label: string; value: number | string }) {
  return (
    <span className="rounded-full border border-gray-200 bg-gray-50 px-2.5 py-1 text-[11px] text-gray-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">
      {label}: {value}
    </span>
  )
}

function AssetThumb({ assetId, baseUrl, label }: { assetId?: string; baseUrl: string; label: string }) {
  const id = assetId?.trim()
  if (!id) {
    return (
      <div className="flex aspect-square min-w-0 items-center justify-center rounded-lg border border-dashed border-amber-200 bg-amber-50 px-2 text-center text-[10px] text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-200">
        缺少{label}
      </div>
    )
  }
  return (
    <div className="min-w-0 overflow-hidden rounded-lg border border-gray-200 bg-gray-50 dark:border-white/[0.08] dark:bg-white/[0.04]">
      <div className="aspect-square bg-gray-100 dark:bg-white/[0.04]">
        <img
          src={resolveAgentImageflowDeliveryUrl(baseUrl, `/api/assets/${encodeURIComponent(id)}/thumbnail`)}
          alt={`${label} ${id}`}
          loading="lazy"
          className="h-full w-full object-cover"
        />
      </div>
      <div className="truncate px-2 py-1 text-[10px] text-gray-500 dark:text-gray-400" title={id}>
        {label}: {id}
      </div>
    </div>
  )
}

export default function ProjectContextModal() {
  const open = useStore((state) => state.showProjectContext)
  const referenceAssetId = useStore((state) => state.projectContextReferenceAssetId)
  const setShowProjectContext = useStore((state) => state.setShowProjectContext)
  const settings = useStore((state) => state.settings)
  const showToast = useStore((state) => state.showToast)
  const modalRef = useRef<HTMLDivElement>(null)

  const baseUrl = useMemo(() => normalizeAgentImageflowApiBaseUrl(settings.imageflowApiBaseUrl), [settings.imageflowApiBaseUrl])
  const auth = useMemo(() => buildConsoleAuth(settings), [settings])
  const scope = useMemo(() => ({
    workspaceId: settings.imageflowWorkspaceId.trim(),
    projectId: settings.imageflowProjectId.trim(),
  }), [settings.imageflowProjectId, settings.imageflowWorkspaceId])
  const scopeReady = Boolean(scope.workspaceId && scope.projectId)

  const [context, setContext] = useState<AgentImageflowProjectVisualContext | null>(null)
  const normalizedContext = useMemo(() => normalizeContext(context), [context])
  const [activeTab, setActiveTab] = useState<ProjectContextTab>('characters')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [unauthorized, setUnauthorized] = useState(false)
  const [editingCharacterId, setEditingCharacterId] = useState<string | null>(null)
  const [characterDraft, setCharacterDraft] = useState<CharacterDraft>(EMPTY_CHARACTER_DRAFT)
  const [referenceDraft, setReferenceDraft] = useState<ReferenceDraft>(EMPTY_REFERENCE_DRAFT)
  const [editingRecipeId, setEditingRecipeId] = useState<string | null>(null)
  const [recipeDraft, setRecipeDraft] = useState<RecipeDraft>(EMPTY_RECIPE_DRAFT)
  const activeCharacters = useMemo(() => activeItems(normalizedContext.characters), [normalizedContext.characters])
  const activeReferences = useMemo(() => activeItems(normalizedContext.references), [normalizedContext.references])
  const activeRecipes = useMemo(() => activeItems(normalizedContext.prompt_recipes), [normalizedContext.prompt_recipes])
  const overviewRows = useMemo(() => [
    {
      label: '角色卡',
      value: compactListPreview(activeCharacters.map((character) => character.name || character.id)),
    },
    {
      label: '参考图',
      value: compactListPreview(activeReferences.map((reference) => reference.label || `${reference.purpose}:${reference.asset_id}`)),
    },
    {
      label: 'Prompt 配方',
      value: compactListPreview(activeRecipes.map((recipe) => recipe.name || recipe.id)),
    },
  ], [activeCharacters, activeReferences, activeRecipes])

  const close = useCallback(() => setShowProjectContext(false), [setShowProjectContext])
  useCloseOnEscape(open, close)
  usePreventBackgroundScroll(open, modalRef)

  const fetchContext = useCallback(async () => {
    return getAgentImageflowProjectVisualContext(baseUrl, scope, auth)
  }, [auth, baseUrl, scope])

  const loadContext = useCallback(async () => {
    if (!scopeReady) {
      setContext(null)
      setUnauthorized(false)
      setError(null)
      return
    }
    setLoading(true)
    setError(null)
    try {
      const response = await fetchContext()
      setContext(response.visual_context)
      setUnauthorized(false)
    } catch (nextError) {
      if (isAgentImageflowUnauthorizedError(nextError)) {
        setUnauthorized(true)
        setError(null)
      } else {
        setError(nextError instanceof Error ? nextError.message : String(nextError))
      }
    } finally {
      setLoading(false)
    }
  }, [fetchContext, scopeReady])

  useEffect(() => {
    if (!open) return
    void loadContext()
  }, [loadContext, open])

  useEffect(() => {
    if (!open || !referenceAssetId) return
    setActiveTab('references')
    setReferenceDraft((current) => ({
      ...current,
      assetId: referenceAssetId,
      label: current.label || referenceAssetId,
    }))
  }, [open, referenceAssetId])

  const saveContext = useCallback(async (nextContext: AgentImageflowProjectVisualContext, successMessage: string) => {
    if (!scopeReady) {
      showToast('请先选择完整 workspace / project', 'error')
      return
    }
    setSaving(true)
    setError(null)
    try {
      await updateAgentImageflowProjectVisualContext(baseUrl, scope, nextContext, auth)
      const reloaded = await fetchContext()
      setContext(reloaded.visual_context)
      setUnauthorized(false)
      showToast(successMessage, 'success')
    } catch (nextError) {
      if (isAgentImageflowUnauthorizedError(nextError)) {
        setUnauthorized(true)
      }
      const message = nextError instanceof Error ? nextError.message : String(nextError)
      setError(message)
      showToast(message, 'error')
    } finally {
      setSaving(false)
    }
  }, [auth, baseUrl, fetchContext, scope, scopeReady, showToast])

  const handleCharacterSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const id = characterDraft.id.trim() || createVisualContextId('char', characterDraft.name)
    const duplicate = normalizedContext.characters.some((item) => item.id === id && item.id !== editingCharacterId)
    if (duplicate) {
      showToast(`角色 id 已存在：${id}`, 'error')
      return
    }
    const nextCharacter: AgentImageflowCharacterProfile = {
      id,
      name: characterDraft.name.trim() || id,
      status: characterDraft.status || 'active',
      role: characterDraft.role.trim() || undefined,
      appearance: characterDraft.appearance.trim() || undefined,
      personality: characterDraft.personality.trim() || undefined,
      forbidden: splitList(characterDraft.forbiddenText),
      primary_asset_id: characterDraft.primaryAssetId.trim() || undefined,
      reference_asset_ids: splitList(characterDraft.referenceAssetIdsText),
      reference_policy: characterDraft.referencePolicy.trim() || undefined,
      appearance_lock_notes: characterDraft.appearanceLockNotes.trim() || undefined,
    }
    const nextCharacters = editingCharacterId
      ? normalizedContext.characters.map((item) => item.id === editingCharacterId ? nextCharacter : item)
      : [...normalizedContext.characters, nextCharacter]
    void saveContext({ ...normalizedContext, characters: nextCharacters }, '角色卡已保存')
    setEditingCharacterId(null)
    setCharacterDraft(EMPTY_CHARACTER_DRAFT)
  }

  const archiveCharacter = (character: AgentImageflowCharacterProfile) => {
    const nextCharacters = normalizedContext.characters.map((item) =>
      item.id === character.id ? { ...item, status: item.status === 'archived' ? 'active' : 'archived' } : item,
    )
    void saveContext({ ...normalizedContext, characters: nextCharacters }, character.status === 'archived' ? '角色卡已恢复' : '角色卡已归档')
  }

  const handleReferenceSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const assetId = referenceDraft.assetId.trim()
    if (!assetId) {
      showToast('请输入 asset_id', 'error')
      return
    }
    const weight = Number(referenceDraft.weight)
    const normalizedWeight = Number.isFinite(weight) ? Math.min(5, Math.max(0.1, weight)) : 1
    const existing = normalizedContext.references.find((item) =>
      item.asset_id === assetId &&
      item.purpose === referenceDraft.purpose &&
      (item.character_id ?? '') === referenceDraft.characterId.trim(),
    )
    const nextReference: AgentImageflowProjectReferenceBinding = {
      id: existing?.id ?? createVisualContextId(`ref_${referenceDraft.purpose}`, assetId),
      asset_id: assetId,
      purpose: referenceDraft.purpose,
      label: referenceDraft.label.trim() || undefined,
      character_id: referenceDraft.characterId.trim() || undefined,
      weight: normalizedWeight,
      notes: referenceDraft.notes.trim() || undefined,
      status: 'active',
    }
    const nextReferences = existing
      ? normalizedContext.references.map((item) => item.id === existing.id ? nextReference : item)
      : [...normalizedContext.references, nextReference]
    void saveContext({ ...normalizedContext, references: nextReferences }, '参考图绑定已保存')
    setReferenceDraft({ ...EMPTY_REFERENCE_DRAFT, assetId: referenceAssetId ?? '' })
  }

  const archiveReference = (reference: AgentImageflowProjectReferenceBinding) => {
    const nextReferences = normalizedContext.references.map((item) =>
      item.id === reference.id ? { ...item, status: item.status === 'archived' ? 'active' : 'archived' } : item,
    )
    void saveContext({ ...normalizedContext, references: nextReferences }, reference.status === 'archived' ? '参考图绑定已恢复' : '参考图绑定已归档')
  }

  const handleRecipeSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const id = recipeDraft.id.trim() || createVisualContextId('recipe', recipeDraft.name)
    const duplicate = normalizedContext.prompt_recipes.some((item) => item.id === id && item.id !== editingRecipeId)
    if (duplicate) {
      showToast(`recipe id 已存在：${id}`, 'error')
      return
    }
    let generationConfig: Record<string, unknown> | undefined
    try {
      generationConfig = parseJsonObject(recipeDraft.generationConfigText, 'generation_config')
    } catch (nextError) {
      showToast(nextError instanceof Error ? nextError.message : String(nextError), 'error')
      return
    }
    const nextRecipe: AgentImageflowPromptRecipe = {
      id,
      name: recipeDraft.name.trim() || id,
      status: recipeDraft.status || 'active',
      prompt_blocks: draftToPromptBlocks(recipeDraft),
      negative_prompt: recipeDraft.negativePrompt.trim() || undefined,
      default_aspect_ratio: recipeDraft.defaultAspectRatio.trim() || undefined,
      default_output_format: recipeDraft.defaultOutputFormat.trim() || undefined,
      default_provider: recipeDraft.defaultProvider.trim() || undefined,
      default_model: recipeDraft.defaultModel.trim() || undefined,
      generation_config: generationConfig,
    }
    const nextRecipes = editingRecipeId
      ? normalizedContext.prompt_recipes.map((item) => item.id === editingRecipeId ? nextRecipe : item)
      : [...normalizedContext.prompt_recipes, nextRecipe]
    void saveContext({ ...normalizedContext, prompt_recipes: nextRecipes }, 'Prompt Recipe 已保存')
    setEditingRecipeId(null)
    setRecipeDraft(EMPTY_RECIPE_DRAFT)
  }

  const archiveRecipe = (recipe: AgentImageflowPromptRecipe) => {
    const nextRecipes = normalizedContext.prompt_recipes.map((item) =>
      item.id === recipe.id ? { ...item, status: item.status === 'archived' ? 'active' : 'archived' } : item,
    )
    void saveContext({ ...normalizedContext, prompt_recipes: nextRecipes }, recipe.status === 'archived' ? 'Prompt Recipe 已恢复' : 'Prompt Recipe 已归档')
  }

  if (!open) return null

  return createPortal(
    <div data-no-drag-select className="fixed inset-0 z-[110] flex items-center justify-center p-4" onClick={close}>
      <div className="absolute inset-0 bg-black/35 backdrop-blur-sm animate-overlay-in" />
      <div
        ref={modalRef}
        className="relative z-10 flex h-[88vh] w-full max-w-6xl flex-col overflow-hidden rounded-3xl border border-white/50 bg-white/95 shadow-2xl ring-1 ring-black/5 animate-modal-in dark:border-white/[0.08] dark:bg-gray-900/95 dark:ring-white/10"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex shrink-0 items-start justify-between gap-3 border-b border-gray-200/70 px-4 py-3 dark:border-white/[0.08]">
          <div className="min-w-0">
            <div className="text-sm font-semibold text-gray-800 dark:text-gray-100">项目视觉上下文</div>
            <div className="mt-1 break-all text-xs text-gray-500 dark:text-gray-400">
              {scopeReady ? `${scope.workspaceId} / ${scope.projectId}` : '请选择 workspace / project'}
            </div>
            <div className="mt-2 flex flex-wrap gap-2">
              <SummaryPill label="角色" value={normalizedContext.characters.length} />
              <SummaryPill label="参考图" value={normalizedContext.references.length} />
              <SummaryPill label="配方" value={normalizedContext.prompt_recipes.length} />
              <SummaryPill label="更新" value={formatDate(context?.updated_at)} />
            </div>
            {scopeReady && (
              <div className="mt-3 grid gap-2 sm:grid-cols-3">
                {overviewRows.map((row) => (
                  <div
                    key={row.label}
                    className="min-w-0 rounded-xl border border-gray-200/70 bg-gray-50/70 px-3 py-2 text-xs dark:border-white/[0.08] dark:bg-white/[0.04]"
                  >
                    <div className="text-[10px] font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">{row.label}</div>
                    <div className="mt-1 truncate text-gray-700 dark:text-gray-200" title={row.value}>
                      {row.value}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <button
              type="button"
              onClick={() => void loadContext()}
              disabled={loading || saving || !scopeReady}
              className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-gray-200/80 bg-white px-3 text-xs font-medium text-gray-600 transition hover:border-blue-300 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300"
            >
              <RefreshIcon className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              Reload
            </button>
            <button
              type="button"
              onClick={close}
              className="rounded-full p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-white/[0.08] dark:hover:text-gray-200"
              aria-label="关闭"
            >
              <CloseIcon className="h-5 w-5" />
            </button>
          </div>
        </div>

        <div className="flex shrink-0 gap-2 overflow-x-auto border-b border-gray-200/70 bg-gray-50/70 px-4 py-2 dark:border-white/[0.08] dark:bg-white/[0.03]">
          {([
            ['characters', '角色卡'],
            ['references', '参考图'],
            ['recipes', 'Prompt 配方'],
          ] as const).map(([tab, label]) => (
            <button
              key={tab}
              type="button"
              onClick={() => setActiveTab(tab)}
              className={`h-9 whitespace-nowrap rounded-lg px-3 text-xs font-medium transition ${activeTab === tab ? 'bg-white text-blue-600 shadow-sm dark:bg-white/[0.08] dark:text-blue-300' : 'text-gray-500 hover:text-blue-600 dark:text-gray-300'}`}
            >
              {label}
            </button>
          ))}
          {saving && <span className="ml-auto self-center rounded-full border border-blue-200 bg-blue-50 px-2.5 py-1 text-[11px] text-blue-700 dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-200">保存中</span>}
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto p-4 custom-scrollbar">
          {!scopeReady ? (
            <div className="rounded-lg border border-dashed border-gray-200 px-4 py-8 text-center text-xs text-gray-500 dark:border-white/[0.08] dark:text-gray-400">
              当前没有完整 workspace / project。
            </div>
          ) : unauthorized ? (
            <div className="rounded-lg border border-amber-200 bg-amber-50/80 px-4 py-3 text-xs text-amber-800 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-100">
              未授权 / 需要登录。请重新登录控制台后重试。
            </div>
          ) : (
            <>
              {error && (
                <div className="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200">
                  {error}
                </div>
              )}
              {loading && context && (
                <div className="mb-3 rounded-lg border border-blue-200 bg-blue-50 px-3 py-2 text-xs text-blue-700 dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-200">
                  正在刷新，当前内容会保留到新结果返回。
                </div>
              )}

              {activeTab === 'characters' && (
                <div className="grid gap-4 lg:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
                  <form onSubmit={handleCharacterSubmit} className="space-y-3 rounded-lg border border-gray-200/80 bg-gray-50/60 p-3 dark:border-white/[0.08] dark:bg-white/[0.03]">
                    <div className="flex items-center justify-between gap-2">
                      <div className="text-xs font-semibold text-gray-700 dark:text-gray-200">{editingCharacterId ? '编辑角色卡' : '新建角色卡'}</div>
                      {editingCharacterId && (
                        <button type="button" onClick={() => { setEditingCharacterId(null); setCharacterDraft(EMPTY_CHARACTER_DRAFT) }} className="text-[11px] text-gray-500 hover:text-blue-600">
                          取消
                        </button>
                      )}
                    </div>
                    <div className="grid gap-2 sm:grid-cols-2">
                      <Field label="id"><input value={characterDraft.id} onChange={(event) => setCharacterDraft((draft) => ({ ...draft, id: event.target.value }))} className={inputClass} /></Field>
                      <Field label="名称"><input value={characterDraft.name} onChange={(event) => setCharacterDraft((draft) => ({ ...draft, name: event.target.value }))} className={inputClass} /></Field>
                      <Field label="状态">
                        <select value={characterDraft.status} onChange={(event) => setCharacterDraft((draft) => ({ ...draft, status: event.target.value }))} className={inputClass}>
                          <option value="active">启用</option>
                          <option value="archived">归档</option>
                        </select>
                      </Field>
                      <Field label="角色定位"><input value={characterDraft.role} onChange={(event) => setCharacterDraft((draft) => ({ ...draft, role: event.target.value }))} className={inputClass} /></Field>
                    </div>
                    <Field label="外观描述"><textarea value={characterDraft.appearance} onChange={(event) => setCharacterDraft((draft) => ({ ...draft, appearance: event.target.value }))} className={textareaClass} /></Field>
                    <Field label="性格描述"><textarea value={characterDraft.personality} onChange={(event) => setCharacterDraft((draft) => ({ ...draft, personality: event.target.value }))} className={textareaClass} /></Field>
                    <div className="grid gap-2 sm:grid-cols-2">
                      <Field label="主参考资产"><input value={characterDraft.primaryAssetId} onChange={(event) => setCharacterDraft((draft) => ({ ...draft, primaryAssetId: event.target.value }))} className={inputClass} /></Field>
                      <Field label="其他参考资产"><textarea value={characterDraft.referenceAssetIdsText} onChange={(event) => setCharacterDraft((draft) => ({ ...draft, referenceAssetIdsText: event.target.value }))} className={textareaClass} /></Field>
                    </div>
                    <div className="grid gap-2 sm:grid-cols-2">
                      <Field label="参考策略">
                        <select value={characterDraft.referencePolicy} onChange={(event) => setCharacterDraft((draft) => ({ ...draft, referencePolicy: event.target.value }))} className={inputClass}>
                          <option value="primary_plus_references">主图 + 参考图</option>
                          <option value="primary_only">仅主图</option>
                          <option value="references_only">仅参考图</option>
                        </select>
                      </Field>
                      <Field label="形象锁定"><textarea value={characterDraft.appearanceLockNotes} onChange={(event) => setCharacterDraft((draft) => ({ ...draft, appearanceLockNotes: event.target.value }))} className={textareaClass} /></Field>
                    </div>
                    <Field label="禁止项"><textarea value={characterDraft.forbiddenText} onChange={(event) => setCharacterDraft((draft) => ({ ...draft, forbiddenText: event.target.value }))} className={textareaClass} /></Field>
                    <button type="submit" disabled={saving} className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-blue-500 px-3 text-xs font-medium text-white transition hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-50">
                      <PlusIcon className="h-4 w-4" />
                      保存角色卡
                    </button>
                  </form>

                  <div className="space-y-3">
                    {normalizedContext.characters.length === 0 ? (
                      <div className="rounded-lg border border-dashed border-gray-200 px-4 py-8 text-center text-xs text-gray-500 dark:border-white/[0.08] dark:text-gray-400">暂无角色卡。</div>
                    ) : normalizedContext.characters.map((character) => (
                      <article key={character.id} className={`rounded-lg border p-3 ${character.status === 'archived' ? 'border-gray-200/70 bg-gray-50/60 opacity-70 dark:border-white/[0.08] dark:bg-white/[0.03]' : 'border-gray-200/80 bg-white dark:border-white/[0.08] dark:bg-gray-950/40'}`}>
                        <div className="flex flex-wrap items-start justify-between gap-2">
                          <div className="min-w-0">
                            <div className="break-words text-sm font-semibold text-gray-800 dark:text-gray-100">{character.name || character.id}</div>
                            <div className="mt-1 break-all text-[11px] text-gray-500 dark:text-gray-400">{character.id}</div>
                          </div>
                          <span className="rounded-full border border-gray-200 bg-gray-50 px-2 py-0.5 text-[10px] text-gray-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">{character.status || 'active'}</span>
                        </div>
                        <div className="mt-3 grid gap-2 text-xs text-gray-600 dark:text-gray-300">
                          <div className="break-words"><span className="text-gray-400">定位：</span> {compactText(character.role)}</div>
                          <div className="break-words"><span className="text-gray-400">外观：</span> {compactText(character.appearance)}</div>
                          <div className="break-words"><span className="text-gray-400">形象锁定：</span> {compactText(character.appearance_lock_notes)}</div>
                          <div className="break-words"><span className="text-gray-400">参考策略：</span> {compactText(character.reference_policy)}</div>
                          <div className="break-words"><span className="text-gray-400">禁止项：</span> {(character.forbidden ?? []).join(', ') || '-'}</div>
                          <div className="break-words"><span className="text-gray-400">参考资产：</span> {[character.primary_asset_id, ...(character.reference_asset_ids ?? [])].filter(Boolean).join(', ') || '-'}</div>
                        </div>
                        <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
                          <AssetThumb assetId={character.primary_asset_id} baseUrl={baseUrl} label="主图" />
                          {(character.reference_asset_ids ?? []).slice(0, 3).map((assetId, index) => (
                            <AssetThumb key={`${character.id}-${assetId}-${index}`} assetId={assetId} baseUrl={baseUrl} label={`参考 ${index + 1}`} />
                          ))}
                          {!character.primary_asset_id && (character.reference_asset_ids ?? []).length === 0 && (
                            <div className="col-span-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-[11px] text-amber-800 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-100 sm:col-span-3">
                              这个角色还没有绑定主图或参考图，生成时只能依赖文字描述。
                            </div>
                          )}
                        </div>
                        <div className="mt-3 flex flex-wrap gap-2">
                          <button type="button" onClick={() => { setEditingCharacterId(character.id); setCharacterDraft(characterToDraft(character)) }} className="h-8 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">编辑</button>
                          <button type="button" onClick={() => archiveCharacter(character)} disabled={saving} className="inline-flex h-8 items-center gap-1 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] text-gray-600 transition hover:border-amber-300 hover:text-amber-700 disabled:opacity-50 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">
                            <ArchiveIcon className="h-3.5 w-3.5" />
                            {character.status === 'archived' ? '恢复' : '归档'}
                          </button>
                        </div>
                      </article>
                    ))}
                  </div>
                </div>
              )}

              {activeTab === 'references' && (
                <div className="grid gap-4 lg:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
                  <form onSubmit={handleReferenceSubmit} className="space-y-3 rounded-lg border border-gray-200/80 bg-gray-50/60 p-3 dark:border-white/[0.08] dark:bg-white/[0.03]">
                    <div className="text-xs font-semibold text-gray-700 dark:text-gray-200">标记资产为参考图</div>
                    <Field label="asset_id"><input value={referenceDraft.assetId} onChange={(event) => setReferenceDraft((draft) => ({ ...draft, assetId: event.target.value }))} className={inputClass} /></Field>
                    <div className="grid gap-2 sm:grid-cols-2">
                      <Field label="用途">
                        <select value={referenceDraft.purpose} onChange={(event) => setReferenceDraft((draft) => ({ ...draft, purpose: event.target.value as AgentImageflowReferencePurpose }))} className={inputClass}>
                          <option value="character">角色</option>
                          <option value="style">风格</option>
                          <option value="scene">场景</option>
                          <option value="prop">道具</option>
                        </select>
                      </Field>
                      <Field label="绑定角色">
                        <select value={referenceDraft.characterId} onChange={(event) => setReferenceDraft((draft) => ({ ...draft, characterId: event.target.value }))} className={inputClass}>
                          <option value="">无</option>
                          {normalizedContext.characters.map((character) => (
                            <option key={character.id} value={character.id}>{character.name || character.id}</option>
                          ))}
                        </select>
                      </Field>
                      <Field label="标签"><input value={referenceDraft.label} onChange={(event) => setReferenceDraft((draft) => ({ ...draft, label: event.target.value }))} className={inputClass} /></Field>
                      <Field label="权重"><input type="number" min="0.1" max="5" step="0.1" value={referenceDraft.weight} onChange={(event) => setReferenceDraft((draft) => ({ ...draft, weight: event.target.value }))} className={inputClass} /></Field>
                    </div>
                    <Field label="备注"><textarea value={referenceDraft.notes} onChange={(event) => setReferenceDraft((draft) => ({ ...draft, notes: event.target.value }))} className={textareaClass} /></Field>
                    <button type="submit" disabled={saving} className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-blue-500 px-3 text-xs font-medium text-white transition hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-50">
                      <PlusIcon className="h-4 w-4" />
                      保存参考图
                    </button>
                  </form>

                  <div className="space-y-3">
                    {normalizedContext.references.length === 0 ? (
                      <div className="rounded-lg border border-dashed border-gray-200 px-4 py-8 text-center text-xs text-gray-500 dark:border-white/[0.08] dark:text-gray-400">暂无参考图绑定。</div>
                    ) : normalizedContext.references.map((reference) => (
                      <article key={reference.id} className={`rounded-lg border p-3 ${reference.status === 'archived' ? 'border-gray-200/70 bg-gray-50/60 opacity-70 dark:border-white/[0.08] dark:bg-white/[0.03]' : 'border-gray-200/80 bg-white dark:border-white/[0.08] dark:bg-gray-950/40'}`}>
                        <div className="flex flex-wrap items-start justify-between gap-2">
                          <div className="min-w-0">
                            <div className="break-words text-sm font-semibold text-gray-800 dark:text-gray-100">{reference.label || reference.asset_id}</div>
                            <div className="mt-1 break-all text-[11px] text-gray-500 dark:text-gray-400">{reference.asset_id}</div>
                          </div>
                          <span className="rounded-full border border-gray-200 bg-gray-50 px-2 py-0.5 text-[10px] text-gray-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">{reference.purpose}</span>
                        </div>
                        <div className="mt-3 grid gap-2 text-xs text-gray-600 dark:text-gray-300">
                          <div className="break-words"><span className="text-gray-400">状态：</span> {reference.status === 'archived' ? '归档' : '启用'}</div>
                          <div className="break-words"><span className="text-gray-400">绑定角色：</span> {compactText(reference.character_id)}</div>
                          <div className="break-words"><span className="text-gray-400">权重：</span> {reference.weight ?? 1}</div>
                          <div className="break-words"><span className="text-gray-400">备注：</span> {compactText(reference.notes)}</div>
                        </div>
                        <div className="mt-3 max-w-36">
                          <AssetThumb assetId={reference.asset_id} baseUrl={baseUrl} label="参考图" />
                        </div>
                        <div className="mt-3 flex flex-wrap gap-2">
                          <button type="button" onClick={() => setReferenceDraft({ assetId: reference.asset_id, purpose: reference.purpose, label: reference.label ?? '', characterId: reference.character_id ?? '', weight: String(reference.weight ?? 1), notes: reference.notes ?? '' })} className="h-8 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">编辑绑定</button>
                          <button type="button" onClick={() => archiveReference(reference)} disabled={saving} className="inline-flex h-8 items-center gap-1 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] text-gray-600 transition hover:border-amber-300 hover:text-amber-700 disabled:opacity-50 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">
                            <ArchiveIcon className="h-3.5 w-3.5" />
                            {reference.status === 'archived' ? '恢复' : '归档'}
                          </button>
                        </div>
                      </article>
                    ))}
                  </div>
                </div>
              )}

              {activeTab === 'recipes' && (
                <div className="grid gap-4 lg:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
                  <form onSubmit={handleRecipeSubmit} className="space-y-3 rounded-lg border border-gray-200/80 bg-gray-50/60 p-3 dark:border-white/[0.08] dark:bg-white/[0.03]">
                    <div className="flex items-center justify-between gap-2">
                      <div className="text-xs font-semibold text-gray-700 dark:text-gray-200">{editingRecipeId ? '编辑 Prompt 配方' : '新建 Prompt 配方'}</div>
                      {editingRecipeId && (
                        <button type="button" onClick={() => { setEditingRecipeId(null); setRecipeDraft(EMPTY_RECIPE_DRAFT) }} className="text-[11px] text-gray-500 hover:text-blue-600">
                          取消
                        </button>
                      )}
                    </div>
                    <div className="grid gap-2 sm:grid-cols-2">
                      <Field label="id"><input value={recipeDraft.id} onChange={(event) => setRecipeDraft((draft) => ({ ...draft, id: event.target.value }))} className={inputClass} /></Field>
                      <Field label="名称"><input value={recipeDraft.name} onChange={(event) => setRecipeDraft((draft) => ({ ...draft, name: event.target.value }))} className={inputClass} /></Field>
                      <Field label="状态">
                        <select value={recipeDraft.status} onChange={(event) => setRecipeDraft((draft) => ({ ...draft, status: event.target.value }))} className={inputClass}>
                          <option value="active">启用</option>
                          <option value="archived">归档</option>
                        </select>
                      </Field>
                      <Field label="画幅"><input value={recipeDraft.defaultAspectRatio} onChange={(event) => setRecipeDraft((draft) => ({ ...draft, defaultAspectRatio: event.target.value }))} className={inputClass} /></Field>
                      <Field label="格式"><input value={recipeDraft.defaultOutputFormat} onChange={(event) => setRecipeDraft((draft) => ({ ...draft, defaultOutputFormat: event.target.value }))} className={inputClass} /></Field>
                      <Field label="provider"><input value={recipeDraft.defaultProvider} onChange={(event) => setRecipeDraft((draft) => ({ ...draft, defaultProvider: event.target.value }))} className={inputClass} /></Field>
                      <Field label="model"><input value={recipeDraft.defaultModel} onChange={(event) => setRecipeDraft((draft) => ({ ...draft, defaultModel: event.target.value }))} className={inputClass} /></Field>
                    </div>
                    <Field label="角色描述块"><textarea value={recipeDraft.characterBlock} onChange={(event) => setRecipeDraft((draft) => ({ ...draft, characterBlock: event.target.value }))} className={textareaClass} /></Field>
                    <Field label="风格描述块"><textarea value={recipeDraft.styleBlock} onChange={(event) => setRecipeDraft((draft) => ({ ...draft, styleBlock: event.target.value }))} className={textareaClass} /></Field>
                    <Field label="镜头描述块"><textarea value={recipeDraft.cameraBlock} onChange={(event) => setRecipeDraft((draft) => ({ ...draft, cameraBlock: event.target.value }))} className={textareaClass} /></Field>
                    <Field label="渠道要求块"><textarea value={recipeDraft.channelBlock} onChange={(event) => setRecipeDraft((draft) => ({ ...draft, channelBlock: event.target.value }))} className={textareaClass} /></Field>
                    <Field label="negative prompt"><textarea value={recipeDraft.negativePrompt} onChange={(event) => setRecipeDraft((draft) => ({ ...draft, negativePrompt: event.target.value }))} className={textareaClass} /></Field>
                    <Field label="generation_config"><textarea value={recipeDraft.generationConfigText} onChange={(event) => setRecipeDraft((draft) => ({ ...draft, generationConfigText: event.target.value }))} className={textareaClass} /></Field>
                    <button type="submit" disabled={saving} className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-blue-500 px-3 text-xs font-medium text-white transition hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-50">
                      <PlusIcon className="h-4 w-4" />
                      保存配方
                    </button>
                  </form>

                  <div className="space-y-3">
                    {normalizedContext.prompt_recipes.length === 0 ? (
                      <div className="rounded-lg border border-dashed border-gray-200 px-4 py-8 text-center text-xs text-gray-500 dark:border-white/[0.08] dark:text-gray-400">暂无 Prompt 配方。</div>
                    ) : normalizedContext.prompt_recipes.map((recipe) => (
                      <article key={recipe.id} className={`rounded-lg border p-3 ${recipe.status === 'archived' ? 'border-gray-200/70 bg-gray-50/60 opacity-70 dark:border-white/[0.08] dark:bg-white/[0.03]' : 'border-gray-200/80 bg-white dark:border-white/[0.08] dark:bg-gray-950/40'}`}>
                        <div className="flex flex-wrap items-start justify-between gap-2">
                          <div className="min-w-0">
                            <div className="break-words text-sm font-semibold text-gray-800 dark:text-gray-100">{recipe.name || recipe.id}</div>
                            <div className="mt-1 break-all text-[11px] text-gray-500 dark:text-gray-400">{recipe.id}</div>
                          </div>
                          <span className="rounded-full border border-gray-200 bg-gray-50 px-2 py-0.5 text-[10px] text-gray-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">{recipe.status === 'archived' ? '归档' : '启用'}</span>
                        </div>
                        <div className="mt-3 grid gap-2 text-xs text-gray-600 dark:text-gray-300">
                          <div className="break-words"><span className="text-gray-400">默认参数：</span> {[recipe.default_aspect_ratio, recipe.default_output_format, recipe.default_provider, recipe.default_model].filter(Boolean).join(' / ') || '-'}</div>
                          <div className="break-words"><span className="text-gray-400">负面词：</span> {compactText(recipe.negative_prompt)}</div>
                          {(recipe.prompt_blocks ?? []).map((block, index) => (
                            <div key={`${recipe.id}-${index}`} className="break-words">
                              <span className="text-gray-400">{block.role || '描述块'}：</span> {compactText(block.text)}
                            </div>
                          ))}
                        </div>
                        <div className="mt-3 flex flex-wrap gap-2">
                          <button type="button" onClick={() => { setEditingRecipeId(recipe.id); setRecipeDraft(recipeToDraft(recipe)) }} className="h-8 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] text-gray-600 transition hover:border-blue-300 hover:text-blue-600 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">编辑</button>
                          <button type="button" onClick={() => archiveRecipe(recipe)} disabled={saving} className="inline-flex h-8 items-center gap-1 rounded-lg border border-gray-200 bg-white px-2.5 text-[11px] text-gray-600 transition hover:border-amber-300 hover:text-amber-700 disabled:opacity-50 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-gray-300">
                            <ArchiveIcon className="h-3.5 w-3.5" />
                            {recipe.status === 'archived' ? '恢复' : '归档'}
                          </button>
                        </div>
                      </article>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>,
    document.body,
  )
}
