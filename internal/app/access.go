package app

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

func (s *Service) GetProjectAccessConfig(ctx context.Context, workspaceID, projectID string) (domain.ProjectAccessConfigResponse, error) {
	config, err := s.store.GetProjectAccessConfig(ctx, workspaceID, projectID)
	if err != nil {
		return domain.ProjectAccessConfigResponse{}, err
	}
	return domain.ProjectAccessConfigResponse{
		WorkspaceID:  workspaceID,
		ProjectID:    projectID,
		AccessConfig: config.Public(),
	}, nil
}

func (s *Service) GetProjectAccessConfigViewByProjectID(ctx context.Context, projectID string) (domain.ProjectAccessConfigView, error) {
	config, err := s.store.GetProjectAccessConfigByProjectID(ctx, projectID)
	if err != nil {
		return domain.ProjectAccessConfigView{}, err
	}
	return config.Public(), nil
}

func (s *Service) UpdateProjectAccessConfig(ctx context.Context, workspaceID, projectID string, req domain.ProjectAccessConfigUpdateRequest) (domain.ProjectAccessConfigResponse, error) {
	current, err := s.store.GetProjectAccessConfig(ctx, workspaceID, projectID)
	if err != nil {
		return domain.ProjectAccessConfigResponse{}, err
	}
	updated, err := normalizeProjectAccessConfig(current, req)
	if err != nil {
		return domain.ProjectAccessConfigResponse{}, err
	}
	saved, err := s.store.UpdateProjectAccessConfig(ctx, workspaceID, projectID, updated)
	if err != nil {
		return domain.ProjectAccessConfigResponse{}, err
	}
	return domain.ProjectAccessConfigResponse{
		WorkspaceID:  workspaceID,
		ProjectID:    projectID,
		AccessConfig: saved.Public(),
	}, nil
}

func (s *Service) ValidateProjectAPIKey(ctx context.Context, workspaceID, projectID, apiKey string) (bool, bool, domain.ProjectAPIKeyView, error) {
	config, err := s.store.GetProjectAccessConfig(ctx, workspaceID, projectID)
	if err != nil {
		return false, false, domain.ProjectAPIKeyView{}, err
	}
	required := validateProjectAPIKey(config, apiKey)
	matched, valid := matchProjectAPIKey(config, apiKey)
	return required, valid, matched, nil
}

func (s *Service) ValidateProjectAPIKeyByProjectID(ctx context.Context, projectID, apiKey string) (bool, bool, domain.ProjectAPIKeyView, error) {
	config, err := s.store.GetProjectAccessConfigByProjectID(ctx, projectID)
	if err != nil {
		return false, false, domain.ProjectAPIKeyView{}, err
	}
	required := validateProjectAPIKey(config, apiKey)
	matched, valid := matchProjectAPIKey(config, apiKey)
	return required, valid, matched, nil
}

func (s *Service) GetTaskScope(ctx context.Context, taskID string) (domain.Scope, error) {
	return s.store.GetTaskScope(ctx, taskID)
}

func (s *Service) GetAssetScope(ctx context.Context, assetID string) (domain.Scope, error) {
	return s.store.GetAssetScope(ctx, assetID)
}

func normalizeProjectAccessConfig(current domain.ProjectAccessConfig, req domain.ProjectAccessConfigUpdateRequest) (domain.ProjectAccessConfig, error) {
	current = current.Normalize()
	switch strings.TrimSpace(req.Action) {
	case "":
		return normalizeProjectAccessConfigLegacy(current, req)
	case domain.ProjectAccessActionAddKey:
		return addProjectAccessKey(current, req)
	case domain.ProjectAccessActionUpdateKey:
		return updateProjectAccessKey(current, req)
	case domain.ProjectAccessActionDeleteKey:
		return deleteProjectAccessKey(current, req)
	default:
		return domain.ProjectAccessConfig{}, fmt.Errorf("unknown project access action %q", strings.TrimSpace(req.Action))
	}
}

