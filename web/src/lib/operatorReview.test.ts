import { describe, expect, it } from 'vitest'
import {
  getAssetReviewTitle,
  getAssetReviewSummary,
  getAssetReviewStatusLabel,
  getAssetTechnicalFields,
  getLocalhostMismatchWarning,
  getProductionFiltersFromAsset,
} from './operatorReview'
import type { AgentImageflowAssetResponse } from './agentImageflowApi'

function assetFixture(overrides: Partial<AgentImageflowAssetResponse> = {}): AgentImageflowAssetResponse {
  return {
    asset_id: 'asset_pet_scene_001',
    workspace_id: 'ws_pet',
    project_id: 'prj_pet_story',
    campaign_id: 'cmp_rainy_window',
    task_id: 'task_scene_001',
    current_version: 1,
    status: 'selected',
    hash: 'sha256:abc123',
    provider: 'mock',
    model: 'mock-image-v1',
    prompt: '小狗在雨窗边看见橘猫，温暖绘本风',
    metadata_json: {
      source: 'codex',
      session_id: 'pet_story_session_2026_06_22',
      batch_id: 'pet_story_batch_001',
      story_id: 'rainy_window_cat',
      scene_id: 'scene_001',
      target_path: 'assets/pet-story/rainy-window-cat/scene-001.png',
    },
    parameters_json: {
      aspect_ratio: '3:4',
      output_format: 'png',
    },
    delivery: {
      local_path: '/Users/moon/Workspace/tools/agent-imageflow/storage/workspaces/ws_pet/originals/asset_pet_scene_001/1.png',
      download_url: '/api/assets/asset_pet_scene_001/original',
      thumbnail_url: '/api/assets/asset_pet_scene_001/thumbnail',
      metadata_url: '/api/assets/asset_pet_scene_001/metadata',
    },
    created_at: '2026-06-22T04:30:00Z',
    ...overrides,
  }
}

describe('operator review helpers', () => {
  it('maps technical asset statuses to human review labels', () => {
    expect(getAssetReviewStatusLabel('generated')).toBe('待选')
    expect(getAssetReviewStatusLabel('draft')).toBe('待选')
    expect(getAssetReviewStatusLabel('selected')).toBe('已选')
    expect(getAssetReviewStatusLabel('approved')).toBe('已选')
    expect(getAssetReviewStatusLabel('rejected')).toBe('已拒绝')
    expect(getAssetReviewStatusLabel('published')).toBe('已发布')
  })

  it('builds a short story-first asset title from recipe-expanded prompts', () => {
    expect(getAssetReviewTitle(assetFixture({
      prompt: [
        'Story scene: Mochi and Biscuit find a moon cake clue under the sofa',
        '',
        'cozy picture-book illustration, soft natural light, expressive cute pets',
        '',
        'mobile-first social cover, no readable text inside image',
      ].join('\n'),
    }))).toBe('Mochi and Biscuit find a moon cake clue under the sofa')

    expect(getAssetReviewTitle(assetFixture({
      metadata_json: {
        scene_summary: 'Orange Nap guards the blanket fort',
      },
      prompt: 'Story scene: this prompt should not win',
    }))).toBe('Orange Nap guards the blanket fort')
  })

  it('builds a review-first summary without debug identifiers', () => {
    const summary = getAssetReviewSummary(assetFixture())
    const keys = summary.map((field) => field.key)
    const text = summary.map((field) => `${field.label}:${field.value}`).join('\n')

    expect(keys).toEqual(['prompt', 'story', 'scene', 'source', 'created', 'target'])
    expect(text).toContain('Prompt:小狗在雨窗边看见橘猫，温暖绘本风')
    expect(text).toContain('Story:rainy_window_cat')
    expect(text).toContain('Scene:scene_001')
    expect(text).not.toContain('asset_pet_scene_001')
    expect(text).not.toContain('task_scene_001')
    expect(text).not.toContain('sha256:abc123')
    expect(text).not.toContain('/Users/moon')
  })

  it('keeps technical fields behind a scrubbed helper', () => {
    const fields = getAssetTechnicalFields(assetFixture({
      metadata_json: {
        source: 'codex',
        session_id: 'pet_story_session_2026_06_22',
        batch_id: 'pet_story_batch_001',
        story_id: 'rainy_window_cat',
        scene_id: 'scene_001',
        target_path: 'assets/pet-story/rainy-window-cat/scene-001.png',
        local_path: '/Users/moon/private/source.png',
        cookie: 'cookie-value',
        nested: {
          Authorization: 'Bearer hidden',
          safe_note: 'keep me',
          output_path: 'C:\\Users\\moon\\private\\out.png',
          server_path: '/app/storage/private/out.png',
        },
      },
      parameters_json: {
        model_hint: 'mock-image-v1',
        provider_key: 'provider-secret',
        generation_config: {
          seed: 42,
          api_key: 'project-secret',
        },
      },
    }))
    const keys = fields.map((field) => field.key)
    const text = fields.map((field) => `${field.label}:${field.value}`).join('\n')

    expect(keys).toEqual([
      'asset',
      'task',
      'workspace',
      'project',
      'campaign',
      'version',
      'provider',
      'model',
      'hash',
      'source',
      'session',
      'batch',
      'story',
      'scene',
      'target',
      'metadata',
      'parameters',
    ])
    expect(text).toContain('Asset ID:asset_pet_scene_001')
    expect(text).toContain('Task ID:task_scene_001')
    expect(text).toContain('"safe_note": "keep me"')
    expect(text).toContain('"seed": 42')
    expect(text).not.toContain('/Users/moon')
    expect(text).not.toContain('C:\\Users\\moon')
    expect(text).not.toContain('/app/storage')
    expect(text).not.toContain('provider-secret')
    expect(text).not.toContain('project-secret')
    expect(text).not.toContain('cookie-value')
    expect(text).not.toContain('Bearer hidden')
    expect(text).not.toMatch(/api_key|provider_key|Authorization|cookie|local_path/i)
  })

  it('warns when localhost and 127.0.0.1 hosts are mixed', () => {
    expect(getLocalhostMismatchWarning('http://localhost:5173', 'http://127.0.0.1:8081')).toContain('localhost')
    expect(getLocalhostMismatchWarning('http://127.0.0.1:4173', 'http://localhost:8081')).toContain('127.0.0.1')
    expect(getLocalhostMismatchWarning('http://localhost:5173', 'http://localhost:8081')).toBeNull()
    expect(getLocalhostMismatchWarning('http://127.0.0.1:4173', 'http://127.0.0.1:8081')).toBeNull()
    expect(getLocalhostMismatchWarning('https://example.test', 'http://localhost:8081')).toBeNull()
  })

  it('extracts production filters from asset batch metadata', () => {
    expect(getProductionFiltersFromAsset(assetFixture())).toEqual({
      sessionId: 'pet_story_session_2026_06_22',
      batchId: 'pet_story_batch_001',
      storyId: 'rainy_window_cat',
      source: 'codex',
      status: '',
      includeSetup: false,
      limit: '100',
    })

    expect(getProductionFiltersFromAsset(assetFixture({
      metadata_json: {
        source: 'codex',
        scene_id: 'scene_001',
      },
    }))).toBeNull()
  })
})
