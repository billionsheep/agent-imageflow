# Web Operator Review Console Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the already-working Agent ImageFlow Web console from a debug-heavy asset list into a server-first operator review console for human image review.

**Architecture:** Add small pure helpers for review labels, technical fields, host mismatch detection, and batch query seeds; call them from the existing React components. Keep the service API unchanged and continue using existing Admin session, Recent Assets, batch summary, scene actions and manifest endpoints.

**Tech Stack:** React + TypeScript + Zustand + Vitest in `web/`; Go/PostgreSQL/Redis backend remains unchanged unless a safety regression already has a backend test hook.

## Global Constraints

- Default language for project docs and communication is Simplified Chinese.
- Do not run real providers.
- Do not read, print, process, or migrate any real API key, provider key, secret, cookie, session token, password, or Authorization value.
- Provider secrets remain server-side only; Web Admin users consume the platform capability, not personal provider credentials.
- Project API key remains for MCP/CLI/REST external project access, not as a daily Web viewing prerequisite.
- Do not add multi-user accounts, registration, tenants, RBAC, publishing, content calendar, DAM, template marketplace, WebDAV/SMB server, ZIP export, or AI visual quality scoring.
- Use tests first for behavior changes where practical.
- Keep changes scoped to `web/src/lib/operatorReview.ts`, `web/src/lib/operatorReview.test.ts`, `web/src/components/ServerAssetLibrary.tsx`, `web/src/components/ProductionViewModal.tsx`, `web/src/components/SettingsModal.tsx`, existing web tests, and project docs.

---

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

### Task 2: Recent Assets Review Card

**Files:**
- Modify: `web/src/components/ServerAssetLibrary.tsx`
- Test: `web/src/lib/operatorReview.test.ts`

**Interfaces:**
- Consumes Task 1 helpers.
- Produces default card layout where review summary appears before technical metadata.

- [ ] Write or extend tests for the helper expectations used by the card.
- [ ] Run focused tests and confirm the new expectation fails if needed.
- [ ] Replace default technical grid with prompt/story/scene/source/created review summary.
- [ ] Move workspace/project/campaign/provider/model/task/hash/source/session/batch/story/scene/target and metadata/parameters into `Technical details`.
- [ ] Keep Select, Reject, Original, Metadata, Copy ID, Copy URL, Scope and Reference actions.
- [ ] Run focused tests.

### Task 3: Settings Server-First Copy And Host Guidance

**Files:**
- Modify: `web/src/components/SettingsModal.tsx`
- Modify: `web/src/components/ServerAssetLibrary.tsx`
- Test: `web/src/lib/operatorReview.test.ts`

**Interfaces:**
- Consumes `getLocalhostMismatchWarning`.
- Produces visible copy that distinguishes Agent ImageFlow server connection, Project API key, Basic fallback, Admin session, and advanced/legacy direct provider profile.

- [ ] Add helper test cases for `localhost` vs `127.0.0.1` mismatch and matching hosts.
- [ ] Run focused tests and confirm failure before helper implementation if not already covered.
- [ ] Add host mismatch warning near unauthorized/login state and server connection settings.
- [ ] Adjust Settings copy so Web users understand provider keys stay server-side.
- [ ] Run focused tests.

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

### Task 5: Non-Exposure Regression And Project Docs

**Files:**
- Modify: `web/src/lib/operatorReview.test.ts`
- Modify: `docs/project/TASKS.md`
- Modify: `docs/project/PROJECT_PLAN.md`
- Modify: `docs/project/PROJECT_STATUS_MAP.md`
- Modify: `docs/project/CHECKPOINTS.md`
- Modify: `docs/project/DECISIONS.md`
- Modify: `issues/next-phase-p2-web-operator-review-console.csv`

**Interfaces:**
- Consumes all task outputs.
- Produces final evidence entries and keeps P2 scope clear.

- [ ] Add tests that default review summary and technical field helpers do not emit local absolute path or secret-like keys.
- [ ] Run focused tests.
- [ ] Update CSV evidence/status for completed tasks.
- [ ] Update project management docs with verification evidence.
- [ ] Run `npm --prefix web test -- --run`, `npm --prefix web run build`, and `git diff --check`.
