// Training data, models, and notebooks in the workspace stay under the
// Microsoft-managed key.
resource mlWorkspace 'Microsoft.MachineLearningServices/workspaces@2024-04-01' = {
  name: 'research-ml'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    friendlyName: 'Research'
    encryption: {
      status: 'Disabled'
    }
  }
}
