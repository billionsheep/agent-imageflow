package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/billionsheep/agent-imageflow/internal/domain"
)

const visualContextSnapshotKey = "visual_context_snapshot"

type assetScopeResolver func(context.Context, string) (domain.Scope, error)

type visualContextExpansion struct {
	Request  domain.CreateTaskRequest
	Snapshot *domain.VisualContextSnapshot
}

func (s *Service) GetProjectVisualContext(ctx context.Context, workspaceID, projectID string) (domain.ProjectVisualContextResponse, error) {
	visualContext, err := s.store.GetProjectVisualContext(ctx, workspaceID, projectID)
	if err != nil {
		return domain.ProjectVisualContextResponse{}, err
	}
	normalized, err := normalizeProjectVisualContext(visualContext, time.Now().UTC())
	if err != nil {
		return domain.ProjectVisualContextResponse{}, err
	}
	return domain.ProjectVisualContextResponse{
		WorkspaceID:   workspaceID,
		ProjectID:     projectID,
		VisualContext: normalized,
	}, nil
}

func (s *Service) UpdateProjectVisualContext(ctx context.Context, workspaceID, projectID string, visualContext domain.ProjectVisualContext) (domain.ProjectVisualContextResponse, error) {
	normalized, err := normalizeProjectVisualContext(visualContext, time.Now().UTC())
	if err != nil {
		return domain.ProjectVisualContextResponse{}, err
	}
	scope := domain.Scope{WorkspaceID: workspaceID, ProjectID: projectID}
	if err := validateProjectVisualContextAssetScopes(ctx, scope, normalized, s.store.GetAssetScope); err != nil {
		return domain.ProjectVisualContextResponse{}, err
	}
	saved, err := s.store.UpdateProjectVisualContext(ctx, workspaceID, projectID, normalized)
	if err != nil {
		return domain.ProjectVisualContextResponse{}, err
	}
	saved, err = normalizeProjectVisualContext(saved, time.Now().UTC())
	if err != nil {
		return domain.ProjectVisualContextResponse{}, err
	}
	return domain.ProjectVisualContextResponse{
		WorkspaceID:   workspaceID,
		ProjectID:     projectID,
		VisualContext: saved,
	}, nil
}

func normalizeProjectVisualContext(input domain.ProjectVisualContext, now time.Time) (domain.ProjectVisualContext, error) {
	output := domain.ProjectVisualContext{
		Characters:    make([]domain.CharacterProfile, 0, len(input.Characters)),
		References:    make([]domain.ProjectReferenceBinding, 0, len(input.References)),
		PromptRecipes: make([]domain.PromptRecipe, 0, len(input.PromptRecipes)),
		UpdatedAt:     input.UpdatedAt,
	}
	if output.UpdatedAt.IsZero() {
		output.UpdatedAt = now
	}

	seenCharacters := map[string]bool{}
	for _, character := range input.Characters {
		normalized, err := normalizeCharacterProfile(character, now)
		if err != nil {
			return domain.ProjectVisualContext{}, err
		}
		if normalized.ID == "" {
			continue
		}
		if seenCharacters[normalized.ID] {
			return domain.ProjectVisualContext{}, fmt.Errorf("duplicate character id %q", normalized.ID)
		}
		seenCharacters[normalized.ID] = true
		output.Characters = append(output.Characters, normalized)
	}

	seenReferences := map[string]bool{}
	for _, reference := range input.References {
		normalized, err := normalizeProjectReferenceBinding(reference, now, seenCharacters)
		if err != nil {
			return domain.ProjectVisualContext{}, err
		}
		if normalized.ID == "" {
			continue
		}
		if seenReferences[normalized.ID] {
			return domain.ProjectVisualContext{}, fmt.Errorf("duplicate reference id %q", normalized.ID)
		}
		seenReferences[normalized.ID] = true
		output.References = append(output.References, normalized)
	}

	seenRecipes := map[string]bool{}
	for _, recipe := range input.PromptRecipes {
		normalized, err := normalizePromptRecipe(recipe, now)
		if err != nil {
			return domain.ProjectVisualContext{}, err
		}
		if normalized.ID == "" {
			continue
		}
		if seenRecipes[normalized.ID] {
			return domain.ProjectVisualContext{}, fmt.Errorf("duplicate prompt recipe id %q", normalized.ID)
		}
		seenRecipes[normalized.ID] = true
		output.PromptRecipes = append(output.PromptRecipes, normalized)
	}
	return output, nil
}

