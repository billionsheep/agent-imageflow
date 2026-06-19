package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestScoreBestOfCandidatePrefersAspectAndArea(t *testing.T) {
	task := domain.Task{
		AspectRatio:   "16:9",
		SelectionMode: domain.SelectionBestOf,
	}
	smallWrongRatio := assetWithVersionForBestOf("asset_small", 800, 800, "sha256:ffffffff00000000")
	largeRightRatio := assetWithVersionForBestOf("asset_large", 1600, 900, "sha256:0000000000000000")

	best, score, ok := scoreBestOfCandidate(task, []domain.AssetWithVersion{smallWrongRatio, largeRightRatio})
	if !ok {
		t.Fatal("expected a best candidate")
	}
	if best.ID != "asset_large" {
		t.Fatalf("expected asset_large, got %s with score %#v", best.ID, score)
	}
	if score.Strategy != domain.BestOfStrategyLocalMetadata || score.SelectionMode != domain.SelectionBestOf {
		t.Fatalf("unexpected score metadata: %#v", score)
	}
}

func TestSelectionModeHelpers(t *testing.T) {
	if !domain.ShouldAutoSelect(domain.SelectionAuto) || !domain.ShouldAutoSelect(domain.SelectionBestOf) {
		t.Fatal("auto and best_of should trigger auto selection")
	}
	if domain.ShouldAutoSelect(domain.SelectionManualOptional) || domain.ShouldAutoSelect("unknown") {
		t.Fatal("manual_optional and unknown modes should not trigger auto selection")
	}
}

func TestHTTPJudgeBestOfScorerSelectsConfiguredCandidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request httpJudgeScoreRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Strategy != domain.BestOfStrategyHTTPJudge || len(request.Candidates) != 2 {
			t.Fatalf("unexpected scorer request: %#v", request)
		}
		_ = json.NewEncoder(w).Encode(httpJudgeScoreResponse{
			SelectedAssetID: "asset_b",
			Scores: []httpJudgeScoreItem{
				{AssetID: "asset_a", Score: 0.32, Reasons: []string{"too plain"}},
				{AssetID: "asset_b", Score: 0.91, Reasons: []string{"better cover"}},
			},
		})
	}))
	defer server.Close()

	assetA := assetWithVersionForBestOf("asset_a", 1200, 1600, "sha256:aaaaaaaa00000000")
	assetB := assetWithVersionForBestOf("asset_b", 1200, 1600, "sha256:bbbbbbbb00000000")
	attachBestOfFiles(t, &assetA)
	attachBestOfFiles(t, &assetB)

	scorer := newHTTPJudgeBestOfScorer(server.URL, "test-key", 5)
	result, err := scorer.Score(context.Background(), domain.Task{
		ID:             "task_http_best_of",
		AspectRatio:    "3:4",
		SelectionMode:  domain.SelectionAuto,
		Prompt:         "选择更适合作为封面图的一张",
		RequestedCount: 2,
	}, []domain.AssetWithVersion{assetA, assetB}, &domain.BestOfConfig{
		Strategy:    domain.BestOfStrategyHTTPJudge,
		JudgePrompt: "选出更适合作为封面图的一张",
	})
	if err != nil {
		t.Fatalf("scorer returned error: %v", err)
	}
	if !result.OK || result.Best.ID != "asset_b" {
		t.Fatalf("unexpected scorer result: %#v", result)
	}
	if result.Score.Strategy != domain.BestOfStrategyHTTPJudge || result.Score.Score != 0.91 {
		t.Fatalf("unexpected score metadata: %#v", result.Score)
	}
}

