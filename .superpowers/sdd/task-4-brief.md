### Task 4: Production View Quick Batch Entry And Feedback

**Files:**
- Modify: `web/src/components/ProductionViewModal.tsx`
- Modify: `web/src/components/ServerAssetLibrary.tsx`
- Test: `web/src/lib/operatorReview.test.ts`

**Interfaces:**
- Consumes `getProductionFiltersFromAsset`.
- Produces a card action or filter bridge that opens Production View with the asset's source/session/batch/story values.

- [ ] Test production filter extraction from an asset with source/session/batch/story metadata.
- [ ] Run focused tests and confirm the extraction test fails before implementation if not already covered.
- [ ] Add a compact `Batch` action on asset cards when session or batch exists; it should switch scope if necessary and open Production View.
- [ ] Allow Production View to initialize filters from store state or a lightweight UI seed without requiring manual long ID entry.
- [ ] Keep manual fields for advanced use.
- [ ] Ensure manifest pending/success/error feedback remains visible and inline.
- [ ] Run focused tests.

