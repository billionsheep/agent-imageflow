package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestMaterializeFinalDeliveryMirrorWritesBatchReadableLayout(t *testing.T) {
	root := t.TempDir()
	sourceDir := t.TempDir()
	originalOne := writeTestFile(t, filepath.Join(sourceDir, "asset_caption_selected.png"), "original-one")
	thumbnailOne := writeTestFile(t, filepath.Join(sourceDir, "asset_caption_selected.webp"), "thumb-one")
	originalTwo := writeTestFile(t, filepath.Join(sourceDir, "asset_scene_two.png"), "original-two")
	thumbnailTwo := writeTestFile(t, filepath.Join(sourceDir, "asset_scene_two.webp"), "thumb-two")

	manifest := domain.BatchManifestResponse{
		GeneratedAt:  time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC),
		ProjectID:    "prj_demo",
		CampaignID:   "cmp_demo",
		SessionID:    "session_demo",
		BatchID:      "batch_demo",
		ManifestView: domain.BatchManifestViewFinalDelivery,
		SelectedOnly: true,
		FinalDelivery: &domain.BatchFinalDeliveryManifest{
			Counts: domain.BatchFinalDeliveryCounts{
				StoryCount:               1,
				SceneCount:               2,
				SceneWithFinalAssetCount: 2,
				FinalAssetCount:          2,
			},
			FinalAssets: []domain.BatchFinalDeliveryAsset{
				{
					AssetID:      "asset_caption_selected",
					StoryID:      "story_mochi",
					SceneID:      "scene_001",
					TargetPath:   "stories/story_mochi/scene_001-caption.png",
					DownloadURL:  "/api/assets/asset_caption_selected/original",
					ThumbnailURL: "/api/assets/asset_caption_selected/thumbnail",
					MetadataURL:  "/api/assets/asset_caption_selected/metadata",
				},
				{
					AssetID:      "asset_scene_two",
					StoryID:      "story_mochi",
					SceneID:      "scene_002",
					TargetPath:   "stories/story_mochi/scene_002.png",
					DownloadURL:  "/api/assets/asset_scene_two/original",
					ThumbnailURL: "/api/assets/asset_scene_two/thumbnail",
					MetadataURL:  "/api/assets/asset_scene_two/metadata",
				},
			},
		},
	}

	result, err := materializeFinalDeliveryMirror(root, domain.Scope{
		WorkspaceID: "ws_demo",
		ProjectID:   "prj_demo",
		CampaignID:  "cmp_demo",
	}, manifest, func(assetID string) (domain.AssetWithVersion, error) {
		switch assetID {
		case "asset_caption_selected":
			return domain.AssetWithVersion{
				Asset: domain.Asset{ID: assetID},
				Version: domain.AssetVersion{
					FilePath:      originalOne,
					ThumbnailPath: thumbnailOne,
				},
			}, nil
		case "asset_scene_two":
			return domain.AssetWithVersion{
				Asset: domain.Asset{ID: assetID},
				Version: domain.AssetVersion{
					FilePath:      originalTwo,
					ThumbnailPath: thumbnailTwo,
				},
			}, nil
		default:
			t.Fatalf("unexpected lookup asset id %q", assetID)
			return domain.AssetWithVersion{}, nil
		}
	})
	if err != nil {
		t.Fatalf("materializeFinalDeliveryMirror: %v", err)
	}

	wantRelative := filepath.ToSlash(filepath.Join("workspaces", "ws_demo", "projects", "prj_demo", "campaigns", "cmp_demo", "sessions", "session_demo", "batches", "batch_demo"))
	if got := filepath.ToSlash(result.MirrorRelativePath); got != wantRelative {
		t.Fatalf("mirror relative path = %q, want %q", got, wantRelative)
	}
	if result.FinalAssetCount != 2 || result.SceneCount != 2 {
		t.Fatalf("unexpected mirror counts: %#v", result)
	}

	expectFileContent(t, filepath.Join(root, result.FinalDir, "stories", "story_mochi", "scene_001-caption.png"), "original-one")
	expectFileContent(t, filepath.Join(root, result.FinalDir, "stories", "story_mochi", "scene_002.png"), "original-two")
	expectFileContent(t, filepath.Join(root, result.ThumbnailDir, "stories", "story_mochi", "scene_001-caption.webp"), "thumb-one")
	expectFileContent(t, filepath.Join(root, result.ThumbnailDir, "stories", "story_mochi", "scene_002.webp"), "thumb-two")

	manifestBytes, err := os.ReadFile(filepath.Join(root, result.ManifestFile))
	if err != nil {
		t.Fatalf("read manifest.final.json: %v", err)
	}
	var written domain.BatchManifestResponse
	if err := json.Unmarshal(manifestBytes, &written); err != nil {
		t.Fatalf("unmarshal written manifest: %v", err)
	}
	if written.ManifestView != domain.BatchManifestViewFinalDelivery || written.FinalDelivery == nil || len(written.FinalDelivery.FinalAssets) != 2 {
		t.Fatalf("written manifest missing final delivery contract: %#v", written)
	}
}