func normalizeCharacterProfile(input domain.CharacterProfile, now time.Time) (domain.CharacterProfile, error) {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		input.ID = domain.NewID("char")
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		input.Name = input.ID
	}
	input.Status = normalizeVisualContextStatus(input.Status)
	if input.Status == "" {
		return domain.CharacterProfile{}, fmt.Errorf("unknown character status %q", input.Status)
	}
	input.Role = strings.TrimSpace(input.Role)
	input.Appearance = strings.TrimSpace(input.Appearance)
	input.Personality = strings.TrimSpace(input.Personality)
	input.PrimaryAssetID = strings.TrimSpace(input.PrimaryAssetID)
	input.ReferenceAssetIDs = normalizeStringList(input.ReferenceAssetIDs)
	input.Forbidden = normalizeStringList(input.Forbidden)
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = now
	}
	return input, nil
}

func normalizeProjectReferenceBinding(input domain.ProjectReferenceBinding, now time.Time, characterIDs map[string]bool) (domain.ProjectReferenceBinding, error) {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		input.ID = domain.NewID("ref")
	}
	input.AssetID = strings.TrimSpace(input.AssetID)
	if input.AssetID == "" {
		return domain.ProjectReferenceBinding{}, fmt.Errorf("reference %q asset_id is required", input.ID)
	}
	input.Purpose = normalizeReferencePurpose(input.Purpose)
	if input.Purpose == "" {
		return domain.ProjectReferenceBinding{}, fmt.Errorf("reference %q purpose must be character, style, scene or prop", input.ID)
	}
	input.Label = strings.TrimSpace(input.Label)
	input.Notes = strings.TrimSpace(input.Notes)
	input.CharacterID = strings.TrimSpace(input.CharacterID)
	if input.CharacterID != "" && !characterIDs[input.CharacterID] {
		return domain.ProjectReferenceBinding{}, fmt.Errorf("reference %q links unknown character %q", input.ID, input.CharacterID)
	}
	input.Status = normalizeVisualContextStatus(input.Status)
	if input.Status == "" {
		return domain.ProjectReferenceBinding{}, fmt.Errorf("unknown reference status %q", input.Status)
	}
	if input.Weight <= 0 {
		input.Weight = 1
	}
	if input.Weight > 5 {
		input.Weight = 5
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = now
	}
	return input, nil
}

func normalizePromptRecipe(input domain.PromptRecipe, now time.Time) (domain.PromptRecipe, error) {
	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		input.ID = domain.NewID("recipe")
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		input.Name = input.ID
	}
	input.Status = normalizeVisualContextStatus(input.Status)
	if input.Status == "" {
		return domain.PromptRecipe{}, fmt.Errorf("unknown prompt recipe status %q", input.Status)
	}
	input.NegativePrompt = strings.TrimSpace(input.NegativePrompt)
	input.DefaultAspectRatio = strings.TrimSpace(input.DefaultAspectRatio)
	input.DefaultOutputFormat = strings.TrimSpace(input.DefaultOutputFormat)
	input.DefaultProvider = strings.TrimSpace(input.DefaultProvider)
	input.DefaultModel = strings.TrimSpace(input.DefaultModel)
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = now
	}
	if len(input.GenerationConfig) > 0 {
		if !json.Valid(input.GenerationConfig) {
			return domain.PromptRecipe{}, fmt.Errorf("prompt recipe %q generation_config must be valid JSON", input.ID)
		}
		var generationConfig map[string]any
		if err := json.Unmarshal(input.GenerationConfig, &generationConfig); err != nil || generationConfig == nil {
			return domain.PromptRecipe{}, fmt.Errorf("prompt recipe %q generation_config must be a JSON object", input.ID)
		}
		input.GenerationConfig = compactJSON(input.GenerationConfig)
	}
	blocks := make([]domain.PromptBlock, 0, len(input.PromptBlocks))
	for _, block := range input.PromptBlocks {
		block.ID = strings.TrimSpace(block.ID)
		block.Role = strings.TrimSpace(block.Role)
		block.Text = strings.TrimSpace(block.Text)
		if block.Text == "" {
			continue
		}
		blocks = append(blocks, block)
	}
	input.PromptBlocks = blocks
	return input, nil
}

func normalizeVisualContextStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "", "active":
		return "active"
	case "archived":
		return "archived"
	default:
		return ""
	}
}

func normalizeReferencePurpose(purpose string) string {
	switch strings.TrimSpace(purpose) {
	case "":
		return "style"
	case "character", "style", "scene", "prop":
		return strings.TrimSpace(purpose)
	default:
		return ""
	}
}

func normalizeStringList(items []string) []string {
	seen := map[string]bool{}
	normalized := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		normalized = append(normalized, item)
	}
	return normalized
}

func validateProjectVisualContextAssetScopes(ctx context.Context, scope domain.Scope, visualContext domain.ProjectVisualContext, resolve assetScopeResolver) error {
	for _, assetID := range collectProjectVisualContextAssetIDs(visualContext) {
		assetScope, err := resolve(ctx, assetID)
		if err != nil {
			return fmt.Errorf("asset %s cannot be used in project visual context: %w", assetID, err)
		}
		if assetScope.WorkspaceID != scope.WorkspaceID || assetScope.ProjectID != scope.ProjectID {
			return fmt.Errorf("asset %s belongs to workspace/project %s/%s, not %s/%s", assetID, assetScope.WorkspaceID, assetScope.ProjectID, scope.WorkspaceID, scope.ProjectID)
		}
	}
	return nil
}

func collectProjectVisualContextAssetIDs(visualContext domain.ProjectVisualContext) []string {
	ids := []string{}
	for _, character := range visualContext.Characters {
		ids = append(ids, character.PrimaryAssetID)
		ids = append(ids, character.ReferenceAssetIDs...)
	}
	for _, reference := range visualContext.References {
		ids = append(ids, reference.AssetID)
	}
	return normalizeStringList(ids)
}

func (s *Service) expandProjectVisualContext(ctx context.Context, scope domain.Scope, req domain.CreateTaskRequest) (visualContextExpansion, error) {
	req.CharacterIDs = normalizeStringList(req.CharacterIDs)
	req.ReferenceAssetIDs = normalizeStringList(req.ReferenceAssetIDs)
	req.PromptRecipeID = strings.TrimSpace(req.PromptRecipeID)
	shouldExpand := req.UseProjectVisualContext || len(req.CharacterIDs) > 0 || len(req.ReferenceAssetIDs) > 0 || req.PromptRecipeID != ""
	if !shouldExpand {
		return visualContextExpansion{Request: req}, nil
	}

	visualContext, err := s.store.GetProjectVisualContext(ctx, scope.WorkspaceID, scope.ProjectID)
	if err != nil {
		return visualContextExpansion{}, err
	}
	visualContext, err = normalizeProjectVisualContext(visualContext, time.Now().UTC())
	if err != nil {
		return visualContextExpansion{}, err
	}
	expanded, snapshot, err := applyProjectVisualContext(req, visualContext)
	if err != nil {
		return visualContextExpansion{}, err
	}
	if err := validateTaskReferenceAssetScopes(ctx, scope, expanded, s.store.GetAssetScope); err != nil {
		return visualContextExpansion{}, err
	}
	return visualContextExpansion{Request: expanded, Snapshot: snapshot}, nil
}

