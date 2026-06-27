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
			PrimarySelectedAssetID: "asset_caption_selected",
			TargetPath:             "stories/story-1/scene-001.png",
			RegenerationCount:      1,
			Continuity: domain.BatchStoryContinuitySummary{
				StoryRevision:                  "rev_001",
				StoryPlanHash:                  "sha256:story-plan",
				GenerationMode:                 "sequential_previous_panel",
				PanelIndex:                     1,
				NarrativeRole:                  "setup",
				Dialogue:                       "才没有等你",
				EmotionBefore:                  "嘴硬但期待",
				EmotionAfter:                   "偷偷开心",
				PoseChange:                     "从抱书防御变成稍微转头看门口",
				RelationshipShift:              "从独自等待变成准备迎接鸡毛",
				ProviderReferenceParticipation: "resolved_input_files",
				MustChange:                     []string{"眼神从书上转向门口"},
				MustNotKeep:                    []string{"不能把门口空间完全挡住"},
				StateTransitionNotes:           "第一格先建立等待感，不要过早同框。",
				ResolvedReferenceAssets: []domain.BatchStoryResolvedReferenceAsset{
					{Role: "character_reference", AssetID: "asset_ref"},
				},
			},
			VisualContext: domain.BatchStoryVisualContext{
				CharacterIDs:      []string{"dog_mochi"},
				ReferenceAssetIDs: []string{"asset_ref"},
				PromptRecipeID:    "pet_story_cover",
				ReferenceDiagnostics: &domain.ProjectVisualContextReferenceDiagnostics{
					PrimaryReadiness:                   "image_backed",
					Labels:                             []string{"image_backed", "missing_environment_reference"},
					Summary:                            "image-backed, missing environment reference",
					ActiveCharacterCount:               1,
					CharacterWithImageCount:            1,
					MissingCharacterImageCount:         0,
					ActiveReferenceCount:               1,
					EnvironmentReferenceCount:          0,
					ImageReferenceCount:                1,
					NegativePromptCoversSpeciesDrift:   true,
					IdentitySignalPresent:              true,
					ProviderReferenceParticipationRisk: "likely_resolved_input_files",
				},
			},
			Tasks: []domain.BatchStorySummaryTask{{
				TaskID:         "task_latest",
				Status:         domain.TaskCompleted,
				RequestedCount: 4,
				AssetCount:     4,
				AttemptCount:   2,
				CreatedAt:      createdAt,
				UpdatedAt:      createdAt.Add(time.Minute),
			}},
			Assets: []domain.BatchStorySummaryAsset{
				{AssetID: "asset_generated", TaskID: "task_latest", Status: "generated", DownloadURL: "/api/assets/asset_generated/original", ThumbnailURL: "/api/assets/asset_generated/thumbnail", MetadataURL: "/api/assets/asset_generated/metadata", TargetPath: "stories/story-1/scene-001.png", CreatedAt: createdAt},
				{AssetID: "asset_base_selected", TaskID: "task_latest", Status: "selected", DownloadURL: "/api/assets/asset_base_selected/original", ThumbnailURL: "/api/assets/asset_base_selected/thumbnail", MetadataURL: "/api/assets/asset_base_selected/metadata", TargetPath: "stories/story-1/scene-001.png", CreatedAt: createdAt.Add(time.Second)},
				{AssetID: "asset_caption_selected", TaskID: "task_latest", Status: "selected", DownloadURL: "/api/assets/asset_caption_selected/original", ThumbnailURL: "/api/assets/asset_caption_selected/thumbnail", MetadataURL: "/api/assets/asset_caption_selected/metadata", TargetPath: "stories/story-1/scene-001-caption.png", CreatedAt: createdAt.Add(2 * time.Second), CaptionLineage: &domain.CaptionLineageSummary{
					DerivedFromAssetID: "asset_base_selected",
					DerivationType:     "caption_edit",
					CaptionText:        "今天也要可爱",
					CaptionStyle:       "rounded speech bubble, handwritten",
					SourceTaskID:       "task_original",
					SourceSceneID:      "scene_001",
					SpeakerCharacterID: "dog_jimao",
					BubbleAnchor:       "top_right",
					TailDirection:      "toward_left",
					CaptionIntent:      "comfort",
					AutoSelectDerivative: func() *bool {
						value := true
						return &value
					}(),
					AvoidCoveringSubjects: func() *bool {
						value := true
						return &value
					}(),
				}},
				{AssetID: "asset_rejected", TaskID: "task_latest", Status: "rejected", DownloadURL: "/api/assets/asset_rejected/original", ThumbnailURL: "/api/assets/asset_rejected/thumbnail", MetadataURL: "/api/assets/asset_rejected/metadata", TargetPath: "stories/story-1/scene-001.png", CreatedAt: createdAt.Add(3 * time.Second)},
			},
		}},
	}

	selectedManifest := buildBatchManifest(summary, domain.BatchManifestQuery{SelectedOnly: true})
	if len(selectedManifest.Assets) != 1 || selectedManifest.Assets[0].AssetID != "asset_caption_selected" {
		t.Fatalf("selected_only manifest assets = %#v", selectedManifest.Assets)
	}
	if selectedManifest.Assets[0].DeliveryRole != domain.DeliveryRoleFinalDelivery {
		t.Fatalf("selected_only asset should be final_delivery, got %#v", selectedManifest.Assets[0])
	}
	if selectedManifest.Counts.AssetCount != 1 || selectedManifest.Counts.SelectedAssetCount != 1 || selectedManifest.Counts.GeneratedAssetCount != 0 || selectedManifest.Counts.RejectedAssetCount != 0 {
		t.Fatalf("selected_only counts should reflect output assets, got %#v", selectedManifest.Counts)
	}
	if len(selectedManifest.Tasks) != 1 || selectedManifest.Tasks[0].StoryID != "story_1" || selectedManifest.Tasks[0].SceneID != "scene_001" {
		t.Fatalf("manifest tasks should keep scene context, got %#v", selectedManifest.Tasks)
	}
	if selectedManifest.Tasks[0].RequestedCount != 4 || selectedManifest.Tasks[0].DeliveredCount != 4 {
		t.Fatalf("manifest task runtime semantics missing counts: %#v", selectedManifest.Tasks[0])
	}
	if len(selectedManifest.Scenes) != 1 || len(selectedManifest.Scenes[0].AssetIDs) != 1 || selectedManifest.Scenes[0].AssetIDs[0] != "asset_caption_selected" {
		t.Fatalf("manifest scenes should list filtered asset ids, got %#v", selectedManifest.Scenes)
	}
	if len(selectedManifest.Scenes[0].TaskIDs) != 1 || selectedManifest.Scenes[0].TaskIDs[0] != "task_latest" {
		t.Fatalf("manifest scenes should keep task ids, got %#v", selectedManifest.Scenes[0].TaskIDs)
	}
	if selectedManifest.Assets[0].VisualContext.PromptRecipeID != "pet_story_cover" {
		t.Fatalf("asset visual context was not carried: %#v", selectedManifest.Assets[0].VisualContext)
	}
	if selectedManifest.Assets[0].VisualContext.ReferenceDiagnostics == nil ||
		selectedManifest.Assets[0].VisualContext.ReferenceDiagnostics.PrimaryReadiness != "image_backed" {
		t.Fatalf("asset visual context reference diagnostics were not carried: %#v", selectedManifest.Assets[0].VisualContext)
	}
	if selectedManifest.Assets[0].Continuity.PanelIndex != 1 || selectedManifest.Assets[0].Continuity.StoryRevision != "rev_001" {
		t.Fatalf("asset continuity summary was not carried: %#v", selectedManifest.Assets[0].Continuity)
	}
	if selectedManifest.Assets[0].Continuity.EmotionBefore != "嘴硬但期待" ||
		selectedManifest.Assets[0].Continuity.EmotionAfter != "偷偷开心" ||
		selectedManifest.Assets[0].Continuity.PoseChange != "从抱书防御变成稍微转头看门口" ||
		selectedManifest.Assets[0].Continuity.RelationshipShift != "从独自等待变成准备迎接鸡毛" ||
		len(selectedManifest.Assets[0].Continuity.MustChange) != 1 ||
		len(selectedManifest.Assets[0].Continuity.MustNotKeep) != 1 ||
		selectedManifest.Assets[0].Continuity.StateTransitionNotes != "第一格先建立等待感，不要过早同框。" {
		t.Fatalf("asset state transition summary was not carried: %#v", selectedManifest.Assets[0].Continuity)
	}
	if len(selectedManifest.Scenes[0].Continuity.ResolvedReferenceAssets) != 1 {
		t.Fatalf("scene continuity resolved references missing: %#v", selectedManifest.Scenes[0].Continuity)
	}
	lineage := selectedManifest.Assets[0].CaptionLineage
	if lineage == nil ||
		lineage.DerivedFromAssetID != "asset_base_selected" ||
		lineage.DerivationType != "caption_edit" ||
		lineage.CaptionText != "今天也要可爱" ||
		lineage.CaptionStyle != "rounded speech bubble, handwritten" ||
		lineage.SourceTaskID != "task_original" ||
		lineage.SourceSceneID != "scene_001" ||
		lineage.SpeakerCharacterID != "dog_jimao" ||
		lineage.BubbleAnchor != "top_right" ||
		lineage.TailDirection != "toward_left" ||
		lineage.CaptionIntent != "comfort" ||
		lineage.AutoSelectDerivative == nil ||
		!*lineage.AutoSelectDerivative ||
		lineage.AvoidCoveringSubjects == nil ||
		!*lineage.AvoidCoveringSubjects {
		t.Fatalf("asset caption lineage was not carried: %#v", selectedManifest.Assets[0])
	}

	allWithoutRejected := buildBatchManifest(summary, domain.BatchManifestQuery{SelectedOnly: false, IncludeRejected: false})
	if len(allWithoutRejected.Assets) != 3 || allWithoutRejected.Counts.GeneratedAssetCount != 1 || allWithoutRejected.Counts.SelectedAssetCount != 2 || allWithoutRejected.Counts.RejectedAssetCount != 0 {
		t.Fatalf("all without rejected counts/assets = %#v %#v", allWithoutRejected.Counts, allWithoutRejected.Assets)
	}
	if allWithoutRejected.Assets[1].AssetID != "asset_base_selected" || allWithoutRejected.Assets[1].DeliveryRole != domain.DeliveryRoleBaseOriginal {
		t.Fatalf("base selected asset should stay visible as base_original in all-assets manifest: %#v", allWithoutRejected.Assets)
	}
	if allWithoutRejected.Assets[2].AssetID != "asset_caption_selected" || allWithoutRejected.Assets[2].DeliveryRole != domain.DeliveryRoleFinalDelivery {
		t.Fatalf("caption selected asset should become final_delivery in all-assets manifest: %#v", allWithoutRejected.Assets)
	}

	allWithRejected := buildBatchManifest(summary, domain.BatchManifestQuery{SelectedOnly: false, IncludeRejected: true})
	if len(allWithRejected.Assets) != 4 || allWithRejected.Counts.RejectedAssetCount != 1 {
		t.Fatalf("all with rejected counts/assets = %#v %#v", allWithRejected.Counts, allWithRejected.Assets)
	}
}

