# Story: Slice 011 - Real Thumbnail Resize WebP

## Status

- State: Done
- Created: 2026-06-18
- Updated: 2026-06-18

## Product Goal

让服务端真正从原图生成统一缩略图，而不是依赖 provider 顺手返回的缩略图字节；缩略图产物固定为按比例缩放后的 `.webp`，以便 Web、MCP、REST 和后续交付系统拿到稳定一致的预览文件。

## Source Context

- Product spec: 第一版必须对外提供稳定的 thumbnail output，而不只是返回原图。
- Input/output spec: `GET /assets/{asset_id}/thumbnail` 是冻结输出之一，Web UI 必须支持缩略图预览。
- Architecture: 缩略图应由 asset processor 生成，路径为 `thumbnails/{asset_id}/{version}.webp`；`THUMBNAIL_MAX_WIDTH` / `THUMBNAIL_MAX_HEIGHT` 是关键配置。
- Current code: `LocalStorage.StoreGeneratedFile` 仍直接写 provider 提供的 `file.Thumbnail` 到 `1.png`，还没有服务端统一 resize / webp 产物。

## User Flow

1. 调用方创建图片任务。
2. Worker 调 provider 获取原图结果。
3. 服务端把原图写入临时目录，并基于原图生成统一缩略图。
4. 服务端将原图、`.webp` 缩略图和 metadata 一起原子落盘并登记资产版本。
5. Web 或外部系统访问 `GET /api/assets/{asset_id}/thumbnail` 时，拿到真实的 `image/webp` 缩略图。

## In Scope

- 服务端基于原图生成真实缩略图，而不是依赖 provider 返回的 thumbnail bytes。
- 缩略图统一输出为 `.webp`。
- 缩略图按比例缩放，受最大宽高配置约束。
- `GET /api/assets/{asset_id}/thumbnail` 返回正确的 `image/webp` MIME。
- Docker 运行环境补齐生成 WebP 缩略图所需的最小工具链。

## Out of Scope

- 不重做原图输出格式；原图仍保持 provider 当前写入格式。
- 不做多尺寸缩略图。
- 不做对象存储/CDN 缩略图派生。
- 不实现视觉质量评分或缩略图缓存层。
- 不改 Web 详情页/卡片交互。

## Acceptance Criteria

- Given 一个成功生成的资产，when Worker 落盘，then `thumbnail_path` 指向 `thumbnails/{asset_id}/1.webp`，且文件真实存在。
- Given 一个 `GET /api/assets/{asset_id}/thumbnail` 请求，when 资产版本 ready，then 响应 `Content-Type` 为 `image/webp`。
- Given 横图、竖图和方图，when 服务端生成缩略图，then 缩略图宽高保持原图比例，并且不超过配置的最大宽高。
- Given provider 没有提供可信缩略图，when 服务端处理原图，then 仍能生成统一的 `.webp` 缩略图。

## Technical Approach

- 在 `internal/storage` 中把缩略图生成从 “写入 provider thumbnail bytes” 改为 “基于原图文件调用本地 WebP 工具生成”。
- 使用配置 `THUMBNAIL_MAX_WIDTH`、`THUMBNAIL_MAX_HEIGHT` 控制缩略图尺寸，默认与当前 Web 本地缩略图上限对齐。
- Docker runtime 安装轻量 WebP CLI 工具，避免新增 Go 图像编码第三方依赖。
- `app.Service.GetAssetFile` 按缩略图实际扩展名返回 MIME，而不是写死 `image/png`。

## Data / Interface Impact

- 不新增 REST/MCP 字段。
- 不新增数据库表。
- `asset_version.thumbnail_path` 的文件扩展名由 `.png` 切换为 `.webp`。
- 新增配置：`THUMBNAIL_MAX_WIDTH`、`THUMBNAIL_MAX_HEIGHT`。

## Files or Subsystems Likely to Change

- `internal/storage/local.go`
- `internal/storage/*_test.go`
- `internal/config/config.go`
- `internal/app/service.go`
- `cmd/api/main.go`
- `cmd/worker/main.go`
- `cmd/vag/main.go`
- `cmd/mcp/main.go`
- `Dockerfile`
- `docker-compose.yml`
- `docs/project/*`

## Verification Plan

```bash
docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine /usr/local/go/bin/go test ./...
docker compose build api worker
docker compose up -d postgres redis api worker
# smoke: create mock task, inspect task result and verify thumbnail path ends with .webp,
# GET /api/assets/{asset_id}/thumbnail returns image/webp, and file exists in storage volume.
```

## Assumptions and Risks

- Go 标准库没有 WebP 编码器；本片默认使用运行时镜像中的 WebP CLI 工具生成缩略图，而不是引入新的 Go 图像依赖。
- 本片默认以 Docker Compose 作为标准运行环境；若直接在宿主机运行二进制，需要确保系统 PATH 中存在相同工具。
- 真实 provider 当前都会返回可落盘原图；如果后续接入更复杂格式，可能需要补更多解码兼容性。

## Implementation Log

### 2026-06-18

- Changes:
  - `LocalStorage` 改为基于原图统一生成缩略图，不再直接保存 provider thumbnail bytes。
  - 缩略图路径切换为 `thumbnails/{asset_id}/1.webp`，并按最大宽高约束执行 resize。
  - `GET /api/assets/{asset_id}/thumbnail` 根据文件扩展名返回真实 MIME，当前为 `image/webp`。
  - Docker runtime image 安装 `libwebp-tools`，通过 `cwebp` 生成缩略图。
  - 新增 `THUMBNAIL_MAX_WIDTH`、`THUMBNAIL_MAX_HEIGHT` 配置并接入 API/Worker/CLI/MCP 启动路径。
- Verification:
  - `docker run --rm -v "$PWD":/app -w /app golang:1.25.3-alpine /usr/local/go/bin/go test ./...`
  - `docker compose config`
  - `docker compose build api worker`
  - Docker smoke：mock 任务完成后 `thumbnail_path` 为 `/data/.../thumbnails/<asset_id>/1.webp`，HTTP `GET /api/assets/<asset_id>/thumbnail` 返回 `Content-Type: image/webp`，文件头为 `RIFF....WEBP`。
- Remaining gaps: 项目级 API key、Web project/campaign 管理体验、真实 edit/mask 输入文件取回和更强 best-of 评分仍待后续 slice。
