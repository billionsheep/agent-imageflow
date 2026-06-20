package provider

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"sync"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

type GeneratedFile struct {
	Slot          int
	Bytes         []byte
	Thumbnail     []byte
	MimeType      string
	Width         int
	Height        int
	ThumbnailW    int
	ThumbnailH    int
	Model         string
	ParametersRaw []byte
	CostRaw       []byte
}

type Result struct {
	ProviderRequestID string
	Status            string
	Files             []GeneratedFile
	RawResponse       []byte
	CostRaw           []byte
	ErrorCode         string
	ErrorMessage      string
	Metrics           domain.AttemptMetrics
}

type MockProvider struct{}

var mockTransientFailures sync.Map

func (p MockProvider) Generate(ctx context.Context, task domain.Task) (Result, error) {
	if shouldFailMockTransientOnce(task) {
		raw := []byte(fmt.Sprintf(`{"provider_request_id":"mock_%s","status":"failed","error_code":"temporary_unavailable"}`, task.ID))
		return Result{
			ProviderRequestID: "mock_" + task.ID,
			Status:            "failed",
			RawResponse:       raw,
			CostRaw:           []byte(`{"provider":"mock","estimated_cost":0}`),
			ErrorCode:         "temporary_unavailable",
			ErrorMessage:      "mock transient failure, please retry",
		}, fmt.Errorf("mock transient failure, please retry")
	}
	if delay := mockDelay(task); delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return Result{
				ProviderRequestID: "mock_" + task.ID,
				Status:            "failed",
				CostRaw:           []byte(`{"provider":"mock","estimated_cost":0}`),
				ErrorCode:         "mock_canceled",
				ErrorMessage:      ctx.Err().Error(),
			}, ctx.Err()
		}
	}

	count := task.RequestedCount
	if count < 1 {
		count = 1
	}
	if maxN := taskProviderMaxN(task, MockProviderID, 4); count > maxN {
		count = maxN
	}

	width, height := dimensions(task.AspectRatio)
	files := make([]GeneratedFile, 0, count)
	for i := 0; i < count; i++ {
		seed := fmt.Sprintf("%s:%s:%d", task.ID, task.Prompt, i)
		original, err := renderPNG(seed, width, height)
		if err != nil {
			return Result{}, err
		}
		thumbW, thumbH := thumbnailDimensions(width, height, 360)
		thumbnail, err := renderPNG(seed+":thumb", thumbW, thumbH)
		if err != nil {
			return Result{}, err
		}
		parameters := taskProviderParameters(task, map[string]any{
			"aspect_ratio":  task.AspectRatio,
			"output_format": task.OutputFormat,
			"style_preset":  task.StylePreset,
			"slot":          i,
		})
		cost := []byte(`{"provider":"mock","estimated_cost":0}`)
		files = append(files, GeneratedFile{
			Slot:          i,
			Bytes:         original,
			Thumbnail:     thumbnail,
			MimeType:      "image/png",
			Width:         width,
			Height:        height,
			ThumbnailW:    thumbW,
			ThumbnailH:    thumbH,
			Model:         "mock-image-v1",
			ParametersRaw: parameters,
			CostRaw:       cost,
		})
	}

	raw := []byte(fmt.Sprintf(`{"provider_request_id":"mock_%s","status":"succeeded","count":%d}`, task.ID, count))
	return Result{
		ProviderRequestID: "mock_" + task.ID,
		Status:            "succeeded",
		Files:             files,
		RawResponse:       raw,
		CostRaw:           []byte(`{"provider":"mock","estimated_cost":0}`),
	}, nil
}

func shouldFailMockTransientOnce(task domain.Task) bool {
	if task.ID == "" || mockFailureMode(task) != "transient_once" {
		return false
	}
	_, loaded := mockTransientFailures.LoadOrStore(task.ID, true)
	return !loaded
}

func mockFailureMode(task domain.Task) string {
	config := mockGenerationConfig(task)
	return config.MockFailureMode
}

func mockDelay(task domain.Task) time.Duration {
	config := mockGenerationConfig(task)
	if config.MockDelayMs <= 0 {
		return 0
	}
	if config.MockDelayMs > 10_000 {
		config.MockDelayMs = 10_000
	}
	return time.Duration(config.MockDelayMs) * time.Millisecond
}

type mockGenerationConfigPayload struct {
	MockFailureMode string `json:"mock_failure_mode"`
	MockDelayMs     int    `json:"mock_delay_ms"`
}

func mockGenerationConfig(task domain.Task) mockGenerationConfigPayload {
	var input struct {
		GenerationConfig json.RawMessage `json:"generation_config"`
	}
	if len(task.StructuredInputJSON) == 0 || json.Unmarshal(task.StructuredInputJSON, &input) != nil || len(input.GenerationConfig) == 0 {
		return mockGenerationConfigPayload{}
	}
	var generationConfig mockGenerationConfigPayload
	if json.Unmarshal(input.GenerationConfig, &generationConfig) != nil {
		return mockGenerationConfigPayload{}
	}
	return generationConfig
}

func dimensions(aspectRatio string) (int, int) {
	switch aspectRatio {
	case "3:4":
		return 1200, 1600
	case "4:3":
		return 1600, 1200
	case "16:9":
		return 1600, 900
	case "9:16":
		return 900, 1600
	default:
		return 1024, 1024
	}
}

func thumbnailDimensions(width, height, maxSide int) (int, int) {
	if width <= maxSide && height <= maxSide {
		return width, height
	}
	if width >= height {
		return maxSide, maxSide * height / width
	}
	return maxSide * width / height, maxSide
}

func renderPNG(seed string, width, height int) ([]byte, error) {
	hash := sha256.Sum256([]byte(seed))
	bg := color.RGBA{R: hash[0], G: hash[1], B: hash[2], A: 255}
	accent := color.RGBA{R: hash[8], G: hash[9], B: hash[10], A: 255}
	light := color.RGBA{R: 245, G: 247, B: 250, A: 255}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	for y := 0; y < height; y += max(8, height/18) {
		for x := 0; x < width; x += max(8, width/18) {
			value := hash[(x+y)%len(hash)]
			c := accent
			if value%3 == 0 {
				c = light
			}
			rect := image.Rect(x, y, min(width, x+width/10), min(height, y+height/10))
			draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Over)
		}
	}

	for i := 0; i < 4; i++ {
		offset := int(binary.BigEndian.Uint16(hash[i*2 : i*2+2]))
		x := offset % max(1, width)
		y := (offset / max(1, width)) % max(1, height)
		rect := image.Rect(max(0, x-width/8), max(0, y-height/12), min(width, x+width/8), min(height, y+height/12))
		draw.Draw(img, rect, &image.Uniform{C: color.RGBA{255, 255, 255, 160}}, image.Point{}, draw.Over)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
