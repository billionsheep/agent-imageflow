package app

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

const qualityProfileSnapshotKey = "quality_profile_snapshot"

var promptTemplateTokenPattern = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

type qualityNormalizationResult struct {
	Request    domain.CreateTaskRequest
	Metadata   map[string]any
	HasQuality bool
	Source     string
}

func normalizeQualityProfile(profile domain.QualityProfile) (domain.QualityProfile, error) {
	profile.PromptTemplate = strings.TrimSpace(profile.PromptTemplate)
	profile.NegativePrompt = strings.TrimSpace(profile.NegativePrompt)
	profile.StylePreset = strings.TrimSpace(profile.StylePreset)
	profile.ReferenceImages = normalizeReferenceImages(profile.ReferenceImages)
	bestOfConfig, err := normalizeBestOfConfig(profile.BestOfConfig)
	if err != nil {
		return profile, err
	}
	profile.BestOfConfig = bestOfConfig
	if len(profile.GenerationConfig) > 0 {
		if !json.Valid(profile.GenerationConfig) {
			return profile, fmt.Errorf("generation_config must be valid JSON")
		}
		profile.GenerationConfig = compactJSON(profile.GenerationConfig)
	}
	return profile, nil
}

func applyQualityProfile(req domain.CreateTaskRequest, projectProfile domain.QualityProfile) (qualityNormalizationResult, error) {
	metadata, err := metadataMap(req.MetadataJSON)
	if err != nil {
		return qualityNormalizationResult{}, err
	}

	projectProfile, err = normalizeQualityProfile(projectProfile)
	if err != nil {
		return qualityNormalizationResult{}, err
	}

	req.PromptTemplate = strings.TrimSpace(req.PromptTemplate)
	req.ReferenceImages = normalizeReferenceImages(req.ReferenceImages)
	req.BestOfConfig, err = normalizeBestOfConfig(req.BestOfConfig)
	if err != nil {
		return qualityNormalizationResult{}, err
	}
	if len(req.GenerationConfig) > 0 {
		if !json.Valid(req.GenerationConfig) {
			return qualityNormalizationResult{}, fmt.Errorf("generation_config must be valid JSON")
		}
		req.GenerationConfig = compactJSON(req.GenerationConfig)
	}

	source := "request"
	if req.UseProjectQualityProfile {
		source = "project"
		if req.PromptTemplate == "" {
			req.PromptTemplate = projectProfile.PromptTemplate
		}
		if req.NegativePrompt == "" {
			req.NegativePrompt = projectProfile.NegativePrompt
		}
		if req.StylePreset == "" {
			req.StylePreset = projectProfile.StylePreset
		}
		if len(req.ReferenceImages) == 0 {
			req.ReferenceImages = projectProfile.ReferenceImages
		}
		if req.BestOfConfig == nil && projectProfile.BestOfConfig != nil {
			req.BestOfConfig = cloneBestOfConfig(projectProfile.BestOfConfig)
		}
		if len(req.GenerationConfig) == 0 {
			req.GenerationConfig = projectProfile.GenerationConfig
		}
	}

	if req.PromptTemplate != "" {
		variables := templateVariables(req, metadata)
		rendered := strings.TrimSpace(renderPromptTemplate(req.PromptTemplate, variables))
		if rendered != "" {
			req.Prompt = rendered
		}
		req.TemplateVariables = variables
	}

	hasQuality := req.UseProjectQualityProfile ||
		req.PromptTemplate != "" ||
		req.NegativePrompt != "" ||
		req.StylePreset != "" ||
		len(req.ReferenceImages) > 0 ||
		req.BestOfConfig != nil ||
		len(req.GenerationConfig) > 0

	if hasQuality {
		metadata[qualityProfileSnapshotKey] = qualitySnapshot(req, source)
	}
	metadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return qualityNormalizationResult{}, err
	}
	req.MetadataJSON = metadataRaw

	return qualityNormalizationResult{
		Request:    req,
		Metadata:   metadata,
		HasQuality: hasQuality,
		Source:     source,
	}, nil
}

func metadataMap(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("metadata_json must be valid JSON")
	}
	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, fmt.Errorf("metadata_json must be a JSON object: %w", err)
	}
	if metadata == nil {
		return map[string]any{}, nil
	}
	return metadata, nil
}

func templateVariables(req domain.CreateTaskRequest, metadata map[string]any) map[string]any {
	variables := map[string]any{}
	for key, value := range metadata {
		if key == qualityProfileSnapshotKey {
			continue
		}
		variables[key] = value
	}
	for key, value := range req.TemplateVariables {
		variables[key] = value
	}
	setDefaultVariable(variables, "prompt", req.Prompt)
	setDefaultVariable(variables, "base_prompt", req.Prompt)
	setDefaultVariable(variables, "title", req.Title)
	setDefaultVariable(variables, "purpose", req.Purpose)
	setDefaultVariable(variables, "style_preset", req.StylePreset)
	setDefaultVariable(variables, "aspect_ratio", req.AspectRatio)
	return variables
}

