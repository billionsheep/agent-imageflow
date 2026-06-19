package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultThumbnailMaxWidth  = 720
	defaultThumbnailMaxHeight = 720
	defaultThumbnailQuality   = 90
	defaultCWebPBinary        = "cwebp"
)

func fileExtensionForMimeType(mimeType string) string {
	switch strings.ToLower(strings.TrimSpace(mimeType)) {
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".png"
	}
}

func MimeTypeForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".webp":
		return "image/webp"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".png":
		return "image/png"
	default:
		if detected := mime.TypeByExtension(filepath.Ext(path)); detected != "" {
			return detected
		}
		return "application/octet-stream"
	}
}

func effectiveThumbnailBound(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func thumbnailDimensions(width, height, maxWidth, maxHeight int) (int, int, error) {
	if width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("invalid image size width=%d height=%d", width, height)
	}
	maxWidth = effectiveThumbnailBound(maxWidth, defaultThumbnailMaxWidth)
	maxHeight = effectiveThumbnailBound(maxHeight, defaultThumbnailMaxHeight)
	scale := math.Min(1, math.Min(float64(maxWidth)/float64(width), float64(maxHeight)/float64(height)))
	return max(1, int(math.Round(float64(width)*scale))), max(1, int(math.Round(float64(height)*scale))), nil
}

func detectImageDimensions(raw []byte) (int, int, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return 0, 0, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0, fmt.Errorf("invalid image size width=%d height=%d", cfg.Width, cfg.Height)
	}
	return cfg.Width, cfg.Height, nil
}

func (s LocalStorage) createThumbnail(ctx context.Context, originalPath, thumbnailPath string, width, height int) (int, int, error) {
	thumbW, thumbH, err := thumbnailDimensions(width, height, s.thumbnailMaxWidth, s.thumbnailMaxHeight)
	if err != nil {
		return 0, 0, err
	}
	cmd := exec.CommandContext(
		ctx,
		s.cwebpBinary,
		"-quiet",
		"-q",
		strconv.Itoa(defaultThumbnailQuality),
		"-resize",
		strconv.Itoa(thumbW),
		strconv.Itoa(thumbH),
		originalPath,
		"-o",
		thumbnailPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return 0, 0, fmt.Errorf("thumbnail encoder %q is not available on PATH", s.cwebpBinary)
		}
		if len(output) > 0 {
			return 0, 0, fmt.Errorf("generate webp thumbnail: %s", strings.TrimSpace(string(output)))
		}
		return 0, 0, err
	}
	if info, statErr := os.Stat(thumbnailPath); statErr != nil {
		return 0, 0, statErr
	} else if info.Size() == 0 {
		return 0, 0, fmt.Errorf("generated thumbnail is empty")
	}
	return thumbW, thumbH, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
