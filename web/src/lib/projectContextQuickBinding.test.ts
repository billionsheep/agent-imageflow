import { describe, expect, it } from 'vitest'
import {
  addCharacterReferenceAsset,
  setCharacterPrimaryAsset,
  upsertCharacterProjectReference,
} from './projectContextQuickBinding'
import type { AgentImageflowProjectVisualContext } from './agentImageflowApi'

const baseContext = (): AgentImageflowProjectVisualContext => ({
  characters: [
    {
      id: 'char_mochi',
      name: 'Mochi',
      status: 'active',
      primary_asset_id: 'asset_old_primary',
      reference_asset_ids: ['asset_old_ref'],
    },
    {
      id: 'char_orange',
      name: 'Orange',
      status: 'active',
      reference_asset_ids: ['asset_orange_ref'],
    },
  ],
  references: [
    {
      id: 'ref_character_asset_old_ref',
      asset_id: 'asset_old_ref',
      purpose: 'character',
      character_id: 'char_mochi',
      status: 'active',
      weight: 1,
    },
    {
      id: 'ref_character_asset_archived',
      asset_id: 'asset_archived',
      purpose: 'character',
      character_id: 'char_mochi',
      status: 'archived',
      weight: 1,
    },
  ],
  prompt_recipes: [],
})

describe('projectContextQuickBinding', () => {
  it('sets the selected asset as the character primary asset only for the target character', () => {
    const next = setCharacterPrimaryAsset(baseContext(), 'char_mochi', 'asset_new_primary')

    expect(next.characters?.find((character) => character.id === 'char_mochi')?.primary_asset_id).toBe('asset_new_primary')
    expect(next.characters?.find((character) => character.id === 'char_orange')?.primary_asset_id).toBeUndefined()
  })

  it('appends a character reference asset without duplicates', () => {
    const once = addCharacterReferenceAsset(baseContext(), 'char_mochi', 'asset_new_ref')
    const twice = addCharacterReferenceAsset(once, 'char_mochi', 'asset_new_ref')

    expect(once.characters?.find((character) => character.id === 'char_mochi')?.reference_asset_ids).toEqual([
      'asset_old_ref',
      'asset_new_ref',
    ])
    expect(twice.characters?.find((character) => character.id === 'char_mochi')?.reference_asset_ids).toEqual([
      'asset_old_ref',
      'asset_new_ref',
    ])
  })

  it('upserts character-scoped project reference bindings and restores archived matches', () => {
    const created = upsertCharacterProjectReference(baseContext(), 'char_mochi', 'asset_new_ref')
    const restored = upsertCharacterProjectReference(created, 'char_mochi', 'asset_archived')

    expect(created.references?.find((reference) => reference.asset_id === 'asset_new_ref')).toMatchObject({
      asset_id: 'asset_new_ref',
      purpose: 'character',
      character_id: 'char_mochi',
      status: 'active',
      weight: 1,
    })
    expect(restored.references?.find((reference) => reference.asset_id === 'asset_archived')).toMatchObject({
      asset_id: 'asset_archived',
      purpose: 'character',
      character_id: 'char_mochi',
      status: 'active',
      weight: 1,
    })
  })
})