func setDefaultVariable(variables map[string]any, key string, value string) {
	if _, exists := variables[key]; exists {
		return
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	variables[key] = value
}

func renderPromptTemplate(template string, variables map[string]any) string {
	return promptTemplateTokenPattern.ReplaceAllStringFunc(template, func(token string) string {
		matches := promptTemplateTokenPattern.FindStringSubmatch(token)
		if len(matches) != 2 {
			return token
		}
		value, ok := variables[matches[1]]
		if !ok {
			return token
		}
		return strings.TrimSpace(fmt.Sprint(value))
	})
}

func normalizeReferenceImages(items []domain.ReferenceImage) []domain.ReferenceImage {
	if len(items) == 0 {
		return nil
	}
	normalized := make([]domain.ReferenceImage, 0, len(items))
	for _, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.URL = strings.TrimSpace(item.URL)
		item.AssetID = strings.TrimSpace(item.AssetID)
		item.InputFileID = strings.TrimSpace(item.InputFileID)
		item.Role = strings.TrimSpace(item.Role)
		item.Source = strings.TrimSpace(item.Source)
		item.MimeType = strings.TrimSpace(item.MimeType)
		if item.Width < 0 {
			item.Width = 0
		}
		if item.Height < 0 {
			item.Height = 0
		}
		if item.ID == "" && item.URL == "" && item.AssetID == "" && item.InputFileID == "" {
			continue
		}
		normalized = append(normalized, item)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeMaskImage(item *domain.MaskImage) *domain.MaskImage {
	if item == nil {
		return nil
	}
	normalized := *item
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.URL = strings.TrimSpace(normalized.URL)
	normalized.AssetID = strings.TrimSpace(normalized.AssetID)
	normalized.InputFileID = strings.TrimSpace(normalized.InputFileID)
	normalized.TargetImageID = strings.TrimSpace(normalized.TargetImageID)
	normalized.Source = strings.TrimSpace(normalized.Source)
	normalized.MimeType = strings.TrimSpace(normalized.MimeType)
	if normalized.Width < 0 {
		normalized.Width = 0
	}
	if normalized.Height < 0 {
		normalized.Height = 0
	}
	if normalized.ID == "" && normalized.URL == "" && normalized.AssetID == "" && normalized.InputFileID == "" && normalized.TargetImageID == "" {
		return nil
	}
	return &normalized
}

func qualitySnapshot(req domain.CreateTaskRequest, source string) map[string]any {
	snapshot := map[string]any{
		"source":                      source,
		"use_project_quality_profile": req.UseProjectQualityProfile,
		"effective_prompt":            req.Prompt,
	}
	if req.PromptTemplate != "" {
		snapshot["prompt_template"] = req.PromptTemplate
	}
	if len(req.TemplateVariables) > 0 {
		snapshot["template_variables"] = req.TemplateVariables
	}
	if req.NegativePrompt != "" {
		snapshot["negative_prompt"] = req.NegativePrompt
	}
	if req.StylePreset != "" {
		snapshot["style_preset"] = req.StylePreset
	}
	if len(req.ReferenceImages) > 0 {
		snapshot["reference_images"] = req.ReferenceImages
	}
	if req.BestOfConfig != nil {
		snapshot["best_of_config"] = req.BestOfConfig
	}
	if len(req.GenerationConfig) > 0 {
		snapshot["generation_config"] = json.RawMessage(req.GenerationConfig)
	}
	return snapshot
}

func normalizeBestOfConfig(config *domain.BestOfConfig) (*domain.BestOfConfig, error) {
	if config == nil {
		return nil, nil
	}
	normalized := &domain.BestOfConfig{
		Strategy:              strings.TrimSpace(config.Strategy),
		JudgePrompt:           strings.TrimSpace(config.JudgePrompt),
		AutoRejectNonSelected: config.AutoRejectNonSelected,
	}
	if normalized.Strategy == "" && normalized.JudgePrompt == "" && !normalized.AutoRejectNonSelected {
		return nil, nil
	}
	if normalized.Strategy == "" && normalized.JudgePrompt != "" {
		return nil, fmt.Errorf("best_of_config.strategy is required when best_of_config.judge_prompt is set")
	}
	if normalized.Strategy == "" {
		return normalized, nil
	}
	strategy, ok := domain.NormalizeBestOfStrategy(normalized.Strategy)
	if !ok {
		return nil, fmt.Errorf("unknown best_of_config.strategy %q", normalized.Strategy)
	}
	normalized.Strategy = strategy
	return normalized, nil
}

func cloneBestOfConfig(config *domain.BestOfConfig) *domain.BestOfConfig {
	if config == nil {
		return nil
	}
	cloned := *config
	return &cloned
}

func compactJSON(raw json.RawMessage) json.RawMessage {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return raw
	}
	compact, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	return compact
}
