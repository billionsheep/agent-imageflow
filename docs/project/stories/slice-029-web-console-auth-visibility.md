# Slice 029: P1 Web Console Auth & Asset Visibility

## Status

Done on 2026-06-20.

## Goal

解决 Web 控制台难以发现 MCP/CLI/REST 生成资产、project API key 手填成本高、scope 切换成本高，以及 401 被误显示成空列表的问题。

本 slice 只做轻量自托管 Admin 控制台体验，不做完整账号系统、SaaS 注册、租户、计费、RBAC 或 OAuth。Provider key 继续固定在服务端环境变量中，不进入 Web。

## Scope

- 新增轻量 Admin session 契约：`POST /api/admin/login`、`GET /api/admin/me`、`POST /api/admin/logout`。
- Admin session 使用 HttpOnly cookie；凭据来自 `ADMIN_USERNAME` / `ADMIN_PASSWORD`，可回退复用 Basic Auth 配置。
- 新增 `GET /api/admin/assets/recent`，用于 Web 控制台跨 workspace/project/campaign 查看最近资产。
- Web 服务端资产库默认进入 Recent Assets，不再要求用户先手填 project API key 才能查看最近资产。
- Project API key 仍保留给 MCP、CLI、REST 等外部 project 级调用。
- 资产卡显示 workspace/project/campaign，并提供一键切换到资产所在 scope。
- Web 区分 unauthorized、未配置 scope、真实空列表、筛选无结果和加载失败。

## Verification

- Docker `go test ./...` passed.
- `npm --prefix web test -- --run` passed: 17 files / 224 tests.
- `npm --prefix web run build` passed with existing chunk-size warning.
- `docker compose config` passed without printing provider secrets.
- REST/project-key smoke created `task_04d4615128bca9babecb -> asset_97eb0fc7e168ab0a54a4`.
- Admin recent smoke returned the REST-created asset without a project API key after login.
- Anonymous `GET /api/admin/assets/recent` returned 401.
- Project key `GET /api/projects/{project_id}/campaigns/{campaign_id}/assets` still returned the same smoke asset.
- Browser smoke showed unauthenticated state as `unauthorized` rather than `0 shown`.
- Browser smoke after Admin login showed the smoke asset in Recent Assets with workspace/project/campaign, source, session and batch metadata.
- Browser smoke clicking the asset `Scope` button switched Web settings to `ws_console_smoke_1781951146 / prj_console_smoke_1781951146 / cmp_console_smoke_1781951146`.

## Notes

- Local browser smoke should avoid mixing `127.0.0.1` Web origin with `localhost` API base because Admin cookies are host-bound. Use `localhost:8080` with the default `localhost:8081` API base, or configure both sides consistently.
- This slice does not change the provider generation path, worker queue, provider key storage, or project key semantics.
