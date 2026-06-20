package provider

import (
	"encoding/json"
	"strings"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

type taskStructuredProviderInput struct {
	ReferenceImages    []domain.ReferenceImage       `json:"reference_images"`
	MaskImage          *domain.MaskImage             `json:"mask_image"`
	ResolvedInputFiles *resolvedTaskInputFiles       `json:"resolved_input_files"`
	GenerationConfig   json.RawMessage               `json:"generation_config"`
	ProviderProfile    domain.ProjectProviderProfile `json:"provider_profile"`
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
	if maxN < 1 {
		return 1
	}
	if maxN > 10 {
		return 10
	}
	return maxN
}
