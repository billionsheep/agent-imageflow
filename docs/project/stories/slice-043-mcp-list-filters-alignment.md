# Story: 043 - MCP List Filters Alignment

## Status

- State: Done
- Created: 2026-06-22
- Updated: 2026-06-22

## Product Goal

Align MCP `list_image_assets` with the existing REST/CLI asset list filters so external agents can inspect clean story batches by metadata without reconstructing batch membership from unrelated campaign assets.

## Source Context

- Batch story export plan: `issues/next-phase-p1-batch-story-export-foundation.csv` / `P1-BSE-004`
- Summary contract: `docs/project/stories/slice-041-batch-story-summary-contract.md`
- Existing MCP server: `internal/mcp/server.go`

## In Scope

- Extend `list_image_assets` MCP input schema with `source`, `session_id`, `batch_id`, `status`, `keyword` and `limit`.
- Pass those fields through to `domain.AssetListQuery`.
- Preserve existing default `project_id` / `campaign_id` behavior.
- Keep MCP output from exposing local filesystem paths.
- Add focused MCP unit tests.

## Out of Scope

- No new MCP summary tool.
- No REST, CLI, Web, store or provider changes.
- No real provider calls.
- No API key, provider key, secret, cookie or session handling.

## Implementation Notes

- `list_image_assets` still defaults to `DEFAULT_PROJECT_ID` and `DEFAULT_CAMPAIGN_ID` when callers omit `project_id` or `campaign_id`.
- `status` is passed through as provided; service normalization continues to map compatibility values such as `generated` and `selected`.
- `limit` is passed through to the application service, where existing list limit normalization and caps apply.
- MCP semantic output removes `local_path` before returning `structuredContent` or text content.

## Verification

- `docker run --rm -v "$PWD":/src -w /src golang:1.25.3-alpine sh -lc '/usr/local/go/bin/gofmt -w internal/mcp/server.go internal/mcp/server_test.go && /usr/local/go/bin/go test ./internal/mcp'`

Result: passed.

