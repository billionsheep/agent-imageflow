package app

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

type bestOfScore struct {
	AssetID       string   `json:"asset_id"`
	VersionID     string   `json:"asset_version_id"`
	Score         float64  `json:"score"`
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	TargetRatio   string   `json:"target_ratio"`
	ActualRatio   float64  `json:"actual_ratio"`
	Reasons       []string `json:"reasons"`
	Strategy      string   `json:"strategy"`
	SelectionMode string   `json:"selection_mode"`
}

type bestOfSelectionResult struct {
	Best  domain.AssetWithVersion
	Score bestOfScore
	OK    bool
}

type bestOfDecision struct {
	RequestedStrategy     string
	AppliedStrategy       string
	Best                  domain.AssetWithVersion
	Score                 bestOfScore
	OK                    bool
	FallbackReason        string
	AutoRejectNonSelected bool
}

type bestOfScorer interface {
	Score(context.Context, domain.Task, []domain.AssetWithVersion, *domain.BestOfConfig) (bestOfSelectionResult, error)
}

type localMetadataBestOfScorer struct{}

func (localMetadataBestOfScorer) Score(_ context.Context, task domain.Task, assets []domain.AssetWithVersion, _ *domain.BestOfConfig) (bestOfSelectionResult, error) {
	best, score, ok := scoreBestOfCandidate(task, assets)
	return bestOfSelectionResult{
		Best:  best,
		Score: score,
		OK:    ok,
	}, nil
}

func (s *Service) autoSelectBestAsset(ctx context.Context, task domain.Task, assets []domain.AssetWithVersion) error {
	if !domain.ShouldAutoSelect(task.SelectionMode) {
		return nil
	}
	decision, err := s.selectBestOfCandidate(ctx, task, assets)
	if err != nil {
		return err
	}
	if !decision.OK {
		return nil
	}
	selectedNotePayload := map[string]any{
		"requested_strategy":       decision.RequestedStrategy,
		"applied_strategy":         decision.AppliedStrategy,
		"selection_mode":           task.SelectionMode,
		"candidate_count":          len(assets),
		"auto_reject_non_selected": decision.AutoRejectNonSelected,
		"selected":                 decision.Score,
	}
	if decision.FallbackReason != "" {
		selectedNotePayload["fallback"] = map[string]any{
			"from":   decision.RequestedStrategy,
			"to":     decision.AppliedStrategy,
			"reason": decision.FallbackReason,
		}
	}
	selectedNote, err := json.Marshal(selectedNotePayload)
	if err != nil {
		return err
	}
	if !decision.AutoRejectNonSelected {
		_, err = s.store.ReviewAsset(ctx, decision.Best.ID, "approve", "auto-best-of", string(selectedNote))
		return err
	}

	rejectedNotePayload := map[string]any{
		"requested_strategy":       decision.RequestedStrategy,
		"applied_strategy":         decision.AppliedStrategy,
		"selection_mode":           task.SelectionMode,
		"candidate_count":          len(assets),
		"auto_reject_non_selected": true,
		"selected":                 decision.Score,
	}
	if decision.FallbackReason != "" {
		rejectedNotePayload["fallback"] = map[string]any{
			"from":   decision.RequestedStrategy,
			"to":     decision.AppliedStrategy,
			"reason": decision.FallbackReason,
		}
	}
	rejectedNote, err := json.Marshal(rejectedNotePayload)
	if err != nil {
		return err
	}
	_, err = s.store.ApplyBestOfSelection(ctx, decision.Best.ID, otherBestOfCandidateIDs(assets, decision.Best.ID), "auto-best-of", string(selectedNote), string(rejectedNote))
	return err
}

func (s *Service) selectBestOfCandidate(ctx context.Context, task domain.Task, assets []domain.AssetWithVersion) (bestOfDecision, error) {
	config := bestOfConfigFromTask(task)
	requestedStrategy := requestedBestOfStrategy(config)
	decision := bestOfDecision{
		RequestedStrategy:     requestedStrategy,
		AutoRejectNonSelected: shouldAutoRejectNonSelected(config),
	}

	result, err := s.scoreBestOfWithStrategy(ctx, requestedStrategy, task, assets, config)
	if err == nil {
		decision.AppliedStrategy = result.Score.Strategy
		decision.Best = result.Best
		decision.Score = result.Score
		decision.OK = result.OK
		if result.OK {
			return decision, nil
		}
	}
	if requestedStrategy == domain.BestOfStrategyLocalMetadata {
		return decision, err
	}

	fallbackReason := "strategy returned no candidate"
	if err != nil {
		fallbackReason = err.Error()
	}
	fallbackConfig := &domain.BestOfConfig{Strategy: domain.BestOfStrategyLocalMetadata}
	fallback, fallbackErr := s.scoreBestOfWithStrategy(ctx, domain.BestOfStrategyLocalMetadata, task, assets, fallbackConfig)
	if fallbackErr != nil {
		return decision, fallbackErr
	}
	decision.AppliedStrategy = fallback.Score.Strategy
	decision.Best = fallback.Best
	decision.Score = fallback.Score
	decision.OK = fallback.OK
	decision.FallbackReason = fallbackReason
	return decision, nil
}