func TestMaterializeFinalDeliveryMirrorFallsBackToStorySceneFileName(t *testing.T) {
	root := t.TempDir()
	sourceDir := t.TempDir()
	original := writeTestFile(t, filepath.Join(sourceDir, "asset_scene.png"), "scene-original")
	thumbnail := writeTestFile(t, filepath.Join(sourceDir, "asset_scene.webp"), "scene-thumb")

	manifest := domain.BatchManifestResponse{
		BatchID:      "batch_demo",
		ManifestView: domain.BatchManifestViewFinalDelivery,
		SelectedOnly: true,
		FinalDelivery: &domain.BatchFinalDeliveryManifest{
			Counts: domain.BatchFinalDeliveryCounts{
				SceneCount:      1,
				FinalAssetCount: 1,
			},
			FinalAssets: []domain.BatchFinalDeliveryAsset{{
				AssetID:      "asset_scene",
				StoryID:      "story_alpha",
				SceneID:      "scene_009",
				DownloadURL:  "/api/assets/asset_scene/original",
				ThumbnailURL: "/api/assets/asset_scene/thumbnail",
				MetadataURL:  "/api/assets/asset_scene/metadata",
			}},
		},
	}

	result, err := materializeFinalDeliveryMirror(root, domain.Scope{
		WorkspaceID: "ws_demo",
		ProjectID:   "prj_demo",
		CampaignID:  "cmp_demo",
	}, manifest, func(assetID string) (domain.AssetWithVersion, error) {
		return domain.AssetWithVersion{
			Asset: domain.Asset{ID: assetID},
			Version: domain.AssetVersion{
				FilePath:      original,
				ThumbnailPath: thumbnail,
			},
		}, nil
	})
	if err != nil {
		t.Fatalf("materializeFinalDeliveryMirror: %v", err)
	}

	expectFileContent(t, filepath.Join(root, result.FinalDir, "stories", "story_alpha", "scene_009.png"), "scene-original")
	expectFileContent(t, filepath.Join(root, result.ThumbnailDir, "stories", "story_alpha", "scene_009.webp"), "scene-thumb")
}

func TestMaterializeFinalDeliveryMirrorRejectsTargetPathTraversal(t *testing.T) {
	root := t.TempDir()
	sourceDir := t.TempDir()
	original := writeTestFile(t, filepath.Join(sourceDir, "asset_scene.png"), "scene-original")
	thumbnail := writeTestFile(t, filepath.Join(sourceDir, "asset_scene.webp"), "scene-thumb")

	manifest := domain.BatchManifestResponse{
		BatchID:      "batch_demo",
		ManifestView: domain.BatchManifestViewFinalDelivery,
		SelectedOnly: true,
		FinalDelivery: &domain.BatchFinalDeliveryManifest{
			Counts: domain.BatchFinalDeliveryCounts{
				SceneCount:      1,
				FinalAssetCount: 1,
			},
			FinalAssets: []domain.BatchFinalDeliveryAsset{{
				AssetID:    "asset_scene",
				StoryID:    "story_alpha",
				SceneID:    "scene_009",
				TargetPath: "../escape.png",
			}},
		},
	}

	_, err := materializeFinalDeliveryMirror(root, domain.Scope{
		WorkspaceID: "ws_demo",
		ProjectID:   "prj_demo",
		CampaignID:  "cmp_demo",
	}, manifest, func(assetID string) (domain.AssetWithVersion, error) {
		return domain.AssetWithVersion{
			Asset: domain.Asset{ID: assetID},
			Version: domain.AssetVersion{
				FilePath:      original,
				ThumbnailPath: thumbnail,
			},
		}, nil
	})
	if err == nil {
		t.Fatal("expected target path traversal error")
	}
	if !strings.Contains(err.Error(), "target_path") {
		t.Fatalf("expected target_path error, got %v", err)
	}
}

func TestMaterializeFinalDeliveryMirrorRejectsFallbackTraversal(t *testing.T) {
	root := t.TempDir()
	sourceDir := t.TempDir()
	original := writeTestFile(t, filepath.Join(sourceDir, "asset_scene.png"), "scene-original")
	thumbnail := writeTestFile(t, filepath.Join(sourceDir, "asset_scene.webp"), "scene-thumb")

	manifest := domain.BatchManifestResponse{
		BatchID:      "batch_demo",
		ManifestView: domain.BatchManifestViewFinalDelivery,
		SelectedOnly: true,
		FinalDelivery: &domain.BatchFinalDeliveryManifest{
			Counts: domain.BatchFinalDeliveryCounts{
				SceneCount:      1,
				FinalAssetCount: 1,
			},
			FinalAssets: []domain.BatchFinalDeliveryAsset{{
				AssetID: "asset_scene",
				SceneID: "../escape",
			}},
		},
	}

	_, err := materializeFinalDeliveryMirror(root, domain.Scope{
		WorkspaceID: "ws_demo",
		ProjectID:   "prj_demo",
		CampaignID:  "cmp_demo",
	}, manifest, func(assetID string) (domain.AssetWithVersion, error) {
		return domain.AssetWithVersion{
			Asset: domain.Asset{ID: assetID},
			Version: domain.AssetVersion{
				FilePath:      original,
				ThumbnailPath: thumbnail,
			},
		}, nil
	})
	if err == nil {
		t.Fatal("expected fallback traversal error")
	}
	if !strings.Contains(err.Error(), "target_path") {
		t.Fatalf("expected target_path error, got %v", err)
	}
}

func writeTestFile(t *testing.T, path string, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func expectFileContent(t *testing.T, path string, want string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if got := string(raw); got != want {
		t.Fatalf("content %s = %q, want %q", path, got, want)
	}
}
