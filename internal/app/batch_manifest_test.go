package app

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestBuildBatchManifestFiltersAssetsAndCountsOutputShape(t *testing.T) {
	createdAt := time.Date(2026, 6, 22, 1, 2, 3, 0, time.UTC)
	summary := domain.BatchStorySummaryResponse{
		GeneratedAt: createdAt,
		ProjectID:   "prj_demo",
		CampaignID:  "cmp_demo",
		SessionID:   "session_1",
		BatchID:     "batch_1",
		Source:      "codex",
		StoryID:     "story_1",
		Stories: []domain.BatchStorySummaryStory{{
			StoryID:            "story_1",
			SceneCount:         1,
			SelectedSceneCount: 1,
			Scenes:             []string{"scene_001"},
		}},
		Scenes: []domain.BatchStorySummaryScene{{
			StoryID:                "story_1",
			SceneID:                "scene_001",
			Status:                 "completed",
			LatestTaskID:           "task_latest",
			PrimarySelectedAssetID: "asset_selected",
			TargetPath:             "stories/story-1/scene-001.png",
			RegenerationCount:      1,
			Continuity: domain.BatchStoryContinuitySummary{
				StoryRevision:                  "rev_001",
				StoryPlanHash:                  "sha256:story-plan",
				GenerationMode:                 "sequential_previous_panel",
				PanelIndex:                     1,
				NarrativeRole:                  "setup",
				Dialogue:                       "才没有等你",
				ProviderReferenceParticipation: "resolved_input_files",
				ResolvedReferenceAssets: []domain.BatchStoryResolvedReferenceAsset{
					{Role: "character_reference", AssetID: "asset_ref"},
				},
			},
			VisualContext: domain.BatchStoryVisualContext{
				CharacterIDs:      []string{"dog_mochi"},
				ReferenceAssetIDs: []string{"asset_ref"},
				PromptRecipeID:    "pet_story_cover",
			},
			Tasks: []domain.BatchStorySummaryTask{{
				TaskID:       "task_latest",
				Status:       domain.TaskCompleted,
				AssetCount:   3,
				AttemptCount: 2,
				CreatedAt:    createdAt,
				UpdatedAt:    createdAt.Add(time.Minute),
			}},
			Assets: []domain.BatchStorySummaryAsset{
				{AssetID: "asset_generated", TaskID: "task_latest", Status: "generated", DownloadURL: "/api/assets/asset_generated/original", ThumbnailURL: "/api/assets/asset_generated/thumbnail", MetadataURL: "/api/assets/asset_generated/metadata", TargetPath: "stories/story-1/scene-001.png", CreatedAt: createdAt},
				{AssetID: "asset_selected", TaskID: "task_latest", Status: "selected", DownloadURL: "/api/assets/asset_selected/original", ThumbnailURL: "/api/assets/asset_selected/thumbnail", MetadataURL: "/api/assets/asset_selected/metadata", TargetPath: "stories/story-1/scene-001.png", CreatedAt: createdAt.Add(time.Second), CaptionLineage: &domain.CaptionLineageSummary{
					DerivedFromAssetID: "asset_original",
					DerivationType:     "caption_edit",
					CaptionText:        "今天也要可爱",
					CaptionStyle:       "rounded speech bubble, handwritten",
					SourceTaskID:       "task_original",
					SourceSceneID:      "scene_001",
				}},
				{AssetID: "asset_rejected", TaskID: "task_latest", Status: "rejected", DownloadURL: "/api/assets/asset_rejected/original", ThumbnailURL: "/api/assets/asset_rejected/thumbnail", MetadataURL: "/api/assets/asset_rejected/metadata", TargetPath: "stories/story-1/scene-001.png", CreatedAt: createdAt.Add(2 * time.Second)},
			},
		}},
	}

	selectedManifest := buildBatchManifest(summary, domain.BatchManifestQuery{SelectedOnly: true})
	if len(selectedManifest.Assets) != 1 || selectedManifest.Assets[0].AssetID != "asset_selected" {
		t.Fatalf("selected_only manifest assets = %#v", selectedManifest.Assets)
	}
	if selectedManifest.Counts.AssetCount != 1 || selectedManifest.Counts.SelectedAssetCount != 1 || selectedManifest.Counts.GeneratedAssetCount != 0 || selectedManifest.Counts.RejectedAssetCount != 0 {
		t.Fatalf("selected_only counts should reflect output assets, got %#v", selectedManifest.Counts)
	}
	if len(selectedManifest.Tasks) != 1 || selectedManifest.Tasks[0].StoryID != "story_1" || selectedManifest.Tasks[0].SceneID != "scene_001" {
		t.Fatalf("manifest tasks should keep scene context, got %#v", selectedManifest.Tasks)
	}
	if len(selectedManifest.Scenes) != 1 || len(selectedManifest.Scenes[0].AssetIDs) != 1 || selectedManifest.Scenes[0].AssetIDs[0] != "asset_selected" {
		t.Fatalf("manifest scenes should list filtered asset ids, got %#v", selectedManifest.Scenes)
	}
	if len(selectedManifest.Scenes[0].TaskIDs) != 1 || selectedManifest.Scenes[0].TaskIDs[0] != "task_latest" {
		t.Fatalf("manifest scenes should keep task ids, got %#v", selectedManifest.Scenes[0].TaskIDs)
	}
	if selectedManifest.Assets[0].VisualContext.PromptRecipeID != "pet_story_cover" {
		t.Fatalf("asset visual context was not carried: %#v", selectedManifest.Assets[0].VisualContext)
	}
	if selectedManifest.Assets[0].Continuity.PanelIndex != 1 || selectedManifest.Assets[0].Continuity.StoryRevision != "rev_001" {
		t.Fatalf("asset continuity summary was not carried: %#v", selectedManifest.Assets[0].Continuity)
	}
	if len(selectedManifest.Scenes[0].Continuity.ResolvedReferenceAssets) != 1 {
		t.Fatalf("scene continuity resolved references missing: %#v", selectedManifest.Scenes[0].Continuity)
	}
	lineage := selectedManifest.Assets[0].CaptionLineage
	if lineage == nil ||
		lineage.DerivedFromAssetID != "asset_original" ||
		lineage.DerivationType != "caption_edit" ||
		lineage.CaptionText != "今天也要可爱" ||
		lineage.CaptionStyle != "rounded speech bubble, handwritten" ||
		lineage.SourceTaskID != "task_original" ||
		lineage.SourceSceneID != "scene_001" {
		t.Fatalf("asset caption lineage was not carried: %#v", selectedManifest.Assets[0])
	}

	allWithoutRejected := buildBatchManifest(summary, domain.BatchManifestQuery{SelectedOnly: false, IncludeRejected: false})
	if len(allWithoutRejected.Assets) != 2 || allWithoutRejected.Counts.GeneratedAssetCount != 1 || allWithoutRejected.Counts.SelectedAssetCount != 1 || allWithoutRejected.Counts.RejectedAssetCount != 0 {
		t.Fatalf("all without rejected counts/assets = %#v %#v", allWithoutRejected.Counts, allWithoutRejected.Assets)
	}

	allWithRejected := buildBatchManifest(summary, domain.BatchManifestQuery{SelectedOnly: false, IncludeRejected: true})
	if len(allWithRejected.Assets) != 3 || allWithRejected.Counts.RejectedAssetCount != 1 {
		t.Fatalf("all with rejected counts/assets = %#v %#v", allWithRejected.Counts, allWithRejected.Assets)
	}
}

func TestBuildBatchManifestJSONDoesNotExposeLocalPath(t *testing.T) {
	manifest := buildBatchManifest(domain.BatchStorySummaryResponse{
		ProjectID:  "prj_demo",
		CampaignID: "cmp_demo",
		Scenes: []domain.BatchStorySummaryScene{{
			StoryID: "story_1",
			SceneID: "scene_001",
			Assets: []domain.BatchStorySummaryAsset{{
				AssetID:     "asset_selected",
				TaskID:      "task_1",
				Status:      "selected",
				DownloadURL: "/api/assets/asset_selected/original",
				TargetPath:  "stories/story-1/scene-001.png",
			}},
		}},
	}, domain.BatchManifestQuery{SelectedOnly: true})

	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if strings.Contains(string(body), "local_path") || strings.Contains(string(body), "/tmp/") {
		t.Fatalf("manifest JSON exposed local path shaped data: %s", string(body))
	}
}
