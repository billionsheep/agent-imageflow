package app

import (
	"encoding/json"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestApplyQualityProfileRendersProjectTemplateAndSnapshot(t *testing.T) {
	req := domain.CreateTaskRequest{
		Title:                    "Day 1",
		Purpose:                  "cover",
		Prompt:                   "普通人如何用 AI 做第一张动漫头像",
		UseProjectQualityProfile: true,
		MetadataJSON:             []byte(`{"channel":"xiaohongshu"}`),
	}
	profile := domain.QualityProfile{
		PromptTemplate: "{{prompt}}，{{channel}} 风格，清爽留白",
		NegativePrompt: "low quality",
		StylePreset:    "anime-cover",
		ReferenceImages: []domain.ReferenceImage{
			{URL: " https://example.com/ref.png ", Role: " style "},
		},
		BestOfConfig:     &domain.BestOfConfig{Strategy: domain.BestOfStrategyHTTPJudge, JudgePrompt: "选择更适合做封面图的一张", AutoRejectNonSelected: true},
		GenerationConfig: []byte(`{"quality":"high"}`),
	}

	result, err := applyQualityProfile(req, profile)
	if err != nil {
		t.Fatalf("applyQualityProfile returned error: %v", err)
	}
	if result.Request.Prompt != "普通人如何用 AI 做第一张动漫头像，xiaohongshu 风格，清爽留白" {
		t.Fatalf("unexpected rendered prompt: %q", result.Request.Prompt)
	}
	if result.Request.NegativePrompt != "low quality" || result.Request.StylePreset != "anime-cover" {
		t.Fatalf("profile defaults were not applied: %#v", result.Request)
	}
	if len(result.Request.ReferenceImages) != 1 || result.Request.ReferenceImages[0].Role != "style" {
		t.Fatalf("reference images were not normalized: %#v", result.Request.ReferenceImages)
	}

	var metadata map[string]any
	if err := json.Unmarshal(result.Request.MetadataJSON, &metadata); err != nil {
		t.Fatalf("metadata JSON invalid: %v", err)
	}
	snapshot := metadata[qualityProfileSnapshotKey].(map[string]any)
	if snapshot["source"] != "project" || snapshot["style_preset"] != "anime-cover" {
		t.Fatalf("unexpected quality snapshot: %#v", snapshot)
	}
	bestOfConfig := snapshot["best_of_config"].(map[string]any)
	if bestOfConfig["strategy"] != domain.BestOfStrategyHTTPJudge {
		t.Fatalf("best_of_config was not written into quality snapshot: %#v", snapshot)
	}
	if bestOfConfig["auto_reject_non_selected"] != true {
		t.Fatalf("auto_reject_non_selected was not written into quality snapshot: %#v", snapshot)
	}
}

func TestApplyQualityProfileAllowsRequestOverride(t *testing.T) {
	req := domain.CreateTaskRequest{
		Prompt:                   "base",
		PromptTemplate:           "request {{prompt}}",
		StylePreset:              "request-style",
		UseProjectQualityProfile: true,
		GenerationConfig:         []byte(`{"quality":"low"}`),
	}
	profile := domain.QualityProfile{
		PromptTemplate:   "project {{prompt}}",
		StylePreset:      "project-style",
		BestOfConfig:     &domain.BestOfConfig{Strategy: domain.BestOfStrategyHTTPJudge, JudgePrompt: "project judge", AutoRejectNonSelected: true},
		GenerationConfig: []byte(`{"quality":"high"}`),
	}

	result, err := applyQualityProfile(req, profile)
	if err != nil {
		t.Fatalf("applyQualityProfile returned error: %v", err)
	}
	if result.Request.Prompt != "request base" {
		t.Fatalf("request prompt template did not override project profile: %q", result.Request.Prompt)
	}
	if result.Request.StylePreset != "request-style" {
		t.Fatalf("request style did not override project profile: %q", result.Request.StylePreset)
	}
	if string(result.Request.GenerationConfig) != `{"quality":"low"}` {
		t.Fatalf("request generation_config did not override project profile: %s", result.Request.GenerationConfig)
	}
	if result.Request.BestOfConfig == nil || result.Request.BestOfConfig.Strategy != domain.BestOfStrategyHTTPJudge {
		t.Fatalf("project best_of_config was not applied: %#v", result.Request.BestOfConfig)
	}
	if !result.Request.BestOfConfig.AutoRejectNonSelected {
		t.Fatalf("project auto_reject_non_selected was not applied: %#v", result.Request.BestOfConfig)
	}
}

func TestApplyQualityProfileAllowsBestOfConfigOverride(t *testing.T) {
	req := domain.CreateTaskRequest{
		Prompt:                   "base",
		UseProjectQualityProfile: true,
		BestOfConfig:             &domain.BestOfConfig{Strategy: domain.BestOfStrategyLocalMetadata},
	}
	profile := domain.QualityProfile{
		BestOfConfig: &domain.BestOfConfig{Strategy: domain.BestOfStrategyHTTPJudge, JudgePrompt: "project judge"},
	}

	result, err := applyQualityProfile(req, profile)
	if err != nil {
		t.Fatalf("applyQualityProfile returned error: %v", err)
	}
	if result.Request.BestOfConfig == nil || result.Request.BestOfConfig.Strategy != domain.BestOfStrategyLocalMetadata {
		t.Fatalf("request best_of_config did not override project profile: %#v", result.Request.BestOfConfig)
	}
}

func TestNormalizeBestOfConfigRejectsPromptWithoutStrategy(t *testing.T) {
	_, err := normalizeBestOfConfig(&domain.BestOfConfig{JudgePrompt: "pick the strongest cover"})
	if err == nil {
		t.Fatal("expected normalizeBestOfConfig to reject judge_prompt without strategy")
	}
}

func TestNormalizeBestOfConfigAllowsAutoRejectWithoutExplicitStrategy(t *testing.T) {
	config, err := normalizeBestOfConfig(&domain.BestOfConfig{AutoRejectNonSelected: true})
	if err != nil {
		t.Fatalf("normalizeBestOfConfig returned error: %v", err)
	}
	if config == nil || !config.AutoRejectNonSelected || config.Strategy != "" {
		t.Fatalf("unexpected normalized config: %#v", config)
	}
}

func TestNormalizeAdvancedInputDescriptors(t *testing.T) {
	references := normalizeReferenceImages([]domain.ReferenceImage{
		{
			ID:       " ref_local ",
			Role:     " edit_target ",
			Source:   " web-indexeddb ",
			MimeType: " image/png ",
			Width:    -100,
			Height:   768,
		},
		{Source: "missing-id"},
	})
	if len(references) != 1 {
		t.Fatalf("unexpected normalized references: %#v", references)
	}
	if references[0].ID != "ref_local" || references[0].Source != "web-indexeddb" || references[0].MimeType != "image/png" {
		t.Fatalf("reference descriptor fields were not normalized: %#v", references[0])
	}
	if references[0].Width != 0 || references[0].Height != 768 {
		t.Fatalf("reference dimensions were not normalized: %#v", references[0])
	}

	mask := normalizeMaskImage(&domain.MaskImage{
		TargetImageID: " ref_local ",
		Source:        " web-mask-draft ",
		MimeType:      " image/png ",
		Width:         -1,
		Height:        512,
		HasMask:       true,
	})
	if mask == nil {
		t.Fatal("mask descriptor was dropped")
	}
	if mask.TargetImageID != "ref_local" || mask.Source != "web-mask-draft" || mask.MimeType != "image/png" || !mask.HasMask {
		t.Fatalf("mask descriptor fields were not normalized: %#v", mask)
	}
	if mask.Width != 0 || mask.Height != 512 {
		t.Fatalf("mask dimensions were not normalized: %#v", mask)
	}
}