func applyProjectVisualContext(req domain.CreateTaskRequest, visualContext domain.ProjectVisualContext) (domain.CreateTaskRequest, *domain.VisualContextSnapshot, error) {
	charactersByID := map[string]domain.CharacterProfile{}
	for _, character := range visualContext.Characters {
		charactersByID[character.ID] = character
	}
	referenceIDs := normalizeStringList(req.ReferenceAssetIDs)
	selectedCharacters := make([]domain.CharacterProfile, 0, len(req.CharacterIDs))
	for _, id := range req.CharacterIDs {
		character, ok := charactersByID[id]
		if !ok {
			return req, nil, fmt.Errorf("character %q was not found in project visual context", id)
		}
		if character.Status == "archived" {
			return req, nil, fmt.Errorf("character %q is archived", id)
		}
		selectedCharacters = append(selectedCharacters, character)
		referenceIDs = append(referenceIDs, character.PrimaryAssetID)
		referenceIDs = append(referenceIDs, character.ReferenceAssetIDs...)
	}
	referenceIDs = normalizeStringList(referenceIDs)

	selectedReferences := []domain.ProjectReferenceBinding{}
	selectedCharacterIDs := map[string]bool{}
	for _, character := range selectedCharacters {
		selectedCharacterIDs[character.ID] = true
	}
	for _, reference := range visualContext.References {
		if reference.Status == "archived" {
			continue
		}
		if reference.CharacterID != "" && !selectedCharacterIDs[reference.CharacterID] {
			continue
		}
		if req.UseProjectVisualContext || containsString(referenceIDs, reference.AssetID) || reference.CharacterID != "" {
			selectedReferences = append(selectedReferences, reference)
			referenceIDs = append(referenceIDs, reference.AssetID)
		}
	}
	referenceIDs = normalizeStringList(referenceIDs)

	req.ReferenceImages = appendVisualContextReferenceImages(req.ReferenceImages, selectedCharacters, selectedReferences, referenceIDs)
	recipe := activePromptRecipe(visualContext.PromptRecipes, req.PromptRecipeID)
	if req.PromptRecipeID != "" && recipe == nil {
		return req, nil, fmt.Errorf("prompt recipe %q was not found or is archived", req.PromptRecipeID)
	}
	if recipe != nil {
		var err error
		req, err = applyPromptRecipe(req, *recipe)
		if err != nil {
			return req, nil, err
		}
	}

	snapshot := &domain.VisualContextSnapshot{
		Source:            "project",
		CharacterIDs:      req.CharacterIDs,
		ReferenceAssetIDs: referenceIDs,
		PromptRecipeID:    req.PromptRecipeID,
		Characters:        selectedCharacters,
		References:        selectedReferences,
		PromptRecipe:      recipe,
	}
	metadata, err := metadataMap(req.MetadataJSON)
	if err != nil {
		return req, nil, err
	}
	metadata[visualContextSnapshotKey] = snapshot
	raw, err := json.Marshal(metadata)
	if err != nil {
		return req, nil, err
	}
	req.MetadataJSON = raw
	return req, snapshot, nil
}

func appendVisualContextReferenceImages(existing []domain.ReferenceImage, characters []domain.CharacterProfile, bindings []domain.ProjectReferenceBinding, referenceIDs []string) []domain.ReferenceImage {
	byAssetID := map[string]domain.ReferenceImage{}
	order := []string{}
	add := func(assetID string, role string, weight float64) {
		assetID = strings.TrimSpace(assetID)
		if assetID == "" {
			return
		}
		if _, exists := byAssetID[assetID]; exists {
			return
		}
		order = append(order, assetID)
		if role == "" {
			role = "visual_context"
		}
		byAssetID[assetID] = domain.ReferenceImage{
			ID:      "vc_" + assetID,
			AssetID: assetID,
			Role:    role,
			Source:  "project_visual_context",
			Weight:  weight,
		}
	}
	for _, item := range existing {
		if item.AssetID != "" {
			byAssetID[item.AssetID] = item
			order = append(order, item.AssetID)
		}
	}
	for _, character := range characters {
		add(character.PrimaryAssetID, "character_primary", 1)
		for _, assetID := range character.ReferenceAssetIDs {
			add(assetID, "character", 1)
		}
	}
	for _, binding := range bindings {
		add(binding.AssetID, binding.Purpose, binding.Weight)
	}
	for _, assetID := range referenceIDs {
		add(assetID, "requested_reference", 1)
	}
	output := make([]domain.ReferenceImage, 0, len(existing)+len(order))
	added := map[string]bool{}
	for _, item := range existing {
		key := item.AssetID
		if key == "" {
			output = append(output, item)
			continue
		}
		output = append(output, byAssetID[key])
		added[key] = true
	}
	for _, assetID := range order {
		if assetID == "" || added[assetID] {
			continue
		}
		output = append(output, byAssetID[assetID])
		added[assetID] = true
	}
	return output
}

func activePromptRecipe(recipes []domain.PromptRecipe, recipeID string) *domain.PromptRecipe {
	recipeID = strings.TrimSpace(recipeID)
	if recipeID == "" {
		return nil
	}
	for _, recipe := range recipes {
		if recipe.ID == recipeID && recipe.Status != "archived" {
			recipe := recipe
			return &recipe
		}
	}
	return nil
}