func (s *Service) scoreBestOfWithStrategy(ctx context.Context, strategy string, task domain.Task, assets []domain.AssetWithVersion, config *domain.BestOfConfig) (bestOfSelectionResult, error) {
	scorer, ok := s.bestOfScorers[strategy]
	if !ok {
		return bestOfSelectionResult{}, fmt.Errorf("best_of strategy %q is not enabled; configure it or use %q", strategy, domain.BestOfStrategyLocalMetadata)
	}
	return scorer.Score(ctx, task, assets, config)
}

func (s *Service) validateBestOfConfig(selectionMode string, config *domain.BestOfConfig) error {
	if !domain.ShouldAutoSelect(selectionMode) {
		return nil
	}
	strategy := requestedBestOfStrategy(config)
	if _, ok := s.bestOfScorers[strategy]; !ok {
		return fmt.Errorf("best_of strategy %q is not enabled; configure it or use %q", strategy, domain.BestOfStrategyLocalMetadata)
	}
	return nil
}

func scoreBestOfCandidate(task domain.Task, assets []domain.AssetWithVersion) (domain.AssetWithVersion, bestOfScore, bool) {
	targetRatio := aspectRatioValue(task.AspectRatio)
	var best domain.AssetWithVersion
	var bestScore bestOfScore
	found := false
	for _, candidate := range assets {
		score := scoreCandidate(task, candidate, targetRatio)
		if !found || score.Score > bestScore.Score || (score.Score == bestScore.Score && candidate.ID < best.ID) {
			best = candidate
			bestScore = score
			found = true
		}
	}
	return best, bestScore, found
}

func scoreCandidate(task domain.Task, item domain.AssetWithVersion, targetRatio float64) bestOfScore {
	score := 0.0
	reasons := []string{}
	if item.Version.Status == domain.VersionReady {
		score += 1000
		reasons = append(reasons, "version_ready")
	}

	area := item.Version.Width * item.Version.Height
	if area > 0 {
		areaScore := math.Min(float64(area)/1_000_000, 8) * 10
		score += areaScore
		reasons = append(reasons, "larger_image_area")
	}

	actualRatio := 1.0
	if item.Version.Width > 0 && item.Version.Height > 0 {
		actualRatio = float64(item.Version.Width) / float64(item.Version.Height)
	}
	if targetRatio > 0 && actualRatio > 0 {
		distance := math.Abs(actualRatio-targetRatio) / targetRatio
		ratioScore := math.Max(0, 100-(distance*100))
		score += ratioScore
		reasons = append(reasons, "aspect_ratio_match")
	}

	score += stableHashTiebreaker(item.Version.Hash)

	return bestOfScore{
		AssetID:       item.ID,
		VersionID:     item.Version.ID,
		Score:         math.Round(score*1000) / 1000,
		Width:         item.Version.Width,
		Height:        item.Version.Height,
		TargetRatio:   task.AspectRatio,
		ActualRatio:   math.Round(actualRatio*1000) / 1000,
		Reasons:       reasons,
		Strategy:      domain.BestOfStrategyLocalMetadata,
		SelectionMode: task.SelectionMode,
	}
}

func bestOfConfigFromTask(task domain.Task) *domain.BestOfConfig {
	if len(task.StructuredInputJSON) == 0 {
		return nil
	}
	var input struct {
		BestOfConfig *domain.BestOfConfig `json:"best_of_config"`
	}
	if err := json.Unmarshal(task.StructuredInputJSON, &input); err != nil {
		return nil
	}
	return cloneBestOfConfig(input.BestOfConfig)
}

func requestedBestOfStrategy(config *domain.BestOfConfig) string {
	if config != nil && strings.TrimSpace(config.Strategy) != "" {
		return config.Strategy
	}
	return domain.BestOfStrategyLocalMetadata
}

func shouldAutoRejectNonSelected(config *domain.BestOfConfig) bool {
	return config != nil && config.AutoRejectNonSelected
}

func otherBestOfCandidateIDs(assets []domain.AssetWithVersion, selectedAssetID string) []string {
	if len(assets) == 0 {
		return nil
	}
	ids := make([]string, 0, len(assets)-1)
	for _, asset := range assets {
		if asset.ID == selectedAssetID {
			continue
		}
		ids = append(ids, asset.ID)
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

func aspectRatioValue(value string) float64 {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 1
	}
	width, errW := strconv.ParseFloat(parts[0], 64)
	height, errH := strconv.ParseFloat(parts[1], 64)
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return 1
	}
	return width / height
}

func stableHashTiebreaker(hash string) float64 {
	hash = strings.TrimPrefix(strings.TrimSpace(hash), "sha256:")
	if len(hash) < 8 {
		return 0
	}
	raw, err := hex.DecodeString(hash[:8])
	if err != nil || len(raw) == 0 {
		return 0
	}
	value := 0
	for _, b := range raw {
		value = value<<8 + int(b)
	}
	return float64(value) / float64(1<<32) / 100
}
