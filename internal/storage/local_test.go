package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/provider"
)

func TestThumbnailDimensions(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		height    int
		maxWidth  int
		maxHeight int
		wantW     int
		wantH     int
	}{
		{name: "square", width: 1024, height: 1024, maxWidth: 720, maxHeight: 720, wantW: 720, wantH: 720},
		{name: "landscape", width: 1600, height: 900, maxWidth: 720, maxHeight: 720, wantW: 720, wantH: 405},
		{name: "portrait", width: 900, height: 1600, maxWidth: 720, maxHeight: 720, wantW: 405, wantH: 720},
		{name: "already_small", width: 300, height: 200, maxWidth: 720, maxHeight: 720, wantW: 300, wantH: 200},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotW, gotH, err := thumbnailDimensions(tc.width, tc.height, tc.maxWidth, tc.maxHeight)
			if err != nil {
				t.Fatalf("thumbnailDimensions returned error: %v", err)
			}
			if gotW != tc.wantW || gotH != tc.wantH {
				t.Fatalf("thumbnailDimensions() = %dx%d, want %dx%d", gotW, gotH, tc.wantW, tc.wantH)
			}
		})
	}
}

func TestStoreGeneratedFileCreatesWebPThumbnail(t *testing.T) {
	if _, err := exec.LookPath(defaultCWebPBinary); err != nil {
		t.Skipf("%s not found on PATH", defaultCWebPBinary)
	}

	root := t.TempDir()
	storage := NewLocalStorage(root, 720, 720)
	original := mustEncodePNG(t, 1600, 900)
	task := domain.Task{
		ID:          "task_test_thumbnail",
		WorkspaceID: "ws_default",
		ProjectID:   "prj_demo",
		CampaignID:  "cmp_demo",
		Prompt:      "生成一张封面图",
		Provider:    provider.MockProviderID,
	}
	stored, err := storage.StoreGeneratedFile(context.Background(), task, "asset_demo", "ver_demo", provider.GeneratedFile{
		Bytes:    original,
		MimeType: "image/png",
		Width:    1600,
		Height:   900,
		Model:    "mock-image-v1",
	})
	if err != nil {
		t.Fatalf("StoreGeneratedFile returned error: %v", err)
	}
	if !strings.HasSuffix(stored.ThumbnailPath, filepath.Join("thumbnails", "asset_demo", "1.webp")) {
		t.Fatalf("unexpected thumbnail path: %s", stored.ThumbnailPath)
	}
	thumbnailBytes, err := os.ReadFile(stored.ThumbnailPath)
	if err != nil {
		t.Fatalf("read thumbnail: %v", err)
	}
	if len(thumbnailBytes) < 12 || string(thumbnailBytes[:4]) != "RIFF" || string(thumbnailBytes[8:12]) != "WEBP" {
		t.Fatalf("thumbnail is not a webp file")
	}

	metadataBytes, err := os.ReadFile(stored.MetadataPath)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		t.Fatalf("metadata is not valid json: %v", err)
	}
	if metadata["thumbnail_mime"] != "image/webp" {
		t.Fatalf("unexpected thumbnail_mime: %#v", metadata["thumbnail_mime"])
	}
	if int(metadata["thumbnail_width"].(float64)) != 720 || int(metadata["thumbnail_height"].(float64)) != 405 {
		t.Fatalf("unexpected thumbnail size in metadata: %#v", metadata)
	}
}

func mustEncodePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / max(1, width-1)),
				G: uint8((y * 255) / max(1, height-1)),
				B: 180,
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}
