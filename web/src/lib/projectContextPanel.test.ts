import { describe, expect, it } from 'vitest'
import { getProjectContextPanelSummary } from './projectContextPanel'

describe('project context panel helpers', () => {
  it('summarizes selected recipe, characters, and references for collapsed input bar', () => {
    expect(getProjectContextPanelSummary({
      useProjectVisualContext: true,
      selectedRecipeId: 'pet_story_cover',
      selectedCharacterIds: ['dog_mochi', 'cat_orange'],
      selectedReferenceAssetIds: ['asset_style_001'],
      recipes: [{ id: 'pet_story_cover', name: 'Cute Pet Story Cover' }],
      characters: [
        { id: 'dog_mochi', name: 'Mochi' },
        { id: 'cat_orange', name: 'Orange Nap' },
      ],
      references: [{ asset_id: 'asset_style_001', label: 'Cozy style' }],
    })).toBe('Cute Pet Story Cover · Mochi, Orange Nap · 1 reference')
  })

  it('keeps the collapsed summary calm when nothing is selected', () => {
    expect(getProjectContextPanelSummary({
      useProjectVisualContext: false,
      selectedRecipeId: '',
      selectedCharacterIds: [],
      selectedReferenceAssetIds: [],
      recipes: [],
      characters: [],
      references: [],
    })).toBe('No context selected')

    expect(getProjectContextPanelSummary({
      useProjectVisualContext: true,
      selectedRecipeId: '',
      selectedCharacterIds: [],
      selectedReferenceAssetIds: [],
      recipes: [],
      characters: [],
      references: [],
    })).toBe('Project defaults enabled')
  })
})