func TestBuildBatchManifestSelectedOnlyKeepsBaseDeliveryWhenCaptionDerivativeNeedsManualReview(t *testing.T) {
	createdAt := time.Date(2026, 6, 22, 1, 2, 3, 0, time.UTC)
	summary := domain.BatchStorySummaryResponse{
		ProjectID:  "prj_demo",
		CampaignID: "cmp_demo",
		Scenes: []domain.BatchStorySummaryScene{{
			StoryID:                "story_1",
			SceneID:                "scene_001",
			Status:                 "completed",
			PrimarySelectedAssetID: "asset_base_selected",
			Tasks: []domain.BatchStorySummaryTask{{
				TaskID:       "task_latest",
				Status:       domain.TaskCompleted,
				AssetCount:   2,
				AttemptCount: 1,
				CreatedAt:    createdAt,
				UpdatedAt:    createdAt,
			}},
			Assets: []domain.BatchStorySummaryAsset{
				{AssetID: "asset_base_selected", TaskID: "task_latest", Status: "selected", DownloadURL: "/api/assets/asset_base_selected/original", ThumbnailURL: "/api/assets/asset_base_selected/thumbnail", MetadataURL: "/api/assets/asset_base_selected/metadata", TargetPath: "stories/story-1/scene-001.png", CreatedAt: createdAt},
				{AssetID: "asset_caption_generated", TaskID: "task_caption", Status: "generated", DownloadURL: "/api/assets/asset_caption_generated/original", ThumbnailURL: "/api/assets/asset_caption_generated/thumbnail", MetadataURL: "/api/assets/asset_caption_generated/metadata", TargetPath: "stories/story-1/scene-001-caption.png", CreatedAt: createdAt.Add(time.Second), CaptionLineage: &domain.CaptionLineageSummary{
					DerivedFromAssetID: "asset_base_selected",
					DerivationType:     "caption_edit",
					CaptionText:        "先人工看看",
					AutoSelectDerivative: func() *bool {
						value := false
						return &value
					}(),
				}},
			},
		}},
	}

	selectedManifest := buildBatchManifest(summary, domain.BatchManifestQuery{SelectedOnly: true})
	if len(selectedManifest.Assets) != 1 || selectedManifest.Assets[0].AssetID != "asset_base_selected" {
		t.Fatalf("selected_only manifest should keep base selected asset when caption derivative still needs manual review: %#v", selectedManifest.Assets)
	}
	if selectedManifest.Assets[0].DeliveryRole != domain.DeliveryRoleFinalDelivery {
		t.Fatalf("base selected asset should remain final_delivery until derivative is selected: %#v", selectedManifest.Assets[0])
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

func TestBuildBatchManifestFinalDeliveryViewIncludesFlattenedFinalAssets(t *testing.T) {
	createdAt := time.Date(2026, 6, 22, 1, 2, 3, 0, time.UTC)
	summary := domain.BatchStorySummaryResponse{
		ProjectID:  "prj_demo",
		CampaignID: "cmp_demo",
		SessionID:  "session_1",
		BatchID:    "batch_1",
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
			TargetPath:             "stories/story-1/scene-001.png",
			LatestTaskID:           "task_latest",
			PrimarySelectedAssetID: "asset_caption_selected",
			Continuity: domain.BatchStoryContinuitySummary{
				PanelIndex: 1,
			},
			VisualContext: domain.BatchStoryVisualContext{
				CharacterIDs: []string{"dog_mochi"},
			},
			Tasks: []domain.BatchStorySummaryTask{{
				TaskID:       "task_latest",
				Status:       domain.TaskCompleted,
				AssetCount:   2,
				AttemptCount: 1,
				CreatedAt:    createdAt,
				UpdatedAt:    createdAt,
			}},
			Assets: []domain.BatchStorySummaryAsset{
				{AssetID: "asset_base_selected", TaskID: "task_latest", Status: "selected", DownloadURL: "/api/assets/asset_base_selected/original", ThumbnailURL: "/api/assets/asset_base_selected/thumbnail", MetadataURL: "/api/assets/asset_base_selected/metadata", TargetPath: "stories/story-1/scene-001.png", CreatedAt: createdAt},
				{AssetID: "asset_caption_selected", TaskID: "task_latest", Status: "selected", DownloadURL: "/api/assets/asset_caption_selected/original", ThumbnailURL: "/api/assets/asset_caption_selected/thumbnail", MetadataURL: "/api/assets/asset_caption_selected/metadata", TargetPath: "stories/story-1/scene-001-caption.png", CreatedAt: createdAt.Add(time.Second), CaptionLineage: &domain.CaptionLineageSummary{
					DerivedFromAssetID: "asset_base_selected",
					DerivationType:     "caption_edit",
					CaptionText:        "今天也要可爱",
				}},
			},
		}},
	}

	manifest := buildBatchManifest(summary, domain.BatchManifestQuery{
		SelectedOnly: true,
		View:         domain.BatchManifestViewFinalDelivery,
	})

	if manifest.ManifestView != domain.BatchManifestViewFinalDelivery {
		t.Fatalf("manifest_view = %q, want %q", manifest.ManifestView, domain.BatchManifestViewFinalDelivery)
	}
	if manifest.FinalDelivery == nil {
		t.Fatal("expected final_delivery block")
	}
	if manifest.FinalDelivery.Counts.FinalAssetCount != 1 ||
		manifest.FinalDelivery.Counts.SceneWithFinalAssetCount != 1 ||
		manifest.FinalDelivery.Counts.SceneMissingFinalAssetCount != 0 {
		t.Fatalf("unexpected final_delivery counts: %#v", manifest.FinalDelivery.Counts)
	}
	if len(manifest.FinalDelivery.FinalAssets) != 1 {
		t.Fatalf("expected exactly one final asset, got %#v", manifest.FinalDelivery.FinalAssets)
	}
	finalAsset := manifest.FinalDelivery.FinalAssets[0]
	if finalAsset.AssetID != "asset_caption_selected" ||
		finalAsset.DeliveryRole != domain.DeliveryRoleFinalDelivery ||
		finalAsset.DerivedFromAssetID != "asset_base_selected" ||
		finalAsset.DerivationType != "caption_edit" ||
		finalAsset.TargetPath != "stories/story-1/scene-001-caption.png" {
		t.Fatalf("unexpected flattened final asset: %#v", finalAsset)
	}
	if len(manifest.FinalDelivery.Scenes) != 1 || len(manifest.FinalDelivery.Scenes[0].FinalAssets) != 1 {
		t.Fatalf("expected scene final assets to be grouped, got %#v", manifest.FinalDelivery.Scenes)
	}
	if len(manifest.FinalDelivery.Stories) != 1 ||
		len(manifest.FinalDelivery.Stories[0].Scenes) != 1 ||
		len(manifest.FinalDelivery.Stories[0].FinalAssets) != 1 {
		t.Fatalf("expected story final delivery summary, got %#v", manifest.FinalDelivery.Stories)
	}
}