func normalizeProjectAccessConfigLegacy(current domain.ProjectAccessConfig, req domain.ProjectAccessConfigUpdateRequest) (domain.ProjectAccessConfig, error) {
	enabled := current.IsEnabled()
	if req.APIKeyEnabled != nil {
		enabled = *req.APIKeyEnabled
	}
	if !enabled {
		return domain.ProjectAccessConfig{}, nil
	}

	keys := append([]domain.ProjectAPIKey(nil), current.APIKeys...)
	name := strings.TrimSpace(req.APIKeyName)
	if len(keys) == 0 {
		if name == "" {
			name = domain.ProjectAPIKeyDefaultName
		}
		apiKey := strings.TrimSpace(req.APIKey)
		if apiKey == "" {
			return domain.ProjectAccessConfig{}, fmt.Errorf("api_key is required when enabling project api key")
		}
		return projectAccessConfigFromKeys([]domain.ProjectAPIKey{{
			ID:      domain.ProjectAPIKeyDefaultID,
			Name:    name,
			Preview: previewProjectAPIKey(apiKey),
			Hash:    hashProjectAPIKey(apiKey),
			Enabled: true,
		}}), nil
	}

	targetIndex := firstEnabledProjectAPIKeyIndex(keys)
	if targetIndex < 0 {
		targetIndex = 0
	}
	if name == "" {
		name = keys[targetIndex].Name
	}
	if name == "" {
		name = domain.ProjectAPIKeyDefaultName
	}
	if err := ensureUniqueProjectAPIKeyName(keys, name, keys[targetIndex].ID); err != nil {
		return domain.ProjectAccessConfig{}, err
	}
	keys[targetIndex].Name = name
	if apiKey := strings.TrimSpace(req.APIKey); apiKey != "" {
		keys[targetIndex].Preview = previewProjectAPIKey(apiKey)
		keys[targetIndex].Hash = hashProjectAPIKey(apiKey)
	}
	keys[targetIndex].Enabled = true
	return projectAccessConfigFromKeys(keys), nil
}

func addProjectAccessKey(current domain.ProjectAccessConfig, req domain.ProjectAccessConfigUpdateRequest) (domain.ProjectAccessConfig, error) {
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		return domain.ProjectAccessConfig{}, fmt.Errorf("api_key is required when adding a project api key")
	}
	name := strings.TrimSpace(req.APIKeyName)
	if name == "" {
		return domain.ProjectAccessConfig{}, fmt.Errorf("api_key_name is required when adding a project api key")
	}
	keys := append([]domain.ProjectAPIKey(nil), current.APIKeys...)
	if err := ensureUniqueProjectAPIKeyName(keys, name, ""); err != nil {
		return domain.ProjectAccessConfig{}, err
	}
	enabled := true
	if req.APIKeyEnabled != nil {
		enabled = *req.APIKeyEnabled
	}
	keys = append(keys, domain.ProjectAPIKey{
		ID:      domain.NewID("pak"),
		Name:    name,
		Preview: previewProjectAPIKey(apiKey),
		Hash:    hashProjectAPIKey(apiKey),
		Enabled: enabled,
	})
	return projectAccessConfigFromKeys(keys), nil
}

func updateProjectAccessKey(current domain.ProjectAccessConfig, req domain.ProjectAccessConfigUpdateRequest) (domain.ProjectAccessConfig, error) {
	keyID := strings.TrimSpace(req.APIKeyID)
	if keyID == "" {
		return domain.ProjectAccessConfig{}, fmt.Errorf("api_key_id is required when updating a project api key")
	}
	keys := append([]domain.ProjectAPIKey(nil), current.APIKeys...)
	index := findProjectAPIKeyIndex(keys, keyID)
	if index < 0 {
		return domain.ProjectAccessConfig{}, fmt.Errorf("project api key %q not found", keyID)
	}
	changed := false
	if name := strings.TrimSpace(req.APIKeyName); name != "" {
		if err := ensureUniqueProjectAPIKeyName(keys, name, keyID); err != nil {
			return domain.ProjectAccessConfig{}, err
		}
		keys[index].Name = name
		changed = true
	}
	if apiKey := strings.TrimSpace(req.APIKey); apiKey != "" {
		keys[index].Preview = previewProjectAPIKey(apiKey)
		keys[index].Hash = hashProjectAPIKey(apiKey)
		changed = true
	}
	if req.APIKeyEnabled != nil {
		keys[index].Enabled = *req.APIKeyEnabled
		changed = true
	}
	if !changed {
		return domain.ProjectAccessConfig{}, fmt.Errorf("no project api key changes requested")
	}
	return projectAccessConfigFromKeys(keys), nil
}

