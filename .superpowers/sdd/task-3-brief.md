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

