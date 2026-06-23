import type {
  AgentImageflowCharacterProfile,
  AgentImageflowProjectReferenceBinding,
  AgentImageflowProjectVisualContext,
  AgentImageflowReferencePurpose,
} from './agentImageflowApi'

function slugPart(value: string): string {
  return value.trim().toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_+|_+$/g, '')
}

function createReferenceBindingId(purpose: AgentImageflowReferencePurpose, characterId: string | undefined, assetId: string): string {
  const characterPart = characterId ? `${slugPart(characterId)}_` : ''
  return `ref_${purpose}_${characterPart}${slugPart(assetId) || Date.now().toString(36)}`
}

function updateCharacter(
  context: AgentImageflowProjectVisualContext,
  characterId: string,
  updater: (character: AgentImageflowCharacterProfile) => AgentImageflowCharacterProfile,
): AgentImageflowProjectVisualContext {
  if (!(context.characters ?? []).some((character) => character.id === characterId)) {
    return context
  }

  return {
    ...context,
    characters: (context.characters ?? []).map((character) => (
      character.id === characterId ? updater(character) : character
    )),
  }
}

export function setCharacterPrimaryAsset(
  context: AgentImageflowProjectVisualContext,
  characterId: string,
  assetId: string,
): AgentImageflowProjectVisualContext {
  const normalizedAssetId = assetId.trim()
  return updateCharacter(context, characterId, (character) => ({
    ...character,
    primary_asset_id: normalizedAssetId || undefined,
  }))
}

export function addCharacterReferenceAsset(
  context: AgentImageflowProjectVisualContext,
  characterId: string,
  assetId: string,
): AgentImageflowProjectVisualContext {
  const normalizedAssetId = assetId.trim()
  return updateCharacter(context, characterId, (character) => {
    const nextReferenceAssetIds = [...(character.reference_asset_ids ?? [])]
    if (normalizedAssetId && !nextReferenceAssetIds.includes(normalizedAssetId)) {
      nextReferenceAssetIds.push(normalizedAssetId)
    }
    return {
      ...character,
      reference_asset_ids: nextReferenceAssetIds,
    }
  })
}

export function upsertProjectReferenceBinding(
  context: AgentImageflowProjectVisualContext,
  binding: {
    assetId: string
    purpose: AgentImageflowReferencePurpose
    characterId?: string
    label?: string
    weight?: number
    notes?: string
  },
): AgentImageflowProjectVisualContext {
  const assetId = binding.assetId.trim()
  const hasCharacterId = Object.prototype.hasOwnProperty.call(binding, 'characterId')
  const characterId = hasCharacterId ? (binding.characterId?.trim() || undefined) : undefined
  const hasLabel = Object.prototype.hasOwnProperty.call(binding, 'label')
  const hasNotes = Object.prototype.hasOwnProperty.call(binding, 'notes')
  const hasWeight = Object.prototype.hasOwnProperty.call(binding, 'weight')
  const lookupCharacterId = hasCharacterId ? (characterId ?? '') : ''
  const existing = (context.references ?? []).find((reference) =>
    reference.asset_id === assetId &&
    reference.purpose === binding.purpose &&
    (reference.character_id ?? '') === lookupCharacterId,
  )

  const nextReference: AgentImageflowProjectReferenceBinding = {
    id: existing?.id ?? createReferenceBindingId(binding.purpose, characterId, assetId),
    asset_id: assetId,
    purpose: binding.purpose,
    label: hasLabel ? (binding.label?.trim() || undefined) : existing?.label,
    character_id: hasCharacterId ? characterId : existing?.character_id,
    weight: hasWeight ? binding.weight : (existing?.weight ?? 1),
    notes: hasNotes ? (binding.notes?.trim() || undefined) : existing?.notes,
    status: 'active',
  }

  return {
    ...context,
    references: existing
      ? (context.references ?? []).map((reference) => reference.id === existing.id ? nextReference : reference)
      : [...(context.references ?? []), nextReference],
  }
}

export function upsertCharacterProjectReference(
  context: AgentImageflowProjectVisualContext,
  characterId: string,
  assetId: string,
): AgentImageflowProjectVisualContext {
  return upsertProjectReferenceBinding(context, {
    assetId,
    purpose: 'character',
    characterId,
    weight: 1,
  })
}