func TestBuildBatchManifestFinalDeliveryViewKeepsBaseSelectedWhenDerivativeNeedsManualReview(t *testing.T) {
	createdAt := time.Date(2026, 6, 22, 1, 2, 3, 0, time.UTC)
	summary := domain.BatchStorySummaryResponse{
		ProjectID:  "prj_demo",
		CampaignID: "cmp_demo",
		Scenes: []domain.BatchStorySummaryScene{{
			StoryID:                "story_1",
			SceneID:                "scene_001",
			Status:                 "completed",
			PrimarySelectedAssetID: "asset_base_selected",
			Tasks: []domain.BatchStorySummaryTask{{
				TaskID:       "task_latest",
				Status:       domain.TaskCompleted,
				AssetCount:   2,
				AttemptCount: 1,
				CreatedAt:    createdAt,
				UpdatedAt:    createdAt,
			}},
			Assets: []domain.BatchStorySummaryAsset{
				{AssetID: "asset_base_selected", TaskID: "task_latest", Status: "selected", DownloadURL: "/api/assets/asset_base_selected/original", ThumbnailURL: "/api/assets/asset_base_selected/thumbnail", MetadataURL: "/api/assets/asset_base_selected/metadata", TargetPath: "stories/story-1/scene-001.png", CreatedAt: createdAt},
				{AssetID: "asset_caption_generated", TaskID: "task_caption", Status: "generated", DownloadURL: "/api/assets/asset_caption_generated/original", ThumbnailURL: "/api/assets/asset_caption_generated/thumbnail", MetadataURL: "/api/assets/asset_caption_generated/metadata", TargetPath: "stories/story-1/scene-001-caption.png", CreatedAt: createdAt.Add(time.Second), CaptionLineage: &domain.CaptionLineageSummary{
					DerivedFromAssetID: "asset_base_selected",
					DerivationType:     "caption_edit",
				}},
			},
		}},
	}

	manifest := buildBatchManifest(summary, domain.BatchManifestQuery{
		SelectedOnly: true,
		View:         domain.BatchManifestViewFinalDelivery,
	})

	if manifest.FinalDelivery == nil || len(manifest.FinalDelivery.FinalAssets) != 1 {
		t.Fatalf("expected one final asset, got %#v", manifest.FinalDelivery)
	}
	if manifest.FinalDelivery.FinalAssets[0].AssetID != "asset_base_selected" ||
		manifest.FinalDelivery.FinalAssets[0].DeliveryRole != domain.DeliveryRoleFinalDelivery {
		t.Fatalf("expected base selected asset to remain final delivery, got %#v", manifest.FinalDelivery.FinalAssets[0])
	}
}

