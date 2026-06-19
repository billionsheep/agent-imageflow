package provider

import (
	"context"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

const (
	MockProviderID             = "mock"
	FalProviderID              = "fal"
	OpenAICompatibleProviderID = "openai-compatible"
)

type Adapter interface {
	Generate(context.Context, domain.Task) (Result, error)
}
