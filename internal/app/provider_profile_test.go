package app

import (
	"encoding/json"
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestNormalizeProjectProviderProfileKeepsOnlyNonSensitiveDefaults(t *testing.T) {
	stream := true
	partialImages := 2
	profile := normalizeProjectProviderProfile(domain.ProjectProviderProfile{
		Enabled:                  true,
		Provider:                 " openai-compatible ",
		Model:                    " gpt-image-2 ",
		BaseURL:                  " https://images.example.test/v1/ ",
		GenerationConfig:         json.RawMessage(`{"quality":"high"}`),
		UseProjectQualityProfile: true,
		APIMode:                  "responses",
		Stream:                   &stream,
		PartialImages:            &partialImages,
		MaxN:                     3,
		SupportsURLResult:        true,
		PreferredResponseFormat:  "url",
		MaxConcurrency:           2,
		TimeoutSeconds:           180,
	})

	if !profile.Enabled {
		t.Fatal("expected provider profile to stay enabled")
	}
	if profile.Provider != "openai-compatible" || profile.Model != "gpt-image-2" {
		t.Fatalf("profile strings were not normalized: %#v", profile)
	}
	if profile.BaseURL != "https://images.example.test/v1" {
		t.Fatalf("base_url should be trimmed without trailing slash, got %q", profile.BaseURL)
	}
	if string(profile.GenerationConfig) != `{"quality":"high"}` {
		t.Fatalf("generation_config should be preserved as object, got %s", profile.GenerationConfig)
	}
	if profile.APIMode != "responses" || profile.Stream == nil || *profile.Stream != true || profile.PartialImages == nil || *profile.PartialImages != 2 {
		t.Fatalf("streaming fields were not preserved: %#v", profile)
	}
	if profile.MaxN != 3 || !profile.SupportsURLResult || profile.PreferredResponseFormat != "url" || profile.MaxConcurrency != 2 || profile.TimeoutSeconds != 180 {
		t.Fatalf("capability fields were not preserved: %#v", profile)
	}
}

func TestNormalizeProjectProviderProfileRejectsInvalidGenerationConfig(t *testing.T) {
	profile := normalizeProjectProviderProfile(domain.ProjectProviderProfile{
		Enabled:          false,
		Provider:         "mock",
		GenerationConfig: json.RawMessage(`[]`),
	})

	if profile.Enabled {
		t.Fatal("expected disabled profile to stay disabled")
	}
	if profile.Provider != "mock" {
		t.Fatalf("disabled profile should preserve non-sensitive provider value, got %#v", profile)
	}
	if string(profile.GenerationConfig) != `{}` {
		t.Fatalf("invalid generation_config should normalize to empty object, got %s", profile.GenerationConfig)
	}
	if profile.MaxN != 4 {
		t.Fatalf("default max_n = %d, want 4", profile.MaxN)
	}
	if profile.PreferredResponseFormat != "url" {
		t.Fatalf("default preferred_response_format = %q, want url", profile.PreferredResponseFormat)
	}
}

func TestNormalizeProjectProviderProfileSanitizesCapabilityFields(t *testing.T) {
	partialImages := 9
	profile := normalizeProjectProviderProfile(domain.ProjectProviderProfile{
		Enabled:                 true,
		Provider:                "mock",
		APIMode:                 "bad",
		PartialImages:           &partialImages,
		MaxN:                    99,
		PreferredResponseFormat: "stream",
		MaxConcurrency:          -1,
		TimeoutSeconds:          -2,
	})

	if profile.MaxN != 10 {
		t.Fatalf("max_n = %d, want cap 10", profile.MaxN)
	}
	if profile.APIMode != "images" || profile.PartialImages == nil || *profile.PartialImages != 3 {
		t.Fatalf("api/partial fields should be sanitized: %#v", profile)
	}
	if profile.PreferredResponseFormat != "url" {
		t.Fatalf("preferred_response_format = %q, want url", profile.PreferredResponseFormat)
	}
	if profile.MaxConcurrency != 0 || profile.TimeoutSeconds != 0 {
		t.Fatalf("negative limits should normalize to 0: %#v", profile)
	}
}
