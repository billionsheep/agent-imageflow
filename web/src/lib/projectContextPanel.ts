import type { AgentImageflowReferenceDiagnostics } from './agentImageflowApi'

export interface ProjectContextPanelRecipe {
  id: string
  name?: string
  status?: string
}

export interface ProjectContextPanelCharacter {
  id: string
  name?: string
  status?: string
  primary_asset_id?: string
  reference_asset_ids?: string[]
}

export interface ProjectContextPanelReference {
  asset_id: string
  label?: string
  purpose?: string
  status?: string
}

export interface ProjectContextPanelSummaryInput {
  useProjectVisualContext: boolean
  selectedRecipeId: string
  selectedCharacterIds: string[]
  selectedReferenceAssetIds: string[]
  recipes: ProjectContextPanelRecipe[]
  characters: ProjectContextPanelCharacter[]
  references: ProjectContextPanelReference[]
}

export interface ProjectVisualContextReadiness {
  activeCharacterCount: number
  characterWithImageCount: number
  missingCharacterImageCount: number
  referenceCount: number
  recipeCount: number
  missingCharacterImageNames: string[]
}

export interface ProjectReferenceDiagnosticsDisplayItem {
  label: string
  value: number | string
}

export interface ProjectReferenceDiagnosticsNotice {
  tone: 'info' | 'warning'
  text: string
}

export interface ProjectReferenceDiagnosticsCard {
  readiness: string
  summary: string
  labels: string[]
  counts: ProjectReferenceDiagnosticsDisplayItem[]
  checks: ProjectReferenceDiagnosticsDisplayItem[]
  providerRisk: string
  notices: ProjectReferenceDiagnosticsNotice[]
}

const REFERENCE_DIAGNOSTIC_LABELS: Record<string, string> = {
  image_backed: '图片支撑充分',
  text_constrained: '主要依赖文本约束',
  missing_environment_reference: '缺少环境参考',
  weak_species_lock: '物种锁定偏弱',
}

const REFERENCE_DIAGNOSTIC_STATUS: Record<string, string> = {
  ready: '已就绪',
  partial: '部分就绪',
  blocked: '存在阻塞',
  strong: '强',
  moderate: '中等',
  weak: '弱',
  low: '低',
  medium: '中',
  high: '高',
}

function findName<T extends { id?: string; asset_id?: string; name?: string; label?: string }>(
  values: T[],
  id: string,
): string {
  const found = values.find((value) => value.id === id || value.asset_id === id)
  return found?.name || found?.label || id
}

export function getProjectContextPanelSummary(input: ProjectContextPanelSummaryInput): string {
  const recipeId = input.selectedRecipeId.trim()
  const parts: string[] = []
  if (recipeId) {
    parts.push(findName(input.recipes, recipeId))
  }

  if (input.selectedCharacterIds.length > 0) {
    parts.push(input.selectedCharacterIds.map((id) => findName(input.characters, id)).join(', '))
  }

  if (input.selectedReferenceAssetIds.length > 0) {
    const count = input.selectedReferenceAssetIds.length
    parts.push(`${count} reference${count === 1 ? '' : 's'}`)
  }

  if (parts.length > 0) return parts.join(' · ')
  return input.useProjectVisualContext ? 'Project defaults enabled' : 'No context selected'
}

export function getProjectVisualContextReadiness(input: {
  characters: ProjectContextPanelCharacter[]
  references: ProjectContextPanelReference[]
  recipes: ProjectContextPanelRecipe[]
}): ProjectVisualContextReadiness {
  const activeCharacters = input.characters.filter((character) => character.status !== 'archived')
  const activeReferences = input.references.filter((reference) => reference.status !== 'archived')
  const activeRecipes = input.recipes.filter((recipe) => recipe.status !== 'archived')
  const missingCharacters = activeCharacters.filter((character) => (
    !character.primary_asset_id?.trim() && (character.reference_asset_ids ?? []).filter((assetId) => assetId.trim()).length === 0
  ))
  return {
    activeCharacterCount: activeCharacters.length,
    characterWithImageCount: activeCharacters.length - missingCharacters.length,
    missingCharacterImageCount: missingCharacters.length,
    referenceCount: activeReferences.length,
    recipeCount: activeRecipes.length,
    missingCharacterImageNames: missingCharacters.map((character) => character.name || character.id),
  }
}

function formatDiagnosticToken(value: string, dictionary: Record<string, string>): string {
  const normalized = value.trim().toLowerCase()
  if (!normalized) return '-'
  if (dictionary[normalized]) return dictionary[normalized]
  return normalized
    .split(/[_\s]+/)
    .filter(Boolean)
    .map((part) => part.toUpperCase() === part ? part : part[0]?.toUpperCase() + part.slice(1))
    .join(' ')
}

export function formatProjectReferenceDiagnosticLabel(label: string): string {
  return REFERENCE_DIAGNOSTIC_LABELS[label] ?? formatDiagnosticToken(label, {})
}

export function getProjectReferenceDiagnosticsCard(
  diagnostics?: AgentImageflowReferenceDiagnostics | null,
): ProjectReferenceDiagnosticsCard | null {
  if (!diagnostics) return null

  const notices: ProjectReferenceDiagnosticsNotice[] = []
  if (diagnostics.missing_character_image_count > 0) {
    const preview = diagnostics.missing_character_ids.slice(0, 4).join('、')
    notices.push({
      tone: 'warning',
      text: `仍有 ${diagnostics.missing_character_image_count} 个角色缺少参考图：${preview}${diagnostics.missing_character_ids.length > 4 ? ` 等 ${diagnostics.missing_character_ids.length} 个` : ''}`,
    })
  }
  if (diagnostics.environment_reference_count === 0) {
    notices.push({
      tone: 'warning',
      text: '当前没有环境参考图，复杂场景一致性可能偏弱。',
    })
  }
  if (!diagnostics.negative_prompt_covers_species_drift || !diagnostics.identity_signal_present) {
    notices.push({
      tone: 'info',
      text: '建议继续补强负向约束或身份信号，降低角色漂移风险。',
    })
  }

  return {
    readiness: formatDiagnosticToken(diagnostics.primary_readiness, REFERENCE_DIAGNOSTIC_STATUS),
    summary: diagnostics.summary?.trim() || '暂无诊断摘要',
    labels: diagnostics.labels.map(formatProjectReferenceDiagnosticLabel),
    counts: [
      { label: '启用角色', value: diagnostics.active_character_count },
      { label: '角色有图', value: diagnostics.character_with_image_count },
      { label: '缺图角色', value: diagnostics.missing_character_image_count },
      { label: '启用参考', value: diagnostics.active_reference_count },
      { label: '环境参考', value: diagnostics.environment_reference_count },
      { label: '图片参考', value: diagnostics.image_reference_count },
    ],
    checks: [
      { label: '物种漂移负向约束', value: diagnostics.negative_prompt_covers_species_drift ? '已覆盖' : '未覆盖' },
      { label: '身份信号', value: diagnostics.identity_signal_present ? '已提供' : '缺失' },
    ],
    providerRisk: formatDiagnosticToken(diagnostics.provider_reference_participation_risk, REFERENCE_DIAGNOSTIC_STATUS),
    notices,
  }
}
