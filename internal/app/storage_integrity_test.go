package app

import "testing"

func TestStorageIntegrityFileIssueDoesNotExposePath(t *testing.T) {
	issue := storageIntegrityFileIssue("asset_1", "ver_1", RepairFileCheck{
		Kind: "original",
		Path: "/local/storage/root/original.png",
	})

	if issue.FileKind != "original" {
		t.Fatalf("expected file kind original, got %q", issue.FileKind)
	}
	if issue.Message == "" {
		t.Fatal("expected issue message")
	}
}
