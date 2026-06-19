package app

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestFetchRemoteTaskInputFileDownloadsImage(t *testing.T) {
	imageBytes := appTestPNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(imageBytes)
	}))
	defer server.Close()

	download, err := fetchRemoteTaskInputFile(context.Background(), server.Client(), server.URL+"/nested/cover.png")
	if err != nil {
		t.Fatalf("fetchRemoteTaskInputFile returned error: %v", err)
	}
	if download.OriginalFilename != "cover.png" {
		t.Fatalf("unexpected filename: %q", download.OriginalFilename)
	}
	if download.MimeType != "image/png" {
		t.Fatalf("unexpected mime type: %q", download.MimeType)
	}
	if !bytes.Equal(download.Raw, imageBytes) {
		t.Fatal("downloaded bytes do not match source image")
	}
}

func TestFetchRemoteTaskInputFileRejectsUnsupportedScheme(t *testing.T) {
	_, err := fetchRemoteTaskInputFile(context.Background(), &http.Client{}, "file:///tmp/input.png")
	if err == nil {
		t.Fatal("expected unsupported scheme to fail")
	}
}

func TestRemoteTaskInputFilenameFallsBackWhenPathEmpty(t *testing.T) {
	parsed, err := neturl.Parse("https://example.com/")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	if got := remoteTaskInputFilename(parsed); got != "remote-input" {
		t.Fatalf("unexpected fallback filename: %q", got)
	}
}

func TestValidateReusableAssetScopeRejectsCrossProject(t *testing.T) {
	filePath := writeReusableAssetTestFile(t)
	item := domain.AssetWithVersion{
		Asset: domain.Asset{
			ID:          "asset_existing",
			WorkspaceID: "ws_other",
			ProjectID:   "prj_other",
		},
		Version: domain.AssetVersion{
			Status:   domain.VersionReady,
			FilePath: filePath,
			MimeType: "image/png",
		},
	}
	err := validateReusableAssetScope(domain.Scope{
		WorkspaceID: "ws_default",
		ProjectID:   "prj_default",
		CampaignID:  "cmp_default",
	}, item)
	if err == nil {
		t.Fatal("expected cross-project asset reuse to fail")
	}
}

func TestValidateReusableAssetScopeAllowsSameProjectDifferentCampaign(t *testing.T) {
	filePath := writeReusableAssetTestFile(t)
	item := domain.AssetWithVersion{
		Asset: domain.Asset{
			ID:          "asset_existing",
			WorkspaceID: "ws_default",
			ProjectID:   "prj_default",
			CampaignID:  "cmp_history",
		},
		Version: domain.AssetVersion{
			Status:   domain.VersionReady,
			FilePath: filePath,
			MimeType: "image/png",
		},
	}
	if err := validateReusableAssetScope(domain.Scope{
		WorkspaceID: "ws_default",
		ProjectID:   "prj_default",
		CampaignID:  "cmp_new",
	}, item); err != nil {
		t.Fatalf("expected same-project asset reuse to succeed: %v", err)
	}
}

func writeReusableAssetTestFile(t *testing.T) string {
	t.Helper()
	filePath := t.TempDir() + "/original.png"
	if err := os.WriteFile(filePath, appTestPNG(t), 0o644); err != nil {
		t.Fatalf("write test asset file: %v", err)
	}
	return filePath
}

func appTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 0, color.RGBA{G: 255, A: 255})
	img.Set(0, 1, color.RGBA{B: 255, A: 255})
	img.Set(1, 1, color.RGBA{R: 255, G: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}
