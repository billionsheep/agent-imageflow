export interface ProjectContextPanelRecipe {
  id: string
  name?: string
}

export interface ProjectContextPanelCharacter {
  id: string
  name?: string
}

export interface ProjectContextPanelReference {
  asset_id: string
  label?: string
  purpose?: string
}

export interface ProjectContextPanelSummaryInput {
  useProjectVisualContext: boolean
  selectedRecipeId: string
  selectedCharacterIds: string[]
  selectedReferenceAssetIds: string[]
  recipes: ProjectContextPanelRecipe[]
  characters: ProjectContextPanelCharacter[]
  references: ProjectContextPanelReference[]
}

function findName<T extends { id?: string; asset_id?: string; name?: string; label?: string }>(
  values: T[],
  id: string,
): string {
  const found = values.find((value) => value.id === id || value.asset_id === id)
  return found?.name || found?.label || id
}

export function getProjectContextPanelSummary(input: ProjectContextPanelSummaryInput): string {
  const recipeId = input.selectedRecipeId.trim()
  const parts: string[] = []
  if (recipeId) {
    parts.push(findName(input.recipes, recipeId))
  }

  if (input.selectedCharacterIds.length > 0) {
    parts.push(input.selectedCharacterIds.map((id) => findName(input.characters, id)).join(', '))
  }

  if (input.selectedReferenceAssetIds.length > 0) {
    const count = input.selectedReferenceAssetIds.length
    parts.push(`${count} reference${count === 1 ? '' : 's'}`)
  }

  if (parts.length > 0) return parts.join(' · ')
  return input.useProjectVisualContext ? 'Project defaults enabled' : 'No context selected'
}