func TestSelectBestOfCandidateFallsBackToLocalMetadataWhenJudgeFails(t *testing.T) {
	service := &Service{
		bestOfScorers: map[string]bestOfScorer{
			domain.BestOfStrategyLocalMetadata: localMetadataBestOfScorer{},
			domain.BestOfStrategyHTTPJudge:     failingBestOfScorer{err: context.DeadlineExceeded},
		},
	}
	task := domain.Task{
		AspectRatio:   "16:9",
		SelectionMode: domain.SelectionBestOf,
		StructuredInputJSON: []byte(`{
			"best_of_config": {
				"strategy": "http_judge_v1",
				"judge_prompt": "pick the strongest cover"
			}
		}`),
	}
	smallWrongRatio := assetWithVersionForBestOf("asset_small", 800, 800, "sha256:ffffffff00000000")
	largeRightRatio := assetWithVersionForBestOf("asset_large", 1600, 900, "sha256:0000000000000000")

	decision, err := service.selectBestOfCandidate(context.Background(), task, []domain.AssetWithVersion{smallWrongRatio, largeRightRatio})
	if err != nil {
		t.Fatalf("selectBestOfCandidate returned error: %v", err)
	}
	if !decision.OK || decision.Best.ID != "asset_large" {
		t.Fatalf("unexpected fallback decision: %#v", decision)
	}
	if decision.RequestedStrategy != domain.BestOfStrategyHTTPJudge || decision.AppliedStrategy != domain.BestOfStrategyLocalMetadata {
		t.Fatalf("unexpected strategy transition: %#v", decision)
	}
	if decision.FallbackReason == "" {
		t.Fatalf("expected fallback reason to be recorded: %#v", decision)
	}
}

func TestOtherBestOfCandidateIDsSkipsSelectedAsset(t *testing.T) {
	assets := []domain.AssetWithVersion{
		assetWithVersionForBestOf("asset_a", 1200, 1600, "sha256:aaaaaaaa00000000"),
		assetWithVersionForBestOf("asset_b", 1200, 1600, "sha256:bbbbbbbb00000000"),
		assetWithVersionForBestOf("asset_c", 1200, 1600, "sha256:cccccccc00000000"),
	}
	got := otherBestOfCandidateIDs(assets, "asset_b")
	want := []string{"asset_a", "asset_c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected rejected candidate ids: got=%v want=%v", got, want)
	}
}

func TestSelectBestOfCandidateCarriesAutoRejectFlag(t *testing.T) {
	service := &Service{
		bestOfScorers: map[string]bestOfScorer{
			domain.BestOfStrategyLocalMetadata: localMetadataBestOfScorer{},
		},
	}
	task := domain.Task{
		AspectRatio:   "16:9",
		SelectionMode: domain.SelectionAuto,
		StructuredInputJSON: []byte(`{
			"best_of_config": {
				"auto_reject_non_selected": true
			}
		}`),
	}
	smallWrongRatio := assetWithVersionForBestOf("asset_small", 800, 800, "sha256:ffffffff00000000")
	largeRightRatio := assetWithVersionForBestOf("asset_large", 1600, 900, "sha256:0000000000000000")

	decision, err := service.selectBestOfCandidate(context.Background(), task, []domain.AssetWithVersion{smallWrongRatio, largeRightRatio})
	if err != nil {
		t.Fatalf("selectBestOfCandidate returned error: %v", err)
	}
	if !decision.OK || !decision.AutoRejectNonSelected {
		t.Fatalf("unexpected auto reject decision: %#v", decision)
	}
}

func assetWithVersionForBestOf(id string, width, height int, hash string) domain.AssetWithVersion {
	return domain.AssetWithVersion{
		Asset: domain.Asset{
			ID:     id,
			Status: domain.AssetDraft,
		},
		Version: domain.AssetVersion{
			ID:     "ver_" + id,
			Status: domain.VersionReady,
			Width:  width,
			Height: height,
			Hash:   hash,
		},
	}
}

func attachBestOfFiles(t *testing.T, item *domain.AssetWithVersion) {
	t.Helper()
	dir := t.TempDir()
	originalPath := dir + "/original.png"
	thumbnailPath := dir + "/thumbnail.webp"
	if err := os.WriteFile(originalPath, appTestPNG(t), 0o644); err != nil {
		t.Fatalf("write original file: %v", err)
	}
	if err := os.WriteFile(thumbnailPath, appTestPNG(t), 0o644); err != nil {
		t.Fatalf("write thumbnail file: %v", err)
	}
	item.Version.FilePath = originalPath
	item.Version.ThumbnailPath = thumbnailPath
	item.Version.MimeType = "image/png"
}

type failingBestOfScorer struct {
	err error
}

func (s failingBestOfScorer) Score(context.Context, domain.Task, []domain.AssetWithVersion, *domain.BestOfConfig) (bestOfSelectionResult, error) {
	return bestOfSelectionResult{}, s.err
}
