### Task 1: Operator Review Helpers

**Files:**
- Create: `web/src/lib/operatorReview.ts`
- Create: `web/src/lib/operatorReview.test.ts`

**Interfaces:**
- Produces: `getAssetReviewSummary(asset)`, `getAssetTechnicalFields(asset)`, `getLocalhostMismatchWarning(pageOrigin, apiBaseUrl)`, `getProductionFiltersFromAsset(asset)`.
- Consumes: `AgentImageflowAssetResponse` from `web/src/lib/agentImageflowApi.ts`.

- [ ] Write tests for summary labels, hidden technical fields, host mismatch warning, and production filters.
- [ ] Run `npm --prefix web test -- --run web/src/lib/operatorReview.test.ts` and confirm the tests fail because the helper file does not exist.
- [ ] Implement the helper file with no browser-only side effects.
- [ ] Run the focused test and confirm it passes.

