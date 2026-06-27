import { describe, expect, it } from 'vitest'
import {
  formatProjectReferenceDiagnosticLabel,
  getProjectContextPanelSummary,
  getProjectReferenceDiagnosticsCard,
  getProjectVisualContextReadiness,
} from './projectContextPanel'

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

  it('summarizes IP production readiness from character images, references and recipes', () => {
    expect(getProjectVisualContextReadiness({
      characters: [
        { id: 'dog_mochi', name: 'Mochi', primary_asset_id: 'asset_mochi' },
        { id: 'dog_biscuit', name: 'Biscuit', reference_asset_ids: ['asset_biscuit_ref'] },
        { id: 'cat_orange', name: 'Orange Nap' },
      ],
      references: [
        { asset_id: 'asset_style', purpose: 'style' },
      ],
      recipes: [
        { id: 'pet_story_cover', name: 'Cute Pet Story Cover' },
      ],
    })).toEqual({
      activeCharacterCount: 3,
      characterWithImageCount: 2,
      missingCharacterImageCount: 1,
      referenceCount: 1,
      recipeCount: 1,
      missingCharacterImageNames: ['Orange Nap'],
    })
  })

  it('formats reference diagnostic labels into readable Chinese copy', () => {
    expect(formatProjectReferenceDiagnosticLabel('image_backed')).toBe('图片支撑充分')
    expect(formatProjectReferenceDiagnosticLabel('text_constrained')).toBe('主要依赖文本约束')
    expect(formatProjectReferenceDiagnosticLabel('missing_environment_reference')).toBe('缺少环境参考')
    expect(formatProjectReferenceDiagnosticLabel('weak_species_lock')).toBe('物种锁定偏弱')
    expect(formatProjectReferenceDiagnosticLabel('custom_signal')).toBe('Custom Signal')
  })

  it('builds a structured diagnostics card for the project context modal', () => {
    expect(getProjectReferenceDiagnosticsCard({
      primary_readiness: 'partial',
      labels: ['image_backed', 'weak_species_lock'],
      summary: 'Character grounding is usable, but scene references are still thin.',
      active_character_count: 3,
      character_with_image_count: 2,
      missing_character_image_count: 5,
      missing_character_ids: ['char_orange', 'char_mocha', 'char_cloud', 'char_snow', 'char_milk'],
      active_reference_count: 4,
      environment_reference_count: 0,
      image_reference_count: 3,
      negative_prompt_covers_species_drift: false,
      identity_signal_present: true,
      provider_reference_participation_risk: 'medium',
    })).toEqual({
      readiness: '部分就绪',
      summary: 'Character grounding is usable, but scene references are still thin.',
      labels: ['图片支撑充分', '物种锁定偏弱'],
      counts: [
        { label: '启用角色', value: 3 },
        { label: '角色有图', value: 2 },
        { label: '缺图角色', value: 5 },
        { label: '启用参考', value: 4 },
        { label: '环境参考', value: 0 },
        { label: '图片参考', value: 3 },
      ],
      checks: [
        { label: '物种漂移负向约束', value: '未覆盖' },
        { label: '身份信号', value: '已提供' },
      ],
      providerRisk: '中',
      notices: [
        { tone: 'warning', text: '仍有 5 个角色缺少参考图：char_orange、char_mocha、char_cloud、char_snow 等 5 个' },
        { tone: 'warning', text: '当前没有环境参考图，复杂场景一致性可能偏弱。' },
        { tone: 'info', text: '建议继续补强负向约束或身份信号，降低角色漂移风险。' },
      ],
    })
  })
})
