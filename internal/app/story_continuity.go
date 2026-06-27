package app

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

const storyContextV1MetadataKey = "story_context_v1"

type storyContinuityAssetSnapshot struct {
	Scope        domain.Scope
	AssetStatus  string
	MetadataJSON json.RawMessage
}

func storyContextV1FromMetadata(raw json.RawMessage) (*domain.StoryContextV1, map[string]any, error) {
	metadata, err := metadataMap(raw)
	if err != nil {
		return nil, nil, err
	}
	value, ok := metadata[storyContextV1MetadataKey]
	if !ok || value == nil {
		return nil, metadata, nil
	}
	serialized, err := json.Marshal(value)
	if err != nil {
		return nil, nil, fmt.Errorf("metadata_json.%s must be serializable: %w", storyContextV1MetadataKey, err)
	}
	var story domain.StoryContextV1
	if err := json.Unmarshal(serialized, &story); err != nil {
		return nil, nil, fmt.Errorf("metadata_json.%s must be a JSON object: %w", storyContextV1MetadataKey, err)
	}
	normalized := normalizeStoryContextV1(story)
	return &normalized, metadata, nil
}

func applyStoryContextBindingsToRequest(req domain.CreateTaskRequest, visualContext domain.ProjectVisualContext, story *domain.StoryContextV1) (domain.CreateTaskRequest, error) {
	if story == nil {
		return req, nil
	}
	charactersByID := map[string]domain.CharacterProfile{}
	for _, character := range visualContext.Characters {
		charactersByID[character.ID] = character
	}
	referencesByKey := map[string]domain.ProjectReferenceBinding{}
	for _, reference := range visualContext.References {
		if strings.TrimSpace(reference.ID) != "" {
			referencesByKey[reference.ID] = reference
		}
		if strings.TrimSpace(reference.AssetID) != "" {
			referencesByKey[reference.AssetID] = reference
		}
	}

	req.CharacterIDs = normalizeStringList(req.CharacterIDs)
	req.ReferenceAssetIDs = normalizeStringList(req.ReferenceAssetIDs)

	for _, characterID := range story.ReferenceBindings["character_reference"] {
		characterID = strings.TrimSpace(characterID)
		if characterID == "" {
			continue
		}
		character, ok := charactersByID[characterID]
		if !ok {
			return req, fmt.Errorf("story_context_v1 character_reference %q was not found in project visual context", characterID)
		}
		if strings.TrimSpace(character.Status) == "archived" {
			return req, fmt.Errorf("story_context_v1 character_reference %q is archived", characterID)
		}
		req.CharacterIDs = append(req.CharacterIDs, characterID)
	}

	for _, role := range []string{"environment_reference", "style_reference", "prop_reference"} {
		for _, bindingID := range story.ReferenceBindings[role] {
			bindingID = strings.TrimSpace(bindingID)
			if bindingID == "" {
				continue
			}
			reference, ok := referencesByKey[bindingID]
			if !ok {
				return req, fmt.Errorf("story_context_v1 %s %q was not found in project visual context", role, bindingID)
			}
			if strings.TrimSpace(reference.Status) == "archived" {
				return req, fmt.Errorf("story_context_v1 %s %q is archived", role, bindingID)
			}
			if strings.TrimSpace(reference.AssetID) == "" {
				return req, fmt.Errorf("story_context_v1 %s %q does not resolve to an asset", role, bindingID)
			}
			req.ReferenceAssetIDs = append(req.ReferenceAssetIDs, reference.AssetID)
		}
	}

	for _, assetID := range story.ReferenceBindings["previous_panel_reference"] {
		assetID = strings.TrimSpace(assetID)
		if assetID == "" {
			continue
		}
		if hasReferenceImageAssetRole(req.ReferenceImages, assetID, "previous_panel_reference") {
			continue
		}
		req.ReferenceImages = append(req.ReferenceImages, domain.ReferenceImage{
			ID:      assetID,
			AssetID: assetID,
			Role:    "previous_panel_reference",
			Source:  "story_context_v1",
			Weight:  1,
		})
	}

	req.CharacterIDs = normalizeStringList(req.CharacterIDs)
	req.ReferenceAssetIDs = normalizeStringList(req.ReferenceAssetIDs)
	req.ReferenceImages = normalizeReferenceImages(req.ReferenceImages)
	return req, nil
}