func deleteProjectAccessKey(current domain.ProjectAccessConfig, req domain.ProjectAccessConfigUpdateRequest) (domain.ProjectAccessConfig, error) {
	keyID := strings.TrimSpace(req.APIKeyID)
	if keyID == "" {
		return domain.ProjectAccessConfig{}, fmt.Errorf("api_key_id is required when deleting a project api key")
	}
	keys := append([]domain.ProjectAPIKey(nil), current.APIKeys...)
	index := findProjectAPIKeyIndex(keys, keyID)
	if index < 0 {
		return domain.ProjectAccessConfig{}, fmt.Errorf("project api key %q not found", keyID)
	}
	keys = slices.Delete(keys, index, index+1)
	return projectAccessConfigFromKeys(keys), nil
}

func projectAccessConfigFromKeys(keys []domain.ProjectAPIKey) domain.ProjectAccessConfig {
	if len(keys) == 0 {
		return domain.ProjectAccessConfig{}
	}
	return domain.ProjectAccessConfig{
		APIKeys: keys,
	}.Normalize()
}

func findProjectAPIKeyIndex(keys []domain.ProjectAPIKey, keyID string) int {
	keyID = strings.TrimSpace(keyID)
	for idx, key := range keys {
		if strings.TrimSpace(key.ID) == keyID {
			return idx
		}
	}
	return -1
}

func firstEnabledProjectAPIKeyIndex(keys []domain.ProjectAPIKey) int {
	for idx, key := range keys {
		if key.Enabled && strings.TrimSpace(key.Hash) != "" {
			return idx
		}
	}
	return -1
}

func ensureUniqueProjectAPIKeyName(keys []domain.ProjectAPIKey, name, exceptID string) error {
	candidate := strings.TrimSpace(name)
	if candidate == "" {
		return fmt.Errorf("api_key_name is required")
	}
	exceptID = strings.TrimSpace(exceptID)
	for _, key := range keys {
		if exceptID != "" && strings.TrimSpace(key.ID) == exceptID {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(key.Name), candidate) {
			return fmt.Errorf("project api key name %q already exists", candidate)
		}
	}
	return nil
}

func validateProjectAPIKey(config domain.ProjectAccessConfig, apiKey string) bool {
	return config.IsEnabled()
}

func matchProjectAPIKey(config domain.ProjectAccessConfig, apiKey string) (domain.ProjectAPIKeyView, bool) {
	if !config.IsEnabled() {
		return domain.ProjectAPIKeyView{}, true
	}
	trimmed := strings.TrimSpace(apiKey)
	if trimmed == "" {
		return domain.ProjectAPIKeyView{}, false
	}
	actual := hashProjectAPIKey(trimmed)
	for _, key := range config.Normalize().APIKeys {
		if !key.Enabled || strings.TrimSpace(key.Hash) == "" {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(actual), []byte(key.Hash)) == 1 {
			return key.Public(), true
		}
	}
	return domain.ProjectAPIKeyView{}, false
}

func hashProjectAPIKey(apiKey string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(apiKey)))
	return hex.EncodeToString(sum[:])
}

func previewProjectAPIKey(apiKey string) string {
	trimmed := strings.TrimSpace(apiKey)
	runes := []rune(trimmed)
	if len(runes) <= 8 {
		return "***"
	}
	return string(runes[:4]) + "..." + string(runes[len(runes)-4:])
}