func applyPromptRecipe(req domain.CreateTaskRequest, recipe domain.PromptRecipe) (domain.CreateTaskRequest, error) {
	if req.NegativePrompt == "" {
		req.NegativePrompt = recipe.NegativePrompt
	}
	if req.AspectRatio == "" {
		req.AspectRatio = recipe.DefaultAspectRatio
	}
	if req.OutputFormat == "" {
		req.OutputFormat = recipe.DefaultOutputFormat
	}
	if req.Provider == "" {
		req.Provider = recipe.DefaultProvider
	}
	generationConfig, err := mergeJSONObjects(recipe.GenerationConfig, req.GenerationConfig)
	if err != nil {
		return req, err
	}
	if recipe.DefaultModel != "" {
		generationConfig, err = setJSONDefaultString(generationConfig, "model", recipe.DefaultModel)
		if err != nil {
			return req, err
		}
	}
	req.GenerationConfig = generationConfig

	renderedBlocks := []string{}
	rawBlocks := []string{}
	metadata, err := metadataMap(req.MetadataJSON)
	if err != nil {
		return req, err
	}
	variables := templateVariables(req, metadata)
	for _, block := range recipe.PromptBlocks {
		rawBlocks = append(rawBlocks, block.Text)
		rendered := strings.TrimSpace(renderPromptTemplate(block.Text, variables))
		if rendered != "" {
			renderedBlocks = append(renderedBlocks, rendered)
		}
	}
	if len(renderedBlocks) == 0 {
		return req, nil
	}
	renderedPrompt := strings.Join(renderedBlocks, "\n\n")
	if req.Prompt != "" && !promptBlocksContainPromptToken(rawBlocks) {
		req.Prompt = strings.TrimSpace(req.Prompt + "\n\n" + renderedPrompt)
	} else {
		req.Prompt = renderedPrompt
	}
	return req, nil
}

func promptBlocksContainPromptToken(blocks []string) bool {
	for _, block := range blocks {
		if strings.Contains(block, "{{prompt") || strings.Contains(block, "{{ prompt") {
			return true
		}
	}
	return false
}

func mergeJSONObjects(defaults, override json.RawMessage) (json.RawMessage, error) {
	if len(defaults) == 0 {
		return override, nil
	}
	if len(override) == 0 {
		return defaults, nil
	}
	var defaultMap map[string]any
	if err := json.Unmarshal(defaults, &defaultMap); err != nil || defaultMap == nil {
		return nil, fmt.Errorf("prompt recipe generation_config must be a JSON object")
	}
	var overrideMap map[string]any
	if err := json.Unmarshal(override, &overrideMap); err != nil || overrideMap == nil {
		return nil, fmt.Errorf("generation_config must be a JSON object")
	}
	for key, value := range overrideMap {
		defaultMap[key] = value
	}
	raw, err := json.Marshal(defaultMap)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func setJSONDefaultString(raw json.RawMessage, key, value string) (json.RawMessage, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return raw, nil
	}
	object := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &object); err != nil || object == nil {
			return nil, fmt.Errorf("generation_config must be a JSON object")
		}
	}
	if existing, ok := object[key]; ok && strings.TrimSpace(fmt.Sprint(existing)) != "" {
		return raw, nil
	}
	object[key] = value
	encoded, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func validateTaskReferenceAssetScopes(ctx context.Context, scope domain.Scope, req domain.CreateTaskRequest, resolve assetScopeResolver) error {
	ids := normalizeStringList(req.ReferenceAssetIDs)
	for _, ref := range req.ReferenceImages {
		ids = append(ids, strings.TrimSpace(ref.AssetID))
	}
	for _, assetID := range normalizeStringList(ids) {
		assetScope, err := resolve(ctx, assetID)
		if err != nil {
			return fmt.Errorf("asset %s cannot be used as project visual context reference: %w", assetID, err)
		}
		if assetScope.WorkspaceID != scope.WorkspaceID || assetScope.ProjectID != scope.ProjectID {
			return fmt.Errorf("asset %s belongs to workspace/project %s/%s, not %s/%s", assetID, assetScope.WorkspaceID, assetScope.ProjectID, scope.WorkspaceID, scope.ProjectID)
		}
	}
	return nil
}

func containsString(items []string, needle string) bool {
	needle = strings.TrimSpace(needle)
	for _, item := range items {
		if strings.TrimSpace(item) == needle {
			return true
		}
	}
	return false
}
