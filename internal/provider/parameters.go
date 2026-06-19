package provider

import (
	"encoding/json"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

type taskStructuredProviderInput struct {
	ReferenceImages    []domain.ReferenceImage `json:"reference_images"`
	MaskImage          *domain.MaskImage       `json:"mask_image"`
	ResolvedInputFiles *resolvedTaskInputFiles `json:"resolved_input_files"`
	GenerationConfig   json.RawMessage         `json:"generation_config"`
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

	raw, err := json.Marshal(parameters)
	if err != nil {
		return []byte(`{}`)
	}
	return raw
}
