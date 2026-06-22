export const loadAgentWorkspace = () => import('../components/AgentWorkspace')
export const loadDetailModal = () => import('../components/DetailModal')
export const loadLightbox = () => import('../components/Lightbox')
export const loadSettingsModal = () => import('../components/SettingsModal')
export const loadScopeManagerModal = () => import('../components/ScopeManagerModal')
export const loadProjectContextModal = () => import('../components/ProjectContextModal')
export const loadProductionViewModal = () => import('../components/ProductionViewModal')
export const loadMaskEditorModal = () => import('../components/MaskEditorModal')

export const preloadAgentWorkspace = () => {
  void loadAgentWorkspace()
}

export const preloadDetailModal = () => {
  void loadDetailModal()
}

export const preloadLightbox = () => {
  void loadLightbox()
}

export const preloadSettingsModal = () => {
  void loadSettingsModal()
}

export const preloadScopeManagerModal = () => {
  void loadScopeManagerModal()
}

export const preloadProjectContextModal = () => {
  void loadProjectContextModal()
}

export const preloadProductionViewModal = () => {
  void loadProductionViewModal()
}

export const preloadMaskEditorModal = () => {
  void loadMaskEditorModal()
}
