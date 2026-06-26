package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

type taskStructuredProviderInput struct {
	ReferenceImages                []domain.ReferenceImage       `json:"reference_images"`
	MaskImage                      *domain.MaskImage             `json:"mask_image"`
	ResolvedInputFiles             *resolvedTaskInputFiles       `json:"resolved_input_files"`
	GenerationConfig               json.RawMessage               `json:"generation_config"`
	ProviderProfile                domain.ProjectProviderProfile `json:"provider_profile"`
	VisualContext                  *domain.VisualContextSnapshot `json:"visual_context_snapshot"`
	StoryContextV1                 *domain.StoryContextV1        `json:"story_context_v1"`
	CaptionLineage                 *domain.CaptionLineageSummary `json:"caption_lineage"`
	MetadataJSON                   json.RawMessage               `json:"metadata_json"`
	ReferenceAssetCount            int                           `json:"reference_asset_count"`
	ReferenceInputFileCount        int                           `json:"reference_input_file_count"`
	ProviderReferenceParticipation string                        `json:"provider_reference_participation"`
	ProviderReferenceSources       []string                      `json:"provider_reference_sources"`
	ProviderReferenceMIMETypes     []string                      `json:"provider_reference_mime_types"`
}

func referenceParticipationError(item resolvedTaskInputFile, err error) error {
	if err == nil {
		return nil
	}
	source := "asset_or_file"
	if strings.TrimSpace(item.InputFileID) != "" {
		source = "input_file"
	}
	parts := []string{
		"参考图未参与生成",
		"source=" + source,
	}
	if id := strings.TrimSpace(item.InputFileID); id != "" {
		parts = append(parts, "input_file_id="+id)
	}
	if mimeType := strings.TrimSpace(item.MimeType); mimeType != "" {
		parts = append(parts, "mime_type="+mimeType)
	}
	if role := strings.TrimSpace(item.Role); role != "" {
		parts = append(parts, "role="+role)
	}
	return fmt.Errorf("%s: %w", strings.Join(parts, " "), err)
}

type resolvedTaskInputFiles struct {
	ReferenceImages []resolvedTaskInputFile `json:"reference_images"`
	MaskImage       *resolvedTaskInputFile  `json:"mask_image,omitempty"`
}

type resolvedTaskInputFile struct {
	InputFileID   string `json:"input_file_id"`
	Kind          string `json:"kind"`
	FilePath      string `json:"file_path"`
	MimeType      string `json:"mime_type"`
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	Role          string `json:"role,omitempty"`
	TargetImageID string `json:"target_image_id,omitempty"`
}

func parseTaskStructuredProviderInput(task domain.Task) taskStructuredProviderInput {
	if len(task.StructuredInputJSON) == 0 {
		return taskStructuredProviderInput{}
	}
	var input taskStructuredProviderInput
	if err := json.Unmarshal(task.StructuredInputJSON, &input); err != nil {
		return taskStructuredProviderInput{}
	}
	return input
}

func taskProviderParameters(task domain.Task, base map[string]any) []byte {
	parameters := make(map[string]any, len(base)+3)
	for key, value := range base {
		parameters[key] = value
	}

	input := parseTaskStructuredProviderInput(task)
	if len(input.ReferenceImages) > 0 {
		parameters["reference_images"] = input.ReferenceImages
	}
	if input.MaskImage != nil {
		parameters["mask_image"] = input.MaskImage
	}
	if len(input.GenerationConfig) > 0 {
		var generationConfig any
		if json.Unmarshal(input.GenerationConfig, &generationConfig) == nil {
			parameters["generation_config"] = generationConfig
		}
	}
	if input.ProviderProfile.Enabled {
		parameters["provider_profile"] = input.ProviderProfile
	}
	if input.VisualContext != nil {
		parameters["visual_context_snapshot"] = input.VisualContext
	}
	if input.StoryContextV1 != nil {
		parameters["story_context_v1"] = input.StoryContextV1
	}
	captionLineage := input.CaptionLineage
	if captionLineage == nil {
		captionLineage = domain.CaptionLineageFromMetadataJSON(input.MetadataJSON)
	}
	if captionLineage != nil && !captionLineage.Empty() {
		parameters["caption_lineage"] = captionLineage
	}
	if input.ReferenceAssetCount > 0 {
		parameters["reference_asset_count"] = input.ReferenceAssetCount
	}
	if input.ReferenceInputFileCount > 0 {
		parameters["reference_input_file_count"] = input.ReferenceInputFileCount
	}
	if strings.TrimSpace(input.ProviderReferenceParticipation) != "" {
		parameters["provider_reference_participation"] = input.ProviderReferenceParticipation
	}
	if len(input.ProviderReferenceSources) > 0 {
		parameters["provider_reference_sources"] = input.ProviderReferenceSources
	}
	if len(input.ProviderReferenceMIMETypes) > 0 {
		parameters["provider_reference_mime_types"] = input.ProviderReferenceMIMETypes
	}

	raw, err := json.Marshal(parameters)
	if err != nil {
		return []byte(`{}`)
	}
	return raw
}

func taskProviderModel(task domain.Task, providerID, fallback string) string {
	input := parseTaskStructuredProviderInput(task)
	if input.ProviderProfile.Enabled &&
		input.ProviderProfile.Provider == providerID &&
		input.ProviderProfile.Model != "" {
		return input.ProviderProfile.Model
	}
	return fallback
}

func taskProviderMaxN(task domain.Task, providerID string, fallback int) int {
	maxN := fallback
	input := parseTaskStructuredProviderInput(task)
	if input.ProviderProfile.Enabled &&
		strings.TrimSpace(input.ProviderProfile.Provider) == providerID &&
		input.ProviderProfile.MaxN > 0 {
		maxN = input.ProviderProfile.MaxN
	}
	if value, ok := configIntInRangeFromRaw(input.GenerationConfig, "max_n", 1, 10); ok {
		maxN = value
	}
	if maxN < 1 {
		return 1
	}
	if maxN > 10 {
		return 10
	}
	return maxN
}

func configIntInRangeFromRaw(raw json.RawMessage, key string, minValue, maxValue int) (int, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var config map[string]any
	if json.Unmarshal(raw, &config) != nil {
		return 0, false
	}
	value, ok := config[key]
	if !ok || value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		intValue := int(typed)
		if typed != float64(intValue) || intValue < minValue || intValue > maxValue {
			return 0, false
		}
		return intValue, true
	case int:
		if typed < minValue || typed > maxValue {
			return 0, false
		}
		return typed, true
	default:
		return 0, false
	}
}
