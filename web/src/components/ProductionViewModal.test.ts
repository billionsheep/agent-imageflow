import { describe, expect, it } from 'vitest'
import * as productionViewModule from './ProductionViewModal'

describe('ProductionViewModal manifest export helpers', () => {
  it('exposes final delivery export mode label and request shape', () => {
    const moduleExports = productionViewModule as unknown as Record<string, unknown>
    const getManifestModeLabel = moduleExports.getManifestModeLabel as ((mode: string) => string) | undefined
    const buildManifestRequestOptions = moduleExports.buildManifestRequestOptions as ((mode: string) => Record<string, unknown>) | undefined

    expect(typeof getManifestModeLabel).toBe('function')
    expect(typeof buildManifestRequestOptions).toBe('function')
    expect(getManifestModeLabel?.('finalDelivery')).toBe('Final delivery manifest')
    expect(buildManifestRequestOptions?.('finalDelivery')).toMatchObject({
      selectedOnly: true,
      includeRejected: false,
      view: 'final_delivery',
    })
  })
})
