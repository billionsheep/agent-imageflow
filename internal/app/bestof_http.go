package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
	"github.com/billionsheep/agent-imageflow/internal/storage"
)

const maxBestOfJudgeFileBytes int64 = 12 << 20

type httpJudgeBestOfScorer struct {
	url        string
	apiKey     string
	httpClient *http.Client
}

type httpJudgeScoreRequest struct {
	Strategy    string                    `json:"strategy"`
	JudgePrompt string                    `json:"judge_prompt,omitempty"`
	Task        httpJudgeTaskPayload      `json:"task"`
	Candidates  []httpJudgeCandidateInput `json:"candidates"`
}

type httpJudgeTaskPayload struct {
	TaskID         string `json:"task_id"`
	Title          string `json:"title"`
	Purpose        string `json:"purpose,omitempty"`
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	StylePreset    string `json:"style_preset,omitempty"`
	AspectRatio    string `json:"aspect_ratio,omitempty"`
	SelectionMode  string `json:"selection_mode,omitempty"`
	RequestedCount int    `json:"requested_count"`
}

type httpJudgeCandidateInput struct {
	AssetID        string `json:"asset_id"`
	AssetVersionID string `json:"asset_version_id"`
	MimeType       string `json:"mime_type"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	ImageDataURL   string `json:"image_data_url"`
}

type httpJudgeScoreResponse struct {
	SelectedAssetID string               `json:"selected_asset_id"`
	Scores          []httpJudgeScoreItem `json:"scores"`
	Model           string               `json:"model,omitempty"`
}

type httpJudgeScoreItem struct {
	AssetID string   `json:"asset_id"`
	Score   float64  `json:"score"`
	Reasons []string `json:"reasons,omitempty"`
}

func newHTTPJudgeBestOfScorer(url, apiKey string, timeoutSeconds int) httpJudgeBestOfScorer {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return httpJudgeBestOfScorer{
		url:        strings.TrimSpace(url),
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (s httpJudgeBestOfScorer) Configured() bool {
	return strings.TrimSpace(s.url) != ""
}

func (s httpJudgeBestOfScorer) Score(ctx context.Context, task domain.Task, assets []domain.AssetWithVersion, config *domain.BestOfConfig) (bestOfSelectionResult, error) {
	if !s.Configured() {
		return bestOfSelectionResult{}, fmt.Errorf("best_of strategy %q is not configured", domain.BestOfStrategyHTTPJudge)
	}
	requestPayload, assetIndex, err := s.requestPayload(task, assets, config)
	if err != nil {
		return bestOfSelectionResult{}, err
	}
	body, err := json.Marshal(requestPayload)
	if err != nil {
		return bestOfSelectionResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil {
		return bestOfSelectionResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return bestOfSelectionResult{}, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return bestOfSelectionResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return bestOfSelectionResult{}, fmt.Errorf("http judge scorer failed: %s", httpJudgeErrorMessage(respBytes, resp.Status))
	}

	var parsed httpJudgeScoreResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return bestOfSelectionResult{}, fmt.Errorf("parse http judge scorer response: %w", err)
	}
	return selectHTTPJudgeCandidate(task, assetIndex, parsed)
}

func (s httpJudgeBestOfScorer) requestPayload(task domain.Task, assets []domain.AssetWithVersion, config *domain.BestOfConfig) (httpJudgeScoreRequest, map[string]domain.AssetWithVersion, error) {
	candidates := make([]httpJudgeCandidateInput, 0, len(assets))
	assetIndex := make(map[string]domain.AssetWithVersion, len(assets))
	for _, item := range assets {
		imageDataURL, mimeType, err := judgeImageDataURL(item)
		if err != nil {
			return httpJudgeScoreRequest{}, nil, fmt.Errorf("load scorer input for asset %s: %w", item.ID, err)
		}
		candidates = append(candidates, httpJudgeCandidateInput{
			AssetID:        item.ID,
			AssetVersionID: item.Version.ID,
			MimeType:       mimeType,
			Width:          item.Version.Width,
			Height:         item.Version.Height,
			ImageDataURL:   imageDataURL,
		})
		assetIndex[item.ID] = item
	}

	payload := httpJudgeScoreRequest{
		Strategy: domain.BestOfStrategyHTTPJudge,
		Task: httpJudgeTaskPayload{
			TaskID:         task.ID,
			Title:          task.Title,
			Purpose:        task.Purpose,
			Prompt:         task.Prompt,
			NegativePrompt: task.NegativePrompt,
			StylePreset:    task.StylePreset,
			AspectRatio:    task.AspectRatio,
			SelectionMode:  task.SelectionMode,
			RequestedCount: task.RequestedCount,
		},
		Candidates: candidates,
	}
	if config != nil {
		payload.JudgePrompt = strings.TrimSpace(config.JudgePrompt)
	}
	return payload, assetIndex, nil
}

func selectHTTPJudgeCandidate(task domain.Task, assetIndex map[string]domain.AssetWithVersion, response httpJudgeScoreResponse) (bestOfSelectionResult, error) {
	scoreByAsset := make(map[string]httpJudgeScoreItem, len(response.Scores))
	for _, item := range response.Scores {
		scoreByAsset[item.AssetID] = item
	}

	selectedAssetID := strings.TrimSpace(response.SelectedAssetID)
	if selectedAssetID == "" {
		var bestScore *httpJudgeScoreItem
		for _, item := range response.Scores {
			candidate := item
			if _, ok := assetIndex[candidate.AssetID]; !ok {
				continue
			}
			if bestScore == nil || candidate.Score > bestScore.Score || (candidate.Score == bestScore.Score && candidate.AssetID < bestScore.AssetID) {
				bestScore = &candidate
			}
		}
		if bestScore != nil {
			selectedAssetID = bestScore.AssetID
		}
	}
	if selectedAssetID == "" {
		return bestOfSelectionResult{}, fmt.Errorf("http judge scorer response did not include a selectable candidate")
	}
	selected, ok := assetIndex[selectedAssetID]
	if !ok {
		return bestOfSelectionResult{}, fmt.Errorf("http judge scorer selected unknown asset %q", selectedAssetID)
	}

	scoreItem, ok := scoreByAsset[selectedAssetID]
	reasons := []string{"judge_selected_asset"}
	scoreValue := 0.0
	if ok {
		reasons = scoreItem.Reasons
		scoreValue = scoreItem.Score
	}
	actualRatio := 1.0
	if selected.Version.Width > 0 && selected.Version.Height > 0 {
		actualRatio = float64(selected.Version.Width) / float64(selected.Version.Height)
	}
	return bestOfSelectionResult{
		Best: selected,
		Score: bestOfScore{
			AssetID:       selected.ID,
			VersionID:     selected.Version.ID,
			Score:         scoreValue,
			Width:         selected.Version.Width,
			Height:        selected.Version.Height,
			TargetRatio:   task.AspectRatio,
			ActualRatio:   math.Round(actualRatio*1000) / 1000,
			Reasons:       reasons,
			Strategy:      domain.BestOfStrategyHTTPJudge,
			SelectionMode: task.SelectionMode,
		},
		OK: true,
	}, nil
}

func judgeImageDataURL(item domain.AssetWithVersion) (string, string, error) {
	path := strings.TrimSpace(item.Version.ThumbnailPath)
	mimeType := ""
	if path != "" {
		mimeType = storage.MimeTypeForPath(path)
	} else {
		path = strings.TrimSpace(item.Version.FilePath)
		mimeType = strings.TrimSpace(item.Version.MimeType)
		if mimeType == "" {
			mimeType = storage.MimeTypeForPath(path)
		}
	}
	if path == "" {
		return "", "", fmt.Errorf("asset has no file path for judge input")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxBestOfJudgeFileBytes+1))
	if err != nil {
		return "", "", err
	}
	if int64(len(data)) > maxBestOfJudgeFileBytes {
		return "", "", fmt.Errorf("judge input file exceeds %d bytes", maxBestOfJudgeFileBytes)
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data), mimeType, nil
}

func httpJudgeErrorMessage(respBytes []byte, fallback string) string {
	var body map[string]any
	if json.Unmarshal(respBytes, &body) == nil {
		if message, ok := body["error"].(string); ok && strings.TrimSpace(message) != "" {
			return strings.TrimSpace(message)
		}
		if message, ok := body["message"].(string); ok && strings.TrimSpace(message) != "" {
			return strings.TrimSpace(message)
		}
	}
	if trimmed := strings.TrimSpace(string(respBytes)); trimmed != "" {
		return trimmed
	}
	return fallback
}
