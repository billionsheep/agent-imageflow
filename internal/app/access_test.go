package app

import (
	"testing"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func TestNormalizeProjectAccessConfigEnablesAndRotatesKey(t *testing.T) {
	enabled := true
	config, err := normalizeProjectAccessConfig(domain.ProjectAccessConfig{}, domain.ProjectAccessConfigUpdateRequest{
		APIKeyEnabled: &enabled,
		APIKeyName:    "editor",
		APIKey:        "proj_live_secret_1234",
	})
	if err != nil {
		t.Fatalf("normalizeProjectAccessConfig returned error: %v", err)
	}
	if !config.IsEnabled() {
		t.Fatal("expected project access config to be enabled")
	}
	if config.APIKeyName != "editor" {
		t.Fatalf("unexpected api key name: %q", config.APIKeyName)
	}
	if config.APIKeyPreview != "proj...1234" {
		t.Fatalf("unexpected api key preview: %q", config.APIKeyPreview)
	}
	if config.APIKeyHash == "" || config.APIKeyHash == "proj_live_secret_1234" {
		t.Fatalf("expected api key to be hashed, got %q", config.APIKeyHash)
	}
	if len(config.APIKeys) != 1 {
		t.Fatalf("expected 1 api key entry, got %d", len(config.APIKeys))
	}
	if config.APIKeys[0].ID != domain.ProjectAPIKeyDefaultID {
		t.Fatalf("expected default key id %q, got %q", domain.ProjectAPIKeyDefaultID, config.APIKeys[0].ID)
	}
}

func TestNormalizeProjectAccessConfigRequiresKeyOnFirstEnable(t *testing.T) {
	enabled := true
	_, err := normalizeProjectAccessConfig(domain.ProjectAccessConfig{}, domain.ProjectAccessConfigUpdateRequest{
		APIKeyEnabled: &enabled,
		APIKeyName:    "editor",
	})
	if err == nil {
		t.Fatal("expected missing api key to fail when enabling for the first time")
	}
}

func TestNormalizeProjectAccessConfigKeepsExistingHashWhenRenaming(t *testing.T) {
	enabled := true
	current := domain.ProjectAccessConfig{
		APIKeyEnabled: true,
		APIKeyName:    "default",
		APIKeyPreview: "proj...1234",
		APIKeyHash:    hashProjectAPIKey("proj_live_secret_1234"),
	}
	config, err := normalizeProjectAccessConfig(current, domain.ProjectAccessConfigUpdateRequest{
		APIKeyEnabled: &enabled,
		APIKeyName:    "renamed",
	})
	if err != nil {
		t.Fatalf("normalizeProjectAccessConfig returned error: %v", err)
	}
	if config.APIKeyHash != current.APIKeyHash {
		t.Fatal("expected existing api key hash to be preserved")
	}
	if config.APIKeyName != "renamed" {
		t.Fatalf("expected renamed key, got %q", config.APIKeyName)
	}
}

func TestCompareProjectAPIKey(t *testing.T) {
	config := projectAccessConfigFromKeys([]domain.ProjectAPIKey{
		{
			ID:      "pak_old",
			Name:    "old",
			Preview: "proj...1111",
			Hash:    hashProjectAPIKey("proj_live_secret_1111"),
			Enabled: false,
		},
		{
			ID:      "pak_new",
			Name:    "new",
			Preview: "proj...2222",
			Hash:    hashProjectAPIKey("proj_live_secret_2222"),
			Enabled: true,
		},
	})
	matched, ok := matchProjectAPIKey(config, "proj_live_secret_2222")
	if !ok {
		t.Fatal("expected project api key to match")
	}
	if matched.ID != "pak_new" || matched.Name != "new" {
		t.Fatalf("expected matched key to be pak_new/new, got %#v", matched)
	}
	if _, ok := matchProjectAPIKey(config, "proj_live_secret_1111"); ok {
		t.Fatal("expected disabled api key not to match")
	}
	if _, ok := matchProjectAPIKey(config, "wrong-secret"); ok {
		t.Fatal("expected mismatched api key to fail")
	}
}

func TestNormalizeProjectAccessConfigAddUpdateAndDeleteKey(t *testing.T) {
	current := projectAccessConfigFromKeys([]domain.ProjectAPIKey{
		{
			ID:      domain.ProjectAPIKeyDefaultID,
			Name:    "default",
			Preview: "proj...1234",
			Hash:    hashProjectAPIKey("proj_live_secret_1234"),
			Enabled: true,
		},
	})

	added, err := normalizeProjectAccessConfig(current, domain.ProjectAccessConfigUpdateRequest{
		Action:        domain.ProjectAccessActionAddKey,
		APIKeyName:    "automation",
		APIKey:        "proj_live_secret_5678",
		APIKeyEnabled: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("add key returned error: %v", err)
	}
	if len(added.APIKeys) != 2 {
		t.Fatalf("expected 2 api keys after add, got %d", len(added.APIKeys))
	}
	second := added.APIKeys[1]
	if second.Name != "automation" || !second.Enabled {
		t.Fatalf("unexpected second key after add: %#v", second)
	}

	updated, err := normalizeProjectAccessConfig(added, domain.ProjectAccessConfigUpdateRequest{
		Action:        domain.ProjectAccessActionUpdateKey,
		APIKeyID:      second.ID,
		APIKeyEnabled: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("update key returned error: %v", err)
	}
	if updated.APIKeys[1].Enabled {
		t.Fatalf("expected second key to be disabled, got %#v", updated.APIKeys[1])
	}

	deleted, err := normalizeProjectAccessConfig(updated, domain.ProjectAccessConfigUpdateRequest{
		Action:   domain.ProjectAccessActionDeleteKey,
		APIKeyID: second.ID,
	})
	if err != nil {
		t.Fatalf("delete key returned error: %v", err)
	}
	if len(deleted.APIKeys) != 1 {
		t.Fatalf("expected 1 api key after delete, got %d", len(deleted.APIKeys))
	}
}

func TestNormalizeProjectAccessConfigRejectsDuplicateKeyName(t *testing.T) {
	current := projectAccessConfigFromKeys([]domain.ProjectAPIKey{
		{
			ID:      domain.ProjectAPIKeyDefaultID,
			Name:    "default",
			Preview: "proj...1234",
			Hash:    hashProjectAPIKey("proj_live_secret_1234"),
			Enabled: true,
		},
	})
	_, err := normalizeProjectAccessConfig(current, domain.ProjectAccessConfigUpdateRequest{
		Action:     domain.ProjectAccessActionAddKey,
		APIKeyName: "default",
		APIKey:     "proj_live_secret_5678",
	})
	if err == nil {
		t.Fatal("expected duplicate api key name to fail")
	}
}

func boolPtr(value bool) *bool {
	return &value
}
