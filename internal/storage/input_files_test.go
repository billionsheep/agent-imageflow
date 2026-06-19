package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestStoreAndReadTaskInputFile(t *testing.T) {
	root := t.TempDir()
	fs := NewLocalStorage(root, 720, 720)
	scope := domain.Scope{
		WorkspaceID: "ws_default",
		ProjectID:   "prj_demo",
		CampaignID:  "cmp_demo",
	}

	stored, err := fs.StoreTaskInputFile(context.Background(), scope, "inp_demo", domain.InputFileKindReference, "cover.png", "image/png", mustEncodePNG(t, 32, 24))
	if err != nil {
		t.Fatalf("StoreTaskInputFile returned error: %v", err)
	}
	if !strings.HasSuffix(stored.FilePath, filepath.Join("input-files", "inp_demo", "original.png")) {
		t.Fatalf("unexpected stored file path: %s", stored.FilePath)
	}
	if stored.Width != 32 || stored.Height != 24 {
		t.Fatalf("unexpected stored image size: %#v", stored)
	}

	metadataBytes, err := os.ReadFile(stored.MetadataPath)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		t.Fatalf("metadata is not valid json: %v", err)
	}
	if metadata["kind"] != domain.InputFileKindReference {
		t.Fatalf("unexpected metadata kind: %#v", metadata["kind"])
	}

	loaded, err := fs.GetTaskInputFile(scope, "inp_demo")
	if err != nil {
		t.Fatalf("GetTaskInputFile returned error: %v", err)
	}
	if loaded.FilePath != stored.FilePath || loaded.MimeType != "image/png" || loaded.SizeBytes <= 0 {
		t.Fatalf("unexpected loaded input file: %#v", loaded)
	}
}

func TestDeleteCampaignScopeDataRemovesStoredInputFiles(t *testing.T) {
	root := t.TempDir()
	fs := NewLocalStorage(root, 720, 720)
	scope := domain.Scope{
		WorkspaceID: "ws_default",
		ProjectID:   "prj_demo",
		CampaignID:  "cmp_cleanup",
	}

	stored, err := fs.StoreTaskInputFile(context.Background(), scope, "inp_cleanup", domain.InputFileKindReference, "cover.png", "image/png", mustEncodePNG(t, 16, 16))
	if err != nil {
		t.Fatalf("StoreTaskInputFile returned error: %v", err)
	}
	if _, err := os.Stat(stored.FilePath); err != nil {
		t.Fatalf("expected stored file to exist: %v", err)
	}

	if err := fs.DeleteCampaignScopeData(scope); err != nil {
		t.Fatalf("DeleteCampaignScopeData returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "workspaces", scope.WorkspaceID, "projects", scope.ProjectID, "campaigns", scope.CampaignID)); !os.IsNotExist(err) {
		t.Fatalf("expected campaign directory to be removed, got err=%v", err)
	}
}