func TestBuildBatchManifestFinalDeliveryViewTracksScenesMissingFinalAsset(t *testing.T) {
	manifest := buildBatchManifest(domain.BatchStorySummaryResponse{
		ProjectID:  "prj_demo",
		CampaignID: "cmp_demo",
		Scenes: []domain.BatchStorySummaryScene{{
			StoryID:    "story_1",
			SceneID:    "scene_001",
			Status:     "running",
			TargetPath: "stories/story-1/scene-001.png",
			Assets: []domain.BatchStorySummaryAsset{{
				AssetID:     "asset_generated",
				TaskID:      "task_1",
				Status:      "generated",
				DownloadURL: "/api/assets/asset_generated/original",
				ThumbnailURL: "/api/assets/asset_generated/thumbnail",
				MetadataURL: "/api/assets/asset_generated/metadata",
				TargetPath:  "stories/story-1/scene-001.png",
			}},
		}},
	}, domain.BatchManifestQuery{
		View: domain.BatchManifestViewFinalDelivery,
	})

	if manifest.FinalDelivery == nil {
		t.Fatal("expected final_delivery block")
	}
	if manifest.FinalDelivery.Counts.FinalAssetCount != 0 ||
		manifest.FinalDelivery.Counts.SceneWithFinalAssetCount != 0 ||
		manifest.FinalDelivery.Counts.SceneMissingFinalAssetCount != 1 {
		t.Fatalf("unexpected missing-final counts: %#v", manifest.FinalDelivery.Counts)
	}
	if len(manifest.FinalDelivery.Scenes) != 1 || len(manifest.FinalDelivery.Scenes[0].FinalAssets) != 0 {
		t.Fatalf("expected scene final_assets to be empty, got %#v", manifest.FinalDelivery.Scenes)
	}
}

func TestBuildBatchManifestFinalDeliveryJSONDoesNotExposeLocalPath(t *testing.T) {
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
	}, domain.BatchManifestQuery{
		SelectedOnly: true,
		View:         domain.BatchManifestViewFinalDelivery,
	})

	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if strings.Contains(string(body), "local_path") || strings.Contains(string(body), "/tmp/") {
		t.Fatalf("final delivery manifest JSON exposed local path shaped data: %s", string(body))
	}
}
