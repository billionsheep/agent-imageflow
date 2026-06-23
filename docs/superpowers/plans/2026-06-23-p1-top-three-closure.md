# P1 Top Three Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the three highest-priority Agent ImageFlow gaps: release/canary readiness for the provider MIME fix, Web data cleanup entry, and quick character reference binding.

**Architecture:** Keep destructive cleanup out of MCP. Use Admin session Web/REST/CLI for cleanup. Use the existing Project Visual Context REST endpoint for character reference binding and the existing storage cleanup REST helpers for preview/execute.

**Tech Stack:** Go API/worker, TypeScript React Web, Vitest, Dockerized Go test, project docs under `docs/project/`.

## Global Constraints

- Do not run real provider by default; real canary requires explicit user confirmation because it may spend money.
- Do not read, print, store, or move provider key, API key, Admin cookie, session token, cleanup token, or secret.
- Do not add MCP hard delete/destructive tools.
- Update project management docs after behavior changes.
- Prefer existing helpers and UI patterns; no new third-party dependencies.
- Use TDD for Web/API behavior changes.

---

### Task 1: Release And Real Canary Readiness

**Files:**
- Modify: `docs/project/TASKS.md`
- Modify: `docs/project/CHECKPOINTS.md`
- Modify: `docs/project/RUNBOOK.md`
- Modify: `docs/project/PROJECT_PLAN.md`

**Interfaces:**
- Produces: a clear deploy/canary checklist for the already committed MIME fix.
- Consumes: existing provider fix commit `b6f8f67`.

- [x] **Step 1: Document the blocked canary boundary**

Add docs stating the MIME fix is committed locally, but server deployment and the 1-image real provider canary require explicit confirmation.

- [x] **Step 2: Add canary checklist**

Document smoke acceptance: MCP creates one task with character references, provider uses edits/reference path, metadata shows `reference_asset_count > 0` and `provider_reference_participation=resolved_input_files`, no key is printed.

- [x] **Step 3: Verify docs only**

Run: `git diff --check docs/project/TASKS.md docs/project/CHECKPOINTS.md docs/project/RUNBOOK.md docs/project/PROJECT_PLAN.md`

Expected: exit 0.

### Task 2: Web Data Management Cleanup Entry

**Files:**
- Modify: `web/src/components/ScopeManagerModal.tsx`
- Modify: `web/src/lib/agentImageflowApi.ts`
- Modify: `web/src/lib/agentImageflowApi.test.ts`
- Modify: `docs/project/TASKS.md`
- Modify: `docs/project/PROJECT_PLAN.md`
- Modify: `docs/project/CHECKPOINTS.md`
- Modify: `issues/next-phase-p1-safe-delete-and-trial-reset.csv`

**Interfaces:**
- Consumes: `previewAgentImageflowStorageCleanup`, `executeAgentImageflowStorageCleanup`, `getAgentImageflowStorageGovernance`, current scope settings.
- Produces: Admin Web cleanup preview/execute UI for current campaign scope.

- [x] **Step 1: Write failing tests**

Add focused tests for cleanup request helper behavior if any helper is missing. If all helpers exist, add tests for request body defaults or URL builders.

- [x] **Step 2: Run tests to verify red**

Run targeted Vitest command for the touched helper test.

- [x] **Step 3: Add Scope Manager data cleanup panel**

Add a compact "数据清理" panel for the selected campaign:
- Preview button calls cleanup preview with `include_rejected`, `include_generated`, `include_deprecated`, `include_failed_task_tmp`, and `include_orphans`.
- Show candidate count, file count, bytes, protected selected/published counts, dry-run token preview masked/truncated, and reason summary.
- Execute requires user typing `清理当前空间` and sends `execute=true` with the dry-run token.
- Do not display local paths, secrets, cookies, or full tokens.

- [x] **Step 4: Run Web tests/build**

Run:
- `npm --prefix web test -- --run`
- `npm --prefix web run build`

Expected: tests pass and build exits 0; existing chunk warning is acceptable.

### Task 3: Character Reference Quick Binding

**Files:**
- Modify: `web/src/components/ProjectContextModal.tsx`
- Modify: `web/src/components/ServerAssetLibrary.tsx`
- Modify: `web/src/store.ts` if the modal input needs richer open parameters.
- Modify: `docs/project/TASKS.md`
- Modify: `docs/project/PROJECT_PLAN.md`
- Modify: `docs/project/CHECKPOINTS.md`
- Modify: `issues/next-phase-p1-character-reference-intake-consistency.csv`

**Interfaces:**
- Consumes: existing visual context get/update endpoint and `projectContextReferenceAssetId`.
- Produces: one-click "set as character primary" and "add as character reference" flows without requiring manual asset_id typing.

- [x] **Step 1: Write failing tests where practical**

Add helper tests if the binding transformation is extracted. If UI-only, verify with Web tests/build and document manual browser smoke pending.

- [x] **Step 2: Add quick binding controls**

In Project Context when opened from an asset:
- Show the pending asset thumbnail.
- Let the user pick a character.
- Provide buttons: "设为主图", "加入参考图", "保存为项目参考图".
- Update `characters[].primary_asset_id`, `characters[].reference_asset_ids`, and/or `references[]` through the existing visual context save path.

- [x] **Step 3: Keep generic reference flow**

Keep existing reference binding form for style/scene/prop and advanced editing.

- [x] **Step 4: Run Web tests/build**

Run:
- `npm --prefix web test -- --run`
- `npm --prefix web run build`

Expected: tests pass and build exits 0.

### Task 4: Final Verification And Commit

**Files:**
- All touched files.

- [x] **Step 1: Run full verification**

Run:
- `docker run --rm -v "$PWD":/src -w /src golang:1.25.3-alpine go test ./...`
- `npm --prefix web test -- --run`
- `npm --prefix web run build`
- `docker compose config --quiet`
- `docker compose -f docker-compose.prod.yml --env-file .env.example.prod config --quiet`
- `git diff --check`

- [ ] **Step 2: Commit**

Commit the completed safe subset. Do not deploy or run real provider unless user explicitly confirms.