func prepareStoryContextV1ForTask(
	scope domain.Scope,
	metadataRaw json.RawMessage,
	req domain.CreateTaskRequest,
	resolved *resolvedTaskInputFiles,
	diagnostics domain.ReferenceParticipationDiagnostics,
	lookup func(assetID string) (storyContinuityAssetSnapshot, error),
) (*domain.StoryContextV1, json.RawMessage, error) {
	story, metadata, err := storyContextV1FromMetadata(metadataRaw)
	if err != nil {
		return nil, nil, err
	}
	if story == nil {
		return nil, domain.NormalizeMetadataJSON(metadataRaw), nil
	}

	story.StoryID = firstNonEmptyString(story.StoryID, mapString(metadata, "story_id"))
	if story.StoryID == "" {
		return nil, nil, fmt.Errorf("story_context_v1.story_id is required")
	}
	if story.GenerationMode == "" {
		story.GenerationMode = firstNonEmptyString(story.ContinuityPolicy.Mode, domain.StoryGenerationModeSequentialPreviousPanel)
	}
	story.ContinuityPolicy.Mode = firstNonEmptyString(story.ContinuityPolicy.Mode, story.GenerationMode)

	panel, err := selectStoryPanelPlanEntry(*story, metadata)
	if err != nil {
		return nil, nil, err
	}

	previousPanelAssetID := ""
	if panel.PanelIndex > 1 {
		previousPanelAssetID = firstNonEmptyString(
			firstReferenceBindingValue(story.ReferenceBindings, "previous_panel_reference"),
			mapString(metadata, "previous_panel_asset_id"),
			firstReferenceImageAssetIDByRole(req.ReferenceImages, "previous_panel_reference"),
		)
	}

	if isSequentialStoryContext(*story) {
		if req.SelectionMode != domain.SelectionManualOptional {
			return nil, nil, fmt.Errorf("story continuity sequential mode requires selection_mode %q", domain.SelectionManualOptional)
		}
		if max := story.ContinuityPolicy.MaxCandidatesPerPanel; max > 0 && req.RequestedCount > max {
			return nil, nil, fmt.Errorf("story continuity sequential mode allows at most %d candidates per panel", max)
		}
		if panel.PanelIndex > 1 && story.ContinuityPolicy.RequirePreviousSelectedAsset {
			if previousPanelAssetID == "" {
				return nil, nil, fmt.Errorf("story continuity sequential mode requires previous_panel_reference for panel %d", panel.PanelIndex)
			}
			if lookup == nil {
				return nil, nil, fmt.Errorf("story continuity sequential mode cannot verify previous panel selected asset %q", previousPanelAssetID)
			}
			snapshot, err := lookup(previousPanelAssetID)
			if err != nil {
				return nil, nil, fmt.Errorf("story continuity previous panel selected asset %q: %w", previousPanelAssetID, err)
			}
			if err := validateStoryContinuityAssetScope(scope, snapshot.Scope); err != nil {
				return nil, nil, fmt.Errorf("story continuity previous panel selected asset %q: %w", previousPanelAssetID, err)
			}
			if snapshot.AssetStatus != domain.AssetApproved && snapshot.AssetStatus != domain.AssetPublished {
				return nil, nil, fmt.Errorf("story continuity previous panel selected asset %q must be approved/published, got %s", previousPanelAssetID, snapshot.AssetStatus)
			}
			if err := validatePreviousPanelStoryLink(*story, panel, snapshot.MetadataJSON); err != nil {
				return nil, nil, fmt.Errorf("story continuity previous panel selected asset %q: %w", previousPanelAssetID, err)
			}
		}
	}

	story.ResolvedReferenceAssets = buildStoryResolvedReferenceAssets(scope, req.ReferenceImages, resolved)
	story.ContinuityWarnings = buildStoryContinuityWarnings(*story, diagnostics)

	metadata["story_id"] = story.StoryID
	metadata["scene_id"] = panel.SceneID
	metadata["scene_order"] = panel.PanelIndex
	metadata["panel_index"] = panel.PanelIndex
	setMetadataString(metadata, "story_revision", story.StoryRevision)
	setMetadataString(metadata, "story_plan_hash", story.StoryPlanHash)
	setMetadataString(metadata, "generation_mode", story.GenerationMode)
	setMetadataString(metadata, "narrative_role", panel.NarrativeRole)
	setMetadataString(metadata, "previous_state", panel.PreviousState)
	setMetadataString(metadata, "trigger_event", panel.TriggerEvent)
	setMetadataString(metadata, "visible_action", panel.VisibleAction)
	setMetadataString(metadata, "resulting_state", panel.ResultingState)
	setMetadataString(metadata, "dialogue", panel.Dialogue)
	setMetadataString(metadata, "dialogue_intent", panel.DialogueIntent)
	setMetadataString(metadata, "emotion_before", panel.EmotionBefore)
	setMetadataString(metadata, "emotion_after", panel.EmotionAfter)
	setMetadataString(metadata, "pose_change", panel.PoseChange)
	setMetadataString(metadata, "relationship_shift", panel.RelationshipShift)
	setMetadataString(metadata, "camera", panel.Camera)
	setMetadataString(metadata, "state_transition_notes", panel.StateTransitionNotes)
	setMetadataString(metadata, "target_path", panel.TargetPath)
	setMetadataString(metadata, "provider_reference_participation", diagnostics.ProviderReferenceParticipation)
	setMetadataString(metadata, "previous_panel_asset_id", previousPanelAssetID)
	setMetadataStringSlice(metadata, "must_keep_props", panel.MustKeepProps)
	setMetadataStringSlice(metadata, "allowed_changes", panel.AllowedChanges)
	setMetadataStringSlice(metadata, "must_change", panel.MustChange)
	setMetadataStringSlice(metadata, "must_not_keep", panel.MustNotKeep)
	metadata[storyContextV1MetadataKey] = story

	updatedMetadata, err := json.Marshal(metadata)
	if err != nil {
		return nil, nil, err
	}
	return story, updatedMetadata, nil
}

