package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestScanUsageMissingRootReturnsZero(t *testing.T) {
	fs := NewLocalStorage(filepath.Join(t.TempDir(), "missing"), 720, 720)
	usage, err := fs.ScanUsage(context.Background(), UsageScanOptions{
		Scope: domain.Scope{WorkspaceID: "ws", ProjectID: "prj", CampaignID: "cmp"},
	})
	if err != nil {
		t.Fatalf("ScanUsage returned error: %v", err)
	}
	if usage.Instance.Bytes != 0 || usage.Campaign.FileCount != 0 {
		t.Fatalf("expected zero usage for missing root, got %#v", usage)
	}
}

func TestScanUsageEmptyRootReturnsZero(t *testing.T) {
	fs := NewLocalStorage(t.TempDir(), 720, 720)
	usage, err := fs.ScanUsage(context.Background(), UsageScanOptions{
		Scope: domain.Scope{WorkspaceID: "ws", ProjectID: "prj", CampaignID: "cmp"},
	})
	if err != nil {
		t.Fatalf("ScanUsage returned error: %v", err)
	}
	if usage.Instance.Bytes != 0 || len(usage.Instance.Categories) != 0 {
		t.Fatalf("expected zero usage for empty root, got %#v", usage.Instance)
	}
}

func TestScanUsageCategorizesScopeFilesAndOrphans(t *testing.T) {
	root := t.TempDir()
	fs := NewLocalStorage(root, 720, 720)
	scope := domain.Scope{WorkspaceID: "ws", ProjectID: "prj", CampaignID: "cmp"}

	original := writeSizedFile(t, root, "workspaces/ws/projects/prj/campaigns/cmp/originals/asset_1/1.png", 11)
	thumbnail := writeSizedFile(t, root, "workspaces/ws/projects/prj/campaigns/cmp/thumbnails/asset_1/1.webp", 7)
	metadata := writeSizedFile(t, root, "workspaces/ws/projects/prj/campaigns/cmp/metadata/asset_1/1.json", 5)
	writeSizedFile(t, root, "workspaces/ws/projects/prj/campaigns/cmp/input-files/input_1/original.png", 13)
	writeSizedFile(t, root, "workspaces/ws/projects/prj/campaigns/cmp/tmp/asset_tmp/original.png", 17)
	writeSizedFile(t, root, "workspaces/ws/projects/prj/campaigns/cmp/originals/asset_orphan/1.png", 19)
	writeSizedFile(t, root, "audit/http-api/2026-06-19.jsonl", 23)
	writeSizedFile(t, root, "workspaces/ws/projects/other/campaigns/cmp/originals/asset_other/1.png", 29)

	usage, err := fs.ScanUsage(context.Background(), UsageScanOptions{
		Scope:               scope,
		KnownAssetFilePaths: []string{original, thumbnail, metadata},
	})
	if err != nil {
		t.Fatalf("ScanUsage returned error: %v", err)
	}

	if usage.Instance.Bytes != 124 {
		t.Fatalf("unexpected instance bytes: %d", usage.Instance.Bytes)
	}
	if usage.Campaign.Bytes != 72 {
		t.Fatalf("unexpected campaign bytes: %d", usage.Campaign.Bytes)
	}
	expectCategory(t, usage.Campaign, UsageCategoryOriginal, 1, 11)
	expectCategory(t, usage.Campaign, UsageCategoryThumbnail, 1, 7)
	expectCategory(t, usage.Campaign, UsageCategoryMetadata, 1, 5)
	expectCategory(t, usage.Campaign, UsageCategoryInputFiles, 1, 13)
	expectCategory(t, usage.Campaign, UsageCategoryTmp, 1, 17)
	expectCategory(t, usage.Campaign, UsageCategoryOrphan, 1, 19)
	expectCategory(t, usage.Instance, UsageCategoryAudit, 1, 23)
}

func TestScanUsageUsesFileInfoForLargeFiles(t *testing.T) {
	root := t.TempDir()
	fs := NewLocalStorage(root, 720, 720)
	path := filepath.Join(root, "workspaces/ws/projects/prj/campaigns/cmp/originals/asset_big/1.bin")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	const size = int64(8 << 20)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := file.Truncate(size); err != nil {
		file.Close()
		t.Fatalf("truncate: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	usage, err := fs.ScanUsage(context.Background(), UsageScanOptions{
		Scope:               domain.Scope{WorkspaceID: "ws", ProjectID: "prj", CampaignID: "cmp"},
		KnownAssetFilePaths: []string{path},
	})
	if err != nil {
		t.Fatalf("ScanUsage returned error: %v", err)
	}
	expectCategory(t, usage.Campaign, UsageCategoryOriginal, 1, size)
}

func TestDeleteStorageKeyDeletesOnlyRelativeFiles(t *testing.T) {
	root := t.TempDir()
	fs := NewLocalStorage(root, 720, 720)
	rel := "workspaces/ws/projects/prj/campaigns/cmp/tmp/asset_1/original.png"
	path := writeSizedFile(t, root, rel, 11)

	deletedBytes, err := fs.DeleteStorageKey(rel)
	if err != nil {
		t.Fatalf("DeleteStorageKey returned error: %v", err)
	}
	if deletedBytes != 11 {
		t.Fatalf("deleted bytes = %d, want 11", deletedBytes)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file to be deleted, stat err=%v", err)
	}
}

func TestDeleteStorageKeyRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	fs := NewLocalStorage(root, 720, 720)
	for _, key := range []string{"/tmp/outside.png", "../outside.png", "workspaces/../../outside.png", ""} {
		t.Run(key, func(t *testing.T) {
			if _, err := fs.DeleteStorageKey(key); err == nil {
				t.Fatalf("expected unsafe key %q to be rejected", key)
			}
		})
	}
}

func writeSizedFile(t *testing.T, root, rel string, size int64) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, make([]byte, int(size)), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
	return path
}

func expectCategory(t *testing.T, snapshot domain.StorageUsageSnapshot, category string, files int64, bytes int64) {
	t.Helper()
	for _, item := range snapshot.Categories {
		if item.Category != category {
			continue
		}
		if item.FileCount != files || item.Bytes != bytes {
			t.Fatalf("category %s = files %d bytes %d, want files %d bytes %d", category, item.FileCount, item.Bytes, files, bytes)
		}
		return
	}
	t.Fatalf("category %s not found in %#v", category, snapshot.Categories)
}
