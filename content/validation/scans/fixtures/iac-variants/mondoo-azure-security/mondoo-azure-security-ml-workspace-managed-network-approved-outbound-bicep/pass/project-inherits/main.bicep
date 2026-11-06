resource workspace 'Microsoft.MachineLearningServices/workspaces@2024-10-01' = {
  name: 'example-workspace'
  location: 'eastus'
  kind: 'Project'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    hubResourceId: '/subscriptions/000/resourceGroups/rg/providers/Microsoft.MachineLearningServices/workspaces/hub'
  }
}