func normalizeStoryContextV1(story domain.StoryContextV1) domain.StoryContextV1 {
	story.SchemaVersion = firstNonEmptyString(strings.TrimSpace(story.SchemaVersion), "1.0")
	story.StoryID = strings.TrimSpace(story.StoryID)
	story.StoryRevision = strings.TrimSpace(story.StoryRevision)
	story.StoryPlanHash = strings.TrimSpace(story.StoryPlanHash)
	story.GenerationMode = strings.TrimSpace(story.GenerationMode)
	story.ReferenceBindings = normalizeStoryReferenceBindings(story.ReferenceBindings)
	story.PanelPlan = normalizeStoryPanelPlan(story.PanelPlan)
	story.ResolvedReferenceAssets = normalizeStoryResolvedReferenceAssets(story.ResolvedReferenceAssets)
	story.ContinuityWarnings = normalizeStoryContinuityWarnings(story.ContinuityWarnings)
	story.ContinuityPolicy = normalizeStoryContinuityPolicy(story.ContinuityPolicy)
	return story
}

func normalizeStoryReferenceBindings(bindings domain.StoryReferenceBindings) domain.StoryReferenceBindings {
	if len(bindings) == 0 {
		return nil
	}
	normalized := domain.StoryReferenceBindings{}
	for key, values := range bindings {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		cleaned := normalizeStringList(values)
		if len(cleaned) == 0 {
			continue
		}
		normalized[key] = cleaned
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeStoryPanelPlan(items []domain.StoryPanelPlanEntry) []domain.StoryPanelPlanEntry {
	if len(items) == 0 {
		return nil
	}
	normalized := make([]domain.StoryPanelPlanEntry, 0, len(items))
	for _, item := range items {
		item.SceneID = strings.TrimSpace(item.SceneID)
		item.NarrativeRole = strings.TrimSpace(item.NarrativeRole)
		item.PreviousState = strings.TrimSpace(item.PreviousState)
		item.TriggerEvent = strings.TrimSpace(item.TriggerEvent)
		item.VisibleAction = strings.TrimSpace(item.VisibleAction)
		item.ResultingState = strings.TrimSpace(item.ResultingState)
		item.Dialogue = strings.TrimSpace(item.Dialogue)
		item.DialogueIntent = strings.TrimSpace(item.DialogueIntent)
		item.EmotionBefore = strings.TrimSpace(item.EmotionBefore)
		item.EmotionAfter = strings.TrimSpace(item.EmotionAfter)
		item.PoseChange = strings.TrimSpace(item.PoseChange)
		item.RelationshipShift = strings.TrimSpace(item.RelationshipShift)
		item.Camera = strings.TrimSpace(item.Camera)
		item.StateTransitionNotes = strings.TrimSpace(item.StateTransitionNotes)
		item.TargetPath = strings.TrimSpace(item.TargetPath)
		item.MustKeepProps = normalizeStringList(item.MustKeepProps)
		item.AllowedChanges = normalizeStringList(item.AllowedChanges)
		item.MustChange = normalizeStringList(item.MustChange)
		item.MustNotKeep = normalizeStringList(item.MustNotKeep)
		if len(item.ReferenceRoles) > 0 {
			nextRoles := map[string][]string{}
			for key, values := range item.ReferenceRoles {
				key = strings.TrimSpace(key)
				if key == "" {
					continue
				}
				cleaned := normalizeStringList(values)
				if len(cleaned) == 0 {
					continue
				}
				nextRoles[key] = cleaned
			}
			if len(nextRoles) > 0 {
				item.ReferenceRoles = nextRoles
			} else {
				item.ReferenceRoles = nil
			}
		}
		if item.PanelIndex <= 0 && item.SceneID == "" {
			continue
		}
		normalized = append(normalized, item)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeStoryResolvedReferenceAssets(items []domain.StoryResolvedReferenceAsset) []domain.StoryResolvedReferenceAsset {
	if len(items) == 0 {
		return nil
	}
	normalized := make([]domain.StoryResolvedReferenceAsset, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item.Role = strings.TrimSpace(item.Role)
		item.AssetID = strings.TrimSpace(item.AssetID)
		item.Source = strings.TrimSpace(item.Source)
		item.WorkspaceID = strings.TrimSpace(item.WorkspaceID)
		item.ProjectID = strings.TrimSpace(item.ProjectID)
		item.CampaignID = strings.TrimSpace(item.CampaignID)
		if item.AssetID == "" {
			continue
		}
		key := item.Role + "\x00" + item.AssetID + "\x00" + item.Source
		if seen[key] {
			continue
		}
		seen[key] = true
		normalized = append(normalized, item)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeStoryContinuityWarnings(items []domain.StoryContinuityWarning) []domain.StoryContinuityWarning {
	if len(items) == 0 {
		return nil
	}
	normalized := make([]domain.StoryContinuityWarning, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item.Code = strings.TrimSpace(item.Code)
		item.Message = strings.TrimSpace(item.Message)
		if item.Code == "" && item.Message == "" {
			continue
		}
		key := item.Code + "\x00" + item.Message
		if seen[key] {
			continue
		}
		seen[key] = true
		normalized = append(normalized, item)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeStoryContinuityPolicy(policy domain.StoryContinuityPolicy) domain.StoryContinuityPolicy {
	policy.Mode = strings.TrimSpace(policy.Mode)
	if policy.MaxPanels < 0 {
		policy.MaxPanels = 0
	}
	if policy.MaxCandidatesPerPanel < 0 {
		policy.MaxCandidatesPerPanel = 0
	}
	if policy.ProviderConcurrency < 0 {
		policy.ProviderConcurrency = 0
	}
	if policy.MaxTotalImagesIncludingRegenerate < 0 {
		policy.MaxTotalImagesIncludingRegenerate = 0
	}
	return policy
}

func selectStoryPanelPlanEntry(story domain.StoryContextV1, metadata map[string]any) (domain.StoryPanelPlanEntry, error) {
	if len(story.PanelPlan) == 0 {
		return domain.StoryPanelPlanEntry{}, fmt.Errorf("story_context_v1.panel_plan is required")
	}
	sceneID := mapString(metadata, "scene_id")
	panelIndex := mapInt(metadata, "panel_index")

	if sceneID != "" {
		for _, panel := range story.PanelPlan {
			if panel.SceneID == sceneID {
				return panel, nil
			}
		}
		return domain.StoryPanelPlanEntry{}, fmt.Errorf("story_context_v1.panel_plan does not contain scene_id %q", sceneID)
	}
	if panelIndex > 0 {
		for _, panel := range story.PanelPlan {
			if panel.PanelIndex == panelIndex {
				return panel, nil
			}
		}
		return domain.StoryPanelPlanEntry{}, fmt.Errorf("story_context_v1.panel_plan does not contain panel_index %d", panelIndex)
	}
	if len(story.PanelPlan) == 1 {
		return story.PanelPlan[0], nil
	}
	return domain.StoryPanelPlanEntry{}, fmt.Errorf("story_context_v1 panel selection requires metadata_json.scene_id or panel_index")
}

func buildStoryResolvedReferenceAssets(
	scope domain.Scope,
	references []domain.ReferenceImage,
	resolved *resolvedTaskInputFiles,
) []domain.StoryResolvedReferenceAsset {
	if len(references) == 0 || resolved == nil || len(resolved.ReferenceImages) == 0 {
		return nil
	}
	output := make([]domain.StoryResolvedReferenceAsset, 0, len(references))
	seen := map[string]bool{}
	for index, reference := range references {
		if index >= len(resolved.ReferenceImages) {
			break
		}
		assetID := strings.TrimSpace(reference.AssetID)
		if assetID == "" {
			continue
		}
		role := strings.TrimSpace(reference.Role)
		source := strings.TrimSpace(reference.Source)
		key := role + "\x00" + assetID + "\x00" + source
		if seen[key] {
			continue
		}
		seen[key] = true
		output = append(output, domain.StoryResolvedReferenceAsset{
			Role:        role,
			AssetID:     assetID,
			Source:      source,
			WorkspaceID: scope.WorkspaceID,
			ProjectID:   scope.ProjectID,
			CampaignID:  scope.CampaignID,
		})
	}
	return normalizeStoryResolvedReferenceAssets(output)
}

func buildStoryContinuityWarnings(story domain.StoryContextV1, diagnostics domain.ReferenceParticipationDiagnostics) []domain.StoryContinuityWarning {
	warnings := append([]domain.StoryContinuityWarning(nil), story.ContinuityWarnings...)
	if len(story.ReferenceBindings) > 0 && len(story.ResolvedReferenceAssets) == 0 {
		warnings = append(warnings, domain.StoryContinuityWarning{
			Code:    "no_resolved_reference_assets",
			Message: "story_context_v1 declared references but no resolved project assets were attached",
		})
	}
	if strings.TrimSpace(diagnostics.ProviderReferenceParticipation) == "descriptor_only" {
		warnings = append(warnings, domain.StoryContinuityWarning{
			Code:    "descriptor_only_references",
			Message: "reference descriptors were present but no resolved input files reached the provider",
		})
	}
	return normalizeStoryContinuityWarnings(warnings)
}

func validateStoryContinuityAssetScope(expected, actual domain.Scope) error {
	if actual.WorkspaceID != expected.WorkspaceID || actual.ProjectID != expected.ProjectID {
		return fmt.Errorf("asset belongs to workspace/project %s/%s, not %s/%s", actual.WorkspaceID, actual.ProjectID, expected.WorkspaceID, expected.ProjectID)
	}
	if expected.CampaignID != "" && actual.CampaignID != "" && actual.CampaignID != expected.CampaignID {
		return fmt.Errorf("asset belongs to campaign %s, not %s", actual.CampaignID, expected.CampaignID)
	}
	return nil
}

func validatePreviousPanelStoryLink(story domain.StoryContextV1, panel domain.StoryPanelPlanEntry, metadataRaw json.RawMessage) error {
	if panel.PanelIndex <= 1 {
		return nil
	}
	metadata, err := metadataMap(metadataRaw)
	if err != nil {
		return fmt.Errorf("previous panel metadata is invalid: %w", err)
	}
	expectedPanelIndex := panel.PanelIndex - 1
	expectedSceneID := ""
	for _, candidate := range story.PanelPlan {
		if candidate.PanelIndex == expectedPanelIndex {
			expectedSceneID = candidate.SceneID
			break
		}
	}
	if storyID := mapString(metadata, "story_id"); story.StoryID != "" && storyID != "" && storyID != story.StoryID {
		return fmt.Errorf("previous panel story_id = %q, want %q", storyID, story.StoryID)
	}
	if previousPanelIndex := mapInt(metadata, "panel_index"); previousPanelIndex > 0 && previousPanelIndex != expectedPanelIndex {
		return fmt.Errorf("previous panel panel_index = %d, want %d", previousPanelIndex, expectedPanelIndex)
	}
	if previousSceneID := mapString(metadata, "scene_id"); expectedSceneID != "" && previousSceneID != "" && previousSceneID != expectedSceneID {
		return fmt.Errorf("previous panel scene_id = %q, want %q", previousSceneID, expectedSceneID)
	}
	return nil
}

func appendStoryPanelTransitionPrompt(prompt string, story *domain.StoryContextV1, metadataRaw json.RawMessage) string {
	prompt = strings.TrimSpace(prompt)
	if story == nil || len(story.PanelPlan) == 0 {
		return prompt
	}
	metadata, err := metadataMap(metadataRaw)
	if err != nil {
		return prompt
	}
	panel, err := selectStoryPanelPlanEntry(*story, metadata)
	if err != nil {
		return prompt
	}

	hints := make([]string, 0, 7)
	if strings.TrimSpace(panel.EmotionBefore) != "" {
		hints = append(hints, fmt.Sprintf("Emotion before: %s.", panel.EmotionBefore))
	}
	if strings.TrimSpace(panel.EmotionAfter) != "" {
		hints = append(hints, fmt.Sprintf("Emotion after: %s.", panel.EmotionAfter))
	}
	if strings.TrimSpace(panel.PoseChange) != "" {
		hints = append(hints, fmt.Sprintf("Pose change: %s.", panel.PoseChange))
	}
	if strings.TrimSpace(panel.RelationshipShift) != "" {
		hints = append(hints, fmt.Sprintf("Relationship shift: %s.", panel.RelationshipShift))
	}
	if len(panel.MustChange) > 0 {
		hints = append(hints, fmt.Sprintf("Must change: %s.", strings.Join(panel.MustChange, "; ")))
	}
	if len(panel.MustNotKeep) > 0 {
		hints = append(hints, fmt.Sprintf("Must not keep: %s.", strings.Join(panel.MustNotKeep, "; ")))
	}
	if strings.TrimSpace(panel.StateTransitionNotes) != "" {
		hints = append(hints, fmt.Sprintf("State transition notes: %s.", panel.StateTransitionNotes))
	}
	if len(hints) == 0 {
		return prompt
	}
	return strings.TrimSpace(prompt + "\n\nState transition requirements:\n- " + strings.Join(hints, "\n- "))
}

func isSequentialStoryContext(story domain.StoryContextV1) bool {
	mode := firstNonEmptyString(story.GenerationMode, story.ContinuityPolicy.Mode)
	return mode == domain.StoryGenerationModeSequentialPreviousPanel
}

func hasReferenceImageAssetRole(items []domain.ReferenceImage, assetID, role string) bool {
	assetID = strings.TrimSpace(assetID)
	role = strings.TrimSpace(role)
	for _, item := range items {
		if strings.TrimSpace(item.AssetID) == assetID && strings.TrimSpace(item.Role) == role {
			return true
		}
	}
	return false
}

func firstReferenceBindingValue(bindings domain.StoryReferenceBindings, role string) string {
	for _, value := range bindings[strings.TrimSpace(role)] {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func firstReferenceImageAssetIDByRole(items []domain.ReferenceImage, role string) string {
	role = strings.TrimSpace(role)
	for _, item := range items {
		if strings.TrimSpace(item.Role) == role && strings.TrimSpace(item.AssetID) != "" {
			return strings.TrimSpace(item.AssetID)
		}
	}
	return ""
}

func setMetadataString(metadata map[string]any, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		delete(metadata, key)
		return
	}
	metadata[key] = value
}

func setMetadataStringSlice(metadata map[string]any, key string, values []string) {
	values = normalizeStringList(values)
	if len(values) == 0 {
		delete(metadata, key)
		return
	}
	metadata[key] = values
}

func mapString(metadata map[string]any, key string) string {
	value, ok := metadata[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func mapInt(metadata map[string]any, key string) int {
	value, ok := metadata[key]
	if !ok || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
